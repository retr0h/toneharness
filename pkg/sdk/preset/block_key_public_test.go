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

package preset_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// BlockKeyPublicTestSuite covers telling a processor and a chain block apart by
// the key they are stored under.
//
// Here rather than in the compiler, because both ask and the answer decides which
// blocks belong to the chain: a processor's `block3` is a chain entry, and a
// footswitch's or a snapshot's `block3` is what that switch or snapshot does about
// one. Answered differently in two places, a lift would state the same block twice
// or drop it.
type BlockKeyPublicTestSuite struct {
	suite.Suite
}

// TestIsProcessorKey covers IsProcessorKey, which says whether a tone entry is a
// processor and so owns the chain's blocks.
//
// One method and one table, so a case is a row rather than a file.
func (s *BlockKeyPublicTestSuite) TestIsProcessorKey() {
	for _, tt := range []struct {
		name string
		key  string
		want bool
	}{
		{name: "the first processor", key: "dsp0", want: true},
		{name: "the second, which a Stomp also has", key: "dsp1", want: true},
		{name: "a snapshot is not one", key: "snapshot0"},
		{name: "nor the footswitches", key: "footswitch"},
		{name: "nor the global settings", key: "global"},
		{name: "nor the impulse response table", key: "irUuidTable"},
		{name: "nothing at all is not one", key: ""},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, preset.IsProcessorKey(tt.key))
		})
	}
}

// TestIsBlockKey covers IsBlockKey, which says whether an entry inside a
// processor is a chain block rather than its routing.
func (s *BlockKeyPublicTestSuite) TestIsBlockKey() {
	for _, tt := range []struct {
		name string
		key  string
		want bool
	}{
		{name: "the first block", key: "block0", want: true},
		{name: "a two-figure one", key: "block15", want: true},
		{name: "the word on its own is not one", key: "block"},
		{name: "a block of nothing numbered is not one", key: "blockA"},
		{name: "an input is routing", key: "inputA"},
		{name: "so is the split", key: "split"},
		{name: "a cabinet a dual block points at is not a chain block", key: "cab0"},
		{name: "nothing at all is not one", key: ""},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, preset.IsBlockKey(tt.key))
		})
	}
}

func TestBlockKeyPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BlockKeyPublicTestSuite))
}
