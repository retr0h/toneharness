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
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

type ErrorsPublicTestSuite struct {
	suite.Suite
}

// TestError covers Error, which implements the error interface.
//
// One method and one table, so a case is a row rather than a file.
func (s *ErrorsPublicTestSuite) TestError() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Names the model nothing carries.
			name: "unknown block error",
			then: func() {
				tests := []struct {
					name string
					err  error
				}{
					{name: "on its own", err: &plan.UnknownBlockError{Model: "HD2_Nope"}},
					{
						// The model has to survive the wrapping every layer adds, or a
						// caller cannot say which block it was.
						name: "wrapped by a caller",
						err: fmt.Errorf("resolving: %w",
							&plan.UnknownBlockError{Model: "HD2_Nope"}),
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						s.Require().ErrorIs(tt.err, plan.ErrUnknownBlock)
						s.Require().Contains(tt.err.Error(), "HD2_Nope")

						var target *plan.UnknownBlockError
						s.Require().True(errors.As(tt.err, &target))
						s.Require().Equal("HD2_Nope", string(target.Model))
					})
				}
			},
		},
		{
			name: "over budget error names chip and cost",
			then: func() {
				err := &plan.OverBudgetError{Chip: 1, Cost: 1.2, Ceiling: 0.95}

				s.Require().Contains(err.Error(), "chip 1")
				s.Require().ErrorIs(err, plan.ErrOverBudget)
			},
		},
		{
			name: "topology error carries reason",
			then: func() {
				err := &plan.TopologyError{Reason: "too many blocks"}

				s.Require().Contains(err.Error(), "too many blocks")
				s.Require().ErrorIs(err, plan.ErrBadTopology)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func (s *ErrorsPublicTestSuite) TestSentinelsAreDistinct() {
	all := []error{plan.ErrUnknownBlock, plan.ErrOverBudget, plan.ErrBadTopology}

	for i, a := range all {
		for j, b := range all {
			if i != j {
				s.Require().NotErrorIs(a, b)
			}
		}
	}
}

func TestErrorsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
