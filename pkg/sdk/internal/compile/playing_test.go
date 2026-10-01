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

	"github.com/retr0h/toneharness/pkg/sdk/audio"
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
// attackTermIn and midsTermIn are the compensating word on each axis, so a case
// says which axis it expects rather than relying on the order they were added.
func attackTermIn(
	words []Word,
) string {
	return termOn(words, attackAxis)
}

func midsTermIn(
	words []Word,
) string {
	return termOn(words, midsAxis)
}

func termOn(
	words []Word,
	axis string,
) string {
	for _, w := range words {
		axes, ok := axesOf(w.Term)
		if !ok {
			continue
		}

		for _, a := range axes {
			if a == axis {
				return w.Term
			}
		}
	}

	return ""
}

// TestMidsWord covers which way a measured difference points, and a difference
// that is not one.
//
// Read on its own because the real figure cannot produce every case: a pick
// moves the mids in all 468 pairs, so nothing in the committed file measures
// zero, and a branch no data reaches is still a branch.
func (s *PlayingTestSuite) TestMidsWord() {
	for _, tt := range []struct {
		name string
		mean float64
		want string
	}{
		{
			// My hand puts less in the mids than the rig was voiced for, so
			// they go up. Fingers against a rig built from a plectrum.
			name: "a hand of mine with less mid asks for more",
			mean: -0.17,
			want: "mid-forward",
		},
		{
			name: "a hand of mine with more mid asks for less",
			mean: 0.17,
			want: "scooped",
		},
		{
			// A figure nobody measured and a figure that measured zero are the
			// same answer: there is nothing to compensate either way.
			name: "no difference compensates nothing",
		},
	} {
		s.Run(tt.name, func() {
			got, ok := midsWord(audio.Moved{Mean: tt.mean})

			s.Require().Equal(tt.want != "", ok)
			s.Require().Equal(tt.want, got)
		})
	}
}

func (s *PlayingTestSuite) TestCompensate() {
	for _, tt := range []struct {
		name string
		// subject is how the rig was played, and playing how this person does.
		subject string
		playing string
		// spoken is a word the ask already uses for the attack axis, and
		// spokenMids one it uses for the mids. Separate, because each axis
		// stands down on its own.
		spoken     string
		spokenMids string
		// term is the word expected for the front of the note, and mids the one
		// for the mids. Empty for none.
		term string
		mids string
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
			// And the mids, which is the measured half. Fingers put 0.17 less
			// of the energy there than the pick the rig was voiced for.
			mids: "mid-forward",
			says: "compensates for the front of the note",
		},
		{
			name:    "a fingered rig played with a pick asks for less",
			subject: "fingers", playing: "pick",
			term: softer,
			// The other way round, so the other word. A pick puts more in the
			// mids than the fingers the rig was voiced for.
			mids: "scooped",
			says: "soft-attack",
		},
		{
			// Two ranks rather than one, and still one word. Reaching for
			// percussive to mean a wide gap would have worked arithmetically
			// and lied: percussive is the string against the fretboard.
			name:    "a slapped rig played with fingers asks for one word, not a louder one",
			subject: "slap", playing: "fingers",
			term: nearer,
			// Nothing for the mids. Nobody has measured slap against fingers,
			// and the attack axis is ranked rather than measured, so one axis
			// answers and the other says nothing.
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
			mids:   "mid-forward",
			says:   "a second word on that axis would cancel it",
		},
		{
			// The same, on the measured axis. The ask owns the mids and this
			// leaves them alone, while the front of the note is still
			// compensated: the two axes stand down independently.
			name:    "a word the ask uses for the mids stands, and the attack is still compensated",
			subject: "pick", playing: "fingers",
			spokenMids: "scooped",
			term:       nearer,
			says:       "the mids are left to scooped",
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
			got := compensate(
				tt.subject, Playing{Attack: tt.playing}, tt.spoken, tt.spokenMids)

			s.Require().Equal(tt.term, attackTermIn(got.Words),
				"the word for the front of the note")
			s.Require().Equal(tt.mids, midsTermIn(got.Words),
				"the word for the mids")

			if tt.says == "" && tt.term == "" && tt.mids == "" {
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
	s.Require().Equal([]string{nearer, "mid-forward"}, held.Terms)
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
