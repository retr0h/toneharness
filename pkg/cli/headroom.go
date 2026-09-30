// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// This file takes a preset off the measuring loop before it is measured.
//
// The measuring rig is a lead from the pedal's output socket back into its own
// input. A preset compiled from a plan inherits the blank template's routing,
// which sends the chain to destination 1, `Multi (1/4", XLR, Digital, USB
// 1/2)`, and Multi drives the socket that lead comes from. So the chain's
// output arrives at its own input and it feeds itself.
//
// Two things follow, and they are fixed in that order.
//
// The destination is the cause. Sending to `USB 1/2` by itself reaches the
// computer without reaching the quarter-inch socket, so the lead carries only
// the reference recording being played and the loop is open. That is the fix,
// and it is the one that works at any gain.
//
// The gain is the second half, and it was the whole of it before the
// destination was: turning `dsp0.outputA.gain` down drops the loop's gain
// below unity without touching the tone, which ChVol and Master are. It is
// kept because it also bounds what an amplifier's own hiss and any remaining
// path around the rig can do, and because the readings this project has
// committed were taken with it.
//
// What it looked like while the destination was wrong: matt-freeman read 84.3%
// of its energy above 2kHz where the reference has 0.01%, and the same chain
// with the amplifier turned down read 0.0% for three decibels less level. The
// gain alone was not enough for a high-gain amplifier, which has enough of its
// own to close the loop from -78dB: 14 of the first 125 blocks in a campaign
// refused, every one of them an amp or preamp built to distort, the two SV
// Beasts this project's own pipeline uses among them.

// outputSlot is the routing entry the chain's output sits in, and gainKey the
// parameter inside it.
//
// Named rather than indexed. An output entry sends two values and the device
// records them in its model's order, which routeValues reads by name from the
// plan rather than by position.
const (
	outputSlot = "dsp0.outputA"
	gainKey    = "gain"
	sendKey    = "@output"
)

// offTheLoopIs is the destination a measured chain sends to, by the name the
// device's own list gives it.
//
// By name rather than by number. The lists are per device family, so the
// position `USB 1/2` sits at on an HX Stomp is not the position it sits at on
// an LT or in the plugin, and a number written here would be right for one of
// them.
const offTheLoopIs = "USB 1/2"

// ErrNoOutput is a preset with no output entry to turn down.
var ErrNoOutput = errors.New("nothing to give headroom to")

// Quiets is what lowering a preset's output takes: read the preset back for
// the routing it arrived with, and write it again.
type Quiets interface {
	Compiles
	ReadsFiles
}

