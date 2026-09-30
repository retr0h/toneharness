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
// Line 6 tag every amplifier Guitar or Bass and that tag is in the catalog, so
// the chain says what it is for without anybody writing it down.
//
// **Every amplifier, and a bass one anywhere decides it.** That is the rule
// `compile` uses when it lifts a preset, and the two have to agree or this
// refuses a rig for the instrument the rig itself says it is. Returning on the
// first amplifier found is not the same rule: a chain holding a guitar amp
// before a bass amp reads as bass to the compiler and would have read as guitar
// here, so a bass rig pushed a bass recording would have been refused.
//
// Where they differ on purpose: `compile` answers guitar for a chain with no
// amplifier at all, because a rig's `instrument` field has to say something.
// This answers nothing. An effect serves either instrument, so there is no claim
// to make, and a guard that invents one would refuse a delay being measured.
func chainIsFor(
	made plan.Plan,
	cat *catalog.Catalog,
) string {
	found := ""

	for _, b := range made.Blocks {
		blk, known := cat.Block(b.Model)
		if !known || blk.Category != catalog.CategoryAmp {
			continue
		}

		if blk.Subcategory == "Bass" {
			return "bass"
		}

		found = "guitar"
	}

	return found
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
