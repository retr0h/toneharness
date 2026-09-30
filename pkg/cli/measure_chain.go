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

package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// ChainOptions is what reading the loaded chain needs.
type ChainOptions struct {
	Bench    sdk.Bench
	Hardware string
	Dry      string
	Seconds  float64
	Takes    int
	// EndsInACab says the loaded chain finishes with a cabinet, which is what
	// makes the high band a one-way street and the squeal detectable.
	EndsInACab bool
}

// Chain reads what the pedal is playing, and changes nothing.
//
// The primitive everything else here assumes and nothing provided. `tone tune`,
// `tone reach` and every `measure` command build a preset and load it before
// they read, so none of them can answer "what is it doing now". That gap is
// why a chain reading 84% of its energy above 2kHz went a whole session
// unnoticed: the figure was always folded into a distance from a target.
//
// Nothing is built, nothing is loaded and no control is moved. Play a preset,
// turn a knob with `device turn`, and read what that did.
func Chain(
	ctx context.Context,
	w io.Writer,
	opts ChainOptions,
) error {
	signal, err := reference(opts.Dry, opts.Seconds)
	if err != nil {
		return err
	}

	bench, release, err := benchFor(opts.Bench, opts.Hardware)
	if err != nil {
		return err
	}

	defer release()

	floor, settled, err := steady(ctx, bench, signal, opts.Takes)
	if err != nil {
		return err
	}

	got, err := sdk.Fingerprint(ctx, bench, signal)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "\n  through %s, %d takes\n", bench.Name(), opts.Takes)
	_, _ = fmt.Fprintf(w, "    %-12s %12s %12s\n", "FIGURE", "READS", "WANDERS")

	for _, key := range measured.Named() {
		at, ok := figuresOf(got)[key]
		if !ok {
			continue
		}

		_, _ = fmt.Fprintf(w, "    %-12s %12.5g %12.5g\n",
			key, at, inCorpusScale(floor)[key])
	}

	// The bands said plainly, because a share is the figure a fault shows in
	// first and three numbers that sum to one are easier to disbelieve than
	// one centroid.
	now := figuresOf(got)

	_, _ = fmt.Fprintf(w,
		"\n    %.1f%% below 250Hz, %.1f%% to 2kHz, %.1f%% above, centroid %.0fHz "+
			"at %.1fdB\n",
		now[audio.KeyLow]*perCent, now[audio.KeyMid]*perCent,
		now[audio.KeyHigh]*perCent, now[audio.KeyCentroid], settled)

	if say, bad := Squealing(
		now, figuresOfDry(signal), inCorpusScale(floor), opts.EndsInACab); bad {
		_, _ = fmt.Fprintf(w, "\n    [!] %s\n", say)
	}

	return nil
}

// figuresOfDry is what the reference recording measures as on its own.
//
// Measured rather than assumed. The check below compares what came back
// against what went in, and "what went in" is this file rather than a number
// anybody chose.
func figuresOfDry(
	signal []float32,
) map[audio.Figure]float64 {
	samples := make([]float64, len(signal))
	for i, v := range signal {
		samples[i] = float64(v)
	}

	got := audio.Measure(samples, sdk.Rate)

	return map[audio.Figure]float64{
		audio.KeyLow:      got.Low,
		audio.KeyMid:      got.Mid,
		audio.KeyHigh:     got.High,
		audio.KeyCentroid: got.Centroid,
	}
}

// Squealing says the chain gave back more high end than was put in.
//
// A cabinet is a loudspeaker and a loudspeaker is a low pass. A chain ending
// in one can remove energy above 2kHz and cannot add it, so a reading whose
// high band is above the reference's own describes something other than the
// chain.
//
// What it catches, measured on an HX Stomp with the cable rig: the pedal's
// output destination is `Multi (1/4", XLR, Digital, USB 1/2)`, which drives
// the quarter-inch jack the lead returns to the input, and with enough gain
// around that loop it oscillates. A compressor into an Ampeg SVT into an 8x10
// read 84.3% of its energy above 2kHz where the reference has 0.07%, and the
// same chain with the amplifier's ChVol and Master at 0.5 read 0.0% at a level
// three decibels lower.
//
// An invariant rather than a threshold, because the obvious tests both fail.
// Level does not catch it: the squeal sat at -23.6dB against -26.4dB for the
// clean reading. Nor does pushing silence through, which is the definition of
// self-oscillation and catches the chain at full gain and not at 0.7, because
// an amplifier's gain rises with what is put into it.
//
// Only for a chain that ends in a cabinet. Anything else may legitimately be
// brighter than what it was given, and a drive certainly is.
// apart is how much more of the high band a chain may give back than it was
// handed before the reading is of the loop rather than of the chain.
//
// A separator between two measured populations rather than a physical limit.
// Sweeping the headroom on matt-freeman, the readings are 73.9% above 2kHz at
// -15dB, 39.1% at -20, and 0.1% at -25 and everywhere below. Clean readings
// sit between 0.0% and 0.1% and squealing ones between 39% and 84%, which is a
// factor of four hundred with nothing in between.
//
// Five points: fifty times above what a clean reading carries and eight times
// below the quietest squeal. The invariant is exact and the measurement is
// not, and this is the width of that gap rather than a claim about
// loudspeakers.
const apart = 0.05

func Squealing(
	now, dry, wander map[audio.Figure]float64,
	endsInACab bool,
) (string, bool) {
	if !endsInACab {
		return "", false
	}

	was, in := dry[audio.KeyHigh]
	is, out := now[audio.KeyHigh]

	// Separated by a margin, because the comparison is between two
	// measurements rather than two numbers and a quiet reading carries the
	// converter's own broadband floor. A chain with 25dB of headroom reads
	// 0.1% above 2kHz against the reference's 0.01%, at a level of -55dB.
	if !in || !out || is <= was+apart+wander[audio.KeyHigh] {
		return "", false
	}

	return fmt.Sprintf(
		"this chain ends in a cabinet and gave back %.1f%% of its energy above "+
			"2kHz\n      against %.2f%% in the reference. A loudspeaker cannot add "+
			"that.\n      The loop is oscillating: turn the amplifier's output down "+
			"and read it again.",
		is*perCent, was*perCent), true
}

// referenceIsFor is what a reference recording was played on.
//
// Read off the path, because that is where it is said. The dry signals live
// under resources/dry/ as `bass-di.wav` and a corpus is a tree per instrument,
// so the word is already in the name and nothing else carries it.
//
// Empty where the name does not say, which is honest: a reading that cannot
// name its instrument should not claim one. A guitar rig ranked against
// readings taken with a bass is ranked against the wrong distribution, and the
// whole point of recording this is that somebody can tell.
//
// translate's gearIsFor asks a near-identical question of a setup's gear and
// answers guitar where this answers nothing. Both are right for their caller:
// a rig's instrument field has to hold something and says so in a note, and a
// measurement may claim nothing. Merging them would make one of the two
// answers wrong, which is why neither is named for the question they share.
func referenceIsFor(
	dry string,
) string {
	name := strings.ToLower(filepath.Base(dry))

	for _, known := range []string{"bass", "guitar"} {
		if strings.Contains(name, known) {
			return known
		}
	}

	return ""
}
