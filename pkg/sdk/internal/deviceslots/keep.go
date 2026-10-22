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
	"errors"
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/internal/backup"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
	slotpkg "github.com/retr0h/toneharness/pkg/sdk/slot"
)

// holds returns what a slot has in it, or nothing at all.
//
// Unlike slotBytes, a slot holding nothing is an answer rather than a
// failure. Reading the destination of a write is how a backup is taken, and
// writing into an empty slot has to keep working: there is nothing there to
// lose, which is a reason to carry on rather than a reason to stop.
func holds(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
	name string,
) ([]byte, error) {
	// A slot nobody has written is not read at all. [backup.Unwritten] is the
	// rule and what the device does with it.
	//
	// Safe to skip because the backup is all the read is for. `Import` builds
	// the preset from `wire.Blank`, an unused slot the device wrote that is
	// embedded in this binary, so nothing in the write comes from the
	// destination and a slot with nothing in it has nothing to lose.
	if backup.Unwritten(name) {
		return nil, nil
	}

	body, err := s.ReadPreset(ctx, at.Setlist, at.Slot)
	if err != nil {
		return nil, fmt.Errorf("reading slot %s before replacing it: %w",
			slotpkg.Label(at.Slot), unanswered(err))
	}

	return body, nil
}

// unanswered says what a slot read's silence means, where the bare timeout
// does not.
//
// `ReadPreset`'s contract is that no bytes and no error means a slot holding
// nothing. An HX Stomp asked for a slot it has never held a preset in does not
// take that route: it says nothing at all, the read times out after six
// seconds, and **the next call on the interface stalls** so nothing can claim
// it again until the pedal is power cycled.
//
// [backup.Unwritten] keeps this off the common path by not reading such a slot
// at all, from the name the listing gives it. What it cannot cover is a slot
// the listing does not mention and one named like any other that still goes
// quiet, so the timeout stays reachable and a bare `no reply to opcode 4
// within 6s` leaves somebody with a dead interface and no idea why.
//
// Added to rather than replaced, because the opcode and the budget are what
// somebody greps for.
//
// Only silence. A refusal is the device answering, and a bus that ended says
// so already.
func unanswered(
	err error,
) error {
	if !errors.Is(err, device.ErrNoReply) {
		return err
	}

	return fmt.Errorf("%w. A slot that has never held a preset answers this "+
		"with silence rather than with nothing, and the call after it stalls "+
		"the editor interface until the pedal is powered off and on. "+
		"`slots list` says which slots hold anything", err)
}

// replacing reads what a slot holds and keeps it, for a write that is about
// to put something else there.
//
// Answers with the backup paths and whether the slot held nothing at all, so a
// caller can tell "there was nothing to keep" from "nothing was kept".
//
// The read and the keeping together, because a caller that did one without
// the other would be either reading for nothing or replacing something it
// never looked at.
func (f *Flows) replacing(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
	name string,
) ([]string, bool, error) {
	body, err := holds(ctx, s, at, name)
	if err != nil {
		return nil, false, err
	}

	// Nothing held, so nothing kept and no paths to report. A backup of an
	// unused slot would be a file whose only use is restoring an unused slot.
	//
	// Said rather than left to the empty list, which is the second answer
	// here. An absent `kept` line reads the same as a backup that did not
	// happen, and a device has no undo: see [result.Change.HeldNothing].
	if body == nil {
		return nil, true, nil
	}

	kept, err := f.backups().Keep(ctx, backup.Held{At: at, Name: name, Body: body})

	return kept, false, err
}
