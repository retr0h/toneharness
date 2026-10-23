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

package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

type RoundedPublicTestSuite struct {
	suite.Suite
}

// TestRounded covers how fine a computed value is written.
//
// One method and one table, so a case is a row rather than a file. Task 270 is
// the first half, a word's nudge claiming a precision a hundred thousand times
// finer than the arithmetic behind it, and task 275 is the second: a grid of
// thousandths no binary float can hold.
func (s *RoundedPublicTestSuite) TestRounded() {
	for _, tt := range []struct {
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
			// The case task 275 is about. An equaliser's band runs -15 to 15,
			// so a thousandth of its travel is a step of 0.03, and dividing
			// back out of that grid wrote -3.7499999999999996 into five
			// shipped documents. Hundredths hold it, and 0.01 is finer than
			// the 0.03 this promises never to go past.
			name: "a band in decibels keeps hundredths, exactly",
			at:   -3.75,
			p:    catalog.Param{Min: -15, Max: 15},
			want: -3.75,
		},
		{
			name: "and takes the digits off one that needs it",
			at:   -7.123456789,
			p:    catalog.Param{Min: -15, Max: 15},
			want: -7.12,
		},
		{
			// Of the control's own travel, not a flat thousandth. 1,452 of
			// this device's 4,835 float controls do not run zero to one: a
			// cabinet's LowCut runs 19.9 to 500, so a thousandth of its
			// travel is 0.48Hz and three decimal places would be five
			// significant figures of nothing.
			name: "a control running to five hundred keeps tenths of a hertz",
			at:   131.10999999999999,
			p:    catalog.Param{Min: 19.9, Max: 500},
			want: 131.1,
		},
		{
			// A compressor's Attack runs 0.0001 to 0.2, so a thousandth of
			// its travel is 0.0002 and ten-thousandths are what hold it.
			// Those places are the control's own resolution rather than
			// noise, which is what makes this of the travel rather than a
			// fixed count.
			name: "a narrow control keeps what its travel needs",
			at:   0.0629685,
			p:    catalog.Param{Min: 0.0001, Max: 0.2},
			want: 0.063,
		},
		{
			// A thousandth of its travel is wider than one unit, so there is
			// nothing a decimal place could add.
			name: "a control running over a thousand units keeps none",
			at:   1234.5678,
			p:    catalog.Param{Min: 0, Max: 2000},
			want: 1235,
		},
		{
			// The catalog gives a few controls a range of one value, and
			// taking a logarithm of that spread is the one way this could
			// fail.
			name: "a control with no travel is left as it is",
			at:   0.828986013986014,
			p:    catalog.Param{Min: 1, Max: 1},
			want: 0.828986013986014,
		},
		{
			// A parameter nothing filled in, which reads as no travel. It is
			// what a control the catalog does not carry comes back as.
			name: "and so is one the catalog says nothing about",
			at:   0.5,
			p:    catalog.Param{},
			want: 0.5,
		},
		{
			name: "a value already written down does not move",
			at:   0.5,
			p:    catalog.Param{Min: 0, Max: 1},
			want: 0.5,
		},
	} {
		s.Run(tt.name, func() {
			got := catalog.Rounded(tt.at, tt.p)

			s.Require().InDelta(tt.want, got, 1e-9)

			// Exactly, not nearly. The whole of task 275 is that a value came
			// off this with a float's tail on it, so a delta here would pass
			// on the thing being fixed.
			s.Require().Equal(tt.want, got)

			// Idempotent, so a document read back and written again does not
			// drift: a resolve runs over its own output.
			s.Require().Equal(got, catalog.Rounded(got, tt.p),
				"rounding twice moved it")
		})
	}
}

func TestRoundedPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(RoundedPublicTestSuite))
}
