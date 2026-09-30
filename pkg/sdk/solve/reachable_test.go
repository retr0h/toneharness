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

// ReachableTestSuite covers answering "can this chain get there" from readings
// already taken.
type ReachableTestSuite struct {
	suite.Suite
}

// one is a target on the centroid with a tolerance of one, so a gap reads as
// the distance itself.
func one(
	want float64,
) map[audio.Figure]Aim {
	return map[audio.Figure]Aim{audio.KeyCentroid: {Want: want, Tol: 1}}
}

func spanOf(
	from, low, high, swing float64,
) map[audio.Figure]Span {
	return map[audio.Figure]Span{
		audio.KeyCentroid: {From: from, Low: low, High: high, Swing: swing},
	}
}

// TestATargetSomeReadingAlreadyLandedOn is the claim worth trusting.
//
// TestReachableSaysWhetherAChainCanGetThere covers asking what the sweeps already know, before a single reading is taken.
//
// One method and one table, so a case is a row rather than a file.
func (s *ReachableTestSuite) TestReachableSaysWhetherAChainCanGetThere() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// where the target wants, so a setting exists and the loop's job is to find it.
			name: "a target some reading already landed on",
			then: func() {
				got := Reachable(one(500), spanOf(100, 90, 900, 0))

				s.Require().Len(got, 1)
				s.Require().True(got[0].Shown)
				s.Require().True(got[0].Within)
				s.Require().False(got[0].Met, "the chain sits at 100, not 500")
				s.Require().InDelta(400, got[0].Gap, 0.001)
			},
		},
		{
			// the gap. That refuses nothing and promises nothing: the sum assumes every
			// control pulls the same way, which they do not, and finding out is what the
			// loop is for.
			name: "a swing wide enough is not a promise",
			then: func() {
				got := Reachable(one(500), spanOf(100, 90, 110, 800))

				s.Require().False(got[0].Shown, "no reading was near it")
				s.Require().True(got[0].Within, "but nothing rules it out")
			},
		},
		{
			name: "a chain already inside is met",
			then: func() {
				got := Reachable(one(100.5), spanOf(100, 100, 100, 0))

				s.Require().True(got[0].Met)
				s.Require().True(got[0].Within)
			},
		},
		{
			// to its centre, so a reading one tolerance outside the range still shows the
			// target is touchable.
			name: "the tolerance widens the target at both ends",
			then: func() {
				aims := map[audio.Figure]Aim{audio.KeyCentroid: {Want: 110.5, Tol: 1}}

				s.Require().True(Reachable(aims, spanOf(50, 90, 110, 0))[0].Shown,
					"110.5 is inside 110 plus a tolerance of one")

				s.Require().False(Reachable(aims, spanOf(50, 90, 109, 0))[0].Shown)
			},
		},
		{
			// axis is not a chain that cannot reach it, and calling it unreachable would
			// talk somebody out of a run that would have worked.
			name: "an axis nothing measured is left out",
			then: func() {
				s.Require().Empty(Reachable(one(500), map[audio.Figure]Span{}))
			},
		},
		{
			name: "an axis with no tolerance is left out",
			then: func() {
				aims := map[audio.Figure]Aim{audio.KeyCentroid: {Want: 500, Tol: 0}}

				s.Require().Empty(Reachable(aims, spanOf(100, 90, 110, 0)))
			},
		},
		{
			// wall. Reading the wall first is what stops somebody tuning for an afternoon,
			// so it sorts above an axis that is further out and reachable.
			name: "the wall is reported before the distance",
			then: func() {
				aims := map[audio.Figure]Aim{
					audio.KeyCentroid: {Want: 4100, Tol: 1},
					audio.KeyLow:      {Want: 900, Tol: 1},
				}

				got := Reachable(aims, map[audio.Figure]Span{
					// 800 tolerances out, and a reading landed there.
					audio.KeyLow: {From: 100, Low: 50, High: 950, Swing: 0},
					// 100 out, and nothing can close it.
					audio.KeyCentroid: {From: 4000, Low: 3990, High: 4010, Swing: 5},
				})

				s.Require().Equal(audio.KeyCentroid, got[0].Figure,
					"the wall reads first although low is eight times further out")
				s.Require().False(got[0].Within)
				s.Require().Greater(got[1].Gap, got[0].Gap)
			},
		},
		{
			// no order, so a comparator right in one direction and wrong in the other
			// reports a different worst axis run to run, and the worst axis is the whole
			// answer.
			name: "the order is the same whichever way the map was walked",
			then: func() {
				aims := map[audio.Figure]Aim{
					audio.KeyCentroid: {Want: 5000, Tol: 1},
					audio.KeyHigh:     {Want: 5000, Tol: 1},
					audio.KeyLow:      {Want: 300, Tol: 1},
					audio.KeyMid:      {Want: 400, Tol: 1},
				}

				for range 8 {
					got := Reachable(aims, map[audio.Figure]Span{
						// Reachable, and far out.
						audio.KeyLow: {From: 100, Low: 50, High: 350},
						audio.KeyMid: {From: 100, Low: 50, High: 450},
						// Out of reach, and nearer.
						audio.KeyCentroid: {From: 4000, Low: 3990, High: 4010, Swing: 5},
						audio.KeyHigh:     {From: 4500, Low: 4490, High: 4510, Swing: 5},
					})

					s.Require().Len(got, 4)
					s.Require().False(got[0].Within)
					s.Require().False(got[1].Within)
					s.Require().True(got[2].Within)
					s.Require().True(got[3].Within)

					s.Require().Equal(audio.KeyCentroid, got[0].Figure,
						"1000 out beats 500 out among the walls")
					s.Require().Equal(audio.KeyMid, got[2].Figure,
						"300 out beats 200 out among the reachable")
				}
			},
		},
		{
			name: "two axes equally far order the same way twice",
			then: func() {
				aims := map[audio.Figure]Aim{
					audio.KeyCentroid: {Want: 200, Tol: 1},
					audio.KeyLow:      {Want: 200, Tol: 1},
				}

				for range 8 {
					got := Reachable(aims, map[audio.Figure]Span{
						audio.KeyCentroid: {From: 100, Low: 100, High: 250},
						audio.KeyLow:      {From: 100, Low: 100, High: 250},
					})

					s.Require().Equal(audio.KeyCentroid, got[0].Figure)
					s.Require().Equal(audio.KeyLow, got[1].Figure)
				}
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestATargetNoReadingReachedAndNoControlCanClose is the refusal.
//
// TestWorthNamesWhatDecidesTheVerdict covers which axis a refusal is about.
//
// One method and one table, so a case is a row rather than a file.
func (s *ReachableTestSuite) TestWorthNamesWhatDecidesTheVerdict() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// and 110, the target is at 5,000, and every control added together moves the
			// axis 50. No arrangement of them gets there, and saying so costs a second
			// against the five minutes the loop spends discovering it.
			name: "a target no reading reached and no control can close",
			then: func() {
				got := Reachable(one(5000), spanOf(100, 90, 110, 50))

				s.Require().False(got[0].Shown)
				s.Require().False(got[0].Within)
				s.Require().False(got[0].Met)

				worst, ok := Worth(got)
				s.Require().False(ok)
				s.Require().Equal(audio.KeyCentroid, worst.Figure)
			},
		},
		{
			// are, because a target is met only when every axis it names is.
			name: "worth names the axis that decides",
			then: func() {
				aims := map[audio.Figure]Aim{
					audio.KeyCentroid: {Want: 5000, Tol: 1},
					audio.KeyLow:      {Want: 100, Tol: 1},
				}

				worst, ok := Worth(Reachable(aims, map[audio.Figure]Span{
					audio.KeyLow:      {From: 100, Low: 100, High: 100},
					audio.KeyCentroid: {From: 100, Low: 90, High: 110, Swing: 5},
				}))

				s.Require().False(ok)
				s.Require().Equal(audio.KeyCentroid, worst.Figure)
			},
		},
		{
			name: "worth on nothing measured",
			then: func() {
				_, ok := Worth(nil)
				s.Require().False(ok)
			},
		},
		{
			name: "worth on a target every axis of which is inside",
			then: func() {
				worst, ok := Worth(Reachable(one(200), spanOf(100, 50, 250, 0)))

				s.Require().True(ok)
				s.Require().Equal(audio.KeyCentroid, worst.Figure,
					"the worst is still named when the answer is yes")
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestASwingWideEnoughIsNotAPromise covers the weaker claim.
//

// TestTheToleranceWidensTheTargetAtBothEnds covers the edge.
//

// TestAnAxisNothingMeasuredIsLeftOut covers a figure with no readings.
//

// TestTheWallIsReportedBeforeTheDistance covers the order.
//

// TestWorthNamesTheAxisThatDecides covers the whole-target answer.
//

func TestReachableTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReachableTestSuite))
}

