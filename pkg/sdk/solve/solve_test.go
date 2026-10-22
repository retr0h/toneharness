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
	mid      = audio.Figure("mid")
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

// lowCut is the largest lever a mic'd cabinet has, with its slopes read off the
// one committed sweep that holds it, `HD2_CabMicIr_1x15AmpegB15`.
//
// Task 255. It moves a median 190Hz of centroid against the microphone's 29,
// six and a half times as much, and it is far straighter: 0.92 to 0.97 where
// the microphone reads 0.00. **It reaches the centroid by removing the
// bottom.** Taken from 19.9Hz to 500Hz this cabinet goes from 131Hz at 97.09%
// of its energy below 250Hz to 420Hz at 32.18%, and on a bass that band is the
// fundamental.
func (s *SolveTestSuite) lowCut() Knob {
	return Knob{
		Block: 0, Param: 0, Control: "LowCut", Setting: "LowCut",
		At: 19.9, Low: 19.9, High: 500,
		Slope: map[audio.Figure]float64{
			centroid: 0.645, low: -0.155, mid: 0.154,
		},
	}
}

// micPosition is the same cabinet's Position, which is what a solve would
// rather be spending and cannot: across its whole travel it moves the centroid
// 2.3Hz where LowCut moves it 289.
func (s *SolveTestSuite) micPosition() Knob {
	return Knob{
		Block: 0, Param: 1, Control: "Position", Setting: "Position",
		At: 0.5, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{
			centroid: -2.31, low: 0.536, mid: -0.524,
		},
	}
}

// atTheCabinet is what that cabinet reads with LowCut at its bottom stop.
func (s *SolveTestSuite) atTheCabinet() map[audio.Figure]float64 {
	return map[audio.Figure]float64{centroid: 131.2, low: 97.09, mid: 2.90}
}

// TestABandShareIsHeldRatherThanDiscovered covers task 255: what it costs to
// leave a satisfied band share out of the matrix.
//
// The chain is one mic'd cabinet with real slopes, aimed 72Hz brighter than it
// sits and told to keep its low band where it is.
//
// Both halves of the fix are under test here, and they do different jobs.
// `after` predicting every named axis is what makes the damage visible; the
// hold is what stops it being done, and that is the half the hardware cares
// about. A pass applies its steps before anything reads `Arrived`, so without
// the hold LowCut is taken to 131Hz on the pedal and the report says afterwards
// that the bottom went with it.
func (s *SolveTestSuite) TestABandShareIsHeldRatherThanDiscovered() {
	aims := func(hold bool) map[audio.Figure]Aim {
		return map[audio.Figure]Aim{
			centroid: {Want: 203, Tol: 21},
			low:      {Want: 97.09, Tol: 2.5, Hold: hold},
			mid:      {Want: 2.90, Tol: 2.5, Hold: hold},
		}
	}

	knobs := []Knob{s.lowCut(), s.micPosition()}

	cutTo := func(got Result) float64 {
		for _, step := range got.Steps {
			if step.Setting == "LowCut" {
				return step.To
			}
		}

		s.Require().Fail("nothing moved LowCut")

		return 0
	}

	// Unheld, the solve reaches the centroid exactly and spends the whole low
	// band doing it: LowCut goes to its slope's limit and, by that same slope,
	// leaves the band at 79.85% where the target said 97.09%.
	loose, err := Toward(knobs, aims(false), s.atTheCabinet())
	s.Require().NoError(err)
	s.Require().InDelta(131.1, cutTo(loose), 0.5)
	s.Require().InDelta(0, loose.Residual[centroid], 0.01, "it reached the centroid")
	s.Require().Greater(loose.Residual[low], 6.0,
		"and the pass has to report what that cost")
	s.Require().False(loose.Arrived,
		"a pass may not report arriving on an axis its own moves broke")

	// Held, the cost is in the matrix from the first pass. LowCut moves a
	// quarter as far, the band stays inside its tolerance, and the centroid is
	// still out, so the loop takes another pass from where this one landed
	// rather than stopping on a chain with no bottom.
	tight, err := Toward(knobs, aims(true), s.atTheCabinet())
	s.Require().NoError(err)
	s.Require().Less(cutTo(tight), cutTo(loose)/2,
		"holding the band did not shorten the move")
	s.Require().Less(tight.Residual[low], 1.0, "the band is still where it was")
	s.Require().Greater(tight.Residual[centroid], 1.0)
	s.Require().False(tight.Arrived)

	// And over six passes with every control pulling together it still does
	// not get both, which is the honest answer and the one the unheld run
	// skipped past: this cabinet cannot be taken 72Hz brighter and keep its
	// bottom, because LowCut is the only control with any centroid authority
	// and the bottom is what it spends. It stalls 2.7 tolerances out on the
	// centroid rather than claiming to have arrived.
	best, err := Best(knobs, aims(true), s.atTheCabinet(), 6)
	s.Require().NoError(err)
	s.Require().False(best.Arrived)
	s.Require().Greater(best.Residual[centroid], 1.0,
		"it claimed a centroid this chain cannot reach")
}

