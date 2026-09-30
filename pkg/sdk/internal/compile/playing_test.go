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
package compile

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// PlayingTestSuite covers compensating for how somebody plays.
//
// Internal, because what is worth checking is which word comes out and that is
// this package's own decision. What the word then does to a control is move's
// subject, and the whole path through a build is presets'.
type PlayingTestSuite struct {
	suite.Suite
}

// TestCompensate covers compensate, which turns the gap between how the
// subject played and how this person plays into a word.
func (s *PlayingTestSuite) TestCompensate() {
	for _, tt := range []struct {
		name string
		// subject is how the rig was played, and playing how this person does.
		subject string
		playing string
		// spoken is a word the ask already uses for the attack axis.
		spoken string
		// term is the word expected, empty for none.
		term string
		// says is a fragment the sentence has to carry. Empty asks for no
		// sentence at all.
		says string
	}{
		{
			// The case the feature exists for. A rig tuned from a picked
			// recording is duller played with fingers, and the front of the
			// note is the half that can be compensated.
			name:    "a picked rig played with fingers asks for more attack",
			subject: "pick", playing: "fingers",
			term: nearer,
			says: "compensates for the front of the note",
		},
		{
			name:    "a fingered rig played with a pick asks for less",
			subject: "fingers", playing: "pick",
			term: softer,
			says: "so soft-attack",
		},
		{
			// Two ranks rather than one, and still one word. Reaching for
			// percussive to mean a wide gap would have worked arithmetically
			// and lied: percussive is the string against the fretboard.
			name:    "a slapped rig played with fingers asks for one word, not a louder one",
			subject: "slap", playing: "fingers",
			term: nearer,
		},
		{
			name:    "the same right hand both sides compensates nothing",
			subject: "pick", playing: "pick",
		},
		{
			// A thumb is flesh like fingers, and the rank is about the onset
			// rather than the tone.
			name:    "a thumb and fingers rank together",
			subject: "fingers", playing: "thumb",
		},
		{
			name:    "hybrid sits between them",
			subject: "hybrid", playing: "fingers",
			term: nearer,
		},
		{
			// Said rather than done. Two terms on one axis cancel by design,
			// so appending here would take the ask's own word out with it.
			name:    "a word the ask already uses for the axis stands, and this stands down",
			subject: "pick", playing: "fingers",
			spoken: "audible-pick-attack",
			says:   "so that stands rather than a second word on the same axis",
		},
		{
			name:    "a subject that did not say compensates nothing",
			playing: "fingers",
		},
		{
			name:    "a Setup that did not say compensates nothing",
			subject: "pick",
		},
		{
			// A technique this does not rank is one it cannot compare, and
			// inventing a rank would be worse than saying nothing.
			name:    "an attack nothing ranks compensates nothing",
			subject: "pick", playing: "sings it",
		},
	} {
		s.Run(tt.name, func() {
			got := compensate(tt.subject, Playing{Attack: tt.playing}, tt.spoken)

			s.Require().Equal(tt.term, got.Word.Term)

			if tt.says == "" && tt.term == "" {
				s.Require().Empty(got.Said,
					"nothing was compensated, so there is nothing to say")

				return
			}

			s.Require().Contains(got.Said, tt.says)

			if tt.subject != "" && tt.playing != "" {
				s.Require().Contains(got.Said, tt.playing,
					"the sentence names how this person plays")
			}
		})
	}
}

// TestSpokenFor covers spokenFor, which is the word an ask already uses for an
// axis, if any.
func (s *PlayingTestSuite) TestSpokenFor() {
	for _, tt := range []struct {
		name  string
		words []Word
		axis  string
		want  string
	}{
		{
			name:  "a word on that axis",
			words: []Word{{Term: "percussive"}},
			axis:  attackAxis,
			want:  "percussive",
		},
		{
			name:  "a word on another axis",
			words: []Word{{Term: "scooped"}},
			axis:  attackAxis,
		},
		{
			// punchy answers two questions, and the attack half is one of
			// them, so it speaks for this axis as much as percussive does.
			name:  "a word answering two axes speaks for both",
			words: []Word{{Term: "punchy"}},
			axis:  attackAxis,
			want:  "punchy",
		},
		{
			// Sorted, so an ask listing them the other way round gets the
			// same answer.
			name:  "two words on one axis answer in a fixed order",
			words: []Word{{Term: "percussive"}, {Term: "audible-pick-attack"}},
			axis:  attackAxis,
			want:  "audible-pick-attack",
		},
		{
			name:  "a word the vocabulary does not carry speaks for nothing",
			words: []Word{{Term: "sproingy"}},
			axis:  attackAxis,
		},
		{
			name: "no words at all",
			axis: attackAxis,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, spokenFor(tt.words, tt.axis))
		})
	}
}

// TestResolveTakesTheCompensationThroughABuild covers the word reaching a
// chain, which is the whole point of it being a word rather than a move.
//
// Here rather than in the table above, because what this asserts is that
// everything downstream treats it as an ask: the term arrives in the moves
// beside the ones somebody wrote.
func (s *PlayingTestSuite) TestResolveTakesTheCompensationThroughABuild() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	spec := rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "played-with-fingers",
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			{Role: rig.RoleComp, Gear: "LA Studio Comp"},
		},
	}

	_, _, moved, held, err := Resolve(spec, Intent{
		Attack:  "pick",
		Playing: Playing{Attack: "fingers"},
	}, cat, nil)

	s.Require().NoError(err)
	s.Require().Equal(nearer, held.Term)
	s.Require().Contains(held.Said, "you play with fingers")

	terms := make([]string, 0, len(moved))
	for _, m := range moved {
		terms = append(terms, m.Term)
	}

	s.Require().Contains(terms, nearer,
		"the compensating word is an ask like any other by the time it is moved")
}

func TestPlayingTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlayingTestSuite))
}
