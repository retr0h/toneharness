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

package mcp_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/retr0h/toneharness/cmd"
	"github.com/retr0h/toneharness/pkg/mcp"
	"github.com/retr0h/toneharness/pkg/sdk"
)

// MCP is the whole agent surface or it is a trap: an agent that reaches for a
// capability over MCP and does not find one cannot tell "this tool does not do
// that" from "nobody registered it yet". Fifteen tools stood against
// thirty-two commands because six measure commands, four corpus listings and
// three device calls were all added after the server was written, and nothing
// said so.
//
// So the tree is the authority and this pairs it against the registered tools,
// both ways. A command with no tool fails, and a tool naming no command fails
// too: the second is what catches a name the CLI has moved on from, which is
// how `device_hardware` outlived `devices`.

// exempt is a command that is deliberately not a tool, and why.
//
// A reason is required rather than decorative. The list is the one place a
// capability may be missing from the agent surface, so an entry nobody can
// justify in a sentence is an entry that should be a tool.
var exempt = map[string]string{
	"mcp start": "the server itself; a tool that starts it would be the server asking for another one",

	// A sweep is a campaign rather than a call. One reading takes about eight
	// seconds, so a twelve-control amplifier is most of an hour and a library
	// of blocks is a night, and a tool that blocks that long is not a tool
	// anybody can use. They also need the audio loop: the pedal on USB, the
	// reference signal going in, ffmpeg. Run them from a terminal where the
	// progress is visible and Ctrl-C reaches the session holding the pedal.
	"measure blocks":   "a sweep of every block, hours, and it needs the audio loop",
	"measure controls": "a sweep of one block's controls, up to an hour, and it needs the audio loop",
	"measure names":    "a sweep that moves every parameter to learn its name, and it needs the audio loop",

	// The same shape as a sweep and for the same reasons. A pass reads a slope
	// per control and there are three to five passes, so a chain of two dozen
	// dials is twenty minutes of real-time audio, and it holds the pedal for
	// all of it.
	"tone tune": "a measure-solve-apply loop, about twenty minutes, and it needs the audio loop",
}

// toolName is what a command's tool is called: the path under the root, joined
// by underscores.
//
// Derived rather than declared, because the CLI's namespaces were consolidated
// once already and hand-written names did not follow. `device select` is
// `device_select` and there is nowhere for the two to disagree.
func toolName(
	c *cobra.Command,
) string {
	path := strings.Fields(c.CommandPath())

	return strings.Join(path[1:], "_")
}

// leaves is every command an agent could be asked to run: runnable, visible,
// and not cobra's own.
func leaves(
	root *cobra.Command,
) []*cobra.Command {
	var out []*cobra.Command

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			switch {
			case sub.Hidden, sub.Name() == "help", sub.Name() == "completion":
				continue
			case sub.HasSubCommands():
				walk(sub)
			case sub.Runnable():
				out = append(out, sub)
			}
		}
	}
	walk(root)

	return out
}

// TestEveryCommandIsATool holds the agent surface to the command tree.
func TestEveryCommandIsATool(
	t *testing.T,
) {
	t.Parallel()

	registered := registeredTools(t)
	require.NotEmpty(t, registered, "the server registered nothing, so this proves nothing")

	var missing []string

	seen := map[string]bool{}

	for _, c := range leaves(cmd.Root()) {
		name := toolName(c)
		seen[name] = true

		path := strings.Join(strings.Fields(c.CommandPath())[1:], " ")

		if why, ok := exempt[path]; ok {
			assert.NotEmpty(t, why, "%s is exempt with no reason", path)
			assert.NotContains(t, registered, name,
				"%s is exempt and also registered, so one of the two is wrong", path)

			continue
		}

		if !slicesContains(registered, name) {
			missing = append(missing, path+" wants a tool named "+name)
		}
	}

	sort.Strings(missing)
	assert.Empty(t, missing, "commands an agent cannot reach over MCP")
}

// TestEveryToolIsACommand is the other direction, and the one that catches a
// name the CLI has moved on from.
func TestEveryToolIsACommand(
	t *testing.T,
) {
	t.Parallel()

	registered := registeredTools(t)
	require.NotEmpty(t, registered, "the server registered nothing, so this proves nothing")

	named := map[string]bool{}
	for _, c := range leaves(cmd.Root()) {
		named[toolName(c)] = true
	}

	var orphans []string

	for _, tool := range registered {
		if !named[tool] {
			orphans = append(orphans, tool)
		}
	}

	sort.Strings(orphans)
	assert.Empty(
		t,
		orphans,
		"tools naming no command, so an agent is offered something the CLI cannot do",
	)
}

// slicesContains keeps the assertion above reading as one line.
func slicesContains(
	in []string,
	want string,
) bool {
	for _, got := range in {
		if got == want {
			return true
		}
	}

	return false
}

// registeredTools is every tool the server offers, named.
//
// With writes allowed, because the question here is whether a capability is
// reachable at all rather than whether this server was started willing to use
// it. Which tools --allow-writes gates is register's own test.
func registeredTools(
	t *testing.T,
) []string {
	t.Helper()

	serverEnd, clientEnd := gomcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = mcp.New(sdk.New(), mcp.Options{AllowWrites: true}).Serve(ctx, serverEnd) }()

	session, err := gomcp.NewClient(
		&gomcp.Implementation{Name: "test", Version: "test"}, nil,
	).Connect(context.Background(), clientEnd, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	listed, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	out := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		out = append(out, tool.Name)
	}

	return out
}
