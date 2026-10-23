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

// Package translate turns what somebody asked for into what a device can be
// told.
//
// The layer between what a person writes and what the compiler reads. An ask
// says what they want and a Setup says what they have, both in words anybody
// would use; the rig says exactly which models go where, and is meant to be read
// by the compiler rather than by a person.
//
//	ask + Setup        what we mean, and what we have
//	      ↓            this package
//	rig                exact, resolved, deterministic, shareable
//	      ↓            pkg/sdk/internal/compile
//	.hlx               what the pedal eats
//
// The split earns itself at the second step. The rig is the half worth sharing,
// because two people compiling the same one get the same preset, and the ask is
// not: "punk, a bit darker" resolves against a corpus and a library of
// measurements that both move. Both are written down, in one document.
//
// Nothing here guesses quietly. Every choice this makes, and every part of
// the ask it could not honour, comes back in the notes.
package translate

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// Deps is what translating needs to know about the world.
type Deps struct {
	// Catalog is what the device can do.
	Catalog *catalog.Catalog
	// Measured is every block on that device, measured alone. It is what
	// answers "which of these sounds most like the target", which no amount
	// of reasoning about names can.
	Measured measured.Library
	// UnknownWords reports the words an ask uses that the vocabulary does not
	// carry, with the nearest it does.
	//
	// Handed in rather than called, because the vocabulary belongs to the
	// compiler and the compiler runs after this: the pipeline at the top of
	// this file is ask to translate to rig to compile, and importing
	// downhill would invert it. Nil means nothing checks, which is what a
	// caller that only wants a chain resolved gets.
	UnknownWords func(words []string) []UnknownWord
	// RigNamed is the curated rig for a player, band or sound somebody named,
	// and whether there is one.
	//
	// Handed in for the reason UnknownWords is: a rig store reads a directory
	// and the shipped knowledge, and translate sits upstream of both. Nil means
	// nothing is looked up, so an ask naming a player resolves nothing, which is
	// what a caller wanting only its own gear resolved gets.
	RigNamed func(name string) (rig.Spec, bool)
	// CurvesFor is what every control of one block does, measured, and
	// whether that block has been swept.
	//
	// Handed in for the reason RigNamed is: the sweeps are a directory of
	// readings and translate sits upstream of whatever reads it. Nil means
	// nothing is looked up, and a block added to a chain then arrives at the
	// catalog's defaults, which is what every added block used to do.
	CurvesFor func(model catalog.ModelID) (measured.Curves, bool)
}

// UnknownWord is a word somebody used that nothing defines, and the nearest
// one that is defined.
type UnknownWord struct {
	// Term is what they wrote.
	Term string
	// Near are the words the vocabulary does carry that look close, which may
	// be none.
	Near []string
}

// Note is one thing the translation did, or could not do.
//
// Kept and reported rather than folded into the answer, because a rig that
// silently substituted an amplifier reads exactly like one that was asked
// for it.
type Note struct {
	// About is the part of the ask this concerns.
	About string
	// Said is what happened, in a sentence.
	Said string
	// Honoured is false when the ask could not be met.
	Honoured bool
}

// Notes is everything a translation has to say for itself.
type Notes []Note

// Unmet is the notes describing what could not be done.
func (n Notes) Unmet() Notes {
	out := make(Notes, 0, len(n))

	for _, note := range n {
		if !note.Honoured {
			out = append(out, note)
		}
	}

	return out
}

// Translate turns a request and a setup into a rig.
//
// Deterministic given the same catalog, the same measurements, the same ask and
// the same Setup. That is what makes the rig worth sharing where the ask is not:
// the ask resolves against things that move, and the answer does not.
func Translate(
	spec tone.Ask,
	setup tone.Setup,
	deps Deps,
) (rig.Spec, Notes, error) {
	var notes Notes

	if err := agrees(setup, deps, &notes); err != nil {
		return rig.Spec{}, notes, err
	}

	out := rig.Spec{
		Instrument: instrumentFor(setup, &notes),
	}

	// The words are not copied onto the rig, and that is the point of the
	// split. How it should sound is what somebody asked for, so it stays on
	// the ask; the rig says which gear answered. Nothing is lost by leaving
	// them behind, because whoever wants them is holding the ask that has
	// them: they reach the compiler as a compile.Intent beside the rig, which
	// is the one place a word can be turned into a knob position, because it
	// is the only place the resolved chain exists.
	strung(spec, setup, &notes)

	chain, err := chainFor(spec, setup, deps, &notes)

	// Said rather than silently dropped, and said even when there is no chain.
	// A request carrying a genre or a player this cannot resolve is a request
	// half answered, and the half that was not is the part somebody needs to
	// know about. Returning before this was why a refusal arrived with nothing
	// but the refusal: every note explaining it was written after the early
	// return.
	//
	// Told which names the chain already answered, because a player whose rig
	// was found is resolved, and reporting both would say two contradictory
	// things about one name in one table.
	unresolved(spec, answeredBy(notes), &notes)
	unknownWords(spec, deps, &notes)

	if err != nil {
		return rig.Spec{}, notes, err
	}

	out.Chain = chain

	// After the chain, because it is about what ended up in it.
	speakers(setup, chain, &notes)

	if err := rig.Validate(out); err != nil {
		return rig.Spec{}, notes, err
	}

	return out, notes, nil
}

// speakers counts the speakers in the path when the setup says where the pedal
// goes, and says how many there are.
//
// Honoured, always: nothing is changed and nothing is refused. A cabinet block
// is how a chain is made to sound like a recorded rig, so somebody chasing a
// record through their own amplifier wants both and is right to. Somebody who
// wants their amplifier to be the sound wants one. The difference is taste and
// this is not the place it gets decided.
//
// What is said is the count, which is arithmetic on two things the person
// stated: a cabinet block simulates a speaker, an amplifier has one, and
// `amp-return` reaches that speaker past the amplifier's own preamp while
// `amp-front` does not. No claim is made about what two sounds like, because
// nothing here has measured an amplifier in anybody's room.
func speakers(
	setup tone.Setup,
	chain []rig.ChainEntry,
	notes *Notes,
) {
	if setup.PlaysInto == nil {
		return
	}

	into := *setup.PlaysInto
	if into != tone.AmpFront && into != tone.AmpReturn {
		return
	}

	cabs := 0

	for _, held := range chain {
		if held.Role == rig.RoleCab {
			cabs++
		}
	}

	if cabs == 0 {
		return
	}

	held := "a cabinet block"
	if cabs > 1 {
		held = fmt.Sprintf("%d cabinet blocks", cabs)
	}

	*notes = append(*notes, Note{
		About: "plays_into",
		Said: fmt.Sprintf(
			"the chain holds %s and the setup plays into an amplifier, which "+
				"has a speaker of its own. Nothing was changed: a cabinet "+
				"block is how a chain is made to sound like the record it "+
				"came from, and whether that is wanted through an amplifier "+
				"is taste rather than a rule", held),
		Honoured: true,
	})
}

// unknownWords says which of an ask's words nothing can aim at.
//
// The contract has always promised this — "a word that reaches no control
// cannot be aimed at and saying so is better than accepting it and quietly
// doing nothing" — and nothing did it. compile.CheckWords existed, worked, and
// was reached only by `presets make`, which is a later step: a `tone build`
// took an ask saying `sparkly`, resolved the chain, reported every other thing
// it did, and never mentioned the word at all.
//
// Not honoured, because the word did not reach a control. It is not fatal
// either: the rest of the ask still resolves, and refusing a whole request
// over one adjective would throw away the record and the gear beside it.
//
// The near words are offered rather than chosen between. `sparkly` is probably
// `bright`, and probably is not a rig somebody asked for: a guess that lands
// wrong aims the answer somewhere they cannot see. The gear resolver says
// "that name fits 4 models" for the same reason.
func unknownWords(
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) {
	if deps.UnknownWords == nil || spec.Words == nil {
		return
	}

	said := make([]string, 0, len(*spec.Words))
	for _, word := range *spec.Words {
		said = append(said, word.Term)
	}

	for _, got := range deps.UnknownWords(said) {
		near := ""
		if len(got.Near) > 0 {
			near = ". Did you mean " + strings.Join(got.Near, ", ") + "?"
		}

		*notes = append(*notes, Note{
			About: got.Term,
			Said: fmt.Sprintf(
				"no control is moved by that word, so nothing aimed at it%s",
				near),
		})
	}
}

// agrees holds the measurements to the device somebody says they have.
//
// A setup naming no device is taken at its word rather than refused: somebody
// asking what a record sounds like has not necessarily said what they own,
// and the measurements name the device they came from.
func agrees(
	setup tone.Setup,
	deps Deps,
	notes *Notes,
) error {
	if setup.Device == nil || setup.Device.Model == "" {
		*notes = append(*notes, Note{
			About:    "device",
			Said:     fmt.Sprintf("the setup names none, so this is for %s", deps.Measured.Device),
			Honoured: true,
		})

		return nil
	}

	want := strings.ToLower(setup.Device.Model)
	had := strings.ToLower(deps.Measured.Device)

	if want == had {
		return nil
	}

	*notes = append(*notes, Note{
		About: setup.Device.Model,
		Said: fmt.Sprintf(
			"every block was measured on %s, so nothing here can say which "+
				"of a %s's blocks is closest to anything",
			deps.Measured.Device, setup.Device.Model),
	})

	return &WrongDeviceError{
		Measured: deps.Measured.Device,
		Setup:    setup.Device.Model,
	}
}

// strung reports a request made on one set of strings and held on another.
//
// Reported rather than corrected for, which is #128's answer and the honest
// one. Flatwounds against roundwounds is a larger difference than most pedals
// make, and nothing here has measured what it does to the figures: an amplifier
// chosen by measuring a record played on flats is chosen against a spectrum
// nobody is going to reproduce on rounds. Applying a correction for it now
// would be inventing a number, which is the guessing this project removed.
//
// So the mismatch travels in the notes. Somebody reading that knows why the rig
// may sit wrong, and knows it was noticed rather than missed.
func strung(
	spec tone.Ask,
	setup tone.Setup,
	notes *Notes,
) {
	made := stringsOf(spec)
	held := heldStrings(setup)

	if made == "" || held == "" || made == held || made == "unknown" || held == "unknown" {
		return
	}

	*notes = append(*notes, Note{
		About: "strings",
		Said: fmt.Sprintf(
			"the record was played on %s and the setup holds %s, which is a "+
				"larger difference than most pedals make. Nothing here has "+
				"measured what it does, so no correction was applied",
			made, held),
	})
}

// stringsOf is what the request says the record was played on.
func stringsOf(
	spec tone.Ask,
) string {
	if spec.Played == nil {
		return ""
	}

	for _, played := range *spec.Played {
		if played.Strings != nil && *played.Strings != "" {
			return string(*played.Strings)
		}
	}

	return ""
}

