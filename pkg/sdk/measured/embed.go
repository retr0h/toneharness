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
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"sync"
)

// The measurements, one file per device they were taken on.
//
// They ship inside the binary for the reason the catalog does: choosing a
// block should not need the hardware. Measuring needs it, and that happens
// once, on one machine, not on every machine that runs this.
//
// Gzipped because it is repetitive JSON, the same as the catalog beside it.
//
// Taken rather than generated. Every number came off a pedal, through the
// loop in docs/measuring.md, against the one reference signal named inside
// the file. Re-measure with `tonestack measure blocks`; `just generate` packs
// what that wrote into here.
//
//go:embed data/hx-stomp.json.gz
var builtIn []byte

// BuiltIn is the measurements this binary ships, parsed once.
//
// Parsed on the first call and not again, because unpacking and decoding six
// hundred readings is not cheap and they never change inside a binary.
var BuiltIn = sync.OnceValues(func() (Library, error) {
	return unpack(builtIn)
})

// unpack reads a gzipped library.
func unpack(
	packed []byte,
) (Library, error) {
	r, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		return Library{}, fmt.Errorf("unpacking the measurements: %w", err)
	}

	defer func() { _ = r.Close() }()

	lib, err := Load(r)
	if err != nil {
		return Library{}, err
	}

	// A library that was not measured one block at a time cannot be used to
	// choose one. Every reading in it would describe whatever else was in the
	// chain beside it, and nothing downstream could tell.
	if !lib.Isolated {
		return Library{}, fmt.Errorf(
			"these measurements were taken in a chain, so they describe the " +
				"chain rather than the blocks in it")
	}

	return lib, nil
}

// Packed writes a library out the way this package embeds one.
//
// Here rather than in a script so the writer and the reader cannot drift: a
// file this cannot read is a file it will not produce.
func Packed(
	w io.Writer,
	from io.Reader,
) error {
	body, err := io.ReadAll(from)
	if err != nil {
		return fmt.Errorf("reading the measurements: %w", err)
	}

	// Read before it is written, so a library that is not usable is refused
	// at the point somebody could still fix it.
	if _, err := Load(bytes.NewReader(body)); err != nil {
		return err
	}

	z := gzip.NewWriter(w)

	if _, err := z.Write(body); err != nil {
		return fmt.Errorf("packing the measurements: %w", err)
	}

	return z.Close()
}
