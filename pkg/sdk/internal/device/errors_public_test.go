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

package device_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
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
			name: "unknown model error names the product",
			then: func() {
				err := &device.UnknownModelError{Product: 0xBEEF}

				s.Require().Contains(err.Error(), "0xbeef")
				s.Require().ErrorIs(err, device.ErrUnknownModel)
			},
		},
		{
			// A device answering with something nobody can decode, which is
			// how a protocol change becomes visible.
			name: "not a preset error",
			then: func() {
				tests := []struct {
					name   string
					result any
					want   string
				}{
					{
						name:   "a decoded document",
						result: map[any]any{1: "a", 2: "b"},
						want:   "map with 2 keys",
					},
					{
						name:   "a run of bytes",
						result: []byte{1, 2, 3},
						want:   "3 bytes",
					},
					{
						name:   "something else entirely",
						result: 42,
						want:   "int",
					},
					{name: "nothing recognisable at all", want: "<nil>"},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						err := &device.NotAPresetError{Result: tt.result}

						s.Require().Equal(tt.want, err.Shape())
						s.Require().Contains(err.Error(), tt.want)
						s.Require().ErrorIs(err, device.ErrNotAPreset)
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func (s *ErrorsPublicTestSuite) TestUnknownModelErrorSurvivesWrapping() {
	err := fmt.Errorf("opening: %w", &device.UnknownModelError{Product: 0xBEEF})

	var target *device.UnknownModelError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal(uint16(0xBEEF), target.Product)
}

func (s *ErrorsPublicTestSuite) TestSentinelsAreDistinct() {
	s.Require().NotErrorIs(device.ErrNoDevice, device.ErrUnknownModel)
}

func TestErrorsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
