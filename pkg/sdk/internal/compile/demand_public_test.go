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

type DemandPublicTestSuite struct {
	suite.Suite
	cat *catalog.Catalog
}

func (s *DemandPublicTestSuite) SetupSuite() {
	f, err := os.Open(filepath.Join("testdata", "catalog.json"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	s.cat, err = catalog.Load(f)
	s.Require().NoError(err)
}

// svt builds a bass rig naming no gear beyond the amplifier.
//
// The amplifier is the SVT's normal channel, which carries a MidFreq and no
// Mid. That is the shape this whole file is about: an amp that answers most
// questions and not this one.
func svt(
	pedals ...string,
) rig.Spec {
	return bassRig("Ampeg SVT", "", pedals...)
}

// asking is the ask beside that rig: what it should sound like, and how it is
// played.
//
// The attack is a plain string here because it is one on an Intent, which is
// what keeps this package from depending on the shape of the document the
// words were read out of.
func asking(
	terms []string,
	attack string,
) compile.Intent {
	words := make([]compile.Word, 0, len(terms))
	for _, t := range terms {
		words = append(words, compile.Word{Term: t})
	}

	return compile.Intent{Words: words, Attack: attack}
}

// statistics say nothing is near-universal, so fill adds nothing and whatever
// arrives was demanded rather than filled.
//
// Counts are given for every model a case can reach, because commonest skips
// a model no chain for this instrument held.
func (s *DemandPublicTestSuite) quiet(
	cats map[catalog.Category]corpus.CategoryStats,
	models ...catalog.ModelID,
) *corpus.Stats {
	byModel := map[catalog.ModelID]corpus.ModelStats{}
	counts := map[catalog.ModelID]int{}

	for i, m := range models {
		// Descending, so a case offering two models of a kind has a winner
		// that is not a tie broken on the identifier.
		byModel[m] = corpus.ModelStats{Uses: 100 - i}
		counts[m] = 100 - i
	}

	return &corpus.Stats{
		Models: byModel,
		Grammar: map[string]corpus.Grammar{
			"bass": {Chains: 100, Categories: cats, Models: counts},
		},
	}
}

// seated is the blocks a resolve demanded, in the order it demanded them.
func seated(
	added []compile.Added,
) []catalog.ModelID {
	out := make([]catalog.ModelID, 0, len(added))
	for _, a := range added {
		out = append(out, a.Block.ID)
	}

	return out
}

// TestResolveDemand covers blocks that are in a chain because the rig asked
// for them.
//
// The distinction from filling is the whole point. Filling answers what a
// chain of this kind usually has; this answers what this rig said it needs,
// and before it existed a word could only reach a control that happened to
// be there already.
func (s *DemandPublicTestSuite) TestResolveDemand() {
	tests := []struct {
		name string

		terms []string
		// derived is what a genre earned by measuring, as opposed to terms,
		// which somebody wrote. A derived word yields on an axis a written one
		// answers, and a word on its way out must not seat a block.
		derived []string
		attack  string
		pedals  []string
		models  []catalog.ModelID
		// where the corpus puts each kind, for the cases that check which
		// side of the amplifier a demanded block lands on.
		cats map[catalog.Category]corpus.CategoryStats
		// no statistics at all.
		none bool
		// statistics measuring no chains for this instrument.
		silent bool

		wantIDs []catalog.ModelID
		// how many blocks were demanded, for the rows that do not pin which.
		wantAdded int
		// moveOnly is a row whose subject is the move rather than the seating,
		// so what got added is not its question. Without it a nil wantIDs means
		// "nothing was demanded", which these rows do not claim either way.
		moveOnly bool
		// repeat resolves five times and requires the same blocks in the same
		// order. Two claims of equal weight must seat the same block every run,
		// or the same rig yields a different preset for no reason anybody chose.
		repeat bool
		// what the word did, for the rows whose subject is the move rather than
		// the seating: the term and the control it reached, whether the value
		// rose, and what the move says about itself.
		wantTerm    string
		wantParam   string
		wantRose    bool
		wantAlready string
		wantBecause string
		// landed is a move that needs no excuse, which is a different claim
		// from one whose `because` happens to be empty because nothing moved.
		landed bool
		// a fragment the reason for the first demanded block must carry.
		wantReason string
		// the category of the first block in the chain, and of the last.
		first catalog.Category
		last  catalog.Category
	}{
		{
			// The case this was written for. The SVT's normal channel has a
			// MidFreq and no Mid, so the word that earned its place by
			// measurement had nowhere to land.
			name:       "a word whose control no block in the chain carries",
			terms:      []string{"mid-forward"},
			models:     []catalog.ModelID{"HD2_EQTestParametric"},
			wantIDs:    []catalog.ModelID{"HD2_EQTestParametric"},
			wantReason: "the ask says mid-forward and nothing here had a MidGain",
		},
		{
			// Both sides of the axis ask the same question of the same band.
			name:    "the other word on that axis",
			terms:   []string{"scooped"},
			models:  []catalog.ModelID{"HD2_EQTestParametric"},
			wantIDs: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{
			// An equaliser is not a Mid just because it is an equaliser.
			// Seating one without the band would answer nothing and cost
			// DSP.
			name:   "an equaliser that has no such band",
			terms:  []string{"mid-forward"},
			models: []catalog.ModelID{"HD2_EQTestNoMid"},
		},
		{
			// The device has nothing that carries the band. Reported by the
			// word rather than papered over with a block that cannot help.
			name:   "a band this device has nowhere to put",
			terms:  []string{"mid-forward"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
		},
		{
			// A word naming a block asserts the block is there. It is not a
			// setting, so it asks for a filter and turns nothing up.
			name:       "a word that names a block rather than a setting",
			terms:      []string{"envelope-swept"},
			models:     []catalog.ModelID{"HD2_FilterTestMutant"},
			wantIDs:    []catalog.ModelID{"HD2_FilterTestMutant"},
			wantReason: "the ask says envelope-swept, which needs one",
		},
		{
			// How the instrument is played is the same kind of claim as a
			// word that names a block, and arrives from a different field.
			name:       "slap, which is not the sound without compression",
			attack:     "slap",
			models:     []catalog.ModelID{"HD2_CompTestDeluxe"},
			wantIDs:    []catalog.ModelID{"HD2_CompTestDeluxe"},
			wantReason: "the ask says slap, which needs one",
		},
		{
			// Every rig names an attack and most name nothing this reads.
			name:   "an attack that asks for nothing",
			attack: "pick",
			models: []catalog.ModelID{"HD2_CompTestDeluxe"},
		},
		{
			// Asking for what is already there would seat a second one.
			name:   "a filter the rig already named",
			terms:  []string{"envelope-swept"},
			pedals: []string{"Mu-Tron III"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
		},
		{
			// Two claims wanting the same kind of block get one block. The
			// word is read after the technique has seated it.
			name:    "two claims asking for the same kind of block",
			terms:   []string{"percussive"},
			attack:  "slap",
			models:  []catalog.ModelID{"HD2_CompTestDeluxe"},
			wantIDs: []catalog.ModelID{"HD2_CompTestDeluxe"},
		},
		{
			// A term nothing acts on asks for nothing. These are the axes
			// that describe the player rather than the signal.
			name:   "a term that moves nothing and names nothing",
			terms:  []string{"bridge-forward"},
			models: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{
			// Mix at zero and no reverb are the same signal, so the chain
			// already answers this and nothing is added.
			name:   "a word the chain's own shape answers",
			terms:  []string{"dry"},
			models: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{
			// A demanded block sits where the corpus puts that kind, the
			// same question fill asks of the blocks it adds. A filter mostly
			// seen ahead of the amplifier goes ahead of it.
			name:   "a demanded block the corpus puts before the amp",
			terms:  []string{"envelope-swept"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryFilter: {Chains: 40, Before: 40},
			},
			wantIDs: []catalog.ModelID{"HD2_FilterTestMutant"},
			first:   catalog.CategoryFilter,
		},
		{
			name:   "a demanded block the corpus puts after the amp",
			terms:  []string{"envelope-swept"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryFilter: {Chains: 40, After: 40},
			},
			wantIDs: []catalog.ModelID{"HD2_FilterTestMutant"},
			last:    catalog.CategoryFilter,
		},
		{
			// Two words from one axis cancel, and a cancelled word must not
			// drag a block in on its way out.
			name:   "both words from one axis",
			terms:  []string{"mid-forward", "scooped"},
			models: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{
			// The same axis answered twice, by two words of different standing.
			// The written one stands and seats its equaliser; the measured one
			// yields, and like a cancelled word must seat nothing on the way
			// out. One block, not two, and not none.
			name:       "a measured word yielding to a written one",
			terms:      []string{"mid-forward"},
			derived:    []string{"scooped"},
			models:     []catalog.ModelID{"HD2_EQTestParametric"},
			wantIDs:    []catalog.ModelID{"HD2_EQTestParametric"},
			wantReason: "the ask says mid-forward and nothing here had a MidGain",
		},
		{name: "no statistics at all", terms: []string{"mid-forward"}, none: true},
		{
			name:   "no chains measured for this instrument",
			terms:  []string{"mid-forward"},
			silent: true,
		},
		{
			// The order blocks arrive in.
			name:   "two claims of equal weight seat the same way every run",
			terms:  []string{"mid-forward", "envelope-swept"},
			attack: "slap",
			models: []catalog.ModelID{
				"HD2_EQTestParametric",
				"HD2_FilterTestMutant",
				"HD2_CompTestDeluxe",
			},
			wantAdded: 3,
			repeat:    true,
		},
		{
			// The end the rest of this exists for. Seating an equaliser is not
			// the point. The point is that a word which measured a record now
			// reaches a control, so the check is on what moved rather than on
			// what was added.
			name:      "the word lands on a control",
			terms:     []string{"mid-forward"},
			models:    []catalog.ModelID{"HD2_EQTestParametric"},
			moveOnly:  true,
			wantTerm:  "mid-forward",
			wantParam: "MidGain",
			wantRose:  true,
			landed:    true,
		},
		{
			// What a word naming a block reports. "satisfied" invites nobody to
			// check. Which filter got seated is the part somebody reading the
			// preset can disagree with, so it is what gets said.
			name:        "the block a word asked for is there, and is named",
			terms:       []string{"envelope-swept"},
			models:      []catalog.ModelID{"HD2_FilterTestMutant"},
			moveOnly:    true,
			wantAlready: "the Test Mutant Filter is what this asks for",
		},
		{
			name:        "this device has nothing of the kind",
			terms:       []string{"envelope-swept"},
			models:      []catalog.ModelID{"HD2_EQTestParametric"},
			moveOnly:    true,
			wantBecause: "this device has no filter",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var stats *corpus.Stats

			switch {
			case tt.none:
			case tt.silent:
				stats = &corpus.Stats{Grammar: map[string]corpus.Grammar{}}
			default:
				stats = s.quiet(tt.cats, tt.models...)
			}

			ask := asking(tt.terms, tt.attack)
			for _, t := range tt.derived {
				ask.Words = append(ask.Words, compile.Word{Term: t, Derived: true})
			}

			built, added, moved, _, err := compile.Resolve(
				"a-rig", svt(tt.pedals...), ask, s.cat, stats)

			s.Require().NoError(err)

			got := seated(added)

			if !tt.moveOnly {
				want := len(tt.wantIDs)
				if tt.wantAdded > 0 {
					want = tt.wantAdded
				}

				s.Require().Len(got, want)
			}

			if tt.wantIDs != nil {
				s.Require().Equal(tt.wantIDs, got)
			}

			if tt.repeat {
				for range 4 {
					_, again, _, _, err := compile.Resolve(
						"a-rig", svt(tt.pedals...), ask, s.cat, stats)
					s.Require().NoError(err)
					s.Require().Equal(got, seated(again),
						"the same ask must demand the same way")
				}
			}

			if tt.wantTerm != "" {
				s.Require().Len(moved, 1)
				s.Require().Equal(tt.wantTerm, moved[0].Term)
			}

			if tt.wantParam != "" {
				s.Require().Len(moved, 1)
				s.Require().Equal(tt.wantParam, moved[0].Param)
			}

			if tt.wantRose {
				s.Require().Len(moved, 1)
				s.Require().Greater(moved[0].To, moved[0].From)
			}

			if tt.landed {
				s.Require().Len(moved, 1)
				s.Require().Empty(moved[0].Because,
					"the word landed, so nothing excuses it")
			}

			if tt.wantAlready != "" || tt.wantBecause != "" {
				s.Require().Len(moved, 1)
				s.Require().Equal(tt.wantAlready, moved[0].Already)
				s.Require().Equal(tt.wantBecause, moved[0].Because)
			}

			if tt.wantReason != "" {
				s.Require().Equal(tt.wantReason, added[0].Reason)
			}

			if tt.first != "" {
				s.Require().Equal(tt.first, s.categoryAt(built, 0))
			}

			if tt.last != "" {
				s.Require().Equal(
					tt.last, s.categoryAt(built, len(built.Blocks)-1))
			}
		})
	}
}

// categoryAt returns the category of the block at a position.
func (s *DemandPublicTestSuite) categoryAt(
	built plan.Plan,
	i int,
) catalog.Category {
	b, ok := s.cat.Block(built.Blocks[i].Model)
	s.Require().True(ok)

	return b.Category
}

func TestDemandPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DemandPublicTestSuite))
}
