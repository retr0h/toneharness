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
package audio

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
)

// Player is one artist in a music corpus: what their records measure as, and
// what that says about them against everybody else.
type Player struct {
	// ID is the directory the records sit in, which is the rig's identifier.
	ID string `json:"id"`
	// Records is how many of their recordings were measured.
	Records int `json:"records"`
	// Across is what those records measure as together.
	Across Across `json:"across"`
	// Terms are the words their figures earn against the other players.
	// Empty is the ordinary answer with few players, and it means the
	// evidence is mixed rather than that something went wrong.
	Terms []Derived `json:"terms"`
	// Within is the same question asked inside each genre they play, which is
	// a different question and usually a more useful one.
	//
	// Terms above compares a punk bassist against a jazz bassist, so it answers
	// "is this player dark" and the population it answers against is whoever
	// else happens to have been measured. Adding six southern rock players
	// moved every verdict in the corpus without a single record changing.
	//
	// This answers "is this player dark for punk", where the others are the
	// players who play it too. Nothing outside the genre can move it, so a word
	// earned here only comes up for review when that genre gains a player, which
	// is the moment it should.
	//
	// Both are kept because they are not the same claim. One finds a dark
	// bassist, the other builds a dark punk record.
	Within []InGenre `json:"within,omitempty"`
}

// InGenre is what a player's figures say against the others who play the same
// genre.
type InGenre struct {
	// Genre is the slug, as the manifests tag it.
	Genre string `json:"genre"`
	// Of is how many players carry it, this one included.
	Of int `json:"of"`
	// Terms are the words earned against those players and nobody else.
	Terms []Derived `json:"terms"`
}

// peersNeeded is how many players a genre needs before anybody is measured
// inside it, this one included.
//
// Three, which is the count a genre already needs before anything may aim at
// it: under that, a word says a player differs from one or two other people and
// reads as though it said something about the genre. The record threshold is not
// repeated here because this is a comparison between players rather than a
// distribution over records.
const peersNeeded = 3

// Corpus measures every player under a tree and derives what each one's
// figures say against the others.
//
// One directory per player, holding their records somewhere beneath it:
//
//	resources/music/bass/mike-dirnt/stems/htdemucs/longview/bass.wav
//	resources/music/bass/flea/stems/htdemucs/aeroplane/bass.wav
//
// Deriving needs the comparison, which is why this exists at all: a term is
// earned by sitting clear of the other players, so one player's records can
// be measured alone but can never earn a word.
//
// A directory holding no recordings is not a player, and is skipped rather
// than reported as one measuring nothing.
func Corpus(
	fsys fs.FS,
	root string,
) ([]Player, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}

	out := []Player(nil)
	together := map[string]Across{}
	plays := map[string][]string{}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		got, err := MeasureAll(fsys, path.Join(root, e.Name()))
		if err != nil {
			return nil, err
		}

		got, tags, err := named(fsys, path.Join(root, e.Name()), got)
		if err != nil {
			return nil, err
		}

		if len(got) == 0 {
			continue
		}

		a := Together(profilesOf(got))
		together[e.Name()] = a
		plays[e.Name()] = tags

		out = append(out, Player{ID: e.Name(), Records: len(got), Across: a})
	}

	for i := range out {
		out[i].Terms = Derive(out[i].Across, without(together, out[i].ID))
		out[i].Within = within(out[i].ID, together, plays)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// named keeps the recordings the manifest names, where there is one.
//
// The manifest is the statement of what was measured and the disk is
// incidental: a record dropped from a manifest is a record somebody decided
// not to measure, and its stems are still sitting there because separating
// takes minutes and nothing here deletes audio. Reading the tree alone would
// measure it anyway.
//
// That is not hypothetical. Six records were taken out of four manifests for
// being from the wrong era, their stems stayed on disk as the workflow says
// they should, and without this every one of them would still be in the
// figures.
//
// A player with no manifest is measured as found, which is what a directory
// somebody is still assembling looks like.
func named(
	fsys fs.FS,
	dir string,
	got []Named,
) ([]Named, []string, error) {
	f, err := fsys.Open(path.Join(dir, "corpus.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return got, nil, nil
	}

	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	m, err := ReadManifest(f)
	if err != nil {
		return nil, nil, err
	}

	wanted := make(map[string]bool, len(m.Tracks))
	tags := map[string]bool{}

	for _, rec := range m.Tracks {
		wanted[strings.ToLower(rec.Track)] = true

		// The player's genres are their records', because that is where a genre
		// is stated. One record of a kind makes them a player of it: the manifest
		// says what they played, not how much of it.
		for _, g := range rec.Genres {
			tags[slug.Of(g)] = true
		}
	}

	out := make([]Named, 0, len(got))

	for _, n := range got {
		if wanted[strings.ToLower(n.Name)] {
			out = append(out, n)
		}
	}

	return out, sorted(tags), nil
}

// sorted is the keys of a set, in order, so a run answers the same way twice.
func sorted(
	of map[string]bool,
) []string {
	out := make([]string, 0, len(of))
	for k := range of {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

// within is what a player earns inside each genre they play.
//
// A genre with too few players is skipped rather than answered thinly: see
// peersNeeded. A player whose genres are all short earns nothing here, which is
// the ordinary answer for most of this corpus and is not a failure.
func within(
	id string,
	together map[string]Across,
	plays map[string][]string,
) []InGenre {
	out := []InGenre(nil)

	for _, g := range plays[id] {
		peers := map[string]Across{}

		for other, tags := range plays {
			if other != id && slices.Contains(tags, g) {
				peers[other] = together[other]
			}
		}

		if len(peers)+1 < peersNeeded {
			continue
		}

		if terms := Derive(together[id], peers); len(terms) > 0 {
			out = append(out, InGenre{Genre: g, Of: len(peers) + 1, Terms: terms})
		}
	}

	return out
}

// profilesOf is what each of a player's records measured as.
func profilesOf(
	of []Named,
) []Profile {
	out := make([]Profile, 0, len(of))
	for _, n := range of {
		out = append(out, n.Profile)
	}

	return out
}

// without is everybody else, which is what a player is compared against.
func without(
	all map[string]Across,
	id string,
) map[string]Across {
	out := make(map[string]Across, len(all))

	for name, a := range all {
		if name != id {
			out[name] = a
		}
	}

	return out
}
