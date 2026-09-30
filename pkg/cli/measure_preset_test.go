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

package cli

import (
	"os"
	"testing"

	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// writeBlank writes a preset a reader can actually read.
//
// Every double for the compiler used to write the bytes "a preset", which is
// not one. That held while nothing read the file back, and stopped holding the
// moment a sweep had to point its output at USB before playing it: the double
// was describing a file the real compiler never produces.
//
// A double that lies about the shape of what it makes only fails when something
// downstream starts caring, which is the worst time to find out.
func writeBlank(
	t *testing.T,
	at string,
) {
	t.Helper()

	doc, err := preset.Blank()
	if err != nil {
		t.Fatalf("reading the blank preset: %v", err)
	}

	f, err := os.Create(at) //nolint:gosec // a path the test made up
	if err != nil {
		t.Fatalf("writing %s: %v", at, err)
	}

	if err := preset.Write(f, doc); err != nil {
		_ = f.Close()
		t.Fatalf("writing %s: %v", at, err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("writing %s: %v", at, err)
	}
}
