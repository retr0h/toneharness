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
	if err != nil {
		return rig.Spec{}, notes, err
	}

	out.Chain = chain

	// Said rather than silently dropped. A request carrying a genre or a
	// player this cannot resolve is a request half answered, and the half
	// that was not is the part somebody needs to know about.
	unresolved(spec, &notes)

	if err := rig.Validate(out); err != nil {
		return rig.Spec{}, notes, err
	}

	return out, notes, nil
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

	// An amplifier is the one block a chain cannot do without, so when
	// nothing named one and there is something to aim at, the measurements
	// pick the closest.
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
		found := len(matched) == 1

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
			*notes = append(*notes, Note{
				About:    want.Gear,
				Said:     fmt.Sprintf("resolved to %s", block.ID),
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
	want, from, ok := target(spec, notes)
	if !ok {
		return rig.ChainEntry{}, false
	}

	ranked := reachable(
		deps.Measured.Nearest(category, want, measured.Spectral()), setup)
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
func reachable(
	ranked []measured.Match,
	setup tone.Setup,
) []measured.Match {
	loaded := map[string]bool{}

	if setup.Owns != nil {
		for _, held := range *setup.Owns {
			if held.Kind == tone.OwnedIR {
				loaded[strings.ToLower(held.Name)] = true
			}
		}
	}

	out := make([]measured.Match, 0, len(ranked))

	for _, match := range ranked {
		if catalog.NeedsUserIR(catalog.ModelID(match.ID)) &&
			!loaded[strings.ToLower(match.Name)] {
			continue
		}

		out = append(out, match)
	}

	return out
}

// target is what a request is aiming at, in the figures a block is measured
// in, and what it came from.
func target(
	spec tone.Spec,
	notes *Notes,
) (measured.Figures, string, bool) {
	if spec.Like == nil || spec.Like.Recording == nil {
		return measured.Figures{}, "", false
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
) Note {
	switch {
	case !ok:
		return Note{
			About: want,
			Said: "no records carry that genre, so there is no distribution " +
				"to aim at. Tag some and measure them",
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

// unresolved says which parts of a request this cannot yet answer.
func unresolved(
	spec tone.Spec,
	notes *Notes,
) {
	// Every genre the ask names, because a record belongs to more than one and
	// the corpus tags the same players both punk and pop-punk. Each is reported
	// on its own: one may have enough records behind it to compute from while
	// another has three by one band, and collapsing them would hide which.
	for _, named := range spec.Genre {
		if named == "" {
			continue
		}

		got, ok := audio.ShippedGenre(slug.Of(named))
		*notes = append(*notes, genreNote(named, got, ok))
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

			*notes = append(*notes, Note{
				About: *named.who,
				Said: fmt.Sprintf(
					"naming %s an %s aims at their records, which this does "+
						"not read yet; hand it a recording instead",
					*named.who, named.what),
			})
		}
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
	var matched []catalog.Block

	target := strings.ToLower(want)

	for _, block := range cat.Blocks {
		if role != rig.RoleOther && string(block.Category) != string(role) {
			continue
		}

		name := strings.ToLower(block.Name)
		if name != target && !strings.Contains(name, target) {
			continue
		}

		// An exact name wins outright, which is how "Ampeg SVT" reaches the
		// one called that rather than the one called "Ampeg SVT Bright".
		if name == target {
			return block, []catalog.Block{block}
		}

		matched = append(matched, block)
	}

	if len(matched) == 1 {
		return matched[0], matched
	}

	// Sorted, so a request that matched several is told about them in the
	// same order every time.
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].ID < matched[j].ID
	})

	return catalog.Block{}, matched
}

// names lists what a gear name matched, for a message somebody can act on.
//
// By model identifier rather than by name, because the names are what
// collided: 665 models share 469 names, so a note listing four called "Ampeg
// SVT Nrm" tells nobody which to ask for. A model identifier is exact, and is
// what a request's `models` field takes.
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

	name := strings.Join(parts, "-")
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r == ' ':
			return '-'
		}

		return -1
	}, name)

	return strings.Trim(name, "-")
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
				return instrumentOf(held, notes)
			}
		}

		if len(*setup.Instruments) > 0 {
			return instrumentOf((*setup.Instruments)[0], notes)
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

// instrumentOf is which kind of instrument a setup entry describes.
func instrumentOf(
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
