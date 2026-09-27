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
	"errors"
	"fmt"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// Resolve turns a rig into a chain for the device the catalog describes.
//
// A rig is already an ordered chain of roles, so this walks it rather than
// reasoning about what an amplifier is: the ordering decision was made by
// whoever wrote the rig, and second-guessing it here would silently move
// somebody's pedals.
//
// What a rig does not say, the corpus fills, but only where a kind of block is
// near-universal, and never quietly. See fill.
//
// The intent is the ask beside the rig, and the zero value is a rig with no ask
// behind it. That is not a degraded build: a rig off disk carries settings
// somebody already applied, so there is nothing left for a word to decide.
func Resolve(
	spec rig.Spec,
	intent Intent,
	cat *catalog.Catalog,
	stats *corpus.Stats,
) (plan.Plan, []Added, []Moved, error) {
	instrument := string(spec.Instrument)

	blocks := make([]catalog.Block, 0, len(spec.Chain)+1)

	// What the rig said about each block, kept beside it. A chain gains
	// blocks on the way through: an implied cabinet, whatever the corpus
	// fills. Those are nobody's words, so they take no settings.
	said := make([]*rig.Settings, 0, len(spec.Chain)+1)

	// A cabinet is the one miss worth recovering from: Line 6 do not describe
	// every cabinet in terms of real gear, and an amplifier already names the
	// one it was voiced with. Every other role fails, because substituting an
	// amplifier is not a detail.
	var (
		missed    string
		missedErr error
	)

	sub := []Added(nil)

	for _, entry := range spec.Chain {
		b, err := findGear(cat, entry.Gear, categoryFor(entry.Role), instrument)

		// The rig named gear this device cannot do and said what to put
		// there instead. It goes on naming the real thing, so the day the
		// real thing is modelled the substitute is deleted and nothing else
		// in the file moves.
		if err != nil && entry.Substitute != nil && errors.Is(err, ErrNoSuchGear) {
			var stand catalog.Block

			stand, err = findGear(
				cat, entry.Substitute.Gear, categoryFor(entry.Role), instrument)
			if err != nil {
				return plan.Plan{}, nil, nil, fmt.Errorf(
					"%q stands in for %q, and nothing emulates it either: %w",
					entry.Substitute.Gear, entry.Gear, err)
			}

			sub = append(sub, Added{
				Block: stand,
				Reason: fmt.Sprintf(
					"nothing emulates %q — the rig says to use %q",
					entry.Gear, entry.Substitute.Gear),
			})

			b = stand
		}

		if err != nil {
			if entry.Role != rig.RoleCab || !errors.Is(err, ErrNoSuchGear) {
				return plan.Plan{}, nil, nil, err
			}

			missed, missedErr = entry.Gear, err

			continue
		}

		blocks = append(blocks, b)
		said = append(said, entry.Settings)
	}

	// A rig naming an amplifier and no cabinet gets the one Line 6 voiced it
	// with, which is a better answer than picking arbitrarily.
	if cab := impliedCab(cat, blocks); cab != nil {
		if missed != "" {
			sub = append(sub, Added{
				Block: *cab,
				Reason: fmt.Sprintf(
					"nothing emulates %q — used the amp's own pairing", missed),
			})
		}

		blocks = append(blocks, *cab)
		said = append(said, nil)
	} else if missed != "" {
		// Nothing to fall back to, so the rig named a cabinet that cannot be
		// built and saying so is the only honest answer.
		return plan.Plan{}, nil, nil, missedErr
	}

	blocks, added := fill(blocks, cat, stats, instrument)

	// After fill, because what a chain of this kind usually has is the wider
	// claim and should not be displaced by one word. Before specFor, because
	// a block arriving later would miss the corpus medians and start on
	// catalog defaults.
	blocks, asked := demand(blocks, cat, stats, intent, instrument)
	added = append(added, asked...)

	built := specFor(spec, intent, blocks, stats)

	// After the corpus has had its say, because a term is an opinion about
	// where players land rather than a replacement for knowing.
	moved := worded(intent, blocks, built, stats)

	// Last, over the corpus medians and over whatever a word
	// moved: a number somebody wrote down is the most explicit thing in the
	// rig, and the only one that says exactly what they meant.
	if err := saidKnobs(built.Blocks, blocks, said); err != nil {
		return plan.Plan{}, nil, nil, err
	}

	// Not checked here. check reads a plan's target, footswitches and
	// controllers, and this builds none of them: a rig has nowhere to state
	// one. Lower checks, which is where a plan arrives from a file.
	return built, append(sub, added...), moved, nil
}

