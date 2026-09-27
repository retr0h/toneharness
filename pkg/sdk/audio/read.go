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
	"errors"
	"fmt"
	"io"
	"math"

	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// Reading a file into the samples a measurement works on.
//
// One format, WAV, because it is the one a decoder can be trusted with in a
// few lines. Anything else is converted on the way in by whoever has ffmpeg,
// which is the same bargain the gear map strikes with Python: shell out for
// the one job Go should not be doing, and keep the part that matters here.

// ErrNotAudio reports a file that is not audio this can read.
var ErrNotAudio = errors.New("not readable audio")

// NotAudioError says what was wrong with it.
type NotAudioError struct {
	// Why names the thing that was wrong.
	Why string
}

// Error implements the error interface.
func (e *NotAudioError) Error() string {
	return fmt.Sprintf("not readable audio: %s", e.Why)
}

// Unwrap returns ErrNotAudio so callers can match with errors.Is.
func (*NotAudioError) Unwrap() error { return ErrNotAudio }

// Read decodes a WAV into single-channel samples between -1 and 1.
//
// Several channels are averaged into one. A measurement describes a sound
// rather than a stereo image, and a bass part that sits in the middle of a mix
// measures the same either way.
//
// The reader must also seek, which a file does and a pipe does not. Line 6's
// own format needs the same, for the same reason.
//
// Decoded in chunks through a buffer this reuses, rather than in one call to
// the decoder's FullPCMBuffer. That is a 51x difference and not a micro
// optimisation: FullPCMBuffer grows one slice from nothing for the whole file,
// and on the 215-second reference recording it took 4.54 seconds against 88
// milliseconds here, while the averaging below takes 15. Every figure a
// measurement reports is identical either way, which read_public_test.go pins
// against the decoder rather than leaving as an assumption.
//
// It cost most of the test suite's wall clock. One translate case that resolves
// the same ask six times spent 165 seconds of its 353 doing this.
func Read(
	r io.ReadSeeker,
) ([]float64, int, error) {
	dec := wav.NewDecoder(r)

	// Reads the header and leaves the reader at the samples. A file that is not
	// a WAV fails here, which is where FullPCMBuffer used to fail.
	if err := dec.FwdToPCM(); err != nil {
		return nil, 0, &NotAudioError{Why: err.Error()}
	}

	rate := int(dec.SampleRate)
	channels := int(dec.NumChans)

	// A header naming no channels or no width used to be refused by the
	// decoder, as "format not supported" and "unhandled byte depth". Reading
	// the header ourselves means refusing them ourselves, and a nil check is
	// not enough: zero channels would divide by zero below rather than fail.
	if channels < 1 {
		return nil, 0, &NotAudioError{Why: "the header names no channels"}
	}

	if dec.BitDepth < 1 {
		return nil, 0, &NotAudioError{Why: "the header names no bit depth"}
	}

	// What divides a whole sample down to something between -1 and 1. A
	// decoder hands back whatever width the file was written at, and a
	// measurement that skipped this would report a 24-bit recording as
	// hundreds of times louder than the same take at 16.
	full := math.Pow(2, float64(dec.BitDepth-1))

	// One buffer, reused for every chunk. Its size is a trade between syscalls
	// and memory and nothing depends on the value: the samples that come out
	// are the same at 1,000 as at 65,536, which the test checks.
	chunk := &goaudio.IntBuffer{
		Format: &goaudio.Format{NumChannels: channels, SampleRate: rate},
		Data:   make([]int, chunkFrames*channels),
	}

	out := make([]float64, 0, chunkFrames)

	for {
		n, err := dec.PCMBuffer(chunk)

		for at := 0; at+channels <= n; at += channels {
			var sum float64
			for c := range channels {
				sum += float64(chunk.Data[at+c]) / full
			}

			out = append(out, sum/float64(channels))
		}

		// After the samples, not before them: the call that returns the last of
		// them can return the end of the file with them.
		if err != nil || n == 0 {
			break
		}
	}

	return out, rate, nil
}

// chunkFrames is how many frames a read asks the decoder for at a time.
const chunkFrames = 64 * 1024
