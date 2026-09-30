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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
)

// WordsPublicTestSuite covers the words an ask may use for how it sounds.
type WordsPublicTestSuite struct {
	suite.Suite
}

// TestWords covers the shipped vocabulary.
func (s *WordsPublicTestSuite) TestWords() {
	got := compile.Words()

	s.Require().NotEmpty(got)
	s.Require().Contains(got, "mid-forward")
	s.Require().Contains(got, "audible-pick-attack")

	// Sorted, so a suggestion reads the same way twice.
	for i := 1; i < len(got); i++ {
		s.Require().Less(got[i-1], got[i])
	}
}

// TestCheckWords covers reporting the words nothing defines.
func (s *WordsPublicTestSuite) TestCheckWords() {
	tests := []struct {
		name string
		in   []string
		want map[string][]string
	}{
		{name: "a rig saying nothing about how it sounds"},
		{
			name: "every word in the vocabulary",
			in:   []string{"mid-forward", "short-decay"},
		},
		{
			// The spelling every rig in this repository used before there
			// was a vocabulary, which is the case the suggestion is for.
			name: "a sentence where a term belongs",
			in:   []string{"pick attack audible"},
			want: map[string][]string{
				"pick attack audible": {"audible-pick-attack"},
			},
		},
		{
			// Two claims in one line, and the closest single term is all
			// that is offered. Naming both would read as alternatives when
			// what the writer wants is to say two things.
			name: "two claims welded into one line",
			in:   []string{"tight low end, short decay"},
			want: map[string][]string{
				"tight low end, short decay": {"tight-low-end"},
			},
		},
		{
			name: "a word sharing nothing with any of them",
			in:   []string{"zzz"},
			want: map[string][]string{"zzz": nil},
		},
		{
			// A term made of punctuation has no words to compare, so there
			// is nothing to be close to.
			name: "a term that is not a word at all",
			in:   []string{"---"},
			want: map[string][]string{"---": nil},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := compile.CheckWords(tt.in)

			s.Require().Len(got, len(tt.want))

			for _, u := range got {
				near, ok := tt.want[u.Term]
				s.Require().True(ok, "unexpected term %q", u.Term)
				s.Require().Subset(u.Near, near)
			}
		})
	}
}

// TestCheckAxes covers an ask answering one question twice.
func (s *WordsPublicTestSuite) TestCheckAxes() {
	tests := []struct {
		name  string
		terms []string
		want  []compile.ContestedAxis
	}{
		{
			name:  "a direction and its opposite",
			terms: []string{"mid-forward", "scooped"},
			want: []compile.ContestedAxis{
				{Axis: "mids", Terms: []string{"mid-forward", "scooped"}},
			},
		},
		{
			// Not opposites. Two adjacent points on one scale still answer
			// the same question, and still cancel.
			name:  "two points on one scale",
			terms: []string{"minimal-drive", "grit-on-attack"},
			want: []compile.ContestedAxis{
				{Axis: "drive", Terms: []string{"minimal-drive", "grit-on-attack"}},
			},
		},
		{
			name:  "one term per axis",
			terms: []string{"mid-forward", "saturated", "roomy"},
		},
		{
			// Reported elsewhere as an unknown term. Nothing knows which
			// axis it belongs to, so it contests nothing.
			name:  "a word the vocabulary does not carry",
			terms: []string{"sounds like a wet paper bag", "mid-forward"},
		},
		{
			name: "an ask that describes nothing",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, compile.CheckAxes(tt.terms))
		})
	}
}

// TestVocabularyPairsEveryWordWithWhatItMoves covers the shape a reference page
// needs, which Words alone cannot answer.
//
// Words says whether a word exists. This says what it does, and the two have to
// agree: a word in one and not the other is a word the page would omit or invent.
func (s *WordsPublicTestSuite) TestVocabularyPairsEveryWordWithWhatItMoves() {
	axes := compile.Vocabulary()
	s.Require().NotEmpty(axes)

	// How many axes each word answers, because a word answering two is the
	// point of this rather than a mistake.
	seen := map[string]int{}

	var acting, idle int

	for _, axis := range axes {
		s.Require().NotEmpty(axis.Name)
		s.Require().NotEmpty(axis.About, "%s says nothing about itself", axis.Name)
		s.Require().NotEmpty(axis.Words, "%s carries no words", axis.Name)

		for _, w := range axis.Words {
			s.Require().NotEmpty(w.Term)
			s.Require().NotEmpty(w.Means, "%s means nothing", w.Term)
			seen[w.Term]++

			// A word either names a control and a block or names neither. One
			// without the other would move something nowhere.
			if w.Param == "" {
				s.Require().Empty(w.Block)
				s.Require().Zero(w.Steps)

				idle++

				continue
			}

			s.Require().NotEmpty(w.Block, "%s moves %s on nothing", w.Term, w.Param)
			s.Require().NotZero(w.Steps, "%s moves %s by nothing", w.Term, w.Param)

			acting++
		}
	}

	// The two lists are the same list.
	for _, word := range compile.Words() {
		s.Require().Positive(seen[word], "%s is a word and Vocabulary omits it", word)
	}

	s.Require().Len(seen, len(compile.Words()))

	// A word on more than one axis carries a move on every one of them, which
	// the loop above already required of each entry. What is asserted here is
	// that at least one such word exists: the machinery for a compound word is
	// worth nothing if nothing uses it, and a vocabulary that quietly went back
	// to one axis per word would otherwise pass.
	compound := 0

	for term, axes := range seen {
		if axes > 1 {
			compound++

			s.Require().Contains(compile.Words(), term)
		}
	}

	s.Require().Positive(compound,
		"no word answers more than one axis, so nothing exercises that they can")

	// Six axes act and four describe, so both kinds must be represented or the
	// checks above passed over half the vocabulary.
	s.Require().NotZero(acting)
	s.Require().NotZero(idle)
}

func TestWordsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WordsPublicTestSuite))
}
