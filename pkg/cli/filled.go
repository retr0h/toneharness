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

// Filled says which rig went into which slot, and what each write replaced.
//
// One line per slot rather than a summary, because each is a flash write on
// somebody's own pedal and the line carries the path whatever was there was
// kept at. A reply that collapses forty writes into "done" has turned forty
// recoverable changes into lost ones.
//
// A failure is printed where it happened rather than only joined into the
// returned error, so a run that wrote thirty slots and fell over on the
// thirty-first says which thirty.
func Filled(
	w io.Writer,
	placed []sdk.Placed,
) error {
	if len(placed) == 0 {
		return nil
	}

	wrote := 0

	for _, at := range placed {
		if at.Err != nil {
			if _, err := fmt.Fprintf(w, "\n%s%s %s → %s: %v\n",
				paint.Indent, paint.Failure(w, "could not"),
				at.Rig, at.At, at.Err); err != nil {
				return err
			}

			continue
		}

		wrote++

		if _, err := fmt.Fprintf(w, "\n%s%s → %s\n",
			paint.Indent, at.Rig, at.At); err != nil {
			return err
		}

		// What it replaced, by path, which is the only way back.
		if err := Change(w, at.Change); err != nil {
			return err
		}
	}

	// The count last, because it is the one thing somebody wants without
	// reading every line, and because a flash write is worth saying out loud.
	_, err := fmt.Fprintf(w, "\n%s%s\n", paint.Indent, paint.Mute(w, fmt.Sprintf(
		"%d of %d slots written, each one a flash write", wrote, len(placed))))

	return err
}