// heldStrings is what the setup says is on the instrument being played.
//
// The default instrument where one is marked, the first otherwise, which is the
// same instrument instrumentFor picks. Two answers about one instrument would be
// worse than none.
func heldStrings(
	setup tone.Setup,
) string {
	if setup.Instruments == nil {
		return ""
	}

	for _, held := range *setup.Instruments {
		if held.Default != nil && *held.Default && held.Strings != nil {
			return string(*held.Strings)
		}
	}

	if len(*setup.Instruments) > 0 {
		if first := (*setup.Instruments)[0]; first.Strings != nil {
			return string(*first.Strings)
		}
	}

	return ""
}

// chainFor is the signal path a request asks for.
func chainFor(
	spec tone.Ask,
	setup tone.Setup,
	deps Deps,
	notes *Notes,
) ([]rig.ChainEntry, error) {
	named, err := namedGear(spec, deps, notes)
	if err != nil {
		return nil, err
	}

	// An amplifier is the one block a chain cannot do without, and there are
	// two ways to arrive at one. A player somebody named has a rig if anybody
	// researched them, and that rig is what they actually played with a source
	// on every piece of it: better evidence than any measurement, so it is
	// tried first. Failing that, and given something to aim at, the
	// measurements pick the closest.
	if !holds(named, rig.RoleAmp) {
		if from, ok := likeTheirRig(spec, deps, notes); ok {
			named = append(from, named...)
		}
	}

	// And a genre is the same evidence one step wider. Its players each have a
	// researched rig, so what the genre used is what most of them used, which
	// is a fact about the records rather than a figure fitted to them.
	//
	// Above measurement for the reason a named player's rig is: 9 of the 11
	// genres this corpus can aim at were given a MESA/Boogie Bass 400+ by
	// ranking blocks against their figures, and four of punk's seven players
	// resolve to an Ampeg SVT into an 8x10. Improving the figures does not fix
	// that. Aiming punk at the edge of its own range rather than its middle
	// moved its amplifier from the Ampeg its players used to a Pearce BC-1
	// preamp, which is a better fit to a number and a worse answer.
	if !holds(named, rig.RoleAmp) {
		if from, ok := likeTheirGenre(spec, deps, notes); ok {
			named = append(from, named...)
		}
	}

	if !holds(named, rig.RoleAmp) {
		if amp, _, ok := nearestTo(spec, setup, deps, "amp", notes); ok {
			named = append([]placed{{entry: amp}}, named...)
		}
	}

	// And what a foot reaches for. A genre's records cover a range no single
	// setting reaches, so the range goes to somebody's feet rather than being
	// averaged into one point: the blocks that carry the chain further into
	// the genre arrive switched off, each on its own switch, and the base
	// sound is the amplifier until one is stepped on.
	named = append(named, stompedFor(spec, setup, deps, notes)...)

	if len(named) == 0 {
		return nil, ErrNothingToBuildFrom
	}

	// In signal order rather than the order they were typed, because listing
	// gear is not stating a signal path: a compressor belongs in front of the
	// amplifier whichever way round somebody wrote them.
	//
	// Except where the request said otherwise. `after: amp` puts a drive
	// behind the amplifier, which is a known way to use one rather than a
	// mistake, and stable sorting keeps two blocks asked behind the same role
	// in the order they were asked for.
	sort.SliceStable(named, func(i, j int) bool {
		return placeOf(named[i]) < placeOf(named[j])
	})

	out := make([]rig.ChainEntry, 0, len(named))
	for _, at := range named {
		out = append(out, at.entry)
	}

	return out, nil
}

// placeOf is where in the signal path one block sits.
//
// A half past the role it was asked to sit behind, which lands it after that
// role and before the next without needing to know what else is in the chain.
// A request cannot count positions: it does not know the compiler will add a
// cabinet, so "after the amp" survives that and "position 4" does not.
func placeOf(
	at placed,
) float64 {
	if at.after != nil {
		return float64(order(*at.after)) + 0.5
	}

	return float64(order(at.entry.Role))
}

// likeTheirRig is the chain a named player already has researched, if anybody
// has.
//
// `like: { artist: Mike Dirnt }` is a request to sound like him, and somebody
// has read a magazine and written down what he played. Using that is not a
// shortcut around measuring: it is the stronger claim, because every piece of it
// carries a source and a measurement carries none.
//
// The name is slugged to find the rig, and handed over raw as well, because
// `rigs.Find` matches an identifier and the aliases beside it: "claypool" and
// "primus" both reach Les Claypool, and no slug of either produces the other.
//
// Only the roles that make a sound. A rig researched for somebody holds their
// amplifier and cabinet, and the request's own gear is layered over this by the
// caller, so what they named wins where the two overlap.
func likeTheirRig(
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) ([]placed, bool) {
	if deps.RigNamed == nil || spec.Like == nil {
		return nil, false
	}

	who, kind := namedSubject(*spec.Like)
	if who == "" {
		return nil, false
	}

	found, ok := deps.RigNamed(who)
	if !ok {
		*notes = append(*notes, Note{
			About: who,
			Said: "no rig has been researched for that " + kind +
				", so nothing was taken from one",
		})

		return nil, false
	}

	out := make([]placed, 0, len(found.Chain))
	for _, entry := range found.Chain {
		out = append(out, placed{entry: entry})
	}

	*notes = append(*notes, Note{
		About:    who,
		Said:     fmt.Sprintf("%d blocks came from the rig researched for them", len(out)),
		Honoured: true,
	})

	return out, len(out) > 0
}

// likeTheirGenre is the gear most of a genre's players used.
//
// A genre names its players and each of them may have a researched rig, so the
// gear a genre used is answerable from documents rather than fitted to a
// figure. Task 244 established the answer is a real mode and not an average:
// three of punk's five resolved players name an Ampeg SVT, and a fourth names
// an Ampeg SVT-II Pro.
//
// Only the amplifier and the cabinet, and deliberately. Those are the two a
// genre's players really did share, and everything else they held is one
// person's pedal rather than the genre's: taking the mode of a drive across
// seven punk bassists would invent a pedalboard nobody owned. What goes in
// front is the character's to decide, which is the rest of the rule in
// build-a-rig's measuring.md.
//
// Counted per role rather than as an amplifier-and-cabinet pair. Pairing reads
// better, because a head and the cabinet behind it are one decision, and it
// throws the answer away: punk's five resolved players name three Ampeg SVTs
// behind two different spellings of an 8x10, so the amplifier has a majority
// of three and no pair has more than two.
//
// A role whose players reach no majority gets nothing from here and falls
// through to the measurements, which is the honest answer rather than a mode of
// one. `RigNamed` reaches the rigs that ship and the ones somebody wrote, so a
// player with no document is not counted, and the note says how many were.
func likeTheirGenre(
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) ([]placed, bool) {
	if deps.RigNamed == nil || len(spec.Genre) == 0 {
		return nil, false
	}

	got, ok := audio.ShippedGenre(string(spec.Genre[0]))
	if !ok || !got.Usable || len(got.Who) == 0 {
		return nil, false
	}

	// Collected per role before anything is counted, because deciding which
	// two names mean one box needs all of them.
	said := map[rig.Role][]rig.ChainEntry{}
	resolved := 0

	for _, who := range got.Who {
		found, has := deps.RigNamed(who)
		if !has {
			continue
		}

		// Counted once per player per role, so a rig naming two amplifiers
		// does not outvote a rig naming one.
		anything := false

		for _, role := range []rig.Role{rig.RoleAmp, rig.RoleCab} {
			for _, entry := range found.Chain {
				if entry.Role != role || entry.Gear == "" {
					continue
				}

				said[role] = append(said[role], entry)
				anything = true

				break
			}
		}

		if anything {
			resolved++
		}
	}

	out := []placed(nil)
	took := map[rig.Role]string{}

	for _, role := range []rig.Role{rig.RoleAmp, rig.RoleCab} {
		entry, count, ok := commonest(deps.Catalog, said[role], catalog.Category(role))

		// A majority rather than a plurality. Two players out of seven
		// agreeing is not what a genre used, and the measurements are a
		// better answer than a mode nobody holds.
		if !ok || count*2 <= resolved {
			continue
		}

		// The cabinet's own controls, set from its sweep.
		//
		// The amplifier's are left to the solve, which already tells the
		// genres apart: across the eleven it answers with ten distinct sets.
		// The cabinet's are not. Eight of the eleven arrived on the same
		// microphone at the same distance, because nothing chose them and the
		// corpus median is one answer for everybody, and that is what left
		// three genres sharing a preset after their amplifier and cabinet were
		// both taken from players who really did share them.
		//
		// The microphone is the biggest lever on this device and the last one
		// nobody pulled: across the twelve on one cabinet it moves the
		// centroid by a median of 32.7Hz where the whole choice of amplifier
		// for punk turned on 14Hz, and the harmonics from 6.6 to 21.4. Task
		// 242 measured it.
		//
		// A list rather than a dial, which is why it belongs here rather than
		// in the solve: `Mic` and `Position` are positions to pick and not
		// quantities to converge on, and task 253 is about the solver being
		// best at the controls that should not reach a target.
		if role == rig.RoleCab {
			chose(&entry, spec, deps, notes)
		}

		out = append(out, placed{entry: entry})
		took[role] = fmt.Sprintf("%s in %d of %d", entry.Gear, count, resolved)
	}

	if len(out) == 0 {
		*notes = append(*notes, Note{
			About: string(spec.Genre[0]),
			Said: fmt.Sprintf(
				"%d of its %d players have a researched rig and no amplifier or "+
					"cabinet is in most of them, so the gear was measured for instead",
				resolved, len(got.Who)),
		})

		return nil, false
	}

	for _, role := range []rig.Role{rig.RoleAmp, rig.RoleCab} {
		said, chose := took[role]
		if !chose {
			continue
		}

		*notes = append(*notes, Note{
			About:    string(spec.Genre[0]),
			Said:     said + " of its players with a researched rig, so it was taken rather than measured for",
			Honoured: true,
		})
	}

	return out, true
}

