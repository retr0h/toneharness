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

package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/mcp/internal/tools"
	"github.com/retr0h/toneharness/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
)

// connect puts the tools on a server and a client session in front of it,
// over the library's in-memory transport, so a test calls a tool the way an
// agent does.
func connect(
	t *testing.T,
	c tools.Client,
	allowWrites bool,
) *gomcp.ClientSession {
	t.Helper()

	server := gomcp.NewServer(&gomcp.Implementation{Name: "toneharness", Version: "test"}, nil)
	pedal := tools.Register(server, c, allowWrites)

	// The pedal is let go when the test ends, as the server lets it go when it
	// stops.
	t.Cleanup(func() { _ = pedal.Close() })

	return serve(t, server)
}

// connectIdle is connect with the pedal let go once idle has passed, and
// hands back what holds it.
func connectIdle(
	t *testing.T,
	c tools.Client,
	allowWrites bool,
	idle time.Duration,
) (*gomcp.ClientSession, io.Closer) {
	t.Helper()

	server := gomcp.NewServer(&gomcp.Implementation{Name: "toneharness", Version: "test"}, nil)
	pedal := tools.RegisterIdle(server, c, allowWrites, idle)

	// A test that asserts on closing it has already closed it, and a second
	// Close holds nothing.
	t.Cleanup(func() { _ = pedal.Close() })

	return serve(t, server), pedal
}

// serve puts a client session in front of server.
func serve(
	t *testing.T,
	server *gomcp.Server,
) *gomcp.ClientSession {
	t.Helper()

	serverEnd, clientEnd := gomcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, serverEnd, nil)
	require.NoError(t, err)

	session, err := gomcp.NewClient(
		&gomcp.Implementation{Name: "test", Version: "test"}, nil,
	).Connect(ctx, clientEnd, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = session.Close() })

	return session
}

// call calls one tool and fails the test only on a protocol error. A tool
// error comes back as a result with IsError set, which is what the tests read.
func call(
	t *testing.T,
	session *gomcp.ClientSession,
	name string,
	args any,
) *gomcp.CallToolResult {
	t.Helper()

	res, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)

	return res
}

// text is the line a tool said.
func text(
	t *testing.T,
	res *gomcp.CallToolResult,
) string {
	t.Helper()
	require.NotEmpty(t, res.Content)

	said, ok := res.Content[0].(*gomcp.TextContent)
	require.True(t, ok, "first content is %T", res.Content[0])

	return said.Text
}

// structured decodes what a tool answered into a Go value.
func structured(
	t *testing.T,
	res *gomcp.CallToolResult,
	into any,
) {
	t.Helper()

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, into))
}

type RegisterPublicTestSuite struct {
	suite.Suite
}

// TestEveryDeviceToolCarriesItsRemedy is the asymmetry the CLI already fixed.
//
// Hint was called by hand from four commands and none of them were the device
// ones, so an unplugged pedal was the one error with no next step. The MCP
// side had the same shape: remedy existed and four of twenty-nine handlers
// called it, none of them the ones that reach hardware.
//
// Every tool that opens the pedal, so the next one registered is covered by
// having been registered rather than by somebody remembering.
func (s *RegisterPublicTestSuite) TestEveryDeviceToolCarriesItsRemedy() {
	for _, tool := range []struct {
		name string
		args any
	}{
		{"device_current", map[string]any{}},
		{"device_select", map[string]any{"slot": "1A"}},
		{"slots_list", map[string]any{}},
		{"presets_show", map[string]any{"slot": "1A"}},
		{"slots_export", map[string]any{"slot": "1A", "out": "x.hlx"}},
		{"device_turn", map[string]any{"block": 1, "param": 1, "value": 0.5}},
	} {
		s.Run(tool.name, func() {
			ctrl := gomock.NewController(s.T())
			c := mocks.NewMockClient(ctrl)
			c.EXPECT().Open(gomock.Any()).Return(nil, sdk.ErrNoDevice).AnyTimes()

			res := call(s.T(), connect(s.T(), c, true), tool.name, tool.args)

			s.True(res.IsError, "an unattached pedal is an error")
			s.Contains(text(s.T(), res), "USB data port",
				"and one the agent can act on")
		})
	}
}

