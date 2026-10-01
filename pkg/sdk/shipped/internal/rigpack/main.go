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

// rigpack copies the marketplace's core rigs into this package so the binary
// carries them.
//
// It exists because of one Go rule: `go:embed` cannot name a path above its own
// package directory. The rigs people read and submit belong at the top of the
// repository, in marketplace/, and the embed has to live beside the package that
// serves them, so one of the two has to be a copy.
//
// The copy is here and it is committed, which is the same arrangement the
// catalog and the corpus already have: a generated artifact in the tree, so a
// checkout builds without running anything, and a gate that fails when the
// source and the copy disagree.
//
// marketplace/core/ is the source. Edit a rig there, run `just generate`, and
// commit both.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: rigpack <marketplace/core dir> <out dir>")
		os.Exit(2)
	}

	if err := pack(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// pack mirrors the rigs under from into to, and says what it changed.
//
// A mirror rather than an append: a rig deleted from the marketplace has to
// leave the binary too, and copying without removing would keep serving one
// somebody retracted.
func pack(
	from, to string,
) error {
	if _, err := os.Stat(from); errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("rigs: %s is not there, nothing to pack\n", from)

		return nil
	}

	want, err := rigsUnder(from)
	if err != nil {
		return err
	}

	if len(want) == 0 {
		return fmt.Errorf("no rigs under %s, which cannot be right", from)
	}

	have, err := rigsUnder(to)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(to, 0o750); err != nil {
		return fmt.Errorf("making %s: %w", to, err)
	}

	changed := 0

	for name, body := range want {
		at := filepath.Join(to, name)

		if old, ok := have[name]; ok && string(old) == string(body) {
			continue
		}

		if err := os.WriteFile(at, body, 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", at, err)
		}

		changed++
	}

	gone := 0

	for name := range have {
		if _, ok := want[name]; ok {
			continue
		}

		if err := os.Remove(filepath.Join(to, name)); err != nil {
			return fmt.Errorf("removing %s: %w", filepath.Join(to, name), err)
		}

		gone++
	}

	switch {
	case changed == 0 && gone == 0:
		fmt.Printf("rigs: unchanged, %d from %s\n", len(want), from)
	default:
		fmt.Printf("rigs: %d written, %d removed, %d total from %s\n",
			changed, gone, len(want), from)
	}

	return nil
}

// rigsUnder is every YAML file directly in a directory, by name.
//
// Flat rather than a walk, because a rig and the ask beside it are the only two
// kinds of file here and a nested directory would be somebody inventing a layout
// the loader does not read.
func rigsUnder(
	dir string,
) (map[string][]byte, error) {
	out := map[string][]byte{}

	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		names = append(names, entry.Name())
	}

	// Sorted so the output is the same on two machines, which is what makes a
	// committed artifact diffable.
	sort.Strings(names)

	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // a path this read
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}

		out[name] = body
	}

	return out, nil
}
