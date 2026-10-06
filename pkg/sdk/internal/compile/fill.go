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

package compile

import (
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
)

// nearUniversal is how common a kind of block must be before one is added to a
// chain that did not ask for it.
//
// Set high on purpose. A compressor in 88% of bass chains is a convention, and
// leaving it out produces something nobody would recognise. Drive in 62% is a
// choice, and making it silently would be this tool having opinions it cannot
// justify. A rig naming a pedal always gets it, whatever the figure.
const nearUniversal = 0.75

// Added records a block the chain did not ask for, and why it is there.
//
// A generated rig is a set of decisions, and one made on the player's behalf
// has to be visible before they plug in rather than discovered after.
type Added struct {
	Block  catalog.Block
	Reason string
	Share  float64
}

// fill adds the blocks a chain of this kind almost always has.
//
// Only categories missing from the chain are considered, and only those the
// corpus shows to be near-universal for this instrument. What gets added is
// the model players of this instrument reach for most, which is the only
// defensible choice when nobody named one.
func fill(
	blocks []catalog.Block,
	said []*wanted,
	cat *catalog.Catalog,
	stats *corpus.Stats,
	instrument string,
) ([]catalog.Block, []*wanted, []Added) {
	if stats == nil {
		return blocks, said, nil
	}

	g, ok := stats.Grammar[instrument]
	if !ok {
		return blocks, said, nil
	}

	present := map[catalog.Category]bool{}
	for _, b := range blocks {
		present[b.Category] = true
	}

	var added []Added

	for _, c := range sortedByShare(g) {
		s := g.Categories[c]

		share := s.Frequency(g.Chains)
		if present[c] || share < nearUniversal {
			continue
		}

		pick, ok := commonest(cat, stats, c, instrument, "")
		if !ok {
			continue
		}

		blocks, said = insert(blocks, said, pick, s.BeforeAmp() >= 0.5)
		added = append(added, Added{
			Block: pick, Share: share,
			Reason: "almost every chain has one",
		})
	}

	return blocks, said, added
}

// insert places a block on the correct side of the amp.
//
// Position is not decoration: drive ahead of an amp overdrives its input,
// drive after it does something else entirely.
//
// A resolved chain always holds an amp — a rig cannot omit one — so the
// index is found rather than guarded against.
func insert(
	blocks []catalog.Block,
	said []*wanted,
	b catalog.Block,
	beforeAmp bool,
) ([]catalog.Block, []*wanted) {
	if !beforeAmp {
		return append(blocks, b), append(said, nil)
	}

	at := 0

	for i, existing := range blocks {
		if existing.Category == catalog.CategoryAmp {
			at = i

			break
		}
	}

	out := make([]catalog.Block, 0, len(blocks)+1)
	out = append(out, blocks[:at]...)
	out = append(out, b)
	out = append(out, blocks[at:]...)

	return out, spliced(said, at)
}

// spliced opens a gap at the same index the blocks did, holding no settings.
//
// **The parallel slice is the bug this exists to prevent, and it was a live
// one.** `said` is what a rig wrote for each of its own blocks, built index by
// index against the chain the person typed. A block spliced in ahead of the
// amplifier shifts the amplifier and everything after it one to the right, and
// `saidKnobs` pairs `said[i]` with `blocks[i]`, so without this every setting
// after the insertion point lands on its neighbour.
//
// It does not reliably fail, which is why it survived. `knobWords` maps a word
// like `level` or `mix` to several parameter names that different categories
// share, so a value meant for an amplifier's Master can be written to a
// compressor's Level with no error at all: a preset that measures fine and is
// quietly not what was asked for. Where the names do not coincide it fails
// instead with a message naming the position the person typed rather than the
// block it actually checked.
//
// Bootsy Collins' rig is the one that shows it in the shipped data. It names a
// filter then an amplifier, the corpus fills a compressor in ahead of the
// amplifier because 88% of chains have one, and the amplifier arrives at index
// 2 while `said[1]` still describes it. That rig is unharmed only because its
// amplifier entry carries no settings.
//
// A gap past the end is appended, because `said` is never longer than `blocks`
// and a shorter one means the caller added blocks of its own.
func spliced(
	said []*wanted,
	at int,
) []*wanted {
	if at >= len(said) {
		return append(said, nil)
	}

	out := make([]*wanted, 0, len(said)+1)
	out = append(out, said[:at]...)
	out = append(out, nil)

	return append(out, said[at:]...)
}

// commonest returns the model of a category that chains for this instrument
// held most often.
//
// Counted per instrument where the statistics carry it. A total across every
// preset is a guitar figure, and on it bass gets the LA Studio Comp, which
// guitar players use most and bass players do not. Statistics measured before
// the count existed fall back to that total, which is what they have.
//
// Ties break on the identifier so a chain does not change between runs for
// reasons nobody chose.
// holds, where it is not empty, is a control the block must carry. Filling a
// category wants whatever players reach for; answering a word wants a block
// that can answer it, and the commonest equaliser is no use to `mid-forward`
// if it has no Mid.
func commonest(
	cat *catalog.Catalog,
	stats *corpus.Stats,
	c catalog.Category,
	instrument string,
	holds string,
) (catalog.Block, bool) {
	var (
		best catalog.Block
		uses int
		set  bool
	)

	byInstrument := stats.Grammar[instrument].Models

	for id, ms := range stats.Models {
		b, known := cat.Block(id)
		if !known || b.Category != c || catalog.NeedsUserIR(id) {
			continue
		}

		if holds != "" {
			if _, carries := b.Params[holds]; !carries {
				continue
			}
		}

		// A block whose cost was inferred rather than stated cannot be
		// budgeted honestly, and validation refuses one. Choosing it here
		// would produce a chain that is rejected a moment later, blaming a
		// block nobody asked for.
		if !b.DSP.Prov.Trusted() {
			continue
		}

		if !suits(b, instrument) {
			continue
		}

		n := ms.Uses
		if len(byInstrument) > 0 {
			// A model no chain for this instrument held is not what its
			// players reach for, however popular it is elsewhere.
			if n = byInstrument[id]; n == 0 {
				continue
			}
		}

		if !set || n > uses || (n == uses && id < best.ID) {
			best, uses, set = b, n, true
		}
	}

	return best, set
}

// suits reports whether a block is eligible for an instrument.
//
// Line 6 tags amps and cabinets Guitar or Bass; everything else is untagged
// and available to either.
func suits(
	b catalog.Block,
	instrument string,
) bool {
	if b.Subcategory == "" || !isInstrumentTag(b.Subcategory) {
		return true
	}

	return strings.EqualFold(b.Subcategory, instrument)
}

// sortedByShare orders categories by how often they appear, so the chain is
// filled with the most conventional blocks first and a budget runs out on the
// least important.
func sortedByShare(
	g corpus.Grammar,
) []catalog.Category {
	out := make([]catalog.Category, 0, len(g.Categories))
	for c := range g.Categories {
		out = append(out, c)
	}

	sort.Slice(out, func(i, j int) bool {
		a, b := g.Categories[out[i]], g.Categories[out[j]]
		if a.Chains != b.Chains {
			return a.Chains > b.Chains
		}

		return out[i] < out[j]
	})

	return out
}
