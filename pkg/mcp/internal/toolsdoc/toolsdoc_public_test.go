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

package toolsdoc_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/toolsdoc"
)

// ToolsdocPublicTestSuite covers the page the registered tools generate.
type ToolsdocPublicTestSuite struct {
	suite.Suite

	page string
}

func (s *ToolsdocPublicTestSuite) SetupSuite() {
	body, err := toolsdoc.Render(context.Background())
	s.Require().NoError(err)

	s.page = string(body)
}

// TestTheShippedPageIsCurrent is what keeps the page honest.
//
// The set of tools changed twice in a month with no page moving, because there
// was no page. This fails the moment docs/mcp.md and the registration disagree,
// which is the only thing that stops the same drift happening to a generated
// page.
func (s *ToolsdocPublicTestSuite) TestTheShippedPageIsCurrent() {
	at := filepath.Join("..", "..", "..", "..", "docs", "mcp.md")

	got, err := os.ReadFile(at) //nolint:gosec // a path this repository owns
	s.Require().NoError(err)

	s.Require().Equal(s.page, string(got),
		"docs/mcp.md is out of date — run `just generate`")
}

// TestEveryToolTheServerOffersIsOnThePage covers the point of generating it.
//
// The tools are listed here the way the generator lists them, from a server of
// this test's own, so a tool added to the registration and left off the page
// fails here rather than being noticed by somebody reading the page a month
// later.
func (s *ToolsdocPublicTestSuite) TestEveryToolTheServerOffersIsOnThePage() {
	offered := s.offered(true)
	s.Require().NotEmpty(offered)

	for _, t := range offered {
		s.Require().Contains(s.page, "### "+t.Name,
			"%s is registered and the page gives it no section", t.Name)
		s.Require().Contains(s.page, "[`"+t.Name+"`](#"+t.Name+")",
			"%s is registered and the summary table leaves it out", t.Name)

		for name := range properties(t.InputSchema) {
			s.Require().Contains(s.page, "| `"+name+"` |",
				"%s takes %s and the page does not say so", t.Name, name)
		}
	}
}

// TestEveryToolSaysWhatItDoes holds the descriptions to being on the page.
//
// A description is what the model reads before deciding to call the tool, so
// the page and the model have to be shown the same sentence. The page carries
// the whole of it, wrapped.
func (s *ToolsdocPublicTestSuite) TestEveryToolSaysWhatItDoes() {
	for _, t := range s.offered(true) {
		s.Require().NotEmpty(t.Description, "%s describes itself to nobody", t.Name)

		// Wrapped on the page, so the first few words rather than all of it.
		// The whole description is compared word for word in the internal
		// test, where the rendering is reachable.
		s.Require().Contains(s.page, strings.Join(strings.Fields(t.Description)[:4], " "),
			"%s: the page does not carry its description", t.Name)
	}
}

// TestTheWriteToolsAreTheOnesTheFlagAdds holds the claim the page makes about
// --allow-writes.
//
// The page says an agent cannot see a write tool unless the server was started
// with the flag. That is a claim about the registration, not about the page, so
// it is checked against two servers rather than trusted.
func (s *ToolsdocPublicTestSuite) TestTheWriteToolsAreTheOnesTheFlagAdds() {
	without := map[string]bool{}
	for _, t := range s.offered(false) {
		without[t.Name] = true
	}

	gated := 0

	for _, t := range s.offered(true) {
		if without[t.Name] {
			continue
		}

		gated++

		section := strings.SplitN(s.page, "### "+t.Name+"\n", 2)
		s.Require().Len(section, 2, "%s has no section", t.Name)
		s.Require().Contains(strings.SplitN(section[1], "\n### ", 2)[0],
			"Offered only when the server was started with `--allow-writes`",
			"%s is offered only with the flag and its section does not say so", t.Name)
		s.Require().Contains(s.page, "| `"+t.Name+"` | ",
			"%s is not in the table of what the flag adds", t.Name)
	}

	s.Require().Positive(gated, "the flag adds nothing, so the page describes nothing")
}

// TestAPageIsRenderedOnce covers that rendering twice gives the same page.
//
// A generated page compared against the generator has to be stable across
// runs, and a map iterated in whatever order Go felt like would fail that
// intermittently, which is the worst way to find out.
func (s *ToolsdocPublicTestSuite) TestAPageIsRenderedOnce() {
	again, err := toolsdoc.Render(context.Background())
	s.Require().NoError(err)

	s.Require().Equal(s.page, string(again))
}

// offered is every tool a server started with allowWrites lists.
func (s *ToolsdocPublicTestSuite) offered(
	allowWrites bool,
) []*gomcp.Tool {
	ctx := context.Background()

	server := gomcp.NewServer(
		&gomcp.Implementation{Name: "tonestack", Version: "test"}, nil)

	pedal := tools.Register(server, nil, allowWrites)
	s.T().Cleanup(func() { _ = pedal.Close() })

	serverEnd, clientEnd := gomcp.NewInMemoryTransports()

	_, err := server.Connect(ctx, serverEnd, nil)
	s.Require().NoError(err)

	session, err := gomcp.NewClient(
		&gomcp.Implementation{Name: "test", Version: "test"}, nil,
	).Connect(ctx, clientEnd, nil)
	s.Require().NoError(err)

	s.T().Cleanup(func() { _ = session.Close() })

	res, err := session.ListTools(ctx, nil)
	s.Require().NoError(err)

	return res.Tools
}

// properties is a listed schema's own properties, which arrive as a map
// because that is what unmarshalling JSON produces.
func properties(
	of any,
) map[string]any {
	schema, ok := of.(map[string]any)
	if !ok {
		return nil
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}

	return props
}

func TestToolsdocPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ToolsdocPublicTestSuite))
}
