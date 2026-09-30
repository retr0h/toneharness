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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// LeastTestSuite covers the solve itself, including the guards Toward makes
// unreachable.
//
// Reached directly rather than through Toward, because Toward refuses an
// all-zero system before it gets here and a ridged system always has a pivot.
// They are kept because the arithmetic below them is somebody else's to call
// next, and a guard nobody has ever executed is a guard nobody knows works.
type LeastTestSuite struct {
	suite.Suite
}

// TestTheRidgeFallsBackWhenTheScaleUnderflows covers slopes too small to scale.
//
// The ridge is relative to the diagonal's size, and a diagonal that underflows
// to zero would leave no ridge at all and divide by nothing.
func (s *LeastTestSuite) TestTheRidgeFallsBackWhenTheScaleUnderflows() {
	got, err := least([][]float64{{1e-200}}, []float64{1e-200}, Damping)

	s.Require().NoError(err)
	s.Require().Len(got, 1)
}

// TestASystemWithNoPivot covers a column of nothing.
func (s *LeastTestSuite) TestASystemWithNoPivot() {
	_, err := least([][]float64{{0}}, []float64{1}, 0)

	s.Require().ErrorIs(err, ErrSingular)
}

// TestTwoFiguresOneControl covers more rows than columns, which is the
// ordinary shape: a target names more figures than a chain has controls.
func (s *LeastTestSuite) TestTwoFiguresOneControl() {
	got, err := least([][]float64{{2}, {1}}, []float64{4, 2}, Damping)

	s.Require().NoError(err)
	s.Require().InDelta(2, got[0], 0.01, "both rows agree on two")
}

// TestThePivotIsTakenFromALaterRow covers the row swap.
//
// Reached through eliminate rather than through least, and that is the finding
// rather than a convenience. least forms the normal equations, and AᵀA is
// symmetric and positive semi-definite, for which elimination without pivoting
// is already stable: disabling the swap outright changes least's answer in the
// fifteenth digit or not at all, whatever system it is handed. Every attempt to
// catch it from the outside was a test that could not fail.
//
// So the swap is dead weight for the only caller there is today, and it is
// exercised here because eliminate's own comment says the arithmetic below it is
// somebody else's to call next. A guard nobody has ever executed is a guard
// nobody knows works.
//
// A leading entry of exactly nothing is not a singular system when a row below
// has one. Without the swap this is refused as having no pivot at all.
func (s *LeastTestSuite) TestThePivotIsTakenFromALaterRow() {
	got, err := eliminate([][]float64{{0, 1, 3}, {1, 0, 2}}, 2)

	s.Require().NoError(err, "a later row has the pivot this column needs")
	s.Require().Len(got, 2)
	s.Require().InDelta(2, got[0], 0.001)
	s.Require().InDelta(3, got[1], 0.001)
}

// TestThePivotIsTheLargestRather covers which row the swap picks.
//
// The largest remaining entry rather than the first non-zero one, so a tiny
// leading entry is not divided by when a bigger one is available.
func (s *LeastTestSuite) TestThePivotIsTheLargestRather() {
	got, err := eliminate([][]float64{{1e-11, 1, 1}, {4, 0, 8}}, 2)

	s.Require().NoError(err)
	s.Require().InDelta(2, got[0], 0.001, "the row with 4 in it led")
	s.Require().InDelta(1, got[1], 0.001)
}

// TestAnAxisWithNoToleranceBesideOneThatHasSome covers the mixed target.
//
// Toward reports every axis it was given and solves only the ones it can, so an
// unsatisfiable axis must not stop the rest being answered.
func (s *LeastTestSuite) TestAnAxisWithNoToleranceBesideOneThatHasSome() {
	knob := Knob{
		Block: 1, Param: 0, Control: "Treble", At: 0.5, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{centroid: 900, high: 10},
	}

	got, err := Toward(
		[]Knob{knob},
		map[audio.Figure]Aim{
			centroid: {Want: 1000, Tol: 10},
			high:     {Want: 50, Tol: 0},
		},
		map[audio.Figure]float64{centroid: 550, high: 1},
	)

	s.Require().NoError(err)
	s.Require().Len(got.Steps, 1, "the axis with a tolerance is still solved")
	s.Require().NotContains(got.Residual, high,
		"and the one without is not reported as wrong")
}

func TestLeastTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LeastTestSuite))
}
