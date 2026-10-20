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

package compile_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

type FillPublicTestSuite struct {
	suite.Suite
	cat *catalog.Catalog
}

func (s *FillPublicTestSuite) SetupSuite() {
	f, err := os.Open(filepath.Join("testdata", "catalog.json"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	s.cat, err = catalog.Load(f)
	s.Require().NoError(err)
}

// grammar builds statistics saying how often each category appears for bass,
// and, where counts are given, how many bass chains held each model.
func (s *FillPublicTestSuite) grammar(
	cats map[catalog.Category]corpus.CategoryStats,
	models map[catalog.ModelID]corpus.ModelStats,
	counts map[catalog.ModelID]int,
) *corpus.Stats {
	return &corpus.Stats{
		Models: models,
		Grammar: map[string]corpus.Grammar{
			"bass": {Chains: 100, Categories: cats, Models: counts},
		},
	}
}

// TestResolveFill covers filling a chain out, and where the settings land
// when a block is filled in front of them.
//
// One method and one table, so a case is a row rather than a file.
func (s *FillPublicTestSuite) TestResolveFill() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// What the corpus adds to a chain nobody asked for.
			//
			// Every case is resolved three times: two conventions of equal
			// weight must fill in the same order every run, or the same rig
			// yields a different rig for no reason anybody chose.
			name: "resolve fill",
			then: func() {
				tests := []struct {
					name string
					// the amp the rig names, and anything else it names beside it.
					gear  string
					extra []string

					cats   map[catalog.Category]corpus.CategoryStats
					models map[catalog.ModelID]corpus.ModelStats
					// how many bass chains held each model; nil for statistics measured
					// before that was counted.
					counts map[catalog.ModelID]int
					// no statistics at all.
					none bool
					// statistics measuring no chains for this instrument.
					silent bool

					want      int
					wantIDs   []catalog.ModelID
					wantName  string
					wantShare float64
					// whether the added block arrives switched on.
					//
					// False by default, and that is the point rather than a
					// convenience: a row whose statistics say nothing about
					// engagement is a block to carry and not to commit to, and
					// every row here written before engagement was counted
					// reads that way. A block switched off changes no sound,
					// so that is the direction to be wrong in.
					wantOn bool
					// the category of the first block in the chain, and of the last.
					first catalog.Category
					last  catalog.Category
				}{
					{
						// 90 chains hold one and none of them was counted as
						// running it, which reads as a block to carry and not
						// to commit to. Statistics measured before engagement
						// was counted land here too, and bypassed is the safe
						// direction: a block switched off changes no sound.
						name: "a block nearly every chain has",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 90, Before: 90},
						},
						models:    map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
						want:      1,
						wantName:  "Minotaur",
						wantShare: 0.9,
						// Drive feeds the amp's input, so it belongs ahead of it.
						first: catalog.CategoryDrive,
					},
					{
						// The same block, and the chains that hold one run it.
						// A compressor reads 87.0% here on real bass chains, so
						// this is the convention case: added and switched on.
						name: "a block nearly every chain has and uses",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 90, Engaged: 80, Before: 90},
						},
						models:    map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
						want:      1,
						wantShare: 0.9,
						wantOn:    true,
						first:     catalog.CategoryDrive,
					},
					{
						// Held by nearly every chain and run by under half,
						// which is a guitar drive at 47.3% and a guitar delay
						// at 61.6%. Added, and switched off.
						name: "a block nearly every chain has and most leave off",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 90, Engaged: 40, Before: 90},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
						want:   1,
						first:  catalog.CategoryDrive,
					},
					{
						// After the amplifier rather than before it, because
						// the bypass travelled through a different branch of
						// the insert and only one of the two was carrying it.
						name: "a bypassed block that belongs after the amp",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryReverb: {Chains: 95, Engaged: 10, After: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_ReverbTest": {Uses: 40}},
						want:   1,
						last:   catalog.CategoryReverb,
					},
					{
						name: "a block that belongs after the amp",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryReverb: {Chains: 95, After: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_ReverbTest": {Uses: 40}},
						want:   1,
						last:   catalog.CategoryReverb,
					},
					{
						// A compressor in most chains is a convention worth following.
						// One in six-in-ten is a choice, and making it silently would be
						// this tool having opinions it cannot justify.
						name: "agreement too weak to act on",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 60, Before: 60},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
					},
					{
						name:  "a convention the rig already named a pedal for",
						extra: []string{"Klon"},
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 95, Before: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
					},
					{name: "no statistics at all", none: true},
					{name: "no chains measured for this instrument", silent: true},
					{
						name: "a category the catalog has no model for",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.Category("nonsense"): {Chains: 99, Before: 99},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 1}},
					},
					{
						// When nobody named a pedal, the one most people reach for is the
						// only defensible choice.
						name: "two models for the convention, one of them common",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 95, Before: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{
							"HD2_DistMinotaur": {Uses: 40},
							"HD2_StereoDrive":  {Uses: 900},
						},
						want:    1,
						wantIDs: []catalog.ModelID{"HD2_StereoDrive"},
					},
					{
						// A total across instruments is a guitar figure. The pedal bass
						// players reach for wins, however many guitar presets hold the
						// other one.
						name: "the model this instrument reaches for, not the most used overall",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 95, Before: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{
							"HD2_DistMinotaur": {Uses: 40},
							"HD2_StereoDrive":  {Uses: 900},
						},
						counts:  map[catalog.ModelID]int{"HD2_DistMinotaur": 30, "HD2_StereoDrive": 5},
						want:    1,
						wantIDs: []catalog.ModelID{"HD2_DistMinotaur"},
					},
					{
						name: "a model no chain for this instrument held",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 95, Before: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{
							"HD2_DistMinotaur": {Uses: 40},
							"HD2_StereoDrive":  {Uses: 900},
						},
						counts:  map[catalog.ModelID]int{"HD2_DistMinotaur": 3},
						want:    1,
						wantIDs: []catalog.ModelID{"HD2_DistMinotaur"},
					},
					{
						// Popular elsewhere is not a reason to add a block players of
						// this instrument never chose.
						name: "a convention whose models this instrument never held",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 95, Before: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_StereoDrive": {Uses: 900}},
						counts: map[catalog.ModelID]int{"HD2_ReverbTest": 9},
					},
					{
						name: "two categories used equally often",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive:  {Chains: 90, Before: 90},
							catalog.CategoryReverb: {Chains: 90, After: 90},
						},
						models: map[catalog.ModelID]corpus.ModelStats{
							"HD2_DistMinotaur": {Uses: 40},
							"HD2_ReverbTest":   {Uses: 40},
						},
						want: 2,
					},
					{
						name: "two models used equally often, which break on identifier",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 90, Before: 90},
						},
						models: map[catalog.ModelID]corpus.ModelStats{
							"HD2_StereoDrive":  {Uses: 40},
							"HD2_DistMinotaur": {Uses: 40},
						},
						want:    1,
						wantIDs: []catalog.ModelID{"HD2_DistMinotaur"},
					},
					{
						// The corpus measures whatever presets held; a catalog for one
						// device will not carry all of it, and a chain cannot use what
						// the device lacks.
						name: "a model the catalog never heard of",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 95, Before: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_GhostModel": {Uses: 900}},
					},
					{
						// It carries a slot index, not audio, so a generated chain
						// reaching for one would point at whatever happened to be loaded
						// there.
						name: "a block needing the owner's own IR",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryCab: {Chains: 95, After: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{
							"HD2_ImpulseResponse1024": {Uses: 900},
						},
					},
					{
						// Line 6 tags amps and cabinets Guitar or Bass. This amp names no
						// cabinet, so the chain has a cabinet-shaped hole — and a guitar
						// cabinet must not fill it.
						name: "gear for the other instrument",
						gear: "Cabless Bass Head",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryCab: {Chains: 99, After: 99},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_CabGuitarOnly": {Uses: 900}},
					},
					{
						// Validation refuses a chain budgeted on an inferred cost.
						// Choosing one here would produce a chain rejected a moment
						// later, blaming a block nobody asked for.
						name: "a block whose cost was guessed",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryDrive: {Chains: 95, Before: 95},
						},
						models: map[catalog.ModelID]corpus.ModelStats{"HD2_NoDefault": {Uses: 900}},
					},
					{
						// Only amps and cabinets carry an instrument tag; a pedal serves
						// either. The guitar cabinet is refused, the pedal is not.
						name: "an untagged block, beside gear for the other instrument",
						gear: "Cabless Bass Head",
						cats: map[catalog.Category]corpus.CategoryStats{
							catalog.CategoryCab:   {Chains: 99, After: 99},
							catalog.CategoryDrive: {Chains: 80, Before: 80},
						},
						models: map[catalog.ModelID]corpus.ModelStats{
							"HD2_CabGuitarOnly": {Uses: 900},
							"HD2_DistMinotaur":  {Uses: 40},
						},
						want:    1,
						wantIDs: []catalog.ModelID{"HD2_DistMinotaur"},
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						gear := tt.gear
						if gear == "" {
							gear = "Ampeg SVT"
						}

						var stats *corpus.Stats

						switch {
						case tt.none:
						case tt.silent:
							stats = &corpus.Stats{Grammar: map[string]corpus.Grammar{}}
						default:
							stats = s.grammar(tt.cats, tt.models, tt.counts)
						}

						var first []catalog.ModelID

						for range 3 {
							spec, added, _, _, err := compile.Resolve(
								"a-rig", bassRig(gear, "", tt.extra...), compile.Intent{}, s.cat, stats)

							s.Require().NoError(err)
							s.Require().Len(added, tt.want)

							got := make([]catalog.ModelID, 0, len(added))
							for _, a := range added {
								got = append(got, a.Block.ID)
							}

							if first == nil {
								first = got
							}

							s.Require().Equal(first, got, "the same rig must fill the same way")

							if tt.wantIDs != nil {
								s.Require().Equal(tt.wantIDs, got)
							}

							if tt.wantName != "" {
								s.Require().Equal(tt.wantName, added[0].Block.Name)
							}

							if tt.wantShare != 0 {
								s.Require().InDelta(tt.wantShare, added[0].Share, 1e-9)
							}

							// Both the report and the plan, because they were
							// able to disagree: `spliced` put a nil where the
							// bypass should have gone, so a block inserted
							// ahead of the amplifier was reported bypassed and
							// written enabled. Every drive goes ahead of the
							// amplifier.
							if len(added) > 0 {
								s.Require().Equal(!tt.wantOn, added[0].Bypassed,
									"the report says whether it was bypassed")

								at := s.indexOfModel(spec, added[0].Block.ID)
								s.Require().GreaterOrEqual(at, 0)
								s.Require().Equal(tt.wantOn, spec.Blocks[at].Enabled,
									"and the plan agrees with the report")
							}

							if tt.first != "" {
								s.Require().Equal(tt.first, s.categoryAt(spec, 0))
							}

							if tt.last != "" {
								s.Require().Equal(
									tt.last, s.categoryAt(spec, len(spec.Blocks)-1))
							}
						}
					})
				}
			},
		},
		{
			// A bug that shipped.
			//
			// A rig's `settings` were held in a slice built index by index
			// against the chain the person typed, and `saidKnobs` paired
			// `said[i]` with the resolved block at the same index. But a
			// block the corpus fills in ahead of the amplifier is spliced
			// into the middle, shifting the amplifier and everything after it
			// one to the right, and nothing moved the settings with them. So
			// every setting after an insertion point was applied to its
			// neighbour.
			//
			// It did not reliably fail, which is why it survived. `level`
			// maps to several parameter names that different categories
			// share, so a value meant for an amplifier's Master could be
			// written to a compressor's Level with no error at all. That is a
			// preset which measures fine and is not what was asked for.
			//
			// The shipped rig that shows the shift is Bootsy Collins': it
			// names a filter then an amplifier, the corpus fills a compressor
			// in front of the amplifier because almost every chain has one,
			// and the amplifier arrives at index 2 while the settings for
			// index 1 still describe it. That rig escapes only because its
			// amplifier entry carries no settings.
			name: "settings follow their block when one is filled in front",
			then: func() {
				// A compressor in almost every chain, and ahead of the amplifier, which is
				// what the corpus actually says and what does the splicing.
				stats := s.grammar(
					map[catalog.Category]corpus.CategoryStats{
						catalog.CategoryDrive: {Chains: 95, Before: 95},
					},
					map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
					map[catalog.ModelID]int{"HD2_DistMinotaur": 40},
				)

				// The amplifier is the only block the rig names, and it names a drive for it.
				//
				// Drive rather than level, because this suite's fixture catalog is minimal:
				// its amplifier carries a Drive and no Level, ChVol or Master, and the block
				// filled in front of it carries a Gain. So drive is a word only the
				// amplifier can answer, which is what makes the assertion below mean
				// something rather than passing on a coincidence of names.
				spec := bassRig("Ampeg SVT", "")
				const want = 0.11

				drive := want
				spec.Chain[0].Settings = &rig.Settings{Drive: &drive}

				built, added, _, _, err := compile.Resolve("a-rig", spec, compile.Intent{}, s.cat, stats)
				s.Require().NoError(err)
				s.Require().Len(added, 1, "the corpus fills one block in")

				// It went in ahead of the amplifier, which is the condition for the bug.
				filled, ok := s.cat.Block(built.Blocks[0].Model)
				s.Require().True(ok)
				s.Require().Equal(catalog.CategoryDrive, filled.Category,
					"the filled block sits first, so the amplifier has shifted right")

				amp, err2 := s.cat.Block(built.Blocks[1].Model)
				s.Require().True(err2)
				s.Require().Equal(catalog.CategoryAmp, amp.Category)

				// And the level is on the amplifier, not on the block that displaced it.
				s.Require().True(holds(built.Blocks[1].Params, want),
					"the level the rig wrote for its amplifier has to reach the amplifier")
				s.Require().False(holds(built.Blocks[0].Params, want),
					"and must not reach the block the corpus filled in front of it")
			},
		},
		{
			// An empty chain is a statement rather than an omission, and the
			// corpus has nothing to say about it. 102 presets in the corpus hold
			// no block: the MIDI remotes that drive Spotify or Pro Tools from the
			// footswitches, and the blank templates people build from.
			//
			// Filling one turned a Spotify remote into a reverb, a delay and an
			// overdrive the first time an empty chain was allowed through.
			name: "an empty chain is left empty",
			then: func() {
				stats := s.grammar(
					map[catalog.Category]corpus.CategoryStats{
						catalog.CategoryDrive: {Chains: 95, Before: 95},
					},
					map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
					map[catalog.ModelID]int{"HD2_DistMinotaur": 40},
				)

				built, added, _, _, err := compile.Resolve(
					"x",
					rig.Spec{Instrument: rig.InstrumentBass, Chain: []rig.ChainEntry{}},
					compile.Intent{},
					s.cat,
					stats,
				)

				s.Require().NoError(err)
				s.Require().Empty(built.Blocks, "nothing is added to a chain that says none")
				s.Require().Empty(added)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// categoryAt returns the category of the block at a position.
func (s *FillPublicTestSuite) categoryAt(
	spec plan.Plan,
	i int,
) catalog.Category {
	b, ok := s.cat.Block(spec.Blocks[i].Model)
	s.Require().True(ok)

	return b.Category
}

// indexOfModel finds where a model landed in the chain.
//
// By model rather than by position, because the bug this guards against was a
// block reported one way and written another, and the position is what moved.
func (s *FillPublicTestSuite) indexOfModel(
	spec plan.Plan,
	id catalog.ModelID,
) int {
	for i, b := range spec.Blocks {
		if b.Model == id {
			return i
		}
	}

	return -1
}

// TestSuits covers which blocks are eligible for an instrument, which Line 6
// say two different ways.
//
// One method and one table, so a case is a row rather than a file.
func (s *FillPublicTestSuite) TestSuits() {
	for _, tt := range []struct {
		name       string
		block      catalog.Block
		instrument string
		want       bool
	}{
		{
			// A cabinet carries "Single, Dual", which counts microphones, so
			// the mark is the only thing that places it. The reading this
			// exists for: unread, this said yes to all 133 cabinets for every
			// instrument, and what saved it was a corpus too thin to notice.
			name:       "a cabinet marked for bass, asked for bass",
			block:      catalog.Block{Category: catalog.CategoryCab, Subcategory: "Single, Dual", Bass: true},
			instrument: "bass",
			want:       true,
		},
		{
			name:       "a cabinet marked for bass, asked for guitar",
			block:      catalog.Block{Category: catalog.CategoryCab, Subcategory: "Single, Dual", Bass: true},
			instrument: "guitar",
		},
		{
			// Unmarked means the other instrument rather than unplaceable:
			// every cabinet on the device is for one or the other.
			name:       "an unmarked cabinet, asked for guitar",
			block:      catalog.Block{Category: catalog.CategoryCab, Subcategory: "Single, Dual"},
			instrument: "guitar",
			want:       true,
		},
		{
			name:       "an unmarked cabinet, asked for bass",
			block:      catalog.Block{Category: catalog.CategoryCab, Subcategory: "Single, Dual"},
			instrument: "bass",
		},
		{
			// Amplifiers, which Line 6 do group by instrument.
			name:       "an amplifier whose subcategory agrees",
			block:      catalog.Block{Category: catalog.CategoryAmp, Subcategory: "Bass"},
			instrument: "bass",
			want:       true,
		},
		{
			name:       "an amplifier whose subcategory does not",
			block:      catalog.Block{Category: catalog.CategoryAmp, Subcategory: "Guitar"},
			instrument: "bass",
		},
		{
			// Everything else is available to either, which is the right
			// answer for a reverb.
			name:       "a block grouped by neither",
			block:      catalog.Block{Category: catalog.CategoryReverb},
			instrument: "bass",
			want:       true,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, compile.Suits(tt.block, tt.instrument))
		})
	}
}

