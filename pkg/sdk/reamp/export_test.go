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

package reamp

import (
	"time"

	"github.com/gen2brain/malgo"
)

// The parts of the loop that are arithmetic rather than hardware, exposed so
// they can be held to a number without an audio interface, a cable and
// somebody in the room to plug them in.
//
// What is left needing hardware is opening a device and running its callback,
// which is the part that genuinely cannot be checked from a test.
var (
	// Pad puts silence either side of a signal.
	Pad = pad
	// Sample reads one float32 out of a frame buffer, and Put writes one.
	Sample = sample
	Put    = put
	// Over is how long a reading may take before something is wrong.
	Over = over
	// Direction names a device kind the way somebody would say it.
	Direction = direction
	// Matches reports a device the caller meant, named or not.
	Matches = matches

	// ChannelsIn works out a frame's stride from the buffer the callback was
	// handed, because how many channels a device presents varies with the
	// model: a Stomp is 8 in and 8 out and nothing here may assume it.
	ChannelsIn = channelsIn
	// OpenWith is Open with the audio backends named.
	OpenWith = open
)

// OpenClaiming is OpenWith, with the device given less time to hand itself
// over.
//
// The deadline exists for a device that neither opens nor refuses, which on
// macOS is a program with no Microphone permission waiting on a dialog. No
// backend fakes that, so the only way to reach the branch is to make the wait
// short enough that a device which does open still misses it.
func OpenClaiming(
	backends []malgo.Backend,
	want string,
	within time.Duration,
) (*Bench, error) {
	b, err := open(backends, want)
	if err != nil {
		return nil, err
	}

	b.claiming = within

	return b, nil
}

// NullBackend is miniaudio's null backend, which presents a device that takes
// samples and hands back silence.
//
// Not malgo.BackendNull, which does not work. miniaudio's C enum ends
// ...webaudio, custom, null and malgo's Go enum omits custom, so every name
// from there on is one short: malgo.BackendNull is 13 where ma_backend_null
// is 14, and asking for it gets ma_backend_custom, which nothing configured
// and which answers "No backend".
//
// The number rather than the name, because the name is wrong upstream.
const NullBackend = malgo.Backend(14)

// Pass is one reading in progress, exposed because the callback it runs
// inside cannot be reached from a test and everything it does is arithmetic.
type Pass = pass

// NewPass starts a reading of a signal, with the margin a converter answers
// late by.
func NewPass(
	signal []float32,
) *Pass {
	out := pad(signal)

	return &pass{
		out: out,
		got: make([]float32, 0, len(out)+int(Margin.Seconds()*Rate)),
	}
}

// Frame is one turn of the loop.
func (p *Pass) Frame(
	output, input []byte,
	frames int,
) bool {
	return p.frame(output, input, frames)
}

// Got is what has come back so far.
func (p *Pass) Got() []float32 { return p.got }

// Rated reports whether the rates a backend negotiated are the ones every
// committed figure was taken at.
func (b *Bench) Rated(
	in, out uint32,
) error {
	return b.rated(in, out)
}