// TestTheOrderIsTheSameWhicheverWayTheMapWasWalked covers the comparator.
//

// knob is one control with a slope on the centroid.
func knob(
	at, low, high, slope float64,
) Knob {
	return Knob{
		Block: 0, Param: int(low*1000 + high), At: at, Low: low, High: high,
		Slope: map[audio.Figure]float64{audio.KeyCentroid: slope},
	}
}

// TestBestFindsThePositionsThatCloseTheGap covers the search over settings, and where it stops.
//
// One method and one table, so a case is a row rather than a file.
func (s *ReachableTestSuite) TestBestFindsThePositionsThatCloseTheGap() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "best closes a gap one control can cover",
			then: func() {
				got, err := Best(
					[]Knob{knob(0.5, 0, 1, 1000)},
					one(600),
					map[audio.Figure]float64{audio.KeyCentroid: 100},
					5,
				)

				s.Require().NoError(err)
				s.Require().True(got.Arrived)
				s.Require().LessOrEqual(got.Residual[audio.KeyCentroid], 1.0)
			},
		},
		{
			// so five hundred is everything it has and the target is four thousand away.
			// No amount of iterating finds what is not there.
			name: "best refuses a gap no position reaches",
			then: func() {
				got, err := Best(
					[]Knob{knob(0.5, 0, 1, 1000)},
					one(4100),
					map[audio.Figure]float64{audio.KeyCentroid: 100},
					5,
				)

				s.Require().NoError(err)
				s.Require().False(got.Arrived)
				s.Require().Greater(got.Residual[audio.KeyCentroid], 1.0)
			},
		},
		{
			// reachable on its own and no position reaches both, which is the whole reason
			// Reachable checking them one at a time ruled nothing out.
			name: "best trades two axes against one control",
			then: func() {
				both := Knob{
					Block: 0, Param: 1, At: 0.5, Low: 0, High: 1,
					Slope: map[audio.Figure]float64{
						audio.KeyCentroid: 1000,
						audio.KeyLow:      -1000,
					},
				}

				aims := map[audio.Figure]Aim{
					audio.KeyCentroid: {Want: 400, Tol: 1},
					audio.KeyLow:      {Want: 400, Tol: 1},
				}

				from := map[audio.Figure]float64{audio.KeyCentroid: 100, audio.KeyLow: 100}

				// Alone, each is 300 out against 500 of movement, so neither is refused.
				for _, v := range Reachable(aims, map[audio.Figure]Span{
					audio.KeyCentroid: {From: 100, Low: 100, High: 600, Swing: 1000},
					audio.KeyLow:      {From: 100, Low: 100, High: 600, Swing: 1000},
				}) {
					s.Require().True(v.Within, "%s is reachable on its own", v.Figure)
				}

				got, err := Best([]Knob{both}, aims, from, 5)

				s.Require().NoError(err)
				s.Require().False(got.Arrived,
					"one dial cannot raise the centroid and the low band at once")
			},
		},
		{
			// landed.
			name: "best stops when a pass stops improving",
			then: func() {
				got, err := Best(
					[]Knob{knob(0.5, 0, 1, 1000)},
					one(600),
					map[audio.Figure]float64{audio.KeyCentroid: 100},
					50,
				)

				s.Require().NoError(err)
				s.Require().LessOrEqual(got.Residual[audio.KeyCentroid], 1.0,
					"fifty passes are no worse than the pass that arrived")
			},
		},
		{
			name: "best on a chain nothing can turn",
			then: func() {
				_, err := Best(
					[]Knob{knob(0.5, 0, 1, 0)},
					one(600),
					map[audio.Figure]float64{audio.KeyCentroid: 100},
					5,
				)

				s.Require().ErrorIs(err, ErrNoKnobs)
			},
		},
		{
			name: "best on a target already met",
			then: func() {
				got, err := Best(
					[]Knob{knob(0.5, 0, 1, 1000)},
					one(100.5),
					map[audio.Figure]float64{audio.KeyCentroid: 100},
					5,
				)

				s.Require().NoError(err)
				s.Require().True(got.Arrived)
			},
		},
		{
			// it reported a centroid of 141 hertz as 141 tolerances out, while Arrived
			// still said the chain had got there.
			name: "best does not overwrite its own answer",
			then: func() {
				got, err := Best(
					[]Knob{knob(0.5, 0, 1, 1000)},
					one(600),
					map[audio.Figure]float64{audio.KeyCentroid: 100},
					5,
				)

				s.Require().NoError(err)
				s.Require().Equal(got.Arrived, arrived(got.Residual),
					"the verdict and the numbers behind it have to agree")
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestBestRefusesAGapNoPositionReaches is the answer per-axis could not give.
//

// TestBestTradesTwoAxesAgainstOneControl is why the axes go in together.
//

// TestBestStopsWhenAPassStopsImproving covers clamping making things worse.
//

// TestBestDoesNotOverwriteItsOwnAnswer covers the aliasing that bit.
//