// TestAsNamed covers the copy of a block a document naming it would get back.
//
// One method and one table, so a case is a row rather than a file.
func (s *FillPublicTestSuite) TestAsNamed() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	for _, tt := range []struct {
		name  string
		block catalog.ModelID
		want  catalog.ModelID
	}{
		{
			// The reading this exists for. The commonest 8x10 in this
			// instrument's corpus chains is the legacy model, Line 6 document
			// it as an unqualified 8x10" Ampeg SVT, and that name is also the
			// front of the mic'd 810AV's, which wins on Preferred. Eight
			// shipped rigs recorded a cabinet their own text does not resolve
			// to.
			name:  "a legacy cabinet whose name reaches the mic'd one",
			block: "HD2_Cab8x10SVBeast",
			want:  "HD2_CabMicIr_8x10SVTAV",
		},
		{
			// Already the copy its name resolves to, so nothing moves.
			name:  "a block its own name already reaches",
			block: "HD2_CabMicIr_8x10SVTAV",
			want:  "HD2_CabMicIr_8x10SVTAV",
		},
	} {
		s.Run(tt.name, func() {
			held, ok := cat.Block(tt.block)
			s.Require().True(ok, tt.block)

			got := compile.AsNamed(cat, held, held.Category, "bass")
			s.Require().Equal(tt.want, got.ID)
		})
	}
}

