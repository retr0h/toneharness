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

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/retr0h/toneharness/cmd"
)

// named finds a command this repository tells somebody to run.
//
// Quoted, because that is what makes it an invocation rather than a sentence
// about the project: "toneharness builds Line 6 presets" names no command and
// six files say something of that shape. The quote also ends the match, so
// there is no guessing where the command stops and the prose resumes.
var named = regexp.MustCompile("['`]toneharness ([a-z][a-z ]*?)['`]")

// TestEveryCommandThisRepositoryNamesExists holds every command this repository
// names to the command tree.
//
// An error hint is the one piece of documentation somebody reads at the moment
// they are stuck, so a hint naming a command that does not exist sends them
// somewhere with nothing in it. Two did after the namespaces were consolidated:
// `toneharness devices list` and `toneharness presets list`, both of which had
// moved, and both of which still read as plausible advice.
//
// Grep does not catch this. `presets list --help` exits zero and prints the
// parent's help, so asking the binary whether a command exists answers yes for
// anything whose first word is a real namespace. The tree has to be walked.
func TestEveryCommandThisRepositoryNamesExists(
	t *testing.T,
) {
	t.Parallel()

	paths := reachable(cmd.Root())
	require.NotEmpty(t, paths, "walked no commands, so this proves nothing")

	var wrong []string

	for _, file := range sources(t) {
		raw, err := os.ReadFile(file)
		require.NoError(t, err)

		for _, m := range named.FindAllStringSubmatch(string(raw), -1) {
			if at := strings.TrimSpace(m[1]); !paths[at] {
				wrong = append(wrong, file+" names 'toneharness "+at+"'")
			}
		}
	}

	sort.Strings(wrong)
	assert.Empty(t, wrong, "commands this repository names that the tree does not have")
}

// reachable is every command path under the root, runnable or not.
//
// A namespace counts: `toneharness device` on its own prints what it holds,
// which is a reasonable thing to tell somebody to run.
func reachable(
	root *cobra.Command,
) map[string]bool {
	out := map[string]bool{}

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}

			out[strings.Join(strings.Fields(sub.CommandPath())[1:], " ")] = true

			walk(sub)
		}
	}
	walk(root)

	return out
}

// sources is every Go file that could name a command.
//
// Tests are left out: one naming a command deliberately to check a refusal is
// not a hint, and reading it as one would fail on purpose.
func sources(
	t *testing.T,
) []string {
	t.Helper()

	var out []string

	err := filepath.WalkDir(".", func(at string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		switch {
		case d.IsDir() && (d.Name() == ".git" || d.Name() == ".claude" || d.Name() == ".worktrees"):
			return filepath.SkipDir
		case d.IsDir(), !strings.HasSuffix(at, ".go"), strings.HasSuffix(at, "_test.go"):
			return nil
		}

		out = append(out, at)

		return nil
	})
	require.NoError(t, err)

	return out
}
