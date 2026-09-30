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
	"context"
	"testing"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/reamp"
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

// nullDevice names miniaudio's own stand-in, which these tests run the loop
// against.
//
// Named rather than left empty. Naming nothing now means the pedal, and the
// null backend's device is not one, so a test that left it blank would be
// asking for hardware it does not have.
const nullDevice = "null"

// TestDirectionNamesADeviceKind covers what a failure calls the two.
func (s *ReampPublicTestSuite) TestDirectionNamesADeviceKind() {
	s.Require().Equal("input", reamp.Direction(malgo.Capture))
	s.Require().Equal("output", reamp.Direction(malgo.Playback))
}

// TestAnUnnamedDeviceIsThePedal covers choosing without --hardware.
//
// A regression, and the failure was silent. strings.Contains on an empty
// string is true of everything, so naming nothing took whichever device the
// platform enumerated first. That was the HX Stomp all day on one machine and
// then a pair of Bluetooth headphones, and a punk chain measured -80.3dB
// through "John's AirPods Max" instead of refusing.
func (s *ReampPublicTestSuite) TestAnUnnamedDeviceIsThePedal() {
	tests := []struct {
		name string
		got  string
		want string
		is   bool
	}{
		{"the pedal, nothing named", "HX Stomp", "", true},
		{"another of the family", "Helix", "", true},
		{"speakers are not the pedal", "MacBook Pro Speakers", "", false},
		{"nor are headphones", "John’s AirPods Max", "", false},
		{"named, and case folded", "HX Stomp", "stomp", true},
		{"named, and not this one", "MacBook Pro Speakers", "stomp", false},

		// Named outright, so the caller has overruled the preference and gets
		// what they asked for.
		{"speakers, named outright", "MacBook Pro Speakers", "macbook", true},

		// A prefix rather than a substring: an interface with the letters in
		// its name is not the pedal.
		{"an interface that merely says HX", "Behringer HX Mixer", "", false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.is, reamp.Matches(tt.got, tt.want))
		})
	}
}

// TestALoopTheBackendIsResamplingIsRefused covers the rate check.
//
// config.SampleRate is a request. miniaudio answers a device that cannot meet
// it by resampling rather than by refusing, so the loop asks the backend what
// it actually negotiated and stops when that is not the rate every committed
// figure was taken at.
//
// Refused rather than converted: a resampler's artefacts would land in a figure
// whose only purpose is comparison against resources/sweeps/, and a reading
// that is merely plausible is the failure this loop keeps having.
func (s *ReampPublicTestSuite) TestALoopTheBackendIsResamplingIsRefused() {
	tests := []struct {
		name    string
		in, out uint32
		refused bool
	}{
		{name: "the rate everything was measured at", in: 48000, out: 48000},
		{
			// What Line 6's own driver or another interface may give.
			name: "a backend resampling the capture side",
			in:   44100, out: 48000, refused: true,
		},
		{
			// Negotiated separately, so one side is enough to spoil a reading.
			name: "and the playback side",
			in:   48000, out: 96000, refused: true,
		},
		{name: "both", in: 44100, out: 44100, refused: true},
	}

	b := &reamp.Bench{}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := b.Rated(tt.in, tt.out)

			if !tt.refused {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, reamp.ErrRate)

			var wrong *reamp.RateError
			s.Require().ErrorAs(err, &wrong)
			s.Require().Equal(uint32(48000), wrong.Want)
			s.Require().Equal(tt.in, wrong.Capture)
			s.Require().Equal(tt.out, wrong.Playback)

			// It says what to do about it, because the answer is a setting on
			// the machine rather than anything in this repository.
			s.Require().ErrorContains(err, "--hardware")
		})
	}
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

// TestAPassPlaysTheSignalAndKeepsTheAnswer covers the loop's own arithmetic.
//
// The callback this runs inside cannot be reached from a test: miniaudio
// ships a null backend that would allow it and the bindings here compile it
// out. Everything the callback decides is here instead.
func (s *ReampPublicTestSuite) TestAPassPlaysTheSignalAndKeepsTheAnswer() {
	signal := []float32{0.25, 0.5, 0.75}
	run := reamp.NewPass(signal)

	frames := 2
	output := make([]byte, frames*2*4)
	input := make([]byte, frames*2*4)

	// The lead is silence, so the first frames out carry nothing and the
	// signal arrives later.
	s.Require().False(run.Frame(output, input, frames))
	s.Require().Zero(reamp.Sample(output, 0))

	// What comes back is kept in the order it arrived, on the left channel.
	reamp.Put(input, 0, 0.9)
	reamp.Put(input, 2, 0.8)
	s.Require().False(run.Frame(output, input, frames))

	got := run.Got()
	s.Require().Len(got, 2*frames)
	s.Require().InDelta(0.9, got[2], 0.0001)
	s.Require().InDelta(0.8, got[3], 0.0001)
}

