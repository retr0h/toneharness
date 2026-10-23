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

// HarmonicsPublicTestSuite covers the fourth figure a bass is weighed on.
//
// Task 276. Harmonics is the one figure beyond the band shares that both sides
// of a comparison measure the same way: a record and a block go through the
// same `audio.Measure`, and unlike transient and decay the measuring loop's
// second of padding cannot move a share of the energy above the fundamental.
type HarmonicsPublicTestSuite struct {
	suite.Suite
}

// at is a figure with a reading of harmonics on it.
func (s *HarmonicsPublicTestSuite) at(
	harmonics float64,
) measured.Figures {
	held := harmonics

	return measured.Figures{Harmonics: &held}
}

// TestSpectralWeighsHarmonics covers the weight itself.
func (s *HarmonicsPublicTestSuite) TestSpectralWeighsHarmonics() {
	// At one, like a band share, because it is reported on the same scale: a
	// block's reading and a genre's displacement are both hundredths of the
	// energy.
	s.Require().InDelta(1.0, measured.Spectral().Harmonics, 1e-12)

	// And kept for a bass, where the low and high bands are dropped. Those
	// two are dead on this corpus and harmonics is not: four of the eleven
	// genres displace it past what the figure can claim.
	s.Require().InDelta(
		1.0, measured.SpectralFor("bass").Harmonics, 1e-12)
}

// TestHarmonicsCountsTowardTheDistance covers the ranking reading it.
//
// One method and one table, so a case is a row rather than a file.
func (s *HarmonicsPublicTestSuite) TestHarmonicsCountsTowardTheDistance() {
	lib := measured.Library{Blocks: map[string]measured.Block{
		// Two blocks reading the same spectrum, differing only on harmonics.
		"Clean": {
			ID: "Clean", Category: "drive",
			Figures: measured.Figures{
				Centroid: 150, Mid: 9, Level: -20, Harmonics: s.held(10),
			},
		},
		"Dirty": {
			ID: "Dirty", Category: "drive",
			Figures: measured.Figures{
				Centroid: 150, Mid: 9, Level: -20, Harmonics: s.held(40),
			},
		},
		// And one with no reading of it at all, which is what a sweep taken
		// before the figure existed holds.
		"Older": {
			ID: "Older", Category: "drive",
			Figures: measured.Figures{Centroid: 150, Mid: 9, Level: -20},
		},
	}}

	for _, tt := range []struct {
		name string
		want measured.Figures
		by   measured.Weights
		// nearer is two blocks the first of which must rank strictly nearer.
		nearer [2]string
		// alike is two that must rank identically.
		alike [2]string
	}{
		{
			name:   "a target asking for few harmonics reaches the clean one",
			want:   measured.Figures{Centroid: 150, Mid: 9, Harmonics: s.held(12)},
			by:     measured.SpectralFor("bass"),
			nearer: [2]string{"Clean", "Dirty"},
		},
		{
			name:   "and one asking for many reaches the other",
			want:   measured.Figures{Centroid: 150, Mid: 9, Harmonics: s.held(38)},
			by:     measured.SpectralFor("bass"),
			nearer: [2]string{"Dirty", "Clean"},
		},
		{
			// The weight at zero is the whole of what it used to be, so two
			// blocks differing only on harmonics rank alike.
			name:  "unweighed, the two are the same block",
			want:  measured.Figures{Centroid: 150, Mid: 9, Harmonics: s.held(38)},
			by:    measured.Weights{Centroid: 1.0 / 1000.0, Mid: 1},
			alike: [2]string{"Clean", "Dirty"},
		},
		{
			// A reading with no harmonics is not a reading of none: the axis
			// is passed over rather than scored as a miss, which is what
			// `displacedTo` does for a figure one side carries and the other
			// does not.
			//
			// So it can out-rank a block that genuinely matches, and that is
			// a state no shipped reading is in: all 613 usable blocks in the
			// committed library carry harmonics, and so does its baseline.
			// Counting an absent one as zero instead would put every one of
			// those 613 at 38 points of distance from a target asking for any
			// harmonics at all, which is the failure worth avoiding.
			name:   "a block with no reading of it is not scored on it",
			want:   measured.Figures{Centroid: 150, Mid: 9, Harmonics: s.held(38)},
			by:     measured.SpectralFor("bass"),
			nearer: [2]string{"Older", "Dirty"},
		},
		{
			// And the other way: a target with no reading cannot ask for one.
			name:  "a target with no reading of it asks nothing of it",
			want:  measured.Figures{Centroid: 150, Mid: 9},
			by:    measured.SpectralFor("bass"),
			alike: [2]string{"Clean", "Dirty"},
		},
	} {
		s.Run(tt.name, func() {
			at := map[string]float64{}
			for _, m := range lib.Nearest("drive", tt.want, tt.by) {
				at[m.ID] = m.Distance
			}

			if tt.nearer[0] != "" {
				s.Require().Less(at[tt.nearer[0]], at[tt.nearer[1]],
					"%s ranks nearer than %s", tt.nearer[0], tt.nearer[1])

				return
			}

			s.Require().InDelta(
				at[tt.alike[0]], at[tt.alike[1]], 1e-12,
				"%s and %s rank alike", tt.alike[0], tt.alike[1])
		})
	}
}

// held is a reading of one figure, as a pointer.
func (s *HarmonicsPublicTestSuite) held(
	at float64,
) *float64 {
	return &at
}

