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

	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

type MiddlePublicTestSuite struct {
	suite.Suite
}

// some is a figure that may be absent, as the noisy ones are.
func some(v float64) *float64 { return &v }

// TestMiddle covers the median of several readings of the same thing.
//
// One method and one table, so a case is a row rather than a file.
func (s *MiddlePublicTestSuite) TestMiddle() {
	for _, tt := range []struct {
		name  string
		takes []measured.Figures
		then  func(got measured.Figures)
	}{
		{
			// The reading this exists for. Five consecutive reads of an
			// untouched rig spanned 56 Hz, and the drift gate that calls a
			// rewiring is 25 Hz, so the single wild take decided the answer.
			// The middle three here are what the rig actually reads.
			name: "one wild take does not decide it",
			takes: []measured.Figures{
				{Centroid: 195.6, Level: -60.27},
				{Centroid: 159.7, Level: -60.07},
				{Centroid: 173.8, Level: -59.56},
			},
			then: func(got measured.Figures) {
				s.Require().InDelta(173.8, got.Centroid, 1e-9)
				s.Require().InDelta(-60.07, got.Level, 1e-9)
			},
		},
		{
			// An even count has no middle reading, so the two nearest it are
			// averaged rather than one of them chosen.
			name: "an even count averages the two in the middle",
			takes: []measured.Figures{
				{Centroid: 100, Level: -10},
				{Centroid: 200, Level: -20},
			},
			then: func(got measured.Figures) {
				s.Require().InDelta(150.0, got.Centroid, 1e-9)
				s.Require().InDelta(-15.0, got.Level, 1e-9)
			},
		},
		{
			// A figure absent from every take stays absent. Zero would read
			// as measured, and a control whose harmonics are zero everywhere
			// fits a straight line through nothing and reports it with total
			// confidence.
			name:  "a figure nothing measured stays absent",
			takes: []measured.Figures{{Centroid: 100}, {Centroid: 200}, {Centroid: 300}},
			then: func(got measured.Figures) {
				s.Require().Nil(got.Transient)
				s.Require().Nil(got.Decay)
				s.Require().Nil(got.Dynamics)
				s.Require().Nil(got.Harmonics)
				s.Require().Nil(got.Lean)
			},
		},
		{
			// A transient needs a note starting, so a take that found none is
			// silent about it rather than claiming zero. Dropping the takes
			// that did find one would throw away the only answers there are.
			name: "a figure some takes found is the middle of those",
			takes: []measured.Figures{
				{Transient: some(0.9)},
				{},
				{Transient: some(0.5)},
			},
			then: func(got measured.Figures) {
				s.Require().NotNil(got.Transient)
				s.Require().InDelta(0.7, *got.Transient, 1e-9)
			},
		},
		{
			name:  "one take is itself",
			takes: []measured.Figures{{Centroid: 155.27, Level: -58.55, Transient: some(0.4)}},
			then: func(got measured.Figures) {
				s.Require().InDelta(155.27, got.Centroid, 1e-9)
				s.Require().NotNil(got.Transient)
				s.Require().InDelta(0.4, *got.Transient, 1e-9)
			},
		},
		{
			// Rather than a panic. Nothing measured is not a reading.
			name:  "no takes at all",
			takes: nil,
			then: func(got measured.Figures) {
				s.Require().Equal(measured.Figures{}, got)
			},
		},
		{
			// Every band, because the baseline is subtracted from each of
			// them and a figure left at zero would calibrate nothing.
			name: "every band gets its own middle",
			takes: []measured.Figures{
				{Low: 90, Mid: 7, High: 3},
				{Low: 80, Mid: 15, High: 5},
				{Low: 85, Mid: 11, High: 4},
			},
			then: func(got measured.Figures) {
				s.Require().InDelta(85.0, got.Low, 1e-9)
				s.Require().InDelta(11.0, got.Mid, 1e-9)
				s.Require().InDelta(4.0, got.High, 1e-9)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then(measured.Middle(tt.takes))
		})
	}
}

func TestMiddlePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MiddlePublicTestSuite))
}
