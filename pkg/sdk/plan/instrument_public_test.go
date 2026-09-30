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

package plan_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// InstrumentPublicTestSuite covers which instrument a chain's amps say it is
// for.
//
// Three callers had written this three ways and the three disagreed, so what
// matters here is the rule itself: every amplifier, a bass one anywhere wins,
// and a subcategory nobody recognises is a guitar amplifier rather than a
// reason to give up on the chain.
type InstrumentPublicTestSuite struct {
	suite.Suite
}

// blocks is a catalog just big enough to say what each model is.
type blocks map[catalog.ModelID]catalog.Block

func (b blocks) Block(
	id catalog.ModelID,
) (catalog.Block, bool) {
	blk, ok := b[id]

	return blk, ok
}

func (s *InstrumentPublicTestSuite) cat() blocks {
	return blocks{
		"amp-guitar": {Category: catalog.CategoryAmp, Subcategory: "Guitar"},
		"amp-bass":   {Category: catalog.CategoryAmp, Subcategory: "Bass"},
		"amp-preamp": {Category: catalog.CategoryAmp, Subcategory: "Preamp > Mic"},
		"amp-blank":  {Category: catalog.CategoryAmp},
		"drive":      {Category: catalog.CategoryDrive},
		"cab":        {Category: catalog.CategoryCab},
	}
}

func (s *InstrumentPublicTestSuite) chain(
	models ...catalog.ModelID,
) plan.Plan {
	out := plan.Plan{}
	for at, m := range models {
		out.Blocks = append(out.Blocks, plan.Block{Model: m, Pos: at})
	}

	return out
}

func (s *InstrumentPublicTestSuite) TestInstrumentFor() {
	tests := []struct {
		name  string
		chain []catalog.ModelID
		want  string
	}{
		{"a guitar amp", []catalog.ModelID{"drive", "amp-guitar", "cab"}, plan.Guitar},
		{"a bass amp", []catalog.ModelID{"drive", "amp-bass", "cab"}, plan.Bass},
		{
			"a bass amp after a guitar one still decides it",
			[]catalog.ModelID{"amp-guitar", "amp-bass"}, plan.Bass,
		},
		{
			"and before one",
			[]catalog.ModelID{"amp-bass", "amp-guitar"}, plan.Bass,
		},
		{
			"a preamp is a guitar amplifier, not a chain to give up on",
			[]catalog.ModelID{"drive", "amp-preamp"}, plan.Guitar,
		},
		{
			"so is one tagged nothing at all",
			[]catalog.ModelID{"amp-blank"}, plan.Guitar,
		},
		{
			"no amplifier claims no instrument",
			[]catalog.ModelID{"drive", "cab"}, plan.NoAmp,
		},
		{
			"a model the catalog does not carry is not an amplifier",
			[]catalog.ModelID{"drive", "nobody-ships-this"}, plan.NoAmp,
		},
		{"an empty chain", nil, plan.NoAmp},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want,
				plan.InstrumentFor(s.chain(tt.chain...), s.cat()))
		})
	}
}

// TestAmpAtIsTheFirstOne is the other question, and it is not the same one.
//
// What a chain is for is decided by every amplifier; what it is ordered around
// is the first, which is the pivot the corpus counts before and after.
func (s *InstrumentPublicTestSuite) TestAmpAtIsTheFirstOne() {
	tests := []struct {
		name  string
		chain []catalog.ModelID
		want  int
	}{
		{"the only one", []catalog.ModelID{"drive", "amp-bass", "cab"}, 1},
		{
			"the first, even when a later one decides the instrument",
			[]catalog.ModelID{"drive", "amp-guitar", "amp-bass"}, 1,
		},
		{"one nobody tagged still counts", []catalog.ModelID{"amp-blank"}, 0},
		{"none", []catalog.ModelID{"drive", "cab"}, -1},
		{"an empty chain", nil, -1},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, plan.AmpAt(s.chain(tt.chain...), s.cat()))
		})
	}
}

func TestInstrumentPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InstrumentPublicTestSuite))
}