// worded moves whatever in the chain answers for the words the ask used.
//
// After the corpus has had its say, because a term is an opinion about where
// players land rather than a replacement for knowing.
func worded(
	intent Intent,
	blocks []catalog.Block,
	built plan.Plan,
	stats *corpus.Stats,
) []Moved {
	terms := termsOf(intent.Words)
	if len(terms) == 0 {
		return nil
	}

	return move(blocks, built, terms, stats)
}

// categoryFor maps a rig's role onto the catalog's own grouping.
//
// They are deliberately the same words, so this is a conversion rather than a
// translation. A rig that says `delay` and names an amplifier is describing
// something the device cannot do, and failing to find it is the right answer.
//
// `other` is the exception, because the two vocabularies mean different things
// by it. In a rig it means the author did not say what the gear does; in the
// catalog it is Line 6's residual bucket. Reading the first as the second
// would search a handful of blocks for gear that is almost certainly filed
// somewhere else, so an unnamed role searches everything.
func categoryFor(
	role rig.Role,
) catalog.Category {
	if role == "" || role == rig.RoleOther {
		return ""
	}

	return catalog.Category(role)
}

// impliedCab returns the cabinet an amplifier names, when the chain has none.
//
// Line 6 state a cablink for most amps: the cabinet the model was voiced
// with. A chain that ended up without a cabinet is better served by that than
// by whatever the catalog happens to list first.
func impliedCab(
	cat *catalog.Catalog,
	blocks []catalog.Block,
) *catalog.Block {
	for _, b := range blocks {
		if b.Category == catalog.CategoryCab {
			return nil
		}
	}

	for _, b := range blocks {
		if b.Category != catalog.CategoryAmp || b.CabLink == "" {
			continue
		}

		if cab, ok := cat.Block(b.CabLink); ok {
			return &cab
		}
	}

	return nil
}

// gear finds the model a chain entry names, the way this package resolves
// every other one.
//
// Exported because lowering a rig into a preset asks the same question and
// asked it differently: a map range that took whatever matched first, which
// answered a different model each run and would answer with a cabinet for an
// amplifier. Two resolvers cannot both be right about which Ampeg SVT is
// meant.
func gear(
	cat *catalog.Catalog,
	gear string,
	role rig.Role,
	instrument string,
) (catalog.Block, error) {
	return findGear(cat, gear, categoryFor(role), instrument)
}

// findGear returns the block emulating the named gear.
//
// Matching is on what Line 6 says a model is based on, because that is the
// only field naming gear a person recognises. An instrument narrows the search
// to the half of the catalog Line 6 tags that way, which is what keeps a bass
// request out of six hundred guitar models.
func findGear(
	cat *catalog.Catalog,
	gear string,
	category catalog.Category,
	instrument string,
) (catalog.Block, error) {
	want := strings.ToLower(gear)

	var best catalog.Block

	found := false

	for _, b := range cat.Blocks {
		if !eligible(b, want, category, instrument) {
			continue
		}

		if !found || closer(b, best) {
			best, found = b, true
		}
	}

	if !found {
		return catalog.Block{}, &NoSuchGearError{
			Gear: gear, Kind: kindOf(category), Instrument: instrument,
		}
	}

	return best, nil
}

// closer reports whether a is the better answer than b for the same query.
//
// Shorter wins: "Ampeg SVT" should find the SVT rather than the SVT-4 Pro, and
// a shorter description is the closer one. Where two are equally close the
// identifier decides, so the choice is the same every run — an ambiguous
// request such as "Ampeg SVT", which names neither the normal nor the bright
// channel, must not resolve differently because the catalog was regenerated.
//
// The chosen block is reported when a preset is built, so an ambiguity a
// person cares about is visible and can be settled by naming the channel in
// the rig.
func closer(
	a, b catalog.Block,
) bool {
	if len(a.BasedOn) != len(b.BasedOn) {
		return len(a.BasedOn) < len(b.BasedOn)
	}

	return a.ID < b.ID
}

