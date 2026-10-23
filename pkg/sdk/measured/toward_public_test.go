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

// at is a control's swept positions reading these mids, at working level and
// evenly spaced across the control's range.
func (s *TowardPublicTestSuite) at(
	mids ...float64,
) []measured.Point {
	out := make([]measured.Point, 0, len(mids))

	for i, mid := range mids {
		out = append(out, measured.Point{
			Value:   float64(i) / float64(len(mids)-1),
			Figures: measured.Figures{Mid: mid, Level: -20},
		})
	}

	return out
}

// TestToward reads the value of each control that leans the way a target asks.
//
// **A lean rather than a position, which is task 254's correction.** Scoring a
// swept position's own figures against the target directly is only sound while
// both sides came out of the same run, and they do not: the target is built
// over the fingerprint run's baseline and the points come from this block's own
// sweep, two runs that differ by a median 20.6Hz of centroid and 5.5dB of level
// for the same block. So each control is answered by its distance from its own
// median instead.
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
			// A target asking for as much mid as the control has reaches the
			// end of it. The size is read and not only the sign, so this is
			// the ask meeting the reach rather than every ask doing so.
			name: "a target leaning up reaches the top of the control",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 1}},
				measured.Point{Value: 0.5, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{Mid: 9},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Tone": 1},
		},
		{
			// And the other way, which is what makes this a reading of the
			// target rather than of the control's range.
			name: "a target leaning down reaches the bottom",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 1}},
				measured.Point{Value: 0.5, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{Mid: -9},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Tone": 0},
		},
		{
			// The bug the level window guards. Matching figures cannot tell a
			// quieter block from a differently voiced one, and the first run
			// of this pinned an equaliser's Level at -60dB against a block
			// that settles at -21dB, which would have silenced the chain the
			// moment somebody stepped on it.
			//
			// With one position left inside the window there is no lean to
			// read, so the control is left where the corpus put it rather
			// than set from the one reading that survived.
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
			set:  map[string]float64{},
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
			// A target that leans nowhere on the axis being weighed. Nothing
			// says which end of the control would be more like it.
			name: "a target asking for no displacement decides nothing",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{},
		},
		{
			// The fix for task 272. The size of the ask decides how far the
			// control goes, so an ask well inside what the control can do
			// lands part way rather than against the stop. A ten-band
			// equaliser came out with six of its bands at 15dB because every
			// ask read as the full extent of the control.
			name: "an ask inside the control's reach lands part way",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 1}},
				measured.Point{Value: 0.5, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{Mid: 2},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Tone": 0.5},
		},
		{
			// And an ask beyond it is not an ask for this control. The 16kHz
			// band of this device's ten-band moves a bass's midrange by 0.001
			// of a point against genres asking for two to twenty-two, and it
			// sat at 15dB delivering nothing.
			name: "an ask beyond the control's reach leaves it alone",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 1}},
				measured.Point{Value: 0.5, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{Mid: 40},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{},
		},
		{
			// The reach is measured on the side being asked, because a control
			// is rarely symmetric about its middle. One equaliser band here
			// adds 17.5 points of midrange and takes away 2.3, so a single
			// spread is the boost side and asked to cut, every cut in the
			// sweep read as too little and the deepest one won.
			//
			// This control adds 10 and takes away 1. An ask to cut by 2 is
			// beyond the cut side however far the boost side goes.
			name: "the reach is the side being asked for",
			curves: s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 9}},
				measured.Point{Value: 0.5, Figures: measured.Figures{Mid: 10}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			),
			want: measured.Figures{Mid: -2},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{},
		},
		{
			// A sweep moves one control and leaves the rest where they sat, so
			// asking every dial for the whole displacement asks for one thing
			// as many times as there are dials. Two here, so each is asked for
			// half, and half of 16 is inside what one of them reaches.
			//
			// Without the split both would go to their stops and the block
			// would deliver 16 twice over.
			name: "the ask is shared among the dials that can answer it",
			curves: measured.Curves{Controls: map[string]measured.Curve{
				"Bass": {
					Control: "Bass", Settled: -20, SilentBelow: -50,
					Noise:  map[audio.Figure]float64{audio.KeyMid: 0.1},
					Points: s.at(1, 10, 20),
				},
				"Treble": {
					Control: "Treble", Settled: -20, SilentBelow: -50,
					Noise:  map[audio.Figure]float64{audio.KeyMid: 0.1},
					Points: s.at(1, 10, 20),
				},
			}},
			want: measured.Figures{Mid: 16},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Bass": 1, "Treble": 1},
		},
		{
			// A control whose whole range moves the figure less than that
			// figure's own noise is not counted into the share, because
			// counting it would give a real dial a smaller one on the
			// strength of a reading that says nothing.
			name: "a dial that moves nothing takes no share",
			curves: measured.Curves{Controls: map[string]measured.Curve{
				"Tone": {
					Control: "Tone", Settled: -20, SilentBelow: -50,
					Noise:  map[audio.Figure]float64{audio.KeyMid: 0.1},
					Points: s.at(1, 10, 20),
				},
				"Output": {
					Control: "Output", Settled: -20, SilentBelow: -50,
					Noise:  map[audio.Figure]float64{audio.KeyMid: 0.1},
					Points: s.at(10, 10.05, 10),
				},
			}},
			want: measured.Figures{Mid: 9},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Tone": 1},
		},
		{
			// A list is chosen by direction whatever the size, which is the
			// distinction `solve.Choice` and `solve.Knob` are built on. There
			// is no twelfth of a microphone to ask for, and the sizes are not
			// comparable either: 126Hz across a population of records against
			// 22Hz across one cabinet's microphones.
			name: "a list is still reached by direction alone",
			curves: measured.Curves{Controls: map[string]measured.Curve{
				"Mic": {
					Control: "Mic", Kind: "int", Settled: -20, SilentBelow: -50,
					Noise:  map[audio.Figure]float64{audio.KeyMid: 0.1},
					Points: s.at(1, 10, 20),
				},
			}},
			want: measured.Figures{Mid: 400},
			by:   measured.Weights{Mid: 1},
			set:  map[string]float64{"Mic": 1},
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

// TestAsks covers whether a displacement asks for anything at all.
//
// One method and one table, so a case is a row rather than a file. The
// distinction it draws is between "no control of this block reaches what was
// asked" and "nothing was asked", which are two different answers and only one
// of them is about the block.
func (s *TowardPublicTestSuite) TestAsks() {
	for _, tt := range []struct {
		name  string
		asked measured.Figures
		by    measured.Weights
		want  bool
	}{
		{
			name:  "a displacement on a figure the weights carry",
			asked: measured.Figures{Mid: 9},
			by:    measured.Weights{Mid: 1},
			want:  true,
		},
		{
			// What three of the eleven genres are in on a bass, where
			// SpectralFor drops the low and high bands: the only figure
			// displaced is one nothing is weighed on.
			name:  "a displacement only on a figure they do not",
			asked: measured.Figures{Low: 9},
			by:    measured.Weights{Mid: 1, Centroid: 1.0 / 1000.0},
			want:  false,
		},
		{
			name:  "a weighed figure displaced by nothing",
			asked: measured.Figures{Mid: 0},
			by:    measured.Weights{Mid: 1},
			want:  false,
		},
		{
			// hard-rock, which displaces the centroid by zero and the
			// midrange by zero, and is the reason this exists.
			name:  "nothing displaced at all",
			asked: measured.Figures{},
			by:    measured.Weights{Mid: 1, Centroid: 1.0 / 1000.0},
			want:  false,
		},
		{
			name:  "the centroid alone",
			asked: measured.Figures{Centroid: 40},
			by:    measured.Weights{Centroid: 1.0 / 1000.0},
			want:  true,
		},
		{
			name:  "the high band alone",
			asked: measured.Figures{High: 4},
			by:    measured.Weights{High: 1},
			want:  true,
		},
		{
			name:  "the low band alone",
			asked: measured.Figures{Low: 4},
			by:    measured.Weights{Low: 1},
			want:  true,
		},
		{
			name:  "no weights to carry anything",
			asked: measured.Figures{Mid: 9, Centroid: 40},
			want:  false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.asked.Asks(tt.by))
		})
	}
}

