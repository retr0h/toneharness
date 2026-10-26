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
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
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

// TestAsPlayed covers the refusal of blocks that change what is played.
//
// One method and one table, so a case is a row rather than a file. The second
// answer is what makes this worth a table: whether the refusal happened in
// front of everything that survived, which is the only refusal that changed
// which block got chosen.
func (s *GearPublicTestSuite) TestAsPlayed() {
	// Keyed by the model identifiers Line 6 use, because that is what the
	// named half of the refusal reads and a short name of our own would not
	// exercise it.
	held := s.built(map[catalog.ModelID]catalog.Block{
		"fuzz": {
			ID: "HD2_DistTriangleFuzz", Name: "Triangle Fuzz",
			Category: catalog.CategoryDrive,
		},
		"octave": {
			ID: "HD2_DM4OctaveFuzz", Name: "Octave Fuzz",
			Category: catalog.CategoryDrive, BasedOn: "Tycobrahe Octavia",
		},
		"synth": {
			ID: "HD2_FM4OctiSynth", Name: "Octi Synth",
			Category: catalog.CategoryPitch,
		},
		"boost": {
			ID: "HD2_DistKinkyBoost", Name: "Kinky Boost",
			Category: catalog.CategoryDrive,
		},
	})

	for _, tt := range []struct {
		name    string
		cat     *catalog.Catalog
		ranked  []measured.Match
		want    []string
		refused []string
	}{
		{
			name:   "nothing to refuse",
			cat:    held,
			ranked: s.ranked("fuzz", "boost"),
			want:   []string{"fuzz", "boost"},
		},
		{
			// The case this exists for. The octave fuzz read nearest, so its
			// refusal is what put the ordinary fuzz in the chain.
			name:    "the nearest changes the notes, and it is said so",
			cat:     held,
			ranked:  s.ranked("octave", "fuzz", "boost"),
			want:    []string{"fuzz", "boost"},
			refused: []string{"Tycobrahe Octavia"},
		},
		{
			// Refused behind a survivor, so it was never going to be chosen
			// and there is nothing to report.
			name:   "one further out is dropped quietly",
			cat:    held,
			ranked: s.ranked("fuzz", "octave", "boost"),
			want:   []string{"fuzz", "boost"},
		},
		{
			name:    "two ahead of the first survivor are both named",
			cat:     held,
			ranked:  s.ranked("synth", "octave", "boost"),
			want:    []string{"boost"},
			refused: []string{"Octi Synth", "Tycobrahe Octavia"},
		},
		{
			// Nothing left to rank, which no category a build chooses from
			// is in. It is still the contract.
			name:    "every one of them changes the notes",
			cat:     held,
			ranked:  s.ranked("synth", "octave"),
			want:    []string{},
			refused: []string{"Octi Synth", "Tycobrahe Octavia"},
		},
		{
			// The catalog and the measured library drifting apart. Kept, so
			// the drift shows up as a block rather than as an absence.
			name:   "a block the catalog does not hold",
			cat:    held,
			ranked: s.ranked("gone", "fuzz"),
			want:   []string{"gone", "fuzz"},
		},
		{
			// Nothing to ask, so nothing is refused. A caller with no catalog
			// gets its ranking back whole.
			name:   "no catalog to ask",
			ranked: s.ranked("octave", "fuzz"),
			want:   []string{"octave", "fuzz"},
		},
	} {
		s.Run(tt.name, func() {
			kept, refused := translate.AsPlayed(tt.cat, tt.ranked)

			ids := make([]string, 0, len(kept))
			for _, match := range kept {
				ids = append(ids, match.ID)
			}

			s.Require().Equal(tt.want, ids)
			s.Require().Equal(tt.refused, refused)
		})
	}
}

