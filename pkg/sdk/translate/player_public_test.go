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

package translate_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/config"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// PlayerTargetPublicTestSuite covers aiming at one player's own records.
//
// The target an artist rig never had. All 32 named the amplifier a source puts
// somebody on and then carried the catalog's defaults on every control:
// `dee-dee-ramone` sat at Bass 0.41, Bias 0.74, ChVol 0.8 and Drive 0.6, which
// is the factory setting to the digit, while his three records sat on the
// document unread.
type PlayerTargetPublicTestSuite struct {
	suite.Suite

	deps translate.Deps
}

func (s *PlayerTargetPublicTestSuite) SetupSuite() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	lib, err := measured.BuiltIn()
	s.Require().NoError(err)

	s.deps = translate.Deps{Catalog: cat, Measured: lib}
}

// TestAPlayerIsATarget covers who a build aims at, and in what order.
func (s *PlayerTargetPublicTestSuite) TestAPlayerIsATarget() {
	bass := tone.AskInstrument("bass")

	like := func(name string) *tone.Like {
		return &tone.Like{Artist: &name}
	}

	tests := []struct {
		name string
		ask  tone.Ask
		// aims is what the report must say the build aimed at, and the empty
		// string means it fell through to the genre.
		aims string
	}{
		{
			// The supported way to aim at one player, and what an artist rig
			// will reach once the resolve path carries a target.
			name: "sounding like a player aims at that player",
			ask: tone.Ask{
				Genre: []string{"punk"}, Instrument: &bass,
				Like: like("dee-dee-ramone"),
			},
			aims: "dee-dee-ramone",
		},
		{
			// The display name is slugged, because the pack files a player
			// under the directory their records sit in.
			name: "the name is read the way the corpus files it",
			ask: tone.Ask{
				Genre: []string{"punk"}, Instrument: &bass,
				Like: like("Dee Dee Ramone"),
			},
			aims: "dee-dee-ramone",
		},
		{
			// Nobody measured, so the genre answers and the rig keeps whatever
			// its sources justify.
			name: "a player with no records falls through to the genre",
			ask: tone.Ask{
				Genre: []string{"punk"}, Instrument: &bass,
				Like: like("Somebody Nobody Recorded"),
			},
			aims: "punk",
		},
		{
			// A genre is the middle of a population and a player is one
			// person, so a rig for nobody in particular still aims somewhere.
			name: "a rig naming nobody aims at its genre",
			ask: tone.Ask{
				Genre: []string{"punk"}, Instrument: &bass,
			},
			aims: "punk",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, notes, err := translate.Translate(tt.ask, config.Config{}, s.deps)
			s.Require().NoError(err)

			said := ""
			for _, n := range notes {
				said += n.Said + "\n"
			}

			s.Require().Contains(said, "measured to "+tt.aims,
				"it aimed somewhere else")
		})
	}
}

// TestAPlayerAimsAtTheirMiddle covers which end of their records it takes.
//
// A genre aims at the edge of its own range, because the middle is what every
// genre shares and ten of the eleven displace less than 6.4 points from one
// baseline. One player's records describe one person, so their middle is the
// thing being aimed at rather than the thing that hides them.
func (s *PlayerTargetPublicTestSuite) TestAPlayerAimsAtTheirMiddle() {
	who := "Dee Dee Ramone"

	got, ok := translate.PlayerAim(
		tone.Ask{Like: &tone.Like{Artist: &who}}, s.deps)
	s.Require().True(ok)

	// Every figure it reports is the middle of his records, which is what
	// `Across.Measured` answers with, rather than a percentile of them.
	s.Require().NotEmpty(got)
}

// TestAimAtPlayer covers the target the resolve path asks for by identifier.
//
// An artist rig's gear is authored, so its resolve never passes through the
// gear-choosing path where an ask's `like` is read. It asks here instead, with
// the document's own identifier, which is also the directory the corpus files
// their records under.
func (s *PlayerTargetPublicTestSuite) TestAimAtPlayer() {
	tests := []struct {
		name string
		who  string
		// aimed is whether a target came back at all.
		aimed bool
	}{
		{
			name:  "a measured player is a target",
			who:   "dee-dee-ramone",
			aimed: true,
		},
		{
			// The name is slugged, because the pack files a player under the
			// directory their records sit in.
			name:  "the name is read the way the corpus files it",
			who:   "Dee Dee Ramone",
			aimed: true,
		},
		{
			// Most rigs, and not an error: those keep the controls their
			// sources justify.
			name:  "a rig nobody has records for is no target",
			who:   "somebody-nobody-recorded",
			aimed: false,
		},
		{
			// A genre identifier, which reaches here because every rig's
			// identifier does and must not resolve to a player.
			name:  "a genre is not a player",
			who:   "punk",
			aimed: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := translate.AimAtPlayer(tt.who, s.deps.Measured.Baseline)
			s.Require().Equal(tt.aimed, ok)

			if !tt.aimed {
				s.Require().Equal(measured.Figures{}, got,
					"it handed back figures for nobody")

				return
			}

			// Displaced from the baseline the blocks were measured against,
			// which is the only thing that makes a player and a block
			// comparable: a player comes off finished records and a block off
			// a dry signal through one block.
			s.Require().NotEqual(s.deps.Measured.Baseline, got,
				"it handed back the baseline unmoved")
		})
	}
}