// TestToward covers Toward, which answers one pass: the moves that close the
// gap from got to the aims.
//
// One method and one table, so a case is a row rather than a file.
func (s *SolveTestSuite) TestToward() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The simplest case there is.
			name: "one axis one control",
			then: func() {
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
			},
		},
		{
			// The target that is met.
			//
			// Left alone rather than defended. Every row costs controls, and
			// a genre target pins three figures and shrugs at six by design.
			name: "an axis already inside its tolerance is left alone",
			then: func() {
				got, err := Toward(
					[]Knob{s.treble()},
					map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 100}},
					map[audio.Figure]float64{centroid: 960},
				)

				s.Require().NoError(err)
				s.Require().Empty(got.Steps, "nothing to do")
				s.Require().True(got.Arrived)
				s.Require().InDelta(0.4, got.Residual[centroid], 0.001)
			},
		},
		{
			// A partial target.
			name: "an axis the target does not name is free",
			then: func() {
				got, err := Toward(
					[]Knob{s.bass(), s.treble()},
					map[audio.Figure]Aim{low: {Want: 98, Tol: 1}},
					map[audio.Figure]float64{low: 90, centroid: 150, high: 1},
				)

				s.Require().NoError(err)
				s.Require().NotContains(got.Residual, centroid,
					"a figure nobody asked about is not reported as wrong")
				s.Require().NotContains(got.Residual, high)
			},
		},
		{
			// Level, and is a regression.
			//
			// The rows were built by walking audio.MeasuredKeys, which is
			// what a corpus states. Level is not one: a record's loudness is
			// a mastering decision, so no genre has a level to aim at. An aim
			// on it was accepted and silently dropped.
			//
			// What that cost is on the amplifier. Level is the axis every
			// gain control moves furthest, so an unconstrained solve spends
			// it: asked for punk, the loop walked an SV Beast's Master to
			// zero in two passes and turned the amp off.
			name: "an axis no corpus reports is still solved",
			then: func() {
				s.Require().NotContains(audio.MeasuredKeys(), string(audio.KeyLevel),
					"the premise: no corpus reports a level")

				// Where that run left it: Master all but shut, the chain 8.3dB below the
				// level it settled at before a single dial was turned.
				master := Knob{
					Block: 1, Param: 2, Control: "Master", At: 0.024, Low: 0, High: 1,
					Slope: map[audio.Figure]float64{audio.KeyLevel: 40},
				}

				got, err := Toward(
					[]Knob{master},
					map[audio.Figure]Aim{audio.KeyLevel: {Want: -23.7, Tol: 6}},
					map[audio.Figure]float64{audio.KeyLevel: -32.0},
				)

				s.Require().NoError(err)
				s.Require().Len(got.Steps, 1, "the aim became a row")
				s.Require().Positive(got.Steps[0].By, "and the row turns the amplifier back up")
			},
		},
		{
			// The case the whole package exists for.
			//
			// The controls disagree: Bass pulls the centre down while lifting
			// the low band, Treble pushes it up. A word at a time cannot
			// answer this, and a least-squares solve does it in one pass.
			name: "two axes two controls",
			then: func() {
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
			},
		},
		{
			// A target past what a knob has.
			name: "a move is clamped to the controls range",
			then: func() {
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
			},
		},
		{
			// A control with its own span.
			//
			// 1,452 of the device's float controls do not run zero to one,
			// and one swept as though it did never leaves its bottom stop.
			name: "a range that is not zero to one",
			then: func() {
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
			},
		},
		{
			// The row of zeroes.
			//
			// MidFreq at a flat MidGain is the real case: it says where the
			// mid band sits and there is no boost for it to move, so its
			// whole row is zero. A solver that took that for an inert control
			// would never touch it again, and it is not inert, it is
			// conditional.
			name: "a control that moves nothing",
			then: func() {
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
			},
		},
		{
			// A target that constrains nothing.
			//
			// Which is what a genre earning no words would give, and punk is
			// one. Answered as arrived rather than refused: nothing was asked
			// for and nothing is wrong.
			name: "no axis named",
			then: func() {
				got, err := Toward(
					[]Knob{s.treble()},
					map[audio.Figure]Aim{},
					map[audio.Figure]float64{centroid: 550},
				)

				s.Require().NoError(err)
				s.Require().Empty(got.Steps)
				s.Require().True(got.Arrived)
			},
		},
		{
			// An axis nobody can satisfy.
			//
			// Zero tolerance means arrived is unreachable and every pass
			// would keep spending controls on it, so it is read as an axis
			// nobody stated.
			name: "a tolerance of zero is not a constraint",
			then: func() {
				got, err := Toward(
					[]Knob{s.treble()},
					map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 0}},
					map[audio.Figure]float64{centroid: 550},
				)

				s.Require().NoError(err)
				s.Require().Empty(got.Steps)
				s.Require().NotContains(got.Residual, centroid)
			},
		},
		{
			// The singular system.
			//
			// Two tone stacks in one chain come close enough that the
			// difference is rounding. The ridge is what keeps it solvable,
			// and it splits the move between them rather than picking one.
			name: "two controls that do the same thing",
			then: func() {
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
			},
		},
		{
			// What the ridge is for.
			name: "small moves are preferred",
			then: func() {
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
			},
		},
		{
			// The identity of a knob.
			//
			// Every block in a chain has a parameter 0, so a param index
			// alone names nothing. Keying on it credited one block's move to
			// another block's slope, which did not fail: it predicted a
			// centroid of 2.8MHz, and the loop concluded from that prediction
			// that the chain could not reach the target.
			//
			// The two knobs here are deliberately opposed. If their moves are
			// swapped the prediction lands the wrong side of the target, so
			// the residual catches it.
			name: "two blocks sharing a param index",
			then: func() {
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
			},
		},
		{
			// A control with nothing to travel, which a chain really carries: a
			// catalog gives a few blocks a parameter whose range is one value.
			// Dropped rather than divided by, because its authority over every
			// axis is zero and a solve weighted on zero has nothing to weight.
			name: "a control with no travel",
			then: func() {
				stuck := Knob{
					Block: 1, Param: 2, Control: "Type", At: 1, Low: 1, High: 1,
					Slope: map[audio.Figure]float64{centroid: 900},
				}

				got, err := Toward(
					[]Knob{s.treble(), stuck},
					map[audio.Figure]Aim{centroid: {Want: 900, Tol: 5}},
					map[audio.Figure]float64{centroid: 500},
				)

				s.Require().NoError(err)
				s.Require().Len(got.Steps, 1)
				s.Require().Equal("Treble", got.Steps[0].Control)
			},
		},
		{
			// Task 253. A slope read at one position over one small step stops
			// being true almost immediately on a control that is not a line,
			// and least squares is free to answer with a swing from stop to
			// stop on it.
			name: "a move is no longer than the slope carries",
			then: func() {
				k := s.treble()
				k.Carry = 0.2

				got, err := Toward(
					[]Knob{k},
					map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
					map[audio.Figure]float64{centroid: 550},
				)

				s.Require().NoError(err)
				s.Require().Len(got.Steps, 1)
				s.Require().InDelta(0.2, got.Steps[0].By, 0.001,
					"half a turn was asked for and a fifth is what carries")
				s.Require().True(got.Steps[0].Capped,
					"a shortened move has to say so")

				// And the prediction is of the move that will be made rather
				// than the one asked for, so the pass does not report arriving
				// somewhere it stopped short of.
				s.Require().False(got.Arrived)
				s.Require().Greater(got.Residual[centroid], 1.0)
			},
		},
		{
			// The ordinary case, and what every caller that says nothing gets.
			// 436 of this device's 614 blocks have never been swept, so there
			// is nothing to say how far their slopes carry.
			name: "no carry stated caps nothing",
			then: func() {
				got, err := Toward(
					[]Knob{s.treble()},
					map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
					map[audio.Figure]float64{centroid: 550},
				)

				s.Require().NoError(err)
				s.Require().Len(got.Steps, 1)
				s.Require().InDelta(0.5, got.Steps[0].By, 0.01)
				s.Require().False(got.Steps[0].Capped)
			},
		},
		{
			// A carry longer than the move, which must not mark a step capped
			// and must not round it.
			name: "a carry the move is inside of changes nothing",
			then: func() {
				k := s.treble()
				k.Carry = 0.9

				got, err := Toward(
					[]Knob{k},
					map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
					map[audio.Figure]float64{centroid: 550},
				)

				s.Require().NoError(err)
				s.Require().Len(got.Steps, 1)
				s.Require().InDelta(0.5, got.Steps[0].By, 0.01)
				s.Require().False(got.Steps[0].Capped)
			},
		},
		{
			// The other direction, because a cap is a distance rather than a
			// ceiling and a control coming down has to be shortened too.
			name: "a move downward is capped the same way",
			then: func() {
				k := s.treble()
				k.Carry = 0.1

				got, err := Toward(
					[]Knob{k},
					map[audio.Figure]Aim{centroid: {Want: 100, Tol: 10}},
					map[audio.Figure]float64{centroid: 550},
				)

				s.Require().NoError(err)
				s.Require().Len(got.Steps, 1)
				s.Require().InDelta(-0.1, got.Steps[0].By, 0.001)
				s.Require().True(got.Steps[0].Capped)
			},
		},
		{
			// A cap shorter than the smallest move worth making drops the step
			// rather than sending the pedal a message that cannot change a
			// reading. Every step is a message, and a chain of sixteen of them
			// back to back is what stopped the device replying.
			name: "a carry under the least move drops the step",
			then: func() {
				k := s.treble()
				k.Carry = Least / 2

				got, err := Toward(
					[]Knob{k},
					map[audio.Figure]Aim{centroid: {Want: 1000, Tol: 10}},
					map[audio.Figure]float64{centroid: 550},
				)

				s.Require().NoError(err)
				s.Require().Empty(got.Steps)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
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