// TestAPassPlaysTheSignalOnBothChannels covers what the loop is.
//
// A cable from the device's output back to its own input, so the signal goes
// out of both and the left one comes back.
func (s *ReampPublicTestSuite) TestAPassPlaysTheSignalOnBothChannels() {
	run := reamp.NewPass([]float32{1})

	lead := int(reamp.Lead.Seconds() * reamp.Rate)
	output := make([]byte, 2*4)
	input := make([]byte, 2*4)

	// Far enough in that the signal itself is what is being played.
	for range lead {
		run.Frame(output, input, 1)
	}

	run.Frame(output, input, 1)

	s.Require().InDelta(1, reamp.Sample(output, 0), 0.0001)
	s.Require().InDelta(1, reamp.Sample(output, 1), 0.0001,
		"both channels carry it")
}

// TestAPassEndsWhenBothSidesAreDone covers the reading finishing.
//
// Not when the last sample is played: a converter answers late, so the
// capture runs on past the end of the signal and the tail is what a decay is
// measured from.
func (s *ReampPublicTestSuite) TestAPassEndsWhenBothSidesAreDone() {
	run := reamp.NewPass([]float32{1, 1, 1})

	frames := 4096
	output := make([]byte, frames*2*4)
	input := make([]byte, frames*2*4)

	var done bool
	for range 200 {
		if run.Frame(output, input, frames) {
			done = true

			break
		}
	}

	s.Require().True(done, "the reading ends rather than running forever")
	s.Require().Len(run.Got(), cap(run.Got()),
		"and it kept the margin a converter answers late by")
}

// TestAPassPlaysSilenceOnceTheSignalRunsOut covers the tail.
func (s *ReampPublicTestSuite) TestAPassPlaysSilenceOnceTheSignalRunsOut() {
	run := reamp.NewPass(nil)

	frames := 4096
	output := make([]byte, frames*2*4)
	input := make([]byte, frames*2*4)

	for range 200 {
		if run.Frame(output, input, frames) {
			break
		}
	}

	s.Require().Zero(reamp.Sample(output, 0),
		"nothing is played once there is nothing left to play")
}

