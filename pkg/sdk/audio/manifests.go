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
	"fmt"
	"io/fs"
	"path"
	"sort"
)

// manifestName is the file a player's directory holds.
const manifestName = "corpus.yaml"

// Held is one artist's manifest, with the directory it was found in.
//
// Beside Player rather than part of it, because the two are read at different
// costs. This is what a manifest says, which is a file read; a Player is what
// the recordings measure as, which needs the audio and minutes of work.
//
// The directory rather than the manifest's own `artist`, because that is what
// joins a corpus to a rig: a rig is measured by the directory named for it, and
// a directory spelled any other way is measured by nobody.
type Held struct {
	// Instrument is the directory above the player, where there is one. A
	// comparison is only meaningful within one of these: a word is earned by
	// sitting clear of the other players, and a guitar's centre of gravity sits
	// an octave above a bass guitar's.
	Instrument string
	// ID is the directory, which is the rig's identifier.
	ID string
	// Artist is the name the manifest gives, which is for reading.
	Artist string
	// Tracks is what the manifest names.
	Tracks []Record
}

// Manifests reads every manifest under a tree, wherever it sits in it.
//
// Found by walking rather than by knowing the shape, so one root answers for
// every instrument at once. `resources/music` holds `bass/mike-dirnt` and would
// hold `guitar/somebody`, and a listing that had to be told which instrument to
// look at could not say what the tree holds.
//
// A directory with no manifest is skipped rather than refused: a player whose
// records are still being chosen has a directory before it has a manifest.
//
// Refused loudly where a manifest is unreadable, because the alternative is a
// genre counted short and nothing saying why.
func Manifests(
	fsys fs.FS,
	dir string,
) ([]Held, error) {
	out := []Held(nil)

	err := fs.WalkDir(fsys, dir, func(at string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || d.Name() != manifestName {
			return nil
		}

		m, err := manifestAt(fsys, at)
		if err != nil {
			return err
		}

		// The player is the directory holding the manifest, and the instrument
		// the one above it where the tree goes that deep.
		player := path.Dir(at)

		out = append(out, Held{
			Instrument: instrumentOf(dir, player),
			ID:         path.Base(player),
			Artist:     m.Artist,
			Tracks:     m.Tracks,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Instrument != out[j].Instrument {
			return out[i].Instrument < out[j].Instrument
		}

		return out[i].ID < out[j].ID
	})

	return out, nil
}

// instrumentOf names the directory between the root and the player, or nothing
// where the root is the instrument's own tree.
func instrumentOf(
	root, player string,
) string {
	above := path.Dir(player)
	if above == root || above == "." {
		return ""
	}

	return path.Base(above)
}

// manifestAt reads one manifest.
func manifestAt(
	fsys fs.FS,
	at string,
) (Manifest, error) {
	f, err := fsys.Open(at)
	if err != nil {
		return Manifest{}, fmt.Errorf("reading %s: %w", at, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	m, err := ReadManifest(f)
	if err != nil {
		return Manifest{}, fmt.Errorf("reading %s: %w", at, err)
	}

	return m, nil
}

// Grouping turns what Manifests read into what Genres and Bands group over.
func Grouping(
	of []Held,
) []Manifest {
	out := make([]Manifest, 0, len(of))
	for _, p := range of {
		out = append(out, Manifest{Artist: p.Artist, Tracks: p.Tracks})
	}

	return out
}