// TestAsNamedKeepsTheBlock covers the two ways a block's own name answers
// nothing, both of which keep the block rather than dropping it.
//
// One method and one table, so a case is a row rather than a file.
func (s *FillPublicTestSuite) TestAsNamedKeepsTheBlock() {
	for _, tt := range []struct {
		name  string
		block catalog.Block
		// asking is the category to resolve the name in, where that is not the
		// block's own. Empty means its own, which is what both callers pass.
		asking catalog.Category
	}{
		{
			// Nothing to resolve. A block with neither gear nor a name has no
			// text a document could carry, which is a name nothing can ask for
			// rather than a reason to drop the block.
			name:  "a block with no name at all",
			block: catalog.Block{ID: "only", Category: catalog.CategoryCab},
		},
		{
			// A name no model of the category answers. The fallback cannot
			// fire from either caller — the corpus fill and the amplifier's
			// own pairing both ask about a block's own category, and
			// `anyInstrument` ends in an unfiltered search, so a block in the
			// catalog always resolves to itself. It is here so that a caller
			// who asks a different question gets the block back rather than a
			// zero one, which would reach the device as a chain entry naming
			// nothing.
			name: "a name no model of that category answers",
			block: catalog.Block{
				ID:       "only",
				Name:     "Only",
				Category: catalog.CategoryCab,
				BasedOn:  "a cabinet nothing in this catalog emulates",
			},
			asking: catalog.CategoryAmp,
		},
	} {
		s.Run(tt.name, func() {
			cat := &catalog.Catalog{
				Device: "HX Stomp",
				Blocks: map[catalog.ModelID]catalog.Block{tt.block.ID: tt.block},
			}

			asking := tt.asking
			if asking == "" {
				asking = tt.block.Category
			}

			got := compile.AsNamed(cat, tt.block, asking, "bass")
			s.Require().Equal(catalog.ModelID("only"), got.ID)
		})
	}
}

func TestFillPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FillPublicTestSuite))
}

// holds reports a parameter set to a value, whatever the control is called.
//
// By value rather than by name, because the point is which block received it:
// `level` resolves to Master or ChVol on an amplifier and to Level on a
// compressor, so naming the control would assume the answer.
func holds(
	params plan.Params,
	want float64,
) bool {
	for _, got := range params {
		if at, ok := got.Float(); ok && math.Abs(at-want) < 0.0001 {
			return true
		}
	}

	return false
}