// TestAnErrorWithNoRemedyIsLeftAlone is the other half of the wrapper.
//
// remedy adds to the errors it recognises and hands the rest back as they
// were, so a tool that fails for its own reasons still says its own thing.
func (s *RegisterPublicTestSuite) TestAnErrorWithNoRemedyIsLeftAlone() {
	ctrl := gomock.NewController(s.T())
	c := mocks.NewMockClient(ctrl)
	c.EXPECT().Open(gomock.Any()).
		Return(nil, errors.New("the editor interface is in use, quit HX Edit")).
		AnyTimes()

	res := call(s.T(), connect(s.T(), c, true), "device_current", map[string]any{})

	s.True(res.IsError)
	s.Contains(text(s.T(), res), "quit HX Edit")
	s.NotContains(text(s.T(), res), "USB data port")
}

// TestRegister covers which tools an agent is offered.
func (s *RegisterPublicTestSuite) TestRegister() {
	reads := []string{
		"catalog_show", "catalog_list", "corpus_presets_show", "corpus_presets_chains",
		"corpus_music_players", "corpus_music_bands", "corpus_music_genres",
		"corpus_music_records",
		"tone_build", "presets_make", "rigs_show", "rigs_list", "rigs_records",
		"measure_genres", "measure_players", "measure_recordings",
		"device_hardware", "slots_list", "presets_show", "slots_export", "device_select",
		"device_current", "device_play", "device_turn",
	}

	tests := []struct {
		name        string
		allowWrites bool
		want        []string
		readOnly    map[string]bool
	}{
		{
			name: "without writes",
			want: reads,
			readOnly: map[string]bool{
				"catalog_show": true, "catalog_list": true, "corpus_presets_show": true,
				"tone_build": true, "presets_make": false,
				"rigs_show": true, "rigs_list": true,
				"device_hardware": true, "slots_list": true, "presets_show": true,
				"slots_export": false, "device_select": false,
				"corpus_presets_chains": true, "corpus_music_players": true,
				"corpus_music_bands": true, "corpus_music_genres": true,
				"corpus_music_records": true, "rigs_records": true,
				"measure_genres": true, "measure_players": true,
				"measure_recordings": true, "device_current": true,
			},
		},
		{
			name:        "with writes",
			allowWrites: true,
			want: append(
				slices.Clone(reads),
				"rigs_new",
				"presets_compile",
				"slots_import",
				"slots_copy",
				"slots_swap",
			),
			readOnly: map[string]bool{
				"rigs_new":        false,
				"presets_compile": false,
				"slots_import":    false,
				"slots_copy":      false,
				"slots_swap":      false,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			client := mocks.NewMockClient(gomock.NewController(s.T()))
			session := connect(s.T(), client, tt.allowWrites)

			listed, err := session.ListTools(context.Background(), nil)
			s.Require().NoError(err)

			var names []string
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)

				if want, ok := tt.readOnly[tool.Name]; ok {
					s.Equal(want, tool.Annotations.ReadOnlyHint, tool.Name)
				}

				if tool.Name == "device_select" {
					s.Require().NotNil(tool.Annotations.DestructiveHint)
					s.False(*tool.Annotations.DestructiveHint)
					s.True(tool.Annotations.IdempotentHint)
				}

				switch tool.Name {
				case "slots_import",
					"slots_copy",
					"slots_swap",
					"slots_export",
					"presets_make":
					s.Require().NotNil(tool.Annotations.DestructiveHint, tool.Name)
					s.True(*tool.Annotations.DestructiveHint, tool.Name)
				}
			}

			s.ElementsMatch(tt.want, names)
		})
	}
}

func TestRegisterPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RegisterPublicTestSuite))
}
