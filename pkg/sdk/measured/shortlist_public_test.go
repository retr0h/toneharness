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

package measured_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// ShortlistPublicTestSuite covers choosing which block a chain is missing.
//
// The solver moves the dials of the blocks in front of it. When a target wants
// more of something than anything in the chain produces, every dial is already
// at its limit and the answer is a block rather than a position. This is how
// that block gets found, out of six hundred and sixty one.
type ShortlistPublicTestSuite struct {
	suite.Suite
}

// reading builds one block's figures, with harmonics carried as the pointer
// it is.
func reading(
	id string,
	kind catalog.Category,
	centroid, harmonics float64,
) measured.Block {
	return measured.Block{
		ID: id, Name: id, Category: kind,
		Figures: measured.Figures{
			// A level a real reading has. Left at zero these fixtures were all
			// at full scale, which Measured excludes as the converters' own
			// shaping, so every one of them was silently dropped and the tests
			// passed on empty shortlists.
			Level:    -60,
			Centroid: centroid, Harmonics: &harmonics,
		},
	}
}

// library is an empty loop at 100Hz with no harmonics, and blocks against it.
func library(
	blocks ...measured.Block,
) measured.Library {
	none := 0.0

	by := make(map[string]measured.Block, len(blocks))
	for _, b := range blocks {
		by[b.ID] = b
	}

	return measured.Library{
		Device:   "HX Stomp",
		Isolated: true,
		Baseline: measured.Figures{Centroid: 100, Harmonics: &none},
		Blocks:   by,
	}
}

