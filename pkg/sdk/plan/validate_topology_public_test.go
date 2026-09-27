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

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/plan"
)

type ValidateTopologyPublicTestSuite struct {
	suite.Suite
}

func (*ValidateTopologyPublicTestSuite) limits() plan.Limits {
	return plan.Limits{MaxBlocks: 6, Paths: 2, ChipCeiling: 95.0}
}

// snapshot is a chain of one block with one override, keyed by at.
func (s *ValidateTopologyPublicTestSuite) snapshot(
	at string,
) plan.Plan {
	return plan.Plan{
		Blocks: []plan.Block{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []plan.Snapshot{{
			Name:      "Lead",
			Overrides: map[string]plan.Params{at: {"Gain": catalog.Float(0.9)}},
		}},
	}
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
			name: "a snapshot overriding a block that is there",
			spec: s.snapshot("0"),
			ok:   true,
		},
		{
			name: "one overriding a block that is not",
			spec: s.snapshot("4"),
			says: "snapshot",
		},
		{
			name: "one naming a position below the first",
			spec: s.snapshot("-1"),
			says: "",
		},
		{
			// Overrides are keyed by string because JSON object keys are
			// strings. A key that is not a number is a rig nobody can act on.
			name: "one keyed by something that is not an index",
			spec: s.snapshot("amp"),
			says: "not an index",
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
