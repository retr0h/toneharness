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
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// VolumePublicTestSuite covers pinning the computer's own output level.
//
// The level is in the signal path on the rig where the computer plays the
// reference into the pedal, and it is a tone control rather than a level
// control: an amplifier's distortion depends on how hard it is driven, so two
// campaigns at different settings measure two different amplifiers.
//
// Reading it needs the platform's own interface, so what can be tested without
// one is the contract around it: the range it refuses, what it reports when the
// platform will not answer, and that `Held` reports the level in force rather
// than the level asked for.
type VolumePublicTestSuite struct {
	suite.Suite
}

// TestSetVolumeRefusesWhatIsNotALevel covers the range.
//
// Checked before the platform is asked, because a platform told to set 300 may
// clamp it silently and the rig then sits somewhere nobody chose.
func (s *VolumePublicTestSuite) TestSetVolumeRefusesWhatIsNotALevel() {
	for _, at := range []int{-1, 101, 1000} {
		err := reamp.SetVolume(at)

		s.Require().Error(err, "%d is not a level", at)
		s.Require().ErrorContains(err, "0 to 100")

		var fault *reamp.VolumeError
		s.Require().ErrorAs(err, &fault)
		s.Require().Equal("setting", fault.Doing,
			"the error says which way it failed, so a caller can tell reading "+
				"the level from changing it")
	}
}

// TestVolumeErrorSaysWhichWayItFailed covers the type's own contract.
func (s *VolumePublicTestSuite) TestVolumeErrorSaysWhichWayItFailed() {
	inner := errors.New("the platform declined")
	err := &reamp.VolumeError{Doing: "reading", Err: inner}

	s.Require().ErrorContains(err, "reading")
	s.Require().ErrorContains(err, "output volume")
	s.Require().ErrorContains(err, "the platform declined")
	s.Require().ErrorIs(err, inner, "the platform's own error has to stay reachable")
}

// TestWhatThePlatformAnswers covers the two worlds this compiles into.
//
// On macOS the level is readable, so `Volume` answers one and `Held` leaves it
// where it already is. Everywhere else there is no portable interface to a
// system output level, and the honest answer is to say so rather than to record
// a number nobody checked.
func (s *VolumePublicTestSuite) TestWhatThePlatformAnswers() {
	at, err := reamp.Volume()

	if runtime.GOOS != "darwin" {
		s.Require().ErrorIs(err, reamp.ErrNoVolume,
			"a platform that cannot be asked says so")

		_, _, held := reamp.Held(38)
		s.Require().ErrorIs(held, reamp.ErrNoVolume)

		return
	}

	s.Require().NoError(err)
	s.Require().GreaterOrEqual(at, 0)
	s.Require().LessOrEqual(at, 100)

	// Asking for the level it is already at moves nothing and says so, which is
	// what keeps a campaign from reporting a change it did not make.
	now, moved, err := reamp.Held(at)
	s.Require().NoError(err)
	s.Require().False(moved, "it was already there")
	s.Require().Equal(at, now, "and the level in force is what comes back")
}

func TestVolumePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VolumePublicTestSuite))
}
