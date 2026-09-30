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

package solve

import (
	"maps"
	"math"
	"slices"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// Span is what a chain was measured doing on one axis, in the target's units.
type Span struct {
	// From is where the chain sits, and Low and High the ends of every
	// reading ever taken of its blocks.
	From, Low, High float64
	// Swing is the most its controls could move the axis.
	Swing float64
}

// Verdict is what one axis of a target is worth attempting.
type Verdict struct {
	// Figure is the axis, Want where the target sits and Tol how close counts
	// as arriving.
	Figure audio.Figure `json:"figure"`
	Want   float64      `json:"want"`
	Tol    float64      `json:"tolerance"`
	// From is where the chain reads before anything moves, and Gap how far
	// that is from Want in units of Tol. The same number the loop reports and
	// stops on, so "2.4 tolerances out" means one thing everywhere.
	From float64 `json:"from"`
	Gap  float64 `json:"gap"`
	// Swing is the most the chain's controls could move this axis, in the
	// same tolerances.
	Swing float64 `json:"swing"`
	// Met is a chain already inside the tolerance.
	//
	// Shown is the one that can be trusted: some reading actually taken of
	// these blocks landed where the target wants, so a setting exists that
	// gets there and the only question is finding it. No slope, no model, no
	// assumption that controls add.
	//
	// Within is the weaker claim, and it is the one to read sceptically. The
	// gap is smaller than the controls' summed movement, which assumes every
	// slope holds across a range it was read locally and that every control
	// pulls the same way. It refuses well and promises badly: outside it is
	// out of reach, inside it means only that nothing rules the target out.
	Met    bool `json:"met"`
	Shown  bool `json:"shown"`
	Within bool `json:"within"`
	// Together is what is left on this axis once every axis is solved at
	// once, in the same tolerances. The number that answers the question
	// somebody meant.
	Together float64 `json:"together"`
	// Straight is how straight the slopes behind Swing were, at their worst.
	// Near zero, Swing is arithmetic on a number describing nothing.
	Straight float64 `json:"straight"`
}

// Reachable says which axes of a target are worth spending readings on.
//
// The loop can already report a chain that will not reach its target. It does
// it by running: three to five passes, one reading per control per pass, five
// minutes of real-time audio with the pedal held throughout, and the answer at
// the end is a sentence. This answers the same question from readings already
// taken, before anything is plugged in.
//
// Three outcomes and they mean different things:
//
//   - Met, and no reading need be spent on this axis at all.
//   - Shown, and a reading actually taken of these blocks landed where the
//     target wants. A setting exists; the loop's job is to find it. This is the
//     claim worth trusting, because nothing was modelled to make it.
//   - Within, so the gap is smaller than the controls' summed movement. Not a
//     promise: that sum assumes every control pulls the same way, which they do
//     not. It says only that nothing rules the target out.
//   - Neither, so the gap is wider than everything the chain has, added up
//     under a model that flatters it. No arrangement of these controls reaches
//     this target, and that is an answer about the gear rather than a failure.
//     It is the useful one: the gear is wrong for the sound is a real thing to
//     tell somebody who owns that gear.
//
// of is per figure and in the units the aims are stated in. The caller
// converts, because a sweep reports a band share as a percentage and a corpus
// reports the same share as a fraction, and a comparison across those two is
// out by a hundred.
//
// Sorted worst first, so the axis that decides the run reads first.
func Reachable(
	aims map[audio.Figure]Aim,
	of map[audio.Figure]Span,
) []Verdict {
	out := make([]Verdict, 0, len(aims))

	for _, figure := range slices.Sorted(maps.Keys(aims)) {
		aim := aims[figure]
		if aim.Tol <= 0 {
			continue
		}

		span, read := of[figure]
		if !read {
			continue
		}

		gap := math.Abs(aim.Want-span.From) / aim.Tol
		reach := span.Swing / aim.Tol

		// The tolerance widens the target at both ends, because a chain has to
		// be carried to the edge of what counts as arriving rather than to its
		// centre.
		shown := span.Low-aim.Tol <= aim.Want && aim.Want <= span.High+aim.Tol

		out = append(out, Verdict{
			Figure: figure,
			Want:   aim.Want,
			Tol:    aim.Tol,
			From:   span.From,
			Gap:    gap,
			Swing:  reach,
			Met:    gap <= 1,
			Shown:  shown,
			Within: gap <= 1 || shown || reach >= gap-1,
		})
	}

	worstFirst(out)

	return out
}

// Worth says whether a whole target is worth attempting, and which axis
// decides it.
//
// The worst axis, because a target is met only when every axis it names is,
// so one unreachable axis is an unreachable target however comfortable the
// others are.
func Worth(
	verdicts []Verdict,
) (Verdict, bool) {
	if len(verdicts) == 0 {
		return Verdict{}, false
	}

	for _, v := range verdicts {
		if !v.Within {
			return v, false
		}
	}

	return verdicts[0], true
}

// worstFirst puts the axis furthest out of reach first, then the furthest out.
//
// Out of reach before merely far, because they are different answers. A wide
// gap the controls can close is work; a narrow one they cannot is a wall, and
// reading the wall first is what stops somebody tuning for an afternoon.
func worstFirst(
	of []Verdict,
) {
	slices.SortFunc(of, func(a, b Verdict) int {
		if a.Within != b.Within {
			if a.Within {
				return 1
			}

			return -1
		}

		switch {
		case a.Gap > b.Gap:
			return -1
		case a.Gap < b.Gap:
			return 1
		}

		return int(a.Figure[0]) - int(b.Figure[0])
	})
}

// Best is the nearest the model says a chain can get, with every axis solved
// together.
//
// Reachable checks each axis on its own, and that turned out to rule nothing
// out: across five shipped rigs and three measured genres every axis came back
// reachable, every time. It is not a surprising result once stated. A
// cabinet's Distance can put the centroid where a target wants it, and can put
// the low band where the target wants it, and those are two different
// positions of the same dial. Nine axes each reachable alone says almost
// nothing about nine reachable at once, which is the question somebody is
// actually asking.
//
// So this solves them together, which is what the loop does, using the slopes
// already committed instead of reading fresh ones off a device. It is the same
// arithmetic Toward performs on hardware: least squares over every constrained
// axis at once, clamped to what each control actually has.
//
// Iterated because one pass clamps. A solve wanting more of a control than it
// has takes what there is, and the axes that control was carrying are then
// short by the remainder, which the next pass spends other controls on. The
// slopes are fixed here, so this converges rather than wandering: it is the
// same projection repeated, and it stops as soon as a pass stops improving.
//
// What it is not is a promise. The slopes are local and were read one control
// at a time in whatever chain the sweep ran, so a residual this says is
// reachable may not be. The direction of the error is the useful one: a
// residual the model cannot close with every control pulling together is one
// the hardware will not close either.
func Best(
	knobs []Knob,
	aims map[audio.Figure]Aim,
	from map[audio.Figure]float64,
	passes int,
) (Result, error) {
	at := make(map[audio.Figure]float64, len(from))
	maps.Copy(at, from)

	_, out := wanted(aims, at)

	for range passes {
		step, err := Toward(knobs, aims, at)
		if err != nil {
			return out, err
		}

		if len(step.Steps) == 0 {
			break
		}

		if Worst(step.Residual) >= Worst(out.Residual) {
			// Clamping can make a pass worse than the one before it, and the
			// answer wanted here is the nearest the chain got rather than
			// wherever the last pass landed.
			break
		}

		out = step

		// Back into figures, because a residual is in tolerances and the next
		// pass solves from a reading. Into a fresh map rather than over
		// step.Residual, which is the Result's own: aliasing it overwrote the
		// answer with the readings it was computed from, and a centroid of 141
		// hertz was reported as 141 tolerances out.
		//
		// Signed the way the gap was, so an axis overshot stays overshot.
		next := make(map[audio.Figure]float64, len(at))
		maps.Copy(next, at)

		for key, aim := range aims {
			if aim.Tol <= 0 {
				continue
			}

			if aim.Want < at[key] {
				next[key] = aim.Want + step.Residual[key]*aim.Tol
			} else {
				next[key] = aim.Want - step.Residual[key]*aim.Tol
			}
		}

		at = next

		for i := range knobs {
			for _, s := range step.Steps {
				if knobs[i].Where() == s.Where() {
					knobs[i].At = s.To
				}
			}
		}

		if step.Arrived {
			break
		}
	}

	return out, nil
}