// TestAsHeld covers the refusal of blocks no measured chain holds.
//
// One method and one table, so a case is a row rather than a file. The second
// answer is what makes this worth a table, as in [TestAsPlayed]: whether the
// refusal happened in front of everything that survived, which is the only
// refusal that changed which block got chosen.
func (s *GearPublicTestSuite) TestAsHeld() {
	held := s.built(map[catalog.ModelID]catalog.Block{
		"rat": {
			ID: "HD2_DistVerminDist", Name: "Vermin Dist",
			Category: catalog.CategoryDrive, BasedOn: "Pro Co RAT",
		},
		"metalzone": {
			ID: "KillerZ", Name: "Killer Z",
			Category: catalog.CategoryDrive, BasedOn: "BOSS Metal Zone MT-2",
		},
		"bigmuff": {
			ID: "HD2_DM4FuzzPi", Name: "Fuzz Pi",
			Category: catalog.CategoryDrive,
			BasedOn:  "Electro-Harmonix Big Muff Pi",
		},
		"boost": {
			ID: "HD2_DistKinkyBoost", Name: "Kinky Boost",
			Category: catalog.CategoryDrive, BasedOn: "Xotic EP Booster",
		},
	})

	// What the shipped bass corpus says about those four: a Pro Co RAT is in
	// 14 of the 172 chains and an EP Booster in 7, and not one holds a Metal
	// Zone or any Big Muff variant.
	chains := &corpus.Stats{
		Grammar: map[string]corpus.Grammar{
			"bass": {
				Chains: 172,
				// Keyed the way the ranking names them, which is how
				// `Nearest` answers and what the lookup uses.
				Models: map[catalog.ModelID]int{"rat": 14, "boost": 7},
			},
		},
	}

	for _, tt := range []struct {
		name       string
		cat        *catalog.Catalog
		stats      *corpus.Stats
		instrument string
		ranked     []measured.Match
		want       []string
		refused    []string
	}{
		{
			name:       "nothing to refuse",
			cat:        held,
			stats:      chains,
			instrument: "bass",
			ranked:     s.ranked("rat", "boost"),
			want:       []string{"rat", "boost"},
		},
		{
			// The reading this exists for. A Metal Zone read nearest to rock
			// and not one of 172 measured bass chains holds one, so its
			// refusal is what puts a pedal somebody plays in the chain.
			name:       "the nearest is one nobody holds, and it is said so",
			cat:        held,
			stats:      chains,
			instrument: "bass",
			ranked:     s.ranked("metalzone", "rat", "boost"),
			want:       []string{"rat", "boost"},
			refused:    []string{"BOSS Metal Zone MT-2"},
		},
		{
			// Refused behind a survivor, so it was never going to be chosen
			// and there is nothing to report.
			name:       "one further out is dropped quietly",
			cat:        held,
			stats:      chains,
			instrument: "bass",
			ranked:     s.ranked("rat", "bigmuff", "boost"),
			want:       []string{"rat", "boost"},
		},
		{
			name:       "two ahead of the first survivor are both named",
			cat:        held,
			stats:      chains,
			instrument: "bass",
			ranked:     s.ranked("metalzone", "bigmuff", "boost"),
			want:       []string{"boost"},
			refused: []string{
				"BOSS Metal Zone MT-2", "Electro-Harmonix Big Muff Pi",
			},
		},
		{
			// Every modulation block this device has swept is in this state,
			// so it is reachable today rather than the contract. An empty
			// pool would turn a block somebody can hear into no block, so the
			// ranking it arrived with is the answer.
			name:       "nothing in the pool is held, so the ranking stands",
			cat:        held,
			stats:      chains,
			instrument: "bass",
			ranked:     s.ranked("metalzone", "bigmuff"),
			want:       []string{"metalzone", "bigmuff"},
		},
		{
			// The catalog and the measured library drifting apart. Kept, so
			// the drift shows up as a block rather than as an absence, which
			// is what [asPlayed] does with the same case.
			name:       "a block the catalog does not hold",
			cat:        held,
			stats:      chains,
			instrument: "bass",
			ranked:     s.ranked("gone", "rat"),
			want:       []string{"gone", "rat"},
		},
		{
			// An instrument nobody has measured chains for. There is nothing
			// to say, so nothing is said.
			name:       "an instrument the corpus has no chains for",
			cat:        held,
			stats:      chains,
			instrument: "sitar",
			ranked:     s.ranked("metalzone", "rat"),
			want:       []string{"metalzone", "rat"},
		},
		{
			// A corpus that names the instrument and has measured nothing,
			// which is a manifest rather than a measurement.
			name:       "a corpus with no chains counted",
			cat:        held,
			stats:      &corpus.Stats{Grammar: map[string]corpus.Grammar{"bass": {}}},
			instrument: "bass",
			ranked:     s.ranked("metalzone", "rat"),
			want:       []string{"metalzone", "rat"},
		},
		{
			name:       "no corpus to ask",
			cat:        held,
			instrument: "bass",
			ranked:     s.ranked("metalzone", "rat"),
			want:       []string{"metalzone", "rat"},
		},
		{
			name:       "no catalog to ask",
			stats:      chains,
			instrument: "bass",
			ranked:     s.ranked("metalzone", "rat"),
			want:       []string{"metalzone", "rat"},
		},
	} {
		s.Run(tt.name, func() {
			kept, refused := translate.AsHeld(
				tt.cat, tt.stats, tt.instrument, tt.ranked)

			ids := make([]string, 0, len(kept))
			for _, match := range kept {
				ids = append(ids, match.ID)
			}

			s.Require().Equal(tt.want, ids)
			s.Require().Equal(tt.refused, refused)
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
		// bare drops the baseline's own harmonics, for the row about a
		// library written before the figure existed.
		bare bool
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
				Elsewhere: map[audio.Figure]float64{
					audio.KeyLow: 0.95, audio.KeyMid: 0.05,
					audio.KeyHigh: 0, audio.KeyCentroid: 134,
				},
			},
			// Two points of energy lower and ten hertz brighter than the records
			// elsewhere, applied to the signal the blocks were measured with.
			//
			// And nothing on the high band, where the two sides sit 0.001 of a
			// share apart: a share resolves to two places because the third
			// moves with which windows the playing fell in, so that is not a
			// difference anything may act on. The floor is stated on the
			// difference since task 273, where it used to fall out of the
			// figures either side being rounded before the subtraction.
			want: measured.Figures{Low: 92, Mid: 7, High: 0.3, Centroid: 165},
		},
		{
			// Harmonics on the same scale as the band shares, because that is
			// what it is: a share of the energy, held zero to one by the
			// corpus and reported as a percentage by the library. Task 276.
			name: "harmonics is shifted like a band share",
			got: audio.Genre{
				Across: audio.Across{
					Tracks:    1,
					Centroid:  audio.Spread{Mid: 144},
					Harmonics: audio.Spread{Mid: 0.11},
				},
				Elsewhere: map[audio.Figure]float64{
					audio.KeyCentroid: 134, audio.KeyHarmonics: 0.17,
				},
			},
			// Six points of harmonics below the records elsewhere, applied to
			// the signal the blocks were measured with.
			want: measured.Figures{
				Low: 90, Mid: 9, High: 0.3, Centroid: 165,
				Harmonics: ptr(31.794),
			},
		},
		{
			// A baseline with no reading of harmonics has nothing to shift,
			// which is what a library written before the figure existed is.
			// No shipped reading is in that state: all 613 usable blocks in
			// the committed library carry it, and so does its baseline.
			name: "a baseline carrying no harmonics keeps none",
			bare: true,
			got: audio.Genre{
				Across: audio.Across{
					Tracks:    1,
					Centroid:  audio.Spread{Mid: 144},
					Harmonics: audio.Spread{Mid: 0.11},
				},
				Elsewhere: map[audio.Figure]float64{
					audio.KeyCentroid: 134, audio.KeyHarmonics: 0.17,
				},
			},
			want: measured.Figures{Low: 90, Mid: 9, High: 0.3, Centroid: 165},
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
			was := baseline
			if !tt.bare {
				held := 37.794
				was.Harmonics = &held
			}

			got := translate.DisplacedTo(tt.got, was)

			s.Require().InDelta(tt.want.Low, got.Low, 0.01)
			s.Require().InDelta(tt.want.Mid, got.Mid, 0.01)
			s.Require().InDelta(tt.want.High, got.High, 0.01)
			s.Require().InDelta(tt.want.Centroid, got.Centroid, 0.01)

			if tt.want.Harmonics == nil {
				if tt.bare {
					s.Require().Nil(got.Harmonics)
				}

				return
			}

			s.Require().NotNil(got.Harmonics)
			s.Require().InDelta(*tt.want.Harmonics, *got.Harmonics, 0.01)
		})
	}
}

