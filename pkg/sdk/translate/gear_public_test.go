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

// TestGroupedBy covers which of the two things Line 6 group blocks by, and
// what each grouping then admits.
//
// One method and one table, so a case is a row rather than a file.
func (s *GearPublicTestSuite) TestGroupedBy() {
	for _, tt := range []struct {
		name       string
		blocks     map[catalog.ModelID]catalog.Block
		ids        []string
		instrument string
		// grouped is whether these blocks are grouped by instrument at all.
		grouped bool
		// admits is what the test says of each id, where there is one.
		admits map[string]bool
	}{
		{
			// Amplifiers, which Line 6 group "Guitar" and "Bass".
			name: "a subcategory that names an instrument",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Guitar"}, "b": {Subcategory: "Bass"},
			},
			ids:        []string{"a", "b"},
			instrument: "bass",
			grouped:    true,
			admits:     map[string]bool{"a": false, "b": true},
		},
		{
			// The strict reading. 23 amplifiers carry no subcategory, some of
			// them bass models and some not, so an unmarked one is a block
			// nobody can place and it loses rather than wins.
			name: "a block the subcategory does not place",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Bass"}, "b": {},
			},
			ids:        []string{"a", "b"},
			instrument: "bass",
			grouped:    true,
			admits:     map[string]bool{"a": true, "b": false},
		},
		{
			// Cabinets. Their subcategory counts microphones and says nothing
			// about the instrument, so the mark is what places them. Unmarked
			// means the other instrument here rather than unplaceable: every
			// cabinet is for one or the other.
			name: "a mark where the subcategory names something else",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Single, Dual", Bass: true},
				"b": {Subcategory: "Single, Dual"},
			},
			ids:        []string{"a", "b"},
			instrument: "bass",
			grouped:    true,
			admits:     map[string]bool{"a": true, "b": false},
		},
		{
			// And the same pool the other way up, which is the case that would
			// have emptied it: nothing is marked for guitar, so a guitar build
			// takes what is not marked for bass.
			name: "the marked pool asked for the other instrument",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Single, Dual", Bass: true},
				"b": {Subcategory: "Single, Dual"},
			},
			ids:        []string{"a", "b"},
			instrument: "guitar",
			grouped:    true,
			admits:     map[string]bool{"a": false, "b": true},
		},
		{
			// Neither said, which is every other category: a reverb is not a
			// guitar reverb, and nothing should be filtered out of one.
			name: "blocks grouped by neither",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Single, Dual"},
			},
			ids:        []string{"a"},
			instrument: "bass",
		},
		{
			// Measured and not in the catalog, which is the two drifting
			// apart. Kept, so the drift shows up as a block in the chain
			// rather than as a chain that is merely shorter.
			name: "a shortlist the catalog does not hold",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Single, Dual", Bass: true},
			},
			ids:        []string{"a", "nowhere"},
			instrument: "bass",
			grouped:    true,
			admits:     map[string]bool{"a": true, "nowhere": true},
		},
		{
			name:       "no catalog at all",
			ids:        []string{"a"},
			instrument: "bass",
		},
		{
			// An unstated instrument is nothing to filter against.
			name: "no instrument stated",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Subcategory: "Bass"},
			},
			ids: []string{"a"},
		},
	} {
		s.Run(tt.name, func() {
			var held *catalog.Catalog
			if tt.blocks != nil {
				held = s.built(tt.blocks)
			}

			fits := translate.GroupedBy(held, s.ranked(tt.ids...), tt.instrument)

			if !tt.grouped {
				s.Require().Nil(fits)

				return
			}

			s.Require().NotNil(fits)

			for id, want := range tt.admits {
				s.Require().Equal(want, fits(id), id)
			}
		})
	}
}

