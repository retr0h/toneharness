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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

type ValidateStructurePublicTestSuite struct {
	suite.Suite
}

// TestValidateStructure covers ValidateStructure, which reports the first
// block in s whose model the catalog does not hold.
//
// One method and one table, so a case is a row rather than a file.
func (s *ValidateStructurePublicTestSuite) TestValidateStructure() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "accepts a rig of known blocks",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{{Model: "HD2_AmpTest"}}}

				s.Require().NoError(plan.ValidateStructure(newCatalog(testAmp()), spec))
			},
		},
		{
			name: "accepts an empty rig",
			then: func() {
				s.Require().NoError(plan.ValidateStructure(newCatalog(), plan.Plan{}))
			},
		},
		{
			name: "rejects an unknown model",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{
					{Model: "HD2_AmpTest"},
					{Model: "HD2_Nope"},
				}}

				err := plan.ValidateStructure(newCatalog(testAmp()), spec)

				s.Require().ErrorIs(err, plan.ErrUnknownBlock)

				var target *plan.UnknownBlockError
				s.Require().True(errors.As(err, &target))
				s.Require().Equal("HD2_Nope", target.Model)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestValidateStructurePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidateStructurePublicTestSuite))
}
