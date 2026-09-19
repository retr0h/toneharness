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

// Command pack puts a measured library into the form the sdk embeds.
//
// The readings are taken by `tonestack measure blocks` and land in
// resources/sweeps/, which is where somebody looks at them. This is the step
// that gets them into the binary, and it goes through the same reader the
// binary uses, so a library that will not load is refused here rather than at
// the next build.
package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/measured"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: pack <measurements.json> <out.json.gz>")
		os.Exit(2)
	}

	if err := pack(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// pack writes the readings into the form the sdk embeds.
//
// Skips and says so when the readings are not there, and writes only when what
// it built differs from what is committed. Both for the same reason as the
// catalog and the corpus beside it: `just generate` runs on machines that have
// never measured anything, and a generator that rewrites an identical file
// puts a diff in front of somebody on every run.
func pack(
	from, to string,
) error {
	body, err := os.ReadFile(from) //nolint:gosec // a path the operator gave
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("  measurements: skipped, no %s\n", from)

		return nil
	}

	if err != nil {
		return fmt.Errorf("reading the measurements: %w", err)
	}

	same, err := matches(to, body)
	if err != nil {
		return err
	}

	if same {
		fmt.Printf("  measurements: unchanged\n")

		return nil
	}

	var packed bytes.Buffer
	if err := measured.Packed(&packed, bytes.NewReader(body)); err != nil {
		return err
	}

	if err := os.WriteFile(to, packed.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing the measurements: %w", err)
	}

	fmt.Printf("  measurements: packed %s into %s\n", from, to)

	return nil
}

// matches reports whether the committed file already holds these readings.
//
// Compared unpacked rather than byte for byte, because two gzip streams of the
// same bytes are not required to be identical and a compressor that changed
// its mind would read as a change nobody made.
func matches(
	to string,
	body []byte,
) (bool, error) {
	f, err := os.Open(to) //nolint:gosec // a path the operator gave
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("reading the packed measurements: %w", err)
	}

	defer func() { _ = f.Close() }()

	r, err := gzip.NewReader(f)
	if err != nil {
		// Unreadable is not the same as different, but it is handled the
		// same way: write a good one over it.
		return false, nil //nolint:nilerr // a corrupt file is rewritten
	}

	defer func() { _ = r.Close() }()

	held, err := io.ReadAll(r)
	if err != nil {
		return false, nil //nolint:nilerr // a corrupt file is rewritten
	}

	return bytes.Equal(held, body), nil
}
