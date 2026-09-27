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
// Package solve turns a target in measured figures into knob positions.
//
// The space is too big to search. One amplifier with nine controls at five
// positions each is 1,953,125 combinations, which is 226 days of measuring, so
// nothing here tries combinations. It measures what each control does on its
// own, stacks those slopes into a matrix, and solves the matrix for the moves
// that close the gap. The measuring is linear in the number of controls and the
// solving is arithmetic.
//
// The model is local and it is honest about that. A slope is true near where it
// was read and drifts away from it, so one solve overshoots and the answer is
// another pass from where the last one landed rather than a better model. The
// caller runs that loop; this package answers one step of it.
//
// See docs/superpowers/specs/2026-09-27-solving-for-knob-positions-design.md.
package solve

import (
	"errors"
	"fmt"
	"math"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// ErrNoKnobs is a chain with nothing to turn.
var ErrNoKnobs = errors.New("no control can move any figure the target names")

// Aim is one axis of a target: where to get to, and how close is close enough.
type Aim struct {
	// Want is the figure to reach, in that figure's own units.
	Want float64
	// Tol is how far off is still arrived.
	//
	// Never zero, and it does more work than it looks. It is what makes the
	// axes comparable: a centroid is hundreds of hertz and a band share is a
	// percentage, so a solver weighing them as given would spend every control
	// on the centroid and ignore the rest. Each row is divided by its own
	// tolerance, which puts them all in units of "how much of the error
	// anybody could hear".
	//
	// For a player or a genre it is the spread across their records: an axis
	// they disagree about is one the answer need not be precise on. For a
	// single recording it is the loop's own noise floor.
	Tol float64
}

// Knob is one control the solver may turn, and what turning it does.
type Knob struct {
	// Block is the position in the chain, and Param the control's index in
	// that model's own list. Together they are what a live edit addresses.
	Block int
	Param int
	// Control is what the catalog calls it, for saying what was moved.
	Control string
	// At is where it sits now. Low and High are the ends of its range, from
	// the catalog rather than assumed: 1,452 of the device's float controls do
	// not run zero to one.
	At   float64
	Low  float64
	High float64
	// Slope is how much each figure moves per unit of this control, read at
	// or near At rather than averaged over the range.
	Slope map[audio.Figure]float64
}

// Step is how far to move one control, and why.
type Step struct {
	Knob
	// By is the move, already clamped to the control's range.
	By float64
	// To is where it lands.
	To float64
}

// Result is one pass of the loop.
type Result struct {
	// Steps are the moves to apply, one per control that moved.
	Steps []Step
	// Residual is what is still wrong per axis after the moves this predicts,
	// in units of its own tolerance. Below one is arrived.
	Residual map[audio.Figure]float64
	// Arrived says every axis the target constrained is inside its tolerance.
	Arrived bool
}

// Damping is how strongly small moves are preferred where several answers fit.
//
// The system is usually underdetermined, because nine figures and a dozen
// controls means many combinations close the same gap, and the smallest is the
// one least likely to have left the range a slope was measured in. It also
// keeps the normal equations solvable when two controls do the same thing,
// which two tone stacks in one chain very nearly do.
const Damping = 1e-3

// Toward answers one pass: the moves that close the gap from got to the aims.
//
// Least squares, because there are usually more controls than constrained axes
// and no exact answer. Only the axes the target names are solved for, and an
// axis already inside its tolerance is left out rather than defended: a target
// that pins three figures and shrugs at six is the ordinary case for a genre,
// and the freedom goes into satisfying the three.
func Toward(
	knobs []Knob,
	aims map[audio.Figure]Aim,
	got map[audio.Figure]float64,
) (Result, error) {
	rows, out := wanted(aims, got)

	if len(rows) == 0 {
		return out, nil
	}

	// Rows are scaled by their own tolerance, so every residual below is in
	// the same unit and the solve does not spend the whole chain on hertz.
	a := make([][]float64, len(rows))
	b := make([]float64, len(rows))

	for i, r := range rows {
		a[i] = make([]float64, len(knobs))
		b[i] = (aims[r].Want - got[r]) / aims[r].Tol

		for j, k := range knobs {
			a[i][j] = k.Slope[r] / aims[r].Tol
		}
	}

	if allZero(a) {
		return Result{}, fmt.Errorf("%w: every slope against %v is zero",
			ErrNoKnobs, rows)
	}

	move, err := least(a, b, Damping)
	if err != nil {
		return Result{}, err
	}

	out.Steps = stepsOf(knobs, move)
	out.Residual = after(rows, aims, got, knobs, out.Steps)
	out.Arrived = arrived(out.Residual)

	return out, nil
}

// wanted is the axes still worth solving, and the state of every axis named.
//
// An axis the target does not name contributes no row: it is free, and pinning
// it to whatever it happens to read would spend controls defending a figure
// nobody asked about.
func wanted(
	aims map[audio.Figure]Aim,
	got map[audio.Figure]float64,
) ([]audio.Figure, Result) {
	out := Result{Residual: make(map[audio.Figure]float64, len(aims))}

	var rows []audio.Figure

	// In the order the figures are reported, so two runs of the same request
	// produce the same answer rather than whichever order a map yielded.
	for _, key := range audio.MeasuredKeys() {
		r := audio.Figure(key)

		aim, named := aims[r]
		if !named || aim.Tol <= 0 {
			continue
		}

		off := math.Abs(aim.Want-got[r]) / aim.Tol
		out.Residual[r] = off

		if off > 1 {
			rows = append(rows, r)
		}
	}

	out.Arrived = arrived(out.Residual)

	return rows, out
}

// stepsOf clamps each move to the control's own range and drops the ones that
// did not move.
//
// Clamped rather than refused. A solve asking for more than a control has is
// answering the right direction with the wrong magnitude, and the loop's next
// pass reads the slopes again from wherever this landed.
func stepsOf(
	knobs []Knob,
	move []float64,
) []Step {
	out := make([]Step, 0, len(knobs))

	for i, k := range knobs {
		to := math.Min(math.Max(k.At+move[i], k.Low), k.High)
		if to == k.At {
			continue
		}

		out = append(out, Step{Knob: k, By: to - k.At, To: to})
	}

	return out
}

// after is what the model says is left wrong once the steps are applied.
//
// The model's own prediction rather than a measurement, and the difference
// matters: this says whether the arithmetic thinks it has arrived, and only
// pushing the signal through the device says whether it has.
func after(
	rows []audio.Figure,
	aims map[audio.Figure]Aim,
	got map[audio.Figure]float64,
	knobs []Knob,
	steps []Step,
) map[audio.Figure]float64 {
	by := make(map[int]float64, len(steps))
	for _, s := range steps {
		by[s.Param] = s.By
	}

	out := make(map[audio.Figure]float64, len(aims))

	for r, aim := range aims {
		if aim.Tol <= 0 {
			continue
		}

		out[r] = math.Abs(aim.Want-got[r]) / aim.Tol
	}

	for _, r := range rows {
		moved := got[r]
		for _, k := range knobs {
			moved += k.Slope[r] * by[k.Param]
		}

		out[r] = math.Abs(aims[r].Want-moved) / aims[r].Tol
	}

	return out
}

// arrived is every constrained axis inside its tolerance.
func arrived(
	residual map[audio.Figure]float64,
) bool {
	for _, off := range residual {
		if off > 1 {
			return false
		}
	}

	return true
}

// allZero reports a system where nothing the solver may turn moves anything it
// was asked to move.
func allZero(
	a [][]float64,
) bool {
	for _, row := range a {
		for _, v := range row {
			if v != 0 {
				return false
			}
		}
	}

	return true
}
