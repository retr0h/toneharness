//go:build !darwin

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
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
)

// USBOtherPublicTestSuite covers an operating system with no USB backend.
//
// Everything else in the tool works there, so reaching for a device has to
// say plainly that it cannot, rather than fail to build or hang.
type USBOtherPublicTestSuite struct {
	suite.Suite
}

// TestOpen covers starting a session.
//
// **The guard answers before the platform does**, so this asserts the guard.
// `NewUSB` hands a test binary an Opener that will not open, whatever backend
// is underneath, because the one thing it exists to stop is a test reaching
// the pedal on somebody's desk. On an operating system with no backend there
// is nothing to reach and the refusal is redundant, but a guard that is
// skipped on some platforms is a guard with a platform-shaped hole in it.
//
// The platform's own refusal is not left untested: [USBOtherPublicTestSuite.TestList]
// asserts it twice, through the lister and through the Opener, and neither of
// those is guarded.
func (s *USBOtherPublicTestSuite) TestOpen() {
	_, err := device.NewUSB(nil).Open(context.Background())

	s.Require().ErrorIs(err, device.ErrUnderTest)
}

// TestList covers listing what is attached, through the lister and through
// the Opener built on it.
//
// Also where this platform's refusal is established, since the guard takes
// [USBOtherPublicTestSuite.TestOpen]. Listing is not guarded, because it
// changes nothing on a device, so both of these reach the backend and both
// report what it has: nothing.
func (s *USBOtherPublicTestSuite) TestList() {
	l := device.NewUSBLister()

	_, err := l.List(context.Background())

	s.Require().ErrorIs(err, device.ErrNoUSBSupport)
	s.Require().NoError(l.Close())

	_, err = device.NewUSB(nil).List(context.Background())

	s.Require().ErrorIs(err, device.ErrNoUSBSupport)
}

func TestUSBOtherPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(USBOtherPublicTestSuite))
}