// TestTowardRefusesAnIsolatedAmplifier covers the other half of task 254.
//
// A sweep of an amplifier alone has no cabinet in front of it and so no
// speaker rolloff, and what comes back is the buzz a cabinet exists to absorb:
// 43 of the 121 usable curves hold a position reading more than half its energy
// above 2kHz and all 43 are amplifiers. `Reach.Alone` records the size of it,
// and the part a lean cannot survive is that **four of eleven controls changed
// sign** in a real chain. `StartAt` has always refused one; this is the same
// refusal in the same package, which until now disagreed with itself.
func (s *TowardPublicTestSuite) TestTowardRefusesAnIsolatedAmplifier() {
	tests := []struct {
		name     string
		block    string
		isolated bool
		// set is whether any control comes back.
		set bool
	}{
		{
			name:  "an amplifier swept alone sets nothing",
			block: "HD2_AmpSVBeastBrt", isolated: true, set: false,
		},
		{
			// The front half of an amplifier with no cabinet after it, which
			// is the same hazard for the same reason.
			name:  "a preamp swept alone sets nothing either",
			block: "HD2_PreampSVT4Pro", isolated: true, set: false,
		},
		{
			// What task 271 would produce. A reading taken through a cabinet
			// describes a chain, and the refusal is about the isolation
			// rather than about the kind of block.
			name:  "an amplifier swept through a cabinet is read",
			block: "HD2_AmpSVBeastBrt", isolated: false, set: true,
		},
		{
			// Everything else, and that is the evidence rather than a line
			// drawn for tidiness: the inverted curves are all amplifiers, and
			// no drive, compressor or equaliser swept alone reads upside
			// down.
			name:  "a drive swept alone is read",
			block: "HD2_DistKinkyBoost", isolated: true, set: true,
		},
		{
			// A file written before the block identifier was recorded says
			// nothing about what it swept, and a prefix nothing matches is
			// not an amplifier.
			name:  "a sweep that does not say what it swept is read",
			block: "", isolated: true, set: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			curves := s.curve("Tone", 0.1,
				measured.Point{Value: 0, Figures: measured.Figures{Mid: 1}},
				measured.Point{Value: 1, Figures: measured.Figures{Mid: 20}},
			)
			curves.Block, curves.Isolated = tt.block, tt.isolated

			got := curves.Toward(
				measured.Figures{Mid: 9}, measured.Weights{Mid: 1})

			if !tt.set {
				s.Require().Empty(got,
					"an amplifier's controls have to stay where the corpus "+
						"medians put them")

				return
			}

			s.Require().Len(got, 1)
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
		want measured.Figures
		set  bool
	}{
		{
			name: "a control that only moves the centroid",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{Centroid: 100, Level: -20}},
				{Value: 1, Figures: measured.Figures{Centroid: 400, Level: -20}},
			},
			by:   measured.Weights{Centroid: 1.0 / 1000.0},
			want: measured.Figures{Centroid: 50},
			set:  true,
		},
		{
			name: "a control that only moves the low band",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{Low: 90, Level: -20}},
				{Value: 1, Figures: measured.Figures{Low: 70, Level: -20}},
			},
			by:   measured.Weights{Low: 1},
			want: measured.Figures{Low: -5},
			set:  true,
		},
		{
			name: "a control that only moves the high band",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{High: 1, Level: -20}},
				{Value: 1, Figures: measured.Figures{High: 9, Level: -20}},
			},
			by:   measured.Weights{High: 1},
			want: measured.Figures{High: 2},
			set:  true,
		},
		{
			// Every position below where the block settles, so there is
			// nothing to choose between and the defaults stand.
			name: "a control swept entirely below working level",
			at: []measured.Point{
				{Value: 0, Figures: measured.Figures{Mid: 10, Level: -60}},
				{Value: 1, Figures: measured.Figures{Mid: 20, Level: -55}},
			},
			by:   measured.Weights{Mid: 1},
			want: measured.Figures{Mid: 5},
			set:  false,
		},
	} {
		s.Run(tt.name, func() {
			got := measured.Curves{
				Controls: map[string]measured.Curve{
					"It": {Control: "It", Settled: -20, SilentBelow: -50, Points: tt.at},
				},
			}.Toward(tt.want, tt.by)

			if !tt.set {
				s.Require().Empty(got)

				return
			}

			s.Require().Len(got, 1)
		})
	}
}

