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
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// InstrumentPublicTestSuite covers which instrument a chain is for, and the
// gear it refuses on the strength of it.
//
// Task 266. Nothing here works for guitar because there is no guitar corpus,
// no guitar-referenced sweep and no guitar genre, and the answer to "does it
// refuse or guess" turned out to be guess: a request carrying
// `instrument: guitar` was answered "the setup names none, so this is for
// bass" and handed hard-rock's Ampeg SVT.
type InstrumentPublicTestSuite struct {
	suite.Suite
}

// TestInstrumentFor covers where the answer comes from.
//
// One method and one table, so a case is a row rather than a file.
func (s *InstrumentPublicTestSuite) TestInstrumentFor() {
	bass := tone.AskInstrument("bass")
	guitar := tone.AskInstrument("guitar")

	owns := func(gear string) config.Config {
		yes := true
		held := []config.Held{{Gear: gear, Default: &yes}}

		return config.Config{Instruments: &held}
	}

	for _, tt := range []struct {
		name string
		spec tone.Ask
		held config.Config
		want rig.Instrument
		// says is a fragment the note has to carry.
		says string
	}{
		{
			// What somebody owns is a fact and what they want changes every
			// time they ask, so the setup still wins where it says.
			name: "the setup wins where it names one",
			spec: tone.Ask{Instrument: &guitar},
			held: owns("Fender Jazz Bass"),
			want: rig.InstrumentBass,
		},
		{
			// Task 266's trap, now shut. A Rickenbacker 4001 is a bass and
			// its name does not carry the word, so reading the name alone
			// answered guitar for an ask that said bass in as many words.
			//
			// Two things stop it and the order matters. The name is known
			// here, so this never reaches the request; the case below is the
			// one that proves the ordering rather than the list.
			name: "a bass whose name does not carry the word",
			spec: tone.Ask{Instrument: &bass},
			held: owns("Rickenbacker 4001"),
			want: rig.InstrumentBass,
			says: "read as a bass, which is what that name is",
		},
		{
			// The half of the fix that does not depend on a list being
			// complete. Nothing recognises this name, so the guess would have
			// said guitar, and the request is asked first and says bass.
			name: "a name nothing knows, with a request that says",
			spec: tone.Ask{Instrument: &bass},
			held: owns("Wal Mk II"),
			want: rig.InstrumentBass,
			says: "the request asks for bass",
		},
		{
			// And the guess survives where there is nothing to prefer over
			// it. A Stratocaster in a setup and silence in the ask is a
			// guitar, and answering bass there would be worse than answering
			// from the name.
			name: "a name nothing knows, with nothing to fall back on",
			held: owns("Fender Stratocaster"),
			want: rig.InstrumentGuitar,
			says: "read as a guitar rather than a bass",
		},
		{
			// What the list is for: a bass named without the word and no
			// request to rescue it.
			name: "a known bass with nothing to fall back on",
			held: owns("1962 Precision"),
			want: rig.InstrumentBass,
			says: "read as a bass, which is what that name is",
		},
		{
			// The case task 266 found. The contract's own description of the
			// field is "what this is for, where the request asserts it rather
			// than leaving it to the Setup".
			name: "the ask answers where the setup is silent",
			spec: tone.Ask{Instrument: &guitar},
			want: "guitar",
			says: "the request asks for guitar",
		},
		{
			name: "and for a bass the same way",
			spec: tone.Ask{Instrument: &bass},
			want: rig.InstrumentBass,
			says: "the request asks for bass",
		},
		{
			// A setup that carries the field and names nothing in it, which
			// is a file somebody started and did not fill in. There is no
			// instrument to read, so it is the same as having no setup.
			name: "a setup naming an empty list of instruments",
			spec: tone.Ask{Instrument: &guitar},
			held: config.Config{Instruments: &[]config.Held{}},
			want: "guitar",
			says: "the request asks for guitar",
		},
		{
			// An assumption that was made is a thing that happened, so it is
			// reported as done rather than as a failure.
			name: "a bass where neither says, and it says so",
			want: rig.InstrumentBass,
			says: "neither the setup nor the request names one",
		},
		{
			name: "an empty instrument on the ask is not an answer",
			spec: tone.Ask{Instrument: func() *tone.AskInstrument {
				was := tone.AskInstrument("")

				return &was
			}()},
			want: rig.InstrumentBass,
			says: "neither the setup nor the request names one",
		},
	} {
		s.Run(tt.name, func() {
			notes := translate.Notes{}
			got := translate.InstrumentFor(tt.spec, tt.held, &notes)

			s.Require().Equal(tt.want, got)

			if tt.says == "" {
				return
			}

			said := ""
			for _, note := range notes {
				said += note.Said + "\n"
			}

			s.Require().Contains(said, tt.says)
		})
	}
}

// TestGearIsRefusedAcrossInstruments covers the two evidence tiers above
// measurement.
//
// `genreTarget` had this guard on the figures and nothing had it on the gear,
// so a guitar ask naming hard-rock was refused its figures with "bass sits
// about an octave from guitar" in the same breath as being handed the Ampeg
// SVT three of its five players used. The gear is cited and it is cited for a
// different instrument, which is a worse answer than none because every claim
// on it is true.
func (s *InstrumentPublicTestSuite) TestGearIsRefusedAcrossInstruments() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	lib, err := measured.BuiltIn()
	s.Require().NoError(err)

	// A researched rig for a bass player, which is what all 32 shipped ones
	// are.
	theirs := rig.Spec{
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			{Role: rig.RoleCab, Gear: "Ampeg 8x10"},
		},
	}

	deps := translate.Deps{
		Catalog: cat, Measured: lib,
		RigNamed: func(string) (rig.Spec, bool) { return theirs, true },
	}

	who := "Mike Dirnt"
	ask := tone.Ask{
		Genre: []string{"hard-rock"},
		Like:  &tone.Like{Artist: &who},
	}

	for _, tt := range []struct {
		name  string
		wants rig.Instrument
		// took says the gear came across.
		took bool
		says string
	}{
		{
			name:  "a bass chain takes a bass player's gear",
			wants: rig.InstrumentBass,
			took:  true,
		},
		{
			name:  "a guitar chain takes none of it",
			wants: "guitar",
			says:  "is for bass and this is for guitar",
		},
		{
			// Nothing said, so nothing is refused: a caller with no
			// instrument gets what it always got.
			name:  "and an unstated instrument refuses nothing",
			wants: "",
			took:  true,
		},
	} {
		s.Run(tt.name, func() {
			notes := translate.Notes{}
			got, ok := translate.LikeTheirRig(ask, tt.wants, deps, &notes)

			s.Require().Equal(tt.took, ok)

			if tt.took {
				s.Require().NotEmpty(got)

				return
			}

			s.Require().Empty(got)

			said := ""
			for _, note := range notes {
				said += note.Said + "\n"
			}

			s.Require().Contains(said, tt.says)
		})
	}

	// And the same on the genre tier, where the players are the corpus's
	// rather than a fixture's.
	s.Run("a genre measured on bass gives a guitar chain nothing", func() {
		notes := translate.Notes{}
		got, ok := translate.LikeTheirGenre(
			tone.Ask{Genre: []string{"hard-rock"}}, "guitar", deps, &notes)

		s.Require().False(ok)
		s.Require().Empty(got)

		said := ""
		for _, note := range notes {
			said += note.Said + "\n"
		}

		s.Require().Contains(said,
			"its players are measured on bass and this is for guitar")
	})
}

func TestInstrumentPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(InstrumentPublicTestSuite))
}
