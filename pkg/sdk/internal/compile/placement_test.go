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
)

// PlacementTestSuite covers where a block the person did not write goes.
//
// Task 278, and the thing that made it hard to see. Position is not
// decoration: a cabinet behind an equaliser puts the speaker simulation after
// the equaliser that was meant to shape it, and it moves every block after the
// amplifier one place, which is what made three footswitches look bound to the
// wrong blocks.
type PlacementTestSuite struct {
	suite.Suite
}

// kinds names the category of each block, so a case reads as a chain.
func kinds(
	blocks []catalog.Block,
) []catalog.Category {
	out := make([]catalog.Category, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, b.Category)
	}

	return out
}

// chainOf builds blocks of the categories named.
func chainOf(
	cats ...catalog.Category,
) []catalog.Block {
	out := make([]catalog.Block, 0, len(cats))
	for _, c := range cats {
		out = append(out, catalog.Block{Category: c})
	}

	return out
}

// TestInsert covers which side of the amplifier a filled block lands on.
//
// One method and one table, so a case is a row rather than a file.
func (s *PlacementTestSuite) TestInsert() {
	for _, tt := range []struct {
		name string
		// chain is the chain before the block goes in.
		chain []catalog.Category
		// add is the kind of block being filled in, and beforeAmp what the
		// corpus says about which side of the amplifier it belongs.
		add       catalog.Category
		beforeAmp bool
		want      []catalog.Category
	}{
		{
			// Drive ahead of an amp overdrives its input; drive after it does
			// something else entirely.
			name:      "a block that belongs in front goes in front",
			chain:     []catalog.Category{catalog.CategoryAmp, catalog.CategoryCab},
			add:       catalog.CategoryComp,
			beforeAmp: true,
			want: []catalog.Category{
				catalog.CategoryComp, catalog.CategoryAmp, catalog.CategoryCab,
			},
		},
		{
			// Everything else that belongs after an amplifier is appended, so
			// it lands behind the cabinet rather than in front of it.
			name:  "a block that belongs behind goes last",
			chain: []catalog.Category{catalog.CategoryAmp, catalog.CategoryCab},
			add:   catalog.CategoryReverb,
			want: []catalog.Category{
				catalog.CategoryAmp, catalog.CategoryCab, catalog.CategoryReverb,
			},
		},
		{
			// **A cabinet is neither of those.** It goes after an amplifier,
			// so appending it looks right, and a chain that does not end at
			// the amplifier is where that falls over: the equaliser would
			// have been left working on the amplifier's raw output with the
			// speaker simulation behind it.
			name:  "a cabinet goes straight after the amplifier",
			chain: []catalog.Category{catalog.CategoryAmp, catalog.CategoryEQ},
			add:   catalog.CategoryCab,
			want: []catalog.Category{
				catalog.CategoryAmp, catalog.CategoryCab, catalog.CategoryEQ,
			},
		},
		{
			// Nowhere to put one, so it goes where a cabinet ending a chain
			// of effects goes anyway.
			name:  "a cabinet with no amplifier to sit behind",
			chain: []catalog.Category{catalog.CategoryDrive, catalog.CategoryEQ},
			add:   catalog.CategoryCab,
			want: []catalog.Category{
				catalog.CategoryDrive, catalog.CategoryEQ, catalog.CategoryCab,
			},
		},
	} {
		s.Run(tt.name, func() {
			blocks := chainOf(tt.chain...)
			said := make([]*wanted, len(blocks))

			got, kept := insert(
				blocks, said, catalog.Block{Category: tt.add}, tt.beforeAmp, false)

			s.Require().Equal(tt.want, kinds(got))

			// The parallel slice of what a rig wrote stays the same length as
			// the chain. It is paired index by index, and a slip writes one
			// block's settings onto its neighbour with no error at all.
			s.Require().Len(kept, len(got))
		})
	}
}

// TestAmplifierAt covers finding the amplifier a cabinet belongs to.
func (s *PlacementTestSuite) TestAmplifierAt() {
	for _, tt := range []struct {
		name  string
		chain []catalog.Category
		want  int
	}{
		{
			name:  "the amplifier in an ordinary chain",
			chain: []catalog.Category{catalog.CategoryDrive, catalog.CategoryAmp},
			want:  1,
		},
		{
			// The first, because a chain with two amplifiers is a chain whose
			// second is a parallel path and the cabinet belongs to the one
			// the signal reaches first.
			name:  "the first of two",
			chain: []catalog.Category{catalog.CategoryAmp, catalog.CategoryAmp},
			want:  0,
		},
		{
			// Which is what a chain of effects is, and is not an error.
			name:  "a chain with no amplifier at all",
			chain: []catalog.Category{catalog.CategoryDrive, catalog.CategoryEQ},
			want:  -1,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, amplifierAt(chainOf(tt.chain...)))
		})
	}
}

// TestAfterTheAmp covers the implied cabinet, which is the one an amplifier
// names when a rig names none.
func (s *PlacementTestSuite) TestAfterTheAmp() {
	for _, tt := range []struct {
		name  string
		chain []catalog.Category
		// wants is how many entries the parallel slice holds. Shorter than
		// the chain is what a caller that has added blocks of its own leaves,
		// and the gap is then appended rather than indexed.
		wants int
		want  []catalog.Category
	}{
		{
			name:  "ahead of everything the rig wrote after its amplifier",
			chain: []catalog.Category{catalog.CategoryAmp, catalog.CategoryEQ},
			wants: 2,
			want: []catalog.Category{
				catalog.CategoryAmp, catalog.CategoryCab, catalog.CategoryEQ,
			},
		},
		{
			name:  "a settings slice shorter than the chain",
			chain: []catalog.Category{catalog.CategoryDrive, catalog.CategoryAmp},
			wants: 0,
			want: []catalog.Category{
				catalog.CategoryDrive, catalog.CategoryAmp, catalog.CategoryCab,
			},
		},
	} {
		s.Run(tt.name, func() {
			got, kept := afterTheAmp(
				chainOf(tt.chain...),
				make([]*wanted, tt.wants),
				catalog.Block{Category: catalog.CategoryCab})

			s.Require().Equal(tt.want, kinds(got))
			s.Require().Len(kept, tt.wants+1)
		})
	}
}

func TestPlacementTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(PlacementTestSuite))
}
