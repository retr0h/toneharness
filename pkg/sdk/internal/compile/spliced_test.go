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
)

// SplicedTestSuite covers making room for a block the corpus added.
//
// The list it splices into runs parallel to the chain, so what goes in has to
// land at the same index the block did. A nil there used to be hardcoded, which
// dropped the bypass for every block inserted ahead of the amplifier: the
// report said bypassed and the plan said enabled, and every drive goes ahead of
// the amplifier.
type SplicedTestSuite struct {
	suite.Suite
}

// TestSpliced covers where the entry lands and that it is the one given.
func (s *SplicedTestSuite) TestSpliced() {
	off := &wanted{bypassed: true}

	for _, tt := range []struct {
		name string
		said []*wanted
		at   int
		with *wanted

		want int
		// which index must hold the entry that was passed in.
		holding int
	}{
		{
			// Ahead of the amplifier, which is where a drive goes.
			name:    "spliced into the middle",
			said:    []*wanted{nil, nil},
			at:      1,
			with:    off,
			want:    3,
			holding: 1,
		},
		{
			name:    "spliced at the front",
			said:    []*wanted{nil, nil},
			with:    off,
			want:    3,
			holding: 0,
		},
		{
			// Past the end, which is the guard rather than a case fill can
			// reach: it refuses an empty chain, and an amplifier it found has
			// an index inside the chain. Kept because the alternative is a
			// slice read off its end.
			name:    "spliced past the end",
			said:    []*wanted{nil},
			at:      9,
			with:    off,
			want:    2,
			holding: 1,
		},
		{
			// A block nobody is bypassing, which is what a word seating one
			// looks like. The nil still has to take a position, or every
			// index after it names the wrong block.
			name:    "spliced with nothing to say",
			said:    []*wanted{nil, nil},
			at:      1,
			want:    3,
			holding: -1,
		},
	} {
		s.Run(tt.name, func() {
			got := spliced(tt.said, tt.at, tt.with)

			s.Require().Len(got, tt.want)

			if tt.holding < 0 {
				for i, one := range got {
					s.Require().Nil(one, "index %d", i)
				}

				return
			}

			s.Require().Same(tt.with, got[tt.holding],
				"the entry has to land where the block did")

			for i, one := range got {
				if i != tt.holding {
					s.Require().Nil(one, "index %d", i)
				}
			}
		})
	}
}

func TestSplicedTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SplicedTestSuite))
}
