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
	"errors"
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk"
)

// Hint adds the command to run next to an error somebody at a terminal can act
// on.
//
// The SDK says what went wrong and leaves the next step to whoever called it,
// because an agent over MCP calls tools rather than commands. Any other error
// comes back as it was.
func Hint(
	err error,
) error {
	switch {
	case errors.Is(err, sdk.ErrNoSuchBlock):
		return fmt.Errorf("%w, try 'toneharness catalog list'", err)
	case errors.Is(err, sdk.ErrNoSuchRig):
		return fmt.Errorf("%w, try 'toneharness rigs list'", err)
	case errors.Is(err, sdk.ErrNoDevice):
		// A pedal powered from a charger rather than a data port looks
		// exactly like one that is switched off, and the first thing anybody
		// does is check the power light, which is already on.
		return fmt.Errorf("%w: check it is in a USB data port, then "+
			"'toneharness device hardware'", err)
	case errors.Is(err, sdk.ErrBus):
		// The session is gone either way, and reopening is usually enough.
		// When it is not, the endpoint has stalled, and the wire README is
		// unambiguous about the only way out: "the interface will not be
		// claimed again until the device is power cycled." Worth saying in
		// the error, because somebody who does not know that retries into a
		// wall.
		return fmt.Errorf("%w: run it again, and if it keeps failing power "+
			"the pedal off and on", err)
	case errors.Is(err, sdk.ErrNothingToBuildFrom):
		// The next move is a question, not a guess. A recording is worth more
		// than an adjective because it is measured against every block the
		// device has, where a word has to be earned against a population of
		// players before it means anything.
		//
		// "A player somebody has researched" rather than "a player". Naming one
		// reaches their rig where there is one, and this is the answer when
		// there is not, so offering it unqualified sent people back to do what
		// they had already done.
		return fmt.Errorf("%w. Name a record to sound like, a player somebody "+
			"has researched, or the gear itself: `like: { recording: take.wav }` "+
			"resolves fully, `toneharness rigs list` says which players do, and "+
			"an adjective on its own does not", err)
	case errors.Is(err, sdk.ErrEmptySlot):
		return fmt.Errorf("%w, try 'toneharness slots list' to see which "+
			"slots hold anything", err)
	default:
		return err
	}
}
