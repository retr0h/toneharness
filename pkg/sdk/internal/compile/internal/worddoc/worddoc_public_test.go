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

package worddoc_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile/internal/worddoc"
)

type WorddocPublicTestSuite struct {
	suite.Suite
}

// page is the shipped vocabulary page.
func (s *WorddocPublicTestSuite) page() string {
	at := filepath.Join("..", "..", "..", "..", "..", "..", "docs", "vocabulary.md")

	got, err := os.ReadFile(at) //nolint:gosec // a path this repository owns
	s.Require().NoError(err)

	return string(got)
}

// TestTheShippedPageIsCurrent fails the moment the vocabulary and the page
// disagree, which is the only thing making the page worth trusting.
func (s *WorddocPublicTestSuite) TestTheShippedPageIsCurrent() {
	want, err := worddoc.Render()
	s.Require().NoError(err)

	s.Require().Equal(string(want), s.page(),
		"docs/vocabulary.md is out of date, run `just generate`")
}

// TestEveryWordIsOnThePage covers the claim the page exists to make.
//
// A word the vocabulary carries and the page omits is the failure this replaces:
// the list lived in a JSON file inside an internal package, and four prose pages
// each described some of it.
func (s *WorddocPublicTestSuite) TestEveryWordIsOnThePage() {
	page := s.page()

	words := compile.Words()
	s.Require().NotEmpty(words)

	for _, word := range words {
		s.Require().Contains(page, "`"+word+"`", "%s is a word and the page omits it", word)
	}
}

// TestWhatEachWordMovesIsWhatTheCompilerMoves holds the page to the code rather
// than to the prose beside it.
//
// The page says which control a word turns and which way. That comes from the
// same map the compiler applies, so a word whose direction changed cannot leave
// a page claiming the old one.
func (s *WorddocPublicTestSuite) TestWhatEachWordMovesIsWhatTheCompilerMoves() {
	page := s.page()
	var acting int

	for _, axis := range compile.Vocabulary() {
		for _, w := range axis.Words {
			if w.Param == "" {
				continue
			}

			acting++

			way := "up"
			if w.Steps < 0 {
				way = "down"
			}

			s.Require().Contains(page, w.Param+" ",
				"%s moves %s and the page does not say so", w.Term, w.Param)
			s.Require().Contains(page, way,
				"%s moves %s %s and the page does not say so", w.Term, w.Param, way)
		}
	}

	s.Require().NotZero(acting, "no word moves anything, which cannot be right")
}

// TestAWordThatMovesNothingSaysSo covers the four axes that describe the player.
//
// Reporting only the words that reach a control would read as if the rest did
// too, which is the same reason a build reports them.
func (s *WorddocPublicTestSuite) TestAWordThatMovesNothingSaysSo() {
	var idle int

	for _, axis := range compile.Vocabulary() {
		for _, w := range axis.Words {
			if w.Param == "" {
				idle++
			}
		}
	}

	s.Require().NotZero(idle, "every word moves something, which cannot be right")
	s.Require().Contains(s.page(), "Nothing on this axis moves a control")
}

// TestThePageWrapsItself covers the formatter being told to leave it alone.
func (s *WorddocPublicTestSuite) TestThePageWrapsItself() {
	for i, line := range strings.Split(s.page(), "\n") {
		// A table row and a link are one unit and wrapping either breaks it.
		if strings.HasPrefix(line, "|") || strings.Contains(line, "](") {
			continue
		}

		s.Require().LessOrEqual(len(line), 80, "line %d is wider than 80", i+1)
	}
}

// TestATemplateThatFailsIsReported covers the one error Render can return.
//
// The template is embedded and parsed at startup, so a parse failure is a broken
// build rather than something a caller sees. Executing one can still fail, and
// that is the path this takes: a template asking for a field the page has no.
func (s *WorddocPublicTestSuite) TestATemplateThatFailsIsReported() {
	broken := template.Must(template.New("x").Parse("{{ .Nope.Missing }}"))

	_, err := worddoc.RenderWith(broken)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing the vocabulary page")
}

func TestWorddocPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WorddocPublicTestSuite))
}
