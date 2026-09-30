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

package audio_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// MeasurePublicTestSuite covers the numbers a recording reads as.
//
// Every case here is a signal whose answer can be stated before it is
// measured. These definitions are this project's own rather than a known
// algorithm, so they are worth less than the transform underneath them until
// something predictable has been pointed at them.
type MeasurePublicTestSuite struct {
	suite.Suite
}

// TestMeasureReadsWhatASignalIs covers every number a recording measures as.
//
// Each row is a signal whose answer can be stated before it is measured, and
// most compare two signals rather than assert one figure, because what the
// definitions have to get right is the ordering between two sounds. They are
// this project's own rather than a known algorithm, so they are worth less
// than the transform underneath them until something predictable has been
// pointed at them.
//
// One method and one table, so a new figure is a row rather than a file.
func (s *MeasurePublicTestSuite) TestMeasureReadsWhatASignalIs() {
	// struck is silence and then a note starting at once, which is the
	// sharpest start there is and the only shape with a rise to clamp.
	struck := func(amp, pluck float64) []float64 {
		return append(audio.Silence(0.3, rate),
			audio.Plucked(110, 0.7, rate, amp, pluck)...)
	}

	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// One frequency, so its energy is in whichever band holds it, its centroid is
			// that frequency, and there is nothing above it.
			name: "a sine is all fundamental",
			then: func() {
				got := audio.Measure(audio.Sine(100, 1.0, rate, 0.8), rate)

				s.Require().InDelta(1.0, got.Seconds, 0.001)
				s.Require().Equal(rate, got.Rate)

				s.Require().Greater(got.Low, 0.99, "a 100Hz tone is entirely low")
				s.Require().Less(got.Mid, 0.01)
				s.Require().Less(got.High, 0.01)

				s.Require().InDelta(100, got.Centroid, 15, "and sits where it sounds")
				s.Require().Less(got.Harmonics.Mid, 0.05, "with nothing above it")
			},
		},
		{
			name: "the three bands sum to one",
			then: func() {
				for _, hz := range []float64{80, 500, 5000} {
					s.Run("", func() {
						got := audio.Measure(audio.Sine(hz, 0.5, rate, 0.7), rate)

						s.Require().InDelta(1.0, got.Low+got.Mid+got.High, 0.001)
					})
				}
			},
		},
		{
			name: "each band catches its own",
			then: func() {
				low := audio.Measure(audio.Sine(80, 0.5, rate, 0.7), rate)
				mid := audio.Measure(audio.Sine(800, 0.5, rate, 0.7), rate)
				high := audio.Measure(audio.Sine(5000, 0.5, rate, 0.7), rate)

				s.Require().Greater(low.Low, 0.95)
				s.Require().Greater(mid.Mid, 0.95)
				s.Require().Greater(high.High, 0.95)
			},
		},
		{
			name: "a higher note sits higher",
			then: func() {
				deep := audio.Measure(audio.Sine(60, 0.5, rate, 0.7), rate)
				bright := audio.Measure(audio.Sine(900, 0.5, rate, 0.7), rate)

				s.Require().Greater(bright.Centroid, deep.Centroid*5)
			},
		},
		{
			// A square holds every odd multiple of its fundamental and no even ones, so
			// it should read as harmonically rich and leaning odd. That lean is the
			// difference between a valve's warmth and a fuzz's edge.
			name: "a square is odd harmonics",
			then: func() {
				clean := audio.Measure(audio.Sine(200, 0.5, rate, 0.5), rate)
				dirty := audio.Measure(audio.Square(200, 0.5, rate, 0.5), rate)

				s.Require().Greater(dirty.Harmonics.Mid, clean.Harmonics.Mid*3,
					"a square carries far more above its fundamental than a sine")
				s.Require().Negative(dirty.EvenOdd.Mid, "and what it carries is odd")
			},
		},
		{
			name: "a spread is ordered",
			then: func() {
				got := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 4), rate)

				s.Require().LessOrEqual(got.Harmonics.Low, got.Harmonics.Mid)
				s.Require().LessOrEqual(got.Harmonics.Mid, got.Harmonics.High)

				s.Require().LessOrEqual(got.EvenOdd.Low, got.EvenOdd.Mid)
				s.Require().LessOrEqual(got.EvenOdd.Mid, got.EvenOdd.High)
			},
		},
		{
			// A tone that never changes measures the same in every window, so its three
			// numbers sit almost on top of each other. A note that is struck and then
			// dies away does not, and a median alone would report the two as the same
			// kind of measurement.
			name: "a note that changes spreads wider",
			then: func() {
				steady := audio.Measure(audio.Sine(110, 2.0, rate, 0.8), rate)
				plucked := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 4), rate)

				s.Require().Greater(
					plucked.Harmonics.High-plucked.Harmonics.Low,
					steady.Harmonics.High-steady.Harmonics.Low,
				)
			},
		},
		{
			name: "a plucked note decays faster than a tone held",
			then: func() {
				held := audio.Measure(audio.Sine(110, 2.0, rate, 0.8), rate)
				plucked := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 4), rate)

				s.Require().True(plucked.Decay.Known, "a note that dies away has a decay")
				s.Require().False(held.Decay.Known,
					"a tone that never falls to a quarter has none to report")

				s.Require().Less(plucked.Decay.Value, 1.0, "and does it within the recording")
			},
		},
		{
			name: "a harder pluck decays sooner",
			then: func() {
				slow := audio.Measure(audio.Plucked(110, 2.0, rate, 0.8, 2), rate)
				fast := audio.Measure(audio.Plucked(110, 2.0, rate, 0.8, 8), rate)

				s.Require().True(fast.Decay.Known)
				s.Require().True(slow.Decay.Known)
				s.Require().Less(fast.Decay.Value, slow.Decay.Value)
			},
		},
		{
			// Three notes: one long, one short, one long. Timed from the loudest frame in
			// the recording, the answer is whichever single note happened to peak highest.
			// Timed per note, it is the middle of the three, which is what describes the
			// playing rather than one moment of it.
			name: "decay is per note rather than per recording",
			then: func() {
				samples := make([]float64, 0, rate*4)

				// A loud short note first, so a global peak would time that one.
				samples = append(samples, audio.Plucked(110, 0.6, rate, 0.9, 12)...)
				samples = append(samples, audio.Silence(0.2, rate)...)
				samples = append(samples, audio.Plucked(110, 1.4, rate, 0.5, 2)...)
				samples = append(samples, audio.Silence(0.2, rate)...)
				samples = append(samples, audio.Plucked(110, 1.4, rate, 0.5, 2)...)

				got := audio.Measure(samples, rate)

				s.Require().True(got.Decay.Known)
				s.Require().Greater(got.Decay.Value, 0.15,
					"two of the three notes ring, so the middle is not the short one")
			},
		},
		{
			// On a dense line the loudest frame is one accent, and the level drops back to
			// the ongoing playing straight after it. Timing that one fall reported the
			// accent rather than the notes, which is how a bassist whose notes ring read
			// 0.03s across three records.
			name: "an accent does not decide the decay",
			then: func() {
				var samples []float64
				for range 6 {
					samples = append(samples, audio.Plucked(110, 0.5, rate, 0.4, 3)...)
				}

				// One accent, twice as loud, in the middle of the line.
				accent := audio.Plucked(110, 0.5, rate, 0.9, 3)
				samples = append(samples[:len(samples)/2], append(accent, samples[len(samples)/2:]...)...)

				got := audio.Measure(samples, rate)

				s.Require().True(got.Decay.Known)
				s.Require().Greater(got.Decay.Value, 0.05,
					"the notes decide it, not the one frame that was loudest")
			},
		},
		{
			name: "a steady tone is not dynamic",
			then: func() {
				got := audio.Measure(audio.Sine(220, 1.0, rate, 0.7), rate)

				s.Require().Less(got.DynamicRange, 1.0,
					"a tone that never changes has almost no range")
			},
		},
		{
			name: "a plucked note is dynamic",
			then: func() {
				steady := audio.Measure(audio.Sine(110, 2.0, rate, 0.8), rate)
				plucked := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 3), rate)

				s.Require().Greater(plucked.DynamicRange, steady.DynamicRange+5,
					"a note that starts loud and dies away covers real ground")
			},
		},
		{
			// The ordering here was measured before it was asserted. Across a struck
			// note, a swelled one, a held tone, a square and noise, the struck cases land
			// above 0.94 and everything gradual below 0.1. An earlier definition measured
			// how much the rises varied among themselves and put a cleanly struck note at
			// zero, because one rise has nothing to vary against.
			name: "a sudden start reads as transient",
			then: func() {
				// Silence, then a note starting at once: the sharpest start there is.
				struck := audio.Measure(
					struck(0.9, 6), rate)
				ringing := audio.Measure(
					append(audio.Silence(0.3, rate), audio.Plucked(110, 0.7, rate, 0.9, 1)...), rate)

				held := audio.Measure(audio.Sine(110, 1.0, rate, 0.7), rate)
				swelled := audio.Measure(audio.Swell(110, 1.0, rate, 0.8), rate)

				s.Require().Greater(struck.Transient.Value, 0.9, "a struck note arrives at once")
				s.Require().Greater(ringing.Transient.Value, 0.9, "however long it then rings")

				s.Require().False(held.Transient.Known,
					"a held tone is at level from the first frame and never arrives at all")
				s.Require().Less(swelled.Transient.Value, 0.1, "and a swell arrives gradually")

				s.Require().Greater(struck.Transient.Value, swelled.Transient.Value*5,
					"the two are not close")
			},
		},
		{
			// The number alone cannot tell them apart: a swell's largest rise is 2.9% of
			// its peak and a held sine's is 2.5%. What separates them is where the signal
			// starts. A swell begins at 0.2% of its peak and climbs; a sine is already at
			// 92% of its peak in the first frame and has nowhere to climb from.
			// Reporting the sine's 2.5% as a transient read it as a note swelled in over
			// three seconds, which is the opposite of a tone that simply began.
			name: "a tone's attack is not measurable",
			then: func() {
				tone := audio.Measure(audio.Sine(110, 3.0, rate, 0.8), rate)
				swelled := audio.Measure(audio.Swell(110, 1.0, rate, 0.8), rate)

				s.Require().False(tone.Transient.Known,
					"a tone at full level in its first frame has no attack to measure")

				s.Require().True(swelled.Transient.Known,
					"a swell starts from nothing, so its rise is a real one")
				s.Require().Less(swelled.Transient.Value, 0.1, "and a gradual one")
			},
		},
		{
			name: "transient ignores how loud",
			then: func() {
				quiet := audio.Measure(
					append(audio.Silence(0.3, rate), audio.Plucked(110, 0.7, rate, 0.2, 6)...), rate)
				loud := audio.Measure(
					struck(0.9, 6), rate)

				s.Require().InDelta(quiet.Transient.Value, loud.Transient.Value, 0.05)
			},
		},
		{
			// Silence and then a note struck at once: the rise into it is the whole of the
			// peak, measured at exactly 1.0000, and nothing should report more than all of
			// it.
			// The note on its own is not this case. It is at full level in its first
			// frame, so it has no rise to clamp and no attack to report.
			name: "transient never exceeds one",
			then: func() {
				got := audio.Measure(
					append(audio.Silence(0.3, rate), audio.Plucked(110, 0.7, rate, 0.9, 8)...), rate)

				s.Require().True(got.Transient.Known)
				s.Require().LessOrEqual(got.Transient.Value, 1.0)
				s.Require().GreaterOrEqual(got.Transient.Value, 0.0)

				bare := audio.Measure(audio.Plucked(110, 1.0, rate, 0.9, 8), rate)
				s.Require().False(bare.Transient.Known)
			},
		},
		{
			name: "silence measures as nothing",
			then: func() {
				got := audio.Measure(audio.Silence(0.5, rate), rate)

				s.Require().InDelta(0.5, got.Seconds, 0.001, "it still has a length")

				s.Require().InDelta(0, got.Low, 1e-9)
				s.Require().InDelta(0, got.Mid, 1e-9)
				s.Require().InDelta(0, got.High, 1e-9)
				s.Require().InDelta(0, got.Centroid, 1e-9)
				s.Require().Equal(audio.Spread{}, got.Harmonics)
				s.Require().Equal(audio.Spread{}, got.EvenOdd)
				s.Require().InDelta(0, got.DynamicRange, 1e-9)

				s.Require().False(got.Decay.Known, "there is no note to fall")
				s.Require().False(got.Transient.Known, "and none to start")
			},
		},
		{
			name: "no samples at all is an empty profile rather than a guess",
			then: func() {
				got := audio.Measure(nil, rate)

				s.Require().InDelta(0, got.Seconds, 1e-9)
				s.Require().Equal(rate, got.Rate)
				s.Require().InDelta(0, got.Centroid, 1e-9)
			},
		},
		{
			name: "a rate of nothing is a caller's mistake rather than a division by zero",
			then: func() {
				got := audio.Measure(audio.Sine(100, 0.5, rate, 0.5), 0)

				s.Require().InDelta(0, got.Seconds, 1e-9)
				s.Require().InDelta(0, got.Centroid, 1e-9)
			},
		},
		{
			name: "audio below one frame of anything",
			then: func() {
				got := audio.Measure([]float64{0.1, -0.1, 0.2}, rate)

				s.Require().False(got.Transient.Known)
				s.Require().False(got.Decay.Known)
				s.Require().InDelta(0, got.DynamicRange, 1e-9)
			},
		},
		{
			// The opposite of a sine, and the case that catches a band measurement which
			// quietly puts everything in one place.
			name: "noise is spread across everything",
			then: func() {
				got := audio.Measure(audio.Noise(1.0, rate, 0.5, 7), rate)

				s.Require().Greater(got.High, 0.5, "most of the spectrum is above 2kHz")
				s.Require().Greater(got.Centroid, 3000.0, "and its centre is high up")
			},
		},
		{
			// The same signal twice as loud is the same sound. Anything here that changes
			// with amplitude alone is measuring the recording level, not the playing.
			name: "loudness does not move the shape",
			then: func() {
				quiet := audio.Measure(audio.Square(150, 1.0, rate, 0.2), rate)
				loud := audio.Measure(audio.Square(150, 1.0, rate, 0.8), rate)

				s.Require().InDelta(quiet.Low, loud.Low, 0.01)
				s.Require().InDelta(quiet.Mid, loud.Mid, 0.01)
				s.Require().InDelta(quiet.Centroid, loud.Centroid, 1.0)
				s.Require().InDelta(quiet.Harmonics.Mid, loud.Harmonics.Mid, 0.01)
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

func TestMeasurePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasurePublicTestSuite))
}
