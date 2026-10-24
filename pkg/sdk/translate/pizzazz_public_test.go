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
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// PizzazzPublicTestSuite covers the switches a corpus offers where
// measurement asked for nothing.
//
// Against built grammars rather than the shipped one wherever the question is
// about the arithmetic, because the shipped corpus reaches a handful of rows
// and the decisions here have more: a category nothing holds, one every chain
// engages, and a chain with no room left are all states the 43 rigs do not
// produce.
type PizzazzPublicTestSuite struct {
	suite.Suite

	deps translate.Deps
}

func (s *PizzazzPublicTestSuite) SetupSuite() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	stats, err := corpus.BuiltIn()
	s.Require().NoError(err)

	s.deps = translate.Deps{Catalog: cat, Corpus: stats}
}

// bass is a chain of the shape every rig here is built on.
func (s *PizzazzPublicTestSuite) bass() rig.Spec {
	return rig.Spec{
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			{Role: rig.RoleCab, Gear: "8x10 Ampeg SVT-810AV"},
		},
	}
}

// TestCarriedButOff covers ranking a kind by how often it is left off.
//
// One method and one table, so a case is a row rather than a file.
func (s *PizzazzPublicTestSuite) TestCarriedButOff() {
	for _, tt := range []struct {
		name  string
		stats corpus.CategoryStats
		want  float64
	}{
		{
			// A wah: carried by a few and engaged by none, which is the
			// shape of a pedal rather than of a sound.
			name:  "carried and never engaged is the whole of it",
			stats: corpus.CategoryStats{Chains: 9, Engaged: 0},
			want:  1,
		},
		{
			// A compressor: in nearly every chain and running in nearly all
			// of them, so it is the sound and not an effect on it.
			name:  "carried and almost always engaged ranks near nothing",
			stats: corpus.CategoryStats{Chains: 146, Engaged: 127},
			want:  19.0 / 146.0,
		},
		{
			name:  "a drive sits in the middle",
			stats: corpus.CategoryStats{Chains: 107, Engaged: 52},
			want:  55.0 / 107.0,
		},
		{
			// Not a small number, no number: a share over no chains is
			// nothing to rank, and this is what drops the kind.
			name:  "nothing holds one, so there is no share",
			stats: corpus.CategoryStats{},
			want:  0,
		},
		{
			name:  "every chain engages it",
			stats: corpus.CategoryStats{Chains: 20, Engaged: 20},
			want:  0,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().InDelta(
				tt.want, translate.CarriedButOff(tt.stats), 1e-9)
		})
	}
}

// TestOffers covers which kinds are worth a switch, and in what order.
//
// One method and one table, so a case is a row rather than a file.
func (s *PizzazzPublicTestSuite) TestOffers() {
	grammar := func(rows map[catalog.Category]corpus.CategoryStats) corpus.Grammar {
		return corpus.Grammar{Chains: 100, Categories: rows}
	}

	for _, tt := range []struct {
		name    string
		grammar corpus.Grammar
		held    map[catalog.Category]bool
		id      string
		want    []catalog.Category
	}{
		{
			// Ordered by how often each is left off rather than by how often
			// it is held: ranking on presence puts the compressor first and
			// the wah last, which is backwards for this question.
			name: "the one players leave off most comes first",
			grammar: grammar(map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryComp:  {Chains: 146, Engaged: 127},
				catalog.CategoryDrive: {Chains: 107, Engaged: 52},
				catalog.CategoryMod:   {Chains: 54, Engaged: 15},
			}),
			id:   "",
			want: []catalog.Category{catalog.CategoryMod, catalog.CategoryDrive, catalog.CategoryComp},
		},
		{
			name: "a kind the chain already holds is not offered again",
			grammar: grammar(map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryMod:   {Chains: 54, Engaged: 15},
				catalog.CategoryDrive: {Chains: 107, Engaged: 52},
			}),
			held: map[catalog.Category]bool{catalog.CategoryMod: true},
			want: []catalog.Category{catalog.CategoryDrive},
		},
		{
			// An amplifier and a cabinet are the sound. Neither is stompable
			// and neither may be offered.
			name: "a kind nothing steps on is not offered",
			grammar: grammar(map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryAmp: {Chains: 90, Engaged: 1},
				catalog.CategoryCab: {Chains: 90, Engaged: 1},
				catalog.CategoryEQ:  {Chains: 78, Engaged: 65},
			}),
			want: []catalog.Category{catalog.CategoryEQ},
		},
		{
			name: "a kind every chain engages is dropped rather than ranked last",
			grammar: grammar(map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDelay: {Chains: 20, Engaged: 20},
				catalog.CategoryMod:   {Chains: 54, Engaged: 15},
			}),
			want: []catalog.Category{catalog.CategoryMod},
		},
		{
			name:    "a grammar holding nothing offers nothing",
			grammar: grammar(map[catalog.Category]corpus.CategoryStats{}),
			want:    nil,
		},
	} {
		s.Run(tt.name, func() {
			held := tt.held
			if held == nil {
				held = map[catalog.Category]bool{}
			}

			got := translate.Offers(tt.grammar, held, tt.id)

			kinds := make([]catalog.Category, 0, len(got))
			for _, at := range got {
				kinds = append(kinds, at.Kind())
			}

			if len(tt.want) == 0 {
				s.Require().Empty(kinds)

				return
			}

			s.Require().Equal(tt.want, kinds)
		})
	}
}

