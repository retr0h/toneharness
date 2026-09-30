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

// ChooseTestSuite covers ranking a list's settings against a target.
type ChooseTestSuite struct {
	suite.Suite
}

// aimed is a target on one axis, with a tolerance of one so a residual reads as
// the distance itself.
func (s *ChooseTestSuite) aimed(
	want float64,
) map[audio.Figure]Aim {
	return map[audio.Figure]Aim{audio.KeyCentroid: {Want: want, Tol: 1}}
}

// read is what the chain measured at one setting.
func read(
	centroid float64,
) map[audio.Figure]float64 {
	return map[audio.Figure]float64{audio.KeyCentroid: centroid}
}

// TestNearest covers Nearest, which ranks a list's settings by how near the
// target each one read.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestNearest() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The whole point.
			name: "nearest puts the closest setting first",
			then: func() {
				got := Nearest(s.aimed(100), map[int]map[audio.Figure]float64{
					0: read(130),
					1: read(104),
					2: read(180),
				})

				s.Require().Len(got, 3)
				s.Require().Equal(1, got[0].Value)
				s.Require().Equal(0, got[1].Value)
				s.Require().Equal(2, got[2].Value)

				s.Require().InDelta(4, got[0].Worst, 0.001,
					"four tolerances out, which is the distance in units of the tolerance")
			},
		},
		{
			// A list answering on its own.
			//
			// Worth reporting rather than folding into the ranking, because a
			// chain that arrives with no dial moved is one whose dials are
			// all still free for the next round of asking.
			name: "nearest says when a setting already arrives",
			then: func() {
				got := Nearest(s.aimed(100), map[int]map[audio.Figure]float64{
					0: read(100.5),
					1: read(140),
				})

				s.Require().True(got[0].Arrived)
				s.Require().False(got[1].Arrived)
			},
		},
		{
			// The scoring choice.
			//
			// Setting 0 is slightly wrong on both axes and setting 1 is right
			// on one and badly wrong on the other. Summed, setting 1 wins on
			// two axes out of the two; on the worst axis, which is the
			// measure the rest of the loop stops on, setting 0 wins. Ranked
			// any other way "2.4 tolerances out" would mean one thing in the
			// comparison and another in the passes that follow it.
			name: "nearest ranks on the worst axis rather than the sum",
			then: func() {
				aims := map[audio.Figure]Aim{
					audio.KeyCentroid: {Want: 0, Tol: 1},
					audio.KeyLow:      {Want: 0, Tol: 1},
				}

				got := Nearest(aims, map[int]map[audio.Figure]float64{
					0: {audio.KeyCentroid: 2, audio.KeyLow: 2},
					1: {audio.KeyCentroid: 0, audio.KeyLow: 3},
				})

				s.Require().Equal(0, got[0].Value, "the worst axis decides")
				s.Require().InDelta(2, got[0].Worst, 0.001)
				s.Require().InDelta(3, got[1].Worst, 0.001)
			},
		},
		{
			// Keeps one request answering one way.
			//
			// Two settings equally far from a target is not unusual: a target
			// may sit between two microphones, and a map walked twice yields
			// two orders.
			name: "nearest breaks ties on the setting",
			then: func() {
				for range 8 {
					got := Nearest(s.aimed(100), map[int]map[audio.Figure]float64{
						5: read(110),
						2: read(90),
						9: read(110),
					})

					s.Require().Equal([]int{2, 5, 9},
						[]int{got[0].Value, got[1].Value, got[2].Value})
				}
			},
		},
		{
			// The guard's effect.
			//
			// A setting that muted the chain, clipped the converters or was
			// refused never reaches this, and the ranking is over what was
			// readable rather than over every setting the control has. One
			// clipped microphone of twelve read 4,471Hz where the other
			// eleven sat between 126 and 147, so scored it would have won
			// every target asking for brightness.
			name: "nearest scores nothing it was not handed a reading for",
			then: func() {
				got := Nearest(s.aimed(4000), map[int]map[audio.Figure]float64{
					0: read(130),
					1: read(147),
				})

				s.Require().Len(got, 2)
				s.Require().NotContains(
					[]int{got[0].Value, got[1].Value}, 7, "the clipped setting is not here")
			},
		},
		{
			// A list no setting of which was readable.
			name: "nearest on nothing readable",
			then: func() {
				s.Require().Empty(Nearest(s.aimed(100), nil))
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestWhereIdentifiesAChoiceTheWayItIdentifiesAKnob covers the shared address.
//
// One identity for both kinds, because a chain's blocks each have a parameter 0
// and keying on the parameter alone moved one control and recorded it against
// another.
func (s *ChooseTestSuite) TestWhereIdentifiesAChoiceTheWayItIdentifiesAKnob() {
	c := Choice{Block: 2, Param: 3}
	k := Knob{Block: 2, Param: 3}

	s.Require().Equal(k.Where(), c.Where())
}

// TestRunners covers Runners, which is every setting other than each list's
// nearest, in the order worth trying them.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestRunners() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// What an alternative is.
			//
			// The nearest is already applied, so it is not something to back
			// up to. A runners list holding it would spend a whole
			// convergence re-solving the chain it just solved.
			name: "runners leaves out each lists nearest",
			then: func() {
				where := Where{Block: 1, Param: 4}

				got := Runners(
					map[Where][]Option{where: {
						{Value: 3, Worst: 1.2},
						{Value: 7, Worst: 2.0},
						{Value: 1, Worst: 4.5},
					}},
					map[Where]string{where: "a cabinet Mic"},
				)

				s.Require().Len(got, 2)
				s.Require().Equal(7, got[0].Option.Value)
				s.Require().Equal(1, got[1].Option.Value)
				s.Require().Equal("a cabinet Mic", got[0].Control)
			},
		},
		{
			// Why it is one queue.
			//
			// A chain with a cabinet's microphone and an amplifier's mid
			// frequency should try whichever runner-up read nearest, not
			// exhaust one control before touching the other. Each reading
			// costs a whole convergence, so spending the first on the
			// second-best reading anywhere is the order most likely to pay.
			name: "runners orders across every list rather than within one",
			then: func() {
				mic := Where{Block: 1, Param: 4}
				freq := Where{Block: 0, Param: 9}

				got := Runners(
					map[Where][]Option{
						mic:  {{Value: 3, Worst: 1.0}, {Value: 4, Worst: 9.0}},
						freq: {{Value: 0, Worst: 1.1}, {Value: 1, Worst: 1.4}},
					},
					map[Where]string{mic: "Mic", freq: "MidFreq"},
				)

				s.Require().Len(got, 2)
				s.Require().Equal(freq, got[0].Where,
					"1.4 is nearer than 9.0, whichever control it belongs to")
				s.Require().Equal(mic, got[1].Where)
			},
		},
		{
			// Nothing to back up to.
			name: "runners on a list with one readable setting",
			then: func() {
				where := Where{Block: 0, Param: 1}

				s.Require().Empty(Runners(
					map[Where][]Option{where: {{Value: 2, Worst: 3}}},
					map[Where]string{where: "Mic"},
				))
			},
		},
		{
			// Keeps a run reproducible.
			name: "runners ties order the same way twice",
			then: func() {
				first := Where{Block: 0, Param: 1}
				second := Where{Block: 0, Param: 2}
				third := Where{Block: 1, Param: 1}

				for range 8 {
					got := Runners(
						map[Where][]Option{
							third:  {{Value: 0, Worst: 1}, {Value: 5, Worst: 2}},
							first:  {{Value: 0, Worst: 1}, {Value: 5, Worst: 2}},
							second: {{Value: 0, Worst: 1}, {Value: 5, Worst: 2}},
						},
						map[Where]string{},
					)

					s.Require().Equal([]Where{first, second, third},
						[]Where{got[0].Where, got[1].Where, got[2].Where})
				}
			},
		},
		{
			// The last tie-break.
			//
			// Two settings of one control equally far from a target, which is
			// ordinary when a target sits between two microphones. Without
			// the setting to fall back on a map's order decides which one a
			// run spends its second convergence on.
			name: "runners ties within one list order on the setting",
			then: func() {
				where := Where{Block: 0, Param: 1}

				for range 8 {
					got := Runners(
						map[Where][]Option{where: {
							{Value: 0, Worst: 1},
							{Value: 8, Worst: 2},
							{Value: 3, Worst: 2},
						}},
						map[Where]string{where: "Mic"},
					)

					s.Require().Equal([]int{3, 8}, []int{got[0].Option.Value, got[1].Option.Value})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestChooseTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ChooseTestSuite))
}
