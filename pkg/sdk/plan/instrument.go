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

package plan

import "github.com/retr0h/toneharness/pkg/sdk/catalog"

// Instrument names what a chain's amplifiers say it is for.
//
// A device's catalog tags every amplifier Guitar or Bass, so a chain says
// which instrument it was built for without anybody writing it down.
const (
	// Bass is a chain holding a bass amplifier.
	Bass = "bass"
	// Guitar is a chain holding an amplifier, none of them a bass one.
	Guitar = "guitar"
	// NoAmp is a chain with no amplifier in it at all. An effect serves
	// either instrument, so there is nothing to claim.
	NoAmp = ""
)

// InstrumentFor is the instrument a chain is for, by the amplifiers in it.
//
// **Every amplifier, and a bass one anywhere decides it.** Returning on the
// first amplifier found is not the same rule: a chain holding a guitar amp
// before a bass amp reads as bass here and would have read as guitar there.
//
// It lives here because three callers had written it three ways and the three
// disagreed. Over the 4,426 presets in the corpus, the first-amp rule that the
// corpus measurer used got 10 chains' instrument wrong and dropped a further
// 183 from the grammar entirely, for a subcategory it did not recognise where
// the other two rules answered guitar. 193 of 3,804 chains with an amplifier,
// which is 5.1%.
//
// A subcategory that is neither is a guitar amplifier here. Line 6 tag some
// amps Preamp and leave others blank, and none of those is a bass rig: the
// bass tag is the specific claim and everything else is the default, which is
// what the compiler and the measuring guard both already assumed.
//
// NoAmp for a chain with no amplifier. A caller needing a word for one says so
// itself: the compiler answers Guitar because a rig's `instrument` field must
// hold something, and the measuring guard answers nothing because refusing a
// delay for being the wrong instrument would be wrong.
func InstrumentFor(
	c Plan,
	l BlockLookup,
) string {
	out := NoAmp

	for _, b := range c.Blocks {
		blk, known := l.Block(b.Model)
		if !known || blk.Category != catalog.CategoryAmp {
			continue
		}

		if blk.Subcategory == "Bass" {
			return Bass
		}

		out = Guitar
	}

	return out
}

// AmpAt is where the first amplifier sits in a chain, or -1 for none.
//
// The first one, deliberately, and it is a different question from
// InstrumentFor: what an amplifier is for is decided by all of them, and what
// a chain is ordered around is the one everything else sits before or after.
func AmpAt(
	c Plan,
	l BlockLookup,
) int {
	for i, b := range c.Blocks {
		blk, known := l.Block(b.Model)
		if known && blk.Category == catalog.CategoryAmp {
			return i
		}
	}

	return -1
}
