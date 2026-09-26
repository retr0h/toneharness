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

	"github.com/retr0h/tonestack/pkg/sdk/audio"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/tone"
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
		Subject:    subjectFor(spec),
		Instrument: instrumentFor(setup, &notes),
	}

	// Words travel as words. A character term is how it should sound, which
	// is the same thing a ToneSpec's words are, and turning them into knob
	// positions here would be the guessing this project removed.
	if spec.Words != nil && len(*spec.Words) > 0 {
		terms := make([]rig.CharacterTerm, 0, len(*spec.Words))
		for _, word := range *spec.Words {
			terms = append(terms, rig.CharacterTerm{Term: word})
		}

		out.Character = &terms
	}

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
			models := map[string]string{deps.Measured.Device: string(block.ID)}
			entry.Models = &models

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
	models := map[string]string{deps.Measured.Device: best.ID}

	*notes = append(*notes, Note{
		About: string(category),
		Said: fmt.Sprintf(
			"%s is the closest of %d measured to %s, %.0f Hz against %.0f",
			best.Name, len(ranked), from, best.Centroid, want.Centroid),
		Honoured: true,
	})

	return rig.ChainEntry{
		Role:   rig.Role(category),
		Gear:   best.Name,
		Models: &models,
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

// unresolved says which parts of a request this cannot yet answer.
func unresolved(
	spec tone.Spec,
	notes *Notes,
) {
	if spec.Genre != nil && *spec.Genre != "" {
		*notes = append(*notes, Note{
			About: *spec.Genre,
			Said: "no records carry that genre yet, so there is no " +
				"distribution to aim at. See a genre is a corpus with a name",
		})
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

	if spec.Genre != nil && *spec.Genre != "" {
		parts = append(parts, *spec.Genre)
	}

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

// subjectFor is who or what the rig is attributed to.
func subjectFor(
	spec tone.Spec,
) rig.Subject {
	if spec.Like != nil {
		if spec.Like.Artist != nil && *spec.Like.Artist != "" {
			out := rig.Subject{Kind: rig.KindArtist, Name: *spec.Like.Artist}
			if spec.Like.Band != nil && *spec.Like.Band != "" {
				out.Band = spec.Like.Band
			}

			return out
		}

		if spec.Like.Band != nil && *spec.Like.Band != "" {
			return rig.Subject{Kind: rig.KindBand, Name: *spec.Like.Band}
		}

		if spec.Like.Song != nil && *spec.Like.Song != "" {
			return rig.Subject{Kind: rig.KindSong, Name: *spec.Like.Song}
		}
	}

	if spec.Genre != nil && *spec.Genre != "" {
		return rig.Subject{Kind: rig.KindGenre, Name: *spec.Genre}
	}

	return rig.Subject{Kind: rig.KindSound, Name: "a sound somebody asked for"}
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
