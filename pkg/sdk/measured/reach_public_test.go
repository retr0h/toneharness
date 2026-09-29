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

package measured_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// ReachPublicTestSuite covers reading a chain's limits out of its sweeps.
type ReachPublicTestSuite struct {
	suite.Suite
}

// dial is one control swept across a range, with a slope through its readings.
func dial(
	low, high float64,
	fit measured.Fit,
	at ...float64,
) measured.Curve {
	out := measured.Curve{
		Span: measured.Span{Low: low, High: high},
		Fits: map[audio.Figure]measured.Fit{audio.KeyCentroid: fit},
	}

	for i, v := range at {
		out.Points = append(out.Points, measured.Point{
			Value:   float64(i),
			Figures: measured.Figures{Centroid: v},
		})
	}

	return out
}

func curves(
	controls map[string]measured.Curve,
) measured.Curves {
	return measured.Curves{Controls: controls}
}

// TestTheRangeIsWhatWasActuallyRead is the number worth trusting.
//
// Nothing modelled. These are readings the hardware produced, so a target
// inside them is one some setting of this chain demonstrably hit.
func (s *ReachPublicTestSuite) TestTheRangeIsWhatWasActuallyRead() {
	got := measured.Reaches(curves(map[string]measured.Curve{
		"Treble": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 1}, 120, 300, 900),
	}))

	at := got[audio.KeyCentroid]
	s.Require().InDelta(120, at.Low, 0.001)
	s.Require().InDelta(900, at.High, 0.001)
	s.Require().InDelta(300, at.Mid, 0.001, "the middle reading, not the mean")
	s.Require().Equal(3, at.Readings)
}

// TestSwingIsTheSlopeAcrossTheWholeRange covers a dial's contribution.
//
// Across the span it was swept over rather than nought to one: 1,452 of this
// device's float controls are neither.
func (s *ReachPublicTestSuite) TestSwingIsTheSlopeAcrossTheWholeRange() {
	got := measured.Reaches(curves(map[string]measured.Curve{
		"Distance": dial(1, 12, measured.Fit{PerTurn: 10, Straight: 1}, 100, 200),
	}))

	s.Require().InDelta(110, got[audio.KeyCentroid].Swing, 0.001,
		"ten per unit across a range of eleven")
}

// TestABentSlopeContributesLessThanAStraightOne is what keeps Swing honest.
//
// Straight is the fraction of a figure's movement the line accounts for, so a
// control whose readings rise and then fall has a slope describing neither
// half. This amplifier's Bias reads 0.21 straight and its Hum 0.11, and
// counting their whole range as movement is arithmetic on a number with no
// referent.
func (s *ReachPublicTestSuite) TestABentSlopeContributesLessThanAStraightOne() {
	bent := measured.Reaches(curves(map[string]measured.Curve{
		"Bias": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 0.2}, 100, 200),
	}))

	straight := measured.Reaches(curves(map[string]measured.Curve{
		"Treble": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 1}, 100, 200),
	}))

	s.Require().InDelta(20, bent[audio.KeyCentroid].Swing, 0.001)
	s.Require().InDelta(100, straight[audio.KeyCentroid].Swing, 0.001)
	s.Require().InDelta(0.2, bent[audio.KeyCentroid].Straight, 0.001)
}

// TestAListContributesTheDistanceBetweenItsSettings covers a control with no
// slope at all.
//
// Counted as fully straight, because straightness is a statement about a line
// and a list is not one. A cabinet's twelve microphones do not lie on one.
func (s *ReachPublicTestSuite) TestAListContributesTheDistanceBetweenItsSettings() {
	mic := measured.Curve{
		Span:   measured.Span{Low: 0, High: 11},
		Spread: map[audio.Figure]float64{audio.KeyCentroid: 45},
		Points: []measured.Point{
			{Value: 0, Figures: measured.Figures{Centroid: 120}},
			{Value: 1, Figures: measured.Figures{Centroid: 165}},
		},
	}

	got := measured.Reaches(curves(map[string]measured.Curve{"Mic": mic}))

	s.Require().InDelta(45, got[audio.KeyCentroid].Swing, 0.001)
	s.Require().InDelta(1, got[audio.KeyCentroid].Straight, 0.001)
}