// commonest is the one piece of gear most of these name, counting two
// spellings of one box once.
//
// Players write gear the way they say it, so one box arrives under several
// names: punk's five researched rigs name "Ampeg 8x10" twice and "8x10 Ampeg
// SVT-810AV" twice, which is four players on one cabinet and counted as two on
// each. Task 244 found the same in the amplifiers, where "Ampeg SVT" and
// "Ampeg SVT-II Pro" are one model.
//
// Grouped by whether two names reach any model in common rather than by
// resolving each to one. Resolving picks a representative and the pick is not
// stable across spellings: "Ampeg 8x10" reaches three cabinets and "8x10 Ampeg
// SVT-810AV" reaches two of those three, so taking the first of each counts
// them apart while the catalog says they overlap.
//
// A name the catalog places nowhere is its own group, which is the honest
// failure: nothing here can tell two unplaceable spellings of one box apart.
func commonest(
	cat *catalog.Catalog,
	said []rig.ChainEntry,
	kind catalog.Category,
) (rig.ChainEntry, int, bool) {
	type group struct {
		models map[string]bool
		entry  rig.ChainEntry
		count  int
	}

	var groups []*group

	for _, entry := range said {
		fits := modelsFor(cat, entry.Gear, kind)

		// The first group sharing a model with this name, and the groups are
		// walked in the order the names arrived so a tie keeps the spelling
		// the earliest player used.
		var found *group

		for _, g := range groups {
			for m := range fits {
				if g.models[m] {
					found = g

					break
				}
			}

			if found != nil {
				break
			}
		}

		if found == nil {
			groups = append(groups, &group{models: fits, entry: entry, count: 1})

			continue
		}

		for m := range fits {
			found.models[m] = true
		}

		found.count++
	}

	best := -1
	for i, g := range groups {
		if best < 0 || g.count > groups[best].count {
			best = i
		}
	}

	if best < 0 {
		return rig.ChainEntry{}, 0, false
	}

	return groups[best].entry, groups[best].count, true
}

// chose picks the cabinet's microphone, and nothing else on it.
//
// The biggest untouched lever on this device, measured in task 242: across the
// twelve microphones on one cabinet the centroid moves by a median of 32.7Hz
// where the whole choice of amplifier for punk turned on 14Hz, and the
// harmonics run from 6.6 to 21.4. Eight of the eleven genres arrived on the
// same microphone at the same distance because nothing chose them and a corpus
// median is one answer for everybody, which is what left three of them sharing
// a preset after their amplifier and cabinet were both taken from players who
// really did share them.
//
// **The microphone and nothing else, and the reason is worth keeping.** Letting
// the same reading set the rest of the cabinet sent Distance to its maximum and
// LowCut to 79.9Hz, which on a bass cuts the fundamentals, and left all of them
// reading alike anyway: worse than the median it replaced and no more distinct.
// That is task 255, a band share inside tolerance being invisible while LowCut
// exploits it. `Mic` does not have that failure, because twelve impulse
// responses have no order to run along and the answer is a choice between
// twelve measured sounds rather than a rail.
//
// By displacement rather than by position, which is [measured.Curves.TowardFrom]
// and the same argument the genre figures already rest on: a microphone 11Hz
// brighter than the others on its cabinet answers a genre 20Hz brighter than
// the other genres. Comparing the two directly instead sent every genre to one
// microphone, because a genre is measured off finished records and a cabinet
// off a dry signal and the nearest of those is always the same end.
func chose(
	entry *rig.ChainEntry,
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) {
	if deps.CurvesFor == nil {
		return
	}

	at, ok := target(spec, deps, notes)
	if !ok {
		return
	}

	model := sweptFor(deps, entry.Gear, catalog.CategoryCab)

	curves, swept := deps.CurvesFor(model)
	if !swept {
		return
	}

	wants := ""
	if spec.Instrument != nil {
		wants = string(*spec.Instrument)
	}

	// The displacement the genre was given, which is what the target carries
	// over the library's own baseline. The absolute figure is not comparable
	// with a microphone's.
	asked := DisplacedFrom(at.figures, deps.Measured.Baseline)

	pick, chosen := curves.TowardFrom(
		micControl, asked, measured.SpectralFor(wants))
	if !chosen {
		return
	}

	held := map[string]catalog.Setting{}
	if entry.Controls != nil {
		held = *entry.Controls
	}

	held[micControl] = catalog.Set(catalog.Int(int64(pick)))
	entry.Controls = &held

	*notes = append(*notes, Note{
		About:    entry.Gear,
		Said:     fmt.Sprintf("microphone %.0f of its twelve, by what it moves", pick),
		Honoured: true,
	})
}

// micControl is what the catalog calls a cabinet's microphone.
const micControl = "Mic"

// DisplacedFrom is how far a target sits from the library's own baseline.
//
// The inverse of what [displacedTo] built, so a chooser that needs the
// displacement rather than the position can have it without the target
// carrying both.
//
// Exported because the compiler needs the same thing and has no baseline to
// subtract: [measured.Curves.Toward] and [measured.Curves.TowardFrom] both
// read `want` as a lean rather than a position, and handing either an absolute
// figure reads as "more of everything", because a centroid in hertz and a band
// share are both positive whatever the target asks for.
func DisplacedFrom(
	at measured.Figures,
	baseline measured.Figures,
) measured.Figures {
	at.Centroid -= baseline.Centroid
	at.Low -= baseline.Low
	at.Mid -= baseline.Mid
	at.High -= baseline.High

	return at
}

// sweptFor is the model to read a sweep from, among those a gear name reaches.
//
// A name reaches several models and only some were swept: "Ampeg 8x10" reaches
// three cabinets, two of them swept, and the third is a pan variant this device
// refuses. So the choice is the swept one, and where more than one was swept
// the one carrying the most controls, which is the microphone'd model rather
// than the plain port of the same box.
//
// That is the model the compiler resolves the same name to, which is what
// makes the values usable: all three cabinets the eleven genres name come back
// with `Mic`, `Distance` and `Position` on them. If it ever picks differently
// the compile says so, naming the control it could not set, rather than
// quietly writing a value nowhere.
//
// Empty where nothing matches or nothing was swept, which [tuned] reads as
// "no sweep for it" and reports.
func sweptFor(
	deps Deps,
	gear string,
	kind catalog.Category,
) catalog.ModelID {
	// No nil guard on CurvesFor: [chose] is the only caller and returns before
	// this where none was offered, so a guard here is a branch nothing can
	// reach and nobody has checked.
	best, most := catalog.ModelID(""), -1

	for id := range modelsFor(deps.Catalog, gear, kind) {
		at := catalog.ModelID(id)

		if _, swept := deps.CurvesFor(at); !swept {
			continue
		}

		held, known := deps.Catalog.Block(at)
		if !known {
			continue
		}

		if n := len(held.Params); n > most {
			best, most = at, n
		}
	}

	return best
}

// modelsFor is every model of one kind a gear name reaches.
//
// The whole set rather than the nearest, because what [commonest] asks is
// whether two names could mean one box and a single answer cannot say that.
// A name the catalog places nowhere answers with itself, so it groups by
// spelling instead of merging with everything.
func modelsFor(
	cat *catalog.Catalog,
	gear string,
	kind catalog.Category,
) map[string]bool {
	out := map[string]bool{}

	if cat == nil {
		return map[string]bool{gear: true}
	}

	// Sorted so a run answers the same way twice. An unsorted walk over the
	// app catalog's duplicate names was a coin flip once already.
	for _, id := range slices.Sorted(maps.Keys(cat.Blocks)) {
		b := cat.Blocks[id]
		if b.Category == kind && b.Matches(gear) {
			out[string(id)] = true
		}
	}

	if len(out) == 0 {
		return map[string]bool{gear: true}
	}

	return out
}

// namedSubject is who a request says to sound like, and what kind of thing they
// are.
//
// One of three, in the order a narrower claim beats a wider one: a song is one
// recording, an artist is a body of work, and a band is several people's. A
// request naming more than one is answered by the narrowest, because that is the
// one it was most specific about.
func namedSubject(
	like tone.Like,
) (string, string) {
	switch {
	case like.Song != nil && *like.Song != "":
		return *like.Song, "song"
	case like.Artist != nil && *like.Artist != "":
		return *like.Artist, "player"
	case like.Band != nil && *like.Band != "":
		return *like.Band, "band"
	}

	return "", ""
}

// placed is one block and where in the chain it goes.
//
// The request's own wish rather than the role's ordinary place, for the entries
// that asked. Kept beside the entry rather than on it, because a rig's chain is
// already in order by the time it is written and carrying the reason would be
// carrying the question into the answer.
type placed struct {
	entry rig.ChainEntry
	// after is the role this was asked to sit behind, where one was named.
	after *rig.Role
}

// stompedFor is the effects a genre gets, switched off, tuned rather than
// defaulted.
//
// A genre rig carried an amplifier, a cabinet and whatever the corpus added on
// a statistic, which for every one of the eleven genres this corpus can aim at
// was a compressor and nothing else. No drive, no equaliser, nothing to step
// on. The records a genre is measured from span a range far wider than one
// setting reaches, so this puts the far end of that range under a switch
// instead of throwing it away: punk's records span the mid band from 0.013 to
// 0.237, and an amplifier set to one of those is not the other.
//
// **Switched off, and that is the point.** The chain somebody hears when they
// load the preset is the amplifier and the cabinet, which is the genre's own
// gear at the genre's own settings. What a switch adds is a further, dirtier
// reading of the same genre, and nobody gets surprised by three drives at once.
//
// **Tuned from the sweeps rather than left at the factory default.** Every
// position of every control was measured, so the value that moves a block
// toward the genre is a reading. A block arriving at its defaults is a block
// nobody set, and the defaults are not the genre's: see [measured.Curves.Toward],
// which also states what it cannot do, and task 254.
//
// One of each kind, because two drives chosen the same way against the same
// target are the same block twice. Capped by what the device has switches for,
// from [plan.LimitsFor], since a block past the last switch is a block nobody
// can reach with a foot and the whole reason for it was the foot.
func stompedFor(
	spec tone.Ask,
	setup tone.Setup,
	deps Deps,
	notes *Notes,
) []placed {
	// A genre the corpus can aim at, and nothing else. Every ask carries a
	// genre field, and `nearestTo` falls back to a recording where one is
	// named, so without this a request pointing at a record gained a drive
	// and an equaliser nobody asked for: the rule is about building a genre
	// from what its players did, and a recording is one performance measured
	// exactly.
	if len(spec.Genre) == 0 {
		return nil
	}

	if got, ok := audio.ShippedGenre(spec.Genre[0]); !ok || !got.Usable {
		return nil
	}

	// The kinds worth a switch, in the order a chain holds them. An amplifier
	// and a cabinet are the sound rather than an effect on it, and a
	// compressor on a bass is set once and left, which is why neither is here
	// and why `compile.Assign` leaves both unassigned.
	kinds := []catalog.Category{catalog.CategoryDrive, catalog.CategoryEQ}

	// Capped by what the device has switches for, since a block past the last
	// switch is a block nobody can reach with a foot and the foot was the
	// whole reason for it. Taken off the list up front rather than tested each
	// time round: the smallest device here has three switches and this names
	// two kinds, so a test inside the loop is a branch nothing can reach, and
	// a branch nothing reaches is a branch nobody has checked. Adding a third
	// kind makes the cap bind again and this keeps working.
	held := plan.LimitsFor(deps.Catalog.Device).Switches
	kinds = kinds[:min(len(kinds), max(held, 0))]

	out := []placed(nil)

	for _, kind := range kinds {
		entry, model, ok := nearestTo(spec, setup, deps, kind, notes)
		if !ok {
			continue
		}

		// Off, whatever the catalog or the corpus would have said.
		off := false
		entry.Enabled = &off

		tuned(&entry, model, spec, deps, notes)

		out = append(out, placed{entry: entry})
	}

	return out
}

