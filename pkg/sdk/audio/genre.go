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
	"io/fs"
	"path"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
)

// A genre is where its records sit, against the records that are not it.
//
// The middle of a cluster is the least distinctive thing in it. Average every
// punk bassline and the result sounds like nothing, because what makes punk
// recognisable is what separates it from everything else rather than what its
// members share with recorded bass in general.
//
// What it stands against is the players who have none of it, because a player is
// the unit the comparison already uses and a genre is a claim about several of
// them. How the comparison is made differs from a player's, and displaced says
// why: a pooled spread is too wide for the rule Derive applies.

// Genre is one genre measured against the players who do not play it.
type Genre struct {
	// Name is the word as the manifests spell it, and Slug what grouping was
	// done on.
	Name string `json:"name"`
	Slug string `json:"slug"`
	// Records is how many recordings carry it, and Players how many different
	// players those come from.
	Records int `json:"records"`
	Players int `json:"players"`
	// Against is how many players had none of it, which is what the comparison
	// stood on. Fewer than two and nothing can be earned.
	Against int `json:"against"`
	// Across is what its records measure as together.
	Across Across `json:"across"`
	// Elsewhere is what the players holding none of it measure as, per figure.
	//
	// Here so that a figure can be read as a displacement rather than as a
	// position, which is the only way a genre and a device block can be
	// compared. A genre is measured off finished records and a block off a dry
	// signal pushed through it: punk reads 97.1% of its energy low, and the dry
	// signal going into the pedal holds 90.8% before any block touches it, so
	// asking which block reaches 97.1% asks for bottom that is not in the
	// input. Every candidate is then out of range and the nearest is whichever
	// is darkest, which is how a request for punk chose an Ampeg B-15NF.
	//
	// Against the records elsewhere, both sides become "how far from its own
	// normal", and those subtract. The Terms below are already earned this way;
	// this is the same comparison kept for every figure rather than only the
	// ones that earned a word.
	// A median of their medians rather than a spread, because what this answers
	// is "where does everybody else sit", and a width around that would be a
	// tolerance on a comparison rather than on a target.
	Elsewhere map[Figure]float64 `json:"elsewhere,omitempty"`
	// Terms are the words it earns against the players who do not play it.
	// Empty is an ordinary answer and means the figures are mixed rather than
	// that something went wrong.
	Terms []Derived `json:"terms"`
	// Usable says whether enough backs it to be computed from rather than
	// reported: eight records from at least three players.
	Usable bool `json:"usable"`
	// Instrument is what the records these figures came from were played on.
	//
	// Recorded because the figures are meaningless without it and carry no
	// hint of it otherwise. A bass guitar's centre of gravity sits an octave
	// below a guitar's, so this corpus puts every genre's centroid between 90
	// and 182Hz, and a guitar chain aimed at one of those is not close to it:
	// it is being asked to sound like a different instrument.
	//
	// Nothing in the figures refuses that on its own. The solve converges,
	// reports its tolerances met, and is wrong about what it was asked, which
	// is the failure this project has to expect.
	//
	// Empty when the records disagree, which means the tree held more than one
	// instrument and the genre pooled them. Nothing may aim at that: see
	// Usable, which it also clears.
	Instrument string `json:"instrument,omitempty"`
}

