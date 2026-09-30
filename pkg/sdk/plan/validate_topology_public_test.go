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
			name: "a rig with nothing in it",
			spec: plan.Plan{},
			says: "no blocks",
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
			name: "a gap in the run",
			spec: plan.Plan{Blocks: []plan.Block{
				{Model: "A", DSP: 0, Pos: 0},
				{Model: "B", DSP: 0, Pos: 2},
			}},
			says: "",
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
