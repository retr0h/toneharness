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
	"math"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// This file decides what a rig gets on its footswitches.
//
// Every chain the corpus builds held the same two: a drive and an equaliser,
// on all eleven genres, because a list of kinds was written down and that was
// the list. No artist got any, because the list was reached from a genre and
// an artist rig names its own genres, "funk rock" and "jazz fusion", none of
// which the bass corpus can aim at.
//
// So the kinds are read off the records instead. A target earns a word where
// one of its figures sits outside what everybody else measures, and that word
// is the only thing here that says a block of some kind is wanted rather than
// merely available. Nothing else in four figures does: task 267 records what
// choosing a kind without one produced, which was a BOSS Metal Zone on rock
// bass and a clean boost on metal.

// answers is the kind of block that answers one figure a target's records are
// distinctive on, and whether anything on a pedalboard answers it at all.
//
// Four figures ever earn a word across the 43 rigs here: decay 25 times, the
// mid band 20, the centroid 17 and harmonics 4. The other six earn none, so a
// table covering them would be a table nothing has ever read.
//
// `more` is which side of everybody else the records sit, because half of
// these only have an answer one way round.
//
// **The centroid and the mid band are both an equaliser**, and a target
// earning both gets one rather than two: they are the same control of the
// same thing, and the second would land on a switch beside the first with
// nothing different to do.
//
// **More harmonics is a drive.** Energy above the fundamental is what
// distortion makes, and `harmonics` is measured as exactly that share. Fewer
// is nobody's pedal: nothing on a floor takes harmonics out of a signal.
//
// **A longer decay is a reverb.** What the figure measures is a tail, how long
// a note takes to fall to a quarter, and a tail is what a reverb adds. A delay
// would be the other guess and it is the wrong one: a delay makes discrete
// repeats, which is a different musical statement and not what a long decay
// reads as. The measured decay of the blocks cannot arbitrate between the two
// either, because the measuring loop pads every recording with a second of
// silence to catch a reverb's tail and that moves the figure the same way for
// every block, which is why [measured.Spectral] weighs it at zero.
//
// **A shorter decay has no answer**, and 14 of the 43 earn nothing else. A
// gate would shorten a note by cutting it off and a compressor by holding the
// front, and neither is a thing to put under a foot on a bass: the gate chops
// notes and the compressor is set once and left, which is why
// `compile.stomped` excludes it. So those rigs get nothing, and the build says
// so rather than reaching for a block to fill a switch.
func answers(
	key audio.Figure,
	more bool,
) (catalog.Category, bool) {
	switch {
	case key == audio.KeyCentroid,
		key == audio.KeyLow,
		key == audio.KeyMid,
		key == audio.KeyHigh:
		return catalog.CategoryEQ, true
	case key == audio.KeyHarmonics && more:
		return catalog.CategoryDrive, true
	case key == audio.KeyDecay && more:
		return catalog.CategoryReverb, true
	}

	return "", false
}

// asked is one kind of block a target's records ask for, and the word asking.
type asked struct {
	kind catalog.Category
	// term is the word, and margin how far past the line the records sit in
	// that figure's own units. A word is only as good as its margin: 39% of
	// the energy in the mid band against 4% is a word nothing will overturn,
	// and 8% against 4% flipped the first time somebody else was measured.
	term   string
	key    audio.Figure
	margin float64
	// past is the margin as a share of what everybody else measures, which is
	// the only form of it two figures can be compared in.
	//
	// A margin is in its figure's own units, so the raw numbers rank by unit
	// rather than by distinctiveness: across the 43 rigs here the centroid
	// margins run from 10 to 78 hertz and the harmonics margins from 0.03 to
	// 0.08 of a share, so sorting on them put the centroid first every time
	// a target earned a word on it. As a share of the others' median, Jaco
	// Pastorius's 78Hz of centroid is 0.6 and Matt Freeman's 0.08 of
	// harmonics is 0.27, which are two numbers about the same thing.
	past float64
}