// Genres measures every genre the manifests name, against the rest.
//
// Pointed at one instrument's tree, like Corpus and for the same reason: a
// guitar's centre of gravity sits an octave above a bass guitar's, so pooling
// both would earn every bass genre the same word and mean nothing by it.
//
// A genre under the threshold is measured and reported rather than skipped.
// Knowing how far three punk records sit from the rest is worth seeing; what
// the threshold decides is whether anything may aim at it.
func GenresMeasured(
	fsys fs.FS,
	root string,
) ([]Genre, error) {
	held, err := Manifests(fsys, root)
	if err != nil {
		return nil, err
	}

	byPlayer, tagged, err := profilesByGenre(fsys, root, held)
	if err != nil {
		return nil, err
	}

	out := make([]Genre, 0, len(tagged))

	for key, in := range tagged {
		// The players with none of it. A player holding one punk record is not
		// part of what punk is measured against, because then the genre would
		// be standing partly against itself.
		others := map[string]Across{}

		for id, a := range byPlayer {
			if !in.players[id] {
				others[id] = a
			}
		}

		across := Together(in.profiles)

		// One instrument or none. A genre pooled across two is not a genre
		// these figures describe, so it is reported and may not be aimed at.
		enough := len(in.profiles) >= genreRecords &&
			len(in.players) >= genrePlayers

		// Where everybody else sits, which is what makes a figure here a
		// displacement rather than a position.
		rest := elsewhere(others)

		out = append(out, Genre{
			Name: in.name, Slug: key,
			Records: len(in.profiles), Players: len(in.players),
			Against:    len(others),
			Across:     across,
			Elsewhere:  rest,
			Terms:      displaced(across, others),
			Instrument: in.instrument,
			// Usable carries the comparison too, so one check guards it and
			// nothing downstream has to repeat it. A genre nobody can be held
			// against cannot be aimed at: its figures are measured off finished
			// records and a block's off a dry signal, and without somewhere else
			// to subtract there is no displacement to apply.
			Usable: enough && in.instrument != "" && rest != nil,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })

	return out, nil
}

// The threshold, the same two numbers Grouped.Usable states. Eight records from
// at least three players: three records by one band is that band's sound
// wearing a genre's name, and the figures cannot tell those apart.
const (
	genreRecords = 8
	genrePlayers = 3
)

// carried is the records one genre holds, and who made them.
type carried struct {
	name     string
	profiles []Profile
	players  map[string]bool
	// instrument is what every record of it was played on, and empty when they
	// were not all played on the same one. Empty rather than a list because
	// there is nothing to do with a mixed genre but refuse to aim at it.
	instrument string
	mixed      bool
}

// played records what one record was played on, and notices a genre that cannot
// name one instrument.
//
// The first instrument wins and anything disagreeing with it clears the answer. A
// genre drawn from a bass tree and a guitar tree has a centre of gravity that
// describes neither, and it would sit in the middle looking like an ordinary
// answer.
//
// **A record that names no instrument counts as a disagreement**, which is the
// case that is easy to get wrong. A player sitting directly under the corpus root
// rather than under an instrument's tree has no instrument: `instrumentDir`
// answers empty for it. Treated as "nothing yet", that player is absorbed into
// whichever instrument the next record names, and a genre half made of records
// nobody classified reads as pure bass. Not knowing is not agreeing.
func (c *carried) played(
	instrument string,
) {
	if c.mixed {
		return
	}

	if instrument == "" {
		c.instrument, c.mixed = "", true

		return
	}

	if c.instrument == "" {
		c.instrument = instrument

		return
	}

	if c.instrument != instrument {
		c.instrument, c.mixed = "", true
	}
}

// profilesByGenre measures every player and files each record under the genres
// it names.
//
// Two answers from one walk: what each player measures as, which is what a
// genre stands against, and which profiles carry each genre, which is what it
// is made of. Measuring is the slow half, so doing it once matters.
func profilesByGenre(
	fsys fs.FS,
	root string,
	held []Held,
) (map[string]Across, map[string]*carried, error) {
	byPlayer := map[string]Across{}
	tagged := map[string]*carried{}

	for _, p := range held {
		got, err := MeasureAll(fsys, path.Join(root, p.Instrument, p.ID))
		if err != nil {
			return nil, nil, err
		}

		// The manifest is the statement of what was measured, so a record it
		// does not name is not part of this even if its stems are on disk.
		got = Manifest{Artist: p.Artist, Tracks: p.Tracks}.Join(got)

		measured := make([]Profile, 0, len(got))

		for _, n := range got {
			if n.Source.Track == "" {
				continue
			}

			measured = append(measured, n.Profile)

			for _, g := range n.Source.Genres {
				key := slug.Of(g)
				if tagged[key] == nil {
					tagged[key] = &carried{name: g, players: map[string]bool{}}
				}

				tagged[key].profiles = append(tagged[key].profiles, n.Profile)
				tagged[key].players[p.ID] = true
				tagged[key].played(p.Instrument)
			}
		}

		if len(measured) == 0 {
			continue
		}

		byPlayer[p.ID] = Together(measured)
	}

	return byPlayer, tagged, nil
}

// displaced is where a pooled group sits against the others, by its middle.
//
// Not Derive, and the difference is the whole reason this exists. Derive earns a
// term only where a player's entire spread sits outside the middle half of the
// others, which is right for one player's habit over three or four records. A
// genre pools several players, so its spread runs two to three times wider than
// any of theirs: measured on this corpus, 64 to 92Hz of centroid against 17 to
// 41Hz for a player. A spread that wide cannot sit clear of anything, so the
// whole-spread rule earns a genre nothing whatever the figures say.
//
// So the middle against the middle half. Punk earns a word where the median of
// punk records sits outside the quartiles of the players who play none, which is
// the same displacement claim without requiring every punk record to be extreme.
//
// The margin travels with it, because two words that read alike are not alike: a
// median just past the quartile is a word the next player measured could take
// away, and one far past it is not.
// elsewhere is where the players holding none of a genre sit, per figure.
//
// The median of their medians, which is the same number [displaced] compares a
// genre against to earn a word. Kept for every figure rather than only the ones
// that earned one, so a genre can be read as a displacement on any axis.
//
// Nothing where fewer than two players are left. One player is not a population
// and a comparison against it says more about them than about the genre, which
// is the rule displaced already applies.
func elsewhere(
	others map[string]Across,
) map[Figure]float64 {
	if len(others) < 2 {
		return nil
	}

	out := map[Figure]float64{}

	for _, key := range MeasuredKeys() {
		rest := middles(others, key)
		if len(rest) < 2 {
			continue
		}

		sort.Float64s(rest)

		// Rounded to the same places the figures either side of the comparison
		// are. The medians going in are rounded and interpolating between two of
		// them is not, so a share held to two places could be subtracted from one
		// held to seventeen and leave a shift where there is no difference.
		out[key] = to(between(rest, 0.5), places(key))
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func displaced(
	mine Across,
	others map[string]Across,
) []Derived {
	if mine.Tracks == 0 || len(others) < 2 {
		return nil
	}

	var out []Derived

	for _, ax := range Axes {
		rest := middles(others, ax.Key)
		if len(rest) < 2 {
			continue
		}

		sort.Float64s(rest)

		at := mine.Measured()[string(ax.Key)]
		lower, upper := between(rest, 0.25), between(rest, 0.75)

		switch {
		case at > upper:
			out = append(out, made(
				ax, ax.More, mine, rest, len(others)+1, at-upper, lower, upper))
		case at < lower:
			out = append(out, made(
				ax, ax.Less, mine, rest, len(others)+1, lower-at, lower, upper))
		}
	}

	return out
}
