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

	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// BlockAtPublicTestSuite covers addressing a block in a chain.
//
// The compiler and the editor had each written this, identically, under two
// names. It matters because both paths of a device that has two count their
// blocks from zero: a chain laid across them holds two blocks numbered 0, and
// a position on its own finds whichever came first.
type BlockAtPublicTestSuite struct {
	suite.Suite
}

func (s *BlockAtPublicTestSuite) chain() []plan.Block {
	return []plan.Block{
		{Model: "first-on-nothing", DSP: 0, Pos: 0},
		{Model: "second-on-nothing", DSP: 0, Pos: 1},
		{Model: "first-on-one", DSP: 1, Pos: 0},
	}
}

func (s *BlockAtPublicTestSuite) TestBlockAt() {
	tests := []struct {
		name  string
		path  int
		pos   int
		want  string
		found bool
	}{
		{
			name: "position nothing on the first processor",
			path: 0, pos: 0, want: "first-on-nothing", found: true,
		},
		{
			// The whole reason both are needed. Matching on position alone
			// would answer the first processor's block for this.
			name: "position nothing on the second, which is a different block",
			path: 1, pos: 0, want: "first-on-one", found: true,
		},
		{
			name: "a later position on the first",
			path: 0, pos: 1, want: "second-on-nothing", found: true,
		},
		{name: "a position nothing sits at", path: 0, pos: 7},
		{name: "a processor the chain does not use", path: 2, pos: 0},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, found := plan.BlockAt(s.chain(), tt.path, tt.pos)

			s.Require().Equal(tt.found, found)

			if !tt.found {
				s.Require().Empty(got.Model, "and nothing is handed back")

				return
			}

			s.Require().Equal(tt.want, string(got.Model))
		})
	}
}

func TestBlockAtPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BlockAtPublicTestSuite))
}
