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

// Command handspack measures what one right hand does that another does not,
// and writes it where the sdk embeds it.
//
// Its own binary beside genrepack for the same reason that one is: the audio it
// reads is somebody else's and is not in the repository, so this cannot be a
// test. What is committed is the figure, and the test reads that.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// pairs are the hands this measures, and the directories each one's notes sit
// in, relative to the tree named on the command line.
//
// A list, so a second comparison is a line rather than a rewrite. Only one is
// measured today because only one dataset holds the same note played two ways,
// and the names are the plucking styles the ToneSpec contract spells.
var pairs = []struct {
	from, fromDir string
	to, toDir     string
}{
	{from: "fingers", fromDir: "fingerstyle", to: "pick", toDir: "picked"},
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: handspack <dry dir> <out.json>")
		os.Exit(2)
	}

	if err := pack(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// pack measures every pair of hands and writes what it found.
//
// Skips and says so when the notes are not there, and writes only when what it
// measured differs from what is committed. Both for the reason genrepack beside
// it does: `just generate` runs on machines holding no audio, and a generator
// that rewrites an identical file puts a diff in front of somebody every run.
func pack(
	from, to string,
) error {
	if _, err := os.Stat(from); errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("  hands: skipped, no %s\n", from)

		return nil
	}

	tree := os.DirFS(from)

	out := make([]audio.Hands, 0, len(pairs))

	for _, want := range pairs {
		got, err := measure(tree, want.from, want.fromDir, want.to, want.toDir)
		if err != nil {
			return err
		}

		if got.Pairs == 0 {
			continue
		}

		out = append(out, got)
	}

	if len(out) == 0 {
		fmt.Println("  hands: skipped, no paired notes to measure")

		return nil
	}

	return write(out, to)
}

// measure runs one comparison, skipping a pair whose directories are absent.
//
// Absent is not a failure. Somebody holding one dataset and not another should
// get the comparisons they can make rather than an error about the ones they
// cannot.
func measure(
	tree fs.FS,
	from, fromDir, to, toDir string,
) (audio.Hands, error) {
	for _, dir := range []string{fromDir, toDir} {
		if _, err := fs.Stat(tree, dir); err != nil {
			fmt.Printf("  hands: skipped %s against %s, no %s\n", to, from, dir)

			return audio.Hands{}, nil
		}
	}

	got, err := audio.HandsMeasured(tree, from, fromDir, to, toDir)
	if err != nil {
		return audio.Hands{}, fmt.Errorf("measuring %s against %s: %w", to, from, err)
	}

	return got, nil
}

// write commits the measurements, and says what it did.
func write(
	all []audio.Hands,
	to string,
) error {
	body, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return fmt.Errorf("writing the hands: %w", err)
	}

	body = append(body, '\n')

	same, err := matches(to, body)
	if err != nil {
		return err
	}

	for _, got := range all {
		fmt.Printf("  hands: %s against %s, %d pairs, %d agreeing\n",
			got.To, got.From, got.Pairs, got.Agreed)
	}

	if same {
		fmt.Println("  hands: unchanged")

		return nil
	}

	if err := os.WriteFile(to, body, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", to, err)
	}

	fmt.Println("  hands: written")

	return nil
}

// matches says whether the committed file already holds this.
func matches(
	at string,
	body []byte,
) (bool, error) {
	held, err := os.ReadFile(at)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("reading %s: %w", at, err)
	}

	return string(held) == string(body), nil
}
