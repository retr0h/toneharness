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

package toolsdoc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"text/template"

	"github.com/google/jsonschema-go/jsonschema"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

// ToolsdocTestSuite covers what the rendered page cannot be asked about from
// outside: the annotations a tool could carry and does not, and the shape the
// page groups the tools into.
type ToolsdocTestSuite struct {
	suite.Suite
}

// TestTheToolsThatNeedAPedalAreTheOpenWorldOnes holds the claim the page makes
// about what needs hardware attached.
//
// The page says a tool with no pedal answers from what ships, and the only
// thing it has to go on is each tool's own openWorldHint. Nothing makes an
// author set that, so this names the tools that reach the bus. A device tool
// added without the annotation is reported as offline, which is the wrong way
// round for somebody deciding whether to quit HX Edit first.
func (s *ToolsdocTestSuite) TestTheToolsThatNeedAPedalAreTheOpenWorldOnes() {
	page := s.page()

	pedal := map[string]bool{}
	for _, t := range page.Pedal {
		pedal[t.Name] = true
	}

	s.Require().Equal(map[string]bool{
		"devices_list":  true,
		"presets_list":  true,
		"preset_show":   true,
		"preset_export": true,
		"preset_select": true,
		"preset_import": true,
		"presets_copy":  true,
		"presets_swap":  true,
	}, pedal, "the tools that reach the pedal are not the ones marked open-world")

	s.Require().Len(page.Tools, len(page.Pedal)+len(page.Offline))
	s.Require().Equal(len(page.Tools)-len(page.Gated), page.Always)
}

// TestEveryDescriptionReachesThePageWhole covers the wrapping.
//
// The summary table carries a sentence and the tool's own section carries all
// of it. A description arrives as one long line, and the formatter is not
// allowed near a generated page, so the generator wraps it and this is what
// says the wrapping did not drop anything.
func (s *ToolsdocTestSuite) TestEveryDescriptionReachesThePageWhole() {
	page := s.page()

	body, err := render(page)
	s.Require().NoError(err)

	for _, t := range page.Tools {
		s.Require().Contains(string(body), fill(t.Description),
			"%s: its description is not on the page as written", t.Name)
	}
}

// TestAServerRegisteringNothingIsRefused covers a page with no tools on it.
//
// Rather than writing an empty reference over the real one, which is the shape
// of failure that gets committed.
func (s *ToolsdocTestSuite) TestAServerRegisteringNothingIsRefused() {
	_, err := read(nil, nil)

	s.Require().ErrorContains(err, "registers no tools")
}

// TestAToolWhoseSchemaWillNotReadIsRefused covers a tool the library would
// never register.
//
// A schema that does not read back is a tool an agent cannot call either, so
// the generator says which tool rather than rendering a section with an empty
// table under it.
func (s *ToolsdocTestSuite) TestAToolWhoseSchemaWillNotReadIsRefused() {
	for _, tc := range []struct {
		name  string
		given *gomcp.Tool
	}{
		{"its input", &gomcp.Tool{Name: "bad_in", InputSchema: make(chan int)}},
		{"its output", &gomcp.Tool{
			Name:         "bad_out",
			InputSchema:  map[string]any{"type": "object"},
			OutputSchema: make(chan int),
		}},
	} {
		s.Run(tc.name, func() {
			_, err := toolOf(tc.given, false)
			s.Require().ErrorContains(err, tc.given.Name)

			_, err = read(nil, []*gomcp.Tool{tc.given})
			s.Require().ErrorContains(err, tc.given.Name)
		})
	}
}

// TestWhatAToolIsSaidToWrite covers each label, including the two a
// registered tool does not currently produce.
//
// A tool carrying no annotations is reported as writing to the pedal. That is
// the protocol's own default for a tool that says nothing, and it is the safe
// way round: an unannotated tool shows up beside the destructive ones, where
// somebody will look at it.
func (s *ToolsdocTestSuite) TestWhatAToolIsSaidToWrite() {
	no := false
	yes := true

	for _, tc := range []struct {
		name  string
		given *gomcp.ToolAnnotations
		gated bool
		want  string
	}{
		{"nothing said", nil, false, writesPedal},
		{"gated", &gomcp.ToolAnnotations{ReadOnlyHint: true}, true, writesPedal},
		{"read-only", &gomcp.ToolAnnotations{ReadOnlyHint: true}, false, writesNothing},
		{"destructive", &gomcp.ToolAnnotations{DestructiveHint: &yes}, false, writesFile},
		{"not destructive", &gomcp.ToolAnnotations{DestructiveHint: &no}, false, writesLoaded},
	} {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, writes(tc.given, tc.gated))
		})
	}
}

// TestAToolWithNoWorldNamedReachesThePedal covers the other default.
func (s *ToolsdocTestSuite) TestAToolWithNoWorldNamedReachesThePedal() {
	closed := false

	s.Require().True(openWorld(nil))
	s.Require().True(openWorld(&gomcp.ToolAnnotations{}))
	s.Require().False(openWorld(&gomcp.ToolAnnotations{OpenWorldHint: &closed}))
}

