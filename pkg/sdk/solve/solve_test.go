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
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// SolveTestSuite covers turning a target into moves.
//
// All of it is arithmetic, so none of it needs a device. What a device decides
// is whether the slopes were true, which is the loop's job rather than this
// one's.
type SolveTestSuite struct {
	suite.Suite
}

const (
	centroid = audio.Figure("centroid")
	low      = audio.Figure("low")
	high     = audio.Figure("high")
)

// bass is a control that lifts the low band and pulls the centre down.
func (s *SolveTestSuite) bass() Knob {
	return Knob{
		Block: 1, Param: 0, Control: "Bass", At: 0.5, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{low: 20, centroid: -400},
	}
}

// treble is the other end of the same tone stack.
func (s *SolveTestSuite) treble() Knob {
	return Knob{
		Block: 1, Param: 1, Control: "Treble", At: 0.5, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{high: 15, centroid: 900},
	}
}

// TestOneAxisOneControl covers the simplest case there is.
func (s *SolveTestSuite) TestOneAxisOneControl() {
	got, err := Toward(
		[]Knob{s.treble()},
		map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
		map[audio.Figure]float64{centroid: 550},
	)

	s.Require().NoError(err)
	s.Require().Len(got.Steps, 1)
	s.Require().Equal("Treble", got.Steps[0].Control)
	s.Require().InDelta(0.5, got.Steps[0].By, 0.01,
		"450Hz to close at 900Hz per turn is half a turn")
	s.Require().True(got.Arrived)
}

// TestAnAxisAlreadyInsideItsToleranceIsLeftAlone covers the target that is met.
//
// Left alone rather than defended. Every row costs controls, and a genre target
// pins three figures and shrugs at six by design.
func (s *SolveTestSuite) TestAnAxisAlreadyInsideItsToleranceIsLeftAlone() {
	got, err := Toward(
		[]Knob{s.treble()},
		map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 100}},
		map[audio.Figure]float64{centroid: 960},
	)

	s.Require().NoError(err)
	s.Require().Empty(got.Steps, "nothing to do")
	s.Require().True(got.Arrived)
	s.Require().InDelta(0.4, got.Residual[centroid], 0.001)
}

// TestAnAxisTheTargetDoesNotNameIsFree covers a partial target.
func (s *SolveTestSuite) TestAnAxisTheTargetDoesNotNameIsFree() {
	got, err := Toward(
		[]Knob{s.bass(), s.treble()},
		map[audio.Figure]Aim{low: {Want: 98, Tol: 1}},
		map[audio.Figure]float64{low: 90, centroid: 150, high: 1},
	)

	s.Require().NoError(err)
	s.Require().NotContains(got.Residual, centroid,
		"a figure nobody asked about is not reported as wrong")
	s.Require().NotContains(got.Residual, high)
}

// TestTwoAxesTwoControls covers the case the whole package exists for.
//
// The controls disagree: Bass pulls the centre down while lifting the low band,
// Treble pushes it up. A word at a time cannot answer this, and a least-squares
// solve does it in one pass.
func (s *SolveTestSuite) TestTwoAxesTwoControls() {
	got, err := Toward(
		[]Knob{s.bass(), s.treble()},
		map[audio.Figure]Aim{
			low:      {Want: 96, Tol: 0.5},
			centroid: {Want: 200, Tol: 5},
		},
		map[audio.Figure]float64{low: 90, centroid: 150, high: 1},
	)

	s.Require().NoError(err)
	s.Require().Len(got.Steps, 2, "both ends of the stack moved")

	by := map[string]float64{}
	for _, st := range got.Steps {
		by[st.Control] = st.By
	}

	s.Require().Positive(by["Bass"], "the low band has to come up")
	s.Require().Positive(by["Treble"],
		"and the centre has to rise despite Bass pulling it down")

	for r, off := range got.Residual {
		s.Require().Less(off, 1.0, "%s is inside its tolerance", r)
	}

	s.Require().True(got.Arrived)
}

// TestAMoveIsClampedToTheControlsRange covers a target past what a knob has.
func (s *SolveTestSuite) TestAMoveIsClampedToTheControlsRange() {
	got, err := Toward(
		[]Knob{s.treble()},
		map[audio.Figure]Aim{centroid: {Want: 100000, Tol: 10}},
		map[audio.Figure]float64{centroid: 550},
	)

	s.Require().NoError(err)
	s.Require().Len(got.Steps, 1)
	s.Require().InDelta(1, got.Steps[0].To, 0.0001, "hard against its top stop")
	s.Require().False(got.Arrived, "and it says it did not get there")
	s.Require().Greater(got.Residual[centroid], 1.0)
}

// TestARangeThatIsNotZeroToOne covers a control with its own span.
//
// 1,452 of the device's float controls do not run zero to one, and one swept as
// though it did never leaves its bottom stop.
func (s *SolveTestSuite) TestARangeThatIsNotZeroToOne() {
	knob := Knob{
		Block: 1, Param: 3, Control: "MidFreq", At: 500, Low: 125, High: 4000,
		Slope: map[audio.Figure]float64{centroid: 0.5},
	}

	got, err := Toward(
		[]Knob{knob},
		map[audio.Figure]Aim{centroid: {Want: 700, Tol: 5}},
		map[audio.Figure]float64{centroid: 500},
	)

	s.Require().NoError(err)
	s.Require().Len(got.Steps, 1)
	s.Require().InDelta(900, got.Steps[0].To, 1,
		"400Hz of centroid at half a hertz per hertz is 800 of travel")
}