// askedFor is the kinds of block a target's records ask for, in the order they
// are worth a switch.
//
// Ordered by margin, which is where distinctiveness lives: it is how far the
// other players' median would have to move to take the word away, in the
// figure's own units. A rig with more words than switches keeps the ones its
// records are furthest out on.
//
// A kind already in the chain is dropped rather than doubled. Two blocks of
// one kind would also light the same, because `compile.assign`'s colour table
// is keyed by category.
//
// Capped by the switches the device has, since a block past the last switch is
// one no foot can reach and the foot is the whole reason for it.
func askedFor(
	terms []audio.Derived,
	held map[catalog.Category]bool,
	switches int,
) []asked {
	all := make([]asked, 0, len(terms))

	for _, term := range terms {
		kind, ok := answers(term.Key, term.Mine > term.Others)
		if !ok || held[kind] {
			continue
		}

		// Against what everybody else measures, which is what makes two
		// figures comparable. Zero where the others measure nothing, which
		// no figure a word is earned on does: a share of zero earns no word
		// because there is nothing to sit outside of.
		past := 0.0
		if term.Others != 0 {
			past = term.Margin / math.Abs(term.Others)
		}

		all = append(all, asked{
			kind:   kind,
			term:   term.Term,
			key:    term.Key,
			margin: term.Margin,
			past:   past,
		})
	}

	// Sorted before the kinds are deduplicated rather than after, so the word
	// a block is attributed to is the one its records are furthest out on. A
	// target earning both `bright` and `mid-forward` gets one equaliser, and
	// which of the two put it there is the interesting half of the sentence.
	//
	// By how far past everybody else the records sit, then by the kind's own
	// name and then the word's, so two sitting equally far out come back in
	// the same order every run rather than however the slice arrived.
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].past != all[j].past {
			return all[i].past > all[j].past
		}

		if all[i].kind != all[j].kind {
			return all[i].kind < all[j].kind
		}

		return all[i].term < all[j].term
	})

	out := make([]asked, 0, len(all))
	seen := map[catalog.Category]bool{}

	for _, at := range all {
		if seen[at.kind] {
			continue
		}

		seen[at.kind] = true

		out = append(out, at)
	}

	return out[:min(len(out), max(switches, 0))]
}

// Stompable is the effects a rig's own records ask for, each switched off.
//
// Keyed by the document's own identifier rather than by anything in the ask,
// which is the route task 268 established: a rig for somebody aiming at their
// own records is not circular, and reading `ask.subject` as a target inside
// [playerTarget] turned a deliberate refusal into a build. The caller knows
// the identifier, and `presets.Make` already resolves this rig's aim from it.
//
// A player first, then a genre, the same order the aim is resolved in. The two
// kinds of rig here are a person and a sound, 32 artists and 11 genres, and
// nothing is both.
//
// Switched off, every one of them. A block chosen because a figure sits
// outside everybody else's is a block for reaching the far end of what those
// records cover, not for sitting in the signal path: the base sound is the
// amplifier until somebody steps on it.
//
// Nothing where the records ask for nothing, which is 21 of the 43 and not a
// failure. Fourteen of them are distinctive only on a shorter decay, which
// nothing on a floor answers, and seven earn no word at all.
func Stompable(
	id string,
	spec rig.Spec,
	setup tone.Setup,
	deps Deps,
	notes *Notes,
) []Stomped {
	if deps.Catalog == nil || id == "" {
		return nil
	}

	at, terms, ok := aimedAt(id, deps)
	if !ok {
		return nil
	}

	held := map[catalog.Category]bool{}
	for _, entry := range spec.Chain {
		held[catalog.Category(entry.Role)] = true
	}

	// What is left after the chain's own blocks, since a researched rig may
	// already name the pedal its player is documented with: Flea's Maestro
	// Bass Brassmaster is a drive, so nothing adds him another.
	switches := plan.LimitsFor(deps.Catalog.Device).Switches
	for kind := range held {
		if catalog.Stompable(kind) {
			switches--
		}
	}

	want := askedFor(terms, held, switches)
	if len(want) == 0 {
		*notes = append(*notes, Note{
			About: id,
			Said: "its records ask for no effect this device has a pedal for, " +
				"so nothing was put on a switch",
		})

		return nil
	}

	// The instrument the chain is for, which decides which half of the
	// catalog the ranking draws from.
	wants := string(spec.Instrument)

	out := make([]Stomped, 0, len(want))

	for _, asked := range want {
		ask := tone.Ask{}
		if wants != "" {
			was := tone.AskInstrument(wants)
			ask.Instrument = &was
		}

		entry, _, chose := nearestOf(at, ask, setup, deps, asked.kind, notes)
		if !chose {
			continue
		}

		off := false
		entry.Enabled = &off

		// Why this kind, written onto the block beside why this model. The
		// model's evidence says what it measured nearest; this says what asked
		// for a block of the kind at all, which is the decision somebody is
		// most likely to disagree with.
		//
		// Onto the measured evidence rather than beside it as a second entry,
		// because it qualifies that reasoning rather than standing apart from
		// it: the model was ranked against these records and the kind was
		// chosen from the same ones.
		said(&entry, fmt.Sprintf(
			"It is here because %s earns %q against the players who measure "+
				"otherwise, by %.3g of %s, and that is the figure a "+
				"pedalboard answers with a %s. It arrives switched off.",
			at.named, asked.term, asked.margin, asked.key, asked.kind))

		why := fmt.Sprintf(
			"%s earns %q by %.3g of %s, which is what put a %s on a switch",
			at.named, asked.term, asked.margin, asked.key, asked.kind)

		*notes = append(*notes, Note{
			About: string(asked.kind), Said: why, Honoured: true,
		})

		out = append(out, Stomped{Entry: entry, Said: why})
	}

	return out
}

