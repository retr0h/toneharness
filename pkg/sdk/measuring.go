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

package sdk

import (
	"context"
	"fmt"
	"math"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// Bench is somewhere to push a signal through and hear what comes back.
//
// An interface rather than the concrete one, because everything above it is
// arithmetic and ought to be testable without an audio interface, a cable and
// somebody in the room to plug them in.
type Bench interface {
	// Through plays a signal and returns what came back.
	Through(ctx context.Context, signal []float32) ([]float32, error)
	// Name is what the hardware calls itself.
	Name() string
}

// Rate is the sample rate the measuring loop runs at.
//
// Fixed rather than negotiated: an HX Stomp runs at 48kHz and nothing else
// without Line 6's own driver installed.
const Rate = 48000

// Hear plays a signal through the hardware and reports what came back.
//
// Nothing to do with the USB session: the signal goes out of the audio
// interface and comes back in, and the pedal is only in the middle of it.
// Keeping it off Session is not tidiness. A session held open across many
// readings takes the first live preset replace and refuses every one after
// it, so the loop that walks six hundred blocks wants a session per block and
// no session at all while it is listening.
//
// The reading is taken by the same code that measures a record, in
// [audio.Measure], which is the whole point of the method existing here
// rather than in whatever is calling it. A second implementation of the
// measurement read the reference bass as 98.6% low at 95Hz where this one
// says 93% at 138Hz, and made every block incomparable with every record
// while both looked reasonable.
func Hear(
	ctx context.Context,
	bench Bench,
	signal []float32,
) (audio.Profile, []float32, error) {
	back, err := bench.Through(ctx, signal)
	if err != nil {
		return audio.Profile{}, nil, fmt.Errorf(
			"through %s: %w", bench.Name(), err)
	}

	got := make([]float64, len(back))
	for i, v := range back {
		got[i] = float64(v)
	}

	return audio.Measure(got, Rate), back, nil
}

// Fingerprint measures whatever is in front of the device, at its own
// defaults.
//
// The preset has to be loaded already: this plays and measures and changes
// nothing, so a caller sweeping a control and a caller walking six hundred
// blocks both use it without either doing the other's work.
func Fingerprint(
	ctx context.Context,
	bench Bench,
	signal []float32,
) (measured.Figures, error) {
	read, back, err := Hear(ctx, bench, signal)
	if err != nil {
		return measured.Figures{}, err
	}

	// What came back rather than what went out. The level of the signal sent
	// is a property of the file on disk and is the same for every block; the
	// level of the answer is the one thing a volume control moves.
	return Figures(read, Level(back)), nil
}

// Figures turns a reading into the shape a measured library holds.
//
// Shares as percentages, because that is how every other number in one reads,
// and harmonics averaged across the three bands because a block is one sound
// rather than three.
func Figures(
	read audio.Profile,
	loud float64,
) measured.Figures {
	// Measuring a profile answers all three, so each is set. They are
	// pointers because a reading taken before they existed carries none, and
	// that has to stay tellable from a reading of zero.
	dynamics := read.DynamicRange
	harmonics := 100 * (read.Harmonics.Low + read.Harmonics.Mid + read.Harmonics.High) / 3
	lean := (read.EvenOdd.Low + read.EvenOdd.Mid + read.EvenOdd.High) / 3

	out := measured.Figures{
		Low:       100 * read.Low,
		Mid:       100 * read.Mid,
		High:      100 * read.High,
		Centroid:  read.Centroid,
		Dynamics:  &dynamics,
		Harmonics: &harmonics,
		Lean:      &lean,
		Level:     loud,
	}

	// Either can be absent. A transient needs a note starting and a decay
	// needs one ending, and a reading holding neither has no answer rather
	// than an answer of zero.
	if read.Transient.Known {
		v := read.Transient.Value
		out.Transient = &v
	}

	if read.Decay.Known {
		v := read.Decay.Value
		out.Decay = &v
	}

	return out
}

// Level is a signal's loudness in dBFS.
//
// Measured here rather than taken from the reading, because loudness is not
// one of the figures a record carries: a record's is the mastering engineer's
// decision and says nothing about the playing. A block's is a property of the
// block, and a volume control has nothing else to move.
//
// Silence answers a long way down rather than negative infinity, which is not
// a number anything downstream can hold.
func Level(
	signal []float32,
) float64 {
	if len(signal) == 0 {
		return quiet
	}

	var sum float64
	for _, v := range signal {
		sum += float64(v) * float64(v)
	}

	rms := math.Sqrt(sum / float64(len(signal)))
	if rms < 1e-12 {
		return quiet
	}

	return 20 * math.Log10(rms)
}

// quiet is what silence reads as, in dBFS.
const quiet = -999
