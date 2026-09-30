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
	"io"
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
// **The gain is the fix. The destination does nothing, and that is measured.**
//
// Read on matt-freeman on 2026-09-29, one destination per preset, gain at 0, the
// share of energy above 2kHz against a reference carrying 0.01%:
//
//	0  None                                  0.0%   -186.7dB
//	1  Multi (1/4", XLR, Digital, USB 1/2)   84.4%    -23.6dB
//	10 USB 1/2                               84.2%    -23.6dB
//	11 USB 3/4                               84.4%    -23.6dB
//
// The decisive pair is 10 against 11. These readings are taken on USB 1/2, so a
// chain genuinely sent to USB 3/4 alone would read silence. It reads the same as
// USB 1/2 does. **On an HX Stomp this enum does not route**: every non-zero
// destination sends the chain everywhere, the quarter-inch socket included, and
// only None silences it. So the measuring lead always carries the chain back to
// the input whatever this says, and the loop cannot be opened from here.
//
// Turning `dsp0.outputA.gain` down is what makes a reading clean. It drops the
// loop below unity without touching the tone, which ChVol and Master are. At the
// -30dB default the same chain reads 0.03%, which is the empty loop's own figure.
//
// The destination is still set, for two reasons and neither is that it helps.
// `references/signal-path.md` records an earlier measurement where moving off
// Multi took the USB floor from -123.4 to -118.6dBFS, which disagrees with the
// table above and has not been explained. And a firmware that started honouring
// the enum would want it right. It is documented as doing nothing so nobody
// spends another afternoon believing it.
//
// Two earlier versions of this comment were wrong in opposite directions: the
// first said the destination was the fix and worked at any gain, the second that
// it cut the loop from 84.3% to 57.5%. The second figure was one run of an
// oscillation wandering, not an improvement. Both were written before the
// experiment above.
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
// by is in decibels and wants to be negative. Zero asks for no headroom and the
// preset is still rewritten, which costs a compile and buys nothing measurable:
// the destination it sets is measured to make no difference. It stays for the
// reasons above rather than because zero is a case worth serving.
//
// **Zero does not make a reading safe.** The same chain oscillates at 84.2%
// above 2kHz with no headroom, on any destination that is not None.
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

// asCompiled is the plan the device is playing with the measuring rig's own
// output entry put back to what the compiler produced.
//
// Everything else is kept. The dials are what the solve decided and the whole
// point of the run; the output entry is the rig it was measured on. A tuned plan
// written out with it still in place is sent to USB alone, so it is silent at the
// quarter-inch socket, and carries the headroom trim, so it is 30dB quiet
// wherever it is not.
//
// Read off the compiled preset rather than remembered, because that file is what
// the rig actually resolved to: its pan, its model and whatever gain the rig
// itself asked for are the values to restore, and reconstructing them would be
// guessing at a default.
//
// A compiled preset that cannot be read, or that carries no output entry, leaves
// the plan alone and says so. Refusing would throw away a tuning run over the
// one entry nobody listens to, and staying quiet would hand somebody a plan that
// is silent at the quarter-inch socket without telling them why.
//
// The writer is there for that one line. A function that can leave its answer
// wrong in a way nothing downstream detects has to be able to say it did.
func asCompiled(
	ctx context.Context,
	w io.Writer,
	client ReadsFiles,
	playing plan.Plan,
	asBuilt string,
) (plan.Plan, error) {
	if playing.Device == nil || playing.Device.Routing == nil {
		return playing, nil
	}

	was, err := client.PresetFile(ctx, asBuilt)
	if err != nil {
		stillMeasuring(w, fmt.Sprintf("%s could not be read back", asBuilt))

		return playing, nil //nolint:nilerr // the run is worth more than the entry
	}

	if was.Plan.Device == nil || was.Plan.Device.Routing == nil {
		stillMeasuring(w, "the compiled preset carries no routing")

		return playing, nil
	}

	entry, ok := (*was.Plan.Device.Routing)[outputSlot]
	if !ok {
		stillMeasuring(w, "the compiled preset carries no "+outputSlot)

		return playing, nil
	}

	// A copy, so a caller still holding the plan it read keeps what it read.
	next := make(map[string]json.RawMessage, len(*playing.Device.Routing))
	for key, body := range *playing.Device.Routing {
		next[key] = body
	}

	next[outputSlot] = entry

	state := *playing.Device
	state.Routing = &next
	playing.Device = &state

	return playing, nil
}

// stillMeasuring warns that a kept plan carries the measuring rig's own output.
//
// Which means it is sent to USB alone, so it is silent at the quarter-inch
// socket, and carries the headroom trim, so it is quiet everywhere else. Every
// dial the solve found is still in it, so the plan is worth keeping and worth
// fixing by hand rather than throwing away.
func stillMeasuring(
	w io.Writer,
	because string,
) {
	_, _ = fmt.Fprintf(w,
		"\n  [warn] %s, so this plan keeps the output entry it was measured\n"+
			"         through: sent to %s alone and turned down. It will be quiet\n"+
			"         and silent at the quarter-inch socket until that entry is put\n"+
			"         back. The dials the solve found are unaffected.\n",
		because, offTheLoopIs)
}
