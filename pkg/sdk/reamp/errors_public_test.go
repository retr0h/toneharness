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

package reamp_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// ErrorsPublicTestSuite covers what opening hardware refuses, and how.
type ErrorsPublicTestSuite struct {
	suite.Suite
}

// TestANamedDeviceThatIsNotThereListsWhatIs covers the detail reaching a
// caller.
//
// The list travels with the error because the answer to "that device is not
// here" is almost always one of the names that are, spelled differently.
func (s *ErrorsPublicTestSuite) TestANamedDeviceThatIsNotThereListsWhatIs() {
	err := error(&reamp.NoDeviceError{
		Want:      "HX Stomp",
		Direction: "capture",
		Had:       []string{"MacBook Pro Microphone", "Scarlett 2i2"},
	})

	s.Require().ErrorIs(err, reamp.ErrNoDevice)

	var missing *reamp.NoDeviceError
	s.Require().ErrorAs(err, &missing)

	s.Require().Equal("HX Stomp", missing.Want)
	s.Require().Equal("capture", missing.Direction)
	s.Require().Len(missing.Had, 2)
	s.Require().Contains(missing.Error(), "Scarlett 2i2")
	s.Require().Contains(missing.Error(), "capture")
}

// TestNothingNamedAsksForADeviceRatherThanReportingAnEmptyOne covers the
// failure somebody actually sees when no pedal is attached.
//
// Without --hardware there is no name to quote back, and quoting the empty
// string reads as a bug in the tool rather than as a question for the person
// running it.
func (s *ErrorsPublicTestSuite) TestNothingNamedAsksForADeviceRatherThanReportingAnEmptyOne() {
	err := &reamp.NoDeviceError{
		Direction: "output",
		Had:       []string{"MacBook Pro Speakers", "John’s AirPods Max"},
	}

	s.Require().NotContains(err.Error(), `matching ""`)
	s.Require().Contains(err.Error(), "--hardware")
	s.Require().Contains(err.Error(), "John’s AirPods Max",
		"what was attached still travels with it")
}

// TestItUnwrapsToItsSentinel covers matching without the struct.
func (s *ErrorsPublicTestSuite) TestItUnwrapsToItsSentinel() {
	s.Require().True(errors.Is(&reamp.NoDeviceError{}, reamp.ErrNoDevice))
}

func TestErrorsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
