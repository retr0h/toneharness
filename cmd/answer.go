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

package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/cli"
)

// asData is set by --json, and is a persistent flag so it reads the same on
// every command rather than being a thing one of them happens to have.
var asData bool

// Registered here rather than in Execute, because Root() hands the command
// tree to anything that reads it rather than runs it — the generated command
// reference, and the MCP server — and a flag added on the way to running is a
// flag none of them can see. docs/commands.md lost it exactly that way.
func init() {
	rootCmd.PersistentFlags().BoolVar(&asData, "json", false,
		"answer as data rather than as a table")
}

// answer writes what an operation produced, as a table or as data.
//
// The one place that choice is made. Every command ends in a renderer, and
// every renderer takes a writer and one value, so this takes the value and
// the renderer and picks. A call site stays one line, which is the point: a
// command that had to remember to check a flag is a command that will forget.
//
// Generic over the value so the renderer is passed by name rather than
// wrapped in a closure at each of the sixty call sites.
func answer[T any](
	cmd *cobra.Command,
	of T,
	draw func(io.Writer, T) error,
) error {
	w := cmd.OutOrStdout()

	if asData {
		return cli.Data(w, of)
	}

	return draw(w, of)
}
