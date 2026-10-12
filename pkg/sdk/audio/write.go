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

package audio

import (
	"fmt"
	"io"
	"math"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// Write puts samples on disk as a WAV somebody can play.
//
// The one thing this project did not keep. Every reading the measuring loop
// takes is reduced to ten figures and the audio is dropped, which is the right
// default: a pass is a few seconds of bass and a campaign is hundreds of them.
//
// What it costs to never keep any of it is that the figures are the only account
// of what happened, and ten numbers can all sit inside tolerance while the sound
// is plainly wrong. `Squealing` exists because exactly that happened: the chain
// was feeding itself, every figure read fine, and a whole library of
// measurements was of the squeal. A reading somebody can play is the backstop
// for the failure that does not have a name yet.
//
// Mono at the loop's own rate, 24-bit, because that is what came back off the
// interface and converting it would put this function between the measurement
// and the ear.
func Write(
	at string,
	samples []float32,
	rate int,
) error {
	f, err := creates(at)
	if err != nil {
		return fmt.Errorf("creating %s: %w", at, err)
	}

	if err := WriteTo(f, samples, rate); err != nil {
		_ = closes(f)

		return fmt.Errorf("writing %s: %w", at, err)
	}

	if err := closes(f); err != nil {
		return fmt.Errorf("closing %s: %w", at, err)
	}

	return nil
}

// creates opens the file a kept reading goes into, so a test stands its own
// function here.
//
// A variable because it is the only thing in this file that touches a
// filesystem, and the three branches around it — a path that will not open, a
// file that will not take the samples, a file that will not close — are branches
// a real *os.File on a temp directory cannot be made to reach.
var creates = func(
	at string,
) (io.WriteSeeker, error) {
	f, err := os.Create(at) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}

	return f, nil
}

// closes shuts the file, separate from creates because the close is its own
// failure and its own branch.
var closes = func(
	w io.WriteSeeker,
) error {
	f, ok := w.(io.Closer)
	if !ok {
		return nil
	}

	return f.Close()
}

// WriteTo writes samples as a WAV into w.
//
// Separate from Write because a file that cannot be written to is most of what
// can go wrong here and an *os.File will not do it on demand. The encoder's own
// failures are reachable from a test this way, and the one that matters is the
// close: it is what patches the header's lengths, so a file whose encoder was
// not closed is a WAV nothing will play.
func WriteTo(
	w io.WriteSeeker,
	samples []float32,
	rate int,
) error {
	enc := wav.NewEncoder(w, rate, bitDepth, 1, wavFormat)

	buf := &audio.IntBuffer{
		Format:         &audio.Format{NumChannels: 1, SampleRate: rate},
		SourceBitDepth: bitDepth,
		Data:           make([]int, len(samples)),
	}

	// Clamped, because the wrap is silent. A sample at full scale times 2^23 is
	// one past what a signed 24-bit integer holds and comes back as its own
	// negative, so a tone at the top of the range reads back inverted with
	// nothing reporting it. A hot interface hands over samples above 1.0 too.
	for i, v := range samples {
		at := float64(v) * fullScale
		buf.Data[i] = int(math.Max(-fullScale, math.Min(fullScale, at)))
	}

	if err := enc.Write(buf); err != nil {
		return fmt.Errorf("putting the samples in: %w", err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("finishing the header: %w", err)
	}

	return nil
}

// bitDepth is what a kept reading is written at, and fullScale the largest
// value that depth holds.
//
// Twenty-four, matching what the interfaces here hand over, so a kept file is
// the samples that were measured rather than a rounding of them.
const (
	bitDepth = 24
	// fullScale is the largest magnitude a signed 24-bit sample holds, which is
	// one less than 2^23 rather than 2^23.
	fullScale = 1<<23 - 1
	wavFormat = 1
)
