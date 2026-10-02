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

package audio_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// HandsShippedPublicTestSuite covers the measurements that travel in the binary.
//
// The committed figure rather than the audio behind it, which is the split this
// generator exists for: the notes are somebody else's and are not in CI, and
// what has to keep working is the number anything downstream reads.
type HandsShippedPublicTestSuite struct {
	suite.Suite
}

// TestTheMeasuredHandsLoad covers the embedded file parsing and saying
// something.
func (s *HandsShippedPublicTestSuite) TestTheMeasuredHandsLoad() {
	all, err := audio.MeasuredHands()
	s.Require().NoError(err)
	s.Require().NotEmpty(all, "a hand is measured and ships")

	for _, got := range all {
		s.Run(got.From+" against "+got.To, func() {
			s.Require().NotEmpty(got.From)
			s.Require().NotEmpty(got.To)
			s.Require().NotEqual(got.From, got.To, "a hand against itself says nothing")
			s.Require().Positive(got.Pairs)
			s.Require().LessOrEqual(got.Agreed, got.Pairs,
				"more pairs agreed than were measured")
			s.Require().NotEmpty(got.Figures)
		})
	}
}

// TestUnpackHands covers reading the measured hands, and a file somebody edited.
//
// One method and one table, so a case is a row rather than a file.
func (s *HandsShippedPublicTestSuite) TestUnpackHands() {
	for _, tt := range []struct {
		name string
		body string
		// pairs is how many comparisons the body holds, where it parses.
		pairs int
		// err is a fragment the message must carry. It names the measured hands
		// rather than a line of JSON, because a generated file somebody has been
		// into by hand is the case where knowing which file matters.
		err string
	}{
		{
			name: "what the generator writes",
			body: `[{"from":"fingers","to":"pick","pairs":468,` +
				`"figures":{"mid":{"mean":0.17}}}]`,
			pairs: 1,
		},
		{
			// An empty list is a corpus holding no pair of hands anybody has
			// measured, which is what a checkout with no audio generates.
			name: "nothing measured yet",
			body: "[]",
		},
		{
			name: "a file somebody edited",
			body: "not json",
			err:  "reading the measured hands",
		},
	} {
		s.Run(tt.name, func() {
			got, err := audio.UnpackHands([]byte(tt.body))

			if tt.err != "" {
				s.Require().ErrorContains(err, tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got, tt.pairs)
		})
	}
}

// TestHandsBetween covers the lookup, in both directions.
//
// One method and one table, so a case is a row rather than a file.
func (s *HandsShippedPublicTestSuite) TestHandsBetween() {
	for _, tt := range []struct {
		name string
		// from and to are the hands asked for.
		from, to string
		// found says whether anybody has measured them.
		found bool
		// mids is the sign expected of the mid band's mean, where there is one.
		// A pick puts more energy in the mids than fingers do, so the figure is
		// positive in that direction and negative in the other.
		mids float64
	}{
		{
			// The direction the file stores.
			name: "the way it was measured",
			from: "fingers", to: "pick",
			found: true,
			mids:  1,
		},
		{
			// And the other way, which is that one negated rather than a thing
			// nobody measured. A caller should not have to know which way round
			// the file happens to hold it.
			name: "the other way round",
			from: "pick", to: "fingers",
			found: true,
			mids:  -1,
		},
		{
			// Nobody has held a thumb against a plectrum. Absent rather than
			// guessed: a rank would be invented and the whole point of the
			// figure is that it was measured.
			name: "two hands nobody has measured",
			from: "thumb", to: "pick",
		},
		{
			name: "a hand against itself",
			from: "pick", to: "pick",
		},
		{
			name: "a hand nothing names",
			from: "fingers", to: "",
		},
	} {
		s.Run(tt.name, func() {
			got, ok := audio.HandsBetween(tt.from, tt.to)

			s.Require().Equal(tt.found, ok)

			if !tt.found {
				return
			}

			s.Require().Equal(tt.from, got.From)
			s.Require().Equal(tt.to, got.To)

			mid := got.Figures[audio.KeyMid]

			if tt.mids > 0 {
				s.Require().Positive(mid.Mean)
			} else {
				s.Require().Negative(mid.Mean)
			}

			s.Require().Positive(mid.Spread,
				"a spread is reported beside the mean, not folded into it")

			// Reading it reversed must not change what it holds. The negation is
			// a view over the measurement rather than an edit to it, so the pairs
			// are the same pairs and a spread has no direction to reverse.
			back, ok := audio.HandsBetween(tt.to, tt.from)
			s.Require().True(ok)

			again, ok := audio.HandsBetween(tt.from, tt.to)
			s.Require().True(ok)

			s.Require().Equal(got, again)
			s.Require().Equal(got.Pairs, back.Pairs)
			s.Require().Equal(mid.Spread, back.Figures[audio.KeyMid].Spread)
		})
	}
}

func TestHandsShippedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HandsShippedPublicTestSuite))
}
