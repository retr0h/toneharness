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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// CurvesPublicTestSuite covers what every control of one block does.
type CurvesPublicTestSuite struct {
	suite.Suite
}

// at is one measured position of a control.
func at(
	value, centroid float64,
) measured.Point {
	return measured.Point{
		Value:   value,
		Figures: measured.Figures{Centroid: centroid},
	}
}

// TestFittedReadsAStraightLine covers the ordinary case.
func (s *CurvesPublicTestSuite) TestFittedReadsAStraightLine() {
	got := measured.Fitted([]measured.Point{
		at(0, 100), at(0.5, 600), at(1, 1100),
	}, "centroid")

	s.Require().InDelta(1000, got.PerTurn, 0.001,
		"a thousand hertz across a full turn")
	s.Require().InDelta(1, got.Straight, 0.001,
		"a line accounts for all of a line")
}

// TestFittedSaysWhenACurveIsNotALine is what keeps a slope honest.
//
// This amplifier's Master leaps from 130Hz to 10,795Hz over one step and then
// falls steadily to 3,769Hz. No single slope is true anywhere along it, and
// the average of a rise and a fall describes neither.
func (s *CurvesPublicTestSuite) TestFittedSaysWhenACurveIsNotALine() {
	got := measured.Fitted([]measured.Point{
		at(0.125, 130), at(0.25, 10795), at(0.5, 8220),
		at(0.75, 5055), at(1, 3769),
	}, "centroid")

	s.Require().Less(got.Straight, 0.3,
		"a line explains almost none of a rise followed by a fall")
}

// TestFittedRefusesToGuessFromTooLittle covers a curve with one point.
func (s *CurvesPublicTestSuite) TestFittedRefusesToGuessFromTooLittle() {
	for _, tt := range []struct {
		name   string
		points []measured.Point
	}{
		{name: "one position", points: []measured.Point{at(0.5, 100)}},
		{name: "none at all"},
		{
			// Every reading at the same setting is not a sweep. A slope
			// through them is a division by zero wearing a number.
			name:   "the same position twice",
			points: []measured.Point{at(0.5, 100), at(0.5, 200)},
		},
	} {
		s.Run(tt.name, func() {
			got := measured.Fitted(tt.points, "centroid")

			s.Require().Zero(got.PerTurn)
			s.Require().InDelta(1, got.Straight, 0.001)
		})
	}
}

// TestFittedIgnoresAFigureThatIsAbsent covers transient and decay.
//
// A transient needs a note starting and a decay needs one ending, so a
// reading can hold neither. Counting an absence as zero would put a step in
// the curve that nothing measured.
func (s *CurvesPublicTestSuite) TestFittedIgnoresAFigureThatIsAbsent() {
	known := 0.5
	points := []measured.Point{
		{Value: 0, Figures: measured.Figures{Transient: &known}},
		{Value: 0.5, Figures: measured.Figures{}},
		{Value: 1, Figures: measured.Figures{Transient: &known}},
	}

	got := measured.Fitted(points, "transient")

	s.Require().Zero(got.PerTurn, "two equal readings have no slope between them")
}

// TestFittedOnAFigureThatDoesNotMove covers a flat curve.
func (s *CurvesPublicTestSuite) TestFittedOnAFigureThatDoesNotMove() {
	got := measured.Fitted([]measured.Point{
		at(0, 100), at(0.5, 100), at(1, 100),
	}, "centroid")

	s.Require().Zero(got.PerTurn)
	s.Require().InDelta(1, got.Straight, 0.001,
		"a figure with no shape to miss is not a line the fit got wrong")
}

// TestApartIsWhatAListHasInsteadOfASlope covers a categorical control.
func (s *CurvesPublicTestSuite) TestApartIsWhatAListHasInsteadOfASlope() {
	got, ok := measured.Apart([]measured.Point{
		at(0, 146.9), at(1, 133.7), at(6, 125.9), at(11, 134.7),
	}, "centroid")

	s.Require().True(ok)
	s.Require().InDelta(21, got, 0.001, "146.9 down to 125.9")
}

// TestApartSaysWhenNothingAnswered covers a figure absent everywhere.
func (s *CurvesPublicTestSuite) TestApartSaysWhenNothingAnswered() {
	_, ok := measured.Apart([]measured.Point{at(0, 100)}, "decay")

	s.Require().False(ok,
		"no reading carried a decay, which is not a spread of zero")
}

