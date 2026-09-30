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

// Package packed writes a generated file as gzipped bytes, and rewrites it only
// when its contents changed.
//
// Both generators here embed their output in the binary and both commit it, so
// both compress and both compare before writing. Written twice, the pair had
// one invariant between them that neither said out loud: the comparison is
// byte-for-byte, so the compression has to be deterministic for the same input.
// gzip's header carries a modification time, and setting it would make every
// run rewrite the committed file with identical content.
package packed

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
)

// Bytes gzips a generated blob.
//
// Deterministic, which is what the comparison in Refresh rests on: the header's
// ModTime is left at nothing, so the same input compresses to the same bytes on
// every run and on every machine. Setting it would churn the committed file.
//
// Worth the compression rather than committing the JSON: a catalog is 1.5MB of
// repetitive JSON and about 65KB gzipped, which is the difference between the
// device knowledge being worth shipping and not.
func Bytes(
	raw []byte,
) []byte {
	var buf bytes.Buffer

	// Compressing into a buffer cannot fail.
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(raw)
	_ = zw.Close()

	return buf.Bytes()
}

// Refresh writes body to path unless what is there already matches, and says
// whether it wrote.
//
// The comparison is what keeps a regenerate out of a diff when nothing about
// the source changed, which is how `just ready` stays quiet on a branch that
// touched no generated input.
//
// A path that cannot be read is written rather than refused: that is a first
// run, and a fresh clone has no generated file.
func Refresh(
	path string,
	body []byte,
) (bool, error) {
	if was, err := os.ReadFile(path); err == nil &&
		bytes.Equal(was, body) { //nolint:gosec // the path is the generator's own
		return false, nil
	}

	if err := os.WriteFile(path, body, 0o600); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}

	return true, nil
}
