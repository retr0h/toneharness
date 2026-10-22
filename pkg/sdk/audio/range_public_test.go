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

// GenreRangeTestSuite holds the shipped genres to publishing a range, not just
// a middle.
type GenreRangeTestSuite struct {
	suite.Suite
}

// TestEveryAimableGenrePublishesARange holds each one to carrying the spread
// its records cover, on every figure a genre rig is built from.
//
// build-a-rig's measuring.md says to aim a genre rig at the edge of the
// genre's own range rather than at its middle, because the middle is what
// every genre shares: ten of the eleven aimable genres displace between 0.2
// and 6.4 points from one baseline, against an amplifier field 50 points wide,
// and 9 of the 11 shipped genre rigs carry the same amplifier and cabinet as a
// result.
//
// That rule can only be followed while the spread is published. `Across` holds
// a tenth percentile, a median and a ninetieth for each figure, and a pipeline
// that kept only the median would leave the rule describing data that no
// longer exists, with nothing failing to say so. Hence this.
//
// Off the embedded pack rather than off a corpus, so it needs no audio: the
// stems are not in this repository and the measurements of them are what
// ships. See [audio.Shipped].
func (s *GenreRangeTestSuite) TestEveryAimableGenrePublishesARange() {
	all, err := audio.Shipped()
	s.Require().NoError(err)
	s.Require().NotEmpty(all, "the embedded pack carries no genres at all")

	aimable := 0

	for _, g := range all {
		if !g.Usable {
			continue
		}

		aimable++

		// Every figure the rule reads, by the name its own documentation uses.
		for _, figure := range []struct {
			name           string
			low, mid, high float64
		}{
			{"low", g.Across.Low.Low, g.Across.Low.Mid, g.Across.Low.High},
			{"mid", g.Across.Mid.Low, g.Across.Mid.Mid, g.Across.Mid.High},
			{"centroid", g.Across.Centroid.Low, g.Across.Centroid.Mid, g.Across.Centroid.High},
			{"harmonics", g.Across.Harmonics.Low, g.Across.Harmonics.Mid, g.Across.Harmonics.High},
			{"dynamic_range", g.Across.DynamicRange.Low, g.Across.DynamicRange.Mid, g.Across.DynamicRange.High},
		} {
			where := g.Slug + "." + figure.name

			// Ordered, because a rule that aims at the edge needs to know
			// which end is which.
			s.Require().LessOrEqual(figure.low, figure.mid,
				"%s: the tenth percentile sits above the median", where)
			s.Require().LessOrEqual(figure.mid, figure.high,
				"%s: the median sits above the ninetieth percentile", where)

			// And wide enough to have an edge to aim at. A figure whose tenth
			// and ninetieth percentiles are equal offers nothing but its
			// middle, which is the thing the rule exists to stop aiming at.
			s.Require().Greater(figure.high, figure.low,
				"%s: every record reads the same, so there is no edge to aim at", where)
		}
	}

	// A count rather than a floor of one, because the rule is about genres
	// differing from each other and a single genre cannot.
	s.Require().GreaterOrEqual(aimable, 2,
		"fewer than two aimable genres, so nothing can be aimed anywhere")
}

func TestGenreRangeTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(GenreRangeTestSuite))
}
