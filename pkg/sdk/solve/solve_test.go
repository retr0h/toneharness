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

// TestTowardTurnsATargetIntoMoves covers every shape a target can arrive in.
//
// All of it is arithmetic, so a row is a set of controls, what is being aimed
// at, and where the chain reads now. What a device decides is whether the
// slopes were true, which is the loop's job rather than this one's.
func (s *SolveTestSuite) TestTowardTurnsATargetIntoMoves() {
	for _, tt := range []struct {
		name  string
		knobs []Knob
		aims  map[audio.Figure]Aim
		from  map[audio.Figure]float64
		is    error
		then  func(Result)
	}{
		{
			name:  "one axis and one control, the simplest case there is",
			knobs: []Knob{s.treble()},
			aims:  map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
			from:  map[audio.Figure]float64{centroid: 550},
			then: func(got Result) {
				s.Require().Len(got.Steps, 1)
				s.Require().Equal("Treble", got.Steps[0].Control)
				s.Require().InDelta(0.5, got.Steps[0].By, 0.01,
					"450Hz to close at 900Hz per turn is half a turn")
				s.Require().True(got.Arrived)
			},
		},
		{
			// Left alone rather than defended. Every row costs controls, and a
			// genre target pins three figures and shrugs at six by design.
			name:  "an axis already inside its tolerance is left alone",
			knobs: []Knob{s.treble()},
			aims:  map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 100}},
			from:  map[audio.Figure]float64{centroid: 960},
			then: func(got Result) {
				s.Require().Empty(got.Steps, "nothing to do")
				s.Require().True(got.Arrived)
				s.Require().InDelta(0.4, got.Residual[centroid], 0.001)
			},
		},
		{
			name:  "an axis the target does not name is free",
			knobs: []Knob{s.bass(), s.treble()},
			aims:  map[audio.Figure]Aim{low: {Want: 98, Tol: 1}},
			from:  map[audio.Figure]float64{low: 90, centroid: 150, high: 1},
			then: func(got Result) {
				s.Require().NotContains(got.Residual, centroid,
					"a figure nobody asked about is not reported as wrong")
				s.Require().NotContains(got.Residual, high)
			},
		},
		{
			// The rows were built by walking audio.MeasuredKeys, which is what
			// a corpus states. Level is not one: a record's loudness is a
			// mastering decision, so no genre has a level to aim at, and an aim
			// on it was accepted and silently dropped.
			//
			// What that cost is on the amplifier. Level is the axis every gain
			// control moves furthest, so an unconstrained solve spends it:
			// asked for punk, the loop walked an SV Beast's Master to zero in
			// two passes and turned the amp off. The knob here is where that
			// run left it, the chain 8.3dB below where it settled before a
			// single dial was turned.
			name: "an axis no corpus reports is still solved",
			knobs: []Knob{{
				Block: 1, Param: 2, Control: "Master",
				At: 0.024, Low: 0, High: 1,
				Slope: map[audio.Figure]float64{audio.KeyLevel: 40},
			}},
			aims: map[audio.Figure]Aim{audio.KeyLevel: {Want: -23.7, Tol: 6}},
			from: map[audio.Figure]float64{audio.KeyLevel: -32.0},
			then: func(got Result) {
				s.Require().NotContains(audio.MeasuredKeys(),
					string(audio.KeyLevel),
					"the premise: no corpus reports a level")
				s.Require().Len(got.Steps, 1, "the aim became a row")
				s.Require().Positive(got.Steps[0].By,
					"and the row turns the amplifier back up")
			},
		},
		{
			// The controls disagree: Bass pulls the centre down while lifting
			// the low band, Treble pushes it up. A word at a time cannot answer
			// this, and a least-squares solve does it in one pass. This is the
			// case the whole package exists for.
			name:  "two axes and two controls that disagree",
			knobs: []Knob{s.bass(), s.treble()},
			aims: map[audio.Figure]Aim{
				low:      {Want: 96, Tol: 0.5},
				centroid: {Want: 200, Tol: 5},
			},
			from: map[audio.Figure]float64{low: 90, centroid: 150, high: 1},
			then: func(got Result) {
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
			},
		},
		{
			name:  "a move is clamped to the control's range",
			knobs: []Knob{s.treble()},
			aims:  map[audio.Figure]Aim{centroid: {Want: 100000, Tol: 10}},
			from:  map[audio.Figure]float64{centroid: 550},
			then: func(got Result) {
				s.Require().Len(got.Steps, 1)
				s.Require().InDelta(1, got.Steps[0].To, 0.0001,
					"hard against its top stop")
				s.Require().False(got.Arrived,
					"and it says it did not get there")
				s.Require().Greater(got.Residual[centroid], 1.0)
			},
		},
		{
			// 1,452 of the device's float controls do not run zero to one, and
			// one swept as though it did never leaves its bottom stop.
			name: "a range that is not zero to one",
			knobs: []Knob{{
				Block: 1, Param: 3, Control: "MidFreq",
				At: 500, Low: 125, High: 4000,
				Slope: map[audio.Figure]float64{centroid: 0.5},
			}},
			aims: map[audio.Figure]Aim{centroid: {Want: 700, Tol: 5}},
			from: map[audio.Figure]float64{centroid: 500},
			then: func(got Result) {
				s.Require().Len(got.Steps, 1)
				s.Require().InDelta(900, got.Steps[0].To, 1,
					"400Hz of centroid at half a hertz per hertz is 800 of travel")
			},
		},
		{
			// MidFreq at a flat MidGain is the real case: it says where the mid
			// band sits and there is no boost for it to move, so its whole row
			// is zero. A solver that took that for an inert control would never
			// touch it again, and it is not inert, it is conditional.
			name: "a control that moves nothing",
			knobs: []Knob{{
				Block: 1, Param: 3, Control: "MidFreq", At: 0.5, Low: 0, High: 1,
				Slope: map[audio.Figure]float64{centroid: 0},
			}},
			aims: map[audio.Figure]Aim{centroid: {Want: 700, Tol: 5}},
			from: map[audio.Figure]float64{centroid: 500},
			is:   ErrNoKnobs,
		},
		{
			// Which is what a genre earning no words would give, and punk is
			// one. Answered as arrived rather than refused: nothing was asked
			// for and nothing is wrong.
			name:  "no axis named at all",
			knobs: []Knob{s.treble()},
			aims:  map[audio.Figure]Aim{},
			from:  map[audio.Figure]float64{centroid: 550},
			then: func(got Result) {
				s.Require().Empty(got.Steps)
				s.Require().True(got.Arrived)
			},
		},
		{
			// Zero tolerance means arrived is unreachable and every pass would
			// keep spending controls on it, so it is read as an axis nobody
			// stated.
			name:  "a tolerance of zero is not a constraint",
			knobs: []Knob{s.treble()},
			aims:  map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 0}},
			from:  map[audio.Figure]float64{centroid: 550},
			then: func(got Result) {
				s.Require().Empty(got.Steps)
				s.Require().NotContains(got.Residual, centroid)
			},
		},
		{
			// Two tone stacks in one chain come close enough that the
			// difference is rounding. The ridge is what keeps the system
			// solvable, and it splits the move rather than picking one.
			name: "two controls that do the same thing",
			knobs: func() []Knob {
				one, two := s.treble(), s.treble()
				two.Param, two.Control = 9, "Treble2"

				return []Knob{one, two}
			}(),
			aims: map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
			from: map[audio.Figure]float64{centroid: 550},
			then: func(got Result) {
				s.Require().Len(got.Steps, 2)
				s.Require().InDelta(got.Steps[0].By, got.Steps[1].By, 0.001,
					"neither is preferred, so the move is shared")
				s.Require().InDelta(0.5,
					got.Steps[0].By+got.Steps[1].By, 0.01,
					"and together they close the gap")
			},
		},
		{
			name: "small moves are preferred, which is what the ridge is for",
			knobs: func() []Knob {
				weak := s.treble()
				weak.Control = "Fine"
				weak.Slope = map[audio.Figure]float64{centroid: 10}

				strong := s.treble()
				strong.Param, strong.Control = 9, "Coarse"

				return []Knob{weak, strong}
			}(),
			aims: map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
			from: map[audio.Figure]float64{centroid: 550},
			then: func(got Result) {
				by := map[string]float64{}
				for _, st := range got.Steps {
					by[st.Control] = math.Abs(st.By)
				}

				s.Require().Greater(by["Coarse"], by["Fine"],
					"the control that moves the figure furthest per turn does the work")
			},
		},
		{
			// Every block in a chain has a parameter 0, so a param index alone
			// names nothing. Keying on it credited one block's move to another
			// block's slope, which did not fail: it predicted a centroid of
			// 2.8MHz, and the loop concluded from that prediction that the
			// chain could not reach the target.
			//
			// The two knobs are deliberately opposed. If their moves are
			// swapped the prediction lands the wrong side of the target, so the
			// residual catches it.
			name: "two blocks sharing a param index",
			knobs: []Knob{
				{
					Block: 1, Param: 0, Control: "Amp Treble",
					At: 0.5, Low: 0, High: 1,
					Slope: map[audio.Figure]float64{centroid: 1000},
				},
				{
					Block: 2, Param: 0, Control: "Cab HighCut",
					At: 0.5, Low: 0, High: 1,
					Slope: map[audio.Figure]float64{centroid: -200},
				},
			},
			aims: map[audio.Figure]Aim{centroid: {Want: 900, Tol: 5}},
			from: map[audio.Figure]float64{centroid: 500},
			then: func(got Result) {
				s.Require().Len(got.Steps, 2)

				where := map[Where]string{}
				for _, st := range got.Steps {
					where[st.Where()] = st.Control
				}

				s.Require().Equal("Amp Treble",
					where[Where{Block: 1, Param: 0}])
				s.Require().Equal("Cab HighCut",
					where[Where{Block: 2, Param: 0}])

				s.Require().Less(got.Residual[centroid], 1.0,
					"the prediction has to credit each move to its own block's slope")
				s.Require().True(got.Arrived)
			},
		},
	} {
		s.Run(tt.name, func() {
			got, err := Toward(tt.knobs, tt.aims, tt.from)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				return
			}

			s.Require().NoError(err)
			tt.then(got)
		})
	}
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