// TestTowardFrom covers choosing a position by what it moves rather than by
// where it sits.
//
// The failure it exists for: a genre is measured off finished records and a
// cabinet off a dry signal through one block, so comparing the two directly
// sends every genre to whichever end of the control is closest. Eleven genres
// all chose the same microphone that way, and letting the same reading set the
// rest of the cabinet sent Distance to its maximum and LowCut to 79.9Hz.
func (s *TowardPublicTestSuite) TestTowardFrom() {
	// Three positions: one below the median on the centroid, one at it, one
	// above. A genre leaning either way should reach the matching end.
	twelve := measured.Curves{
		Controls: map[string]measured.Curve{
			"Mic": {
				Control: "Mic", Kind: "int", Settled: -20, SilentBelow: -50,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{
						Centroid: 90, Low: 97, High: 0.1, Level: -20,
					}},
					{Value: 1, Figures: measured.Figures{
						Centroid: 118, Low: 95, High: 0.4, Level: -20,
					}},
					{Value: 2, Figures: measured.Figures{
						Centroid: 140, Low: 91, High: 0.9, Level: -20,
					}},
				},
			},
		},
	}

	tests := []struct {
		name string
		// asked is the displacement the genre was given, and by the weights
		// it is read under. Empty weights read the centroid alone.
		asked measured.Figures
		by    measured.Weights
		want  float64
		ok    bool
	}{
		{
			// A genre brighter than the other genres reaches the microphone
			// brighter than the others on its cabinet. The sizes are not
			// comparable and are not compared: 126Hz across a population of
			// records against 22Hz across one box.
			name:  "a brighter genre reaches a brighter microphone",
			asked: measured.Figures{Centroid: 126},
			want:  2, ok: true,
		},
		{
			name:  "and a darker one a darker microphone",
			asked: measured.Figures{Centroid: -25},
			want:  0, ok: true,
		},
		{
			// Nothing asked for on a weighed axis, so there is nothing to lean
			// toward and the default stands.
			name:  "a genre asking for nothing chooses nothing",
			asked: measured.Figures{},
			ok:    false,
		},
		{
			// The low and high bands, read the same way, so each axis the
			// weights carry is exercised rather than only the centroid.
			name:  "the low band leans the other way from the centroid",
			asked: measured.Figures{Low: -9},
			by:    measured.Weights{Low: 1},
			want:  2, ok: true,
		},
		{
			name:  "and the high band",
			asked: measured.Figures{High: 4},
			by:    measured.Weights{High: 1},
			want:  2, ok: true,
		},
		{
			// A figure the weights carry but the sweep does not move. Every
			// position reads the same mid, so that axis offers no lean and the
			// centroid decides alone.
			name:  "an axis the positions do not move is passed over",
			asked: measured.Figures{Centroid: 126, Mid: 9},
			by:    measured.Weights{Centroid: 1.0 / 1000.0, Mid: 1},
			want:  2, ok: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			by := tt.by
			if by == (measured.Weights{}) {
				by = measured.Weights{Centroid: 1.0 / 1000.0}
			}

			got, ok := twelve.TowardFrom("Mic", tt.asked, by)

			s.Require().Equal(tt.ok, ok)

			if tt.ok {
				s.Require().InDelta(tt.want, got, 1e-9)
			}
		})
	}
}

