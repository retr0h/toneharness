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
	"fmt"
	"io"

	"github.com/retr0h/toneharness/pkg/cli/internal/paint"
	"github.com/retr0h/toneharness/pkg/sdk"
)

// Chosen says which favourite is now playing, and where the step came from.
//
// Both ends of the move, because a cycle is only readable if it says what it
// moved away from: "19A of 3" leaves somebody counting, and "11A → 19A, 2 of
// 3" does not.
//
// Where nothing was recognised it says so rather than staying quiet. A pedal
// playing something that is not a favourite lands on the first one, and a
// person who asked for next and got the first is owed the reason.
func Chosen(
	w io.Writer,
	c sdk.Chosen,
) error {
	if c.From != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s → %s\n",
			paint.Indent, paint.Mute(w, c.From),
			paint.Accent(w, c.At)); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "\n%s%s\n",
			paint.Indent, paint.Accent(w, c.At)); err != nil {
			return err
		}
	}

	if c.Rig != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s\n", paint.Indent, c.Rig); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n%s%s\n", paint.Indent, paint.Mute(w,
		fmt.Sprintf("favourite %d of %d", c.Position, c.Of))); err != nil {
		return err
	}

	// Why it landed where it did, when that was not where the step pointed.
	if c.From == "" && c.Playing != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s\n", paint.Indent, paint.Info(w,
			fmt.Sprintf("the pedal was playing %s, which no favourite holds, "+
				"so this went to the first", c.Playing))); err != nil {
			return err
		}
	}

	return Change(w, c.Change)
}
