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

// TowardPublicTestSuite covers setting a block's controls from its sweep.
type TowardPublicTestSuite struct {
	suite.Suite
}

// curve builds one swept control out of the readings given.
//
// Settled at -20dB and every point at -20 unless a case says otherwise, so a
// case has to opt in to the level window mattering.
func (s *TowardPublicTestSuite) curve(
	name string,
	noise float64,
	at ...measured.Point,
) measured.Curves {
	for i := range at {
		if at[i].Level == 0 {
			at[i].Level = -20
		}
	}

	return measured.Curves{
		Controls: map[string]measured.Curve{
			name: {
				Control:     name,
				Settled:     -20,
				SilentBelow: -50,
				Noise:       map[audio.Figure]float64{audio.KeyMid: noise},
				Points:      at,
			},
		},
	}
}

// TestToward reads the value of each control that lands nearest a target.
func (s *TowardPublicTestSuite) TestToward() {
	tests := []struct {
		name   string
		curves measured.Curves
		want   measured.Figures
		by     measured.Weights
		// set is the controls that must come back, at these values.
		set map[string]float64
	}{
		{
			name: "the nearest position wins",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 1}},
				measured.Point{Value: 0.5, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{Mid: 9},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Tone": 0.5},
		},
		{
			// The bug this guards. Matching figures cannot tell a quieter
			// block from a differently voiced one, and the first run of this
			// pinned an equaliser's Level at -60dB against a block that
			// settles at -21dB, which would have silenced the chain the moment
			// somebody stepped on it.
			name: "a position far below where the block settles is not offered",
			curves: s.curve("Level", 0.1,
				measured.Point{
					Value:   -60,
					Figures: measured.Figures{Mid: 9, Level: -45},
				},
				measured.Point{
					Value:   0,
					Figures: measured.Figures{Mid: 20, Level: -20},
				},
			),
			want: measured.Figures{Mid: 9},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Level": 0},
		},
		{
			// A control whose whole range moved the figure less than that
			// figure wandered between repeat takes moved nothing anybody can
			// demonstrate, so setting it spends a control on noise.
			name: "a control that moves less than its own noise is left alone",
			curves: s.curve("Hum", 5,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 12}},
			),
			want: measured.Figures{Mid: 20},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{},
		},
		{
			name: "a figure carrying no weight decides nothing",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{Mid: 20},
			by:   measured.Weights{High: 1},
			set:  map[string]float64{},
		},
		{
			name:   "a block with nothing swept sets nothing",
			curves: measured.Curves{},
			want:   measured.Figures{Mid: 9},
			by:     measured.Weights{Mid: 1},
			set:    map[string]float64{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := tt.curves.Toward(tt.want, tt.by)

			s.Require().Len(got, len(tt.set))

			for name, want := range tt.set {
				s.Require().InDelta(want, got[name].Value, 1e-9, name)
			}
		})
	}
}

// TestSpectralFor covers dropping the axes an instrument cannot use.
func (s *TowardPublicTestSuite) TestSpectralFor() {
	tests := []struct {
		name       string
		instrument string
		// flat is whether the low and high bands carry no weight.
		flat bool
	}{
		{
			// The high band displaces by exactly 0.00 across all eleven
			// genres the bass corpus can aim at, because a bass stem holds
			// almost nothing above 2kHz, and the three shares sum to one so
			// the low band then says nothing the mid band has not.
			name:       "a bass is ranked on its centroid and its midrange",
			instrument: "bass", flat: true,
		},
		{
			name:       "the case it is written in does not matter",
			instrument: "BASS", flat: true,
		},
		{
			// Only the bass corpus has been measured, so only the bass case
			// is a reading. A guitar's centre of gravity sits an octave up
			// and its high band is not dead.
			name:       "an instrument nobody measured keeps every axis",
			instrument: "guitar", flat: false,
		},
		{name: "and so does none at all", instrument: "", flat: false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := measured.SpectralFor(tt.instrument)

			s.Require().Equal(measured.Spectral().Centroid, got.Centroid,
				"the centroid is weighed whatever the instrument")

			if tt.flat {
				s.Require().Zero(got.Low)
				s.Require().Zero(got.High)

				return
			}

			s.Require().Equal(measured.Spectral(), got)
		})
	}
}

func TestTowardPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(TowardPublicTestSuite))
}

// TestTowardAcrossEveryWeighedAxis covers a control read on each figure.
//
// `moves` walks the four weighed axes and answers on the first that moved more
// than its own noise, so a control that moves only the centroid, or only the
// high band, has to be read as moving.
func (s *TowardPublicTestSuite) TestTowardAcrossEveryWeighedAxis() {
	for _, tt := range []struct {
		name string
		at   []measured.Point
		by   measured.Weights
		set  bool
	}{
		{
			name: "a control that only moves the centroid",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{Centroid: 100, Level: -20}},
				{Value: 1, Figures: measured.Figures{Centroid: 400, Level: -20}},
			},
			by:  measured.Weights{Centroid: 1.0 / 1000.0},
			set: true,
		},
		{
			name: "a control that only moves the low band",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{Low: 90, Level: -20}},
				{Value: 1, Figures: measured.Figures{Low: 70, Level: -20}},
			},
			by:  measured.Weights{Low: 1},
			set: true,
		},
		{
			name: "a control that only moves the high band",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{High: 1, Level: -20}},
				{Value: 1, Figures: measured.Figures{High: 9, Level: -20}},
			},
			by:  measured.Weights{High: 1},
			set: true,
		},
		{
			// Every position below where the block settles, so there is
			// nothing to choose between and the defaults stand.
			name: "a control swept entirely below working level",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{Mid: 10, Level: -60}},
				{Value: 1, Figures: measured.Figures{Mid: 20, Level: -55}},
			},
			by:  measured.Weights{Mid: 1},
			set: false,
		},
	} {
		s.Run(tt.name, func() {
			got := measured.Curves{
				Controls: map[string]measured.Curve{
					"It": {Control: "It", Settled: -20, SilentBelow: -50, Points: tt.at},
				},
			}.Toward(measured.Figures{}, tt.by)

			if !tt.set {
				s.Require().Empty(got)

				return
			}

			s.Require().Len(got, 1)
		})
	}
}
