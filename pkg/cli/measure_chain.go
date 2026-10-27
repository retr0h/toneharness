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
	"os"
	"path/filepath"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
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

	floor, at, err := steady(ctx, bench, signal, opts.Takes)
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
		now[audio.KeyHigh]*perCent, now[audio.KeyCentroid], at.Level)

	// Nothing justified. This reads whatever the pedal is playing and is handed
	// `--cab` by a person rather than a chain it can inspect, so it has no list
	// of blocks to sum. That leaves it on the bare invariant, which will call a
	// drive a squeal; it says so as a warning rather than refusing, and `tone
	// tune` is the path that knows what the chain holds.
	if say, bad := Squealing(
		now, figuresOfDry(signal), inCorpusScale(floor), opts.EndsInACab, 0); bad {
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

// apart is how much more of the high band a chain may give back than its own
// blocks account for before the reading is of the loop rather than of the
// chain.
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
// brighter than what it was given.
//
// A chain that ends in a cabinet may still hold something that brightens, and
// justified is how much its own blocks account for. A drive is the case: an
// Arbiter Fuzz Face into an Ampeg SVT into an 8x10 read 6.5% above 2kHz
// through the open loop, which is not oscillation and was refused as it for
// as long as the comparison was against a constant.
//
// Widening apart cannot answer it. Of the 67 drives this device models, a
// Xenomorph Fuzz reaches 34.2% of its energy above 2kHz in its own sweep and a
// Line 6 Distortion 33.6%, while the quietest squeal anybody has recorded here
// is 39%. There is no number between those two populations any more, which is
// what apart used to sit in.
//
// So the ceiling is what the chain's blocks were separately measured reaching,
// which the caller sums from the committed curves. On that fuzz chain the sum
// is 6.437% against the 6.5% the chain read, which is the agreement this rule
// rests on.
//
// It loses its power at the bright end, and says so rather than pretending.
// A chain built on a Xenomorph justifies around 39%, which is the quietest
// squeal, so for the brightest drives this invariant cannot tell the two
// apart. What it must not do is refuse a chain it has no case against.
func Squealing(
	now, dry, wander map[audio.Figure]float64,
	endsInACab bool,
	justified float64,
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
	if !in || !out || is <= was+justified+apart+wander[audio.KeyHigh] {
		return "", false
	}

	return fmt.Sprintf(
		"this chain ends in a cabinet and gave back %.1f%% of its energy above "+
			"2kHz\n      against %.2f%% in the reference and %.1f%% its own blocks "+
			"account for.\n      A loudspeaker cannot add that."+
			"\n      The loop is oscillating: turn the amplifier's output down "+
			"and read it again.",
		is*perCent, was*perCent, justified*perCent), true
}

// justifiedBy is how much of the high band a chain's own blocks account for.
//
// The sum of what each block in the chain was separately measured reaching
// above 2kHz, read from the committed curves. It is the ceiling Squealing
// compares a reading against, because a constant cannot separate a drive from
// a squeal any more: see the note on Squealing for the two populations that
// now overlap.
//
// Summed rather than maxed, on the evidence. For the chain `--extend` proposed
// for matt-freeman, a Deluxe Comp into an Arbiter Fuzz Face into an Ampeg SVT
// into an 8x10:
//
//	HD2_CompressorDeluxeComp   0.010%
//	HD2_DistArbitratorFuzz     1.385%
//	HD2_AmpSVBeastBrt          4.941%
//	HD2_CabMicIr_8x10SVTAV     0.101%
//	                           ------
//	sum                        6.437%
//
// The chain read 6.5%. The maximum is 4.941% and would have refused it, so the
// blocks compound rather than one of them dominating.
//
// Zero where there are no curves, which is every installed release: the sweeps
// live under resources/sweeps/ and `just pack-measured` embeds only the
// fingerprints. That leaves the exact invariant in place, which is the right
// way to be wrong, because it refuses a chain rather than trusting one.
//
// Zero for a block nothing has swept, too. A block with no curves cannot
// justify its own brightness, and guessing a figure for it would be the one
// thing this whole library exists to stop.
func justifiedBy(
	made plan.Plan,
	cat *catalog.Catalog,
	sweeps string,
) float64 {
	if sweeps == "" {
		return 0
	}

	var out float64

	// By model rather than by position, because a chain may hold two of a kind
	// and each of them brightens.
	for _, b := range made.Blocks {
		if _, known := cat.Blocks[b.Model]; !known {
			continue
		}

		out += brightest(filepath.Join(sweeps, string(b.Model)+".json"))
	}

	return out / perCent
}

// brightest is the most of the high band one block's own sweep ever read.
//
// Across every control and every position of it, because a control is swept
// through its range and the chain is at whatever the solve put it at. The
// ceiling is what the block can do, not what it does at its defaults.
//
// A file that is not there, or will not parse, is zero rather than an error.
// This is a guard's input: a missing sweep should leave the guard stricter,
// never stop the run that was about to measure something.
func brightest(
	at string,
) float64 {
	f, err := os.Open(at) //nolint:gosec // a path built from the catalog
	if err != nil {
		return 0
	}

	defer func() { _ = f.Close() }()

	curves, err := measured.LoadCurves(f)
	if err != nil {
		return 0
	}

	var out float64

	for _, c := range curves.Controls {
		for _, p := range c.Points {
			out = max(out, p.High)
		}
	}

	return out
}

// Returned says whether anything came back through the loop.
//
// Asked before any figure is read as a fact about the chain, because a reading
// of the converter's own floor is a reading of noise and noise answers every
// other question wrongly. Band shares are the trap: noise is broadband, so a
// dead return reads more of its energy above 2kHz than a bass reference does and
// trips `Squealing`, which then tells somebody to turn an amplifier down over a
// lead that is not carrying. The two faults have opposite fixes.
//
// Measured against what went in rather than against the settled reading, because
// a dead path settles on its own noise: `steady` measured the same silence and
// every figure relative to it agrees with every other.
//
// The populations are nowhere near each other. A working loop here loses 16dB,
// -13dB of bass going in and -28.9dB coming back; the dead one lost 66. The
// threshold is `silent`, which is already this project's answer to "that reading
// is the noise floor" and sits twice above the working case and half below the
// dead one.
func Returned(
	sent, got float64,
) (string, bool) {
	if sent-got <= silent {
		return "", false
	}

	return fmt.Sprintf(
		"%.1fdB went in and %.1fdB came back, which is %.0fdB of loss and not a "+
			"chain\n      Nothing is returning: check the lead into the "+
			"interface's input, and that the preset's output is the one it is "+
			"plugged into", sent, got, sent-got), true
}
