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
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/retr0h/toneharness/cmd"
)

// ownRigsCall is how a command layers somebody's own rigs over the built-in
// ones. Matched in the source because nothing on a built Client says which
// options made it, and the flag alone does not: `tone reach` had the flag and
// handed it to WithRigs, which replaces the built-in rigs rather than layering
// over them, so a rig of your own stayed unread while the usage said otherwise.
const ownRigsCall = "ownRigs("

// TestEveryCommandNamingARigByIdentifierReadsYourOwn holds each command that
// takes a rig's identifier to reading the rigs somebody wrote.
//
// Rigs live in two places. The ones that ship are beneath, and a person's own
// are in $XDG_DATA_HOME/toneharness/rigs layered over them, which is where
// `rigs new` writes and where somebody puts a rig they were handed. `ownRigs`
// is the option that layers them, and a command that forgets it can only ever
// find a rig that ships.
//
// Three commands had forgotten it, and `tone tune` is the one that mattered:
// its own --extend writes a rig, prints "read it, keep or drop the block, and
// tune it again", and tuning it again answered `no such rig "punk-fuzz" (43
// known)`. The instruction the tool gives could not be followed.
//
// Asking cobra is not enough. A flag named rigs can be wired to either option
// and both compile, so this reads the source of the file that registers the
// command's --id instead. That is coarse, and it is the only thing that
// distinguishes the two wirings.
// Not parallel, deliberately. cmd.Root() is one shared tree and cobra sorts a
// command's children the first time Commands() is called, so two tests walking
// it at once race on that write. The sequential test runs to completion while
// every parallel one in the package is still paused, which is what keeps the
// two walks apart.
func TestEveryCommandNamingARigByIdentifierReadsYourOwn(
	t *testing.T,
) {
	named := namesARig(t, cmd.Root())
	require.NotEmpty(t, named, "found no command taking a rig's identifier, so this proves nothing")

	var wrong []string

	for _, file := range declaring(t, named) {
		raw, err := os.ReadFile(file)
		require.NoError(t, err)

		if !strings.Contains(string(raw), ownRigsCall) {
			wrong = append(wrong, file)
		}
	}

	sort.Strings(wrong)
	assert.Empty(
		t,
		wrong,
		"these name a rig by identifier and never call %s, so a rig of "+
			"somebody's own is invisible to them",
		ownRigsCall,
	)
}

// namesARig is every command whose --id is a rig's identifier.
//
// By the flag's own usage rather than by a list here, because a list is right
// the day it is written. A command with an --id meaning something else, like
// the device identifier `device current` takes, says so in its usage and is
// left out.
func namesARig(
	t *testing.T,
	root *cobra.Command,
) []*cobra.Command {
	t.Helper()

	var out []*cobra.Command

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if id := sub.Flags().Lookup("id"); id != nil && mentionsARig(id) {
				out = append(out, sub)
			}

			walk(sub)
		}
	}
	walk(root)

	return out
}

// mentionsARig is whether a flag's usage says the thing it names is a rig.
func mentionsARig(
	f *pflag.Flag,
) bool {
	return strings.Contains(strings.ToLower(f.Usage), "rig")
}

// declaring is the source file that registers each command, found by the
// command's path rather than guessed: `tone tune` is cmd/tone_tune.go.
//
// A file that is not there fails the test rather than passing it, because a
// renamed file would otherwise silently check nothing.
func declaring(
	t *testing.T,
	cmds []*cobra.Command,
) []string {
	t.Helper()

	out := make([]string, 0, len(cmds))

	for _, c := range cmds {
		parts := strings.Fields(c.CommandPath())[1:]
		at := filepath.Join("cmd", strings.Join(parts, "_")+".go")

		_, err := os.Stat(at)
		require.NoErrorf(t, err, "%s registers %q", at, c.CommandPath())

		out = append(out, at)
	}

	return out
}