// tuned sets a block's controls from its sweep rather than leaving the
// catalog's defaults on it.
//
// A block nobody named arrived at whatever the model ships with, which is a
// value nobody chose for this chain and in most cases not the one the genre
// wants. Every position of every control on 179 of this device's blocks was
// measured, so the value that moves a block toward what is being aimed at is
// a reading.
//
// Silent where there is no sweep for the block, where no sweeps were offered
// at all, or where a control's whole range moved every weighed figure less
// than that figure's own noise. In each case the defaults stand, and the note
// says which happened rather than leaving somebody to wonder why a block
// arrived untouched.
func tuned(
	entry *rig.ChainEntry,
	model catalog.ModelID,
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) {
	if deps.CurvesFor == nil {
		return
	}

	at, ok := target(spec, deps, notes)
	if !ok {
		return
	}

	curves, swept := deps.CurvesFor(model)
	if !swept {
		*notes = append(*notes, Note{
			About: entry.Gear,
			Said: "no sweep for it, so its controls are the catalog's defaults " +
				"rather than set for this",
		})

		return
	}

	wants := ""
	if spec.Instrument != nil {
		wants = string(*spec.Instrument)
	}

	toward := curves.Toward(at.figures, measured.SpectralFor(wants))

	if len(toward) == 0 {
		return
	}

	held := map[string]catalog.Setting{}
	if entry.Controls != nil {
		held = *entry.Controls
	}

	for name, at := range toward {
		// In the control's own kind. A sweep records it, and writing a float
		// where the contract wants a boolean is refused: the Kinky Boost's
		// `Bright` is a boolean and a preset carrying 1.0 on it would not
		// load, which is how four of the eleven genres failed to compile.
		switch at.Kind {
		case "bool":
			held[name] = catalog.Set(catalog.Bool(at.Value != 0))
		case "int":
			held[name] = catalog.Set(catalog.Int(int64(at.Value)))
		default:
			held[name] = catalog.Set(catalog.Float(at.Value))
		}
	}

	entry.Controls = &held

	*notes = append(*notes, Note{
		About:    entry.Gear,
		Said:     fmt.Sprintf("%d of its controls set from its sweep", len(toward)),
		Honoured: true,
	})
}

// namedGear is the blocks a request asked for by name.
func namedGear(
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) ([]placed, error) {
	if spec.Gear == nil {
		return nil, nil
	}

	out := make([]placed, 0, len(*spec.Gear))

	for _, want := range *spec.Gear {
		role := rig.RoleOther
		if want.Role != nil {
			role = rig.Role(*want.Role)
		}

		block, matched := lookup(deps.Catalog, want.Gear, role)
		found := block.ID != ""

		if !found {
			why := "the device has nothing by that name"
			if len(matched) > 1 {
				why = fmt.Sprintf("that name fits %d of the device's models: %s",
					len(matched), names(matched))
			}

			// Refuse rather than substitute is what insist means, and it is
			// what somebody says when the gear is the point rather than the
			// sound.
			if want.Insist != nil && *want.Insist {
				*notes = append(*notes, Note{
					About: want.Gear,
					Said:  why + ", and the request insisted on it",
				})

				return nil, &InsistedError{Gear: want.Gear, Why: why}
			}

			*notes = append(*notes, Note{
				About: want.Gear,
				Said:  why + ", so the compiler will take the nearest it models",
			})
		}

		entry := rig.ChainEntry{Role: role, Gear: want.Gear}

		if found {
			// Said rather than written into the entry. Which model a name
			// resolves to is the plan's answer, and a rig that carried one
			// would be answering it twice.
			said := fmt.Sprintf("resolved to %s", block.ID)

			// And said when the name fitted more than one exactly, because
			// "resolved to X" on its own reads as the only answer when it was
			// one of several. A 1x15 Ampeg B-15 is three models, two of them
			// carrying a microphone list and one not, and which a chain gets
			// decides whether its most powerful control exists at all.
			if len(matched) > 1 {
				said = fmt.Sprintf("%s, and that name fits %d exactly: %s",
					said, len(matched), names(matched))
			}

			*notes = append(*notes, Note{
				About:    want.Gear,
				Said:     said,
				Honoured: true,
			})
		}

		at := placed{entry: entry}

		if want.After != nil {
			behind := rig.Role(*want.After)
			at.after = &behind

			*notes = append(*notes, Note{
				About:    want.Gear,
				Said:     fmt.Sprintf("asked to sit behind the %s", behind),
				Honoured: true,
			})
		}

		out = append(out, at)
	}

	return out, nil
}

// nearestTo picks the block of a category whose measurements sit closest to
// what the request is aiming at.
//
// Only where the request carries something measurable. A recording is the
// case that works today: it is measured through the same figures every block
// was, so the comparison is between two of the same kind of thing.
func nearestTo(
	spec tone.Ask,
	setup tone.Setup,
	deps Deps,
	category catalog.Category,
	notes *Notes,
) (rig.ChainEntry, catalog.ModelID, bool) {
	at, ok := target(spec, deps, notes)
	if !ok {
		return rig.ChainEntry{}, "", false
	}

	want, from := at.figures, at.named

	// The instrument the chain is for. The ask where it says, and otherwise
	// whatever instrumentFor settled on for the rig being written, so the
	// ranking draws from the same half of the catalog the rig claims to be for
	// rather than from a second answer that could disagree with it.
	//
	// Notes are discarded here because instrumentFor has already said whatever
	// it has to say about the Setup, once, where the rig's own instrument was
	// decided. Saying it twice in one report would read as two findings.
	wants := ""
	if spec.Instrument != nil {
		wants = string(*spec.Instrument)
	}

	if wants == "" {
		quiet := Notes{}
		wants = string(instrumentFor(setup, &quiet))
	}

	kept, refused := asPlayed(deps.Catalog, reachable(
		deps.Measured.Nearest(category, want, measured.SpectralFor(wants)),
		setup, deps.Catalog, wants))

	for _, gear := range refused {
		*notes = append(*notes, Note{
			About: string(category),
			Said: fmt.Sprintf(
				"%s read nearest and was refused: it changes the notes rather "+
					"than the tone", gear),
			Honoured: true,
		})
	}

	ranked := wanted(deps.Catalog, kept)
	if len(ranked) == 0 {
		*notes = append(*notes, Note{
			About: string(category),
			Said:  "nothing of that kind has been measured, so none could be chosen",
		})

		return rig.ChainEntry{}, "", false
	}

	best := ranked[0]
	gear := named(deps.Catalog, best)

	*notes = append(*notes, Note{
		About: string(category),
		Said: fmt.Sprintf(
			"%s is the closest of %d measured to %s, %.0f Hz against %.0f",
			gear, len(ranked), from, best.Centroid, want.Centroid),
		Honoured: true,
	})

	// Why this block, written onto the block itself rather than only into the
	// report. The report is read once by whoever ran the build; the document is
	// what everybody downstream has, and a measured choice with nothing behind
	// it reads exactly like a claim somebody made up.
	//
	// It was hand-written before, on every genre rig in the marketplace, and
	// rebuilding them dropped all eleven at once. Nothing but
	// TestEveryShippedGearClaimHasASource said so.
	held := at.because(gear, len(ranked), best.Centroid, wants, category)

	// A measurement off a finished record carries whatever microphone was in
	// front of whatever cabinet, and this block was picked to sit near that
	// figure rather than to reproduce the room. Low confidence because nobody
	// is known to have played it: it is the nearest of what the device models.
	down, weak := tone.CaptureDirect, tone.ConfidenceLow

	return rig.ChainEntry{
		Role:       rig.Role(category),
		Gear:       gear,
		Evidence:   &[]tone.Evidence{held},
		Capture:    &down,
		Confidence: &weak,
	}, catalog.ModelID(best.ID), true
}

// named is the block the way a rig has to say it: the gear it emulates, which
// the measured library does not carry and the catalog does.
//
// The library knows a block by Line 6's name for it, because that is what a
// sweep was filed under. A rig may not: "Cali 400 Ch1" is this manufacturer's
// label for somebody else's amplifier, and a document naming it is one nobody
// can read on other hardware. A block the catalog has no entry for keeps the
// library's name, which is the only one anybody has.
func named(
	cat *catalog.Catalog,
	match measured.Match,
) string {
	if cat == nil {
		return match.Name
	}

	blk, ok := cat.Block(catalog.ModelID(match.ID))
	if !ok {
		return match.Name
	}

	if gear := blk.Gear(); gear != "" {
		return gear
	}

	return match.Name
}

// wanted narrows the candidates to the copies worth ranking.
//
// Line 6 ship the same gear several times and the copies are not equals. An
// amplifier also ships as a preamp with the power stage gone. A cabinet ships
// three times: a legacy impulse response carrying Low Cut, High Cut and Level,
// a mic'd one carrying Mic, Position, Distance and Angle as well, and the mic'd
// one again with a stereo Pan. [catalog.Block.Preferred] is where that order is
// written down, because the gear resolver and the compiler both sort on it.
// This is the third caller and it was ranking without it.
//
// Two rules, and they are separate because one is about copies of one piece of
// gear and the other is about the pool.
//
// A better copy of the same gear wins. Per gear rather than across the pool: 15
// of the 29 bass amplifiers are preamps or legacy ports, and dropping all of
// them lost the Fender Bassman (Silverface) altogether, which this device
// models only as a preamp. A copy with nothing better beside it is the best
// copy there is.
//
// A cabinet with a microphone beats one without. Across the pool, because the
// two are rarely the same cabinet: the nearest cabinet to punk of the 16 this
// device offers for bass was the legacy 1x12 Del Sol, which has no mic'd
// counterpart at all, so no per-gear rule would ever have reached it. It sat
// 86.8 from the target against the 93.9 of the mic'd 1x15 Ampeg B-15, and
// seven units nearer with nothing on it to aim. The microphone list moves a
// cabinet's centroid between 21 and 48Hz on the eight measured for bass,
// against a noise floor of 0.20Hz and the 71.8Hz the whole field spans. The
// cabinet that reads further out is the one that can be brought in.
//
// Asked of the parameters rather than the family, so it is the device's own
// answer to what a block offers. Nothing outside a cabinet carries a Mic, so
// it is no rule at all for an amplifier.
//
// Gear a request names by hand does not come through here. Somebody asking for
// a legacy Ampeg by name has made a decision, and this is not one.
func wanted(
	cat *catalog.Catalog,
	ranked []measured.Match,
) []measured.Match {
	if cat == nil {
		return ranked
	}

	// The best copy of each piece of gear, keyed by the gear rather than the
	// model. Keyed by name where Line 6 state no gear, which is every utility
	// block and no amplifier or cabinet.
	best := map[string]int{}
	offered := false

	for _, match := range ranked {
		block, held := cat.Block(catalog.ModelID(match.ID))
		if !held {
			continue
		}

		if _, has := block.Params[micControl]; has {
			offered = true
		}

		at := block.Preferred()
		if was, seen := best[gearOf(block)]; !seen || at < was {
			best[gearOf(block)] = at
		}
	}

	out := make([]measured.Match, 0, len(ranked))

	for _, match := range ranked {
		block, held := cat.Block(catalog.ModelID(match.ID))
		if !held {
			// Measured and not in the catalog, as everywhere else here: kept,
			// so the drift shows up as a block rather than as an absence.
			out = append(out, match)

			continue
		}

		if block.Preferred() != best[gearOf(block)] {
			continue
		}

		if _, has := block.Params[micControl]; offered && !has {
			continue
		}

		out = append(out, match)
	}

	return out
}

