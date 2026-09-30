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
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// ErrSquealing is a reading of the loop rather than of the chain.
//
// Refused rather than reported, because every figure a run produces is built
// on this one. A warning above a table of numbers nobody can trust is a table
// of numbers somebody will act on.
var ErrSquealing = errors.New("the loop is oscillating")

// endsInACab says the last block in a chain is a cabinet.
func endsInACab(
	made plan.Plan,
	cat *catalog.Catalog,
) bool {
	var last catalog.ModelID

	at := -1

	for _, b := range made.Blocks {
		if b.Pos > at {
			at, last = b.Pos, b.Model
		}
	}

	blk, ok := cat.Blocks[last]

	return ok && blk.Category == catalog.CategoryCab
}

// ReachOptions is what asking whether a chain can meet a target needs.
type ReachOptions struct {
	Client   Tuner
	Genres   Genres
	Bench    sdk.Bench
	Hardware string
	Dry      string
	Seconds  float64
	Takes    int
	// ID is the curated rig to ask about, and Genre what to aim it at.
	ID    string
	Genre string
	// Corpus is the tree the target is measured from. Empty reads the figures
	// shipped with this binary.
	Corpus string
	// Nudge is how far a control moves to read its slope, as a fraction of
	// its range.
	Nudge float64
	// Headroom is how far the chain's own output is turned down before
	// anything is measured, in decibels, and wants to be negative.
	Headroom float64
}

// Reach says how near a chain can get to a target, from one pass of readings.
//
// Read off the chain rather than out of resources/sweeps/, and that is the
// whole design. The first version of this answered from the committed sweeps
// and cost a second. Every one of them was taken with its block alone, which
// is a different signal: an SV Beast swept with no cabinet has a median
// centroid of 8,139Hz where the chain a rig builds reads about 144, and four
// of its eleven controls move the centroid the other way. Nothing computed
// from those numbers was about the chain it was asked about.
//
// So it measures. One reading to settle, one for the baseline, one per
// control, which is a minute and a half on a chain of sixteen against five
// minutes for the tuning run it decides whether to spend. Cheaper than the
// thing it precedes rather than free, and every number about the chain in
// hand.
//
// Nothing is applied. The chain is left where the compiler put it, and what
// comes back is the model's own answer about what could be done from there.
func Reach(
	ctx context.Context,
	w io.Writer,
	opts ReachOptions,
) error {
	target, err := targetFor(ctx, TuneOptions{
		Genres: opts.Genres, Genre: opts.Genre, Corpus: opts.Corpus,
	})
	if err != nil {
		return err
	}

	made, _, _, err := built(ctx, TuneOptions{
		Client: opts.Client, ID: opts.ID, Headroom: opts.Headroom,
	})
	if err != nil {
		return err
	}

	cat, err := catalog.BuiltIn()
	if err != nil {
		return err
	}

	if err := sameInstrument(made.Plan, cat, opts.Dry); err != nil {
		return err
	}

	knobs := knobsOf(made.Plan, cat)
	if len(knobs) == 0 {
		return fmt.Errorf("%w: the chain has no dial to turn", solve.ErrNoKnobs)
	}

	signal, err := reference(opts.Dry, opts.Seconds)
	if err != nil {
		return err
	}

	bench, release, err := benchFor(opts.Bench, opts.Hardware)
	if err != nil {
		return err
	}

	defer release()

	_, _ = fmt.Fprintf(w, "\n  %s against %s, %d dials through %s\n",
		opts.ID, opts.Genre, len(knobs), bench.Name())

	floor, settled, err := steady(ctx, bench, signal, opts.Takes)
	if err != nil {
		return err
	}

	aims := solve.Aims(target, inCorpusScale(floor))
	if len(aims) == 0 {
		return fmt.Errorf("%w: %q measures as nothing", ErrNoTarget, opts.Genre)
	}

	aims[audio.KeyLevel] = solve.Aim{Want: settled, Tol: drift}

	now, err := sdk.Fingerprint(ctx, bench, signal)
	if err != nil {
		return err
	}

	from := figuresOf(now)

	// Before anything is computed from it. Every number below rests on this
	// reading, and a chain giving back more high end than went in is not a
	// chain being read at all.
	if say, bad := Squealing(from, figuresOfDry(signal), inCorpusScale(floor),
		endsInACab(made.Plan, cat)); bad {
		return fmt.Errorf("%w: %s", ErrSquealing, say)
	}

	_, _ = fmt.Fprintf(w,
		"  the loop wanders %.4f of a band and %.1fHz, reading %d controls\n",
		inCorpusScale(floor)[audio.KeyLow], floor[audio.KeyCentroid], len(knobs))

	if err := slopes(ctx, bench, signal, TuneOptions{
		Client: opts.Client, Nudge: opts.Nudge,
	}, knobs, from); err != nil {
		return err
	}

	axes := solve.Reachable(aims, spansOf(knobs, from))

	joint, err := solve.Best(knobs, aims, from, jointPasses)
	if err != nil {
		return err
	}

	for i := range axes {
		axes[i].Together = joint.Residual[axes[i].Figure]
	}

	table(w, axes)
	verdict(w, axes, joint)

	return nil
}

