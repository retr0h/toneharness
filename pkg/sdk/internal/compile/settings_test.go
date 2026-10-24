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

package compile

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
)

// PooledTestSuite covers the figure a block falls back on when the corpus
// barely holds it.
//
// Built tables rather than the shipped corpus, because the decisions here
// turn on states it does not reach: a sibling with a different control set, a
// sibling on a different range, a category with no evidence at all. What the
// shipped data does produce is covered where a preset is built.
type PooledTestSuite struct {
	suite.Suite
}

// mic is two cabinets carrying the same seven controls, the legacy cabinet of
// the same speaker carrying different ones, and a cabinet on another range.
//
// Drawn from the real shapes: a mic'd impulse response carries Mic, Position
// and Angle where the legacy block of that speaker carries EarlyReflections
// and spells its high cut `High Cut`, with a space.
func (s *PooledTestSuite) mic() *catalog.Catalog {
	full := func() map[string]catalog.Param {
		return map[string]catalog.Param{
			"Position": {
				Key: "Position", Type: catalog.ParamFloat,
				Min: 0, Max: 1, Default: catalog.Float(0),
			},
			"HighCut": {
				Key: "HighCut", Type: catalog.ParamFloat,
				Min: 500, Max: 20100, Default: catalog.Float(8000),
			},
		}
	}

	wider := full()
	wider["HighCut"] = catalog.Param{
		Key: "HighCut", Type: catalog.ParamFloat,
		Min: 20, Max: 40000, Default: catalog.Float(8000),
	}

	return &catalog.Catalog{
		Device: "HX Stomp",
		Blocks: map[catalog.ModelID]catalog.Block{
			// The one being built, which the corpus barely holds.
			"MicThin": {ID: "MicThin", Category: catalog.CategoryCab, Params: full()},
			// Its sibling, which players use.
			"MicUsed": {ID: "MicUsed", Category: catalog.CategoryCab, Params: full()},
			// The legacy block of the same speaker: different controls, so
			// its figures do not pool with a mic'd one.
			"Legacy": {
				ID: "Legacy", Category: catalog.CategoryCab,
				Params: map[string]catalog.Param{
					"High Cut":         {Key: "High Cut", Type: catalog.ParamFloat, Min: 500, Max: 20100},
					"EarlyReflections": {Key: "EarlyReflections", Type: catalog.ParamFloat, Max: 1},
				},
			},
			// The same controls on a different rail, which is not one median.
			"MicWider": {ID: "MicWider", Category: catalog.CategoryCab, Params: wider},
			// Another kind entirely.
			"DriveThing": {ID: "DriveThing", Category: catalog.CategoryDrive, Params: full()},
		},
	}
}

// bassOn is statistics where only the named models appear in a bass chain.
func (s *PooledTestSuite) bassOn(
	models map[catalog.ModelID]int,
	params map[catalog.ModelID]corpus.ParamStats,
) *corpus.Stats {
	out := &corpus.Stats{
		Models:  map[catalog.ModelID]corpus.ModelStats{},
		Grammar: map[string]corpus.Grammar{"bass": {Chains: 172, Models: models}},
	}

	for id, p := range params {
		out.Models[id] = corpus.ModelStats{
			Uses: p.N, Params: map[string]corpus.ParamStats{"HighCut": p, "Position": p},
		}
	}

	return out
}