// TestThrough covers Through, which plays a signal and returns what came
// back.
//
// One method and one table, so a case is a row rather than a file.
func (s *ReampPublicTestSuite) TestThrough() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Playing and capturing for real.
			//
			// Against miniaudio's null backend, which presents a device that
			// takes samples and hands back silence. Everything but the
			// converters is exercised: the stream opens, the callback runs,
			// the signal is fed out of it a frame at a time, what arrives is
			// kept, and the reading ends when both are done.
			name: "through runs the whole loop",
			then: func() {
				b, err := reamp.OpenWith([]malgo.Backend{reamp.NullBackend}, nullDevice)
				s.Require().NoError(err)

				defer func() { _ = b.Close() }()

				s.Require().NotEmpty(b.Name())

				signal := make([]float32, reamp.Rate/20)
				got, err := b.Through(context.Background(), signal)

				s.Require().NoError(err)
				s.Require().Len(got,
					len(reamp.Pad(signal))+int(reamp.Margin.Seconds()*reamp.Rate),
					"as many samples as were played, and the margin a converter answers "+
						"late by")
			},
		},
		{
			// The budget.
			//
			// A device that stops delivering callbacks would otherwise leave
			// a reading blocked forever, and a campaign that hangs on block
			// two hundred looks exactly like one still working.
			name: "through gives up on a device that stopped",
			then: func() {
				b, err := reamp.OpenWith([]malgo.Backend{reamp.NullBackend}, nullDevice)
				s.Require().NoError(err)

				defer func() { _ = b.Close() }()

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				_, err = b.Through(ctx, make([]float32, reamp.Rate))

				s.Require().ErrorContains(err, "stopped answering")
			},
		},
		{
			// The claim deadline.
			//
			// The failure it exists for is a device that neither opens nor
			// refuses. On macOS that is a program with no Microphone
			// permission: CoreAudio blocks inside the first capture while the
			// system waits for somebody to answer a dialog, and a terminal
			// running unattended has nobody to answer it. A sweep sat there
			// for six minutes having measured nothing before this had a
			// deadline, which is the failure the budget below was already
			// written to prevent one stage later.
			//
			// No backend fakes a device that blocks on open, so the deadline
			// is made short enough that one which does open still misses it.
			name: "through gives up on a device that never opens",
			then: func() {
				b, err := reamp.OpenClaiming([]malgo.Backend{reamp.NullBackend}, nullDevice,
					time.Nanosecond)
				s.Require().NoError(err)

				defer func() { _ = b.Close() }()

				_, err = b.Through(context.Background(), make([]float32, reamp.Rate))

				s.Require().ErrorIs(err, reamp.ErrUnclaimed)
				s.Require().ErrorContains(err, "Microphone access",
					"the error says what to go and do, not only that it failed")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestOpenRefusesHardwareThatIsNotThere covers a name nothing answers to.
func (s *ReampPublicTestSuite) TestOpenRefusesHardwareThatIsNotThere() {
	_, err := reamp.OpenWith([]malgo.Backend{reamp.NullBackend},
		"no such interface anybody owns")

	s.Require().ErrorIs(err, reamp.ErrNoDevice)
}

// TestAPassStridesByTheDevicesOwnChannelCount covers an eight-channel device.
//
// The regression this exists for cost a day twice. A device is opened at its own
// channel count and an HX Stomp presents eight, so one frame is eight slots wide
// and frame i starts at slot i*8. A loop striding by two walks across the first
// frame's channels instead of down successive frames: a quarter of the signal
// goes out, smeared sideways, and reading it back makes the same error.
//
// It does not fail, which is the problem. It measures: the empty loop read
// 11,990Hz at -46dB where it reads 95Hz at -21dB, and every block in a campaign
// came back with the same wrong figure looking like data.
func (s *ReampPublicTestSuite) TestAPassStridesByTheDevicesOwnChannelCount() {
	const channels = 8

	signal := []float32{0.25, 0.5, 0.75, 1}
	run := reamp.NewPass(signal)

	frames := 4
	output := make([]byte, frames*channels*4)
	input := make([]byte, frames*channels*4)

	lead := int(reamp.Lead.Seconds() * reamp.Rate)
	for range lead / frames {
		run.Frame(output, input, frames)
	}

	// What comes back is read one per frame, at the frame's own start.
	for i := range frames {
		reamp.Put(input, i*channels, float32(i+1)/10)
	}

	run.Frame(output, input, frames)

	got := run.Got()
	s.Require().GreaterOrEqual(len(got), frames)

	tail := got[len(got)-frames:]
	for i := range frames {
		s.Require().InDelta(float32(i+1)/10, tail[i], 0.0001,
			"frame %d is read at slot %d, not %d", i, i*channels, i*2)
	}
}

// TestAPassLeavesTheOtherChannelsAlone covers where the signal is put.
//
// The first pair, because that is where the Main out listens. A device
// presenting eight wants silence on the other six rather than six copies: the
// loop is a lead out of one socket and back into another, and anything on the
// remaining channels is somebody else's problem to explain.
func (s *ReampPublicTestSuite) TestAPassLeavesTheOtherChannelsAlone() {
	const channels = 8

	run := reamp.NewPass([]float32{1})

	lead := int(reamp.Lead.Seconds() * reamp.Rate)
	output := make([]byte, channels*4)
	input := make([]byte, channels*4)

	for range lead {
		run.Frame(output, input, 1)
	}

	run.Frame(output, input, 1)

	s.Require().InDelta(1, reamp.Sample(output, 0), 0.0001, "left")
	s.Require().InDelta(1, reamp.Sample(output, 1), 0.0001, "right")

	for ch := 2; ch < channels; ch++ {
		s.Require().Zero(reamp.Sample(output, ch),
			"channel %d carries nothing", ch)
	}
}

// TestAPassOnAMonoDevice covers an interface with no pair to write.
//
// Not a Helix, which presents eight. `--hardware` names any attached device and
// a mono one has one slot per frame, so writing a pair would put this frame's
// second sample where the next frame's first belongs.
func (s *ReampPublicTestSuite) TestAPassOnAMonoDevice() {
	run := reamp.NewPass([]float32{1, 1})

	frames := 2
	output := make([]byte, frames*4)
	input := make([]byte, frames*4)

	lead := int(reamp.Lead.Seconds() * reamp.Rate)
	for range lead / frames {
		run.Frame(output, input, frames)
	}

	s.Require().NotPanics(func() { run.Frame(output, input, frames) })

	s.Require().InDelta(1, reamp.Sample(output, 0), 0.0001)
	s.Require().InDelta(1, reamp.Sample(output, 1), 0.0001,
		"the next frame's own sample, not a second copy of this one")
}

// TestChannelsInAnswersOneRatherThanDividingByNothing covers the two guards.
//
// The stride is worked out from the buffer rather than assumed, because how
// many channels a device presents varies with the model. Both guards answer
// one rather than failing: a callback is on the audio thread, and an answer
// of zero there divides by nothing on the next frame.
func (s *ReampPublicTestSuite) TestChannelsInAnswersOneRatherThanDividingByNothing() {
	tests := []struct {
		name   string
		buf    []byte
		frames int
		want   int
	}{
		{"stereo float32", make([]byte, 8*2*4), 8, 2},
		{"mono", make([]byte, 8*4), 8, 1},
		{"eight in, which is what a Stomp presents", make([]byte, 4*8*4), 4, 8},
		{"no frames at all", make([]byte, 64), 0, 1},
		{"a negative count", make([]byte, 64), -1, 1},
		{"a buffer too short for one frame", make([]byte, 2), 8, 1},
		{"no buffer", nil, 8, 1},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, reamp.ChannelsIn(tt.buf, tt.frames))
		})
	}
}

