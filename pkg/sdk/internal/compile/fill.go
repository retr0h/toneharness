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
// Set high on purpose. A compressor in 86% of bass chains is a convention, and
// leaving it out produces something nobody would recognise. Drive in 63% is a
// choice, and making it silently would be this tool having opinions it cannot
// justify. A rig naming a pedal always gets it, whatever the figure.
//
// It sits in a gap in the evidence rather than at a round number. Bass runs
// 63.3% for drive and then 86.4% for the compressor with nothing between;
// guitar runs 76.7% for delay and then 55.0% for modulation.
const nearUniversal = 0.75

// usuallyOn is how often a kind of block is running, among the chains that hold
// one, before one is added switched on rather than bypassed.
//
// **Two questions were being answered by one number.** Whether a player would
// have this block and whether they would use it are different facts, and the
// corpus disagrees about them sharply. Of the guitar chains that hold a delay,
// 61.6% run it; of those holding a drive, 47.3%. Both clear the presence gate,
// so both were being added and switched on, which decided for the player in
// exactly the way the comment above exists to prevent. Nobody noticed because
// the figure that would have shown it was not counted.
//
// This one also sits in a gap, and a wide one: across both instruments nothing
// measured falls between 61.6% and 82.1%. Above it are the conventions, a
// compressor at 87.0% on bass and a reverb at 92.1% on guitar. Below it are the
// pedals, and a wah is the clearest of them at 3.8% on guitar, present in one
// chain in six and running in one in twenty-six.
//
// A block that does not clear it is still added, bypassed. It costs the chain
// its DSP and nothing else, the device's own footswitches toggle it, and the
// player decides. That is the answer to a block the corpus says to carry and
// not to commit to.
const usuallyOn = 0.75

// Added records a block the chain did not ask for, and why it is there.
//
// A generated rig is a set of decisions, and one made on the player's behalf
// has to be visible before they plug in rather than discovered after.
type Added struct {
	Block  catalog.Block
	Reason string
	Share  float64
	// Bypassed says the block arrived switched off, because the chains that
	// carry one mostly do not run it, and Running is the share that do.
	Bypassed bool
	Running  float64
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

	// An empty chain is a statement rather than an omission. 102 presets in the
	// corpus hold no block, and they are the MIDI remotes that drive Spotify or
	// Pro Tools from the footswitches and the blank templates people build from.
	// Filling one turns a remote into a reverb, a delay and an overdrive nobody
	// asked for, which is what this did the first time an empty chain was allowed.
	//
	// A chain naming one block is a different thing: somebody said an amplifier
	// and the convention around one is what the corpus is for.
	if len(blocks) == 0 {
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

		// Switched on only where the chains that carry one run it. Otherwise
		// it arrives bypassed, which carries the evidence for its presence
		// without making a decision the evidence does not support.
		running := s.Running()
		bypassed := running < usuallyOn

		blocks, said = insert(blocks, said, pick, s.BeforeAmp() >= 0.5, bypassed)

		why := "almost every chain has one"
		if bypassed {
			why = "almost every chain carries one and most leave it off, " +
				"so this arrives bypassed"
		}

		added = append(added, Added{
			Block: pick, Share: share, Reason: why,
			Bypassed: bypassed, Running: running,
		})
	}

	return blocks, said, added
}

// insert places a block on the correct side of the amp.
//
// Position is not decoration: drive ahead of an amp overdrives its input,
// drive after it does something else entirely.
//
// Index zero where the chain holds no amplifier, which is what a chain of
// effects is. Never an empty chain: filling one is refused above, so the only way
// here is a chain somebody put something in.
func insert(
	blocks []catalog.Block,
	said []*wanted,
	b catalog.Block,
	beforeAmp bool,
	bypassed bool,
) ([]catalog.Block, []*wanted) {
	// A wanted rather than nil, so the bypass travels with the block. Nil is
	// what a block nobody asked for used to carry, and it could say nothing.
	var with *wanted
	if bypassed {
		with = &wanted{bypassed: true}
	}

	if !beforeAmp {
		return append(blocks, b), append(said, with)
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

	return out, spliced(said, at, with)
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
	with *wanted,
) []*wanted {
	if at >= len(said) {
		return append(said, with)
	}

	out := make([]*wanted, 0, len(said)+1)
	out = append(out, said[:at]...)
	out = append(out, with)

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

	if !set {
		return best, false
	}

	return asNamed(cat, best, c, instrument), true
}

// asNamed is the copy of a block that a document naming it would get back.
//
// The corpus counts models and a document names gear, and the two do not have
// to agree: the commonest 8x10 in bass chains is the legacy HD2_Cab8x10SVBeast,
// Line 6 document it as emulating an unqualified 8x10" Ampeg SVT, and that name
// resolves to the mic'd HD2_CabMicIr_8x10SVTAV because it is also the front of
// that cabinet's name and [catalog.Block.Preferred] puts the mic'd one first.
// So eight shipped rigs recorded a cabinet their own text does not resolve to,
// and a document that does not describe the preset it came from is the one
// thing a resolved rig is for.
//
// Resolved through the same function a document's name goes through, rather
// than by comparing families here, so agreeing is what the code does rather
// than something it is checked for.
//
// It is also the better cabinet to land on. 86 of the 89 cabinets in this
// instrument's corpus chains are legacy models, so popularity here is a vote
// about what people clicked years ago rather than about what the device offers:
// the mic'd copy costs a third of the DSP and carries Mic, Position, Distance
// and Angle, and the microphone alone moves a cabinet's centroid between 21 and
// 48Hz.
//
// The block itself where its own name does not resolve, which is a name nothing
// can ask for rather than a reason to drop the block.
func asNamed(
	cat *catalog.Catalog,
	best catalog.Block,
	c catalog.Category,
	instrument string,
) catalog.Block {
	name := best.Gear()
	if name == "" {
		return best
	}

	as, err := findGear(cat, name, c, instrument, nil)
	if err != nil {
		return best
	}

	return as
}

// suits reports whether a block is eligible for an instrument.
//
// Line 6 say it two ways and this reads both. Amplifiers carry a subcategory of
// "Guitar" or "Bass". Cabinets carry "Single, Dual", which counts microphones
// and says nothing about the instrument, and their instrument is a separate
// mark in the app's own catalog: see [catalog.Block.Bass].
//
// Unread until 2026-10-19, so this said yes to every cabinet on the device for
// every instrument. What saved it was the corpus itself, which is thin here:
// three cabinets across this instrument's bass chains, all of them Ampegs. A
// fourth would have been enough to put a 4x12 guitar cabinet in a bass rig.
//
// Everything else is untagged and unmarked, and available to either, which is
// the right answer for a reverb.
func suits(
	b catalog.Block,
	instrument string,
) bool {
	// A cabinet is for bass exactly when Line 6 mark it so. Two-valued because
	// every cabinet on the device is for one instrument or the other, and 16 of
	// the 133 carry the mark.
	if b.Category == catalog.CategoryCab {
		return b.Bass == strings.EqualFold(instrument, bass)
	}

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
