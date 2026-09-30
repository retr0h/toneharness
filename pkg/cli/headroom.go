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
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// This file lowers the one gain that is not part of the tone.
//
// The measuring rig is a lead from the pedal's output back into its own input,
// and the chain's output destination is `Multi (1/4", XLR, Digital, USB 1/2)`,
// which drives the socket that lead comes from. So the chain feeds itself, and
// with enough gain around that it oscillates: matt-freeman read 84.3% of its
// energy above 2kHz where the reference has 0.01%, and the same chain with the
// amplifier turned down read 0.0% for three decibels less level.
//
// Turning the amplifier down is the wrong fix. ChVol and Master are the tone,
// so measuring with them lowered measures a different sound from the one the
// rig describes, and the solver would then solve for that one.
//
// `dsp0.outputA.gain` sits after the whole chain. Lowering it drops what
// reaches the quarter-inch socket and what reaches the computer by the same
// amount, so the loop gain falls below unity while the spectrum stays the
// chain's own.

// outputSlot is the routing entry the chain's output sits in, and gainKey the
// parameter inside it.
//
// Named rather than indexed. An output entry sends two values and the device
// records them in its model's order, which routeValues reads by name from the
// plan rather than by position.
const (
	outputSlot = "dsp0.outputA"
	gainKey    = "gain"
)

// ErrNoOutput is a preset with no output entry to turn down.
var ErrNoOutput = errors.New("nothing to give headroom to")

// Quiets is what lowering a preset's output takes: read the preset back for
// the routing it arrived with, and write it again.
type Quiets interface {
	Compiles
	ReadsFiles
}

// quieter writes the plan again with the chain's output gain lowered, and
// returns the preset compiled from it.
//
// A round trip through a plan rather than a live edit, because routing is not
// a block: `device turn` addresses a block and a parameter, and an output
// entry is neither. What a preset says about its routing is settled when the
// preset is written.
//
// by is in decibels and wants to be negative. Zero leaves the preset alone and
// returns the one already built, which is what a caller asking for no headroom
// means.
func quieter(
	ctx context.Context,
	client Quiets,
	already, work, name string,
	by float64,
) (string, error) {
	if by == 0 {
		return already, nil
	}

	// Read back off the preset that was just built, because a plan compiled
	// from a rig carries no routing at all: the inputs and outputs come from
	// the blank template while the preset is written, so there is nothing in
	// the plan to change. The preset has them, and reading it back is how the
	// entry arrives complete, with its model and its output already set.
	read, err := client.PresetFile(ctx, already)
	if err != nil {
		return "", err
	}

	lowered, err := withGain(read.Plan, by)
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

// withGain is the plan with its output entry's gain set.
//
// The entry is left alone when the plan carries none, and a caller is told
// rather than left guessing: a plan with no output to turn down cannot be given
// headroom, and silently returning the loud one is how a guard comes to pass on
// a chain nobody protected.
func withGain(
	made plan.Plan,
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

	set, err := json.Marshal(by)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("writing %s: %w", gainKey, err)
	}

	fields[gainKey] = set

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