// Stomped is one block a rig's own records asked for, and why.
//
// The sentence comes back beside the block rather than being looked up again
// from the notes: a caller matching a note to a block by its role is a caller
// that gets it wrong the day two blocks of one role are chosen, and it would
// need a fallback sentence for a case that cannot happen.
type Stomped struct {
	// Entry is the block, switched off and carrying its evidence.
	Entry rig.ChainEntry
	// Said is what asked for a block of this kind, in a sentence.
	Said string
}

// aimedAt is what a rig's own records measure as, and the words they earn.
//
// A player first and then a genre, for the reason `presets.aimOf` gives: a
// rig's identifier is who it is for, and the two kinds here are a person and a
// sound with nothing being both.
func aimedAt(
	id string,
	deps Deps,
) (aim, []audio.Derived, bool) {
	if got, ok := audio.ShippedPlayer(id); ok {
		// Through [displacedTo] rather than [AimAtPlayer], which is the same
		// arithmetic behind a second lookup of the records already in hand.
		// Its answer of false means only that the lookup failed, which it
		// cannot have here, and a branch nothing can reach is a branch nobody
		// has checked.
		figures := displacedTo(audio.Genre{
			Across: got.Across, Elsewhere: got.Elsewhere,
		}, deps.Measured.Baseline)

		earned := make([]string, 0, len(got.Terms))
		for _, term := range got.Terms {
			earned = append(earned, term.Term)
		}

		return aim{
			figures: figures,
			named:   id,
			read:    got.Across.Measured(),
			records: got.Records,
			players: 1,
			earned:  earned,
			caveat: "A player measured off their own finished records, so the " +
				"figures carry the amplifier, the microphone, the desk and the " +
				"master of every one of them. This block is what sits nearest " +
				"those figures, not one anybody is known to have used.",
		}, got.Terms, true
	}

	got, ok := audio.ShippedGenre(id)
	if !ok || !got.Usable {
		return aim{}, nil, false
	}

	figures := displacedTo(got, deps.Measured.Baseline)

	earned := make([]string, 0, len(got.Terms))
	for _, term := range got.Terms {
		earned = append(earned, term.Term)
	}

	return aim{
		figures: figures,
		named:   id,
		read:    got.Across.Measured(),
		records: got.Records,
		players: got.Players,
		earned:  earned,
		caveat: "A genre measured off finished records, so the figures carry " +
			"the amplifier, the microphone, the desk and the master of every " +
			"one of them. This block is what sits nearest those figures, not " +
			"one anybody is known to have used. It also moves: a genre is " +
			"measured against the players who play none of it, so this answer " +
			"changes when the corpus gains players.",
	}, got.Terms, true
}

// said adds a sentence to the measured evidence a block already carries.
//
// Onto the audio evidence and nowhere else, because that is the one this
// qualifies. Nothing is added where there is nothing to qualify.
func said(
	entry *rig.ChainEntry,
	sentence string,
) {
	if entry.Evidence == nil {
		return
	}

	held := *entry.Evidence

	for i := range held {
		if held[i].Kind != tone.EvidenceAudio || held[i].Note == nil {
			continue
		}

		was := *held[i].Note + " " + sentence
		held[i].Note = &was

		return
	}
}

// Spliced puts one entry into a chain at the position its role implies.
//
// Before the first entry whose role sits later in a signal path, which lands a
// drive ahead of the amplifier and a reverb behind the cabinet without needing
// to know what else the chain holds. At the end where nothing sits later,
// which is what a reverb added to an amplifier and a cabinet gets.
//
// The chain's own order is otherwise untouched. Whoever wrote it made that
// decision and a sort here would silently move somebody's pedals, which is
// why this answers where one more block goes rather than how a chain should
// be arranged.
//
// Exported because two callers splice the same block for different reasons:
// the build puts it in the chain it compiles, and the resolve puts it in the
// chain it reads evidence off, and a second table of signal order to keep in
// step would be a second table to get wrong.
func Spliced(
	chain []rig.ChainEntry,
	entry rig.ChainEntry,
) []rig.ChainEntry {
	at := len(chain)

	for i, held := range chain {
		if order(held.Role) > order(entry.Role) {
			at = i

			break
		}
	}

	out := make([]rig.ChainEntry, 0, len(chain)+1)
	out = append(out, chain[:at]...)
	out = append(out, entry)

	return append(out, chain[at:]...)
}
