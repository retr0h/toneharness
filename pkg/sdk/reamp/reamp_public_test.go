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
	"testing"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/reamp"
)

// ReampPublicTestSuite covers the arithmetic of pushing a signal through
// hardware, without the hardware.
type ReampPublicTestSuite struct {
	suite.Suite
}

// TestPadPutsSilenceEitherSide covers the shape of what is played.
//
// The front is because latency is not known in advance. The back is because a
// reverb goes on ringing after the input stops, and cutting at the end of the
// signal would report every reverb as a short one.
func (s *ReampPublicTestSuite) TestPadPutsSilenceEitherSide() {
	signal := []float32{1, 1, 1}

	got := reamp.Pad(signal)

	lead := int(reamp.Lead.Seconds() * reamp.Rate)
	tail := int(reamp.Tail.Seconds() * reamp.Rate)

	s.Require().Len(got, lead+len(signal)+tail)
	s.Require().Zero(got[lead-1], "silence right up to the signal")
	s.Require().Equal(float32(1), got[lead], "and the signal where it was put")
	s.Require().Zero(got[lead+len(signal)], "silence again after it")
}

// TestPadOnNothing covers a signal with no samples in it.
func (s *ReampPublicTestSuite) TestPadOnNothing() {
	got := reamp.Pad(nil)

	s.Require().Len(got, int((reamp.Lead+reamp.Tail).Seconds()*reamp.Rate))
}

// TestSamplesRoundTripThroughAFrameBuffer covers reading and writing.
func (s *ReampPublicTestSuite) TestSamplesRoundTripThroughAFrameBuffer() {
	buf := make([]byte, 4*4)

	for at, want := range map[int]float32{0: 0.5, 1: -0.25, 3: 1} {
		reamp.Put(buf, at, want)
		s.Require().InDelta(want, reamp.Sample(buf, at), 0.0001)
	}
}

// TestAFrameBufferIsNeverReadPast covers a short buffer.
//
// The callback is handed a buffer sized by the device rather than by this,
// and reading past it is a crash in somebody else's C.
func (s *ReampPublicTestSuite) TestAFrameBufferIsNeverReadPast() {
	buf := make([]byte, 4)

	s.Require().Zero(reamp.Sample(buf, 9))
	s.Require().NotPanics(func() { reamp.Put(buf, 9, 1) })
	s.Require().Zero(reamp.Sample(buf, 0), "and nothing else was written")
}

// TestOverGrowsWithTheSignal covers the budget a reading gets.
//
// Loose on purpose: a device that is merely slow should be waited for, and
// one that has stopped should not be waited for all evening.
func (s *ReampPublicTestSuite) TestOverGrowsWithTheSignal() {
	short := reamp.Over(reamp.Rate)
	long := reamp.Over(10 * reamp.Rate)

	s.Require().Greater(short, 5*time.Second, "a budget with headroom in it")
	s.Require().Greater(long, short)
	s.Require().InDelta(25*time.Second, long, float64(time.Second),
		"twice ten seconds of signal, and five seconds of slack")
}

// TestDirectionNamesADeviceKind covers what a failure calls the two.
func (s *ReampPublicTestSuite) TestDirectionNamesADeviceKind() {
	s.Require().Equal("input", reamp.Direction(malgo.Capture))
	s.Require().Equal("output", reamp.Direction(malgo.Playback))
}

// TestOpenSaysWhatWasAttachedInstead covers hardware that is not there.
//
// The one test here that touches the audio system. It asks for a device
// nothing is ever called, so it reaches the failure rather than the hardware
// and says what a person would need to know.
func (s *ReampPublicTestSuite) TestOpenSaysWhatWasAttachedInstead() {
	_, err := reamp.Open("no such interface anybody owns")

	s.Require().ErrorIs(err, reamp.ErrNoDevice)
	s.Require().ErrorContains(err, "Attached:")
}

// TestCloseIsSafeTwice covers letting hardware go more than once.
func (s *ReampPublicTestSuite) TestCloseIsSafeTwice() {
	b := &reamp.Bench{}

	s.Require().NoError(b.Close())
	s.Require().NoError(b.Close())
}

func TestReampPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReampPublicTestSuite))
}
