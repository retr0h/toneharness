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

package measured

import (
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// This file answers a different question from the rest of the package.
//
// Everything else here describes chains somebody already built: what a control
// did, how far a figure moved, whether a target sits inside ground the hardware
// stood on. This decides which block to put in a chain that has none of it.
//
// The two are not the same job and the solver cannot do this one. Solving moves
// the dials of the blocks in front of it; if a target wants more harmonics than
// anything in the chain produces, every dial is already at its limit and the
// answer is a block rather than a position. Today that case reports which axes
// were missed and stops, which leaves somebody reading a list of failures with
// nothing to do about it.
//
// **A fingerprint says which blocks are worth trying, and nothing more.** Each
// is one reading, at that block's own defaults, with the block alone in the
// chain. So it does not say what the block will do in this chain at these
// settings: an amplifier measured with no cabinet in front of it has no speaker
// rolloff, and four of eleven controls on one moved the centroid the opposite
// way once a cabinet was there. That is the same caution Curves.Isolated
// carries, and it is why this returns a shortlist for the loop to measure rather
// than a block to add.

// Want is one axis a chain could not reach.
//
// Need and Tol are both in the library's own units, which are not the corpus's:
// this package reports a band as a percentage and the corpus as a fraction, so a
// want built straight off a corpus target is a hundred times wrong and looks
// plausible. The caller converts. It is stated here because the two scales have
// already been mixed once.
type Want struct {
	// Figure is the axis.
	Figure audio.Figure
	// Need is the change the chain needs, signed: positive to raise the
	// figure, negative to lower it.
	Need float64
	// Tol is how close counts as close enough, so a shortlist can weigh an
	// axis measured in Hz against one measured in percent.
	Tol float64
}

// Suggestion is a block worth trying, and what its own reading did.
type Suggestion struct {
	// Block is the reading, carrying the model identifier and the category.
	Block Block
	// Moves is what this block's reading sits away from the empty loop, per
	// wanted axis, in the library's units. The block's own effect, in other
	// words, since the baseline is the signal with nothing in the chain.
	Moves map[audio.Figure]float64
	// Worst is the axis left furthest out once this block's effect is
	// credited, in tolerances. Lower is better and it is what the ranking is
	// on.
	//
	// The same measure `solve.Nearest` ranks a list's settings by, and for the
	// same reason: summing the axes would rank a block that is slightly wrong
	// everywhere above one that is right on everything but the axis the target
	// cares about.
	Worst float64
	// Helps is the axes this block moves the right way, and Hurts the ones it
	// moves the wrong way.
	//
	// Both reported, because a block that closes the gap it was found for
	// while opening another is a real answer and a person has to see the
	// trade. Nothing here refuses one: the loop measures the chain afterwards
	// and that is what settles it.
	Helps []audio.Figure
	Hurts []audio.Figure
}

// Shortlist is the blocks whose own readings move the unmet axes the right way.
//
// Ranked by the axis each leaves furthest out, best first. Every reading that
// cannot describe its block is left out: one that clipped is a reading of the
// clipping, one the device refused has no figures, and a block already in the
// chain is not an addition.
//
// The result is a shortlist rather than a choice, which is the whole contract
// here. Each of these is worth eight seconds of measuring in the chain in hand,
// and what that measures is the answer. This only decides what to spend the
// eight seconds on, out of six hundred and sixty one.
func Shortlist(
	lib Library,
	want []Want,
	already map[string]bool,
	limit int,
) []Suggestion {
	if len(want) == 0 {
		return nil
	}

	out := make([]Suggestion, 0, len(lib.Blocks))

	for id, block := range lib.Blocks {
		if already[id] || block.Refused != "" || block.Clipped {
			continue
		}

		got, ok := suggest(lib.Baseline, block, want)
		if !ok {
			continue
		}

		out = append(out, got)
	}

	// Worst first, then by identifier, so two blocks that score alike come
	// back in the same order every run rather than in whatever order the map
	// was walked.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Worst != out[j].Worst {
			return out[i].Worst < out[j].Worst
		}

		return out[i].Block.ID < out[j].Block.ID
	})

	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}

	return out
}