// TestOf covers which field a figure key means.
//
// One method and one table, so a case is a row rather than a file. Every key
// the corpus names, because this is the one owner of the question and three
// readers had the switch written out before it.
func (s *HarmonicsPublicTestSuite) TestOf() {
	full := measured.Figures{
		Centroid: 150, Level: -20, Low: 90, Mid: 9, High: 0.3,
		Harmonics: s.held(37.8), Dynamics: s.held(6.1), Lean: s.held(0.88),
		Transient: s.held(0.74), Decay: s.held(0.41),
	}

	for _, tt := range []struct {
		key  audio.Figure
		want float64
	}{
		{audio.KeyCentroid, 150},
		{audio.KeyLevel, -20},
		{audio.KeyLow, 90},
		{audio.KeyMid, 9},
		{audio.KeyHigh, 0.3},
		{audio.KeyHarmonics, 37.8},
		{audio.KeyDynamics, 6.1},
		{audio.KeyLean, 0.88},
		{audio.KeyTransient, 0.74},
		{audio.KeyDecay, 0.41},
		// A key nothing names, which reads as the centroid: the figure every
		// reading carries and the one a caller with no key means.
		{audio.Figure("not-a-figure"), 150},
	} {
		s.Run(string(tt.key), func() {
			s.Require().InDelta(tt.want, measured.Of(full, tt.key), 1e-9)
		})
	}

	// And a reading carrying none of the five that may be absent reads them
	// as zero, which every caller handles by the span it leaves rather than
	// by the value.
	bare := measured.Figures{Centroid: 150}
	for _, key := range []audio.Figure{
		audio.KeyHarmonics, audio.KeyDynamics, audio.KeyLean,
		audio.KeyTransient, audio.KeyDecay,
	} {
		s.Require().Zero(measured.Of(bare, key), string(key))
	}
}

// TestOrZeroAndPair covers the two readers of a figure that may be absent.
func (s *HarmonicsPublicTestSuite) TestOrZeroAndPair() {
	held := 37.8

	s.Require().InDelta(37.8, measured.OrZero(&held), 1e-9)
	s.Require().Zero(measured.OrZero(nil))

	mine, theirs, both := measured.Pair(&held, &held)
	s.Require().True(both)
	s.Require().InDelta(37.8, mine, 1e-9)
	s.Require().InDelta(37.8, theirs, 1e-9)

	_, _, both = measured.Pair(&held, nil)
	s.Require().False(both)

	_, _, both = measured.Pair(nil, &held)
	s.Require().False(both)
}

func TestHarmonicsPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(HarmonicsPublicTestSuite))
}

// TestAsksReadsHarmonics covers a displacement that asks for harmonics alone.
//
// One method and one table, so a case is a row rather than a file. hard-rock
// displaces nothing a bass is weighed on and a genre displacing harmonics and
// nothing else would read the same way without this.
func (s *HarmonicsPublicTestSuite) TestAsksReadsHarmonics() {
	by := measured.SpectralFor("bass")

	for _, tt := range []struct {
		name  string
		asked measured.Figures
		by    measured.Weights
		want  bool
	}{
		{
			name:  "harmonics alone is still an ask",
			asked: s.at(-6.33),
			by:    by,
			want:  true,
		},
		{
			name:  "a displacement of none of it is not",
			asked: s.at(0),
			by:    by,
		},
		{
			name:  "nor is no reading of it",
			asked: measured.Figures{},
			by:    by,
		},
		{
			// What it read as before task 276 weighed it.
			name:  "and nor is it where nothing weighs it",
			asked: s.at(-6.33),
			by:    measured.Weights{Centroid: 1.0 / 1000.0, Mid: 1},
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.asked.Asks(tt.by))
		})
	}
}

// TestSharedSplitsHarmonics covers the apportionment reading it too.
//
// A sweep moves one control and leaves the rest where they sat, so asking
// every dial for the whole displacement asks for one thing as many times as
// there are dials. The figure being a pointer is why it cannot join the table
// the other four are in.
func (s *HarmonicsPublicTestSuite) TestSharedSplitsHarmonics() {
	at := func(mids ...float64) []measured.Point {
		out := make([]measured.Point, 0, len(mids))
		for i, mid := range mids {
			held := mid

			out = append(out, measured.Point{
				Value: float64(i) / float64(len(mids)-1),
				Figures: measured.Figures{
					Mid: mid, Level: -20, Harmonics: &held,
				},
			})
		}

		return out
	}

	curves := measured.Curves{Controls: map[string]measured.Curve{
		"Bass": {
			Control: "Bass", Settled: -20, SilentBelow: -50,
			Noise:  map[audio.Figure]float64{audio.KeyHarmonics: 0.1},
			Points: at(1, 10, 20),
		},
		"Treble": {
			Control: "Treble", Settled: -20, SilentBelow: -50,
			Noise:  map[audio.Figure]float64{audio.KeyHarmonics: 0.1},
			Points: at(1, 10, 20),
		},
	}}

	// Two dials move it past their own noise, so each is asked for half.
	got := measured.Shared(
		curves, s.at(16), measured.Weights{Harmonics: 1})

	s.Require().NotNil(got.Harmonics)
	s.Require().InDelta(8.0, *got.Harmonics, 1e-9)

	// And the caller's own figures are untouched, because the two point at
	// the same number until one of them is replaced.
	want := s.at(16)
	_ = measured.Shared(curves, want, measured.Weights{Harmonics: 1})
	s.Require().InDelta(16.0, *want.Harmonics, 1e-9)

	// Nothing to split where one dial alone can answer, which is the common
	// case and the one every drive on this device is in.
	one := measured.Curves{Controls: map[string]measured.Curve{
		"Tone": curves.Controls["Bass"],
	}}

	got = measured.Shared(one, s.at(16), measured.Weights{Harmonics: 1})
	s.Require().InDelta(16.0, *got.Harmonics, 1e-9)

	// And nothing where it is not weighed at all.
	got = measured.Shared(curves, s.at(16), measured.Weights{Mid: 1})
	s.Require().InDelta(16.0, *got.Harmonics, 1e-9)
}
