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

package solve_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// DisplacePublicTestSuite covers a target stated as a distance from where the
// chain already is.
type DisplacePublicTestSuite struct {
	suite.Suite
}

// dirnt is one player's records and what the other players measure as, from the
// bass corpus: 0.0352 of the energy lower, 0.0358 less of it in the mids and 17Hz
// darker than the other twenty-seven bass players.
func (s *DisplacePublicTestSuite) dirnt() audio.Across {
	return audio.Across{
		Tracks:   3,
		Low:      audio.Spread{Low: 0.98, Mid: 0.9852, High: 0.99},
		Mid:      audio.Spread{Low: 0.01, Mid: 0.0142, High: 0.02},
		High:     audio.Spread{Low: 0.0004, Mid: 0.0006, High: 0.0008},
		Centroid: audio.Spread{Low: 110, Mid: 125.9102, High: 140},
	}
}

func (s *DisplacePublicTestSuite) others() map[audio.Figure]float64 {
	return map[audio.Figure]float64{
		audio.KeyLow:      0.95,
		audio.KeyMid:      0.05,
		audio.KeyHigh:     0,
		audio.KeyCentroid: 143,
	}
}

// chain is what a bass chain through a cabinet actually reads, in the units a
// corpus figure is stated in.
func (s *DisplacePublicTestSuite) chain() map[audio.Figure]float64 {
	return map[audio.Figure]float64{
		audio.KeyLow:      0.3156,
		audio.KeyMid:      0.6814,
		audio.KeyHigh:     0.0030,
		audio.KeyCentroid: 1807,
	}
}

// TestDisplaced covers Displaced, which moves a target to where the chain is plus
// how far the subject sits from everybody else.
//
// One method and one table, so a case is a row rather than a file.
func (s *DisplacePublicTestSuite) TestDisplaced() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The case the whole thing exists for. A record centres at 126Hz and a
			// chain through a cabinet sits at 1807Hz, so the old target asked for a
			// fourteenfold change nothing could reach, every control went to its
			// darkest, and the control with the most centroid authority on this
			// amplifier carries +27.6dB of level with it.
			name: "a record's figures become a distance the chain can travel",
			then: func() {
				got := solve.Displaced(s.dirnt(), s.others(), s.chain())

				s.Require().InDelta(0.3556, got.Low.Mid, 1e-9)
				s.Require().InDelta(0.6414, got.Mid.Mid, 1e-9)
				s.Require().InDelta(1790.0, got.Centroid.Mid, 1e-9)
			},
		},
		{
			// Each figure moves by how far the player sits from the others, which
			// is small and measured rather than large and unreachable.
			name: "the shift is the player against the others",
			then: func() {
				got := solve.Displaced(s.dirnt(), s.others(), s.chain())
				at := s.chain()

				s.Require().InDelta(+0.04, got.Low.Mid-at[audio.KeyLow], 1e-9)
				s.Require().InDelta(-0.04, got.Mid.Mid-at[audio.KeyMid], 1e-9)
				s.Require().InDelta(-17.0, got.Centroid.Mid-at[audio.KeyCentroid], 1e-9)
			},
		},
		{
			// The spread is untouched. How far a player's records disagree with
			// each other is a fact about the player, and it is what the tolerance
			// comes from either way.
			name: "the spread is left alone",
			then: func() {
				was := s.dirnt()
				got := solve.Displaced(was, s.others(), s.chain())

				s.Require().InDelta(was.Centroid.Low, got.Centroid.Low, 1e-9)
				s.Require().InDelta(was.Centroid.High, got.Centroid.High, 1e-9)
				s.Require().Equal(was.Tracks, got.Tracks)
			},
		},
		{
			// Nobody to be displaced from, which is one player measured. The
			// target is its own figures, which is what it was before any of this.
			name: "no comparison leaves the target alone",
			then: func() {
				was := s.dirnt()
				got := solve.Displaced(was, nil, s.chain())

				s.Require().Equal(was, got)
			},
		},
		{
			// A figure the others were not measured on keeps its own middle,
			// rather than the axis being dropped.
			name: "a figure with no comparison keeps its own middle",
			then: func() {
				others := s.others()
				delete(others, audio.KeyCentroid)

				got := solve.Displaced(s.dirnt(), others, s.chain())

				s.Require().InDelta(125.9102, got.Centroid.Mid, 1e-9)
				s.Require().InDelta(0.3556, got.Low.Mid, 1e-9)
			},
		},
		{
			// A figure the chain was not measured on, which is the same answer for
			// the same reason: there is no "here" to move from.
			name: "a figure the chain did not report keeps its own middle",
			then: func() {
				at := s.chain()
				delete(at, audio.KeyCentroid)

				got := solve.Displaced(s.dirnt(), s.others(), at)

				s.Require().InDelta(125.9102, got.Centroid.Mid, 1e-9)
			},
		},
		{
			// The other direction, so the arithmetic is not only tested on a player
			// who sits darker. Matt Freeman is 56Hz brighter than the other bass
			// players and holds more of the energy in the mids.
			//
			// 0.15 rather than the 0.145 the raw medians differ by, and 56Hz rather
			// than 55.6, because both sides of the comparison are rounded to the
			// places the figure is reported in: 0.185 reads as 0.19 against an 0.04
			// and 193.6Hz reads as 194 against a 138. That rounding is
			// deliberate and `elsewhere` says why, a share held to two places
			// subtracted from one held to seventeen leaves a shift where there is
			// no difference.
			name: "a brighter player moves the target the other way",
			then: func() {
				freeman := audio.Across{
					Tracks:   3,
					Mid:      audio.Spread{Low: 0.17, Mid: 0.1850, High: 0.20},
					Centroid: audio.Spread{Low: 180, Mid: 193.5972, High: 210},
				}
				others := map[audio.Figure]float64{
					audio.KeyMid: 0.04, audio.KeyCentroid: 138,
				}

				got := solve.Displaced(freeman, others, s.chain())

				s.Require().InDelta(0.6814+0.15, got.Mid.Mid, 1e-9)
				s.Require().InDelta(1807+56, got.Centroid.Mid, 1e-9)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestDisplacePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DisplacePublicTestSuite))
}
