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
	"errors"
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// This file refuses to measure a chain through the wrong instrument.
//
// Every figure this repository has committed was taken with a bass through it,
// and until now nothing compared that to the chain being measured. A guitar rig
// pushed a bass recording reads as a guitar rig that is dark, because the
// signal was, and the solve then spends every dial correcting a difference the
// reference put there.
//
// It does not fail on its own, which is why it is worth a guard. The figures
// are real measurements of a real chain; they are just answers about the wrong
// question, and nothing downstream can tell.
//
// The rule applies to tuning a rig, not to building a library. `measure blocks`
// deliberately pushes one reference through all 661 blocks, guitar amplifiers
// included, because the library's subject is what each block does to that
// signal. There the instrument is recorded rather than enforced.

// ErrWrongInstrument is a chain measured through another instrument's
// reference.
var ErrWrongInstrument = errors.New("the reference is not this chain's instrument")

// chainIsFor is the instrument a chain is for, by the amplifiers in it.
//
// plan.InstrumentFor holds the rule, shared with the compiler and the corpus
// measurer, and the three have to agree: this refuses a rig for the instrument
// the rig itself says it is, so a different answer here is a bass rig turned
// away from a bass recording.
//
// The empty answer is kept rather than turned into a word. An effect serves
// either instrument, so a chain with no amplifier makes no claim, and a guard
// that invented one would refuse a delay being measured.
func chainIsFor(
	made plan.Plan,
	cat *catalog.Catalog,
) string {
	return plan.InstrumentFor(made, cat)
}

// sameInstrument refuses a chain about to be measured through the wrong
// reference.
//
// Refused rather than reported, which is the opposite of how this file's
// neighbours handle a doubtful reading. A squeal or a missing slope makes one
// figure untrustworthy and the run still tells you something. This makes every
// figure in the run an answer about a different instrument, and there is nothing
// to salvage from it: five minutes of measuring would report tolerances met
// against a target the reference invented.
//
// Nothing is claimed where nothing is known. A chain with no amplifier names no
// instrument, and a reference whose name says neither `bass` nor `guitar` is
// somebody's own recording rather than one of the two shipped here.
func sameInstrument(
	made plan.Plan,
	cat *catalog.Catalog,
	dry string,
) error {
	chain, reference := chainIsFor(made, cat), instrumentOf(dry)
	if chain == "" || reference == "" || chain == reference {
		return nil
	}

	return fmt.Errorf(
		"%w: this chain's amplifier is a %s amplifier and %s is a %s recording. "+
			"Every figure measured through it would describe the reference rather "+
			"than the chain, so the solve would spend its dials on the difference. "+
			"Push a %s recording with --dry",
		ErrWrongInstrument, chain, dry, reference, chain)
}