// ptr is one figure as a pointer, which the three added after the first sweeps
// ran are: a reading taken before them has a gap rather than a zero.
func ptr(
	at float64,
) *float64 {
	return &at
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

// TestDisplacedFromCarriesHarmonics covers the inverse keeping the figure.
//
// Task 276 weighs harmonics, so a target read as a lean has to carry it. A
// pointer, because a reading taken before the figure existed has a gap rather
// than a zero, and a displacement from a baseline that has none is not a
// displacement: both sides end up absent, which the ranking skips.
func (s *GearPublicTestSuite) TestDisplacedFromCarriesHarmonics() {
	for _, tt := range []struct {
		name string
		at   *float64
		was  *float64
		want *float64
	}{
		{
			name: "both sides carry it",
			at:   ptr(31.46), was: ptr(37.79), want: ptr(-6.33),
		},
		{
			name: "the target does and the baseline does not",
			at:   ptr(31.46), want: ptr(31.46),
		},
		{
			name: "the baseline does and the target does not",
			was:  ptr(37.79),
		},
		{name: "neither does"},
	} {
		s.Run(tt.name, func() {
			got := translate.DisplacedFrom(
				measured.Figures{Centroid: 165, Harmonics: tt.at},
				measured.Figures{Centroid: 155, Harmonics: tt.was})

			s.Require().InDelta(10.0, got.Centroid, 1e-9)

			if tt.want == nil {
				s.Require().Nil(got.Harmonics)

				return
			}

			s.Require().NotNil(got.Harmonics)
			s.Require().InDelta(*tt.want, *got.Harmonics, 0.01)
		})
	}
}

// TestBeats covers whether being nearest meant anything.
//
// One method and one table, so a case is a row rather than a file. Task 267:
// a ranking always has a first entry and that is not the same as a choice.
func (s *GearPublicTestSuite) TestBeats() {
	at := func(apart ...float64) []measured.Match {
		out := make([]measured.Match, 0, len(apart))
		for i, d := range apart {
			out = append(out, measured.Match{
				Block:    measured.Block{ID: fmt.Sprint(i)},
				Distance: d,
			})
		}

		return out
	}

	// A bass, where the bar is the root of a point of midrange and a point of
	// harmonics: 1.41.
	by := measured.SpectralFor("bass")

	for _, tt := range []struct {
		name   string
		ranked []measured.Match
		want   bool
	}{
		{
			// punk's drive: the nearest at 0.55 against a median of 17.71.
			// Every amplifier, cabinet, drive, compressor and reverb across
			// the eleven genres clears the bar by between 2.1 and 23.9.
			name:   "a field the nearest is plainly out in front of",
			ranked: at(0.55, 3, 17.71, 30, 61.59),
			want:   true,
		},
		{
			// punk's equaliser: 24.49 against a median of 24.51, which is two
			// hundredths on a scale where a hundredth of a share of the
			// midrange is worth one. Ten of the eleven genres are here.
			name:   "and one where it is not",
			ranked: at(24.49, 24.50, 24.51, 24.52, 40.88),
		},
		{
			// The median rather than the runner-up, because two may differ
			// while the rest are alike and what decides whether being first
			// meant anything is the field.
			name:   "a runner-up far off does not make it a choice",
			ranked: at(0.1, 0.2, 0.3, 0.4, 90),
		},
		{
			// Nothing could have gone otherwise.
			name:   "a field of one",
			ranked: at(5),
			want:   true,
		},
		{
			name:   "a field of two the nearest leads",
			ranked: at(1, 30),
			want:   true,
		},
		{
			name:   "and a field of two that are alike",
			ranked: at(1, 1.5),
		},
		{
			// No ranking at all, which the caller refuses before it gets
			// here. It is still the contract.
			name: "nothing ranked",
			want: true,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, translate.Beats(tt.ranked, by))
		})
	}
}
