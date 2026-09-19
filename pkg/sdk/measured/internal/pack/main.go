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
// The readings are taken by resources/fingerprint.py and land in
// resources/sweeps/, which is where somebody looks at them. This is the step
// that gets them into the binary, and it goes through the same reader the
// binary uses, so a library that will not load is refused here rather than at
// the next build.
package main

import (
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/measured"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: pack <measurements.json> <out.json.gz>")
		os.Exit(2)
	}

	in, err := os.Open(os.Args[1]) //nolint:gosec // a path the operator gave
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer func() { _ = in.Close() }()

	out, err := os.Create(os.Args[2]) //nolint:gosec // a path the operator gave
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := measured.Packed(out, in); err != nil {
		_ = out.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := out.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("  packed %s into %s\n", os.Args[1], os.Args[2])
}