// suggest scores one block against what the chain could not reach.
//
// Refused rather than scored when the block's reading carries none of the wanted
// axes. Three of the figures are pointers because they were added after the
// first sweeps ran, and a block measured before that has a gap rather than a
// zero: scoring it as zero would rank an unmeasured block against a measured one
// and prefer whichever direction zero happened to favour.
func suggest(
	baseline Figures,
	block Block,
	want []Want,
) (Suggestion, bool) {
	out := Suggestion{
		Block: block,
		Moves: make(map[audio.Figure]float64, len(want)),
	}

	read := 0

	for _, w := range want {
		was, had := figureOf(baseline, w.Figure)
		now, has := figureOf(block.Figures, w.Figure)

		if !had || !has {
			continue
		}

		read++

		moved := now - was
		out.Moves[w.Figure] = moved

		// What is left once this block's own effect is credited against what
		// the chain needed, in tolerances. A block that moves the figure
		// further than the gap overshoots, and an overshoot is a gap too:
		// its dials have to come back, which is what the solver is for.
		left := (w.Need - moved) / w.Tol
		if left < 0 {
			left = -left
		}

		if left > out.Worst {
			out.Worst = left
		}

		switch {
		case moved == 0:
		case (moved > 0) == (w.Need > 0):
			out.Helps = append(out.Helps, w.Figure)
		default:
			out.Hurts = append(out.Hurts, w.Figure)
		}
	}

	if read == 0 {
		return Suggestion{}, false
	}

	sort.Slice(out.Helps, func(i, j int) bool { return out.Helps[i] < out.Helps[j] })
	sort.Slice(out.Hurts, func(i, j int) bool { return out.Hurts[i] < out.Hurts[j] })

	return out, true
}

// figureOf reads one axis off a reading, and says whether it was measured.
//
// The three pointer figures answer false where they are absent, which is what
// keeps an unmeasured axis out of a score rather than in it as a zero.
func figureOf(
	f Figures,
	key audio.Figure,
) (float64, bool) {
	switch key {
	case audio.KeyCentroid:
		return f.Centroid, true
	case audio.KeyLevel:
		return f.Level, true
	case audio.KeyLow:
		return f.Low, true
	case audio.KeyMid:
		return f.Mid, true
	case audio.KeyHigh:
		return f.High, true
	case audio.KeyTransient:
		return value(f.Transient)
	case audio.KeyDecay:
		return value(f.Decay)
	case audio.KeyDynamics:
		return value(f.Dynamics)
	case audio.KeyHarmonics:
		return value(f.Harmonics)
	case audio.KeyLean:
		return value(f.Lean)
	}

	return 0, false
}

// value is a pointer figure and whether it was measured at all.
func value(
	of *float64,
) (float64, bool) {
	if of == nil {
		return 0, false
	}

	return *of, true
}

// Categories narrows a shortlist to the kinds of block worth adding.
//
// Separate from Shortlist because which categories are eligible is the caller's
// question, not this package's: a chain missing harmonics wants a drive, and one
// that already holds an amplifier does not want a second whatever its reading
// says. The DSP budget and how many blocks a device holds are the same kind of
// question, and `plan.LimitsFor` answers those from the catalog.
func Categories(
	of []Suggestion,
	kinds ...catalog.Category,
) []Suggestion {
	if len(kinds) == 0 {
		return of
	}

	want := make(map[catalog.Category]bool, len(kinds))
	for _, k := range kinds {
		want[k] = true
	}

	out := make([]Suggestion, 0, len(of))

	for _, s := range of {
		if want[s.Block.Category] {
			out = append(out, s)
		}
	}

	return out
}