// gearOf is what identifies the gear a block is a copy of.
//
// The real gear where Line 6 say it, because that is what makes two models
// copies of one thing rather than two things: HD2_AmpSVBeastNrm and
// HD2_PreampSVBeastNrm are both an Ampeg SVT. The name otherwise, which is what
// the utility blocks have and no amplifier or cabinet needs.
func gearOf(
	block catalog.Block,
) string {
	if block.BasedOn != "" {
		return block.BasedOn
	}

	return block.Name
}

// asPlayed is the blocks that leave the notes somebody played alone.
//
// A build choosing a block on somebody's behalf ranks the swept library by
// four figures, and four figures cannot tell a fuzz from an octave fuzz. Both
// answer a request for a dirty midrange, and only one of them is still playing
// the notes that went in. Of the 61 drives this device models, five add an
// interval, and against the eleven genres each read second or third rather
// than first: pop-punk's Sub Oct Fuzz 0.69 from the target against the
// winner's 0.21, rock's Octave Fuzz 0.87 against 0.68. Margins a single
// record moves.
//
// So the pool is narrowed before it is ranked rather than the ranking being
// taught something it cannot measure. [catalog.Block.ChangesTheNotes] owns
// which blocks those are and the argument for each.
//
// A block the catalog does not hold is kept, as everywhere else here: that is
// the catalog and the measured library drifting apart, and a shorter chain
// would hide the drift rather than report it.
//
// The second answer is the gear refused ahead of every block that survived,
// which is the refusal that changed what got chosen. Said out loud, because a
// pool that silently lost its nearest reads exactly like one where nothing
// came close.
func asPlayed(
	cat *catalog.Catalog,
	ranked []measured.Match,
) ([]measured.Match, []string) {
	if cat == nil {
		return ranked, nil
	}

	out := make([]measured.Match, 0, len(ranked))

	var ahead []string

	for _, match := range ranked {
		block, held := cat.Block(catalog.ModelID(match.ID))
		if !held || !block.ChangesTheNotes() {
			out = append(out, match)

			continue
		}

		// Nothing has survived yet, so this one read nearer than whatever
		// ends up being chosen.
		if len(out) == 0 {
			ahead = append(ahead, block.Gear())
		}
	}

	return out, ahead
}

// reachable is the blocks somebody could actually play.
//
// An impulse response block carries an index rather than any audio: what is
// in that slot is whatever its owner put there, so a chain naming slot 82
// sounds like one thing on the device it was built on and like something else
// on anybody else's. Choosing one for somebody who has loaded nothing picks a
// block that will be silent.
//
// A setup saying which impulse responses it holds unlocks them again.
//
// And blocks for the other instrument, which is the filter Subcategory's own
// documentation has always claimed: "It decides which half of the catalog a
// request is allowed to draw from." Nothing read it. A bass build ranked all 224
// amplifiers, 173 of them guitar models, and picked whichever sat nearest by
// spectrum: a request for punk on bass chose a Dr Z Interstate Zed, and one
// aimed at a dry bass recording chose a Fender Super Reverb. Twenty-seven of the
// amplifiers are bass models and the Ampeg SVT is among them.
//
// A positive match is required where the category splits by instrument at all,
// and that qualification is the whole of it. Amplifiers are grouped "Guitar" and
// "Bass"; cabinets are grouped "Single, Dual", by how many microphones they
// offer. Requiring a match everywhere excluded every cabinet on this device,
// because none of them claims to be for an instrument.
//
// Where a category does split, the match is strict. Twenty-three amplifiers carry
// no subcategory at all and nothing else in Line 6's data places them: their
// paired cabinets are grouped by microphone count too. Some are bass models and
// some are not, so an unmarked one is a block nobody can place, and this is the
// tool guessing rather than somebody choosing. It declines: the pool is the 27
// amplifiers Line 6 calls bass models, and a GrammaticoLG Jump does not win a
// bass build on missing metadata. Gear a request names by hand goes through
// namedGear and is unaffected, because naming it is a decision and this is not.
func reachable(
	ranked []measured.Match,
	setup tone.Setup,
	cat *catalog.Catalog,
	instrument string,
) []measured.Match {
	loaded := map[string]bool{}

	if setup.Owns != nil {
		for _, held := range *setup.Owns {
			if held.Kind == tone.OwnedIR {
				loaded[strings.ToLower(held.Name)] = true
			}
		}
	}

	// Whether to filter by instrument at all, and on which of the two things
	// Line 6 say it with. Asked of the blocks rather than hardcoded: amps carry
	// a subcategory and cabinets carry a mark, and a device that says it a third
	// way tomorrow is answered by the same question.
	//
	// Decided once, outside the loop, so the test is only ever run where the
	// answer can be no. A nil catalog and an unstated instrument both land here
	// rather than in the test for each block.
	fits := groupedBy(cat, ranked, instrument)

	out := make([]measured.Match, 0, len(ranked))

	for _, match := range ranked {
		if catalog.NeedsUserIR(catalog.ModelID(match.ID)) &&
			!loaded[strings.ToLower(match.Name)] {
			continue
		}

		if fits != nil && !fits(match.ID) {
			continue
		}

		out = append(out, match)
	}

	return out
}

// playable reports whether a block's subcategory is the instrument in hand.
//
// A positive match is required, so a block Line 6 grouped neither way is
// excluded. That is the strict reading argued for in [reachable]: 23 of the
// amplifiers carry no subcategory, some of them bass models and some not, and
// letting an unplaceable one win a bass build is this tool guessing.
//
// A block the catalog does not hold at all is kept, which is a different case:
// that is the catalog and the measured library drifting apart, and a shorter
// chain would hide the drift rather than report it.
//
// Case-insensitive, because an instrument arrives from a document somebody wrote
// and a subcategory from a file Line 6 wrote.
// Called only where the category is grouped by instrument, so a nil catalog and
// an unstated instrument are the caller's business rather than this one's.
func playable(
	cat *catalog.Catalog,
	id string,
	instrument string,
) bool {
	block, held := cat.Block(catalog.ModelID(id))
	if !held {
		// Measured and not in the catalog, which is the two drifting apart. Kept
		// rather than dropped: refusing a block over that would hide the drift
		// behind a chain that is merely shorter.
		return true
	}

	return strings.EqualFold(block.Subcategory, instrument)
}

// groupedBy is the test for whether a block is for the instrument in hand, or
// nil where these blocks are not grouped by instrument at all.
//
// Asked of the candidates rather than stated, because Line 6 say it two
// different ways and which one applies depends on the category. Amplifiers
// carry a subcategory of "Guitar" or "Bass". Cabinets carry "Single, Dual",
// which counts microphones, and their instrument is a separate mark in the
// app's own catalog. A list written here would be a list to keep in step with
// files Line 6 write.
//
// One block is enough to establish either grouping. A category where any model
// says "Bass" is one where the subcategory means that, and a model in it saying
// nothing is a gap. A category where any model is marked for bass is one where
// the mark is maintained, and an unmarked model in it is for the other
// instrument rather than unplaceable: every cabinet is for one or the other,
// and Line 6 mark the 16 that are for bass.
//
// The subcategory wins where both are present, because it is the grouping the
// strict reading in [reachable] was argued on. Nothing on this device carries
// both.
//
// Without this, a bass build ranked all 133 cabinets and chose a 1x12 Fender
// tweed guitar combo for punk, on 167 Hz against the target's 203. The reading
// was right and the pool was wrong.
func groupedBy(
	cat *catalog.Catalog,
	ranked []measured.Match,
	instrument string,
) func(id string) bool {
	if cat == nil || instrument == "" {
		return nil
	}

	var named, marked bool

	for _, match := range ranked {
		block, held := cat.Block(catalog.ModelID(match.ID))
		if !held {
			continue
		}

		for _, is := range []string{"guitar", "bass"} {
			if strings.EqualFold(block.Subcategory, is) {
				named = true
			}
		}

		if block.Bass {
			marked = true
		}
	}

	switch {
	case named:
		return func(id string) bool {
			return playable(cat, id, instrument)
		}
	case marked:
		wants := strings.EqualFold(instrument, string(tone.InstrumentBass))

		return func(id string) bool {
			block, held := cat.Block(catalog.ModelID(id))
			if !held {
				// Measured and not in the catalog, as in [playable]: the two
				// drifting apart, reported as a block in the chain rather than
				// hidden as a block missing from it.
				return true
			}

			return block.Bass == wants
		}
	}

	return nil
}

// aim is what a request is aiming at, and enough about where that came from to
// write the evidence for whatever gets chosen.
//
// The figures and the name were all this carried, and they are all a ranking
// needs. Evidence needs the rest: how many records stand behind the target, how
// many players made them, what their own figures read, and what their records
// earned. Those are known where the target is worked out and nowhere after it,
// so they travel with it rather than being looked up a second time.
type aim struct {
	// figures is the target in the figures a block is measured in.
	figures measured.Figures
	// named is what it came from, as a report and a document should say it: a
	// genre's name, or the path of the recording.
	named string
	// read is the target's own measurements, which go into the evidence as the
	// figures anybody holding the same records can take again. Empty for a
	// target with nothing to report, which is not an error.
	read map[string]float64
	// records and players are how much stands behind it. Zero for a recording,
	// which is one performance and says so by naming itself.
	records, players int
	// earned is the words the target's records earn against everything measured
	// the same way, which is what moves the controls off the corpus medians.
	earned []string
	// caveat is what measuring this kind of target cannot show.
	caveat string
}