// TestEveryNamedFigureCanBeRead holds the list to the type.
//
// A figure named here and unreadable there is a column of zeroes in every
// curve, which reads as a control that moves nothing.
func (s *CurvesPublicTestSuite) TestEveryNamedFigureCanBeRead() {
	known := 1.0
	full := measured.Point{Value: 1, Figures: measured.Figures{
		Centroid: 1, Level: 1, Low: 1, Mid: 1, High: 1,
		Dynamics: &known, Harmonics: &known, Lean: &known,
		Transient: &known, Decay: &known,
	}}

	for _, name := range measured.Named() {
		s.Run(string(name), func() {
			got, ok := measured.Apart([]measured.Point{full, full}, name)

			s.Require().True(ok, "%s cannot be read off a point", name)
			s.Require().Zero(got)
		})
	}

	s.Require().Len(measured.Named(), 10)
}

// TestAFigureNoReadingCarriedIsAbsentRatherThanZero covers the older sweeps.
//
// Dynamics, harmonics and lean were added after the first curves were
// measured, so a file taken before that carries five figures where the type
// names ten. Decoded into plain numbers they came back as zero and could not
// be told from a control that genuinely does not move one: a whole column of
// zeroes fits a perfectly straight line, and Fitted would report the slope
// with a straightness of one.
func (s *CurvesPublicTestSuite) TestAFigureNoReadingCarriedIsAbsentRatherThanZero() {
	older, err := measured.LoadCurves(strings.NewReader(`{
      "controls": {"Bass": {"index": 1, "control": "Bass", "points": [
        {"value": 0, "centroid": 100, "level": -20, "low": 90, "mid": 8, "high": 2},
        {"value": 1, "centroid": 200, "level": -18, "low": 80, "mid": 15, "high": 5}
      ]}}
    }`))
	s.Require().NoError(err)

	points := older.Controls["Bass"].Points

	for _, name := range []audio.Figure{
		audio.KeyDynamics, audio.KeyHarmonics, audio.KeyLean,
	} {
		s.Run(string(name), func() {
			_, ok := measured.Apart(points, name)

			s.Require().False(ok,
				"%s was never measured here, which is not a move of zero", name)
		})
	}

	// The five that were measured still read, so the guard is about what is
	// missing rather than about the file being old.
	for _, name := range []audio.Figure{
		audio.KeyCentroid, audio.KeyLevel, audio.KeyLow,
		audio.KeyMid, audio.KeyHigh,
	} {
		s.Run(string(name), func() {
			_, ok := measured.Apart(points, name)

			s.Require().True(ok)
		})
	}
}

// TestAFigureNobodyNamedIsNotInvented covers an unknown name.
func (s *CurvesPublicTestSuite) TestAFigureNobodyNamedIsNotInvented() {
	_, ok := measured.Apart([]measured.Point{at(0, 1), at(1, 2)}, "loudness")

	s.Require().False(ok)
}

// TestLoadCurvesReadsWhatWasWritten covers the round trip.
func (s *CurvesPublicTestSuite) TestLoadCurvesReadsWhatWasWritten() {
	got, err := measured.LoadCurves(strings.NewReader(`{
	  "device":"HX Stomp","gear":"2x15 Brute","block":"HD2_CabMicIr_2x15Brute",
	  "slot":1,"isolated":true,
	  "controls":{"Mic":{"index":0,"kind":"int","control":"Mic",
	    "points":[{"value":0,"centroid":146.9}]}}
	}`))

	s.Require().NoError(err)
	s.Require().True(got.Isolated)
	s.Require().Equal("int", got.Controls["Mic"].Kind)
}

// TestLoadCurvesRefusesWhatIsNotCurves covers the two ways reading fails.
func (s *CurvesPublicTestSuite) TestLoadCurvesRefusesWhatIsNotCurves() {
	for _, tt := range []struct{ name, give, want string }{
		{name: "not JSON", give: "{", want: "decoding the curves"},
		{
			// A document that parses and names nothing is worse than one
			// that fails: every lookup against it answers "that control was
			// not measured" rather than "there are no curves here".
			name: "no controls",
			give: `{"device":"HX Stomp"}`,
			want: "name no controls",
		},
	} {
		s.Run(tt.name, func() {
			_, err := measured.LoadCurves(strings.NewReader(tt.give))

			s.Require().ErrorContains(err, tt.want)
		})
	}
}

// TestLoadCurvesReportsAReadFailure covers the reader itself failing.
func (s *CurvesPublicTestSuite) TestLoadCurvesReportsAReadFailure() {
	_, err := measured.LoadCurves(broken{})

	s.Require().ErrorContains(err, "reading the curves")
}

func TestCurvesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CurvesPublicTestSuite))
}
