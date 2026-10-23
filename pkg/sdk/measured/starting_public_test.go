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
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// StartingPublicTestSuite covers where a proposed block's control starts.
//
// A block added with nothing said about its controls arrives at the catalog's
// defaults, and the defaults are not a position chosen for the job the block
// was picked for. The sweeps are the only evidence that could choose one.
type StartingPublicTestSuite struct {
	suite.Suite
}

// brighter is a chain that needs its centre of gravity raised.
func (s *StartingPublicTestSuite) brighter() []measured.Want {
	return []measured.Want{{Figure: audio.KeyCentroid, Need: 900, Tol: 50}}
}

// curve is a dial whose top position raises the centroid well past the noise.
//
// Levels sit where a working chain sits: clear of the converters' ceiling at
// -2dB and clear of the floor this curve reports.
func (s *StartingPublicTestSuite) curve() measured.Curve {
	return measured.Curve{
		Control:     "Treble",
		Noise:       map[audio.Figure]float64{audio.KeyCentroid: 1},
		Settled:     -20,
		SilentBelow: -50,
		Points: []measured.Point{
			{Value: 0, Figures: measured.Figures{Centroid: 100, Level: -20}},
			{Value: 0.5, Figures: measured.Figures{Centroid: 110, Level: -20}},
			{Value: 1, Figures: measured.Figures{Centroid: 400, Level: -20}},
		},
	}
}