// TestOffersRotatesBySubject is what stops two rigs that measure alike being
// offered the same thing.
//
// The ranking still shows through, because this rotates rather than shuffles:
// a rig starting at the third candidate gets the third, fourth and fifth
// rather than three drawn from anywhere. And it is deterministic, so a rebuild
// is byte-identical and two people building one rig get one pedal.
func (s *PizzazzPublicTestSuite) TestOffersRotatesBySubject() {
	grammar := corpus.Grammar{Chains: 100, Categories: map[catalog.Category]corpus.CategoryStats{
		catalog.CategoryMod:    {Chains: 54, Engaged: 15},
		catalog.CategoryFilter: {Chains: 11, Engaged: 5},
		catalog.CategoryDrive:  {Chains: 107, Engaged: 52},
		catalog.CategoryDelay:  {Chains: 28, Engaged: 16},
		catalog.CategoryReverb: {Chains: 49, Engaged: 34},
	}}

	first := func(id string) []catalog.Category {
		got := translate.Offers(grammar, map[catalog.Category]bool{}, id)

		out := make([]catalog.Category, 0, len(got))
		for _, at := range got {
			out = append(out, at.Kind())
		}

		return out
	}

	a, b := first("mike-dirnt"), first("mark-hoppus")
	s.Require().Len(a, 5)
	s.Require().Len(b, 5)
	s.Require().NotEqual(a[0], b[0], "two subjects start at the same candidate")

	// The same answer every run, which is what makes a rebuild comparable.
	s.Require().Equal(a, first("mike-dirnt"))

	// And every candidate is still in the list: a rotation offers a different
	// order rather than a different set.
	s.Require().ElementsMatch(a, b)
}

// TestPopular covers which model of a kind a chain is offered.
//
// One method and one table, so a case is a row rather than a file.
func (s *PizzazzPublicTestSuite) TestPopular() {
	for _, tt := range []struct {
		name    string
		grammar corpus.Grammar
		kind    catalog.Category
		gear    string
		ok      bool
	}{
		{
			// The corpus rather than the measured ranking, which is the whole
			// point: measurement put five genres on one drive.
			name: "the model the most chains hold",
			grammar: corpus.Grammar{Models: map[catalog.ModelID]int{
				"HD2_DistHedgehogD9": 2,
				"HD2_Chorus":         30,
			}},
			kind: catalog.CategoryMod,
			gear: "Chorus",
			ok:   true,
		},
		{
			// Task 328, which holds bypassed as well: an octave is a
			// different note and a build may not choose one for somebody.
			name: "a model that changes the notes is refused",
			grammar: corpus.Grammar{Models: map[catalog.ModelID]int{
				"HD2_MM4RingModulator": 400,
			}},
			kind: catalog.CategoryMod,
			ok:   false,
		},
		{
			name: "a kind no measured chain holds",
			grammar: corpus.Grammar{Models: map[catalog.ModelID]int{
				"HD2_Chorus": 30,
			}},
			kind: catalog.CategoryDelay,
			ok:   false,
		},
		{
			name: "a model the catalog does not carry",
			grammar: corpus.Grammar{Models: map[catalog.ModelID]int{
				"HD2_NothingCallsItselfThis": 90,
			}},
			kind: catalog.CategoryMod,
			ok:   false,
		},
		{
			name:    "a grammar naming no models at all",
			grammar: corpus.Grammar{},
			kind:    catalog.CategoryMod,
			ok:      false,
		},
	} {
		s.Run(tt.name, func() {
			entry, gear, cost, ok := translate.Popular(
				tt.grammar, s.deps.Catalog, tt.kind)

			if !tt.ok {
				s.Require().False(ok)

				return
			}

			s.Require().True(ok)
			s.Require().Equal(tt.gear, gear)
			s.Require().Equal(rig.Role(tt.kind), entry.Role)
			s.Require().Positive(cost, "a block costs the chip something")

			// Off, which is what makes a weaker source honest here.
			s.Require().NotNil(entry.Enabled)
			s.Require().False(*entry.Enabled)
		})
	}
}

// TestSpareOn covers how much of the chip a chain has left.
//
// One method and one table, so a case is a row rather than a file.
func (s *PizzazzPublicTestSuite) TestSpareOn() {
	for _, tt := range []struct {
		name  string
		chain []rig.ChainEntry
		// most is the largest answer this may give, since the point of the
		// figure is that it never overstates what is left.
		most float64
	}{
		{
			// An empty chain has the whole ceiling, which is 95% of one chip
			// on every device the limits table knows.
			name: "an empty chain has spent nothing",
			most: 95.0,
		},
		{
			// An Ampeg SVT is 26.7% of the chip on its own, so a chain
			// holding one cannot have more than the rest.
			name:  "an amplifier has spent most of a chip",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "Ampeg SVT"}},
			most:  95.0 - 26.0,
		},
		{
			// Gear the catalog cannot place costs nothing here, which can
			// only overstate what is left. The budget check is the floor
			// under this rather than this arithmetic.
			name:  "gear nothing emulates is not priced",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "nothing calls itself this"}},
			most:  95.0,
		},
	} {
		s.Run(tt.name, func() {
			got := translate.SpareOn(
				rig.Spec{Instrument: rig.InstrumentBass, Chain: tt.chain}, s.deps)

			s.Require().LessOrEqual(got, tt.most)
		})
	}
}

