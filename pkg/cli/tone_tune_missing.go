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
	"fmt"
	"io"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// This file is what the loop says when the dials are not the problem.
//
// Two different conclusions come out of a tuning run and only one of them used
// to be reported. The dials were wrong, which the solver fixes, or the chain is
// wrong, which it cannot: every dial is already at its limit and the target
// wants something no block in front of it produces. That case printed the axes
// it missed and stopped, which left somebody reading a list of failures with
// nothing to do about it.
//
// What decides a chain today is the gear a rig names plus the corpus grammar
// filling in what it did not. The grammar is a frequency argument rather than a
// measurement: every rig compiled on 2026-09-27 gained a Deluxe Comp because
// "almost every chain has one (88% of chains)". A reasonable default, and not
// aimed at anything.
//
// So this aims. It reads the axes that were missed, searches the 661 committed
// fingerprints for blocks whose own readings move those axes the right way, and
// prints the few worth trying. A block suggested because it closes a 2,000Hz
// centroid gap is a different claim from one added because most presets have
// one, and the output says which it is.

// shortlisted is how many blocks are worth printing.
//
// Five, because this is a prompt for the next run rather than a decision. Each
// one costs a rebuild and a pass to actually test, so a list nobody will work
// through is a list that gets skimmed and ignored.
const shortlisted = 5

// missing prints the blocks that would close what the dials could not.
//
// Said rather than done. A fingerprint is one reading at that block's defaults
// with the block alone in the chain, so it says which blocks are worth trying
// and not what one will do here: an amplifier measured with no cabinet has no
// speaker rolloff, and four of eleven controls on one moved the centroid the
// other way once a cabinet was in front of it. Adding a block on that evidence
// would be the grammar's mistake with extra steps.
//
// Nothing is printed when the library cannot answer, and nothing is refused
// over it. A tuning run that got within 2.7 tolerances is worth keeping whether
// or not a shortlist came out of it, and the run has already reported that.
func missing(
	w io.Writer,
	made plan.Plan,
	aims map[audio.Figure]solve.Aim,
	now map[audio.Figure]float64,
) {
	lib, err := measured.BuiltIn()
	if err != nil {
		return
	}

	want := unreached(aims, now)
	if len(want) == 0 {
		return
	}

	got := measured.Shortlist(lib, want, inTheChain(made), shortlisted)
	if len(got) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w,
		"\n  The dials are at their limits, so this is the chain rather than the\n"+
			"  settings. Measured alone at their own defaults, these blocks move\n"+
			"  what is missing in the right direction:\n\n")

	_, _ = fmt.Fprintf(w, "    %-34s %-9s %s\n", "BLOCK", "KIND", "WHAT ITS OWN READING DOES")

	for _, s := range got {
		_, _ = fmt.Fprintf(w, "    %-34s %-9s %s\n",
			s.Block.Name, s.Block.Category, why(s))
	}

	// The caution is printed rather than left to a page, because this table is
	// the most tempting thing in the output to act on directly.
	_, _ = fmt.Fprintf(w,
		"\n  Each was measured on its own, which is not this chain: the same\n"+
			"  control reads differently with a cabinet in front of it. Add one,\n"+
			"  run again, and let the measurement settle it.\n")
}

// unreached is the axes a target still wants, in the library's own units.
//
// The conversion is the point of this function. A target and a reading are in
// the corpus's units, where a band is a fraction, and the library reports a band
// as a percentage. Handing a corpus-scaled want to the shortlist would be a
// hundred times wrong on four of the ten axes and entirely plausible on the
// other six, which is the shape of mistake this repository keeps a helper for.
//
// An axis already inside its tolerance is left out. Nothing needs finding for it,
// and including it would let a block score well for leaving alone something that
// was never asked about.
func unreached(
	aims map[audio.Figure]solve.Aim,
	now map[audio.Figure]float64,
) []measured.Want {
	out := make([]measured.Want, 0, len(aims))

	for _, key := range measured.Named() {
		aim, asked := aims[key]
		if !asked || aim.Tol == 0 {
			continue
		}

		read, has := now[key]
		if !has {
			continue
		}

		need := aim.Want - read
		if need < 0 && -need < aim.Tol || need >= 0 && need < aim.Tol {
			continue
		}

		out = append(out, measured.Want{
			Figure: key,
			Need:   inLibraryScale(key, need),
			Tol:    inLibraryScale(key, aim.Tol),
		})
	}

	return out
}

// inLibraryScale is the inverse of inCorpusScale, for one figure.
//
// Which axes are shares is audio.Figure.Share's to say, so the two directions
// cannot disagree about the set: an axis added to one and not the other is an
// axis silently off by a hundred.
func inLibraryScale(
	key audio.Figure,
	of float64,
) float64 {
	if key.Share() {
		return of * perCent
	}

	return of
}

// inTheChain is the models already in the plan, which are not additions.
func inTheChain(
	made plan.Plan,
) map[string]bool {
	out := make(map[string]bool, len(made.Blocks))

	for _, b := range made.Blocks {
		out[string(b.Model)] = true
	}

	return out
}

// why says what one block's own reading did, in the axes that were missing.
//
// The figures rather than a verdict, because the verdict is the next run's. Two
// blocks that both "add harmonics" are not the same suggestion when one adds 40
// and the other adds 4.
func why(
	s measured.Suggestion,
) string {
	said := ""

	for _, key := range s.Helps {
		if said != "" {
			said += ", "
		}

		said += fmt.Sprintf("%s %+.3g", key, s.Moves[key])
	}

	if said == "" {
		said = "nothing in the right direction"
	}

	for _, key := range s.Hurts {
		said += fmt.Sprintf(", but %s %+.3g the wrong way", key, s.Moves[key])
	}

	return said
}
