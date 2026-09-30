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
	"os"
	"time"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// record appends what a round of tuning asked for and what it moved to the ask.
//
// The only place a round of correction is written down. Everything else this
// project holds is a measurement or an assertion; a correction is the record of
// somebody asking for something and something moving in answer, and it is the
// one input that cannot be recovered later if it is not captured when it
// happens.
//
// The verdict is deliberately left absent. It means not yet evaluated, which is
// exactly the state after a chain has been tuned and before anybody has heard
// it, and that state is worth showing rather than leaving to be rediscovered.
// Nothing here can hear, so only a person fills it in.
//
// Append-only and never replayed. The plan holds the current settings; this
// holds why they are what they are.
func record(
	at string,
	asked string,
	steps []solve.Step,
	residual map[audio.Figure]float64,
	arrived bool,
) error {
	f, err := os.Open(at) //nolint:gosec // a path the caller named
	if err != nil {
		return fmt.Errorf("reading %s: %w", at, err)
	}

	spec, err := tone.Load(f)

	_ = f.Close()

	if err != nil {
		return fmt.Errorf("reading %s: %w", at, err)
	}

	was := []tone.Correction(nil)
	if spec.Corrections != nil {
		was = *spec.Corrections
	}

	was = append(was, correction(asked, steps, residual, arrived))
	spec.Corrections = &was

	out, err := os.Create(at) //nolint:gosec // the file just read
	if err != nil {
		return fmt.Errorf("writing %s: %w", at, err)
	}

	if err := tone.Write(out, spec); err != nil {
		_ = out.Close()

		return fmt.Errorf("writing %s: %w", at, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", at, err)
	}

	return nil
}

// correction is one round, as the ask records it.
func correction(
	asked string,
	steps []solve.Step,
	residual map[audio.Figure]float64,
	arrived bool,
) tone.Correction {
	when := time.Now().Format(time.DateOnly)
	why := reasonFor(residual, arrived)

	changed := make([]tone.Change, 0, len(steps))

	for _, s := range steps {
		// A path into the plan this moved, which is what a Change is. The
		// block by its position and the control by the name the catalog gives
		// it, because an index on the wire means nothing to a later reader.
		at := fmt.Sprintf("blocks[%d].params.%s", s.Block, s.Setting)

		from, to := s.At, s.To
		changed = append(changed, tone.Change{
			Path: at,
			From: &from,
			To:   &to,
		})
	}

	return tone.Correction{
		Ask:     asked,
		At:      &when,
		Changed: &changed,
		Reason:  &why,
	}
}

// reasonFor says how the solve read the ask, in the figures it was solved
// against.
//
// The worst axis and how far out it finished, because that is what a later
// reader needs to judge the interpretation rather than only the values: a round
// that arrived and a round that ran out of room look identical in the settings
// alone.
func reasonFor(
	residual map[audio.Figure]float64,
	arrived bool,
) string {
	worst, axis := 0.0, audio.Figure("")

	for key, off := range residual {
		if off > worst {
			worst, axis = off, key
		}
	}

	if arrived {
		return fmt.Sprintf(
			"solved against the measured target; every axis inside its "+
				"tolerance, %s the furthest at %.2f", axis, worst)
	}

	return fmt.Sprintf(
		"solved against the measured target and did not reach it; %s is %.2f "+
			"tolerances out", axis, worst)
}
