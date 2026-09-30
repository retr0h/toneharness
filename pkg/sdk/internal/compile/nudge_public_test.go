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

// NudgePublicTestSuite covers turning a word somebody said into the figures to
// aim differently at.
//
// Against the vocabulary and the move table this repository ships rather than
// fixtures, because the answer has to be right about the real words: a nudge
// resolved against a made-up table would pass here and move the wrong knob.
type NudgePublicTestSuite struct {
	suite.Suite
}

// TestNudges covers Nudges, which is what a word asks of the measured
// figures.
//
// One method and one table, so a case is a row rather than a file.
func (s *NudgePublicTestSuite) TestNudges() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "a word names a figure and a direction",
			then: func() {
				tests := []struct {
					name string
					said string
					key  string
					up   bool
				}{
					{"the adjective", "dark", "centroid", false},
					{"and its opposite", "bright", "centroid", true},
					{"mids pushed", "mid-forward", "mid", true},
					{"mids pulled", "scooped", "mid", false},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, err := compile.Nudges(tt.said)

						s.Require().NoError(err)
						s.Require().Len(got, 1)
						s.Require().Equal(tt.key, got[0].Key)
						s.Require().Equal(tt.up, got[0].Up)
					})
				}
			},
		},
		{
			// mean the same axis in the same direction, so refusing "darker" would make
			// somebody look up a spelling to say a thing the tool understands.
			name: "the comparative is what somebody actually says",
			then: func() {
				tests := []struct {
					said string
					key  string
					up   bool
				}{
					{"darker", "centroid", false},
					{"brighter", "centroid", true},
				}

				// A word ending in y takes the other form: "punchier" is "punchy", which
				// answers two axes, so it is checked on its own below rather than here.
				got, err := compile.Nudges("punchier")
				s.Require().NoError(err)
				s.Require().Len(got, 2)

				for _, tt := range tests {
					s.Run(tt.said, func() {
						got, err := compile.Nudges(tt.said)

						s.Require().NoError(err)
						s.Require().Len(got, 1)
						s.Require().Equal(tt.key, got[0].Key)
						s.Require().Equal(tt.up, got[0].Up)
					})
				}
			},
		},
		{
			// vocabulary holds it that way because that is how players talk.
			name: "a word may ask for two things",
			then: func() {
				got, err := compile.Nudges("punchy")

				s.Require().NoError(err)
				s.Require().Len(got, 2)

				keys := []string{got[0].Key, got[1].Key}
				s.Require().Contains(keys, "low")
				s.Require().Contains(keys, "transient")

				// Sorted on the figure, so the same word reports the same order every run.
				s.Require().Equal("low", got[0].Key)
			},
		},
		{
			// pickup was used and whether a filter is moving have no figure at all. A word
			// answering only those cannot move a target however much somebody means it, and
			// saying so is the difference between that and a word that does nothing.
			name: "a word nothing measures is refused",
			then: func() {
				_, err := compile.Nudges("quiet-strings")

				s.Require().ErrorContains(err, "nothing measures")
			},
		},
		{
			name: "a word the vocabulary does not know is refused by name",
			then: func() {
				_, err := compile.Nudges("chunky")

				s.Require().ErrorContains(err, "chunky")
				s.Require().ErrorContains(err, "words.json")
			},
		},
		{
			// something with no figure, and neither answer is a crash or an empty list.
			name: "every word either resolves or says why not",
			then: func() {
				for _, word := range compile.Words() {
					s.Run(word, func() {
						got, err := compile.Nudges(word)
						if err != nil {
							s.Require().ErrorContains(err, "nothing measures",
								"a word this vocabulary holds is refused for one reason only")

							return
						}

						s.Require().NotEmpty(got)

						for _, n := range got {
							s.Require().NotEmpty(n.Key)
							s.Require().NotEmpty(n.Axis)
							s.Require().Equal(word, n.Term)
						}
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestTheComparativeIsWhatSomebodyActuallySays covers the form of the word.
//

// TestAWordMayAskForTwoThings covers a word answering more than one axis.
//

// TestAWordNothingMeasuresIsRefused covers four of the ten axes.
//

// TestEveryWordEitherResolvesOrSaysWhyNot holds the two tables to each other.
//

func TestNudgePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NudgePublicTestSuite))
}