// jointPasses is how many times the joint solve re-solves from where clamping
// left it.
//
// Five, matching what the loop takes on hardware. The slopes are fixed here so
// it converges rather than wandering, and it stops as soon as a pass stops
// improving.
const jointPasses = 5

// spansOf is what each axis reads now and the most the dials could move it.
//
// The swing is every control's slope across its own remaining travel, added
// up. It flatters them: it assumes each slope holds across a range it was read
// locally and that every control pulls the same way, neither of which is true.
// That is the useful direction. A gap wider than this is one nothing
// optimistic closes, so it refuses well and promises badly.
func spansOf(
	knobs []solve.Knob,
	from map[audio.Figure]float64,
) map[audio.Figure]solve.Span {
	out := make(map[audio.Figure]solve.Span, len(from))

	for key, at := range from {
		span := solve.Span{From: at, Low: at, High: at}

		for _, k := range knobs {
			slope := k.Slope[key]

			// Each way separately, because a control sitting near a stop has
			// more travel one way than the other and the reachable range is
			// not symmetric about where it sits.
			up := slope * (k.High - k.At)
			down := slope * (k.Low - k.At)

			span.High += max(up, down)
			span.Low += min(up, down)
			span.Swing += max(up, down) - min(up, down)
		}

		out[key] = span
	}

	return out
}

// table prints one line per axis, worst first.
func table(
	w io.Writer,
	axes []solve.Verdict,
) {
	_, _ = fmt.Fprintf(w, "\n    %-12s %10s %10s %6s %8s %8s\n",
		"AXIS", "READS", "WANTS", "WAY", "OUT BY", "TOGETHER")

	for _, v := range axes {
		mark, say := "  ", "nothing rules it out"

		switch {
		case v.Met:
			mark, say = "ok", "already there"
		case v.Shown:
			mark, say = "ok", "the dials reach it"
		case !v.Within:
			mark, say = "->", "OUT OF REACH"
		}

		if v.Together > 1 {
			mark = "->"
		}

		// Which way, because "390 out" says nothing about what would help and
		// the block that closes a gap is the one pointing along it.
		way := "up"
		if v.Want < v.From {
			way = "down"
		}

		// What it reads and what it wants, not only how far apart they are.
		// A distance in tolerances is unreadable on its own: 390 of them
		// sounds like a broken chain and can be a tenth of a band.
		_, _ = fmt.Fprintf(w, "    %s %-9s %10.4g %10.4g %6s %8.1f %8.1f  %s\n",
			mark, v.Figure, v.From, v.Want, way, v.Gap, v.Together, say)
	}
}

// verdict is the answer the table was working towards.
func verdict(
	w io.Writer,
	axes []solve.Verdict,
	joint solve.Result,
) {
	// Emptiness and refusal are different answers, and Worth reports false for
	// both: no axis named, and some axis out of reach. Reading them as one
	// printed "the target names no axis this chain reads" over a table that
	// had just named one and marked it OUT OF REACH.
	if len(axes) == 0 {
		_, _ = fmt.Fprintf(w,
			"\n  Nothing to aim at: the target names no axis this chain reads.\n")

		return
	}

	worst, _ := solve.Worth(axes)

	if !worst.Within {
		_, _ = fmt.Fprintf(w,
			"\n  Not worth running. %s is %.1f tolerances out and every dial in "+
				"this chain,\n  added up and pulling together, moves it %.1f.\n"+
				"  The gear is wrong for this sound. Change the chain, not the "+
				"knobs.\n", worst.Figure, worst.Gap, worst.Swing)

		return
	}

	if joint.Arrived {
		_, _ = fmt.Fprintf(w,
			"\n  Worth running. One set of positions reaches all %d axes at once, "+
				"in the model.\n  The slopes behind that were read where the chain "+
				"sits now and drift away\n  from it, so the loop will take several "+
				"passes to find them.\n", len(axes))

		return
	}

	var missed []string

	for _, v := range axes {
		if v.Together > 1 {
			missed = append(missed,
				fmt.Sprintf("%s by %.1f", v.Figure, v.Together))
		}
	}

	if len(missed) == 0 {
		_, _ = fmt.Fprintf(w,
			"\n  Worth running, though nothing here can say how it will go: no "+
				"dial in this\n  chain has a slope on any axis the target names.\n")

		return
	}

	_, _ = fmt.Fprintf(w,
		"\n  Every axis is reachable on its own and no one set of positions "+
			"reaches them\n  together. Solved as a whole the model still misses "+
			"%s.\n  Worth running, and expect the loop to trade one axis off "+
			"against another.\n", strings.Join(missed, ", "))
}
