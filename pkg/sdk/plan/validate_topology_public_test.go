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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

type ValidateTopologyPublicTestSuite struct {
	suite.Suite
}

func (*ValidateTopologyPublicTestSuite) limits() plan.Limits {
	return plan.Limits{MaxBlocks: 6, Paths: 2, ChipCeiling: 95.0}
}

// TestValidateTopology checks where blocks sit rather than what they cost.
func (s *ValidateTopologyPublicTestSuite) TestValidateTopology() {
	crowded := make([]plan.Block, 7)
	for i := range crowded {
		crowded[i] = plan.Block{Model: "A", DSP: 0, Pos: i}
	}

	tests := []struct {
		name string
		spec plan.Plan
		ok   bool
		says string
	}{
		{
			name: "positions running 0..n on each chip",
			spec: plan.Plan{Blocks: []plan.Block{
				{Model: "A", DSP: 0, Pos: 0},
				{Model: "B", DSP: 0, Pos: 1},
				{Model: "C", DSP: 1, Pos: 0},
			}},
			ok: true,
		},
		{
			// A preset that makes no sound, which the device represents perfectly
			// well: every rule here is about where blocks sit, so a chain with
			// none has nothing to check. The MIDI remotes that drive Spotify or
			// Pro Tools from the footswitches are exactly this, and 102 presets in
			// the corpus hold no block at all.
			name: "a rig with nothing in it",
			spec: plan.Plan{},
			ok:   true,
		},
		{
			name: "more blocks than the device takes",
			spec: plan.Plan{Blocks: crowded},
			says: "7 blocks",
		},
		{
			name: "two blocks claiming one position",
			spec: plan.Plan{Blocks: []plan.Block{
				{Model: "A", DSP: 0, Pos: 0},
				{Model: "B", DSP: 0, Pos: 0},
			}},
			says: "",
		},
		{
			// A gap is accepted. A position is a slot in the grid the split and
			// the join sit in too, so a real preset leaves them: of 8,970
			// processor-and-path groups in the preset corpus 6,689 have a gap, and
			// requiring the contiguous run 0..n-1 refused every one of those
			// presets outright.
			name: "a gap in the run",
			spec: plan.Plan{Blocks: []plan.Block{
				{Model: "A", DSP: 0, Pos: 0},
				{Model: "B", DSP: 0, Pos: 2},
			}},
			ok: true,
		},
		{
			// What does hold, in all 8,969 groups that state a position: one slot,
			// one block. Two in the same slot is a chain the device cannot lay out.
			name: "two blocks in one slot",
			spec: plan.Plan{Blocks: []plan.Block{
				{Model: "A", DSP: 0, Pos: 1},
				{Model: "B", DSP: 0, Pos: 1},
			}},
			says: "two blocks at position 1",
		},
		{
			// A path that is not a number, which no preset holds and a plan
			// somebody edited could. Read as the first path rather than refused:
			// the position checks below still apply, and a path nobody can parse is
			// not a reason to reject a chain whose blocks are all in their own
			// slots.
			name: "a path that is not a number",
			spec: plan.Plan{Blocks: []plan.Block{
				{Model: "A", DSP: 0, Pos: 0, Attrs: map[string]json.RawMessage{
					"@path": json.RawMessage(`"first"`),
				}},
			}},
			ok: true,
		},
		{
			// The same slot on the other side of a split is a different slot.
			// 1,638 processors in the corpus hold two blocks at one position that
			// way, which is what says a position is per path rather than per
			// processor.
			name: "one slot each side of a split",
			spec: plan.Plan{Blocks: []plan.Block{
				{Model: "A", DSP: 0, Pos: 1},
				{Model: "B", DSP: 0, Pos: 1, Attrs: map[string]json.RawMessage{
					"@path": json.RawMessage("1"),
				}},
			}},
			ok: true,
		},
		{
			name: "a position below the first",
			spec: plan.Plan{Blocks: []plan.Block{{Model: "A", DSP: 0, Pos: -1}}},
			says: "",
		},
		{
			name: "a block on a chip the device does not have",
			spec: plan.Plan{Blocks: []plan.Block{{Model: "A", DSP: 5, Pos: 0}}},
			says: "",
		},
		{
			// Processors are numbered from nothing, so two of them are 0 and
			// 1 and the first one too far is 2. Refusing only what is well
			// past it leaves this exact block accepted by `>` where `>=`
			// refuses it, which is a chain put on a processor the device does
			// not have.
			name: "a block on the first chip past the last one",
			spec: plan.Plan{Blocks: []plan.Block{{Model: "A", DSP: 2, Pos: 0}}},
			says: "",
		},
		{
			name: "and the last chip it does have is fine",
			spec: plan.Plan{Blocks: []plan.Block{{Model: "A", DSP: 1, Pos: 0}}},
			ok:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := plan.ValidateTopology(tt.spec, s.limits())

			if tt.ok {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, plan.ErrBadTopology)

			if tt.says != "" {
				s.Require().Contains(err.Error(), tt.says)
			}
		})
	}
}

func TestValidateTopologyPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidateTopologyPublicTestSuite))
}
