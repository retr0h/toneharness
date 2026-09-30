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

// TestFittedReadsTheSlopeOffACurve covers what a straight line is and when a curve is not one.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestFittedReadsTheSlopeOffACurve() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "fitted reads a straight line",
			then: func() {
				got := measured.Fitted([]measured.Point{
					at(0, 100), at(0.5, 600), at(1, 1100),
				}, "centroid")

				s.Require().InDelta(1000, got.PerTurn, 0.001,
					"a thousand hertz across a full turn")
				s.Require().InDelta(1, got.Straight, 0.001,
					"a line accounts for all of a line")
			},
		},
		{
			// falls steadily to 3,769Hz. No single slope is true anywhere along it, and
			// the average of a rise and a fall describes neither.
			name: "fitted says when a curve is not a line",
			then: func() {
				got := measured.Fitted([]measured.Point{
					at(0.125, 130), at(0.25, 10795), at(0.5, 8220),
					at(0.75, 5055), at(1, 3769),
				}, "centroid")

				s.Require().Less(got.Straight, 0.3,
					"a line explains almost none of a rise followed by a fall")
			},
		},
		{
			name: "fitted refuses to guess from too little",
			then: func() {
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
			},
		},
		{
			// reading can hold neither. Counting an absence as zero would put a step in
			// the curve that nothing measured.
			name: "fitted ignores a figure that is absent",
			then: func() {
				known := 0.5
				points := []measured.Point{
					{Value: 0, Figures: measured.Figures{Transient: &known}},
					{Value: 0.5, Figures: measured.Figures{}},
					{Value: 1, Figures: measured.Figures{Transient: &known}},
				}

				got := measured.Fitted(points, "transient")

				s.Require().Zero(got.PerTurn, "two equal readings have no slope between them")
			},
		},
		{
			name: "fitted on a figure that does not move",
			then: func() {
				got := measured.Fitted([]measured.Point{
					at(0, 100), at(0.5, 100), at(1, 100),
				}, "centroid")

				s.Require().Zero(got.PerTurn)
				s.Require().InDelta(1, got.Straight, 0.001,
					"a figure with no shape to miss is not a line the fit got wrong")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestFittedSaysWhenACurveIsNotALine is what keeps a slope honest.
//

// TestFittedIgnoresAFigureThatIsAbsent covers transient and decay.
//

// TestApartIsWhatAListHasInsteadOfASlope covers the distance between a list's settings.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestApartIsWhatAListHasInsteadOfASlope() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "apart is what a list has instead of a slope",
			then: func() {
				got, ok := measured.Apart([]measured.Point{
					at(0, 146.9), at(1, 133.7), at(6, 125.9), at(11, 134.7),
				}, "centroid")

				s.Require().True(ok)
				s.Require().InDelta(21, got, 0.001, "146.9 down to 125.9")
			},
		},
		{
			name: "apart says when nothing answered",
			then: func() {
				_, ok := measured.Apart([]measured.Point{at(0, 100)}, "decay")

				s.Require().False(ok,
					"no reading carried a decay, which is not a spread of zero")
			},
		},
		{
			name: "a figure nobody named is not invented",
			then: func() {
				_, ok := measured.Apart([]measured.Point{at(0, 1), at(1, 2)}, "loudness")

				s.Require().False(ok)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestWanderIgnoresTheOneReadingThatDisagrees covers the noise floor.
//
// TestWanderIsHowFarRepeatedReadingsMove covers throwing away the reading that disagrees, and what is left.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestWanderIsHowFarRepeatedReadingsMove() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// once and then 31.56 to 31.73 five times over. Apart called that a wander of
			// twelve points of a band when five of the six agreed to three decimal places.
			name: "wander ignores the one reading that disagrees",
			then: func() {
				takes := []measured.Point{
					at(0, 19.3455), at(1, 31.6200), at(2, 31.6314),
					at(3, 31.7295), at(4, 31.5585), at(5, 31.7011),
				}

				apart, ok := measured.Apart(takes, "centroid")
				s.Require().True(ok)
				s.Require().InDelta(12.384, apart, 0.001, "the outlier sets both ends")

				got, ok := measured.Wander(takes, "centroid")
				s.Require().True(ok)
				s.Require().InDelta(0.171, got, 0.001, "31.5585 to 31.7295, without the first")
			},
		},
		{
			// is left is a reading that came back odd for some other reason, and where it
			// sits in the set is not knowable in advance.
			name: "wander throws away one reading whichever it is",
			then: func() {
				tests := []struct {
					name string
					of   []measured.Point
				}{
					{"first", []measured.Point{at(0, 99), at(1, 10), at(2, 11), at(3, 12)}},
					{"in the middle", []measured.Point{at(0, 10), at(1, 99), at(2, 11), at(3, 12)}},
					{"last", []measured.Point{at(0, 10), at(1, 11), at(2, 12), at(3, 99)}},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, ok := measured.Wander(tt.of, "centroid")

						s.Require().True(ok)
						s.Require().InDelta(2, got, 0.001, "10 to 12, whichever one was 99")
					})
				}
			},
		},
		{
			// of it: one reading is thrown away and the two that agree are the floor.
			name: "wander on three readings",
			then: func() {
				got, ok := measured.Wander([]measured.Point{
					at(0, 31.60), at(1, 31.70), at(2, 19.35),
				}, "centroid")

				s.Require().True(ok)
				s.Require().InDelta(0.1, got, 0.001, "31.60 to 31.70, without the odd one")
			},
		},
		{
			// discarding either would leave a spread of nothing and call it certainty.
			name: "wander keeps both of two",
			then: func() {
				got, ok := measured.Wander([]measured.Point{at(0, 10), at(1, 14)}, "centroid")

				s.Require().True(ok)
				s.Require().InDelta(4, got, 0.001)
			},
		},
		{
			name: "wander says when nothing answered",
			then: func() {
				_, ok := measured.Wander([]measured.Point{at(0, 100)}, "decay")

				s.Require().False(ok,
					"no reading carried a decay, which is not a spread of zero")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestWanderThrowsAwayOneReadingWhicheverItIs covers the outlier not being
// first.
//

// TestWanderOnThreeReadings covers what the loop actually asks for.
//

// TestWanderKeepsBothOfTwo covers too few readings to throw one away.
//

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
// TestLoadCurvesReadsWhatASweepWrote covers the document a sweep leaves, and what is refused.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestLoadCurvesReadsWhatASweepWrote() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// measured, so a file taken before that carries five figures where the type
			// names ten. Decoded into plain numbers they came back as zero and could not
			// be told from a control that genuinely does not move one: a whole column of
			// zeroes fits a perfectly straight line, and Fitted would report the slope
			// with a straightness of one.
			name: "a figure no reading carried is absent rather than zero",
			then: func() {
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
			},
		},
		{
			name: "load curves reads what was written",
			then: func() {
				got, err := measured.LoadCurves(strings.NewReader(`{
				  "device":"HX Stomp","gear":"2x15 Brute","block":"HD2_CabMicIr_2x15Brute",
				  "slot":1,"isolated":true,
				  "controls":{"Mic":{"index":0,"kind":"int","control":"Mic",
				    "points":[{"value":0,"centroid":146.9}]}}
				}`))

				s.Require().NoError(err)
				s.Require().True(got.Isolated)
				s.Require().Equal("int", got.Controls["Mic"].Kind)
			},
		},
		{
			name: "load curves refuses what is not curves",
			then: func() {
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
			},
		},
		{
			name: "load curves reports a read failure",
			then: func() {
				_, err := measured.LoadCurves(broken{})

				s.Require().ErrorContains(err, "reading the curves")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestCurvesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CurvesPublicTestSuite))
}
