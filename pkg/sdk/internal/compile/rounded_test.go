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

package compile

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// RoundedTestSuite covers how fine a value this package computed is kept.
//
// Task 270. `built:` carried controls like Treble 0.828986013986014 and Bass
// 0.6067073170731707, which claim a precision a hundred thousand times finer
// than the arithmetic that produced them.
type RoundedTestSuite struct {
	suite.Suite
}

// TestRounded covers rounding to a thousandth of a control's own travel.
//
// One method and one table, so a case is a row rather than a file.
func (s *RoundedTestSuite) TestRounded() {
	tests := []struct {
		name string
		at   float64
		p    catalog.Param
		want float64
	}{
		{
			// The value a word's nudge actually wrote, on a control running
			// zero to one, which is most of them.
			name: "a nudge on an ordinary knob keeps three places",
			at:   0.828986013986014,
			p:    catalog.Param{Min: 0, Max: 1},
			want: 0.829,
		},
		{
			// Float noise from adding three numbers, which is the other half
			// of what this is for: nothing chose the digit at the sixteenth
			// place.
			name: "and float noise goes with them",
			at:   0.9760000000000001,
			p:    catalog.Param{Min: 0, Max: 1},
			want: 0.976,
		},
		{
			// Of the control's own travel, not a flat thousandth. 1,452 of
			// this device's 4,835 float controls do not run zero to one: a
			// cabinet's LowCut runs 19.9 to 500, so a thousandth of its
			// travel is 0.4801Hz and three decimal places would be five
			// significant figures of nothing.
			name: "a control running to five hundred keeps halves of a hertz",
			at:   131.10999999999999,
			p:    catalog.Param{Min: 19.9, Max: 500},
			want: 131.0673,
		},
		{
			// And the other way: an equaliser's band runs -15 to 15, so a
			// thousandth of its travel is 0.03dB, which is far below anything
			// anybody hears and far above float noise.
			name: "a band in decibels keeps hundredths",
			at:   -7.123456789,
			p:    catalog.Param{Min: -15, Max: 15},
			want: -7.11,
		},
		{
			// A compressor's Attack runs 0.0001 to 0.2, so a thousandth of
			// its travel needs seven places to write exactly. Those digits
			// are the control's own resolution rather than noise, which is
			// what makes this of the travel rather than a fixed count.
			name: "a narrow control keeps what its travel needs",
			at:   0.0629685,
			p:    catalog.Param{Min: 0.0001, Max: 0.2},
			want: 0.0629685,
		},
		{
			// The catalog gives a few controls a range of one value, and
			// dividing by that spread is the one way this could fail.
			name: "a control with no travel is left as it is",
			at:   0.828986013986014,
			p:    catalog.Param{Min: 1, Max: 1},
			want: 0.828986013986014,
		},
		{
			// A parameter nothing filled in, which reads as no travel.
			name: "and so is one the catalog says nothing about",
			at:   0.5,
			p:    catalog.Param{},
			want: 0.5,
		},
		{
			name: "a value already on the grid does not move",
			at:   0.5,
			p:    catalog.Param{Min: 0, Max: 1},
			want: 0.5,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := rounded(tt.at, tt.p)

			s.Require().InDelta(tt.want, got, 1e-9)

			// Idempotent, so a document read back and written again does not
			// drift: a resolve runs over its own output.
			s.Require().InDelta(got, rounded(got, tt.p), 1e-12,
				"rounding twice moved it")
		})
	}
}

func TestRoundedTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(RoundedTestSuite))
}
