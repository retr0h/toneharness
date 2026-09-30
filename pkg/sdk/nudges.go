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
	"errors"
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// Nudges turns what somebody said into figures to aim differently at.
//
// The vocabulary owns which question a word answers and which way it points,
// and the solver owns how far a tolerance is, so this is the seam between
// them: it resolves the words and hands over what to move.
//
// Every word is resolved, and every failure is reported. A nudge that is
// quietly dropped is the worst outcome available, because the run then reports
// a tone it was never asked for and nothing says the instruction was ignored.
func Nudges(
	of []tone.Nudge,
) ([]solve.Nudged, error) {
	out := []solve.Nudged(nil)
	bad := []error(nil)

	for _, one := range of {
		got, err := compile.Nudges(one.Word)
		if err != nil {
			bad = append(bad, err)

			continue
		}

		steps := 1.0
		if one.Steps != nil {
			steps = float64(*one.Steps)
		}

		for _, n := range got {
			out = append(out, solve.Nudged{
				Term:  one.Word,
				Axis:  n.Axis,
				Key:   audio.Figure(n.Key),
				Up:    n.Up,
				Steps: steps,
			})
		}
	}

	if len(bad) > 0 {
		return nil, fmt.Errorf("reading what was asked for: %w", errors.Join(bad...))
	}

	return out, nil
}