// TestStartAt covers the position it picks and every way it picks none.
func (s *StartingPublicTestSuite) TestStartAt() {
	for _, tt := range []struct {
		name string
		of   catalog.Category
		in   measured.Curves
		want []measured.Want
		keep []measured.Want

		found   bool
		control string
		value   float64
		figure  audio.Figure
	}{
		{
			// The top position moves the centroid 290 times the noise beside
			// it, against 10 times for the middle one.
			name:    "the position a sweep argues for",
			of:      catalog.CategoryEQ,
			in:      measured.Curves{Controls: map[string]measured.Curve{"Treble": s.curve()}},
			want:    s.brighter(),
			found:   true,
			control: "Treble",
			value:   1,
			figure:  audio.KeyCentroid,
		},
		{
			// Every sweep is of one block alone, and an amplifier alone has no
			// cabinet lowpassing it, so what comes back is the buzz a cabinet
			// exists to absorb. 43 of the 121 committed curves hold a position
			// reading more than half its energy high and all 43 are
			// amplifiers.
			name:  "an amplifier swept with no cabinet",
			of:    catalog.CategoryAmp,
			in:    measured.Curves{Isolated: true, Controls: map[string]measured.Curve{"Treble": s.curve()}},
			want:  s.brighter(),
			found: false,
		},
		{
			// A curve whose working level is at the ceiling was swept through
			// an output turned up too far, so the figures move because the
			// converters are shaping the signal. All seven of an
			// HD2_AmpWoodyBlue's curves settle above this line.
			name: "a curve that settles at the ceiling",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.Settled = 0.46

					return c
				}(),
			}},
			want:  s.brighter(),
			found: false,
		},
		{
			// A solve may not move the valve's guts to close a band share, so
			// neither may a document written before the solve runs. Read from
			// the catalog rather than handed over, because a caller that
			// forgets gets no error and the wrong document: task 277.
			name: "the valve's guts",
			of:   catalog.CategoryAmp,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Sag": func() measured.Curve {
					c := s.curve()
					c.Control = "Sag"

					return c
				}(),
			}},
			want:  s.brighter(),
			found: false,
		},
		{
			// The position that would win carries the level 10dB from where
			// the curve settles, and level is held because it was already
			// inside its tolerance. That is a solve spending the level to buy
			// the tone, one step earlier.
			name: "a position that spends a satisfied axis",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.Points[2].Level = -10

					return c
				}(),
			}},
			want:  s.brighter(),
			keep:  []measured.Want{{Figure: audio.KeyLevel, Tol: 6}},
			found: false,
		},
		{
			// Two decibels below full scale, measured rather than taken from
			// the flag, because a Deluxe Comp's Level at its top stop reads
			// -1.14dB with an empty clippedAt.
			//
			// The top position is the one that would have won. What comes back
			// is the next best, which is the answer worth having: the guard
			// drops a reading of the converters rather than the whole curve.
			name: "a position at the converters' ceiling",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.Points[2].Level = -1

					return c
				}(),
			}},
			want:    s.brighter(),
			found:   true,
			control: "Treble",
			value:   0.5,
			figure:  audio.KeyCentroid,
		},
		{
			// Quiet enough that the figures describe hiss. The curve says
			// where that is, so there is no threshold to pick. The next
			// position down still answers.
			name: "a position below the curve's own floor",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.Points[2].Level = -60

					return c
				}(),
			}},
			want:    s.brighter(),
			found:   true,
			control: "Treble",
			value:   0.5,
			figure:  audio.KeyCentroid,
		},
		{
			// A position the sweep flagged, which is the case the measured
			// ceiling is a floor under rather than a replacement for.
			name: "a position the sweep flagged as clipped",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.ClippedAt = []float64{1}

					return c
				}(),
			}},
			want:    s.brighter(),
			found:   true,
			control: "Treble",
			value:   0.5,
			figure:  audio.KeyCentroid,
		},
		{
			name: "a position the sweep flagged as muted",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.MutedAt = []float64{1}

					return c
				}(),
			}},
			want:    s.brighter(),
			found:   true,
			control: "Treble",
			value:   0.5,
			figure:  audio.KeyCentroid,
		},
		{
			// Three noise-widths is the bar. One is the size of the
			// measurement's own wobble, and the sweeps are full of controls
			// that clear it while deciding nothing.
			name: "a move no larger than the noise",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.Noise = map[audio.Figure]float64{audio.KeyCentroid: 1000}

					return c
				}(),
			}},
			want:  s.brighter(),
			found: false,
		},
		{
			// A control that reads the same everywhere moves nothing, which is
			// not the same as moving something by less than the noise: there
			// is no direction to score at all.
			name: "a control that moves it nowhere",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					for i := range c.Points {
						c.Points[i].Centroid = 100
					}

					return c
				}(),
			}},
			want:  s.brighter(),
			found: false,
		},
		{
			// A curve carrying none of the wanted axes cannot be compared,
			// which is different from comparing to zero.
			name: "a curve that says nothing about the wanted axis",
			of:   catalog.CategoryEQ,
			in:   measured.Curves{Controls: map[string]measured.Curve{"Treble": s.curve()}},
			want: []measured.Want{
				{Figure: audio.KeyDecay, Need: 1, Tol: 0.1},
			},
			found: false,
		},
		{
			// Darker, and this curve's bottom position is darker than its
			// middle. A needed direction that points down is scored the same
			// way as one that points up.
			name:    "a position that moves it the way a lower aim needs",
			of:      catalog.CategoryEQ,
			in:      measured.Curves{Controls: map[string]measured.Curve{"Treble": s.curve()}},
			want:    []measured.Want{{Figure: audio.KeyCentroid, Need: -900, Tol: 50}},
			found:   true,
			control: "Treble",
			value:   0,
			figure:  audio.KeyCentroid,
		},
		{
			// A figure the curve records no noise for cannot be scored in
			// noise-widths, which is the only unit here. Skipped rather than
			// scored against a floor of zero, which would make every move
			// infinitely convincing.
			name: "a figure the curve records no noise for",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": func() measured.Curve {
					c := s.curve()
					c.Noise = nil

					return c
				}(),
			}},
			want:  s.brighter(),
			found: false,
		},
		{
			// A position carrying none of the wanted axes. The five figures
			// that can be absent are absent rather than zero, because a
			// reading taken before they were measured has a gap.
			name: "positions that carry none of the wanted axes",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": {
					Control:     "Treble",
					Noise:       map[audio.Figure]float64{audio.KeyDecay: 0.01},
					Settled:     -20,
					SilentBelow: -50,
					Points: []measured.Point{
						{Value: 0, Figures: measured.Figures{Level: -20}},
						{Value: 1, Figures: measured.Figures{Level: -20}},
					},
				},
			}},
			want:  []measured.Want{{Figure: audio.KeyDecay, Need: 1, Tol: 0.1}},
			found: false,
		},
		{
			// Only level has a within-sweep reference to hold it to, so a kept
			// axis that is not level passes here. Said rather than quietly
			// dropped: it is a real gap, and the position that wins below
			// moves the centroid well past any tolerance a caller would set.
			name:    "a kept axis that is not level",
			of:      catalog.CategoryEQ,
			in:      measured.Curves{Controls: map[string]measured.Curve{"Treble": s.curve()}},
			want:    s.brighter(),
			keep:    []measured.Want{{Figure: audio.KeyCentroid, Tol: 1}},
			found:   true,
			control: "Treble",
			value:   1,
			figure:  audio.KeyCentroid,
		},
		{
			// Three wanted axes against a curve that is uneven about them, and
			// the five figures that can be absent are why: a reading taken
			// before they were measured carries a gap rather than a zero. The
			// centroid is on every position, decay on one, dynamics on none.
			//
			// So the curve has somewhere to sit for two of the three and the
			// third is skipped, and the position that carries no decay is
			// skipped for that axis while still being scored for the centroid.
			// It wins on the centroid, which is the answer: an axis a curve
			// cannot speak to is not an axis it got wrong.
			name: "a curve uneven about which axes its positions carry",
			of:   catalog.CategoryEQ,
			in: measured.Curves{Controls: map[string]measured.Curve{
				"Treble": {
					Control: "Treble",
					Noise: map[audio.Figure]float64{
						audio.KeyCentroid: 1,
						audio.KeyDecay:    0.01,
						audio.KeyDynamics: 0.05,
					},
					Settled:     -20,
					SilentBelow: -50,
					Points: []measured.Point{
						{Value: 0, Figures: measured.Figures{
							Centroid: 100, Level: -20,
							Decay: func() *float64 { v := 1.0; return &v }(),
						}},
						{Value: 1, Figures: measured.Figures{
							Centroid: 400, Level: -20,
						}},
					},
				},
			}},
			want: []measured.Want{
				{Figure: audio.KeyCentroid, Need: 900, Tol: 50},
				{Figure: audio.KeyDecay, Need: 1, Tol: 0.1},
				{Figure: audio.KeyDynamics, Need: 1, Tol: 0.1},
			},
			found:   true,
			control: "Treble",
			value:   1,
			figure:  audio.KeyCentroid,
		},
		{
			name:  "a block with no curves at all",
			of:    catalog.CategoryEQ,
			in:    measured.Curves{},
			want:  s.brighter(),
			found: false,
		},
		{
			name:  "a chain that wants nothing",
			of:    catalog.CategoryEQ,
			in:    measured.Curves{Controls: map[string]measured.Curve{"Treble": s.curve()}},
			found: false,
		},
	} {
		s.Run(tt.name, func() {
			got, found := measured.StartAt(tt.in, tt.of, tt.want, tt.keep)

			s.Require().Equal(tt.found, found)

			if !tt.found {
				return
			}

			s.Require().Equal(tt.control, got.Control)
			s.Require().InDelta(tt.value, got.Value, 1e-9)
			s.Require().Equal(tt.figure, got.Figure)
			s.Require().Greater(got.Noises, 3.0)
		})
	}
}

func TestStartingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(StartingPublicTestSuite))
}
