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

type ValidateBudgetPublicTestSuite struct {
	suite.Suite
}

func (*ValidateBudgetPublicTestSuite) limits() plan.Limits {
	return plan.Limits{MaxBlocks: 6, Paths: 2, ChipCeiling: 95.0}
}

// blocks is a chain of n identical amps on one chip.
func (s *ValidateBudgetPublicTestSuite) blocks(
	n, dsp int,
	enabled bool,
) plan.Plan {
	out := plan.Plan{}
	for i := range n {
		out.Blocks = append(out.Blocks, plan.Block{
			Model: "HD2_AmpTest", DSP: dsp, Pos: i, Enabled: enabled,
		})
	}

	return out
}

// TestValidateBudget checks a chain against what the chips can carry.
func (s *ValidateBudgetPublicTestSuite) TestValidateBudget() {
	stereo := testAmp()
	stereo.Stereo = true

	assumed := testAmp()
	assumed.DSP.Prov = catalog.ProvAssumed

	tests := []struct {
		name   string
		block  catalog.Block
		spec   plan.Plan
		limits plan.Limits
		is     error
		says   string
	}{
		{
			name: "one amp on each of two chips",
			spec: plan.Plan{
				Blocks: append(s.blocks(1, 0, true).Blocks, s.blocks(1, 1, true).Blocks...),
			},
			limits: s.limits(),
		},
		{
			name:   "four on one, at 26.67 each",
			spec:   s.blocks(4, 0, true),
			limits: s.limits(),
			is:     plan.ErrOverBudget,
		},
		{
			// Processors are numbered from nothing, so two of them are 0 and
			// 1 and the first one too far is 2. `>` accepts exactly this
			// block and indexes a costs slice of length two at two, which is
			// a chain on a processor the device does not have.
			name:   "a block on the first chip past the last one",
			spec:   s.blocks(1, 2, true),
			limits: s.limits(),
			is:     plan.ErrBadTopology,
		},
		{
			name:   "and the last chip it does have is fine",
			spec:   s.blocks(1, 1, true),
			limits: s.limits(),
		},
		{
			// A bypassed block still occupies its position and still costs
			// DSP, which is why the same four fail switched off.
			name:   "four bypassed ones, which cost the same",
			spec:   s.blocks(4, 0, false),
			limits: s.limits(),
			is:     plan.ErrOverBudget,
		},
		{
			// Three stereo instances cost 120.3; three mono are 80.01.
			name:   "three stereo, charged the stereo figure",
			block:  stereo,
			spec:   s.blocks(3, 0, true),
			limits: s.limits(),
			is:     plan.ErrOverBudget,
		},
		{
			name:   "the same three, mono",
			spec:   s.blocks(3, 0, true),
			limits: s.limits(),
		},
		{
			// A cost nobody stated must not decide whether somebody's rig
			// fits.
			name:   "a cost this project guessed at",
			block:  assumed,
			spec:   s.blocks(1, 0, false),
			limits: s.limits(),
			is:     catalog.ErrBadParam,
			says:   "assumed",
		},
		{
			name:   "a block on a chip the device does not have",
			spec:   s.blocks(1, 5, false),
			limits: s.limits(),
			is:     plan.ErrBadTopology,
		},
		{
			name:   "one on a chip below the first",
			spec:   s.blocks(1, -1, false),
			limits: s.limits(),
			is:     plan.ErrBadTopology,
		},
		{
			name:   "limits declaring no chips at all",
			spec:   s.blocks(1, 0, false),
			limits: plan.Limits{Paths: 0},
			is:     plan.ErrBadTopology,
		},
		{
			name:   "a model the catalog does not carry",
			spec:   plan.Plan{Blocks: []plan.Block{{Model: "HD2_Nope"}}},
			limits: s.limits(),
			is:     plan.ErrUnknownBlock,
		},
		{
			// Regression: ChipCeiling was once a fraction while the catalog
			// states cost in percent, so every rig was rejected as over
			// budget — including a single amp. A chain the device would
			// happily load must validate.
			name:   "three amps on the one path a Stomp has",
			spec:   s.blocks(3, 0, true),
			limits: plan.HXStompLimits(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			block := tt.block
			if block.ID == "" {
				block = testAmp()
			}

			err := plan.ValidateBudget(newCatalog(block), tt.spec, tt.limits)

			if tt.is == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.is)

			if tt.says != "" {
				s.Require().Contains(err.Error(), tt.says)
			}
		})
	}
}

// TestValidateBudgetNamesTheChipThatOverflowed covers the detail a caller
// reads, which is one error rather than a set of cases.
func (s *ValidateBudgetPublicTestSuite) TestValidateBudgetNamesTheChipThatOverflowed() {
	err := plan.ValidateBudget(
		newCatalog(testAmp()), s.blocks(4, 0, true), s.limits())

	var got *plan.OverBudgetError
	s.Require().ErrorAs(err, &got)
	s.Require().Equal(0, got.Chip)
	s.Require().InDelta(106.68, got.Cost, 1e-9)
}

// TestCostAndCeilingShareUnits is a property of the limits themselves.
//
// The catalog states an Ampeg SVT at 26.67. A ceiling below that would mean
// no amp ever fits, which is how the units drifted apart before.
func (s *ValidateBudgetPublicTestSuite) TestCostAndCeilingShareUnits() {
	s.Require().Greater(plan.HXStompLimits().ChipCeiling, 26.67,
		"the ceiling must admit at least one amp")
}

func TestValidateBudgetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidateBudgetPublicTestSuite))
}