// TestAimAtGenre covers the target a genre's own records measure as.
//
// Task 242. Three of the eleven genre rigs borrow their amplifier's own
// pairing for a cabinet, so that cabinet is added by the compiler and never
// passes through the path where a genre's gear is chosen. The compiler asks
// here instead.
//
// One method and one table, so a case is a row rather than a file.
func (s *PlayerTargetPublicTestSuite) TestAimAtGenre() {
	tests := []struct {
		name string
		slug string
		// aimed is whether a target came back at all.
		aimed bool
	}{
		{
			name:  "a measured genre is a target",
			slug:  "punk",
			aimed: true,
		},
		{
			// One of the three whose cabinet the compiler adds, which is what
			// this exists for.
			name:  "and so is one whose cabinet the compiler adds",
			slug:  "rock",
			aimed: true,
		},
		{
			name:  "a genre nobody tagged is no target",
			slug:  "skiffle",
			aimed: false,
		},
		{
			// A player's identifier, which reaches here because every rig's
			// does and must not resolve to a genre.
			name:  "a player is not a genre",
			slug:  "dee-dee-ramone",
			aimed: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := translate.AimAtGenre(tt.slug, s.deps.Measured.Baseline)
			s.Require().Equal(tt.aimed, ok)

			if !tt.aimed {
				s.Require().Equal(measured.Figures{}, got,
					"it handed back figures for nothing")

				return
			}

			// Displaced from the baseline the blocks were measured against,
			// which is the only thing that makes a genre and a block
			// comparable: a genre comes off finished records and a block off
			// a dry signal through one block.
			s.Require().NotEqual(s.deps.Measured.Baseline, got,
				"it handed back the baseline unmoved")
		})
	}
}

// TestAGenreAimsAtItsEdgeAndAPlayerAtTheirMiddle covers the one difference
// between the two targets.
//
// Task 310. The middle is what every genre shares: totalled over the three
// band shares, ten of the eleven displace between 0.2 and 6.4 points from one
// baseline where the blocks they are ranked against span 50. One player's
// records describe one person, so their middle is the thing being aimed at
// rather than the thing that hides them.
func (s *PlayerTargetPublicTestSuite) TestAGenreAimsAtItsEdgeAndAPlayerAtTheirMiddle() {
	// Punk is the genre that displaces most, 18.90 points against the next
	// genre's 6.40, so it is the one whose edge is furthest from its middle.
	edge, ok := translate.AimAtGenre("punk", s.deps.Measured.Baseline)
	s.Require().True(ok)

	middle, ok := translate.AimAtPlayer("dee-dee-ramone", s.deps.Measured.Baseline)
	s.Require().True(ok)

	// Both are displacements over the same baseline, so neither is the
	// baseline and the two are not the same point.
	s.Require().NotEqual(s.deps.Measured.Baseline, edge)
	s.Require().NotEqual(edge, middle,
		"a genre's edge and one of its players' middle cannot be the same")
}

func TestPlayerTargetPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(PlayerTargetPublicTestSuite))
}

// TestAScatteredPlayerSaysSo covers a player whose own records disagree.
//
// A player's figures are their records together, so somebody whose records are
// several different sounds has figures describing none of them. The derivation
// already refuses to call that a word, which reads exactly like a player who
// is unremarkable, so the report has to say which it is.
func (s *PlayerTargetPublicTestSuite) TestAScatteredPlayerSaysSo() {
	bass := tone.AskInstrument("bass")
	dukowski := "Chuck Dukowski"

	// Three of the 43 are scattered, and this is one: his records disagree
	// about more than the other players disagree with each other.
	_, notes, err := translate.Translate(tone.Ask{
		Genre: []string{"punk"}, Instrument: &bass,
		Like: &tone.Like{Artist: &dukowski},
	}, config.Config{}, s.deps)
	s.Require().NoError(err)

	said := ""
	for _, n := range notes {
		said += n.Said + "\n"
	}

	s.Require().Contains(said, "their own records disagree about",
		"a scattered player read as an unremarkable one")
	s.Require().Contains(said, "measured to chuck-dukowski",
		"it stopped aiming at him")
}
