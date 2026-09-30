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
// Playback and capture are one device by default, so they share one clock. Two
// separate streams drift, and a drifting pair shows up as a measurement that
// changes slowly over an evening for no reason anybody can find.
//
// Two devices are allowed anyway, because on an HX Stomp one device is what
// closes the measuring loop. The chain can only be reached through the physical
// input jack, and the only way to get the computer's signal to that jack is a
// cable off the pedal's own output, which carries the chain's output with it. So
// the chain hears itself, and 23 of 661 blocks have enough gain to keep that
// going past any trim. Playing through the computer's own output instead opens
// the loop rather than holding it below unity: that cable carries nothing but
// what the computer plays.
//
// The drift is the price and it is worth naming. It is harmless for what this
// measures, which is where energy sits and how loud it is, and it is not
// harmless for anything deconvolving an impulse response.
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

// Rate is the sample rate every committed figure was taken at.
//
// Not an assumption about the device. An HX Stomp presents itself
// class-compliant at 48kHz with nothing installed, which is why this has never
// bitten, but Line 6's own driver and any other interface --hardware names may
// run at something else. So the loop asks the device for this rate and then
// checks what the backend actually gave it, rather than setting it and
// believing it.
//
// What makes it fixed rather than negotiated is the data, not the hardware:
// resources/dry/bass-di.wav is 48kHz and so is every reading in
// resources/sweeps/. A figure taken at another rate cannot be filed beside
// those, so the loop refuses instead.
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
// The name is matched loosely and case-insensitively, so "hx stomp" finds what
// CoreAudio calls "HX Stomp".
//
// **Two names separated by a comma play through the first and record from the
// second**, as in "MacBook Pro Speakers,HX Stomp". That is how the measuring
// loop is opened rather than quieted: see the package comment. One name means
// one device both ways, which is the default and the only arrangement that
// shares a clock.
func Open(
	want string,
) (*Bench, error) {
	return open(nil, want)
}

// sides splits what a caller asked for into the device to play through and the
// device to record from.
//
// One name is both. Two are taken in the order the signal travels, out of the
// first and back into the second, which is the order somebody describes a rig
// in. Surrounding spaces go, so "speakers, stomp" works.
func sides(
	want string,
) (string, string) {
	play, rec, split := strings.Cut(want, ",")
	if !split {
		return want, want
	}

	return strings.TrimSpace(play), strings.TrimSpace(rec)
}

// TwoSided reports a name that asks for one device to play through and another
// to record from.
//
// Off sides, so the syntax has one home. pkg/cli decides whether to apply the
// headroom trim from this, and it had answered by looking for a comma itself:
// two readings of one rule, and the one that only sniffed for the character
// would have disagreed the moment this took a second separator or refused an
// empty half.
//
// What it means is a rig where the pedal's output reaches nothing, because the
// cable into the pedal carries only what the computer plays. That is the whole
// reason the measuring loop can be open.
func TwoSided(
	want string,
) bool {
	play, rec := sides(want)

	return play != rec
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

	wantPlay, wantRec := sides(want)

	play, playing, err := find(ctx, malgo.Playback, wantPlay)
	if err != nil {
		b.Close()

		return nil, err
	}

	rec, recording, err := find(ctx, malgo.Capture, wantRec)
	if err != nil {
		b.Close()

		return nil, err
	}

	b.play, b.rec = play, rec

	// Named as the signal travels when the two differ, because an error saying
	// one device's name while the other is the broken one sends somebody to the
	// wrong end of the rig.
	b.name = playing
	if playing != recording {
		b.name = playing + " into " + recording
	}

	if b.claiming == 0 {
		b.claiming = Claiming
	}

	return b, nil
}

// Name is what the hardware calls itself.
func (b *Bench) Name() string { return b.name }

// rated refuses a loop the backend is resampling.
//
// Both directions, because they are negotiated separately and a reading is only
// as comparable as the worse of the two.
//
// Takes the two rates rather than the device, so the decision can be checked
// without an audio interface: reading them off a device is the caller's job and
// deciding what they mean is this one's.
func (b *Bench) rated(
	in, out uint32,
) error {
	if in == Rate && out == Rate {
		return nil
	}

	return &RateError{Name: b.name, Want: Rate, Capture: in, Playback: out}
}

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

		if matches(name, want) {
			return d.ID, name, nil
		}
	}

	return malgo.DeviceID{}, "", &NoDeviceError{
		Want: want, Direction: direction(kind), Had: had,
	}
}