// TestEveryBlockInTheChainCounts covers a chain rather than a block.
//
// A chain is its blocks, and the question is what the chain can do rather than
// what any one of them can.
func (s *ReachPublicTestSuite) TestEveryBlockInTheChainCounts() {
	got := measured.Reaches(
		curves(map[string]measured.Curve{
			"Treble": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 1}, 100, 200),
		}),
		curves(map[string]measured.Curve{
			"HighCut": dial(0, 1, measured.Fit{PerTurn: 50, Straight: 1}, 80, 300),
		}),
	)

	at := got[audio.KeyCentroid]
	s.Require().InDelta(150, at.Swing, 0.001, "both blocks' movement")
	s.Require().InDelta(80, at.Low, 0.001)
	s.Require().InDelta(300, at.High, 0.001)
	s.Require().Equal(4, at.Readings)
}

// TestAFigureNoReadingCarriedIsAbsent covers an axis nothing measured.
//
// Absent rather than zero, because a chain nobody swept on an axis is not a
// chain sitting at nought on it, and the difference decides whether somebody
// is told a target is out of reach.
func (s *ReachPublicTestSuite) TestAFigureNoReadingCarriedIsAbsent() {
	got := measured.Reaches(curves(map[string]measured.Curve{
		"Treble": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 1}, 100, 200),
	}))

	s.Require().NotContains(got, audio.KeyTransient,
		"nothing measured a transient here")
	s.Require().Contains(got, audio.KeyCentroid)
}

// TestNoSweepsAtAll covers being handed nothing.
func (s *ReachPublicTestSuite) TestNoSweepsAtAll() {
	s.Require().Empty(measured.Reaches())
}

// TestAControlWithNoSlopeAndNoSpreadAddsNothing covers a curve that was
// measured but produced neither.
func (s *ReachPublicTestSuite) TestAControlWithNoSlopeAndNoSpreadAddsNothing() {
	bare := measured.Curve{
		Span: measured.Span{Low: 0, High: 1},
		Points: []measured.Point{
			{Value: 0, Figures: measured.Figures{Centroid: 100}},
			{Value: 1, Figures: measured.Figures{Centroid: 140}},
		},
	}

	got := measured.Reaches(curves(map[string]measured.Curve{"Bare": bare}))

	at := got[audio.KeyCentroid]
	s.Require().InDelta(0, at.Swing, 0.001, "nothing says how far it moves")
	s.Require().InDelta(100, at.Low, 0.001, "but its readings still happened")
	s.Require().InDelta(140, at.High, 0.001)
}

func TestReachPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReachPublicTestSuite))
}

// TestASweepTakenWithTheBlockAloneSaysSo is the field that decides whether the
// answer is about the chain.
//
// Every committed sweep is isolated, and summing them describes a set of
// blocks apart rather than a chain. Measured on an HX Stomp: an SV Beast swept
// with no cabinet reads its Treble at 12,763Hz of centroid per turn, the same
// control in the chain a rig builds reads 3,250, and four of its eleven
// controls change sign.
func (s *ReachPublicTestSuite) TestASweepTakenWithTheBlockAloneSaysSo() {
	one := curves(map[string]measured.Curve{
		"Treble": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 1}, 100, 200),
	})
	one.Isolated = true

	s.Require().True(measured.Reaches(one)[audio.KeyCentroid].Alone)

	together := curves(map[string]measured.Curve{
		"Treble": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 1}, 100, 200),
	})

	s.Require().False(measured.Reaches(together)[audio.KeyCentroid].Alone)
}

// TestOneIsolatedSweepTaintsTheWholeAnswer covers a mixed chain.
//
// A chain measured half in place and half apart is not half trustworthy: the
// blocks swept apart are in the same sum as the ones swept in place, so the
// sum is of both.
func (s *ReachPublicTestSuite) TestOneIsolatedSweepTaintsTheWholeAnswer() {
	apart := curves(map[string]measured.Curve{
		"Treble": dial(0, 1, measured.Fit{PerTurn: 100, Straight: 1}, 100, 200),
	})
	apart.Isolated = true

	inPlace := curves(map[string]measured.Curve{
		"HighCut": dial(0, 1, measured.Fit{PerTurn: 50, Straight: 1}, 80, 300),
	})

	s.Require().True(measured.Reaches(inPlace, apart)[audio.KeyCentroid].Alone)
}
