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
package deviceslots

import (
	"context"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

// Turn moves one control on the preset a device is playing.
//
// The only operation here that changes neither what the device holds nor
// which preset it is on. Selecting swaps one stored preset for another and
// writing replaces what a slot holds; this reaches into the preset in front
// of somebody and moves a knob, exactly as a hand would.
//
// It is what a sweep is made of. Writing a preset per step does not work at
// all, because a slot given a new document goes on sounding like what it held
// before, and it would wear the device's storage out to no purpose.
func (*Flows) Turn(
	ctx context.Context,
	s device.Editor,
	at device.Address,
	value float32,
) error {
	t, ok := s.(device.Turner)
	if !ok {
		return fmt.Errorf("this session cannot move a control")
	}

	if err := t.SetParam(ctx, at, value); err != nil {
		return fmt.Errorf("setting parameter %d on block %d to %g: %w",
			at.Param, at.Block, value, err)
	}

	return nil
}

// Choose picks one of a parameter's settings, for the ones that are a list
// rather than a range.
//
// A cabinet's microphone is the one that matters most: which of eight sits in
// front of the speaker changes the sound more than any of its knobs.
//
// Separate from Turn because a device does not coerce. The value's tag is its
// type on the wire, so a parameter wanting an index refuses a float with the
// same error it gives for a block that is not there, which reads as the block
// being wrong rather than the value.
func (*Flows) Choose(
	ctx context.Context,
	s device.Editor,
	at device.Address,
	value int,
) error {
	t, ok := s.(device.Turner)
	if !ok {
		return fmt.Errorf("this session cannot move a control")
	}

	if err := t.SetChoice(ctx, at, value); err != nil {
		return fmt.Errorf("setting parameter %d on block %d to choice %d: %w",
			at.Param, at.Block, value, err)
	}

	return nil
}

// Switch turns one of a parameter's switches on or off.
//
// The same refusal applies in the other direction: an amplifier's Bright is a
// switch and declines 1.0 where it wants true.
func (*Flows) Switch(
	ctx context.Context,
	s device.Editor,
	at device.Address,
	on bool,
) error {
	t, ok := s.(device.Turner)
	if !ok {
		return fmt.Errorf("this session cannot move a control")
	}

	if err := t.SetSwitch(ctx, at, on); err != nil {
		return fmt.Errorf("setting parameter %d on block %d to %t: %w",
			at.Param, at.Block, on, err)
	}

	return nil
}