// TestItFindsTheBlockThatClosesTheGap is the whole point.
//
// TestShortlist covers Shortlist, which is the blocks whose own readings move
// the unmet axes the right way.
//
// One method and one table, so a case is a row rather than a file.
func (s *ShortlistPublicTestSuite) TestShortlist() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// block, and the fingerprints already say which blocks make harmonics. Before
			// this, the loop reported the axis it missed and stopped, which left somebody
			// reading a failure with nothing to do about it.
			name: "it finds the block that closes the gap",
			then: func() {
				lib := library(
					reading("Drive", catalog.Category("drive"), 100, 40),
					reading("Delay", catalog.Category("delay"), 100, 1),
					reading("Reverb", catalog.Category("reverb"), 100, 0),
				)

				got := measured.Shortlist(lib, []measured.Want{
					{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
				}, nil, 0)

				s.Require().Len(got, 3)
				s.Require().Equal("Drive", got[0].Block.ID,
					"the one whose own reading makes the harmonics asked for")
				s.Require().InDelta(0, got[0].Worst, 0.001, "it closes the gap exactly")
				s.Require().Contains(got[0].Helps, audio.KeyHarmonics)
				s.Require().Empty(got[0].Hurts)

				s.Require().Equal("Reverb", got[2].Block.ID, "and the one that makes none")
			},
		},
		{
			// overshoots by eight tolerances is not a better answer than one that lands. The
			// distance is what is ranked, not the direction.
			name: "an overshoot is a gap too",
			then: func() {
				lib := library(
					reading("Lands", catalog.Category("drive"), 100, 40),
					reading("Overshoots", catalog.Category("drive"), 100, 90),
				)

				got := measured.Shortlist(lib, []measured.Want{
					{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
				}, nil, 0)

				s.Require().Equal("Lands", got[0].Block.ID)
				s.Require().InDelta(10, got[1].Worst, 0.001, "50 over, at a tolerance of 5")
				s.Require().Contains(got[1].Helps, audio.KeyHarmonics,
					"it still moves the right way, which is worth reporting")
			},
		},
		{
			// is right on everything but the axis the target cares about, and then "2.4
			// tolerances out" would mean two different things in one run. The same measure
			// the list comparison ranks a cabinet's microphones by.
			name: "the worst axis decides rather than the sum",
			then: func() {
				lib := library(
					// Perfect on harmonics, useless on centroid.
					reading("Lopsided", catalog.Category("drive"), 100, 40),
					// Half right on both, so a sum would prefer it.
					reading("Middling", catalog.Category("drive"), 150, 20),
				)

				want := []measured.Want{
					{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
					{Figure: audio.KeyCentroid, Need: 100, Tol: 10},
				}

				got := measured.Shortlist(lib, want, nil, 0)

				s.Require().Equal("Middling", got[0].Block.ID,
					"5 tolerances out at worst beats 10")
				s.Require().InDelta(5, got[0].Worst, 0.001)
				s.Require().InDelta(10, got[1].Worst, 0.001)
			},
		},
		{
			// another is a real answer, and the loop measures the chain afterwards, which is
			// what settles it. What is not acceptable is the trade being invisible.
			name: "a block that hurts an axis says so",
			then: func() {
				lib := library(reading("Bright", catalog.Category("drive"), 4000, 40))

				got := measured.Shortlist(lib, []measured.Want{
					{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
					{Figure: audio.KeyCentroid, Need: -50, Tol: 10},
				}, nil, 0)

				s.Require().Len(got, 1)
				s.Require().Contains(got[0].Helps, audio.KeyHarmonics)
				s.Require().Contains(got[0].Hurts, audio.KeyCentroid,
					"it was asked to go down 50Hz and went up 3,900")
				s.Require().InDelta(395, got[0].Worst, 0.001)
			},
		},
		{
			// were never in the signal, so it reads as the brightest, most saturated block on
			// the device and is not one. That is the one that would win this ranking every
			// time if it were let in.
			name: "a reading that cannot describe its block is left out",
			then: func() {
				clipped := reading("Clipped", catalog.Category("drive"), 100, 99)
				clipped.Clipped = true

				refused := reading("Refused", catalog.Category("drive"), 100, 99)
				refused.Refused = "will not load alone"

				// The one the flag misses, and the reason this asks Measured rather than
				// the flags. A converter shapes a signal well before it hard-clips, so a
				// reading this loud is the converter's harmonics and not the block's. Two
				// of the readings it excludes on this device are bass amplifiers.
				loud := reading("NearTheCeiling", catalog.Category("drive"), 100, 99)
				loud.Level = -1

				lib := library(
					clipped,
					refused,
					loud,
					reading("InTheChain", catalog.Category("drive"), 100, 99),
					reading("Honest", catalog.Category("drive"), 100, 40),
				)

				got := measured.Shortlist(lib, []measured.Want{
					{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
				}, map[string]bool{"InTheChain": true}, 0)

				s.Require().Len(got, 1)
				s.Require().Equal("Honest", got[0].Block.ID)
			},
		},
		{
			// measured before that has a gap. Scored as zero it would rank against measured
			// blocks and win or lose on whichever direction zero happened to favour, which
			// is a confident answer about something nobody measured.
			name: "an unmeasured axis is not a zero",
			then: func() {
				old := measured.Block{
					ID: "Unmeasured", Name: "Unmeasured",
					Category: catalog.Category("drive"),
					Figures:  measured.Figures{Centroid: 100, Level: -60},
				}

				got := measured.Shortlist(library(old), []measured.Want{
					{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
				}, nil, 0)

				s.Require().Empty(got,
					"no reading of the axis asked about is no answer, not an answer of zero")
			},
		},
		{
			name: "nothing wanted is no shortlist",
			then: func() {
				s.Require().Empty(measured.Shortlist(
					library(reading("Drive", catalog.Category("drive"), 100, 40)), nil, nil, 0))
			},
		},
		{
			name: "the limit takes the best few",
			then: func() {
				lib := library(
					reading("A", catalog.Category("drive"), 100, 40),
					reading("B", catalog.Category("drive"), 100, 35),
					reading("C", catalog.Category("drive"), 100, 5),
				)

				got := measured.Shortlist(lib, []measured.Want{
					{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
				}, nil, 2)

				s.Require().Len(got, 2)
				s.Require().Equal("A", got[0].Block.ID)
				s.Require().Equal("B", got[1].Block.ID)
			},
		},
		{
			// between runs on identical data, which reads as the measurement moving.
			name: "ties come back in the same order",
			then: func() {
				lib := library(
					reading("Zed", catalog.Category("drive"), 100, 40),
					reading("Alpha", catalog.Category("drive"), 100, 40),
				)

				for range 8 {
					got := measured.Shortlist(lib, []measured.Want{
						{Figure: audio.KeyHarmonics, Need: 40, Tol: 5},
					}, nil, 0)

					s.Require().Equal("Alpha", got[0].Block.ID)
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestAnOvershootIsAGapToo covers a block that goes too far.
//

// TestTheWorstAxisDecidesRatherThanTheSum is the scoring choice.
//

// TestABlockThatHurtsAnAxisSaysSo covers the trade being visible.
//

// TestAReadingThatCannotDescribeItsBlockIsLeftOut covers the four exclusions.
//

// TestAnUnmeasuredAxisIsNotAZero is the pointer figures earning their keep.
//

// TestTiesComeBackInTheSameOrder covers ranging a map.
//

// TestCategoriesNarrowsToWhatIsWorthAdding covers the caller's own question.
//
// Which kinds are eligible is not this package's business. A chain missing
// harmonics wants a drive, and one already holding an amplifier does not want a
// second whatever its reading says.
func (s *ShortlistPublicTestSuite) TestCategoriesNarrowsToWhatIsWorthAdding() {
	of := []measured.Suggestion{
		{Block: reading("Amp", catalog.CategoryAmp, 100, 40)},
		{Block: reading("Drive", catalog.Category("drive"), 100, 40)},
	}

	got := measured.Categories(of, catalog.Category("drive"))
	s.Require().Len(got, 1)
	s.Require().Equal("Drive", got[0].Block.ID)

	s.Require().Len(measured.Categories(of), 2, "no kinds named narrows nothing")
}

func TestShortlistPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShortlistPublicTestSuite))
}
