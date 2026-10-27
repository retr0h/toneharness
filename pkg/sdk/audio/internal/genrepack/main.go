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

// Command genrepack measures the music corpus and writes what the sdk embeds.
//
// The recordings cannot ship: they are somebody else's and a repository is not
// a way around that. What each genre measured as can, and it is the half
// anything downstream needs, so this is the step that gets it into the binary.
//
// The same split the preset corpus uses, where the fetch script and the
// attribution are committed and the payload is not.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: genrepack <corpus dir> <out.json>")
		os.Exit(2)
	}

	if err := pack(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// pack measures every genre and writes it where the sdk embeds it.
//
// Skips and says so when the corpus is not there, and writes only when what it
// measured differs from what is committed. Both for the reason every generator
// beside it does: `just generate` runs on machines that hold no audio, and a
// generator that rewrites an identical file puts a diff in front of somebody on
// every run.
func pack(
	from, to string,
) error {
	if _, err := os.Stat(from); errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("  genres: skipped, no %s\n", from)

		return nil
	}

	// `from` is the whole corpus, `resources/music`, so its base names no
	// instrument and every player's is read from the directory above it. Passed
	// anyway rather than left blank, because this generator would otherwise be
	// the one caller that silently stops working if somebody pointed it at one
	// instrument's tree.
	got, err := audio.GenresMeasured(os.DirFS(from), ".", filepath.Base(from))
	if err != nil {
		return fmt.Errorf("measuring the genres: %w", err)
	}

	if len(got) == 0 {
		fmt.Println("  genres: skipped, no records carry one")

		return nil
	}

	// And every player, measured the same way and over the same tree.
	//
	// Beside the genres rather than in their own pack: both are the same
	// recordings read two ways, so a pack holding one without the other could
	// describe a corpus that no longer exists on one half and not the other.
	// One instrument's tree at a time, because a player is a directory inside
	// one and `from` is the corpus above them all: pointed at the corpus it
	// reads `bass` as a player with 425 records. The genres generator above
	// takes the whole tree because a genre pools across it.
	tops, err := os.ReadDir(from)
	if err != nil {
		return fmt.Errorf("reading %s: %w", from, err)
	}

	who := []audio.Player(nil)

	for _, e := range tops {
		if !e.IsDir() {
			continue
		}

		// The directory is the instrument, and it is both the tree this reads
		// and the name to fall back on.
		some, err := audio.Corpus(os.DirFS(from), e.Name(), e.Name())
		if err != nil {
			return fmt.Errorf("measuring the players in %s: %w", e.Name(), err)
		}

		who = append(who, some...)
	}

	// Which corpus these figures were measured over, recorded beside them.
	// The audio cannot ship so no test can re-derive a figure, and without
	// this the pack could not even say which records it was taken across: a
	// pack and a corpus that had drifted apart agreed with each other and
	// nothing could tell.
	sha, err := audio.CorpusSHA(os.DirFS(from), ".")
	if err != nil {
		return fmt.Errorf("fingerprinting %s: %w", from, err)
	}

	body, err := json.MarshalIndent(
		audio.Packed{Corpus: sha, Genres: got, Players: who}, "", "  ")
	if err != nil {
		return fmt.Errorf("writing the genres: %w", err)
	}

	body = append(body, '\n')

	same, err := matches(to, body)
	if err != nil {
		return err
	}

	if same {
		fmt.Printf("  genres: unchanged, %d measured\n", len(got))

		return nil
	}

	if err := os.WriteFile(to, body, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", to, err)
	}

	fmt.Printf("  genres: written, %d measured\n", len(got))

	return nil
}

// matches says whether the committed file already holds this.
func matches(
	at string,
	body []byte,
) (bool, error) {
	was, err := os.ReadFile(at) //nolint:gosec // a path the operator gave
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("reading %s: %w", at, err)
	}

	return bytes.Equal(was, body), nil
}