// eligible reports whether a block could be the gear being looked for.
func eligible(
	b catalog.Block,
	want string,
	category catalog.Category,
	instrument string,
) bool {
	if catalog.NeedsUserIR(b.ID) {
		return false
	}

	if !b.Matches(want) {
		return false
	}

	if category != "" && b.Category != category {
		return false
	}

	// Line 6 tags amps and cabinets Guitar or Bass; everything else is
	// untagged and available to either.
	if instrument != "" && b.Subcategory != "" && isInstrumentTag(b.Subcategory) {
		return strings.EqualFold(b.Subcategory, instrument)
	}

	return true
}

// isInstrumentTag reports whether a subcategory names an instrument rather
// than a routing shape such as "Mono, Stereo".
func isInstrumentTag(
	sub string,
) bool {
	return strings.EqualFold(sub, "guitar") || strings.EqualFold(sub, "bass")
}

// kindOf names a category for an error message.
func kindOf(
	c catalog.Category,
) string {
	if c == "" {
		return "block"
	}

	return string(c)
}

// specFor lays blocks out as a chain the device can represent.
func specFor(
	spec rig.Spec,
	intent Intent,
	blocks []catalog.Block,
	stats *corpus.Stats,
) plan.Plan {
	// The name is what the device prints on its screen, and "Mike Dirnt" is
	// what somebody wants to read there. That is the subject, which lives on
	// the ask, so the ask is asked first.
	//
	// The rig's identifier where there is no ask. A rig read off disk has no
	// subject to be named after, and its identifier is the one name it has of
	// its own, which beats a blank heading.
	name := intent.Name
	if name == "" {
		name = spec.ID
	}

	out := plan.Plan{
		Name:   name,
		Blocks: make([]plan.Block, 0, len(blocks)),
	}

	for i, b := range blocks {
		out.Blocks = append(out.Blocks, plan.Block{
			Model:   b.ID,
			Params:  settings(b, stats),
			DSP:     0,
			Pos:     i,
			Enabled: true,
		})
	}

	return out
}

// Fit reports whether a chain fits the device, moving blocks to the second
// processor when the first fills up.
//
// Line 6 states each block's cost as a percentage of one processor, so a chain
// that overflows is not a preset anyone can load.
func Fit(
	spec plan.Plan,
	cat *catalog.Catalog,
	lim plan.Limits,
) plan.Plan {
	used, path := 0.0, 0

	for i := range spec.Blocks {
		b, ok := cat.Block(spec.Blocks[i].Model)
		if !ok {
			continue
		}

		cost := b.DSP.Mono
		if b.Stereo {
			cost = b.DSP.Stereo
		}

		// Filled in order, one processor at a time. Moving a block back to a
		// processor an earlier one overflowed would reorder somebody's chain,
		// which is a different preset rather than the same one laid out
		// differently.
		for used+cost > lim.ChipCeiling && path+1 < lim.Paths {
			path++
			used = 0
		}

		// Nowhere left to put it. Leaving the block where it is lets
		// validation reject the chain, which is the honest answer; writing it
		// onto a processor with no room would produce a file the device
		// refuses.
		if used+cost > lim.ChipCeiling {
			continue
		}

		spec.Blocks[i].DSP = path
		used += cost
	}

	return renumber(spec)
}

// renumber gives each processor a contiguous run of positions.
func renumber(
	spec plan.Plan,
) plan.Plan {
	next := map[int]int{}

	for i := range spec.Blocks {
		dsp := spec.Blocks[i].DSP
		spec.Blocks[i].Pos = next[dsp]
		next[dsp]++
	}

	return spec
}

// saidKnobs puts each entry's settings onto the block it resolved to.
//
// Blocks the rig did not ask for are at the end of the chain and have no
// settings beside them, so the lists run out together.
func saidKnobs(
	built []plan.Block,
	blocks []catalog.Block,
	said []*rig.Settings,
) error {
	out := []error(nil)

	for i, set := range said {
		if set == nil || i >= len(built) {
			continue
		}

		out = append(out, setKnobs(
			built[i].Params, blocks[i], set, fmt.Sprintf("chain[%d].settings", i)))
	}

	return errors.Join(out...)
}