// TestPizzazz covers the whole of an offer, against the shipped corpus.
//
// One method and one table, so a case is a row rather than a file.
func (s *PizzazzPublicTestSuite) TestPizzazz() {
	section := rig.Spec{
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			{Role: rig.RoleCab, Gear: "8x10 Ampeg SVT-810AV"},
		},
		Sections: &[]rig.Section{{Name: "Chorus"}},
	}

	for _, tt := range []struct {
		name string
		id   string
		spec rig.Spec
		deps func() translate.Deps
		want bool
	}{
		{
			// The case this exists for: a chain of amplifier and cabinet
			// reached the hardware with nothing on any switch.
			name: "a chain with room is offered something",
			id:   "mike-dirnt",
			spec: s.bass(),
			want: true,
		},
		{
			// A section says which roles play, and a chain of amplifier and
			// cabinet under `play: [drive]` is an author who meant something
			// they did not write. An offered drive turns their mistake into a
			// preset where the section plays a pedal they never chose.
			name: "a document naming its own sections is offered nothing",
			id:   "mike-dirnt",
			spec: section,
			want: false,
		},
		{
			name: "no corpus means nothing to offer",
			id:   "mike-dirnt",
			spec: s.bass(),
			deps: func() translate.Deps {
				return translate.Deps{Catalog: s.deps.Catalog}
			},
			want: false,
		},
		{
			name: "no catalog means nothing to place",
			id:   "mike-dirnt",
			spec: s.bass(),
			deps: func() translate.Deps {
				return translate.Deps{Corpus: s.deps.Corpus}
			},
			want: false,
		},
		{
			name: "a rig with no identifier is offered nothing",
			id:   "",
			spec: s.bass(),
			want: false,
		},
		{
			// Nothing here has measured a chain for one, so there is no
			// grammar to draw on and a guess would be somebody else's.
			name: "an instrument nobody has measured",
			id:   "mike-dirnt",
			spec: rig.Spec{
				Instrument: "sitar",
				Chain:      s.bass().Chain,
			},
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			deps := s.deps
			if tt.deps != nil {
				deps = tt.deps()
			}

			notes := translate.Notes{}
			got := translate.Pizzazz(tt.id, tt.spec, deps, &notes)

			if !tt.want {
				s.Require().Empty(got)

				return
			}

			s.Require().NotEmpty(got)

			for _, at := range got {
				// Bypassed, every one, which is what makes the corpus an
				// honest source for this and a dishonest one for a control.
				s.Require().NotNil(at.Entry.Enabled, at.Entry.Gear)
				s.Require().False(*at.Entry.Enabled, at.Entry.Gear)

				// And said as an offer rather than as a measurement.
				s.Require().Contains(
					at.Said, "which is a pedal rather than a sound")
			}

			s.Require().NotEmpty(notes)
		})
	}
}

// TestPizzazzSaysWhyASwitchStaysEmpty covers the chip running out.
//
// An offer that breaks the thing it was added to is worse than no offer: one
// chain reached 99.03% of a 95% ceiling and the build refused outright, which
// turned three extra switches into no preset at all. So a block that will not
// fit is declined and the reason is printed, because a switch somebody
// expected and did not get is a decision they may disagree with.
func (s *PizzazzPublicTestSuite) TestPizzazzSaysWhyASwitchStaysEmpty() {
	// Four Ampeg SVTs and a cabinet. One is 26.7% of the chip, so four is more
	// than the 95% ceiling on its own and nothing can be offered. A chain
	// nobody would build, which is the point: the arithmetic has to hold at
	// the ceiling, and the build refuses a chain this shape for its own
	// reasons rather than leaving it to this.
	full := rig.Spec{
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			{Role: rig.RoleAmp, Gear: "MESA/Boogie Bass 400+"},
			{Role: rig.RoleAmp, Gear: "Ampeg B-15NF Portaflex"},
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			{Role: rig.RoleCab, Gear: "8x10 Ampeg SVT-810AV"},
		},
	}

	notes := translate.Notes{}
	got := translate.Pizzazz("mike-dirnt", full, s.deps, &notes)

	s.Require().Empty(got, "nothing fits, so nothing is offered")

	said := ""
	for _, at := range notes {
		said += at.Said + "\n"
	}

	s.Require().Contains(said, "the switch stays empty")
	s.Require().Contains(said, "of the chip")
}

func TestPizzazzPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PizzazzPublicTestSuite))
}
