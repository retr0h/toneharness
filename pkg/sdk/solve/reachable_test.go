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
// Nothing is modelled to make it. A reading actually taken of these blocks sat
// where the target wants, so a setting exists and the loop's job is to find it.
func (s *ReachableTestSuite) TestATargetSomeReadingAlreadyLandedOn() {
	got := Reachable(one(500), spanOf(100, 90, 900, 0))

	s.Require().Len(got, 1)
	s.Require().True(got[0].Shown)
	s.Require().True(got[0].Within)
	s.Require().False(got[0].Met, "the chain sits at 100, not 500")
	s.Require().InDelta(400, got[0].Gap, 0.001)
}

// TestATargetNoReadingReachedAndNoControlCanClose is the refusal.
//
// The whole point of the thing. Every reading of these blocks sat between 90
// and 110, the target is at 5,000, and every control added together moves the
// axis 50. No arrangement of them gets there, and saying so costs a second
// against the five minutes the loop spends discovering it.
func (s *ReachableTestSuite) TestATargetNoReadingReachedAndNoControlCanClose() {
	got := Reachable(one(5000), spanOf(100, 90, 110, 50))

	s.Require().False(got[0].Shown)
	s.Require().False(got[0].Within)
	s.Require().False(got[0].Met)

	worst, ok := Worth(got)
	s.Require().False(ok)
	s.Require().Equal(audio.KeyCentroid, worst.Figure)
}

// TestASwingWideEnoughIsNotAPromise covers the weaker claim.
//
// No reading landed near the target, but the controls have more movement than
// the gap. That refuses nothing and promises nothing: the sum assumes every
// control pulls the same way, which they do not, and finding out is what the
// loop is for.
func (s *ReachableTestSuite) TestASwingWideEnoughIsNotAPromise() {
	got := Reachable(one(500), spanOf(100, 90, 110, 800))

	s.Require().False(got[0].Shown, "no reading was near it")
	s.Require().True(got[0].Within, "but nothing rules it out")
}

// TestAChainAlreadyInsideIsMet covers an axis needing no work.
func (s *ReachableTestSuite) TestAChainAlreadyInsideIsMet() {
	got := Reachable(one(100.5), spanOf(100, 100, 100, 0))

	s.Require().True(got[0].Met)
	s.Require().True(got[0].Within)
}

// TestTheToleranceWidensTheTargetAtBothEnds covers the edge.
//
// A chain has to be carried to the edge of what counts as arriving rather than
// to its centre, so a reading one tolerance outside the range still shows the
// target is touchable.
func (s *ReachableTestSuite) TestTheToleranceWidensTheTargetAtBothEnds() {
	aims := map[audio.Figure]Aim{audio.KeyCentroid: {Want: 110.5, Tol: 1}}

	s.Require().True(Reachable(aims, spanOf(50, 90, 110, 0))[0].Shown,
		"110.5 is inside 110 plus a tolerance of one")

	s.Require().False(Reachable(aims, spanOf(50, 90, 109, 0))[0].Shown)
}

// TestAnAxisNothingMeasuredIsLeftOut covers a figure with no readings.
//
// Left out rather than reported as out of reach. A chain nobody swept on an
// axis is not a chain that cannot reach it, and calling it unreachable would
// talk somebody out of a run that would have worked.
func (s *ReachableTestSuite) TestAnAxisNothingMeasuredIsLeftOut() {
	s.Require().Empty(Reachable(one(500), map[audio.Figure]Span{}))
}

// TestAnAxisWithNoToleranceIsLeftOut covers a target that pins nothing.
func (s *ReachableTestSuite) TestAnAxisWithNoToleranceIsLeftOut() {
	aims := map[audio.Figure]Aim{audio.KeyCentroid: {Want: 500, Tol: 0}}

	s.Require().Empty(Reachable(aims, spanOf(100, 90, 110, 0)))
}

// TestTheWallIsReportedBeforeTheDistance covers the order.
//
// A wide gap the controls can close is work; a narrow one they cannot is a
// wall. Reading the wall first is what stops somebody tuning for an afternoon,
// so it sorts above an axis that is further out and reachable.
func (s *ReachableTestSuite) TestTheWallIsReportedBeforeTheDistance() {
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
}

// TestWorthNamesTheAxisThatDecides covers the whole-target answer.
//
// One unreachable axis is an unreachable target however comfortable the others
// are, because a target is met only when every axis it names is.
func (s *ReachableTestSuite) TestWorthNamesTheAxisThatDecides() {
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
}

// TestWorthOnNothingMeasured covers a chain with no readings at all.
func (s *ReachableTestSuite) TestWorthOnNothingMeasured() {
	_, ok := Worth(nil)
	s.Require().False(ok)
}

func TestReachableTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReachableTestSuite))
}

// TestTheOrderIsTheSameWhicheverWayTheMapWasWalked covers the comparator.
//
// Two axes out of reach and two inside, handed over in both orders. A map has
// no order, so a comparator right in one direction and wrong in the other
// reports a different worst axis run to run, and the worst axis is the whole
// answer.
func (s *ReachableTestSuite) TestTheOrderIsTheSameWhicheverWayTheMapWasWalked() {
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
}

// TestTwoAxesEquallyFarOrderTheSameWayTwice covers the last tie-break.
func (s *ReachableTestSuite) TestTwoAxesEquallyFarOrderTheSameWayTwice() {
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
}

// TestWorthOnATargetEveryAxisOfWhichIsInside covers the other exit.
func (s *ReachableTestSuite) TestWorthOnATargetEveryAxisOfWhichIsInside() {
	worst, ok := Worth(Reachable(one(200), spanOf(100, 50, 250, 0)))

	s.Require().True(ok)
	s.Require().Equal(audio.KeyCentroid, worst.Figure,
		"the worst is still named when the answer is yes")
}
