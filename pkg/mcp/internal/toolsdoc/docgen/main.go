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

// Command docgen writes the MCP tool reference.
//
// Run by `just generate` through the directive in generate.go, and checked by a
// test that fails when the page and the registered tools disagree.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/retr0h/tonestack/pkg/mcp/internal/toolsdoc"
)

// out is where the page goes, worked out from this file rather than from
// wherever somebody ran the command. CONTRIBUTING asks a generator to write by
// a path relative to itself, so which directory you are in does not matter.
func out() (string, error) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot tell where this generator lives")
	}

	// pkg/mcp/internal/toolsdoc/docgen/main.go, so the repository is five
	// directories above the one this file is in.
	root := self
	for range 6 {
		root = filepath.Dir(root)
	}

	return filepath.Join(root, "docs", "mcp.md"), nil
}

func main() {
	if err := write(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// write renders the page and puts it where the docs live.
func write() error {
	to, err := out()
	if err != nil {
		return err
	}

	body, err := toolsdoc.Render(context.Background())
	if err != nil {
		return err
	}

	if held, err := os.ReadFile(to); err == nil && string(held) == string(body) {
		fmt.Printf("  mcp page: unchanged\n")

		return nil
	}

	if err := os.WriteFile(to, body, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", to, err)
	}

	fmt.Printf("  wrote %s\n", to)

	return nil
}
