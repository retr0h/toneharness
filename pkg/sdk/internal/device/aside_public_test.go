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

// AsidePublicTestSuite covers the guard between a test binary and the pedal on
// somebody's desk.
//
// Why it is here, on 2026-10-24. A test built a client without WithDevices,
// which falls back to the real USB bus. It then built a preset and wrote it to
// slot 11B of the pedal on the desk. The slot happened to be empty so nothing
// was lost, and that was luck: the same test against a slot holding somebody's
// work would have replaced it, and a flash write has no undo.
//
// Opening is the narrowest point and the only one guarded. Every read and
// every write goes through a session and a session comes from Open, so one
// refusal covers all of them and nothing above here loses the ability to be
// tested against a fake.
//
// **The pass-through is deliberately not tested, which is the one asymmetry.**
// A test that made the guard let go and then opened would claim whatever pedal
// is on the bus of the machine running it, which is the thing this exists to
// stop. `usb_darwin.go` is excused from the coverage target for the same
// reason: a line that cannot be reached without somebody's hardware is a line
// no test should reach.
type AsidePublicTestSuite struct {
	suite.Suite
}

// TestOpenIsRefusedInATestBinary covers the guard doing its job.
//
// The error names WithDevices, because the fix is never to work around this:
// it is to inject a bus, and a refusal that does not say so is one somebody
// spends an afternoon on.
func (s *AsidePublicTestSuite) TestOpenIsRefusedInATestBinary() {
	got, err := device.NewUSB(nil).Open(context.Background())

	s.Require().ErrorIs(err, device.ErrUnderTest)
	s.Require().ErrorContains(err, "WithDevices")
	s.Require().Nil(got, "nothing was opened")
}

// TestListPassesThrough covers the half that is not guarded.
//
// Listing the bus changes nothing: it reports what is attached and returns. So
// refusing it would buy no safety, and would take away the one test that
// enumerates against a real bus.
//
// What comes back depends on what is plugged into the machine running this, so
// only the absence of a refusal is asserted. A device on the bus and an empty
// bus are both a pass.
func (s *AsidePublicTestSuite) TestListPassesThrough() {
	_, err := device.NewUSB(nil).List(context.Background())

	s.Require().NotErrorIs(err, device.ErrUnderTest)
}

func TestAsidePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AsidePublicTestSuite))
}
