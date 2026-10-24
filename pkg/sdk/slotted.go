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
package sdk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/retr0h/toneharness/pkg/sdk/slot"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// ErrNoSlots is returned when nothing says which slot anything goes in.
var ErrNoSlots = errors.New("the setup names no slots")

// Placed is one rig written to one slot.
type Placed struct {
	// At is the slot, as the pedal prints it.
	At string `json:"at"`
	// Rig is the identifier of what went there.
	Rig string `json:"rig"`
	// Change is what the write did, including where whatever the slot held
	// was kept.
	Change Change `json:"change"`
	// Err is why this one did not happen, where it did not. The rest are
	// still attempted: a rig that will not build is not a reason to leave
	// somebody's pedal half updated.
	//
	// Not serialised, because an error has no fields: marshalling one gives
	// `{}` and an agent reading that learns a rig failed and nothing about
	// why. [Placed.Why] is the same thing in a form that survives the wire,
	// and both are set together.
	Err error `json:"-"`
	// Why is Err as text, for a caller that is not holding the error.
	//
	// Empty where nothing went wrong, so a reader can test either field.
	Why string `json:"why,omitempty"`
}

// Fill writes every rig a setup places into the slot it names.
//
// The thing that makes "update my pedal" askable twice. A slot map lives on
// the Setup because it is a preference about somebody's own pedal and nothing
// anybody else would share: two people building one rig want the same preset
// and different slots, since one of them already keeps their own work in bank
// one. Without it the layout is remembered by whoever ran it last, and a
// second run either lands somewhere else or has to be told again.
//
// Every slot is a flash write and a device has no undo, so each one keeps what
// it held and the answer says where that went. The caller is expected to
// repeat those paths: a reply that drops them has turned a recoverable change
// into a lost one.
//
// **One rig failing does not stop the rest.** A rig that will not build is a
// document to fix, and leaving the other forty written is better than leaving
// a pedal half updated with no record of which half. Each failure travels on
// its own entry and the error returned at the end joins them, so a caller that
// only checks the error still learns something went wrong.
func (c *Client) Fill(
	ctx context.Context,
) ([]Placed, error) {
	held, err := c.setup()
	if err != nil {
		return nil, err
	}

	if held.Slots == nil || len(*held.Slots) == 0 {
		return nil, ErrNoSlots
	}

	// Somewhere to build each preset on the way to the pedal. A temporary
	// directory rather than beside the rigs, because these are a means of
	// getting bytes to flash rather than something to keep: what is worth
	// keeping is the rig, which is already written down.
	work, err := os.MkdirTemp("", "toneharness-fill")
	if err != nil {
		return nil, fmt.Errorf("making somewhere to build the presets: %w", err)
	}

	defer func() { _ = os.RemoveAll(work) }()

	out := make([]Placed, 0, len(*held.Slots))
	faults := []error(nil)

	for _, at := range *held.Slots {
		placed := c.fillOne(ctx, at, work)
		out = append(out, placed)

		if placed.Err != nil {
			faults = append(faults, placed.Err)
		}
	}

	return out, errors.Join(faults...)
}

// fillOne builds one rig and writes it to its slot.
func (c *Client) fillOne(
	ctx context.Context,
	at tone.Slotted,
	work string,
) Placed {
	placed := Placed{At: at.At, Rig: at.Rig}

	index, err := slot.Parse(at.At)
	if err != nil {
		return placed.failed(
			fmt.Errorf("slot %q for %q: %w", at.At, at.Rig, err))
	}

	built := filepath.Join(work, at.Rig+".hlx")

	if _, err := c.Make(ctx, Build{RigID: at.Rig, Out: built}); err != nil {
		return placed.failed(
			fmt.Errorf("building %q for slot %s: %w", at.Rig, at.At, err))
	}

	// Setlist zero, the one a Stomp shows, because a setup naming a slot the
	// way the pedal prints it is naming a position in what is on screen.
	change, err := c.Import(ctx, built, slot.Address{Slot: index})
	if err != nil {
		return placed.failed(
			fmt.Errorf("writing %q to slot %s: %w", at.Rig, at.At, err))
	}

	placed.Change = change

	return placed
}

// failed records why one slot did not happen, in both the forms it travels
// in.
//
// One call rather than two assignments at each of the three places a fill can
// fall over, because the two have to agree and a caller that reads only the
// serialised one would otherwise read an empty string for a real failure.
func (p Placed) failed(
	err error,
) Placed {
	p.Err = err
	p.Why = err.Error()

	return p
}

// setup reads what the person has, which is where their slot map lives.
func (c *Client) setup() (tone.Setup, error) {
	if c.opts.setup == "" {
		return tone.Setup{}, fmt.Errorf(
			"%w: name one with --setup, which is where a slot map lives",
			ErrNoSlots)
	}

	f, err := os.Open(c.opts.setup) //nolint:gosec // the path is the user's own
	if err != nil {
		return tone.Setup{}, fmt.Errorf("reading %s: %w", c.opts.setup, err)
	}

	defer func() { _ = f.Close() }()

	return tone.LoadSetup(f)
}
