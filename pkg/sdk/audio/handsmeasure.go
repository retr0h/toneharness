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
	"math"
	"path"
	"regexp"
	"sort"
	"strings"
)

// noteNamed reads what a file in IDMT-SMT-Bass is: which bass, which pickup
// setting, which hand, which expression, and then the note.
//
// A regexp rather than a split, because the fields are positional and a name
// that does not match this shape is a file from somewhere else. One that does
// not match is skipped rather than guessed at: a wrong pairing would compare
// two different notes and report the difference as a hand.
var noteNamed = regexp.MustCompile(`^BS_(\d+)_EQ_(\d+)_([A-Z]+)_([A-Z]+)_(\d+)_(\d+)\.wav$`)

// HandsMeasured holds one hand's notes against another's, note for note.
//
// Each directory is one plucking style, named by the key the contract uses for
// it. What comes back is the difference To reads minus From reads, per pair,
// averaged, with the spread and how many pairs agreed on the direction.
//
// Pairing is on everything but the hand: the same bass, the same pickup
// setting, the same expression and the same note. A note either side has no
// partner for is dropped rather than averaged in, because the whole point of
// pairing is that nothing but the hand differs.
func HandsMeasured(
	fsys fs.FS,
	from, fromDir, to, toDir string,
) (Hands, error) {
	first, err := notesIn(fsys, fromDir)
	if err != nil {
		return Hands{}, err
	}

	second, err := notesIn(fsys, toDir)
	if err != nil {
		return Hands{}, err
	}

	keys := make([]string, 0, len(first))
	for key := range first {
		if _, both := second[key]; both {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return Hands{}, nil
	}

	// Sorted, so the answer does not depend on map order. Nothing downstream
	// reads the order, and a generator writing a different file each run puts a
	// diff in front of somebody for nothing.
	sort.Strings(keys)

	return handsFrom(from, to, keys, first, second), nil
}

// handsFrom is the arithmetic, once the pairs are known.
func handsFrom(
	from, to string,
	keys []string,
	first, second map[string]Profile,
) Hands {
	out := Hands{
		From:    from,
		To:      to,
		Pairs:   len(keys),
		Figures: map[Figure]Moved{},
	}

	deltas := map[Figure][]float64{}

	// Profile.Measured rather than a second mapping, so a difference is taken
	// between the same figures a rig's evidence carries, rounded the same way.
	// Two mappings would drift and the drift would look like a measurement.
	for _, key := range keys {
		mine, theirs := second[key].Measured(), first[key].Measured()

		for _, which := range MeasuredKeys() {
			a, held := mine[string(which)]
			b, also := theirs[string(which)]

			if !held || !also {
				continue
			}

			deltas[which] = append(deltas[which], a-b)
		}
	}

	for which, all := range deltas {
		out.Figures[which] = movedBy(all, places(which))
	}

	// Agreement is counted on the centroid, which is the figure this exists to
	// answer: where the sound sits. A per-figure count would be four numbers
	// saying nearly the same thing, and the one that matters is whether the
	// hand moves the centre of gravity reliably.
	if centroid, held := out.Figures[KeyCentroid]; held {
		for _, d := range deltas[KeyCentroid] {
			if (d > 0) == (centroid.Mean > 0) {
				out.Agreed++
			}
		}
	}

	return out
}

// places is how many decimals a difference in this figure is written to.
//
// The same precision the figure itself is written to, because a difference
// between two readings cannot be finer than the readings. A share of the energy
// is measured to two places and a centroid to whole hertz, so a mean of 468 of
// them carries 17 significant figures and claims a precision nobody measured.
func places(
	which Figure,
) int {
	switch {
	case which.Share():
		return 2
	case which == KeyCentroid:
		return 0
	default:
		return 2
	}
}

// movedBy is the mean, the median and the spread of one figure's differences.
func movedBy(
	all []float64,
	places int,
) Moved {
	sorted := make([]float64, len(all))
	copy(sorted, all)
	sort.Float64s(sorted)

	var sum float64
	for _, d := range sorted {
		sum += d
	}

	mean := sum / float64(len(sorted))

	var squares float64
	for _, d := range sorted {
		squares += (d - mean) * (d - mean)
	}

	return Moved{
		Mean:   rounded(mean, places),
		Median: rounded(sorted[len(sorted)/2], places),
		Spread: rounded(math.Sqrt(squares/float64(len(sorted))), places),
	}
}

// rounded is to, without a negative zero.
//
// A tiny negative mean rounds to -0, which Go prints and JSON keeps, so the
// committed file said `"mean": -0` for the figure a pick does not move. It reads
// as a measurement with a direction and it is the absence of one.
func rounded(
	v float64,
	places int,
) float64 {
	if out := to(v, places); out != 0 {
		return out
	}

	return 0
}

// notesIn measures every note in one directory, keyed by everything but the
// hand, so two directories can be paired on the key.
func notesIn(
	fsys fs.FS,
	dir string,
) (map[string]Profile, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	out := make(map[string]Profile, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		named := noteNamed.FindStringSubmatch(e.Name())
		if named == nil {
			continue
		}

		read, err := measureOne(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}

		// Everything but the hand, which is group 3. Two notes sharing this
		// key differ in nothing else anybody recorded.
		out[strings.Join([]string{
			named[1], named[2], named[4], named[5], named[6],
		}, "_")] = read
	}

	return out, nil
}