// TestTwoSidedIsWhatOpensTheLoop covers the answer pkg/cli asks for.
//
// The headroom trim exists because a lead from the pedal's output to its own
// input makes the chain hear itself. Two devices means the pedal's output
// reaches nothing, so there is no path back and the trim spends thirty
// decibels on a problem that is absent. Getting this wrong either squeals or
// throws away signal, and both are silent.
func (s *ReampPublicTestSuite) TestTwoSidedIsWhatOpensTheLoop() {
	tests := []struct {
		name string
		want string
		two  bool
	}{
		{"one device is the loop", "HX Stomp", false},
		{"two devices open it", "External Headphones,HX Stomp", true},
		{"spaces round the comma still open it", "speakers , stomp", true},
		{"naming nothing is one device", "", false},
		{
			// Somebody typing the same name twice has described one device.
			name: "the same device twice is not two", want: "HX Stomp,HX Stomp",
		},
		{
			name: "and the same name with spaces round it is still not two",
			want: "HX Stomp, HX Stomp",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.two, reamp.TwoSided(tt.want))
		})
	}
}

func TestReampPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReampPublicTestSuite))
}

// TestSidesSplitsThePlaybackDeviceFromTheCaptureOne is how the measuring loop is
// opened.
//
// One device both ways is what closes it. The chain can only be reached through
// the physical input jack, and the only cable that reaches it from the computer
// comes off the pedal's own output, carrying the chain's output with it. Playing
// through the computer's own output instead means the cable carries nothing but
// the reference, so there is no path back at all.
func (s *ReampPublicTestSuite) TestSidesSplitsThePlaybackDeviceFromTheCaptureOne() {
	for _, tt := range []struct {
		name      string
		give      string
		play, rec string
	}{
		{
			name: "one name is one device both ways",
			give: "hx stomp",
			play: "hx stomp", rec: "hx stomp",
		},
		{
			// In the order the signal travels, which is the order somebody
			// describes a rig in: out of the first, back into the second.
			name: "two names play through the first and record from the second",
			give: "MacBook Pro Speakers,HX Stomp",
			play: "MacBook Pro Speakers", rec: "HX Stomp",
		},
		{
			name: "spaces around the comma are somebody typing, not a device",
			give: "speakers , stomp",
			play: "speakers", rec: "stomp",
		},
		{
			// Naming nothing still means the pedal rather than whichever device
			// the platform enumerates first, which is matches' job and must not
			// be broken by splitting.
			name: "nothing named stays nothing named on both sides",
			give: "",
			play: "", rec: "",
		},
	} {
		s.Run(tt.name, func() {
			play, rec := reamp.Sides(tt.give)
			s.Require().Equal(tt.play, play, "the device played through")
			s.Require().Equal(tt.rec, rec, "the device recorded from")
		})
	}
}
