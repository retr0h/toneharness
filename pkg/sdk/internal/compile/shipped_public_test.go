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

package compile_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/shipped"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// ShippedPublicTestSuite holds the asks this repository ships to the
// vocabulary, which nothing holds anybody else's ask to.
//
// An ask somebody writes gets a note naming the terms nothing defines and
// still builds, because a word nothing defines moves no knob and refusing a
// preset over one would be refusing them the right to describe a sound. The
// asks here are different: they are the examples everybody copies, and every
// one of them said what it meant in a sentence until there was a list.
//
// The asks rather than the rigs, because the words left the rig: how it should
// sound is what somebody wanted, and a rig says only which gear answered.
type ShippedPublicTestSuite struct {
	suite.Suite
}

// words reads what an ask says it should sound like.
func (s *ShippedPublicTestSuite) words(
	open func() (fs.File, error),
) []string {
	f, err := open()
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	spec, err := tone.Load(f)
	s.Require().NoError(err)

	if spec.Words == nil {
		return nil
	}

	// The terms alone. What the vocabulary can say about a word does not
	// depend on why it is believed.
	out := make([]string, 0, len(*spec.Words))
	for _, w := range *spec.Words {
		out = append(out, w.Term)
	}

	return out
}

// TestEveryShippedAskUsesTheVocabulary covers the terms every shipped ask
// describes a sound with.
//
// Read through the embedded copy rather than off disk, so this counts no
// directories and travels wherever the package does.
func (s *ShippedPublicTestSuite) TestEveryShippedAskUsesTheVocabulary() {
	paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.tone.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no asks found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			for _, u := range compile.CheckWords(
				s.words(func() (fs.File, error) { return shipped.FS.Open(path) }),
			) {
				s.Require().Fail("no such character term",
					"%q. Add it to pkg/sdk/compile/data/words.json "+
						"with a definition, or use one of: %v", u.Term, u.Near)
			}
		})
	}
}

// TestEveryShippedAskAnswersEachAxisOnce covers an ask arguing with itself.
//
// An ask claiming two terms from one axis has claimed nothing: the two cancel,
// the control stays where the corpus left it, and a build says so on every
// run. mike-dirnt shipped claiming both minimal-drive and grit-on-attack, and
// nothing caught it until the words started moving knobs.
func (s *ShippedPublicTestSuite) TestEveryShippedAskAnswersEachAxisOnce() {
	paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.tone.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no asks found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			for _, c := range compile.CheckAxes(
				s.words(func() (fs.File, error) { return shipped.FS.Open(path) }),
			) {
				s.Require().Fail("one axis answered twice",
					"%q: %v. Keep the term that says the most and drop the "+
						"rest, or neither will be applied.", c.Axis, c.Terms)
			}
		})
	}
}

// TestEveryExampleUsesTheVocabulary covers the asks the docs point at.
func (s *ShippedPublicTestSuite) TestEveryExampleUsesTheVocabulary() {
	paths, err := filepath.Glob(
		filepath.Join("..", "..", "..", "..", "marketplace", "core", "examples", "tonespec", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no examples found to check")

	for _, path := range paths {
		// The directory holds a setup beside the asks, which says what
		// somebody owns rather than how it should sound and has no words in
		// it at all.
		body, err := os.ReadFile(path) //nolint:gosec // a path this test globbed
		s.Require().NoError(err)

		if bytes.Contains(body, []byte("schema: "+tone.SetupSchema)) {
			continue
		}

		s.Run(filepath.Base(path), func() {
			words := s.words(func() (fs.File, error) { return os.Open(path) })

			s.Require().Empty(compile.CheckWords(words))
			s.Require().Empty(compile.CheckAxes(words))
		})
	}
}

func TestShippedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShippedPublicTestSuite))
}