// TestWanted covers narrowing a ranking to the models that are the best of
// their gear, which is the order catalog.Block.Preferred states.
//
// One method and one table, so a case is a row rather than a file.
func (s *GearPublicTestSuite) TestWanted() {
	for _, tt := range []struct {
		name   string
		blocks map[catalog.ModelID]catalog.Block
		ids    []string
		want   []string
	}{
		{
			// The reading this exists for. A bass rig aimed at punk took the
			// legacy cabinet, which carries nothing to tune, over the mic'd one
			// seven units further out that carries a twelve-microphone list.
			name: "a mic'd cabinet beats a legacy one",
			blocks: map[catalog.ModelID]catalog.Block{
				"legacy": {Family: "cab", BasedOn: "one cabinet"},
				"mic":    {Family: "cabmicirs", BasedOn: "one cabinet"},
			},
			ids:  []string{"legacy", "mic"},
			want: []string{"mic"},
		},
		{
			// An amplifier before a preamp, which is the same list's first
			// rule: the preamp is the same gear with the power stage gone.
			name: "an amplifier beats its preamp",
			blocks: map[catalog.ModelID]catalog.Block{
				"pre": {Family: "preamp", BasedOn: "one amplifier"},
				"amp": {Family: "amp", BasedOn: "one amplifier"},
			},
			ids:  []string{"pre", "amp"},
			want: []string{"amp"},
		},
		{
			// The pan variant sits behind the plain one: pan is a stereo
			// placement control and a bass rig has no use for it.
			name: "the plain mic'd cabinet beats the pan variant",
			blocks: map[catalog.ModelID]catalog.Block{
				"pan": {
					Family:  "cabmicirswithpan",
					BasedOn: "one cabinet",
					Params:  map[string]catalog.Param{"Mic": {}},
				},
				"plain": {
					Family:  "cabmicirs",
					BasedOn: "one cabinet",
					Params:  map[string]catalog.Param{"Mic": {}},
				},
			},
			ids:  []string{"pan", "plain"},
			want: []string{"plain"},
		},
		{
			// The microphone rule, which is about the pool rather than about
			// copies of one cabinet: these two are different gear, and the
			// legacy one has no mic'd counterpart at all, so no per-gear rule
			// would reach it. It is the case the bass rig aimed at punk hit.
			name: "a cabinet offering a microphone beats one that does not",
			blocks: map[catalog.ModelID]catalog.Block{
				"legacy": {Family: "cab", BasedOn: "a 1x12"},
				"mic": {
					Family:  "cabmicirs",
					BasedOn: "a 1x15",
					Params:  map[string]catalog.Param{"Mic": {}},
				},
			},
			ids:  []string{"legacy", "mic"},
			want: []string{"mic"},
		},
		{
			// And where none is offered, nothing is dropped for want of one.
			// An amplifier carries no Mic, so this is no rule at all for one.
			name: "no microphone offered is no rule",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Family: "amp", BasedOn: "one amp"},
				"b": {Family: "amp", BasedOn: "another amp"},
			},
			ids:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			// Nothing is excluded for being the only thing there is. A tier
			// rather than a sort means the best tier present wins, and a pool
			// of legacy cabinets is still a pool.
			name: "the only tier there is survives",
			blocks: map[catalog.ModelID]catalog.Block{
				"a": {Family: "cab", BasedOn: "one cabinet"},
				"b": {Family: "cab", BasedOn: "another cabinet"},
			},
			ids:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			// Measured and not in the catalog, kept for the same reason it is
			// kept everywhere else here.
			name: "a block the catalog does not hold",
			blocks: map[catalog.ModelID]catalog.Block{
				"mic": {Family: "cabmicirs", BasedOn: "one cabinet"},
			},
			ids:  []string{"mic", "nowhere"},
			want: []string{"mic", "nowhere"},
		},
		{
			name: "no catalog at all",
			ids:  []string{"a"},
			want: []string{"a"},
		},
	} {
		s.Run(tt.name, func() {
			var held *catalog.Catalog
			if tt.blocks != nil {
				held = s.built(tt.blocks)
			}

			kept := translate.Wanted(held, s.ranked(tt.ids...))

			got := make([]string, 0, len(kept))
			for _, m := range kept {
				got = append(got, m.ID)
			}

			s.Require().ElementsMatch(tt.want, got)
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
		// want is the target this produces.
		want measured.Figures
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
			want: measured.Figures{Low: 92, Mid: 7, High: 0.3, Centroid: 165},
		},
		{
			// Usable carries that check, so this is not reached through a build.
			// What it does is leave the baseline alone: no displacement is
			// known, so none is applied.
			name: "a genre with nobody to compare against",
			got: audio.Genre{
				Across: audio.Across{Tracks: 1, Centroid: audio.Spread{Mid: 144}},
			},
			want: baseline,
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
			want: measured.Figures{Low: 90, Mid: 9, High: 0.3, Centroid: 165},
		},
	} {
		s.Run(tt.name, func() {
			got := translate.DisplacedTo(tt.got, baseline)

			s.Require().InDelta(tt.want.Low, got.Low, 0.01)
			s.Require().InDelta(tt.want.Mid, got.Mid, 0.01)
			s.Require().InDelta(tt.want.High, got.High, 0.01)
			s.Require().InDelta(tt.want.Centroid, got.Centroid, 0.01)
		})
	}
}

// TestNamed covers the name a chosen block is written into a rig as.
//
// One method and one table, so a case is a row rather than a file.
func (s *GearPublicTestSuite) TestNamed() {
	for _, tt := range []struct {
		name  string
		cat   *catalog.Catalog
		match measured.Match
		want  string
	}{
		{
			name: "the gear the model emulates, never the model's own name",
			cat: s.built(map[catalog.ModelID]catalog.Block{
				"HD2_AmpCali400Ch1": {
					Name:    "Cali 400 Ch1",
					BasedOn: "MESA/Boogie Bass 400+ (channel 1)",
				},
			}),
			match: measured.Match{Block: measured.Block{
				ID: "HD2_AmpCali400Ch1", Name: "Cali 400 Ch1",
			}},
			want: "MESA/Boogie Bass 400+ (channel 1)",
		},
		{
			name: "a Line 6 original emulates nothing, so its own name stands",
			cat: s.built(map[catalog.ModelID]catalog.Block{
				"HD2_CompressorDeluxeComp": {
					Name:    "Deluxe Comp",
					BasedOn: "Line 6 Original",
				},
			}),
			match: measured.Match{Block: measured.Block{
				ID: "HD2_CompressorDeluxeComp", Name: "Deluxe Comp",
			}},
			want: "Deluxe Comp",
		},
		{
			name: "a model the catalog has no entry for keeps the library's name",
			cat:  s.built(map[catalog.ModelID]catalog.Block{}),
			match: measured.Match{Block: measured.Block{
				ID: "HD2_SweptButNotInThisCatalog", Name: "Swept Thing",
			}},
			want: "Swept Thing",
		},
		{
			name: "no catalog to ask",
			cat:  nil,
			match: measured.Match{Block: measured.Block{
				ID: "HD2_AmpCali400Ch1", Name: "Cali 400 Ch1",
			}},
			want: "Cali 400 Ch1",
		},
		{
			name: "a block the catalog holds under no name at all",
			cat: s.built(map[catalog.ModelID]catalog.Block{
				"HD2_Nameless": {},
			}),
			match: measured.Match{Block: measured.Block{
				ID: "HD2_Nameless", Name: "Nameless",
			}},
			want: "Nameless",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, translate.Named(tt.cat, tt.match))
		})
	}
}

func TestGearPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GearPublicTestSuite))
}
