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

// Package reamp plays a recording through hardware and keeps what comes back.
//
// Re-amping is the studio term for exactly this: a signal that was recorded
// once is played out through gear and recorded again, so the difference
// between the two is the gear and nothing else. It is what makes a
// measurement of a pedal possible at all, because a record is the far end of
// a whole signal chain and cannot say what any one control did.
//
// Playback and capture share one clock. Two separate streams drift, and a
// drifting pair would show up as a measurement that changed slowly over an
// evening for no reason anybody could find.
package reamp

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gen2brain/malgo"
)

// Rate is the only sample rate an HX Stomp runs at without Line 6's own
// driver installed, so everything here is fixed to it rather than negotiated.
const Rate = 48000

// Silence before and after the signal.
//
// The front is because latency is not known in advance: the first samples
// back are whatever the converters had in hand. The back is because a reverb
// goes on ringing after the input stops, and cutting at the end of the signal
// would clip the tail off the thing being measured.
const (
	Lead = 300 * time.Millisecond
	Tail = time.Second
)

// Margin is how much longer than the signal the recording runs for.
//
// A converter answers late. Capturing exactly as many samples as were played
// therefore stops short by however long the round trip takes, and what falls
// off the end is the tail: the part a decay is measured from. Measured
// against a Python implementation that aligns differently, the same bypassed
// loop read 0.76 seconds of decay one way and 1.19 the other, and the
// difference was entirely this.
const Margin = 500 * time.Millisecond

// Bench is an open connection to a piece of audio hardware.
//
// Held open across many readings rather than opened per reading. Opening a
// duplex stream costs the best part of a second, and a campaign takes one
// reading per block across six hundred blocks.
type Bench struct {
	ctx  *malgo.AllocatedContext
	play malgo.DeviceID
	rec  malgo.DeviceID
	// claiming is how long the device is given to hand itself over. Zero is
	// Claiming, and a test names a shorter one.
	claiming time.Duration
	// stuck is whether a claim was abandoned while still inside CoreAudio.
	//
	// The goroutine holding it cannot be cancelled: it is blocked in a C call
	// that returns when the system stops waiting, and it reads this Bench's
	// audio context when it does. So Close leaves that context alone rather
	// than freeing memory something is still going to touch, and the process
	// exit reclaims it. Freeing it was a segfault.
	stuck atomic.Bool
	// width is how many channels to open, and 0 means the device's own
	// count.
	//
	// Asking for two on a device that presents eight is refused by CoreAudio
	// with nothing to say but "invalid argument", and an HX Stomp presents
	// eight. Naming a number here would only be right for one interface, so
	// the device decides and the loop reads the width back off the buffer.
	width uint32
	name  string

	// One reading at a time. The callback writes into buffers this owns, and
	// two readings at once would interleave into each other's.
	mu sync.Mutex
}

// Open finds a device by name and gets ready to push signal through it.
//
// The name is matched loosely and case-insensitively, so "hx stomp" finds
// what CoreAudio calls "HX Stomp". Both directions have to be the same piece
// of hardware: playing into one device and recording from another is two
// clocks, and the drift between them is not measurable after the fact.
func Open(
	want string,
) (*Bench, error) {
	return open(nil, want)
}

// open is Open with the audio backends named.
//
// Nil is every backend the platform has, which is what a person wants. A test
// passes the null one: it presents a device that takes samples and hands back
// silence, so the loop's own arithmetic can be checked without an audio
// interface, a cable and somebody in the room to plug them in.
func open(
	backends []malgo.Backend,
	want string,
) (*Bench, error) {
	ctx, err := malgo.InitContext(backends, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("starting audio: %w", err)
	}

	b := &Bench{ctx: ctx}

	play, name, err := find(ctx, malgo.Playback, want)
	if err != nil {
		b.Close()

		return nil, err
	}

	rec, _, err := find(ctx, malgo.Capture, want)
	if err != nil {
		b.Close()

		return nil, err
	}

	b.play, b.rec, b.name = play, rec, name

	if b.claiming == 0 {
		b.claiming = Claiming
	}

	return b, nil
}

