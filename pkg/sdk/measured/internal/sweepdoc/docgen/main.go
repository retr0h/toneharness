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

// Command docgen writes docs/measurements.md from the measurements.
//
// Skips and says so when the readings are not there, the way every other
// generator here does: `just generate` runs on machines that have never
// measured anything.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/retr0h/tonestack/pkg/sdk/measured"
	"github.com/retr0h/tonestack/pkg/sdk/measured/internal/sweepdoc"
)

// root is the repository, from this package: six levels up.
const root = "../../../../.."

// swept is the block whose controls the page reads.
//
// One block rather than every file in the tree, because the page explains what
// a curve is by walking one, and a page walking eleven would explain nothing
// eleven times.
const swept = "us-dripman-norm.json"

func main() {
	if err := write(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// write renders the page and puts it where the docs live.
func write() error {
	at := filepath.Join(root, "resources", "sweeps", "hx-stomp", swept)

	f, err := os.Open(at) //nolint:gosec // a path this repository owns
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("  measurements page: skipped, no %s\n", swept)

		return nil
	}

	if err != nil {
		return fmt.Errorf("reading %s: %w", at, err)
	}

	defer func() { _ = f.Close() }()

	curves, err := measured.LoadCurves(f)
	if err != nil {
		return err
	}

	lib, err := measured.BuiltIn()
	if err != nil {
		return err
	}

	body, err := sweepdoc.Render(lib, curves)
	if err != nil {
		return err
	}

	to := filepath.Join(root, "docs", "measurements.md")

	if held, err := os.ReadFile(to); err == nil && string(held) == string(body) {
		fmt.Printf("  measurements page: unchanged\n")

		return nil
	}

	if err := os.WriteFile(to, body, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", to, err)
	}

	fmt.Printf("  wrote %s\n", to)

	return nil
}