// TestTheSummaryTakesTheFirstSentence covers a description written as one
// sentence, which has no full stop to cut at.
func (s *ToolsdocTestSuite) TestTheSummaryTakesTheFirstSentence() {
	s.Require().Equal("One slot on the pedal",
		sentence("One slot on the pedal. Read back as a rig."))
	s.Require().Equal("The rigs that ship", sentence("The rigs that ship."))
	s.Require().Empty(sentence(""))
}

// TestATypeIsNamedInAWord covers the types the page reports, including the
// ones no registered tool answers with yet.
func (s *ToolsdocTestSuite) TestATypeIsNamedInAWord() {
	for _, tc := range []struct {
		name  string
		given *jsonschema.Schema
		want  string
	}{
		{"nothing", nil, "any"},
		{"one type", &jsonschema.Schema{Type: "string"}, "string"},
		{"a nullable one", &jsonschema.Schema{Types: []string{"null", "object"}}, "object"},
		{"any JSON value", &jsonschema.Schema{
			Types: []string{"null", "boolean", "number", "string", "array", "object"},
		}, "any"},
		{"a list", &jsonschema.Schema{
			Types: []string{"array"}, Items: &jsonschema.Schema{Type: "string"},
		}, "array of string"},
		{"a list of nothing named", &jsonschema.Schema{Types: []string{"array"}}, "array"},
	} {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, typeOf(tc.given))
		})
	}
}

// TestASchemaThatIsNotOneIsRefused covers a tool carrying no output schema, and
// a schema that will not read back.
//
// Every tool registered here has both schemas, and a tool added without an
// output schema would otherwise render a heading over an empty table.
func (s *ToolsdocTestSuite) TestASchemaThatIsNotOneIsRefused() {
	held, err := schemaOf(nil)
	s.Require().NoError(err)
	s.Require().Nil(held)
	s.Require().Nil(fields(nil))

	_, err = schemaOf(make(chan int))
	s.Require().Error(err, "a value that is not JSON at all")

	_, err = schemaOf("not a schema")
	s.Require().Error(err, "JSON that is not a schema")
}

// TestAPipeDoesNotEndAColumnEarly covers a description carrying the character
// that separates a table's columns.
func (s *ToolsdocTestSuite) TestAPipeDoesNotEndAColumnEarly() {
	s.Require().Equal(`a \| b`, cell("  a | b  "))
	s.Require().Equal("yes", yesno(true))
	s.Require().Equal("no", yesno(false))
}

// TestProseWrapsWhereTheFormatterWouldHave covers the width.
func (s *ToolsdocTestSuite) TestProseWrapsWhereTheFormatterWouldHave() {
	for _, line := range strings.Split(fill(strings.Repeat("word ", 60)), "\n") {
		s.Require().LessOrEqual(len(line), 80)
		s.Require().NotEmpty(line)
	}

	s.Require().Equal("short enough", fill("  short   enough  "))
	s.Require().Empty(fill(""))
}

// page is what the registered tools say about themselves.
func (s *ToolsdocTestSuite) page() Page {
	ctx := context.Background()

	held, err := listed(ctx, false)
	s.Require().NoError(err)

	all, err := listed(ctx, true)
	s.Require().NoError(err)

	page, err := read(held, all)
	s.Require().NoError(err)

	return page
}

// TestATemplateThatFailsIsReported covers the error render can return.
//
// The shipped template is embedded and parsed at startup, so nothing a caller
// does reaches this. A test can, by handing it one that asks for a field the
// page has not got.
func (s *ToolsdocTestSuite) TestATemplateThatFailsIsReported() {
	_, err := Drawn(template.Must(template.New("x").Parse("{{ .Nope.Missing }}")), Page{})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing the MCP page")
}

// TestAContextAlreadyDoneIsReported covers listing the tools failing.
//
// The page is built by standing a server up in memory and asking it what it
// offers, so the failure a caller could actually see is the context going away
// underneath that.
func (s *ToolsdocTestSuite) TestAContextAlreadyDoneIsReported() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Listed(ctx, false)
	s.Require().Error(err)
}

// TestRenderReportsEachWayListingCanFail covers the three failures between
// asking a server what it offers and having a page.
//
// The real lister stands a server up in memory and cannot be made to fail on
// demand past a cancelled context, so each is provoked here instead: the first
// listing failing, the second failing, and both succeeding with nothing to
// report, which is a server that registered no tools at all.
func (s *ToolsdocTestSuite) TestRenderReportsEachWayListingCanFail() {
	boom := errors.New("no server")

	tests := []struct {
		name string
		list func(context.Context, bool) ([]*gomcp.Tool, error)
		want string
	}{
		{
			name: "the first listing fails",
			list: func(_ context.Context, _ bool) ([]*gomcp.Tool, error) {
				return nil, boom
			},
			want: "no server",
		},
		{
			name: "the second listing fails",
			list: func(_ context.Context, writes bool) ([]*gomcp.Tool, error) {
				if writes {
					return nil, boom
				}

				return []*gomcp.Tool{{Name: "catalog_list"}}, nil
			},
			want: "no server",
		},
		{
			name: "a server offering nothing",
			list: func(_ context.Context, _ bool) ([]*gomcp.Tool, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := Rendered(context.Background(), tt.list)
			s.Require().Error(err)

			if tt.want != "" {
				s.Require().ErrorIs(err, boom)
			}
		})
	}
}

func TestToolsdocTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ToolsdocTestSuite))
}