// TestAcross covers pooling one control over a block's own kind.
//
// One method and one table, so a case is a row rather than a file.
func (s *PooledTestSuite) TestAcross() {
	for _, tt := range []struct {
		name       string
		models     map[catalog.ModelID]int
		params     map[catalog.ModelID]corpus.ParamStats
		instrument string
		key        string
		want       float64
		ok         bool
	}{
		{
			// The sibling players actually use, over enough of their
			// presets to be worth following.
			name:       "a sibling carrying the same controls answers",
			models:     map[catalog.ModelID]int{"MicUsed": 30},
			params:     map[catalog.ModelID]corpus.ParamStats{"MicUsed": {N: 65, Median: 4000}},
			instrument: "bass",
			key:        "HighCut",
			want:       4000,
			ok:         true,
		},
		{
			// The whole of what went wrong before this test existed: the
			// legacy cabinet of the same speaker sits at 20,000 and pooling
			// it in made a mic'd cabinet brighter than its own default.
			name:       "the legacy block of the same speaker does not pool",
			models:     map[catalog.ModelID]int{"Legacy": 200},
			params:     map[catalog.ModelID]corpus.ParamStats{"Legacy": {N: 200, Median: 20000}},
			instrument: "bass",
			key:        "HighCut",
			ok:         false,
		},
		{
			// A median is a position on a rail and two rails of different
			// lengths do not share one.
			name:       "a sibling on another range does not pool",
			models:     map[catalog.ModelID]int{"MicWider": 90},
			params:     map[catalog.ModelID]corpus.ParamStats{"MicWider": {N: 90, Median: 30000}},
			instrument: "bass",
			key:        "HighCut",
			ok:         false,
		},
		{
			// Guitar chains outnumber bass twenty to one, so pooling over
			// everything pools over guitar.
			name:       "a sibling no chain for this instrument holds",
			models:     map[catalog.ModelID]int{},
			params:     map[catalog.ModelID]corpus.ParamStats{"MicUsed": {N: 65, Median: 4000}},
			instrument: "bass",
			key:        "HighCut",
			ok:         false,
		},
		{
			name:       "too few presets behind the pool",
			models:     map[catalog.ModelID]int{"MicUsed": 1},
			params:     map[catalog.ModelID]corpus.ParamStats{"MicUsed": {N: 2, Median: 4000}},
			instrument: "bass",
			key:        "HighCut",
			ok:         false,
		},
		{
			name:       "an instrument nobody has measured",
			models:     map[catalog.ModelID]int{"MicUsed": 30},
			params:     map[catalog.ModelID]corpus.ParamStats{"MicUsed": {N: 65, Median: 4000}},
			instrument: "sitar",
			key:        "HighCut",
			ok:         false,
		},
		{
			name:       "no instrument named at all",
			models:     map[catalog.ModelID]int{"MicUsed": 30},
			params:     map[catalog.ModelID]corpus.ParamStats{"MicUsed": {N: 65, Median: 4000}},
			instrument: "",
			key:        "HighCut",
			ok:         false,
		},
		{
			name:       "a control the block does not carry",
			models:     map[catalog.ModelID]int{"MicUsed": 30},
			params:     map[catalog.ModelID]corpus.ParamStats{"MicUsed": {N: 65, Median: 4000}},
			instrument: "bass",
			key:        "NoSuchControl",
			ok:         false,
		},
	} {
		s.Run(tt.name, func() {
			cat := s.mic()
			block := cat.Blocks["MicThin"]

			got, ok := across(block, tt.key, cat, s.bassOn(tt.models, tt.params), tt.instrument)

			if !tt.ok {
				s.Require().False(ok)

				return
			}

			s.Require().True(ok)
			s.Require().InDelta(tt.want, got.Median, 1e-9)
		})
	}
}

// TestAcrossWeightsByPresets is why the figure is a weighted median rather
// than a mean of medians.
//
// A model hundreds of people used says more about what players do than one two
// people used, and one preset with a control at a stop drags a mean where it
// moves a median by nothing.
func (s *PooledTestSuite) TestAcrossWeightsByPresets() {
	cat := s.mic()
	cat.Blocks["MicOther"] = cat.Blocks["MicUsed"]
	cat.Blocks["MicOther"] = catalog.Block{
		ID: "MicOther", Category: catalog.CategoryCab,
		Params: cat.Blocks["MicUsed"].Params,
	}

	stats := s.bassOn(
		map[catalog.ModelID]int{"MicUsed": 30, "MicOther": 2},
		map[catalog.ModelID]corpus.ParamStats{
			"MicUsed":  {N: 200, Median: 4000},
			"MicOther": {N: 6, Median: 20000},
		})

	got, ok := across(cat.Blocks["MicThin"], "HighCut", cat, stats, "bass")
	s.Require().True(ok)
	s.Require().InDelta(4000.0, got.Median, 1e-9,
		"the model 200 people used decides it, not the one six did")
	s.Require().Equal(206, got.N)
}

// TestSameControls covers whether two blocks are the same kind of thing.
//
// One method and one table, so a case is a row rather than a file.
func (s *PooledTestSuite) TestSameControls() {
	cat := s.mic()
	want := map[string]bool{}

	for name := range cat.Blocks["MicThin"].Params {
		want[name] = true
	}

	for _, tt := range []struct {
		name  string
		block catalog.ModelID
		same  bool
	}{
		{
			name:  "the same controls",
			block: "MicUsed",
			same:  true,
		},
		{
			// Different names for the same job, which is exactly the legacy
			// cabinet's `High Cut` against a mic'd one's `HighCut`.
			name:  "different controls",
			block: "Legacy",
			same:  false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(
				tt.same, sameControls(cat.Blocks[tt.block], want))
		})
	}
}

// TestSameControlsCountsBothWays is the half a length test alone would miss.
//
// A block with every one of these and one more is a different block: an extra
// knob changes what the others do, so its figures are not these figures.
func (s *PooledTestSuite) TestSameControlsCountsBothWays() {
	cat := s.mic()
	want := map[string]bool{}

	for name := range cat.Blocks["MicThin"].Params {
		want[name] = true
	}

	extra := cat.Blocks["MicUsed"].Params
	with := map[string]catalog.Param{"Extra": {Key: "Extra"}}

	for k, v := range extra {
		with[k] = v
	}

	s.Require().False(
		sameControls(catalog.Block{Params: with}, want),
		"one control more is a different block")

	fewer := map[string]catalog.Param{"Position": extra["Position"]}
	s.Require().False(
		sameControls(catalog.Block{Params: fewer}, want),
		"one control fewer is a different block")
}

func TestPooledTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PooledTestSuite))
}
