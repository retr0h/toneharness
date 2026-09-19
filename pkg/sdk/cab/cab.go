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

// Package cab builds a cabinet the device does not have.
//
// An impulse response is a short recording of what a speaker does to a click,
// and a device that loads one is playing a filter. That makes a cabinet a
// file: a real, loadable, shareable artefact, which is the half of "build gear
// the device is missing" that is actually possible. An amplifier is not,
// because Line 6 publish no format for a user-made one.
//
// Two jobs, and they are not the same.
//
// Capture is for a cabinet somebody has. A known signal goes through it, the
// answer is recorded, and dividing one by the other in the frequency domain
// recovers what the cabinet did. That is the standard method and it needs the
// physical cabinet.
//
// Match is for a cabinet somebody does not have: a recording, a measurement,
// or another cabinet theirs should resemble. There is nothing to deconvolve
// against, so instead the target and what is in hand are both measured and
// the difference between them becomes the filter.
//
// What neither can do is in Limits, and saying so matters more than the
// method: a matched cabinet sold as the real thing is a lie somebody paid
// for.
package cab

import (
	"math"
	"math/cmplx"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// Taps is how long an impulse response a device will load.
//
// An HX Stomp takes 1024 or 2048 samples. Longer is not better here: the
// extra length buys resolution at the bottom of the spectrum, and a cabinet
// has little happening there that a bass player would miss.
const (
	Short = 1024
	Long  = 2048
)

// Rate is the sample rate an impulse response is written at.
//
// The device's own, because a file at another rate is resampled on the way in
// by something nobody here wrote and cannot measure.
const Rate = 48000

// Capture recovers what a cabinet did to a signal.
//
// Deconvolution: what came back, divided by what went in, bin by bin. The
// quotient is the cabinet's own response, because everything else in the path
// was in both.
//
// Regularised, because division by a bin holding nothing is division by
// nothing. A sweep has energy everywhere by design and a recording of music
// does not, so the quiet bins of a real signal would otherwise come back as
// enormous numbers that are entirely noise.
func Capture(
	sent, back []float64,
	taps int,
) ([]float64, error) {
	if err := usable(taps); err != nil {
		return nil, err
	}

	if len(sent) < taps || len(back) < taps {
		return nil, &TooShortError{
			Sent: len(sent), Back: len(back), Taps: taps,
			What: [2]string{"what went out", "what came back"},
		}
	}

	n := next(max(len(sent), len(back)))

	out := spectrum(sent, n)
	in := spectrum(back, n)

	// The largest bin sets the floor, so the guard scales with the signal
	// rather than with whatever units it happens to be in.
	var loudest float64
	for _, v := range out {
		loudest = math.Max(loudest, cmplx.Abs(v))
	}

	if loudest == 0 {
		return nil, &SilenceError{Side: "what went in"}
	}

	floor := loudest * 1e-6

	got := make([]complex128, n)

	for i := range got {
		if cmplx.Abs(out[i]) < floor {
			continue
		}

		got[i] = in[i] / out[i]
	}

	return trim(samples(got), taps), nil
}

// Match builds the filter that turns one response into another.
//
// The ratio between two magnitude responses is itself a filter, so a cabinet
// somebody has can be made to measure like one they do not: measure the
// target, measure what is in hand, and keep the difference.
//
// Minimum phase, because a magnitude response says nothing about phase and
// something has to be chosen. A speaker is roughly minimum phase, so for a
// cabinet the choice is close; for anything with a reflection in it, a room
// or a distant second microphone, it is not, and the difference is audible as
// the sense of space rather than as tone. See Limits.
func Match(
	target, have []float64,
	taps int,
) ([]float64, error) {
	if err := usable(taps); err != nil {
		return nil, err
	}

	if len(target) < taps || len(have) < taps {
		return nil, &TooShortError{
			Sent: len(target), Back: len(have), Taps: taps,
			What: [2]string{"target", "what is in hand"},
		}
	}

	n := next(max(len(target), len(have)))

	want := magnitudes(spectrum(target, n))
	got := magnitudes(spectrum(have, n))

	var loudest float64
	for _, v := range got {
		loudest = math.Max(loudest, v)
	}

	if loudest == 0 {
		return nil, &SilenceError{Side: "what is in hand"}
	}

	floor := loudest * 1e-6
	ratio := make([]float64, n)

	for i := range ratio {
		if got[i] < floor {
			ratio[i] = 1

			continue
		}

		ratio[i] = want[i] / got[i]
	}

	return trim(minimumPhase(ratio), taps), nil
}

// minimumPhase builds the shortest impulse response with a given magnitude.
//
// Through the cepstrum: the log of the magnitude, transformed back, folded so
// everything before time zero is added to what is after it, and transformed
// forwards again. What comes out has the magnitude it was given and all of
// its energy as early as that magnitude allows.
func minimumPhase(
	magnitude []float64,
) []float64 {
	n := len(magnitude)
	re := make([]float64, n)
	im := make([]float64, n)

	// A log needs somewhere to stand. A bin holding nothing has a magnitude
	// of zero and a logarithm of minus infinity, which nothing downstream
	// survives.
	for i, v := range magnitude {
		re[i] = math.Log(math.Max(v, 1e-12))
	}

	audio.Inverse(re, im)

	// Fold: the response is made causal by moving the energy that sits
	// before time zero onto its mirror after it.
	for i := 1; i < n/2; i++ {
		re[i] *= 2
		im[i] *= 2
	}

	for i := n/2 + 1; i < n; i++ {
		re[i], im[i] = 0, 0
	}

	audio.Forward(re, im)

	// Back out of the log, which is where the phase arrives.
	for i := range re {
		mag := math.Exp(re[i])
		re[i], im[i] = mag*math.Cos(im[i]), mag*math.Sin(im[i])
	}

	audio.Inverse(re, im)

	return re
}

// spectrum is a signal's frequency content, zero padded to length.
func spectrum(
	of []float64,
	n int,
) []complex128 {
	re := make([]float64, n)
	im := make([]float64, n)

	copy(re, of)
	audio.Forward(re, im)

	out := make([]complex128, n)
	for i := range out {
		out[i] = complex(re[i], im[i])
	}

	return out
}

// samples turns a spectrum back into a signal.
func samples(
	of []complex128,
) []float64 {
	n := len(of)
	re := make([]float64, n)
	im := make([]float64, n)

	for i, v := range of {
		re[i], im[i] = real(v), imag(v)
	}

	audio.Inverse(re, im)

	return re
}

// magnitudes is how much is in each bin, ignoring when it arrived.
func magnitudes(
	of []complex128,
) []float64 {
	out := make([]float64, len(of))
	for i, v := range of {
		out[i] = cmplx.Abs(v)
	}

	return out
}

// trim cuts an impulse response to the length a device will load.
//
// Windowed at the end rather than cut square. A response that stops abruptly
// has a step in it, and a step is broadband: it reads as a click on every
// note, which is not what the cabinet did.
func trim(
	of []float64,
	taps int,
) []float64 {
	out := make([]float64, taps)
	copy(out, of)

	// The last eighth, which is long enough to be gentle and short enough to
	// leave the body of the response alone.
	fade := taps / 8

	for i := range fade {
		at := taps - fade + i
		out[at] *= 0.5 * (1 + math.Cos(math.Pi*float64(i)/float64(fade)))
	}

	return normalise(out)
}

// normalise scales an impulse response so it does not clip.
//
// To just under full scale rather than to it, because a device writing the
// result into 24 bits has nowhere to round up to otherwise.
func normalise(
	of []float64,
) []float64 {
	var peak float64
	for _, v := range of {
		peak = math.Max(peak, math.Abs(v))
	}

	if peak == 0 || peak <= 0.99 {
		return of
	}

	for i := range of {
		of[i] *= 0.99 / peak
	}

	return of
}

// usable reports whether a device would load an impulse response this long.
func usable(
	taps int,
) error {
	if taps != Short && taps != Long {
		return &BadLengthError{Taps: taps}
	}

	return nil
}

// next is the power of two at or above a length.
func next(
	n int,
) int {
	out := 1
	for out < n {
		out *= 2
	}

	return out
}