// TestAControlThatMovesNothing covers the row of zeroes.
//
// MidFreq at a flat MidGain is the real case: it says where the mid band sits
// and there is no boost for it to move, so its whole row is zero. A solver that
// took that for an inert control would never touch it again, and it is not
// inert, it is conditional.
func (s *SolveTestSuite) TestAControlThatMovesNothing() {
	knob := Knob{
		Block: 1, Param: 3, Control: "MidFreq", At: 0.5, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{centroid: 0},
	}

	_, err := Toward(
		[]Knob{knob},
		map[audio.Figure]Aim{centroid: {Want: 700, Tol: 5}},
		map[audio.Figure]float64{centroid: 500},
	)

	s.Require().ErrorIs(err, ErrNoKnobs)
}

// TestNoAxisNamed covers a target that constrains nothing.
//
// Which is what a genre earning no words would give, and punk is one. Answered
// as arrived rather than refused: nothing was asked for and nothing is wrong.
func (s *SolveTestSuite) TestNoAxisNamed() {
	got, err := Toward(
		[]Knob{s.treble()},
		map[audio.Figure]Aim{},
		map[audio.Figure]float64{centroid: 550},
	)

	s.Require().NoError(err)
	s.Require().Empty(got.Steps)
	s.Require().True(got.Arrived)
}

// TestAToleranceOfZeroIsNotAConstraint covers an axis nobody can satisfy.
//
// Zero tolerance means arrived is unreachable and every pass would keep
// spending controls on it, so it is read as an axis nobody stated.
func (s *SolveTestSuite) TestAToleranceOfZeroIsNotAConstraint() {
	got, err := Toward(
		[]Knob{s.treble()},
		map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 0}},
		map[audio.Figure]float64{centroid: 550},
	)

	s.Require().NoError(err)
	s.Require().Empty(got.Steps)
	s.Require().NotContains(got.Residual, centroid)
}

// TestTwoControlsThatDoTheSameThing covers the singular system.
//
// Two tone stacks in one chain come close enough that the difference is
// rounding. The ridge is what keeps it solvable, and it splits the move between
// them rather than picking one.
func (s *SolveTestSuite) TestTwoControlsThatDoTheSameThing() {
	one := s.treble()
	two := s.treble()
	two.Param = 9
	two.Control = "Treble2"

	got, err := Toward(
		[]Knob{one, two},
		map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
		map[audio.Figure]float64{centroid: 550},
	)

	s.Require().NoError(err)
	s.Require().Len(got.Steps, 2)
	s.Require().InDelta(got.Steps[0].By, got.Steps[1].By, 0.001,
		"neither is preferred, so the move is shared")

	total := got.Steps[0].By + got.Steps[1].By
	s.Require().InDelta(0.5, total, 0.01, "and together they close the gap")
}

// TestSmallMovesArePreferred covers what the ridge is for.
func (s *SolveTestSuite) TestSmallMovesArePreferred() {
	weak := s.treble()
	weak.Control = "Fine"
	weak.Slope = map[audio.Figure]float64{centroid: 10}

	strong := s.treble()
	strong.Param = 9
	strong.Control = "Coarse"

	got, err := Toward(
		[]Knob{weak, strong},
		map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
		map[audio.Figure]float64{centroid: 550},
	)

	s.Require().NoError(err)

	by := map[string]float64{}
	for _, st := range got.Steps {
		by[st.Control] = math.Abs(st.By)
	}

	s.Require().Greater(by["Coarse"], by["Fine"],
		"the control that moves the figure furthest per turn does the work")
}

// TestTwoBlocksSharingAParamIndex covers the identity of a knob.
//
// Every block in a chain has a parameter 0, so a param index alone names
// nothing. Keying on it credited one block's move to another block's slope,
// which did not fail: it predicted a centroid of 2.8MHz, and the loop concluded
// from that prediction that the chain could not reach the target.
//
// The two knobs here are deliberately opposed. If their moves are swapped the
// prediction lands the wrong side of the target, so the residual catches it.
func (s *SolveTestSuite) TestTwoBlocksSharingAParamIndex() {
	amp := Knob{
		Block: 1, Param: 0, Control: "Amp Treble", At: 0.5, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{centroid: 1000},
	}
	cab := Knob{
		Block: 2, Param: 0, Control: "Cab HighCut", At: 0.5, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{centroid: -200},
	}

	got, err := Toward(
		[]Knob{amp, cab},
		map[audio.Figure]Aim{centroid: {Want: 900, Tol: 5}},
		map[audio.Figure]float64{centroid: 500},
	)

	s.Require().NoError(err)
	s.Require().Len(got.Steps, 2)

	where := map[Where]string{}
	for _, st := range got.Steps {
		where[st.Where()] = st.Control
	}

	s.Require().Equal("Amp Treble", where[Where{Block: 1, Param: 0}])
	s.Require().Equal("Cab HighCut", where[Where{Block: 2, Param: 0}])

	s.Require().Less(got.Residual[centroid], 1.0,
		"the prediction has to credit each move to its own block's slope")
	s.Require().True(got.Arrived)
}

// TestReachedNeedsNoSlopes covers asking whether there is anything to do.
//
// Cheaper than asking what to do about it, and the difference is a reading per
// control: a pass measures every slope from where the chain sits, at about eight
// seconds each, so a dozen dials spend a minute and a half learning what
// arithmetic already knew.
func (s *SolveTestSuite) TestReachedNeedsNoSlopes() {
	aims := map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 100}}

	got, arrived := Reached(aims, map[audio.Figure]float64{centroid: 960})
	s.Require().True(arrived)
	s.Require().InDelta(0.4, got.Residual[centroid], 0.001)

	got, arrived = Reached(aims, map[audio.Figure]float64{centroid: 500})
	s.Require().False(arrived)
	s.Require().InDelta(5, got.Residual[centroid], 0.001)
	s.Require().Empty(got.Steps, "it says how far off, not what to do about it")
}

func TestSolveTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SolveTestSuite))
}
