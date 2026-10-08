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
	id string,
	spec rig.Spec,
	intent Intent,
	cat *catalog.Catalog,
	stats *corpus.Stats,
) (plan.Plan, []Added, []Moved, Compensated, error) {
	instrument := string(spec.Instrument)

	blocks := make([]catalog.Block, 0, len(spec.Chain)+1)

	// What the rig said about each block, kept beside it. A chain gains
	// blocks on the way through: an implied cabinet, whatever the corpus
	// fills. Those are nobody's words, so they take no settings.
	wants := make([]*wanted, 0, len(spec.Chain)+1)

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
		b, err := findGear(
			cat, entry.Gear, categoryFor(entry.Role), instrument, stated(entry))

		// The rig named gear this device cannot do and said what to put
		// there instead. It goes on naming the real thing, so the day the
		// real thing is modelled the substitute is deleted and nothing else
		// in the file moves.
		if err != nil && entry.Substitute != nil && errors.Is(err, ErrNoSuchGear) {
			var stand catalog.Block

			// The controls too: a substitute stands in for the gear and the
			// document's values are still the values it has to take.
			stand, err = findGear(
				cat, entry.Substitute.Gear, categoryFor(entry.Role), instrument,
				stated(entry))
			if err != nil {
				return plan.Plan{}, nil, nil, Compensated{}, fmt.Errorf(
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
				return plan.Plan{}, nil, nil, Compensated{}, err
			}

			missed, missedErr = entry.Gear, err

			continue
		}

		blocks = append(blocks, b)
		wants = append(wants, &wanted{
			words: entry.Settings, stated: entry.Controls, entry: &entry,
		})
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
		wants = append(wants, nil)
	} else if missed != "" {
		// Nothing to fall back to, so the rig named a cabinet that cannot be
		// built and saying so is the only honest answer.
		return plan.Plan{}, nil, nil, Compensated{}, missedErr
	}

	// Before fill and before demand, because the compensating word is an ask
	// like any other from here on: demand will pull a compressor into a chain
	// that has none because this word wants one, and the corpus decides how
	// far the term travels.
	held := compensate(
		intent.Attack, intent.Playing,
		spokenFor(intent.Words, attackAxis), spokenFor(intent.Words, midsAxis))
	intent.Words = append(intent.Words, held.Words...)

	blocks, wants, added := fill(blocks, wants, cat, stats, instrument)

	// After fill, because what a chain of this kind usually has is the wider
	// claim and should not be displaced by one word. Before specFor, because
	// a block arriving later would miss the corpus medians and start on
	// catalog defaults.
	blocks, wants, asked := demand(blocks, wants, cat, stats, intent, instrument)
	added = append(added, asked...)

	built := specFor(id, intent, blocks, wants, stats)

	// After the corpus has had its say, because a term is an opinion about
	// where players land rather than a replacement for knowing.
	moved := worded(intent, blocks, built, stats)

	// Last, over the corpus medians and over whatever a word
	// moved: a number somebody wrote down is the most explicit thing in the
	// rig, and the only one that says exactly what they meant.
	if err := saidKnobs(built.Blocks, blocks, wants); err != nil {
		return plan.Plan{}, nil, nil, Compensated{}, err
	}

	// After the words, so a value beats one. A word is a request and a value is
	// an answer: `setKnobs` writes over whatever is there, so running the words
	// second let `drive: 0.5` overwrite a `Drive` somebody had set by ear. The two
	// naming one control is allowed and the value wins.
	if err := statedControls(built.Blocks, blocks, wants); err != nil {
		return plan.Plan{}, nil, nil, Compensated{}, err
	}

	// The members a rig states beside its chain are not folded into the plan.
	// They reach the file through ApplyMembers, which merges them over what the
	// preset underneath holds, and a plan's own `device:` is what a lift fills.
	// Both writing them would be two places setting one control.
	//
	// Not checked here. check reads a plan's target, footswitches and
	// controllers, and this builds none of them from musical intent: `sections`
	// and `moves` are turned into them later, by Sections and Moves. Lower is
	// where a plan arriving from a file is checked.
	return built, append(sub, added...), moved,
		Compensated{Terms: termsIn(held.Words), Said: held.Said}, nil
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
// holds names the controls the document states for it, which is what decides
// between models sharing a name.
func gear(
	cat *catalog.Catalog,
	gear string,
	role rig.Role,
	instrument string,
	holds []string,
) (catalog.Block, error) {
	return findGear(cat, gear, categoryFor(role), instrument, holds)
}

// findGear returns the block emulating the named gear.
//
// Matching is on what Line 6 says a model is based on, because that is the
// only field naming gear a person recognises. An instrument narrows the search
// to the half of the catalog Line 6 tags that way, which is what keeps a bass
// request out of six hundred guitar models.
//
// Narrows rather than decides. A rig that names gear has made a claim, and the
// instrument tag is Line 6 saying who a model is sold to rather than what it is:
// they tag the Fender Bassman as a guitar amplifier, which is what it is used
// for now and not what Fender built it for. Refusing a named Bassman to a
// bassist is the filter answering a question nobody asked.
//
// So the tagged half is searched first and the rest only if that finds nothing.
// Every name that resolved before resolves to the same model, and a name that
// resolved to nothing can now reach the model that does emulate it.
func findGear(
	cat *catalog.Catalog,
	gear string,
	category catalog.Category,
	instrument string,
	holds []string,
) (catalog.Block, error) {
	// The controls first, where the document states any. 661 models answer to
	// only 468 names, so a name is often several models and the shortest one wins
	// by default. That is right for a rig somebody typed, where the name is all
	// there is, and wrong for one read off a preset: three models are called
	// `1x12 US Deluxe` and only one carries a `Pan` and a `Delay`, so the default
	// chose a model the preset's own values do not fit and the build refused them.
	//
	// 60% of the preset corpus failed to rebuild that way, which is the single
	// biggest reason a document could not be handed to somebody else.
	// The controls first, where the document states any.
	if len(holds) > 0 {
		if b, found := anyInstrument(cat, gear, category, instrument, holds, true); found {
			return b, nil
		}
	}

	// Then the closest, which ranks by how many of those controls a model carries
	// before it falls back to the name. One rung rather than two, and that is the
	// bug it fixes: dropping straight from "every control" to "no controls at all"
	// threw away every control that did match.
	//
	// A document naming `Teletronix LA-2A` with the LA Studio Comp's six controls
	// and one more besides matched six of seven on the LA Studio Comp and one of
	// seven on a legacy Tube Comp carrying only a level. Nothing carried all
	// seven, so the controls were ignored and the shorter name won: the Tube
	// Comp, which cannot hold the document's own values.
	//
	// Falling back rather than failing is still right. The error belongs to the
	// gear a person named, not to a control name they got slightly wrong, and a
	// rig nobody lifted off a preset states no controls and reaches this rung
	// with nothing to rank by.
	if b, found := anyInstrument(cat, gear, category, instrument, holds, false); found {
		return b, nil
	}

	return catalog.Block{}, &NoSuchGearError{
		Gear: gear, Kind: kindOf(category), Instrument: instrument,
	}
}

// anyInstrument looks for the gear among an instrument's blocks, then among all
// of them.
//
// Line 6 tag amps and cabinets Guitar or Bass and leave everything else untagged,
// so a bass rig naming a pedal finds it on the second pass.
func anyInstrument(
	cat *catalog.Catalog,
	gear string,
	category catalog.Category,
	instrument string,
	holds []string,
	every bool,
) (catalog.Block, bool) {
	if b, found := nearest(cat, gear, category, instrument, holds, every); found {
		return b, true
	}

	// No guard for an empty instrument. The contract requires one on every rig, so
	// the second pass is never the same search twice, and a guard against it would
	// be a branch nothing can reach.
	return nearest(cat, gear, category, "", holds, every)
}

// stated names the controls a chain entry writes down.
//
// Which model a name means is a question the values answer, where there are any.
func stated(
	entry rig.ChainEntry,
) []string {
	if entry.Controls == nil {
		return nil
	}

	out := make([]string, 0, len(*entry.Controls))
	for name := range *entry.Controls {
		out = append(out, name)
	}

	return out
}

// nearest is the closest block emulating the named gear, among those an
// instrument leaves eligible. Empty takes the whole catalog.
//
// holds, where it is not empty, names the controls the document states. every
// requires a block to carry all of them; without it they rank, and a block
// carrying more of them beats a closer name.
//
// Ranked rather than filtered on the second pass, so one control nothing carries
// does not discard the evidence of the ones that do. Where holds is empty every
// block scores nothing and the name decides, which is the whole of what a rig
// nobody lifted off a preset asks for.
func nearest(
	cat *catalog.Catalog,
	gear string,
	category catalog.Category,
	instrument string,
	holds []string,
	every bool,
) (catalog.Block, bool) {
	want := strings.ToLower(gear)

	var best catalog.Block

	bestHas := 0
	found := false

	for _, b := range cat.Blocks {
		if !eligible(b, want, category, instrument) {
			continue
		}

		has := countOf(b, holds)
		if every && has < len(holds) {
			continue
		}

		if !found || has > bestHas || (has == bestHas && closer(b, best)) {
			best, bestHas, found = b, has, true
		}
	}

	return best, found
}

// countOf is how many of the named controls a block carries.
//
// A count rather than a yes or no, because the two questions asked of it are
// different: whether a model fits the document exactly, and which model fits it
// best when none fits exactly. A model missing one control is a model the
// document's own values will be refused against, so an exact fit is still
// preferred over every near one.
func countOf(
	b catalog.Block,
	holds []string,
) int {
	out := 0

	for _, name := range holds {
		if _, has := b.Params[name]; has {
			out++
		}
	}

	return out
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

	// Then the family, because which of two equally-named models a name means
	// is a decision and the identifier is not one.
	if x, y := a.Preferred(), b.Preferred(); x != y {
		return x < y
	}

	return a.ID < b.ID
}

// eligible reports whether a block could be the gear being looked for.
//
// A block that plays an impulse response is eligible here, which is the
// difference between resolving a name and choosing one. Choosing an IR block for
// a chain nobody asked for would put a block in a preset that plays nothing until
// somebody loads a file, so `commonest` refuses to pick one. Resolving `IR 1024`
// is a document saying which block it means, and refusing that made 31.4% of the
// preset corpus impossible to rebuild after being read: 90% of those failures were
// an impulse response the preset named and this would not give back.
//
// What the IR itself is stays in the preset, where the device keeps it: the block
// names the slot and `irUuidTable` names the file in it.
func eligible(
	b catalog.Block,
	want string,
	category catalog.Category,
	instrument string,
) bool {
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
	id string,
	intent Intent,
	blocks []catalog.Block,
	said []*wanted,
	stats *corpus.Stats,
) plan.Plan {
	// The name is what the device prints on its screen, and "Mike Dirnt" is
	// what somebody wants to read there. That is the subject, which lives on
	// the ask, so the ask is asked first.
	//
	// The document's identifier where there is no ask. A rig read off disk has
	// no subject to be named after, and the name of the document it sits in
	// beats a blank heading.
	name := intent.Name
	if name == "" {
		name = id
	}

	out := plan.Plan{
		Name:   name,
		Blocks: make([]plan.Block, 0, len(blocks)),
	}

	for i, b := range blocks {
		// What the document said about this block, where it said anything. A
		// block the corpus added has no entry and takes the chain's own order and
		// the device's defaults, which is what every block did before a lift
		// could state these.
		var entry rig.ChainEntry
		if i < len(said) && said[i] != nil && said[i].entry != nil {
			entry = *said[i].entry
		}

		out.Blocks = append(out.Blocks, plan.Block{
			Model:   b.ID,
			Params:  settings(b, stats),
			DSP:     0,
			Pos:     at(entry.Position, i),
			Enabled: playing(entry),
			Attrs:   attrsFrom(entry),
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

// renumber gives each processor a contiguous run of keys to file its blocks under.
//
// Only where they collide. A key is what the preset writer names the entry with,
// and a chain built from gear has none to start with, so this hands out 0, 1, 2 in
// order. A chain lifted off a preset arrived with the keys the device used, and
// renumbering those moves every block in the file to a different entry: a preset
// exported and built again had its amplifier filed where its drive had been.
//
// Collisions are what has to go. Two blocks cannot share a key, and `fill` and the
// fit both add blocks that have none, so this fills the gaps rather than
// flattening what is there.
func renumber(
	spec plan.Plan,
) plan.Plan {
	var (
		taken = map[int]map[int]bool{}
		kept  = make([]bool, len(spec.Blocks))
	)

	// The keys a lift gave, first, so a block the document placed keeps its own.
	// Where two ask for one key the first keeps it and the other moves, because two
	// blocks cannot be the same entry.
	for i, b := range spec.Blocks {
		if b.Pos < 0 || claim(taken, b.DSP, b.Pos) {
			continue
		}

		kept[i] = true
	}

	// Then everything else, into the lowest key nothing has claimed: a block the
	// corpus filled in, one the fit moved, and every block of a chain built from
	// gear rather than lifted.
	for i := range spec.Blocks {
		if kept[i] {
			continue
		}

		dsp := spec.Blocks[i].DSP

		at := 0
		for claim(taken, dsp, at) {
			at++
		}

		spec.Blocks[i].Pos = at
	}

	return spec
}

// claim takes a key on a processor, and says whether somebody already had it.
func claim(
	taken map[int]map[int]bool,
	dsp int,
	at int,
) bool {
	if taken[dsp] == nil {
		taken[dsp] = map[int]bool{}
	}

	if taken[dsp][at] {
		return true
	}

	taken[dsp][at] = true

	return false
}

// wanted is what one chain entry asked for, kept beside the block it resolved to.
//
// One slice rather than two, because `fill` and `demand` insert blocks into the
// middle of a chain and keep this aligned as they go. A second parallel slice
// did not go through them: every control landed on the block next door the first
// time a chain changed shape, and the amplifier's Drive was checked against a
// compressor.
type wanted struct {
	// words is the seven-term vocabulary, which the compiler turns into values.
	words *rig.Settings
	// stated is the controls the rig gives outright, applied after the words so
	// a value beats a word.
	stated *map[string]catalog.Setting
	// entry is what the document said about the block, which is where its
	// position, its parallel path and the rest of the device's own attributes
	// come from. A block the corpus added has none and takes the defaults.
	entry *rig.ChainEntry
}

// statedControls puts the values a rig states outright onto their blocks.
//
// After the words, never before. A word is a request and a value is an answer,
// and `setKnobs` writes over whatever is already there, so the order is what
// decides which survives. Running this first let `drive: 0.5` overwrite a
// `Drive` somebody had set by ear, which is the one thing this section exists to
// prevent.
//
// Checked against the catalog by name and by range. A control the model does not
// have is refused and named, with the ones it does take listed, because the
// alternative is a document that reads correctly and builds something else.
func statedControls(
	built []plan.Block,
	blocks []catalog.Block,
	asked []*wanted,
) error {
	out := []error(nil)

	for i, entry := range asked {
		if entry == nil || entry.stated == nil || i >= len(built) {
			continue
		}

		for name, v := range *entry.stated {
			spec, ok := blocks[i].Params[name]
			if !ok {
				out = append(out, &NoSuchValueError{
					Field: fmt.Sprintf("chain[%d].controls.%s", i, name),
					Value: name,
					Near:  takes(blocks[i]),
					Whole: true,
				})

				continue
			}

			if err := within(spec, v.ParamValue, i, name); err != nil {
				out = append(out, err)

				continue
			}

			built[i].Params[name] = v.ParamValue
		}
	}

	return errors.Join(out...)
}

// within refuses a value the control cannot take.
//
// The catalog carries a range for all 5,602 controls, so this is checkable
// rather than hopeful. A device handed a value past the end of a control refuses
// the whole preset, and finding that out from the pedal is worse than finding it
// out from a message naming the control.
func within(
	spec catalog.Param,
	v catalog.ParamValue,
	at int,
	name string,
) error {
	got, ok := v.Float()
	if !ok {
		return nil
	}

	// Two ways to pass, in one condition because the first is a guard rather
	// than a case worth its own branch: a control the catalog gives no range for
	// has nothing to be outside of, and comparing against 0..0 would refuse
	// every value it could hold.
	//
	// A millionth of the control's own travel either side of the bound, because a
	// preset's own file does not spell a value to the precision the catalog states
	// a bound in.
	//
	// A parameter is a float32 on the wire, so a reading off hardware is a widened
	// float32 and lands a hair outside: a cabinet's low cut came back
	// 19.899999618530273 against a minimum of 19.9. Worse, 188 values in the preset
	// corpus are spelled `0.00999999` against a minimum of `0.01`, which is six
	// significant figures and ten float32 steps under it. Comparing at float32
	// covers the first and not the second.
	//
	// The number is not a judgement anybody had to make finely. The two populations
	// are seven orders of magnitude apart: the rounding sits at 1e-6 of the travel
	// and the 64 values that are genuinely out of range miss by 3.5 to 99 times it,
	// a `Drive` of 8.0 against 0..1 and a `Predelay` of 20.0 against 0..0.2. Any
	// threshold between those separates them, and a millionth of the travel is far
	// below what a float32 can even represent across most ranges.
	slack := (spec.Max - spec.Min) * rounding

	if spec.Min == spec.Max ||
		(got >= spec.Min-slack && got <= spec.Max+slack) {
		return nil
	}

	return &catalog.BadParamError{
		Model:  name,
		Key:    fmt.Sprintf("chain[%d].controls.%s", at, name),
		Reason: fmt.Sprintf("%v is outside %v..%v", got, spec.Min, spec.Max),
	}
}

// rounding is how much of a control's travel a preset's own file may round a value
// by, as a fraction.
const rounding = 1e-6

// saidKnobs puts each entry's settings onto the block it resolved to.
//
// Blocks the rig did not ask for are at the end of the chain and have no
// settings beside them, so the lists run out together.
func saidKnobs(
	built []plan.Block,
	blocks []catalog.Block,
	asked []*wanted,
) error {
	out := []error(nil)

	for i, held := range asked {
		if held == nil || held.words == nil || i >= len(built) {
			continue
		}

		out = append(out, setKnobs(
			built[i].Params, blocks[i], held.words,
			fmt.Sprintf("chain[%d].settings", i)))
	}

	return errors.Join(out...)
}
