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
package translate

import (
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// This file fills the switches a rig's own records cannot ask for.
//
// [Stompable] answers from measurement, and measurement runs out. A word is
// earned on one of four axes — mid, centroid, harmonics, decay — which answer
// with an equaliser, a drive and a reverb and nothing else, so three switches
// is the ceiling and only a target earning all three words reaches it. Of the
// 43 rigs here, 21 earn nothing at all: fourteen are distinctive only on a
// shorter decay, which nothing on a floor answers, and seven earn no word.
//
// So two of seven presets loaded onto a pedal on 2026-10-23 arrived with
// nothing on any switch, and five genres shared one drive because the ranking
// had nothing to tell them apart with.
//
// **A switch is not a claim.** Everything here arrives bypassed, so nothing it
// chooses changes how the preset sounds until a foot says so. That is what
// makes a weaker source honest for this and dishonest for an amplifier
// control: a block switched off is an offer, and the figure behind it is
// printed either way.

// carriedButOff ranks a kind of block by how often players put one in a chain
// and leave it switched off.
//
// The corpus's own words for what this measures, from `CategoryStats.Engaged`:
// "5.7% hold a wah and not one does, which is a footswitch effect sitting
// there waiting to be stepped on rather than a block anybody wanted in the
// signal."
//
// Across the 172 bass chains measured, carried and left off runs:
//
//	wah     9 of 9     100.0%
//	pitch   34 of 46    73.9%
//	mod     39 of 54    72.2%
//	filter  6 of 11     54.5%
//	drive   55 of 107   51.4%
//	delay   12 of 28    42.9%
//	reverb  15 of 49    30.6%
//	eq      13 of 78    16.7%
//	comp    19 of 146   13.0%
//
// The order is the point. A compressor is in 85% of chains and running in
// nearly all of them, so it is the sound rather than an effect; a wah is in
// 5% and running in none, so it is only ever a switch. Ranking on presence
// alone would put the compressor first and the wah last, which is backwards
// for this question and right for the one `fill` asks.
//
// Zero where nothing holds one, which drops the kind rather than ranking it
// last: a figure over no chains is not a small number, it is no number.
func carriedButOff(
	stats corpus.CategoryStats,
) float64 {
	if stats.Chains == 0 {
		return 0
	}

	return float64(stats.Chains-stats.Engaged) / float64(stats.Chains)
}

// usuallyOn is how often a kind is running among the chains that hold one,
// above which it is the sound rather than something to stomp.
//
// The same figure `compile.fill` decides with, repeated rather than shared
// because the two packages do not import each other, and named the same so a
// reader finds both.
const usuallyOn = 0.75

// offer is one kind of block worth a switch, and how strongly.
type offer struct {
	kind catalog.Category
	// share is how often a chain holding one leaves it off, and held how many
	// chains hold one at all. Both travel because the sentence says both: a
	// share over nine chains and a share over a hundred are different claims.
	share float64
	held  int
}

// Pizzazz is the effects a corpus says are worth a switch, filling whatever
// [Stompable] left.
//
// Taken after the measured ones rather than instead of them, because a block
// a target's own records ask for is a better offer than one strangers' presets
// suggest. This only ever fills what measurement could not.
//
// **Rotated by the subject's own name**, so two rigs that measure alike are
// still offered different things to try. Every candidate is one the corpus
// supports, and which of them a given rig gets is arbitrary among those: that
// is honest for an offer and would not be for a setting. Deterministic, so a
// rebuild is byte-identical and two people building the same rig get the same
// pedal.
//
// Nothing that changes the notes, which is [catalog.Block.ChangesTheNotes] and
// the whole of task 328. Pitch ranks second here, 34 of 46 bass chains
// carrying one switched off, and it is still refused: an octave is a different
// note and a build may not choose one on somebody's behalf, bypassed or not.
// A rig naming one by hand is a decision somebody made and never comes here.
func Pizzazz(
	id string,
	spec rig.Spec,
	deps Deps,
	notes *Notes,
) []Stomped {
	if deps.Catalog == nil || deps.Corpus == nil || id == "" {
		return nil
	}

	// Nothing is offered to a rig that names its own song sections, and this
	// is not a performance concern.
	//
	// A section says which blocks play in that part of the song, by role, and
	// a block nobody put in the chain is a mistake the build refuses: a
	// document saying `play: [drive]` over a chain of amplifier and cabinet is
	// an author who meant something they did not write. An offered drive makes
	// that document resolve, which silently turns their error into a preset
	// where the section plays a pedal they never chose.
	//
	// So a sectioned document gets exactly what it says and nothing more. The
	// refusal is worth more than the offer.
	if spec.Sections != nil && len(*spec.Sections) > 0 {
		return nil
	}

	grammar, ok := deps.Corpus.Grammar[string(spec.Instrument)]
	if !ok || grammar.Chains == 0 {
		return nil
	}

	held := map[catalog.Category]bool{}

	// Room is counted against what will actually take a switch.
	//
	// A kind the rig names holds one, because somebody chose it and a block
	// chosen by hand outranks a block the corpus guessed at. A kind the
	// corpus runs when it carries one does not: it is the sound rather than
	// something to stomp, `compile.Assign` puts it behind every offer, and
	// counting it here held a switch open for something that would not use
	// it.
	//
	// In practice that kind is the compressor. Every artist rig names one and
	// nearly all of them run it, so a pedal somebody is documented with lost
	// its place to a compressor nobody steps on: Les Claypool came back with
	// a Mu-Tron, a reverb and a Deluxe Comp, which is one of his three
	// switches spent on the corpus and one on a compressor.
	//
	// It still gets a switch where nothing else wants one, which is what
	// keeps a chain of amplifier, cabinet and compressor from reaching the
	// pedal with an empty board.
	room := plan.LimitsFor(deps.Catalog.Device).Switches

	for _, entry := range spec.Chain {
		kind := catalog.Category(entry.Role)
		held[kind] = true

		if !catalog.Stompable(kind) {
			continue
		}

		if stats, known := grammar.Categories[kind]; known &&
			stats.Running() >= usuallyOn {
			continue
		}

		room--
	}

	if room <= 0 {
		return nil
	}

	want := offers(grammar, held, id)

	// What is left of the chip, because a block nobody can fit is not an
	// offer. Matt Freeman's chain reached 99.03% against a 95% ceiling and the
	// build refused outright, which turned three extra switches into no
	// preset at all: an offer that breaks the thing it was added to is worse
	// than no offer.
	spare := spareOn(spec, deps)

	out := make([]Stomped, 0, min(room, len(want)))

	for _, at := range want {
		if len(out) >= room {
			break
		}

		entry, gear, cost, ok := popular(grammar, deps.Catalog, at.kind)
		if !ok {
			continue
		}

		if cost > spare {
			*notes = append(*notes, Note{
				About: string(at.kind),
				Said: fmt.Sprintf(
					"%s would answer for a %s and costs %.3g%% of the chip "+
						"where %.3g%% is left, so the switch stays empty",
					gear, at.kind, cost, spare),
			})

			continue
		}

		spare -= cost

		why := fmt.Sprintf(
			"%d of the %d %s chains the corpus measured hold a %s and leave "+
				"it switched off, which is a pedal rather than a sound, and "+
				"%s is the one most of them carry. It arrives switched off",
			int(at.share*float64(at.held)+0.5), at.held,
			spec.Instrument, at.kind, gear)

		said(&entry, why+".")

		*notes = append(*notes, Note{
			About: string(at.kind), Said: why, Honoured: true,
		})

		out = append(out, Stomped{Entry: entry, Said: why})
	}

	return out
}

// offers is every kind worth a switch, best first, rotated for this subject.
func offers(
	grammar corpus.Grammar,
	held map[catalog.Category]bool,
	id string,
) []offer {
	all := make([]offer, 0, len(grammar.Categories))

	for kind, stats := range grammar.Categories {
		if held[kind] || !catalog.Stompable(kind) || stats.Chains == 0 {
			continue
		}

		share := carriedButOff(stats)
		if share <= 0 {
			continue
		}

		// **A kind the corpus runs when it carries one is the sound, not an
		// offer.** This file has argued it from the first line — "a
		// compressor is in 85% of chains and running in nearly all of them,
		// so it is the sound rather than an effect" — and then offered one
		// anyway whenever a switch was free.
		//
		// What that did: `fill` adds a compressor enabled because nearly
		// every chain runs one, and this reached the category first and put
		// a bypassed one on a switch instead, so `fill` found one present
		// and left it alone. Four genres built on 2026-10-25 came out two
		// with the compressor in the signal and two with it switched off,
		// decided by nothing more than whether the rig named a drive and
		// left a switch spare.
		//
		// The same threshold `fill` uses, so the two cannot disagree about
		// which side of the line a kind falls.
		if stats.Running() >= usuallyOn {
			continue
		}

		all = append(all, offer{kind: kind, share: share, held: stats.Chains})
	}

	// By share, then by name, so two kinds players leave off equally often
	// come back in the same order every run.
	sort.Slice(all, func(i, j int) bool {
		if all[i].share != all[j].share {
			return all[i].share > all[j].share
		}

		return all[i].kind < all[j].kind
	})

	if len(all) == 0 {
		return nil
	}

	// Rotated rather than shuffled, so the ranking still shows through: a rig
	// starting at the third candidate gets the third, fourth and fifth rather
	// than three drawn from anywhere.
	//
	// Nothing to rotate by means no rotation, which leaves the ranking itself.
	// [Pizzazz] refuses a rig with no identifier, so this is the order a test
	// reads rather than one a build reaches.
	if id == "" {
		return all
	}

	sum := fnv.New32a()
	_, _ = sum.Write([]byte(id))

	at := int(sum.Sum32()) % len(all)

	return append(all[at:], all[:at]...)
}

// popular is the model of one kind that the most chains for this instrument
// hold, and what a person calls it.
//
// The corpus rather than the measured ranking, which is the point of this
// file: measurement is what put five genres on one drive, and what most bass
// players reach for is a different question with a different answer.
//
// Nothing that changes the notes, and nothing the catalog does not carry.
func popular(
	grammar corpus.Grammar,
	cat *catalog.Catalog,
	kind catalog.Category,
) (rig.ChainEntry, string, float64, bool) {
	best, chains := catalog.ModelID(""), 0

	for model, held := range grammar.Models {
		one, known := cat.Block(model)
		if !known || one.Category != kind || one.ChangesTheNotes() {
			continue
		}

		// Ties by identifier, so a corpus that counts two models alike
		// resolves the same way every run.
		if held > chains || (held == chains && model < best) {
			best, chains = model, held
		}
	}

	if chains == 0 {
		return rig.ChainEntry{}, "", 0, false
	}

	one, _ := cat.Block(best)
	off := false

	return rig.ChainEntry{
		Role:    rig.Role(kind),
		Gear:    one.Gear(),
		Enabled: &off,
	}, one.Gear(), one.DSP.Mono, true
}

// spareOn is how much of the chip the chain has not spent, in the percent
// Line 6 state a block's cost in.
//
// Mono, because a chain built here is one mono path: `measure controls` says
// the same and for the same reason, that a device loads one unless a preset
// asks otherwise.
//
// A gear name the catalog cannot place costs nothing here rather than
// guessing, which can only overstate what is left. The build still refuses a
// chain that does not fit, so the floor under this is the budget check rather
// than this arithmetic.
func spareOn(
	spec rig.Spec,
	deps Deps,
) float64 {
	spent := 0.0

	for _, entry := range spec.Chain {
		kind := catalog.Category(entry.Role)

		// The dearest model of that name rather than the first the map
		// hands over, so this can only overstate what has been spent. A
		// gear name resolves to more than one model — 661 models share 468
		// names — and understating the bill is what lets an offer break the
		// build it was added to.
		dearest := 0.0

		for id := range modelsFor(deps.Catalog, entry.Gear, kind) {
			one, known := deps.Catalog.Block(catalog.ModelID(id))
			if !known {
				continue
			}

			dearest = max(dearest, one.DSP.Mono)
		}

		spent += dearest
	}

	return plan.LimitsFor(deps.Catalog.Device).ChipCeiling - spent
}