// Name is what the hardware calls itself.
func (b *Bench) Name() string { return b.name }

// find picks one device of a kind, or says what was attached instead.
func find(
	ctx *malgo.AllocatedContext,
	kind malgo.DeviceType,
	want string,
) (malgo.DeviceID, string, error) {
	found, err := ctx.Devices(kind)
	if err != nil {
		return malgo.DeviceID{}, "", fmt.Errorf("listing audio devices: %w", err)
	}

	had := make([]string, 0, len(found))

	for _, d := range found {
		name := d.Name()
		had = append(had, name)

		if strings.Contains(strings.ToLower(name), strings.ToLower(want)) {
			return d.ID, name, nil
		}
	}

	return malgo.DeviceID{}, "", &NoDeviceError{
		Want: want, Direction: direction(kind), Had: had,
	}
}

// direction names a device kind the way somebody would say it.
func direction(
	kind malgo.DeviceType,
) string {
	if kind == malgo.Capture {
		return "input"
	}

	return "output"
}

// Through plays a signal and returns what came back.
//
// The signal goes out on both channels and the left one comes back, which is
// what the loop is: a cable from the device's output to its own input, and a
// chain in between.
//
// Padded with silence either side, and the padding travels in the answer
// rather than being trimmed. A reverb's tail is part of what that block does,
// and a measurement that cut it would report every reverb as a short one.
func (b *Bench) Through(
	ctx context.Context,
	signal []float32,
) ([]float32, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := pad(signal)
	run := &pass{
		out: out,
		got: make([]float32, 0, len(out)+int(Margin.Seconds()*Rate)),
	}

	var (
		done = make(chan struct{})
		once sync.Once
	)

	finish := func() { once.Do(func() { close(done) }) }

	config := malgo.DefaultDeviceConfig(malgo.Duplex)
	config.SampleRate = Rate
	config.Playback.DeviceID = b.play.Pointer()
	config.Playback.Format = malgo.FormatF32
	config.Playback.Channels = b.width
	config.Capture.DeviceID = b.rec.Pointer()
	config.Capture.Format = malgo.FormatF32
	config.Capture.Channels = b.width

	device, err := b.claim(b.claiming, config, func(output, input []byte, frames uint32) {
		if run.frame(output, input, int(frames)) {
			finish()
		}
	})
	if err != nil {
		return nil, err
	}

	defer device.Uninit()
	defer func() { _ = device.Stop() }()

	// A budget rather than a wait. A device that stops delivering callbacks
	// leaves this blocked forever otherwise, and a campaign that hangs on
	// block two hundred looks exactly like one still working.
	budget, cancel := context.WithTimeout(ctx, over(len(out)))
	defer cancel()

	select {
	case <-done:
		return run.got, nil
	case <-budget.Done():
		return nil, fmt.Errorf(
			"%s stopped answering after %d of %d samples",
			b.name, len(run.got), cap(run.got))
	}
}

// pass is one reading in progress: what is left to play, and what has come
// back so far.
//
// Separate from the device because everything it does is arithmetic on two
// buffers, and the callback it runs inside cannot be reached from a test:
// miniaudio ships a null backend that would allow it and the bindings here
// compile it out.
type pass struct {
	out  []float32
	got  []float32
	sent int
}

// frame is one turn of the loop, and reports whether the reading is finished.
//
// Capture happens first. What arrives in a callback is the answer to what the
// previous one played, and keeping it before the buffer is refilled is what
// holds the two in step.
//
// The signal goes out on both channels, which is what the loop is: a cable
// from the device's output back to its own input.
func (p *pass) frame(
	output, input []byte,
	frames int,
) bool {
	for i := range frames {
		if len(p.got) < cap(p.got) {
			p.got = append(p.got, sample(input, i*2))
		}
	}

	for i := range frames {
		var s float32
		if p.sent < len(p.out) {
			s = p.out[p.sent]
			p.sent++
		}

		put(output, i*2, s)
		put(output, i*2+1, s)
	}

	return p.sent >= len(p.out) && len(p.got) >= cap(p.got)
}