// matches reports a device the caller meant.
//
// A named device is a substring, case folded, because somebody types "stomp"
// rather than the full name CoreAudio gives.
//
// Naming nothing means the pedal, not the first device the platform happens to
// enumerate. That distinction is the whole of this function: strings.Contains
// on an empty string is true of everything, so an unnamed device used to take
// whichever one came back first. On this machine that was the HX Stomp all day
// and then a pair of Bluetooth headphones, and the loop reported a punk chain
// measuring -80.3dB through "John's AirPods Max" rather than refusing.
//
// Silence is the worst thing this could have returned. It is a reading, so
// every figure downstream is a number rather than an error, and a measurement
// tool that quietly measures the wrong device is worse than one that stops.
func matches(
	name string,
	want string,
) bool {
	got := strings.ToLower(name)

	if want != "" {
		return strings.Contains(got, strings.ToLower(want))
	}

	return helix(got)
}

// helix reports an audio device belonging to the family this loop measures,
// by the name its own driver presents: "HX Stomp", "Helix", "HX Effects".
//
// A prefix rather than a substring, so a mixer with "HX" somewhere in its name
// is not mistaken for the pedal. Which models exist is the catalog's business
// and not repeated here; this only has to tell the pedal apart from the
// speakers, and --hardware names one outright when it cannot.
func helix(
	got string,
) bool {
	return strings.HasPrefix(got, "helix") || strings.HasPrefix(got, "hx ")
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

	// Asked for above and checked here, because the two are different
	// questions. config.SampleRate is a request, and miniaudio answers a
	// device that cannot meet it by resampling rather than by refusing: the
	// internal rates are what the backend actually negotiated, and they differ
	// from the requested one exactly when a converter is in the path.
	if err := b.rated(
		device.CaptureInternalSampleRate(),
		device.PlaybackInternalSampleRate(),
	); err != nil {
		return nil, err
	}

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
// The signal goes out on the first channel only. A mono cable takes that one,
// and writing a copy per channel would put this frame's second sample into the
// next frame on a device presenting one.
func (p *pass) frame(
	output, input []byte,
	frames int,
) bool {
	// The stride is the device's, not this loop's. A device is opened at its
	// own channel count and an HX Stomp presents eight, so frame i starts at
	// slot i*channels and a loop striding by two walks across one frame's
	// channels instead of down successive frames. That played a quarter of the
	// signal scattered across the wrong channels, which measures as something
	// quiet and bright: the empty loop read 11,990Hz at -46dB where it should
	// read 95Hz at -21dB.
	out := channelsIn(output, frames)
	in := channelsIn(input, frames)

	for i := range frames {
		if len(p.got) < cap(p.got) {
			p.got = append(p.got, sample(input, i*in))
		}
	}

	for i := range frames {
		var s float32
		if p.sent < len(p.out) {
			s = p.out[p.sent]
			p.sent++
		}

		// The first pair, where the Main out listens, and silence on whatever
		// else the device presents rather than a copy per channel. A mono
		// interface has no pair, and writing one would put this frame's second
		// sample into the next frame.
		put(output, i*out, s)

		if out > 1 {
			put(output, i*out+1, s)
		}
	}

	return p.sent >= len(p.out) && len(p.got) >= cap(p.got)
}

// channelsIn is how many channels a frame buffer holds, read off the buffer.
//
// The callback is handed a buffer the device sized, so this is the one place
// that knows the width, and asking it beats recording a number that is right
// for one interface and wrong for the next. An HX Stomp presents eight, a
// Floor presents more, and an ordinary interface presents two.
//
// One is a real answer and not an error: a mono device has no pair to write,
// which the caller handles. Anything that cannot be divided at all gets one,
// because a stride of zero would write every frame on top of the first.
func channelsIn(
	buf []byte,
	frames int,
) int {
	if frames <= 0 {
		return 1
	}

	got := len(buf) / (4 * frames)
	if got < 1 {
		return 1
	}

	return got
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