// because is the evidence for choosing one block, which is the reasoning the
// ranking already did written where the document keeps it.
//
// Assembled from both halves and neither alone: this value knows what was aimed
// at and [nearestTo] knows what came nearest, so the sentence is built where the
// second half arrives.
// plural is what several blocks of one kind are called, as a sentence says it.
//
// The sentence said "amplifiers" whatever the block was, which was true while
// only an amplifier was chosen by measurement. A genre rig now carries a drive
// and an equaliser picked the same way, and both arrived claiming to be the
// closest of the 57 and the 8 bass amplifiers this device models.
//
// A kind not named here keeps its own word with nothing added. That reads a
// little flatly for a category nobody has thought about yet, and it is better
// than a sentence confidently calling a reverb an amplifier.
func plural(
	kind catalog.Category,
) string {
	switch kind {
	case catalog.CategoryAmp:
		return "amplifiers"
	case catalog.CategoryCab:
		return "cabinets"
	case catalog.CategoryDrive:
		return "drives"
	case catalog.CategoryEQ:
		return "equalisers"
	case catalog.CategoryComp:
		return "compressors"
	}

	return string(kind)
}

func (a aim) because(
	gear string,
	ranked int,
	sat float64,
	instrument string,
	kind catalog.Category,
) tone.Evidence {
	// The kind of block rather than the word "amplifiers", which this said for
	// every one of them. A genre rig now carries a drive and an equaliser
	// chosen the same way, and both arrived claiming to be the closest of the
	// 57 and the 8 bass amplifiers this device models.
	said := fmt.Sprintf(
		"Chosen by measurement rather than by anybody's say-so. %s is the "+
			"closest of the %d %s %s this device models to %s, sitting "+
			"at %.0f Hz against %.0f.",
		gear, ranked, instrument, plural(kind), a.named, sat, a.figures.Centroid)

	if a.records > 0 {
		// Counted straight, with no pluraliser. The only target that fills these
		// is a genre and [audio.Genre.Usable] holds one to eight records from
		// three players before it may be aimed at, so neither can read "1
		// records". A recording leaves them zero and skips this, since it is one
		// performance and the name above says which.
		//
		// The name goes mid-sentence rather than first. A genre is spelled the
		// way the ask spells it, which is lower case, and starting a sentence
		// with it reads as a typo.
		said += fmt.Sprintf(" Across %d records from %d players, %s ",
			a.records, a.players, a.named)

		// A target that earns nothing says so. It is the ordinary answer for
		// something sitting where everything else sits, and a note that goes
		// quiet instead reads as though the words were simply not looked up.
		if len(a.earned) == 0 {
			said += "earns no word against the players who play none of it, so " +
				"nothing moves the controls off the medians this instrument's " +
				"presets sit at."
		} else {
			said += fmt.Sprintf("earns %s, measured against the players who "+
				"play none of it, and that is what moves the controls off the "+
				"medians this instrument's presets sit at.", said3(a.earned))
		}
	}

	held := tone.Evidence{Kind: tone.EvidenceAudio, Note: &said}

	if len(a.read) > 0 {
		read := a.read
		held.Measured = &read
	}

	if a.caveat != "" {
		why := a.caveat
		held.Caveat = &why
	}

	return held
}

// said3 is a list the way a sentence says one, because these notes are read as
// prose and "scooped, dark, long-decay" is not a sentence.
func said3(
	words []string,
) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}

	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// target is what a request is aiming at, in the figures a block is measured
// in, and what it came from.
func target(
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) (aim, bool) {
	if spec.Like == nil || spec.Like.Recording == nil {
		// A player, where one is named and their records are measured.
		//
		// Narrower than a genre and wider than a recording, which is the order
		// these three sit in: one performance measured exactly, then one
		// player's body of work, then the middle of a population. So it
		// answers where a recording does not and before a genre does.
		//
		// This is what an artist rig was missing. All 32 named the amplifier a
		// source puts somebody on and then carried the catalog's defaults on
		// every control: `dee-dee-ramone` sat at Bass 0.41, Bias 0.74, ChVol
		// 0.8 and Drive 0.6, which is the factory setting to the digit. The
		// records are listed on the document for exactly this and nothing read
		// them.
		if at, ok := playerTarget(spec, deps, notes); ok {
			return at, true
		}

		// A genre is the weaker target and the one almost every ask carries, so
		// it answers where a recording does not rather than instead of one. A
		// record is one performance measured exactly; a genre is the middle of a
		// population, which is less precise and still a measurement.
		return genreTarget(spec, deps, notes)
	}

	at := *spec.Like.Recording

	f, err := os.Open(at) //nolint:gosec // the path is the requester's own file
	if err != nil {
		*notes = append(*notes, Note{
			About: at,
			Said:  fmt.Sprintf("cannot be read, so nothing was measured from it: %v", err),
		})

		return aim{}, false
	}

	defer func() { _ = f.Close() }()

	samples, rate, err := audio.Read(f)
	if err != nil {
		*notes = append(*notes, Note{
			About: at,
			Said:  fmt.Sprintf("is not audio this can read: %v", err),
		})

		return aim{}, false
	}

	// Through the same code every block went through, which is the only
	// reason the two are comparable at all.
	read := audio.Measure(samples, rate)

	return aim{
		figures: measured.Figures{
			Low:      100 * read.Low,
			Mid:      100 * read.Mid,
			High:     100 * read.High,
			Centroid: read.Centroid,
		},
		named: at,
		read:  read.Measured(),
		// No record or player count, because a recording is one performance and
		// the path above names which. The caveat is the one thing a reader needs
		// that the figures do not say.
		caveat: "A recording measured as it was finished, so the figures " +
			"carry the amplifier, the microphone, the desk and the master as " +
			"well as the player. This block is what sits nearest those " +
			"figures, not one anybody is known to have used.",
	}, true
}

// genreTarget is the middle of a genre's records, for an ask that names no
// recording.
//
// This is what lets a request made of nothing but a genre and some adjectives
// resolve. It used to be refused with "nothing in it can be measured against",
// which was true of the code and not of the data: the figures a genre is measured
// to are the same four [measured.Nearest] ranks the device's amplifiers on, and
// they already ship.
//
// The first genre that can answer, in the order the ask names them, so a sound
// that is both punk and pop-punk aims at the one it was named for first. Taking
// the mean of two populations would invent a third that nobody measured.
//
// Two guards, both the ones genreWords already applies. Under the threshold is
// reported and never computed from: eight records from three players, or the
// genre is one band's sound wearing a genre's name. And a genre measured on
// another instrument is refused rather than used weakly, because a bass
// centroid sits an octave below a guitar's and the nearest amplifier to the
// wrong octave is not a weaker answer, it is a different question.
func genreTarget(
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) (aim, bool) {
	wants := ""
	if spec.Instrument != nil {
		wants = string(*spec.Instrument)
	}

	// No guard against an empty name. The contract holds each genre to
	// `minLength: 1` and `pattern: "\S"`, so a blank one does not load, and a
	// branch no document can reach is a branch nothing can test.
	for _, named := range spec.Genre {
		got, ok := audio.ShippedGenre(slug.Of(named))
		if !ok || !got.Usable {
			continue
		}

		if wants != "" && got.Instrument != "" && wants != got.Instrument {
			*notes = append(*notes, Note{
				About: named,
				Said: fmt.Sprintf("is measured on %s and this is for %s, so its "+
					"figures aim at nothing here", got.Instrument, wants),
			})

			continue
		}

		// No second check on the comparison population. Usable carries it, so a
		// genre that passed the test above has somewhere else to be measured
		// against by construction.
		earned := make([]string, 0, len(got.Terms))
		for _, term := range got.Terms {
			earned = append(earned, term.Term)
		}

		return aim{
			figures: displacedTo(got, deps.Measured.Baseline),
			named:   named,
			read:    got.Across.Measured(),
			records: got.Records,
			players: got.Players,
			earned:  earned,
			// The same caveat for every genre, because it is about the method
			// rather than about the genre: it says what measuring finished
			// records cannot separate, and that the answer moves when the
			// corpus gains players.
			caveat: "A genre measured off finished records, so the figures " +
				"carry the amplifier, the microphone, the desk and the master " +
				"of every one of them. This block is what sits nearest " +
				"those figures, not one anybody is known to have used. It " +
				"also moves: a genre is measured against the players who play " +
				"none of it, so this answer changes when the corpus gains " +
				"players, and it is worth rebuilding rather than trusting " +
				"indefinitely.",
		}, true
	}

	return aim{}, false
}

// displacedTo is the genre as a target a block can be ranked against.
//
// Not the genre's own figures. Those are measured off finished records and a
// block's are measured off a dry signal pushed through it, and the two do not
// subtract: punk reads 97.1% of its energy low where the dry signal going into
// the pedal holds 90.8% before any block touches it. Asking which block reaches
// 97.1% asks for bottom that is not in the input, every candidate is out of
// range, and the nearest becomes whichever is darkest. That is how a request for
// punk chose an Ampeg B-15NF, a Motown flip-top.
//
// So the genre is read as a displacement and applied to the signal the blocks
// were measured with. Punk sits 0.021 of the energy lower and 9.6Hz brighter
// than the records of players who hold none of it, and the target is the
// baseline shifted by exactly that: 92.9% low at 164.9Hz, which blocks can
// reach. Both sides are then "how far from its own normal", which is the
// comparison [audio.Genre.Terms] already earns a word from.
//
// Shares convert and the centroid does not. A genre holds a share of the energy
// from zero to one; the measured library reports the same figure as a
// percentage.
func displacedTo(
	got audio.Genre,
	baseline measured.Figures,
) measured.Figures {
	at := got.Across.Measured()
	wide := got.Across.Spreads()

	// Which way each word the genre earns points, by figure. `Mine` above
	// `Others` is a word for more of something and below it is a word for
	// less, which is the same comparison the word was earned by.
	toward := map[audio.Figure]float64{}

	for _, term := range got.Terms {
		if term.Mine > term.Others {
			toward[term.Key] = 1
		} else if term.Mine < term.Others {
			toward[term.Key] = -1
		}
	}

	shift := func(key audio.Figure, scale float64) (float64, bool) {
		mine, held := at[string(key)]
		theirs, also := got.Elsewhere[key]

		if !held || !also {
			return 0, false
		}

		// The edge of the genre's own range rather than its middle, on a
		// figure a word points at.
		//
		// The middle is what every genre shares. Totalled over the three band
		// shares, ten of the eleven genres this corpus can aim at displace
		// between 0.2 and 6.4 points from one baseline, where the blocks they
		// are ranked against span 50, so ten of eleven targets are the same
		// point and 9 of the 11 rigs built from them carried one amplifier.
		//
		// A percentile of the genre's own records is still one of its records,
		// the one that sounds most like the word, so this aims at the most
		// mid-forward punk rather than at the average of all punk. It stays a
		// displacement from the records elsewhere, which is the only thing
		// that makes a genre and a block comparable at all, and the reasoning
		// above this function for that is unchanged.
		//
		// A figure no word points at keeps the middle. Nothing says which end
		// of it would be more like the genre.
		if way, pointed := toward[key]; pointed {
			if spread, ranged := wide[string(key)]; ranged {
				if way > 0 {
					mine = spread.High
				} else {
					mine = spread.Low
				}
			}
		}

		return scale * (mine - theirs), true
	}

	out := baseline

	for _, want := range []struct {
		key   audio.Figure
		scale float64
		onto  *float64
	}{
		{audio.KeyLow, 100, &out.Low},
		{audio.KeyMid, 100, &out.Mid},
		{audio.KeyHigh, 100, &out.High},
		{audio.KeyCentroid, 1, &out.Centroid},
	} {
		by, ok := shift(want.key, want.scale)
		if !ok {
			continue
		}

		*want.onto += by
	}

	return out
}

