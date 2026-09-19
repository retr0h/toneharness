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
package specdoc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/specdoc"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	page "github.com/retr0h/tonestack/pkg/sdk/rig/internal/specdoc"
)

// PagePublicTestSuite covers the shipped grammar page for a rig.
type PagePublicTestSuite struct {
	suite.Suite
}

// TestTheShippedPageIsCurrent is what keeps the page honest.
//
// A generated reference that nobody regenerates is a hand-written one with
// extra steps, which is how docs/recipes.md came to describe a format that
// had moved. This fails the moment the contract and the page disagree, in the
// ordinary test run rather than in a step somebody has to remember.
func (s *PagePublicTestSuite) TestTheShippedPageIsCurrent() {
	want, err := specdoc.Render(rig.Schema, page.Page)
	s.Require().NoError(err)

	at := filepath.Join("..", "..", "..", "..", "..", "docs", "rigspec.md")

	got, err := os.ReadFile(at) //nolint:gosec // a path this repository owns
	s.Require().NoError(err)

	s.Require().Equal(string(want), string(got),
		"docs/rigspec.md is out of date — run `just generate`")
}

// TestThePageNamesItsOwnRoot holds the page to the contract it describes.
func (s *PagePublicTestSuite) TestThePageNamesItsOwnRoot() {
	s.Require().Equal("RigSpec", page.Page.Root)
	s.Require().NotEmpty(page.Page.Title)
	s.Require().NotEmpty(page.Page.Sits)
	s.Require().NotEmpty(page.Page.Checked)
	s.Require().NotEmpty(page.Page.Other.At,
		"each page points at the other half of the pipeline")
}

func TestPagePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PagePublicTestSuite))
}
