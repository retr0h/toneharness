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

package translate_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// ErrorsPublicTestSuite covers what translating refuses, and how.
//
// Both refusals carry their detail as fields. A caller deciding what to do
// about a wrong device needs the two names, not a sentence holding them.
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
			// The detail reaching a caller.
			//
			// Both, because which one is wrong decides the fix: re-measure on
			// the device in hand, or point the setup at the one the readings
			// came from.
			name: "a wrong device names both of them",
			then: func() {
				err := error(&translate.WrongDeviceError{
					Measured: "HX Stomp",
					Setup:    "Helix Floor",
				})

				s.Require().ErrorIs(err, translate.ErrWrongDevice)

				var wrong *translate.WrongDeviceError
				s.Require().ErrorAs(err, &wrong)

				s.Require().Equal("HX Stomp", wrong.Measured)
				s.Require().Equal("Helix Floor", wrong.Setup)
				s.Require().Contains(wrong.Error(), "HX Stomp, not Helix Floor")
			},
		},
		{
			// The same for a refusal to substitute.
			name: "insisting carries the gear and the reason",
			then: func() {
				err := error(&translate.InsistedError{
					Gear: "Ampeg B-15",
					Why:  "the catalog has no such model",
				})

				s.Require().ErrorIs(err, translate.ErrInsisted)

				var insisted *translate.InsistedError
				s.Require().ErrorAs(err, &insisted)

				s.Require().Equal("Ampeg B-15", insisted.Gear)
				s.Require().Contains(insisted.Error(), "Ampeg B-15")
				s.Require().Contains(insisted.Error(), "the catalog has no such model")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestEachErrorUnwrapsToItsSentinel covers matching without the struct.
func (s *ErrorsPublicTestSuite) TestEachErrorUnwrapsToItsSentinel() {
	s.Require().True(errors.Is(&translate.InsistedError{}, translate.ErrInsisted))
	s.Require().True(errors.Is(&translate.WrongDeviceError{}, translate.ErrWrongDevice))
}

func TestErrorsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