// Claiming is how long opening a device may take before something is wrong.
//
// Generous, because a real interface takes a moment to hand itself over, and
// bounded, because the thing that goes wrong here does not fail: on macOS the
// first capture by a program with no Microphone permission blocks in CoreAudio
// while the system waits for somebody to answer a dialog. Nobody answers a
// dialog raised behind a terminal running unattended, so without a deadline a
// campaign sits there all night having measured nothing.
const Claiming = 20 * time.Second

// claim opens a device and starts it, or says what to go and do about it.
//
// Both calls run on their own goroutine because neither takes a context and
// either can block indefinitely. A goroutine left behind is the price of
// answering at all: it is holding a CoreAudio call that will return when the
// system stops waiting, and the process is on its way out by then anyway.
// The caller's context is deliberately not consulted. A caller who gave up is
// not a device that went quiet, and the reading's own budget already reports
// cancellation: looking at it here would turn a Ctrl-C into advice about
// granting a permission.
func (b *Bench) claim(
	within time.Duration,
	config malgo.DeviceConfig,
	data func(output, input []byte, frames uint32),
) (*malgo.Device, error) {
	type opened struct {
		device *malgo.Device
		err    error
	}

	out := make(chan opened, 1)

	// Read before the goroutine starts, because Close may set b.ctx to nil
	// while the call below is still inside CoreAudio.
	audio := b.ctx.Context

	go func() {
		device, err := malgo.InitDevice(
			audio, config, malgo.DeviceCallbacks{Data: data})
		if err != nil {
			out <- opened{err: fmt.Errorf("opening %s: %w", b.name, err)}

			return
		}

		if err := device.Start(); err != nil {
			device.Uninit()
			out <- opened{err: fmt.Errorf("starting %s: %w", b.name, err)}

			return
		}

		out <- opened{device: device}
	}()

	waiting := time.NewTimer(within)
	defer waiting.Stop()

	select {
	case got := <-out:
		return got.device, got.err
	case <-waiting.C:
		b.stuck.Store(true)

		return nil, &UnclaimedError{Name: b.name, After: within}
	}
}

// over is how long a reading may take before something is wrong.
//
// Twice the signal and a few seconds, which is loose on purpose: a device
// that is merely slow should be waited for, and one that has stopped should
// not be waited for all evening.
func over(
	samples int,
) time.Duration {
	return 2*time.Duration(samples)*time.Second/Rate + 5*time.Second
}

// pad puts silence either side of a signal.
func pad(
	signal []float32,
) []float32 {
	lead := int(Lead.Seconds() * Rate)
	tail := int(Tail.Seconds() * Rate)

	out := make([]float32, lead+len(signal)+tail)
	copy(out[lead:], signal)

	return out
}

// sample reads one float32 out of a frame buffer.
func sample(
	buf []byte,
	at int,
) float32 {
	if (at+1)*4 > len(buf) {
		return 0
	}

	return math.Float32frombits(
		binary.LittleEndian.Uint32(buf[at*4 : at*4+4]))
}

// put writes one float32 into a frame buffer.
func put(
	buf []byte,
	at int,
	v float32,
) {
	if (at+1)*4 > len(buf) {
		return
	}

	binary.LittleEndian.PutUint32(
		buf[at*4:at*4+4], math.Float32bits(v))
}

// Close lets the hardware go.
func (b *Bench) Close() error {
	if b.ctx == nil {
		return nil
	}

	// A claim still inside CoreAudio reads this context when it returns, and
	// nothing here can tell it not to. Dropping the reference without freeing
	// it costs one context for the life of the process; freeing it was a
	// segfault in the goroutine.
	if b.stuck.Load() {
		b.ctx = nil

		return nil
	}

	err := b.ctx.Uninit()
	b.ctx.Free()
	b.ctx = nil

	if err != nil {
		return fmt.Errorf("stopping audio: %w", err)
	}

	return nil
}
