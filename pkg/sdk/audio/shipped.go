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
	return unpackGenres(shipped)
})

// unpackGenres reads the measured genres.
//
// Apart from the embed so the failure has a test. The committed file always
// parses, and go:embed refuses to compile without it, so nothing else can
// reach this error.
func unpackGenres(
	body []byte,
) ([]Genre, error) {
	var out []Genre
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("reading the measured genres: %w", err)
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
