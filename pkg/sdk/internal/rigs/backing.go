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
package rigs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// manifestName is what a player's corpus directory calls its manifest.
const manifestName = "corpus.yaml"

// Backing reads which records back each rig, and holds them to its era.
//
// The two halves of a measured claim live in different files: a rig says
// which years its gear describes, and a manifest says when each measured
// record was made. Nothing joined them until this, and four of the nine rigs
// here turned out to be deriving words from records made on other gear. Flea's
// rig is his 2012 touring rig and his records are from 1989 to 1995; Mike
// Dirnt's is the American Idiot rig and his records are Dookie and Insomniac.
//
// Reported rather than refused. Which half is wrong is a judgement: the rig
// may describe the wrong period, or the records may be the wrong records, and
// only somebody who knows the player can say which.
func Backing(
	src Source,
	corpus string,
) ([]result.Backing, error) {
	all, err := read(src)
	if err != nil {
		return nil, err
	}

	out := []result.Backing(nil)
	claimed := map[string]bool{}

	for _, e := range all.merged() {
		spec := e.specOf()
		claimed[e.idOf()] = true
		one := result.Backing{ID: e.idOf()}

		// The era and the years are the ask's. Which records back a rig is a
		// question about the subject and the window somebody claimed for them,
		// and a rig with no ask claims no window: its records are listed and
		// none of them can be outside a range nobody stated.
		if e.askOf() != nil && e.askOf().Subject != nil {
			if e.askOf().Subject.Era != nil {
				one.Era = *e.askOf().Subject.Era
			}

			if e.askOf().Subject.Years != nil {
				one.From, one.To = e.askOf().Subject.Years.From, e.askOf().Subject.Years.To
			}
		}

		one.Direct, one.Both, one.Captured, one.Stage = rooms(spec.Chain)

		records, err := backing(corpus, e)
		if err != nil {
			return nil, err
		}

		if e.askOf() != nil {
			one.Misnamed = misnamed(e.askOf().Played, records)
		}

		for _, r := range records {
			one.Records = append(one.Records, result.Record{
				Track:   r.Track,
				Year:    r.Year,
				Outside: one.Stated() && (r.Year < one.From || r.Year > one.To),
			})
		}

		out = append(out, one)
	}

	orphans, err := unclaimed(corpus, claimed)
	if err != nil {
		return nil, err
	}

	out = append(out, orphans...)

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// unclaimed finds records sitting in a directory no rig is named for.
//
// A rig reaches its records by the directory carrying its identifier, and
// nothing else joins the two. A directory called anything else is measured by
// nobody, and it reads exactly like a rig nobody has measured yet, which is
// how a typo survives. Records fetched before the rig that will use them look
// the same and are fine, so this reports rather than refuses.
func unclaimed(
	corpus string,
	claimed map[string]bool,
) ([]result.Backing, error) {
	entries, err := os.ReadDir(corpus)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", corpus, err)
	}

	out := []result.Backing(nil)

	for _, e := range entries {
		if !e.IsDir() || claimed[e.Name()] {
			continue
		}

		records, err := recordsFor(corpus, e.Name())
		if err != nil {
			return nil, err
		}

		if len(records) == 0 {
			continue
		}

		one := result.Backing{ID: e.Name(), NoRig: true}
		for _, r := range records {
			one.Records = append(one.Records, result.Record{Track: r.Track, Year: r.Year})
		}

		out = append(out, one)
	}

	return out, nil
}

// recordsFor reads one player's manifest, or none where nobody has measured
// them.
//
// A rig with no corpus is the ordinary case rather than a fault: most players
// have gear evidence long before anybody owns their records.
// backing is the records behind one rig, which is a different join per subject.
//
// A rig for a person reaches its records through the directory carrying its
// identifier. A rig for a genre has no directory and never will: its records
// are other people's, sitting under the players who made them, and what joins
// them to it is the genre each one is tagged with.
//
// Without this the four genre rigs read "nothing measured for it", which is the
// opposite of true. Measurement is the only evidence a genre rig has.
func backing(
	corpus string,
	e stored,
) ([]audio.Record, error) {
	ask := e.askOf()
	if ask == nil || ask.Subject == nil || ask.Subject.Kind != tone.KindGenre {
		return recordsFor(corpus, e.idOf())
	}

	return genreRecords(corpus, ask.Genre)
}

// genreRecords is every record in the corpus carrying one of the genres given.
//
// Every player's manifest rather than one, because a genre is what its records
// have in common and they come from different people. Matched on the slug, so
// "pop-punk" and "Pop-Punk" are one genre, which is what the tag counts do.
//
// A record tagged with two of the genres asked for is counted once. The order
// is the walk's, which is the players in directory order.
func genreRecords(
	corpus string,
	of []string,
) ([]audio.Record, error) {
	// No guard on an empty list. `genre` is required on an ask and carries
	// minItems: 1, so a document that loads always names one, and a second check
	// here is a branch nothing can reach: a fixture written to cover it is
	// refused at load with "ask.genre property \"genre\" is missing".
	want := map[string]bool{}
	for _, g := range of {
		want[slug.Of(g)] = true
	}

	// A corpus that is not there is no records rather than an error, which is
	// what the directory join answers for a player nobody has measured. A
	// command run outside a checkout reaches this.
	held, err := audio.Manifests(os.DirFS(corpus), ".")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	out := []audio.Record(nil)

	for _, h := range held {
		for _, r := range h.Tracks {
			if carries(r, want) {
				out = append(out, r)
			}
		}
	}

	return out, nil
}

// carries says whether a record is tagged with any of the genres wanted.
func carries(
	r audio.Record,
	want map[string]bool,
) bool {
	for _, g := range r.Genres {
		if want[slug.Of(g)] {
			return true
		}
	}

	return false
}

func recordsFor(
	corpus, id string,
) ([]audio.Record, error) {
	at := filepath.Join(corpus, id, manifestName)

	f, err := os.Open(at) //nolint:gosec // a path built from the corpus given
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", at, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	m, err := audio.ReadManifest(f)
	if err != nil {
		return nil, err
	}

	return m.Tracks, nil
}

// rooms counts the chain entries that never met a microphone, and those whose
// only evidence is a tour.
//
// Both are the same mistake wearing different clothes: a figure measured off a
// record attributed to gear that was not in the room. The era check catches
// the wrong decade; this catches the right decade and the wrong room.
func rooms(
	chain []rig.ChainEntry,
) (direct, both, captured, stage int) {
	for _, e := range chain {
		if e.Capture != nil {
			captured++

			switch *e.Capture {
			case rig.CaptureDirect:
				direct++
			case rig.CaptureBoth:
				both++
			case rig.CaptureMiked:
			}
		}

		if e.Stage != nil && *e.Stage {
			stage++
		}
	}

	return direct, both, captured, stage
}

// misnamed finds track names an instrument claims that no manifest carries.
//
// `played[].records` joins an instrument to the records it made by their track
// names, which is the same join the corpus directory uses and fails the same
// way: a name matching nothing is silently attributed to nothing, and reads
// like an instrument nobody has got to yet.
func misnamed(
	played *[]tone.Played,
	have []audio.Record,
) []string {
	known := map[string]bool{}
	for _, r := range have {
		known[strings.ToLower(r.Track)] = true
	}

	var out []string

	if played == nil {
		return nil
	}

	for _, p := range *played {
		if p.Records == nil {
			continue
		}

		for _, name := range *p.Records {
			if !known[strings.ToLower(name)] {
				out = append(out, name)
			}
		}
	}

	return out
}
