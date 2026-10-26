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

package measured

import (
	"os"
	"path/filepath"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// InFrontOf reads the library of blocks measured with one named block behind
// them, from a directory of readings.
//
// Task 288. Every figure in the isolated library was taken with one block in
// the chain and nothing else, and a drive's whole job is how it hits the stage
// behind it. Measured both ways against one settled rig, every one of the
// eleven genres chose a different drive and eight traded a clean boost for a
// Darkglass B7K Ultra.
//
// Named `fingerprints-into-<model>.json` beside the isolated
// `fingerprints.json`, which is the convention the control sweeps already use
// for a chain reading. Kept as its own file rather than merged, because
// [BuiltIn] refuses a library saying `isolated: false` and that refusal is
// right: these figures describe the pair rather than the blocks in it.
//
// Here rather than beside either caller, because both the ask path and the
// preset build want the same lookup and a second copy of it is a second place
// for the filename convention to drift.
//
// Nil for an empty directory, which is what a release gets: these are a
// directory of readings rather than something embedded, the same as the
// control sweeps.
func InFrontOf(
	dir string,
) func(after catalog.ModelID) (Library, bool) {
	if dir == "" {
		return nil
	}

	return func(after catalog.ModelID) (Library, bool) {
		f, err := os.Open(filepath.Join( //nolint:gosec // a path built here
			dir, "fingerprints-into-"+string(after)+".json"))
		if err != nil {
			return Library{}, false
		}

		defer func() { _ = f.Close() }()

		got, err := Load(f)
		if err != nil {
			return Library{}, false
		}

		// Refused rather than used. A library claiming isolated readings is
		// what the caller's own `Measured` already is, and ranking the same
		// figures twice is not a chain reading. One that cannot say which
		// block sat behind it cannot be matched to a chain at all.
		if got.Isolated || got.After == "" {
			return Library{}, false
		}

		return got, true
	}
}
