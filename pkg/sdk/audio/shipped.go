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
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

// What each genre measured as, for the binary to read without any audio.
//
// The recordings cannot ship and the figures can, which is the split the whole
// corpus uses. Measuring needs the audio and minutes per record; anything
// choosing a block from a genre needs only the result.
//
// Measured rather than written. Every number came from the records in
// resources/music, through GenresMeasured, against the players who play none of
// that genre. Re-measure by running `just generate` on a machine holding the
// audio; it skips where there is none rather than emptying the file.
//
//go:embed data/genres.json
var shipped []byte

// Shipped is what each genre measured as, parsed once.
//
// Empty where nothing has been measured, because a binary built on a machine
// with no audio is the ordinary case rather than a broken one: the generator
// writes an empty list rather than deleting the file. A caller asking for a
// genre nothing measured gets nothing and should say so.
var Shipped = sync.OnceValues(func() ([]Genre, error) {
	got, err := packed()

	return got.Genres, err
})

// packed is the embedded file parsed once, which both answers above it read.
//
// One parse rather than two. [ShippedCorpus] had its own, which gave it an
// error to handle that nothing could reach: the committed file always parses,
// and the embed directive refuses to compile without it, so the branch was dead
// and counted against the coverage gate. Sharing the parse removes it rather
// than testing a state that cannot happen.
var packed = sync.OnceValues(func() (Packed, error) {
	return unpackGenres(shipped)
})

// ShippedCorpus is the fingerprint of the manifests the shipped figures were
// measured from, or empty where the file predates one.
//
// What it is for: the measurements are committed because nobody else can take
// them again, and that leaves nothing able to say whether they describe the
// records this checkout holds. A manifest edited without regenerating is
// invisible otherwise, and the measuring commands keep printing the new numbers
// while the embedded file holds the old ones.
//
// Compared against [CorpusSHA] over the committed manifests by this package's
// own tests, which is a check that needs no audio and therefore runs where the
// audio never is.
func ShippedCorpus() string {
	// The error is dropped rather than branched on, for the reason [Shipped]
	// drops it: a Packed that failed to parse is the zero value and its Corpus
	// is empty, which is the same answer a file predating the fingerprint
	// gives. Both mean "this pack does not say", and a caller has one thing to
	// check instead of two.
	got, _ := packed()

	return got.Corpus
}

// Packed is what data/genres.json holds: the measured genres, and the
// fingerprint of the manifests they were measured from.
//
// A wrapper rather than a bare list, which is what the file was. The list alone
// could not say what it was measured over, so a pack and a corpus that had
// drifted apart agreed with each other perfectly and nothing could tell.
type Packed struct {
	// Corpus is [CorpusSHA] over the manifest tree at the time of packing.
	Corpus string `json:"corpus"`
	// Genres is what each genre measured as.
	Genres []Genre `json:"genres"`
}

// unpackGenres reads the measured genres.
//
// Apart from the embed so the failure has a test. The committed file always
// parses, and go:embed refuses to compile without it, so nothing else can
// reach this error.
func unpackGenres(
	body []byte,
) (Packed, error) {
	var out Packed
	if err := json.Unmarshal(body, &out); err != nil {
		return Packed{}, fmt.Errorf("reading the measured genres: %w", err)
	}

	return out, nil
}

// ShippedGenre is one measured genre by its slug, and whether there is one.
//
// No error, because there is none to report. The file is embedded and has its
// own test, so parsing it cannot fail here, the same way reading the blank
// preset cannot. Absent is said with the bool: a request may name any word, and
// nothing measured is an answer rather than a fault.
func ShippedGenre(
	slug string,
) (Genre, bool) {
	all, _ := Shipped()

	for _, g := range all {
		if g.Slug == slug {
			return g, true
		}
	}

	return Genre{}, false
}
