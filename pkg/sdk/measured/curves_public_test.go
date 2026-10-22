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

// TestFitted covers Fitted, which is the slope of one figure across a
// control's measured positions.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestFitted() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary case.
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
			// What keeps a slope honest.
			//
			// This amplifier's Master leaps from 130Hz to 10,795Hz over one
			// step and then falls steadily to 3,769Hz. No single slope is
			// true anywhere along it, and the average of a rise and a fall
			// describes neither.
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
			// A curve with one point.
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
			// Transient and decay.
			//
			// A transient needs a note starting and a decay needs one ending,
			// so a reading can hold neither. Counting an absence as zero
			// would put a step in the curve that nothing measured.
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
			// A flat curve.
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

// TestApart covers Apart, which is how far a figure's readings sit from each
// other.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestApart() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A categorical control.
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
			// A figure absent everywhere.
			name: "apart says when nothing answered",
			then: func() {
				_, ok := measured.Apart([]measured.Point{at(0, 100)}, "decay")

				s.Require().False(ok,
					"no reading carried a decay, which is not a spread of zero")
			},
		},
		{
			// An unknown name.
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

// TestWander covers Wander, which is how far a figure's readings sit from
// each other once the one furthest from the middle is thrown away.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestWander() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The noise floor.
			//
			// The real numbers: six takes of one untouched chain read the low
			// band at 19.35 once and then 31.56 to 31.73 five times over.
			// Apart called that a wander of twelve points of a band when five
			// of the six agreed to three decimal places.
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
			// The outlier not being first.
			//
			// The settling take always is, and it is discarded before these
			// are taken. What is left is a reading that came back odd for
			// some other reason, and where it sits in the set is not knowable
			// in advance.
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
			// What the loop actually asks for.
			//
			// Three takes is the default, so this is the ordinary case rather
			// than an edge of it: one reading is thrown away and the two that
			// agree are the floor.
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
			// Too few readings to throw one away.
			//
			// With two there is no middle for one of them to be furthest
			// from, and discarding either would leave a spread of nothing and
			// call it certainty.
			name: "wander keeps both of two",
			then: func() {
				got, ok := measured.Wander([]measured.Point{at(0, 10), at(1, 14)}, "centroid")

				s.Require().True(ok)
				s.Require().InDelta(4, got, 0.001)
			},
		},
		{
			// A figure absent everywhere.
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

// TestLoadCurves covers LoadCurves, which reads what every control of one
// block does.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestLoadCurves() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The older sweeps.
			//
			// Dynamics, harmonics and lean were added after the first curves
			// were measured, so a file taken before that carries five figures
			// where the type names ten. Decoded into plain numbers they came
			// back as zero and could not be told from a control that
			// genuinely does not move one: a whole column of zeroes fits a
			// perfectly straight line, and Fitted would report the slope with
			// a straightness of one.
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
			// The round trip.
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
			// The two ways reading fails.
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
			// The reader itself failing.
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

// TestCarries covers Carries, which is how far a slope read off a control may
// be taken.
//
// Task 253: the controls a least-squares solve models best are the ones that
// reach a figure by removing signal, and the ones that matter musically are
// the ones it models worst. A drive's MinFreq reads 0.99 straight over
// 2,399Hz; an equaliser's Mode reads 0.00 over 1,463Hz.
//
// One method and one table, so a case is a row rather than a file.
func (s *CurvesPublicTestSuite) TestCarries() {
	fits := func(straight ...float64) map[audio.Figure]measured.Fit {
		keys := []audio.Figure{
			audio.KeyCentroid, audio.KeyLow, audio.KeyMid, audio.KeyHigh,
		}

		out := map[audio.Figure]measured.Fit{}
		for i, v := range straight {
			out[keys[i]] = measured.Fit{PerTurn: 1, Straight: v}
		}

		return out
	}

	tests := []struct {
		name  string
		curve measured.Curve
		// figures is the axes being solved. Empty reads every fit.
		figures []audio.Figure
		// want is the fraction, and fitted whether there was one to give.
		want   float64
		fitted bool
	}{
		{
			// A filter, which is what a solve models well and what nobody
			// reaches for to voice a chain.
			name:   "a control that is a line carries almost its whole range",
			curve:  measured.Curve{Fits: fits(0.99)},
			want:   0.99,
			fitted: true,
		},
		{
			// A voicing control. A slope read here is true where it was read
			// and nowhere else, which is the whole of this task.
			name:   "a control that is not a line carries nothing",
			curve:  measured.Curve{Fits: fits(0)},
			want:   0,
			fitted: true,
		},
		{
			// The worst figure, which is the choice Reach makes for the same
			// reason: a solve puts every axis in one matrix, so a control that
			// is a line for the centroid and a hairpin for the mid band is not
			// a control whose slope carries.
			name:   "the worst figure decides",
			curve:  measured.Curve{Fits: fits(0.98, 0.95, 0.02, 0.9)},
			want:   0.02,
			fitted: true,
		},
		{
			// Straight is 1 minus a ratio of sums of squares, so it goes
			// negative where the line misses by more than the readings' own
			// spread. A negative fraction of a range is not a shorter
			// distance, it is no distance.
			name:   "a line that misses by more than the spread carries nothing",
			curve:  measured.Curve{Fits: fits(-3.4)},
			want:   0,
			fitted: true,
		},
		{
			// Clamped the other way too, because nothing may be moved further
			// than its own range on the strength of a rounding.
			name:   "straightness above one is still one range",
			curve:  measured.Curve{Fits: fits(1.2)},
			want:   1,
			fitted: true,
		},
		{
			// No figures named, which is a caller asking about the control
			// rather than about a target. Every fit counts, and the decay a
			// drive's Bass moved by less than its own noise is one of them.
			name:   "no figures named reads every fit",
			curve:  measured.Curve{Fits: fits(0.98, 0.95, 0.02, 0.9)},
			want:   0.02,
			fitted: true,
		},
		{
			// A figure this control has no fit for, which answers the same
			// way a list does: there is nothing to carry a slope over.
			name:    "a figure with no fit carries nothing",
			curve:   measured.Curve{Fits: fits(0.98)},
			figures: []audio.Figure{audio.KeyDecay},
			fitted:  false,
		},
		{
			// Every list. A list has no order, so a line through it is
			// meaningless by construction and there is no distance to carry a
			// slope over: a cabinet's twelve microphones are compared rather
			// than solved.
			name:   "a list has no fit and no carry",
			curve:  measured.Curve{Spread: map[audio.Figure]float64{audio.KeyCentroid: 11}},
			fitted: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, fitted := tt.curve.Carries(tt.figures...)

			s.Require().Equal(tt.fitted, fitted)
			s.Require().InDelta(tt.want, got, 1e-9)
		})
	}
}

func TestCurvesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CurvesPublicTestSuite))
}
