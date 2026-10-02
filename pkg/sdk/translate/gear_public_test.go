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

package translate_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// GearPublicTestSuite covers which blocks a build may draw from, and what a
// genre is aimed at.
//
// Built catalogs rather than the shipped one. Every amplifier the library
// measures is in the catalog and all three measured genres have a population
// behind them, so the states these answer cannot be reached through the data
// that ships, and they are still the contract.
type GearPublicTestSuite struct {
	suite.Suite
}

// built is a catalog holding exactly these blocks.
func (s *GearPublicTestSuite) built(
	blocks map[catalog.ModelID]catalog.Block,
) *catalog.Catalog {
	return &catalog.Catalog{Device: "HX Stomp", Blocks: blocks}
}

// ranked is a shortlist naming these models, as Nearest would answer.
func (s *GearPublicTestSuite) ranked(
	ids ...string,
) []measured.Match {
	out := make([]measured.Match, 0, len(ids))
	for _, id := range ids {
		out = append(out, measured.Match{Block: measured.Block{ID: id}})
	}

	return out
}

// TestPlayable covers whether a block is for the instrument in hand.
//
// One method and one table, so a case is a row rather than a file.
func (s *GearPublicTestSuite) TestPlayable() {
	held := s.built(map[catalog.ModelID]catalog.Block{
		"bass":   {Subcategory: "Bass"},
		"guitar": {Subcategory: "Guitar"},
		"blank":  {},
	})

	for _, tt := range []struct {
		name       string
		id         string
		instrument string
		want       bool
	}{
		{
			name: "the instrument's own", id: "bass", instrument: "bass",
			want: true,
		},
		{
			// Whatever case each side was written in. An instrument comes from a
			// document somebody wrote and a subcategory from a file Line 6 wrote.
			name: "whatever case either was written in", id: "guitar", instrument: "GUITAR",
			want: true,
		},
		{
			name: "the other instrument", id: "guitar", instrument: "bass",
		},
		{
			// Twenty-three amplifiers carry no subcategory and nothing else in
			// Line 6's data places them. This is the tool guessing rather than
			// somebody choosing, so it declines.
			name: "a block nobody marked", id: "blank", instrument: "bass",
		},
		{
			// Measured and not in the catalog, which is the two drifting apart.
			// Kept, because refusing it would hide the drift behind a chain that
			// is merely shorter.
			name: "a block the catalog does not hold", id: "nowhere", instrument: "bass",
			want: true,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, translate.Playable(held, tt.id, tt.instrument))
		})
	}
}

// TestSplitsByInstrument covers whether a category is grouped by what it is
// played with.
func (s *GearPublicTestSuite) TestSplitsByInstrument() {
	for _, tt := range []struct {
		name   string
		blocks map[catalog.ModelID]catalog.Block
		ids    []string
		want   bool
	}{
		{
			// Amplifiers, which Line 6 groups "Guitar" and "Bass".
			name: "a category that names an instrument",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Guitar"}, "b": {},
			},
			ids:  []string{"a", "b"},
			want: true,
		},
		{
			// Cabinets, which it groups "Single, Dual", by how many microphones
			// they offer. Requiring a match here excluded every cabinet on the
			// device, because none claims to be for an instrument.
			name: "a category that names something else",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Single, Dual"},
			},
			ids: []string{"a"},
		},
		{
			name: "a shortlist the catalog does not hold",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Bass"},
			},
			ids: []string{"nowhere"},
		},
		{
			name: "no catalog at all",
			ids:  []string{"a"},
		},
	} {
		s.Run(tt.name, func() {
			var held *catalog.Catalog
			if tt.blocks != nil {
				held = s.built(tt.blocks)
			}

			s.Require().Equal(tt.want,
				translate.SplitsByInstrument(held, s.ranked(tt.ids...)))
		})
	}
}

// TestDisplacedTo covers turning a genre into a target a block can be ranked
// against.
//
// The arithmetic is the whole of it: a genre is measured off finished records
// and a block off a dry signal pushed through it, so the genre's own figures do
// not subtract from a block's. Its displacement from other records does.
func (s *GearPublicTestSuite) TestDisplacedTo() {
	baseline := measured.Figures{Low: 90, Mid: 9, High: 0.3, Centroid: 155}

	for _, tt := range []struct {
		name string
		got  audio.Genre
		// want is the target, where there is one.
		want  measured.Figures
		found bool
	}{
		{
			// Shares convert and the centroid does not. A genre holds a share of
			// the energy from zero to one; the library reports it as a
			// percentage.
			name: "a genre is applied as a displacement",
			got: audio.Genre{
				Across: audio.Across{
					Tracks:   1,
					Low:      audio.Spread{Mid: 0.97},
					Mid:      audio.Spread{Mid: 0.03},
					High:     audio.Spread{Mid: 0.001},
					Centroid: audio.Spread{Mid: 144},
				},
				// Both sides on the same grid, as the generator writes them: a
				// share held to two places subtracted from one held to
				// seventeen leaves a shift where there is no difference.
				Elsewhere: map[audio.Figure]float64{
					audio.KeyLow: 0.95, audio.KeyMid: 0.05,
					audio.KeyHigh: 0, audio.KeyCentroid: 134,
				},
			},
			// Two points of energy lower and ten hertz brighter than the records
			// elsewhere, applied to the signal the blocks were measured with.
			want:  measured.Figures{Low: 92, Mid: 7, High: 0.3, Centroid: 165},
			found: true,
		},
		{
			// Nobody to be measured against, which is a genre with fewer than two
			// other players. How far it sits from other records is then not
			// known, and no block can be aimed at it.
			name: "a genre with nobody to compare against",
			got: audio.Genre{
				Across: audio.Across{Tracks: 1, Centroid: audio.Spread{Mid: 144}},
			},
		},
		{
			// A figure one side holds and the other does not is skipped rather
			// than treated as a zero shift, which would aim at the baseline and
			// call that a measurement.
			name: "a figure only one side carries",
			got: audio.Genre{
				Across: audio.Across{
					Tracks:   1,
					Centroid: audio.Spread{Mid: 144},
				},
				Elsewhere: map[audio.Figure]float64{audio.KeyCentroid: 134},
			},
			want:  measured.Figures{Low: 90, Mid: 9, High: 0.3, Centroid: 165},
			found: true,
		},
	} {
		s.Run(tt.name, func() {
			got, ok := translate.DisplacedTo(tt.got, baseline)

			s.Require().Equal(tt.found, ok)

			if !tt.found {
				return
			}

			s.Require().InDelta(tt.want.Low, got.Low, 0.01)
			s.Require().InDelta(tt.want.Mid, got.Mid, 0.01)
			s.Require().InDelta(tt.want.High, got.High, 0.01)
			s.Require().InDelta(tt.want.Centroid, got.Centroid, 0.01)
		})
	}
}

func TestGearPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GearPublicTestSuite))
}