// playerTarget is what one player's own records measure as.
//
// A displacement, like a genre's and for the same reason: a player's figures
// come off finished records and a block's off a dry signal through one block,
// so the only comparable thing is how far each sits from its own normal.
// `Player.Elsewhere` is what the other players measure as and is packed
// beside the figures for this.
//
// **The median rather than the edge of their range, which is the opposite of a
// genre.** A genre aims at its edge because the middle is what every genre
// shares, and ten of the eleven displace less than 6.4 points from one
// baseline. One player's records are a much tighter distribution describing one
// person, so their middle is the thing being aimed at rather than the thing
// that hides them. Where a player's own records disagree more than players
// disagree with each other, `Scattered` says so and the figures describe none
// of their sounds; that is reported rather than aimed at.
//
// Answers false where nobody is named, where the name is not a player, or
// where the pack does not carry them, and the caller then falls through to the
// genre. A rig for somebody nobody has measured keeps the controls its sources
// justify, which is the honest answer.
func playerTarget(
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) (aim, bool) {
	// The person the ask says to sound like, and only that.
	//
	// **Not `subject`**, which says who the ask is *for*. A subject-only ask
	// refuses and names `like` instead, deliberately: a rig for Mike Dirnt
	// resolving itself from the rig researched for Mike Dirnt is circular, and
	// `TestAnAskMayNameAPlayer` holds that refusal with the note that explains
	// which field somebody wanted.
	//
	// Reading a subject here broke that: a target made the measurements
	// answer, so a refusal became a build. The two are separable in principle,
	// since a rig for somebody aiming at their own records is not circular, and
	// separating them means changing where the refusal is decided rather than
	// quietly removing it.
	//
	// So an artist rig's own records are not reached from here. The gear in one
	// is authored and the resolve never passes through `chainFor`, so the
	// target it needs belongs on the resolve path where the document's own
	// identifier is known. Task 268 carries that.
	if spec.Like == nil || spec.Like.Artist == nil || *spec.Like.Artist == "" {
		return aim{}, false
	}

	who := slug.Of(*spec.Like.Artist)

	got, ok := audio.ShippedPlayer(who)
	if !ok {
		*notes = append(*notes, Note{
			About: who,
			Said: "no records of theirs are measured, so their controls are " +
				"whatever their sources justify rather than aimed at anything",
		})

		return aim{}, false
	}

	// No guard on `Elsewhere` being absent. `Corpus` fills it from the other
	// players and all 43 in this corpus carry it, so a branch for one player
	// alone is one nothing can reach and nobody has checked. `displacedTo`
	// already answers that case by leaving the baseline where it is, which is
	// the right answer: there is nothing to read them against.

	// Their records' own scatter, said rather than hidden. A player whose
	// records are several different sounds has figures describing none of them.
	for _, at := range got.Scattered {
		*notes = append(*notes, Note{
			About: who,
			Said: fmt.Sprintf(
				"their own records disagree about %s more than players disagree "+
					"with each other, so that figure describes none of their "+
					"sounds", at.Key),
		})
	}

	earned := make([]string, 0, len(got.Terms))
	for _, term := range got.Terms {
		earned = append(earned, term.Term)
	}

	figures, _ := AimAtPlayer(who, deps.Measured.Baseline)

	return aim{
		figures: figures,
		named:   who,
		read:    got.Across.Measured(),
		records: got.Records,
		players: 1,
		earned:  earned,
		caveat: "A player measured off their own finished records, so the " +
			"figures carry the amplifier, the microphone, the desk and the " +
			"master of every one of them. The gear is what their sources say " +
			"they played and is not moved by this; only the controls on it are " +
			"set, and they are set to what those records measure as.",
	}, true
}

// AimAtGenre is what a genre's records measure as, as a displacement a block
// can be ranked against.
//
// The sibling of [AimAtPlayer] and the same arithmetic, with one difference
// that is the whole of task 310: **a genre aims at the edge of its own range
// and a player at the middle of theirs.** The middle is what every genre
// shares, and totalled over the three band shares ten of the eleven displace
// between 0.2 and 6.4 points from one baseline where the blocks they are
// ranked against span 50. So `displacedTo` is given the genre's terms, which
// is what makes it take an edge, and a player's are deliberately withheld.
//
// Exported for the reason [AimAtPlayer] is: a rig's identifier is who it is
// for, and three of the eleven genre rigs borrow their amplifier's own
// pairing, so their cabinet is added by the compiler and never passes through
// the gear-choosing path here. See task 242.
//
// False where the pack carries no genre by that name, which every artist rig
// and every slug that is not one of the eleven answers with.
func AimAtGenre(
	slug string,
	baseline measured.Figures,
) (measured.Figures, bool) {
	got, ok := audio.ShippedGenre(slug)
	if !ok || got.Instrument == "" {
		return measured.Figures{}, false
	}

	return displacedTo(got, baseline), true
}

// AimAtPlayer is what one player's own records measure as, as a displacement
// a block can be ranked against.
//
// A displacement, like a genre's and for the same reason: a player's figures
// come off finished records and a block's off a dry signal through one block,
// so the only comparable thing is how far each sits from its own normal.
// [audio.Player.Elsewhere] is what the other players measure as and is packed
// beside the figures for this.
//
// **The middle of their records rather than an edge of them, which is the
// opposite of a genre.** It passes no terms to `displacedTo`, and that is what
// makes it take the middle: a genre aims at its edge because the middle is
// what every genre shares and ten of the eleven displace less than 6.4 points
// from one baseline, while one player's records describe one person.
//
// Exported because two paths need the same answer and the arithmetic has one
// owner. An ask saying `like: <artist>` arrives through `playerTarget`; an
// artist rig's gear is authored, so its resolve never reaches that and asks
// from the resolve path instead, where the document's own identifier says who
// it is for.
//
// False where the pack carries nobody by that name, and the caller then keeps
// whatever the rig's sources justify. The name is slugged, because the pack
// files a player under the directory their records sit in.
func AimAtPlayer(
	who string,
	baseline measured.Figures,
) (measured.Figures, bool) {
	got, ok := audio.ShippedPlayer(slug.Of(who))
	if !ok {
		return measured.Figures{}, false
	}

	return displacedTo(audio.Genre{
		Across:    got.Across,
		Elsewhere: got.Elsewhere,
	}, baseline), true
}

// genreNote says what is known about a genre somebody asked for.
//
// Four answers rather than one, because "cannot answer that" was true of every
// genre and is now true of some. What is measured ships in the binary, so this
// needs no audio: see audio.Shipped.
//
// Given what was found rather than finding it, so every one of the four has a
// test. Only three are reachable through the shipped data, and which three
// depends on whichever records somebody has tagged.
func genreNote(
	want string,
	got audio.Genre,
	ok bool,
	wants string,
) Note {
	switch {
	case !ok:
		return Note{
			About: want,
			Said: "no records carry that genre, so there is no distribution " +
				"to aim at. Tag some and measure them",
		}
	case got.Instrument == "":
		// A genre pooled across two instruments. Its centre of gravity sits
		// between them and describes neither.
		return Note{
			About: want,
			Said: fmt.Sprintf(
				"the %d records carrying that genre were not all played on one "+
					"instrument, so their figures describe none of them and "+
					"nothing may aim at them",
				got.Records),
		}
	case wants != "" && wants != got.Instrument:
		// The one that would otherwise converge and be wrong. A bass corpus
		// puts every genre's centroid between 90 and 182Hz, and a guitar chain
		// solved against that is not near it: it is being asked to sound like
		// another instrument, and every dial would be spent doing it.
		return Note{
			About: want,
			Said: fmt.Sprintf(
				"that genre is measured on %s and this asks for %s, so its "+
					"figures are not a target for this: %s sits about an octave "+
					"from %s and the solve would spend every control getting "+
					"there. Measure %s records carrying it",
				got.Instrument, wants, got.Instrument, wants, wants),
		}
	case !got.Usable:
		return Note{
			About: want,
			Said: fmt.Sprintf(
				"%d records from %d players carry that genre, under the eight "+
					"from three it takes to aim at one: fewer is a band's sound "+
					"wearing a genre's name",
				got.Records, got.Players),
		}
	case len(got.Terms) == 0:
		// The honest and least expected answer. Punk clears the threshold on
		// this corpus and sits inside the middle half of everything else on
		// every axis, so there is nothing to aim at even though there is plenty
		// behind it.
		return Note{
			About: want,
			Said: fmt.Sprintf(
				"%d records from %d players carry that genre and none of the "+
					"figures set it apart from the players who play none of it, "+
					"so there is nothing to aim at",
				got.Records, got.Players),
		}
	default:
		return Note{
			About: want,
			Said: fmt.Sprintf("measured across %d records from %d players as %s",
				got.Records, got.Players, strings.Join(termsOf(got.Terms), ", ")),
		}
	}
}

// termsOf names what a genre earned, in the order it earned them.
func termsOf(
	of []audio.Derived,
) []string {
	out := make([]string, 0, len(of))
	for _, t := range of {
		out = append(out, t.Term)
	}

	return out
}

// answeredBy is every name the chain was built from, so nothing reports a name
// as unanswered that a block came from.
//
// Read off the notes rather than tracked separately: a note that honoured a name
// is exactly the record of that name having been answered, and a second list
// would be a second thing to keep in step.
func answeredBy(
	notes Notes,
) map[string]bool {
	out := map[string]bool{}

	for _, n := range notes {
		if n.Honoured {
			out[n.About] = true
		}
	}

	return out
}

