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

package cab

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

// Made is where an impulse response came from.
//
// It travels with the file because the tooling cannot tell the three apart
// and the difference decides whether it may be sold.
//
// One captured from a cabinet somebody owns is theirs, and is what the people
// selling impulse responses are selling. One derived by measuring somebody
// else's commercial impulse response is a copy with extra steps, whatever the
// method: measuring a competitor's product and reproducing its response is
// not made acceptable by the reproduction being approximate. One derived from
// a record carries somebody's recording in it, and whether a filter fitted to
// a master is a derivative work is not a question to find out by selling it.
type Made struct {
	// How it was made: captured from hardware, or matched to a target.
	How Method
	// From is what it was made from, named so a later reader can tell which
	// of the three this is.
	From string
	// Through is the chain the measurement ran down, where there was one.
	Through string
	// When it was made.
	When time.Time
	// Taps is how long it is.
	Taps int
}

// Method is how an impulse response was arrived at.
type Method string

// The two ways to get one.
const (
	// Captured is deconvolved from a cabinet somebody put a signal through.
	Captured Method = "captured"
	// Matched is fitted to a target nobody had: a recording, a measurement,
	// or another cabinet.
	Matched Method = "matched"
)

// Write puts an impulse response where a device can load it.
//
// A 24-bit WAV at the device's own rate. Twenty four rather than sixteen
// because an impulse response is mostly very quiet: the tail is where a
// cabinet's character is, and sixteen bits puts the quantisation noise floor
// right through it.
func Write(
	w io.Writer,
	of []float64,
	made Made,
) error {
	if len(of) != Short && len(of) != Long {
		return &BadLengthError{Taps: len(of)}
	}

	const (
		channels = 1
		bits     = 24
		bytes    = bits / 8
	)

	body := make([]byte, 0, len(of)*bytes)

	for _, v := range of {
		body = append(body, sample24(v)...)
	}

	// A LIST/INFO chunk, so where the file came from travels inside it
	// rather than in a note beside it that gets lost the first time somebody
	// shares the file on its own.
	info := chunk(made)

	header := []any{
		[4]byte{'R', 'I', 'F', 'F'},
		uint32(4 + 24 + len(info) + 8 + len(body)), //nolint:gosec // bounded
		[4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '},
		uint32(16),
		uint16(1), // uncompressed
		uint16(channels),
		uint32(Rate),
		uint32(Rate * channels * bytes),
		uint16(channels * bytes),
		uint16(bits),
	}

	for _, part := range header {
		if err := binary.Write(w, binary.LittleEndian, part); err != nil {
			return fmt.Errorf("writing the impulse response: %w", err)
		}
	}

	if _, err := w.Write(info); err != nil {
		return fmt.Errorf("writing the impulse response: %w", err)
	}

	if err := binary.Write(w, binary.LittleEndian,
		[4]byte{'d', 'a', 't', 'a'}); err != nil {
		return fmt.Errorf("writing the impulse response: %w", err)
	}

	if err := binary.Write(w, binary.LittleEndian,
		uint32(len(body))); err != nil { //nolint:gosec // bounded by taps
		return fmt.Errorf("writing the impulse response: %w", err)
	}

	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("writing the impulse response: %w", err)
	}

	return nil
}

// chunk is the provenance, as a WAV metadata chunk.
func chunk(
	made Made,
) []byte {
	comment := fmt.Sprintf(
		"%s from %s on %s",
		made.How, made.From, made.When.UTC().Format(time.RFC3339))

	if made.Through != "" {
		comment += ", through " + made.Through
	}

	fields := [][2]string{
		{"ISFT", "tonestack"},
		{"ICMT", comment},
	}

	var body []byte

	body = append(body, 'I', 'N', 'F', 'O')

	for _, f := range fields {
		text := append([]byte(f[1]), 0)
		if len(text)%2 == 1 {
			text = append(text, 0)
		}

		body = append(body, f[0][0], f[0][1], f[0][2], f[0][3])
		body = binary.LittleEndian.AppendUint32(body, uint32(len(text))) //nolint:gosec // bounded
		body = append(body, text...)
	}

	out := []byte{'L', 'I', 'S', 'T'}
	out = binary.LittleEndian.AppendUint32(out, uint32(len(body))) //nolint:gosec // bounded

	return append(out, body...)
}

// sample24 is one sample as three bytes, little endian and signed.
func sample24(
	v float64,
) []byte {
	// Clamped rather than wrapped. A sample past full scale that wrapped
	// would come back as a loud sample of the opposite sign, which is a
	// click rather than a clip.
	const most = 1<<23 - 1

	at := int32(math.Round(math.Max(-1, math.Min(1, v)) * most))

	return []byte{byte(at), byte(at >> 8), byte(at >> 16)}
}