// quieter writes the plan again off the measuring loop, and returns the preset
// compiled from it.
//
// A round trip through a plan rather than a live edit, because routing is not
// a block: `device turn` addresses a block and a parameter, and an output
// entry is neither. What a preset says about its routing is settled when the
// preset is written.
//
// by is in decibels and wants to be negative. Zero asks for no headroom, and
// the preset is still rewritten: the destination is what opens the loop and a
// caller wanting full level wants it open too. Before this took the
// destination on as well, zero returned the preset untouched, which was a
// preset still sending to the socket the measuring lead comes from.
func quieter(
	ctx context.Context,
	client Quiets,
	already, work, name string,
	by float64,
) (string, error) {
	// Read back off the preset that was just built, because a plan compiled
	// from a rig carries no routing at all: the inputs and outputs come from
	// the blank template while the preset is written, so there is nothing in
	// the plan to change. The preset has them, and reading it back is how the
	// entry arrives complete, with its model and its output already set.
	read, err := client.PresetFile(ctx, already)
	if err != nil {
		return "", err
	}

	to, err := sendTo(read.Plan)
	if err != nil {
		return "", err
	}

	lowered, err := offTheLoop(read.Plan, to, by)
	if err != nil {
		return "", err
	}

	at := filepath.Join(work, name+".headroom.yaml")

	f, err := os.Create(at) //nolint:gosec // a path this builds in the temp dir
	if err != nil {
		return "", fmt.Errorf("writing %s: %w", at, err)
	}

	if err := plan.Write(f, lowered); err != nil {
		_ = f.Close()

		return "", fmt.Errorf("writing %s: %w", at, err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("writing %s: %w", at, err)
	}

	out := filepath.Join(work, name+".headroom.hlx")

	if _, err := client.Compile(ctx, sdk.Compile{
		Plan: at, Out: out, Existing: sdk.ReplaceExisting,
	}); err != nil {
		return "", err
	}

	return out, nil
}

// sendTo is the number this preset's own device files the off-the-loop
// destination under.
//
// Resolved against the device the preset says it is for, not against the
// built-in catalog, which is the HX Stomp's whatever is attached.
//
// All four catalogs shipped today put `USB 1/2` at 10, so this changes no
// number. It is the difference between a number that happens to be right and one
// that was asked for: the destination lists are per device family, the same file
// carries a separate one for a Stomp, an LT and the plugin, and a Stomp's lists
// four Returns it has no sockets for. Nothing promises the next firmware keeps
// them aligned, and a wrong destination here does not fail. It sends the chain
// back down the measuring lead and every reading after it is of the rig
// listening to itself.
//
// A preset naming no device, or one no catalog ships for, falls back to the
// built-in. That is the HX Stomp's, which is the only device anything here has
// been written to over USB, so it is the right guess rather than no answer.
func sendTo(
	made plan.Plan,
) (int, error) {
	cat, err := catalogFor(made)
	if err != nil {
		return 0, err
	}

	to, ok := cat.DestinationAt(offTheLoopIs)
	if !ok {
		return 0, fmt.Errorf("%w: this device has no %s to send to",
			ErrNoOutput, offTheLoopIs)
	}

	return to, nil
}

// catalogFor is the catalog of the device a preset was written by.
func catalogFor(
	made plan.Plan,
) (*catalog.Catalog, error) {
	if made.Device == nil || made.Device.Id == nil {
		return catalog.BuiltIn()
	}

	cat, err := catalog.For(*made.Device.Id)
	if err != nil {
		return catalog.BuiltIn()
	}

	return cat, nil
}

// offTheLoop is the plan with its output entry sent somewhere the measuring
// lead does not reach, and its gain set.
//
// Both at once because both live in the same entry and the entry is rewritten
// whole. A caller with no output entry is told rather than left guessing: a
// plan with no output to redirect cannot be taken off the loop, and silently
// returning the one that feeds itself is how a guard comes to pass on a chain
// nobody protected.
func offTheLoop(
	made plan.Plan,
	to int,
	by float64,
) (plan.Plan, error) {
	if made.Device == nil || made.Device.Routing == nil {
		return plan.Plan{}, fmt.Errorf("%w: it carries no routing", ErrNoOutput)
	}

	routing := *made.Device.Routing

	raw, ok := routing[outputSlot]
	if !ok {
		return plan.Plan{}, fmt.Errorf("%w: no %s", ErrNoOutput, outputSlot)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return plan.Plan{}, fmt.Errorf("reading %s: %w", outputSlot, err)
	}

	send, err := json.Marshal(to)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("writing %s: %w", sendKey, err)
	}

	fields[sendKey] = send

	// Zero leaves the gain as the preset had it, so asking for no headroom
	// changes only where the chain is sent.
	if by != 0 {
		set, err := json.Marshal(by)
		if err != nil {
			return plan.Plan{}, fmt.Errorf("writing %s: %w", gainKey, err)
		}

		fields[gainKey] = set
	}

	body, err := json.Marshal(fields)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("writing %s: %w", outputSlot, err)
	}

	// A copy of the map rather than the plan's own, so a caller that keeps the
	// plan it handed over still holds what it built.
	next := make(map[string]json.RawMessage, len(routing))
	for key, was := range routing {
		next[key] = was
	}

	next[outputSlot] = body

	state := *made.Device
	state.Routing = &next
	made.Device = &state

	return made, nil
}
