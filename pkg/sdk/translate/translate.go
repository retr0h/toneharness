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
// The layer between the two documents a person writes and the one the tool
// writes. A ToneSpec says what they want and a Setup says what they have,
// both in words anybody would use; a RigSpec says exactly which models go
// where, and is meant to be read by the compiler rather than by a person.
//
//	ToneSpec + Setup   what we mean, and what we have
//	      ↓            this package
//	RigSpec            exact, resolved, deterministic, shareable
//	      ↓            pkg/sdk/internal/compile
//	.hlx               what the pedal eats
//
// The split earns itself at the second step. A RigSpec is the thing worth
// sharing, because two people compiling the same one get the same preset, and
// a ToneSpec is not: "punk, a bit darker" resolves against a corpus and a
// library of measurements that both move.
//
// Nothing here guesses quietly. Every choice this makes, and every part of
// the ask it could not honour, comes back in the notes.
package translate

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
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
	// this file is ToneSpec to translate to RigSpec to compile, and importing
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
// Deterministic given the same catalog, the same measurements and the same
// two documents. That is what makes a RigSpec worth sharing where a ToneSpec
// is not: the ask resolves against things that move, and the answer does not.
func Translate(
	spec tone.Spec,
	setup tone.Setup,
	deps Deps,
) (rig.Spec, Notes, error) {
	var notes Notes

	if err := agrees(setup, deps, &notes); err != nil {
		return rig.Spec{}, notes, err
	}

	out := rig.Spec{
		Schema:     rig.SchemaName,
		ID:         identify(spec),
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
	spec tone.Spec,
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
	spec tone.Spec,
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
	spec tone.Spec,
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
	spec tone.Spec,
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

	if !holds(named, rig.RoleAmp) {
		if amp, ok := nearestTo(spec, setup, deps, "amp", notes); ok {
			named = append([]placed{{entry: amp}}, named...)
		}
	}

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
	spec tone.Spec,
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
// that asked. Kept beside the entry rather than on it, because a RigSpec's
// chain is already in order by the time it is written and carrying the reason
// would be carrying the question into the answer.
type placed struct {
	entry rig.ChainEntry
	// after is the role this was asked to sit behind, where one was named.
	after *rig.Role
}

// namedGear is the blocks a request asked for by name.
func namedGear(
	spec tone.Spec,
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
	spec tone.Spec,
	setup tone.Setup,
	deps Deps,
	category catalog.Category,
	notes *Notes,
) (rig.ChainEntry, bool) {
	want, from, ok := target(spec, deps, notes)
	if !ok {
		return rig.ChainEntry{}, false
	}

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

	ranked := reachable(
		deps.Measured.Nearest(category, want, measured.Spectral()),
		setup, deps.Catalog, wants)
	if len(ranked) == 0 {
		*notes = append(*notes, Note{
			About: string(category),
			Said:  "nothing of that kind has been measured, so none could be chosen",
		})

		return rig.ChainEntry{}, false
	}

	best := ranked[0]

	*notes = append(*notes, Note{
		About: string(category),
		Said: fmt.Sprintf(
			"%s is the closest of %d measured to %s, %.0f Hz against %.0f",
			best.Name, len(ranked), from, best.Centroid, want.Centroid),
		Honoured: true,
	})

	return rig.ChainEntry{
		Role: rig.Role(category),
		Gear: best.Name,
	}, true
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

	// Whether to filter by instrument at all, asked of the blocks rather than
	// hardcoded: amps are grouped by it and cabs are not, and a device that
	// groups something else tomorrow is answered by the same question.
	//
	// Decided once, outside the loop, so playable is only ever called where the
	// answer can be no. A nil catalog and an unstated instrument both land here
	// rather than in the test for each block.
	split := instrument != "" && splitsByInstrument(cat, ranked)

	out := make([]measured.Match, 0, len(ranked))

	for _, match := range ranked {
		if catalog.NeedsUserIR(catalog.ModelID(match.ID)) &&
			!loaded[strings.ToLower(match.Name)] {
			continue
		}

		if split && !playable(cat, match.ID, instrument) {
			continue
		}

		out = append(out, match)
	}

	return out
}

// playable reports whether a block is for the instrument in hand.
//
// False only where the catalog says the other one. A block the catalog does not
// hold, or holds without a subcategory, is kept: this is a filter on what Line 6
// grouped rather than a claim about what sounds right, and refusing a model
// because a field is empty would be refusing it for no reason.
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

// splitsByInstrument reports whether these blocks are grouped by what they are
// played with.
//
// Asked of the candidates rather than stated, because the answer differs by
// category and a list written here would be a list to keep in step with a file
// Line 6 writes. One block claiming an instrument is enough: a category where
// any model says "Bass" is one where the grouping means that, and a model in it
// saying nothing is a gap rather than a different kind of thing.
func splitsByInstrument(
	cat *catalog.Catalog,
	ranked []measured.Match,
) bool {
	if cat == nil {
		return false
	}

	for _, match := range ranked {
		block, held := cat.Block(catalog.ModelID(match.ID))
		if !held {
			continue
		}

		for _, named := range []string{"guitar", "bass"} {
			if strings.EqualFold(block.Subcategory, named) {
				return true
			}
		}
	}

	return false
}

// target is what a request is aiming at, in the figures a block is measured
// in, and what it came from.
func target(
	spec tone.Spec,
	deps Deps,
	notes *Notes,
) (measured.Figures, string, bool) {
	if spec.Like == nil || spec.Like.Recording == nil {
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

		return measured.Figures{}, "", false
	}

	defer func() { _ = f.Close() }()

	samples, rate, err := audio.Read(f)
	if err != nil {
		*notes = append(*notes, Note{
			About: at,
			Said:  fmt.Sprintf("is not audio this can read: %v", err),
		})

		return measured.Figures{}, "", false
	}

	// Through the same code every block went through, which is the only
	// reason the two are comparable at all.
	read := audio.Measure(samples, rate)

	return measured.Figures{
		Low:      100 * read.Low,
		Mid:      100 * read.Mid,
		High:     100 * read.High,
		Centroid: read.Centroid,
	}, at, true
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
	spec tone.Spec,
	deps Deps,
	notes *Notes,
) (measured.Figures, string, bool) {
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

		// Nothing said about a genre with no comparison population, because the
		// data cannot hold one: Usable needs eight records from three players
		// and a displacement needs two other players, out of sixteen. If it ever
		// happened the generic refusal is already accurate, and it says nothing
		// in the request can be measured against.
		shifted, ok := displacedTo(got, deps.Measured.Baseline)
		if !ok {
			continue
		}

		return shifted, named, true
	}

	return measured.Figures{}, "", false
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
) (measured.Figures, bool) {
	if got.Elsewhere == nil {
		return measured.Figures{}, false
	}

	at := got.Across.Measured()

	shift := func(key audio.Figure, scale float64) (float64, bool) {
		mine, held := at[string(key)]
		theirs, also := got.Elsewhere[key]

		if !held || !also {
			return 0, false
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

	return out, true
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
	spec tone.Spec,
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
	spec tone.Spec,
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
func lookup(
	cat *catalog.Catalog,
	want string,
	role rig.Role,
) (catalog.Block, []catalog.Block) {
	var exact, matched []catalog.Block

	target := strings.ToLower(want)

	for _, block := range cat.Blocks {
		if role != rig.RoleOther && string(block.Category) != string(role) {
			continue
		}

		name := strings.ToLower(block.Name)
		if name != target && !strings.Contains(name, target) {
			continue
		}

		// An exact name wins outright over a partial one, which is how "Ampeg
		// SVT" reaches the one called that rather than the one called "Ampeg
		// SVT Bright". Collected rather than returned on sight: a name is
		// exactly right for more than one model far more often than it looks,
		// and returning the first walked handed back whichever this map
		// happened to yield.
		if name == target {
			exact = append(exact, block)

			continue
		}

		matched = append(matched, block)
	}

	if len(exact) > 0 {
		matched = exact
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

// identify names the rig after the ask.
func identify(
	spec tone.Spec,
) string {
	parts := make([]string, 0, 3)

	// Not the genre, though every ask now carries one. A name is for a person to
	// recognise, and a field that is always present adds nothing to one: when
	// genre was optional it distinguished the asks that named it, and now it
	// would only prefix every identifier in the repository with a word.
	//
	if spec.Like != nil {
		for _, who := range []*string{spec.Like.Artist, spec.Like.Band, spec.Like.Song} {
			if who != nil && *who != "" {
				parts = append(parts, *who)
			}
		}
	}

	if len(parts) == 0 {
		parts = append(parts, "a sound")
	}

	// slug.Of rather than a rule of its own, and this used to have one. It
	// mapped a space to a hyphen, deleted everything else and never collapsed
	// the runs it left behind, so a band with an ampersand in it produced two
	// hyphens together: "Earth, Wind & Fire" became "earth-wind--fire". The
	// contract's pattern for an id is `^[a-z0-9]+(-[a-z0-9]+)*$`, which forbids
	// that, so Translate resolved the whole chain and then refused to write the
	// rig it had built. Any ampersand, comma or double space did it.
	return slug.Of(strings.Join(parts, "-"))
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
	held tone.Instrument,
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
