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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
)

// CorpusSHA fingerprints every manifest under a tree, so a packed measurement
// can say which corpus it was measured from.
//
// The audio cannot ship and the manifests do, which is what makes this
// checkable where nothing else about the measurements is. A figure in
// data/genres.json came off recordings nobody else here holds, so no test can
// re-derive it; the set of records it was taken over is committed, and a pack
// built from a different set is a pack nobody can account for.
//
// **It fingerprints the manifests and not the audio.** Two machines holding
// different downloads of the same track agree here and would measure
// differently, and nothing in a repository can tell them apart. What this
// catches is the manifest that moved while the pack stood still, which is the
// failure [measure-music]'s finishing.md calls the one that gets forgotten and
// which is invisible: the measuring commands print the new numbers while the
// embedded file still holds the old ones, so a build aims where the corpus used
// to be.
//
// The path goes into the hash as well as the content, because moving a player's
// directory re-files every record it holds and changes nothing inside the file.
// Sorted first, since fs.WalkDir's order is its own and a fingerprint that
// depends on it is not one.
func CorpusSHA(
	fsys fs.FS,
	root string,
) (string, error) {
	at := []string(nil)

	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && d.Name() == manifestName {
			at = append(at, p)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking %s: %w", root, err)
	}

	sort.Strings(at)

	sum := sha256.New()

	for _, p := range at {
		body, err := fs.ReadFile(fsys, p)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", p, err)
		}

		// Length-prefixed rather than separated. Two manifests whose contents
		// run together would otherwise hash the same as one holding both, and a
		// separator can appear in a file.
		_, _ = fmt.Fprintf(sum, "%d %s\n%d\n", len(p), p, len(body))
		_, _ = sum.Write(body)
	}

	return hex.EncodeToString(sum.Sum(nil)), nil
}

// corpusRelative is the manifest tree as this package's own tests reach it,
// which is where the shipped fingerprint is checked against the source.
//
// Here rather than in the test, so the one place that knows where the corpus
// sits relative to the package is beside the code that embeds it.
var corpusRelative = path.Join("..", "..", "..", "resources", "music")