// unresolved says which parts of a request this cannot yet answer.
func unresolved(
	spec tone.Ask,
	answered map[string]bool,
	notes *Notes,
) {
	// What the ask says it is for, which is what a genre's figures have to
	// agree with. Empty when the request leaves it to the Setup, and then
	// nothing is claimed either way.
	wants := ""
	if spec.Instrument != nil {
		wants = string(*spec.Instrument)
	}

	// Every genre the ask names, because a record belongs to more than one and
	// the corpus tags the same players both punk and pop-punk. Each is reported
	// on its own: one may have enough records behind it to compute from while
	// another has three by one band, and collapsing them would hide which.
	for _, named := range spec.Genre {
		if named == "" {
			continue
		}

		got, ok := audio.ShippedGenre(slug.Of(named))
		*notes = append(*notes, genreNote(named, got, ok, wants))
	}

	// Named in a fixed order, because ranging a map is not one and a request
	// naming both a band and a song would report them differently each run.
	if spec.Like != nil {
		for _, named := range []struct {
			what string
			who  *string
		}{
			{"artist", spec.Like.Artist},
			{"band", spec.Like.Band},
			{"song", spec.Like.Song},
		} {
			if named.who == nil || *named.who == "" {
				continue
			}

			// A rig researched for them answered it, so there is nothing
			// unresolved about the name.
			if answered[*named.who] {
				continue
			}

			*notes = append(*notes, Note{
				About: *named.who,
				Said: fmt.Sprintf(
					"naming %s an %s aims at their records, which this does "+
						"not read yet; hand it a recording instead",
					*named.who, named.what),
			})
		}
	}

	// A subject names who the ask is for; `like` names what to aim at. The two
	// are different claims and the contract keeps them apart, so a subject does
	// not resolve gear and should not: a marketplace pair says who it is for on
	// the ask and what answered on the rig beside it.
	//
	// Said out loud, because the resemblance is close enough that somebody
	// writing `subject: { kind: artist, name: Mike Dirnt }` and getting a
	// refusal has no way to see which field they wanted.
	if spec.Subject != nil && spec.Subject.Name != "" && spec.Like == nil {
		*notes = append(*notes, Note{
			About: spec.Subject.Name,
			Said: "is who the ask is for, which aims at nothing. `like: " +
				"{ artist: " + spec.Subject.Name + " }` is the field that " +
				"resolves their rig",
		})
	}

	if spec.Nudges != nil && len(*spec.Nudges) > 0 {
		*notes = append(*notes, Note{
			About: "nudges",
			Said: "a nudge moves from wherever the last answer landed, and " +
				"this builds a first answer, so there is nothing to move from",
		})
	}

	corrected(spec, notes)
}

// corrected reports a correction history this rebuild does not carry.
//
// A correction's changed paths point into the plan it was made against, and
// this builds a new one from the ask. So the settings a correction arrived at
// are not in the answer, and the entries are worth reading out rather than
// sitting in the file unmentioned: they are the only record of what somebody
// heard, and the reason a knob was where it was.
//
// An entry with no verdict is the one that matters most. It has been built and
// not yet listened to, which is the state a person needs shown rather than
// left to be rediscovered.
func corrected(
	spec tone.Ask,
	notes *Notes,
) {
	if spec.Corrections == nil {
		return
	}

	for _, was := range *spec.Corrections {
		said := "corrected before, and this rebuild does not replay it"
		if was.Verdict != nil && *was.Verdict != "" {
			said = fmt.Sprintf("heard as %q, and this rebuild does not "+
				"replay it", *was.Verdict)
		} else if was.Changed != nil && len(*was.Changed) > 0 {
			said = fmt.Sprintf("%d setting(s) moved for it and nobody has "+
				"said what it sounded like", len(*was.Changed))
		}

		*notes = append(*notes, Note{About: was.Ask, Said: said})
	}
}

// lookup finds a block by the name a person uses.
//
// Loosely, because somebody writes "Ampeg SVT" where the catalog says
// "Ampeg SVT® (normal channel)", and exactly enough that a name matching two
// blocks in the same role answers with neither.
//
// Three tiers, each consulted only where the one above it found nothing. The
// first two match Line 6's own name for the block and the third is
// catalog.Block.Matches, which also reads the real-world gear the block
// emulates, strips a manufacturer's ® and matches word by word.
//
// Tiered rather than merged, because the tiers answer different questions and a
// wider one must not widen a narrower one's answer. "Ampeg SVT" names four
// models by Line 6's name and that ambiguity is the true answer; letting
// BasedOn in beside it would add more and change what every committed document
// resolves to.
//
// The third tier is the one this format is for, and it was missing. A rig names
// real-world gear and never a Line 6 model, and this matched the model name
// only: "Paul Cochrane Timmy Overdrive" found nothing while the catalog's own
// matcher accepted it, because Line 6 call that block "Teemah!". Every name
// that worked did so by Line 6 having chosen the gear's name as their own, and
// gear where they did not was unreachable. Worse than a bad note: `insist`
// refuses on this answer, so insisting on gear the device models was refused.
func lookup(
	cat *catalog.Catalog,
	want string,
	role rig.Role,
) (catalog.Block, []catalog.Block) {
	var exact, partial, emulated []catalog.Block

	target := strings.ToLower(want)

	for _, block := range cat.Blocks {
		if role != rig.RoleOther && string(block.Category) != string(role) {
			continue
		}

		switch name := strings.ToLower(block.Name); {
		// An exact name wins outright over a partial one, which is how "Ampeg
		// SVT" reaches the one called that rather than the one called "Ampeg
		// SVT Bright". Collected rather than returned on sight: a name is
		// exactly right for more than one model far more often than it looks,
		// and returning the first walked handed back whichever this map
		// happened to yield.
		case name == target:
			exact = append(exact, block)
		case strings.Contains(name, target):
			partial = append(partial, block)
		case block.Matches(want):
			emulated = append(emulated, block)
		}
	}

	matched := emulated

	switch {
	case len(exact) > 0:
		matched = exact
	case len(partial) > 0:
		matched = partial
	}

	// Sorted before anything is chosen from it, so one request answers one way.
	// 355 of this device's 661 models share a name with another of their own
	// category, and unsorted the same ask compiled to three different cabinets
	// across twelve runs: two of them carrying a microphone list and one of
	// them not, at 2.5 DSP against 7.2.
	//
	// By family first, because which of two models a name means is a decision
	// and the alphabet is not one. catalog.Block.Preferred holds that decision,
	// shared with the compiler's own sort so the two cannot answer "which Ampeg
	// SVT" differently. A model identifier still breaks the tie inside a family,
	// where nothing else distinguishes them.
	//
	// A rig cannot pin a model and is not supposed to: it names gear a person
	// recognises, which is what makes it portable. A plan names the model
	// outright, which is what `presets compile --plan` takes and `tone tune
	// --out` writes, so anything needing the legacy cabinet says so there.
	sort.Slice(matched, func(i, j int) bool {
		if a, b := matched[i].Preferred(), matched[j].Preferred(); a != b {
			return a < b
		}

		return matched[i].ID < matched[j].ID
	})

	if len(matched) == 1 || len(exact) > 0 {
		return matched[0], matched
	}

	return catalog.Block{}, matched
}

// names lists what a gear name matched, for a message somebody can act on.
//
// By model identifier rather than by name, because the names are what
// collided: 661 models share 468 names, so a note listing four called "Ampeg
// SVT Nrm" tells nobody which to ask for. A model identifier is exact.
//
// Exact and, today, unaskable: no field on either contract takes one, so this
// tells somebody which model they got and not how to ask for another. That is
// why 92 mic'd cabinets are unreachable, every one of them sharing its name
// with a legacy model.
//
// Capped, because some names fit a dozen and a note listing all of them is
// one nobody reads.
func names(
	matched []catalog.Block,
) string {
	const most = 4

	out := make([]string, 0, most)

	for i, block := range matched {
		if i == most {
			out = append(out, fmt.Sprintf("and %d more", len(matched)-most))

			break
		}

		out = append(out, string(block.ID))
	}

	return strings.Join(out, ", ")
}

// holds reports whether a chain already has a block in a role.
func holds(
	chain []placed,
	role rig.Role,
) bool {
	for _, at := range chain {
		if at.entry.Role == role {
			return true
		}
	}

	return false
}

// order is where a role sits in a signal path.
//
// The ordinary shape of a chain rather than an opinion about anybody's: a
// compressor and a drive ahead of the amplifier, time and space behind it.
// A rig somebody wrote keeps its own order, because whoever wrote it made
// that decision; this is for a request that made none.
func order(
	role rig.Role,
) int {
	at := map[rig.Role]int{
		rig.RoleWah: 0, rig.RoleFilter: 1, rig.RoleComp: 2, rig.RoleDrive: 3,
		rig.RoleAmp: 4, rig.RoleCab: 5, rig.RoleEQ: 6, rig.RoleMod: 7,
		rig.RoleDelay: 8, rig.RoleReverb: 9,
	}

	if where, known := at[role]; known {
		return where
	}

	return 10
}

// instrumentFor is what the rig is played on.
//
// From the setup rather than from the request, because what somebody owns
// changes when they buy something and what they want changes every time they
// ask. A setup naming several says which to assume; one naming none is a bass
// here, and says so.
func instrumentFor(
	setup tone.Setup,
	notes *Notes,
) rig.Instrument {
	if setup.Instruments != nil {
		for _, held := range *setup.Instruments {
			if held.Default != nil && *held.Default {
				return gearIsFor(held, notes)
			}
		}

		if len(*setup.Instruments) > 0 {
			return gearIsFor((*setup.Instruments)[0], notes)
		}
	}

	// Honoured, because an assumption that was made is a thing that happened.
	// It read "could not" beside the device note saying the same thing about
	// the same absent setup, which told somebody the tool had failed at
	// something it had in fact decided.
	*notes = append(*notes, Note{
		About:    "instrument",
		Said:     "the setup names none, so this is for bass",
		Honoured: true,
	})

	return rig.InstrumentBass
}

// gearIsFor is which kind of instrument a setup entry describes.
//
// Guitar where the gear's name does not say bass, and the note says so rather
// than the assumption being silent. pkg/cli's referenceIsFor asks the same
// shape of question of a reference recording's filename and answers nothing
// where it cannot tell: a measurement may claim no instrument, and a rig's
// instrument field may not be empty. Neither is named for the question they
// share, because merging them would make one of the two answers wrong.
func gearIsFor(
	held tone.Held,
	notes *Notes,
) rig.Instrument {
	name := strings.ToLower(held.Gear)

	if strings.Contains(name, "bass") {
		return rig.InstrumentBass
	}

	*notes = append(*notes, Note{
		About:    held.Gear,
		Said:     "read as a guitar rather than a bass",
		Honoured: true,
	})

	return rig.InstrumentGuitar
}
