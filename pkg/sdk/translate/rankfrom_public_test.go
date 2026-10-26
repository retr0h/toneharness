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

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// RankFromPublicTestSuite covers which library a block is ranked against.
//
// A fingerprint taken alone is not how a pedal behaves in front of an
// amplifier: measured both ways on one settled rig, every one of the eleven
// genres chose a different drive. So a pair reading is preferred where one
// exists, and the whole difficulty is knowing when one does. A gear name is
// not a model, and "Ampeg SVT" names four of them.
type RankFromPublicTestSuite struct {
	suite.Suite
}

// pair is a library of readings taken behind one named block.
func (s *RankFromPublicTestSuite) pair(
	after catalog.ModelID,
) measured.Library {
	return measured.Library{
		Device: "HX Stomp", Isolated: false, After: string(after),
		Baseline: measured.Figures{Level: -51.03},
	}
}

// TestRankFrom covers when a pair reading answers and when it does not.
//
// One method and one table, so a case is a row rather than a file.
func (s *RankFromPublicTestSuite) TestRankFrom() {
	// Two amplifiers whose names say one model each, and one name that says
	// four, which is what the shipped rigs actually carry.
	cat := &catalog.Catalog{Device: "HX Stomp", Blocks: map[catalog.ModelID]catalog.Block{
		"HD2_AmpSVBeastBrt": {
			ID: "HD2_AmpSVBeastBrt", Name: "Ampeg SVT Brt",
			Category: catalog.CategoryAmp, BasedOn: "Ampeg SVT (bright channel)",
		},
		"HD2_AmpSVBeastNrm": {
			ID: "HD2_AmpSVBeastNrm", Name: "Ampeg SVT Nrm",
			Category: catalog.CategoryAmp, BasedOn: "Ampeg SVT (normal channel)",
		},
		"HD2_AmpTucknGo": {
			ID: "HD2_AmpTucknGo", Name: "Tuck n Go",
			Category: catalog.CategoryAmp, BasedOn: "Ampeg B-15NF Portaflex",
		},
	}}

	isolated := measured.Library{
		Device: "HX Stomp", Isolated: true,
		Baseline: measured.Figures{Level: -51.03},
	}

	// A compressor ahead of the amplifier in every chain, so the walk has a
	// non-amplifier entry to step over. Every real chain has several and a
	// test holding only an amplifier never exercises the skip.
	amp := func(gear string) []rig.ChainEntry {
		return []rig.ChainEntry{
			{Role: rig.RoleComp, Gear: "Deluxe Comp"},
			{Role: rig.RoleAmp, Gear: gear},
		}
	}

	for _, tt := range []struct {
		name  string
		chain []rig.ChainEntry
		kind  catalog.Category
		// have is which models a pair library exists for.
		have []catalog.ModelID
		// inChain is whether a pair reading answered.
		inChain bool
	}{
		{
			// The case this exists for. "Ampeg SVT Brt" names one model and
			// that model has been measured behind, so the drive is ranked on
			// how it reads there.
			name:    "an amplifier one pair reading was taken behind",
			chain:   amp("Ampeg SVT Brt"),
			kind:    catalog.CategoryDrive,
			have:    []catalog.ModelID{"HD2_AmpSVBeastBrt"},
			inChain: true,
		},
		{
			// Nothing has been measured behind a Tuck n Go, so the isolated
			// ranking stands. This is what keeps the eight genres whose
			// chains hold another amplifier on the answer they had.
			name:  "an amplifier nothing was measured behind",
			chain: amp("Tuck n Go"),
			kind:  catalog.CategoryDrive,
			have:  []catalog.ModelID{"HD2_AmpSVBeastBrt"},
		},
		{
			// A name that reaches two models which both have readings. Which
			// of the two the chain meant is not knowable from the name, and
			// guessing is how a figure from one pair gets ranked against a
			// target from another.
			name:  "a name reaching two models that both have readings",
			chain: amp("Ampeg SVT"),
			kind:  catalog.CategoryDrive,
			have: []catalog.ModelID{
				"HD2_AmpSVBeastBrt", "HD2_AmpSVBeastNrm",
			},
		},
		{
			// The same ambiguous name where only one candidate was measured,
			// which is the shipped state: the choice is unambiguous and taken.
			name:    "a name reaching two models where one has readings",
			chain:   amp("Ampeg SVT"),
			kind:    catalog.CategoryDrive,
			have:    []catalog.ModelID{"HD2_AmpSVBeastBrt"},
			inChain: true,
		},
		{
			// An amplifier is not ranked behind itself.
			name:  "the amplifier being chosen",
			chain: amp("Ampeg SVT Brt"),
			kind:  catalog.CategoryAmp,
			have:  []catalog.ModelID{"HD2_AmpSVBeastBrt"},
		},
		{
			// A cabinet sits after the amplifier, so a reading taken in front
			// of one says nothing about it.
			name:  "a cabinet, which sits the other side",
			chain: amp("Ampeg SVT Brt"),
			kind:  catalog.CategoryCab,
			have:  []catalog.ModelID{"HD2_AmpSVBeastBrt"},
		},
		{
			name:  "a chain with no amplifier in it yet",
			chain: nil,
			kind:  catalog.CategoryDrive,
			have:  []catalog.ModelID{"HD2_AmpSVBeastBrt"},
		},
		{
			// A release, which ships the isolated library and no pair
			// readings: they are a directory rather than something embedded.
			name:  "nothing to look pair readings up with",
			chain: amp("Ampeg SVT Brt"),
			kind:  catalog.CategoryDrive,
		},
	} {
		s.Run(tt.name, func() {
			deps := translate.Deps{Catalog: cat, Measured: isolated}

			if len(tt.have) > 0 {
				deps.InFrontOf = func(
					after catalog.ModelID,
				) (measured.Library, bool) {
					for _, one := range tt.have {
						if one == after {
							return s.pair(one), true
						}
					}

					return measured.Library{}, false
				}
			}

			lib, inChain := translate.RankFrom(tt.chain, deps, tt.kind)

			s.Require().Equal(tt.inChain, inChain)

			if !tt.inChain {
				s.Require().True(lib.Isolated,
					"the isolated library is what answers")

				return
			}

			s.Require().False(lib.Isolated)
			s.Require().NotEmpty(lib.After,
				"a pair reading says which pair it describes")
		})
	}
}

// TestRankFromWithoutACatalog covers the lookup having nothing to resolve a
// gear name with.
//
// A gear name reaches a model through the catalog, so without one there is no
// way to know which amplifier the chain names and the isolated ranking is the
// only honest answer.
func (s *RankFromPublicTestSuite) TestRankFromWithoutACatalog() {
	deps := translate.Deps{
		Measured: measured.Library{Isolated: true},
		InFrontOf: func(catalog.ModelID) (measured.Library, bool) {
			return s.pair("HD2_AmpSVBeastBrt"), true
		},
	}

	lib, inChain := translate.RankFrom(
		[]rig.ChainEntry{{Role: rig.RoleAmp, Gear: "Ampeg SVT Brt"}},
		deps, catalog.CategoryDrive)

	s.Require().False(inChain)
	s.Require().True(lib.Isolated)
}

func TestRankFromPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RankFromPublicTestSuite))
}
