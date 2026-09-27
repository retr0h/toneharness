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

// Package musicview answers what the music corpus holds, from the manifests.
//
// The manifests rather than the audio. Every other question of the corpus
// measures recordings, which needs files nobody may redistribute and minutes of
// work each. What was written down is a file read, and it is what decides
// whether a genre can be aimed at yet.
package musicview

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// ErrNoCorpus reports a tree holding no manifests.
var ErrNoCorpus = errors.New("no manifests in the corpus")

// Players is every player the corpus names, with what their records say.
func Players(
	fsys fs.FS,
	dir string,
) ([]result.MusicPlayer, error) {
	held, err := read(fsys, dir)
	if err != nil {
		return nil, err
	}

	out := make([]result.MusicPlayer, 0, len(held))

	for _, p := range held {
		bands, genres, untagged := saysOf(p)

		out = append(out, result.MusicPlayer{
			Instrument: p.Instrument,
			ID:         p.ID, Artist: p.Artist, Records: len(p.Tracks),
			Bands: bands, Genres: genres, Untagged: untagged,
		})
	}

	return out, nil
}

// saysOf is every band and genre one player's records name, each once.
func saysOf(
	p audio.Held,
) (bands, genres []string, untagged int) {
	seenBand, seenGenre := map[string]bool{}, map[string]bool{}

	for _, rec := range p.Tracks {
		if rec.Band != "" && !seenBand[slug.Of(rec.Band)] {
			seenBand[slug.Of(rec.Band)] = true
			bands = append(bands, rec.Band)
		}

		if len(rec.Genres) == 0 {
			untagged++
		}

		for _, g := range rec.Genres {
			if !seenGenre[slug.Of(g)] {
				seenGenre[slug.Of(g)] = true
				genres = append(genres, g)
			}
		}
	}

	slices.Sort(bands)
	slices.Sort(genres)

	return bands, genres, untagged
}

// Genres is every genre the corpus names, and how far each is from usable.
func Genres(
	fsys fs.FS,
	dir string,
) ([]result.MusicGroup, error) {
	held, err := read(fsys, dir)
	if err != nil {
		return nil, err
	}

	return grouped(audio.Genres(audio.Grouping(held))), nil
}

// Bands is every band the corpus names.
func Bands(
	fsys fs.FS,
	dir string,
) ([]result.MusicGroup, error) {
	held, err := read(fsys, dir)
	if err != nil {
		return nil, err
	}

	return grouped(audio.Bands(audio.Grouping(held))), nil
}

// Records is every recording the corpus names, and whether it has stems yet.
func Records(
	fsys fs.FS,
	dir string,
) ([]result.MusicRecord, error) {
	held, err := read(fsys, dir)
	if err != nil {
		return nil, err
	}

	out := []result.MusicRecord(nil)

	for _, p := range held {
		for _, rec := range p.Tracks {
			out = append(out, result.MusicRecord{
				Instrument: p.Instrument,
				Player:     p.ID, Track: rec.Track, Year: rec.Year,
				Band: rec.Band, Genres: rec.Genres,
				DecidedBy: string(rec.GenresBy),
				Separated: separated(fsys, dir, p.Instrument, p.ID, rec.Track),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Instrument != out[j].Instrument {
			return out[i].Instrument < out[j].Instrument
		}

		if out[i].Player != out[j].Player {
			return out[i].Player < out[j].Player
		}

		return out[i].Track < out[j].Track
	})

	return out, nil
}

// separated says whether the bass has been split out of one record.
//
// By the directory the separator writes, which is the same join `measure`
// makes: a stem directory named for the track, holding the isolated part.
func separated(
	fsys fs.FS,
	dir, instrument, id, track string,
) bool {
	// htdemucs for bass, htdemucs_6s for guitar. Either counts, because the
	// question is whether anything separated it.
	for _, model := range []string{"htdemucs", "htdemucs_6s"} {
		at := path.Join(dir, instrument, id, "stems", model, track)
		if _, err := fs.Stat(fsys, at); err == nil {
			return true
		}
	}

	return false
}

// grouped turns what audio counted into what a caller reads, adding how far
// each one still is from the threshold.
func grouped(
	of []audio.Grouped,
) []result.MusicGroup {
	out := make([]result.MusicGroup, 0, len(of))

	for _, g := range of {
		out = append(out, result.MusicGroup{
			Name: g.Name, Slug: g.Slug,
			Records: g.Records, Artists: g.Artists, Who: g.Who,
			Unsighted:    g.Unsighted,
			Usable:       g.Usable(),
			ShortRecords: short(g.Records, wantRecords),
			ShortArtists: short(g.Artists, wantArtists),
		})
	}

	return out
}

// The threshold, as the two numbers a caller is short of rather than the one
// question audio.Grouped answers. "Two more records" is actionable and "not
// usable" is not.
const (
	wantRecords = 8
	wantArtists = 3
)

// short is how many more of something a group needs, never below zero.
func short(
	have, want int,
) int {
	if have >= want {
		return 0
	}

	return want - have
}

// read reads every manifest, refusing a tree that holds none.
//
// Refused rather than answered empty, because the ordinary cause is a path
// pointing at the wrong directory, and an empty table reads as a corpus that
// exists and holds nothing.
func read(
	fsys fs.FS,
	dir string,
) ([]audio.Held, error) {
	held, err := audio.Manifests(fsys, dir)
	if err != nil {
		return nil, err
	}

	if len(held) == 0 {
		return nil, fmt.Errorf("%w under %s: one directory per player, each "+
			"holding a corpus.yaml", ErrNoCorpus, dir)
	}

	return held, nil
}