// TestTowardFromRefuses covers the sweeps that offer no choice.
func (s *TowardPublicTestSuite) TestTowardFromRefuses() {
	at := func(points ...measured.Point) measured.Curves {
		return measured.Curves{Controls: map[string]measured.Curve{
			"Mic": {
				Control: "Mic", Settled: -20, SilentBelow: -50, Points: points,
			},
		}}
	}

	by := measured.Weights{Centroid: 1.0 / 1000.0}
	asked := measured.Figures{Centroid: 50}

	for _, tt := range []struct {
		name   string
		curves measured.Curves
		on     string
	}{
		{name: "a control nobody swept", curves: at(), on: "Mic"},
		{
			name:   "a control with one position",
			curves: at(measured.Point{Value: 0, Figures: measured.Figures{Centroid: 90, Level: -20}}),
			on:     "Mic",
		},
		{
			// Every position below working level, so there is nothing to
			// choose between: a quieter block reads as a close match to
			// anything.
			name: "positions all below working level",
			curves: at(
				measured.Point{Value: 0, Figures: measured.Figures{Centroid: 90, Level: -60}},
				measured.Point{Value: 1, Figures: measured.Figures{Centroid: 140, Level: -55}},
			),
			on: "Mic",
		},
		{
			// Every position reading alike offers no choice either.
			name: "positions that all read the same",
			curves: at(
				measured.Point{Value: 0, Figures: measured.Figures{Centroid: 118, Level: -20}},
				measured.Point{Value: 1, Figures: measured.Figures{Centroid: 118, Level: -20}},
			),
			on: "Mic",
		},
		{name: "a control the block does not have", curves: at(), on: "Distance"},
	} {
		s.Run(tt.name, func() {
			_, ok := tt.curves.TowardFrom(tt.on, asked, by)
			s.Require().False(ok)
		})
	}
}
