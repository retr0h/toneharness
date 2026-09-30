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

package corpus

import (
	"fmt"
	"os"
)

// Open reads statistics from a path, or the ones that ship when there is none.
//
// The sibling of catalog.Open, and here for the same reason: two packages had
// each written it, byte for byte, down to the //nolint and the comment on the
// deferred Close. Where the statistics come from is this package's to answer.
//
// No path means the ones compiled into the binary, which is the case for
// anybody who has not measured their own corpus.
func Open(
	path string,
) (*Stats, error) {
	if path == "" {
		return BuiltIn()
	}

	f, err := os.Open(path) //nolint:gosec // a path the caller named
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	return Load(f)
}
