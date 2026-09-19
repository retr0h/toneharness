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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/tone"
	"github.com/retr0h/tonestack/pkg/sdk/translate"
)

// TranslatePublicTestSuite covers turning what somebody asked for into what a
// device can be told.
type TranslatePublicTestSuite struct {
	suite.Suite

	deps translate.Deps
}

func (s *TranslatePublicTestSuite) SetupSuite() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	lib, err := measured.BuiltIn()
	s.Require().NoError(err)

	s.deps = translate.Deps{Catalog: cat, Measured: lib}
}

// ask reads a request written the way a person writes one.
func (s *TranslatePublicTestSuite) ask(
	body string,
) tone.Spec {
	spec, err := tone.Load(strings.NewReader("schema: ToneSpec\n" + body))
	s.Require().NoError(err)

	return spec
}

// setup reads what somebody has.
func (s *TranslatePublicTestSuite) setup(
	body string,
) tone.Setup {
	out, err := tone.LoadSetup(strings.NewReader("schema: Setup\n" + body))
	s.Require().NoError(err)

	return out
}

// recording is the one file this repository can always measure.
func (s *TranslatePublicTestSuite) recording() string {
	return filepath.Join("..", "..", "..", "resources", "dry", "bass-di.wav")
}

// TestNamedGearResolvesToAModel covers the deterministic half.
func (s *TranslatePublicTestSuite) TestNamedGearResolvesToAModel() {
	got, notes, err := translate.Translate(
		s.ask("gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)
	s.Require().NoError(rig.Validate(got))
	s.Require().Len(got.Chain, 1)
	s.Require().Equal("LA Studio Comp", got.Chain[0].Gear)
	s.Require().Equal("HD2_CompressorLAStudioComp",
		(*got.Chain[0].Models)["HX Stomp"])
	s.Require().NotEmpty(notes)
}

// TestARecordingChoosesTheAmplifier is the half no reasoning about names can
// do.
//
// The recording is measured through the same figures every block was, so the
// comparison is between two of the same kind of thing, and the answer is the
// nearest of two hundred and twenty four rather than whichever name somebody
// wrote down.
func (s *TranslatePublicTestSuite) TestARecordingChoosesTheAmplifier() {
	got, notes, err := translate.Translate(
		s.ask("like:\n  recording: "+s.recording()+"\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)
	s.Require().NoError(rig.Validate(got))
	s.Require().Len(got.Chain, 1)
	s.Require().Equal(rig.RoleAmp, got.Chain[0].Role)
	s.Require().NotEmpty((*got.Chain[0].Models)["HX Stomp"])

	var said string
	for _, note := range notes {
		if note.About == "amp" {
			said = note.Said
		}
	}

	s.Require().Contains(said, "closest of 224 measured",
		"the note says what it chose from and why")
}

// TestTheSameAskTwiceIsTheSameRig is what makes a RigSpec worth sharing.
func (s *TranslatePublicTestSuite) TestTheSameAskTwiceIsTheSameRig() {
	body := "words: [punchy]\nlike:\n  recording: " + s.recording() + "\n"

	first, _, err := translate.Translate(s.ask(body), s.setup(""), s.deps)
	s.Require().NoError(err)

	for range 5 {
		again, _, err := translate.Translate(s.ask(body), s.setup(""), s.deps)

		s.Require().NoError(err)
		s.Require().Equal(first, again)
	}
}

// TestWordsTravelAsWords is the line this does not cross.
//
// A character term is how it should sound, which is the same thing a
// ToneSpec's words are. Turning them into knob positions here would be the
// guessing the whole project removed.
func (s *TranslatePublicTestSuite) TestWordsTravelAsWords() {
	got, _, err := translate.Translate(
		s.ask("words: [punchy, dark]\ngear:\n  - gear: LA Studio Comp\n    role: comp\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)
	s.Require().NotNil(got.Character)
	s.Require().Len(*got.Character, 2)
	s.Require().Equal("punchy", (*got.Character)[0].Term)
}

// TestAChainIsOrderedBySignalPath covers gear listed in any order.
//
// A drive ahead of an amplifier is a different sound from one behind it, and
// a request listing gear is not stating a signal path.
func (s *TranslatePublicTestSuite) TestAChainIsOrderedBySignalPath() {
	got, _, err := translate.Translate(
		s.ask(`gear:
  - gear: Glitz Reverb
    role: reverb
  - gear: LA Studio Comp
    role: comp
  - gear: Minotaur
    role: drive
like:
  recording: `+s.recording()+"\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)

	roles := make([]rig.Role, 0, len(got.Chain))
	for _, entry := range got.Chain {
		roles = append(roles, entry.Role)
	}

	s.Require().Equal([]rig.Role{
		rig.RoleComp, rig.RoleDrive, rig.RoleAmp, rig.RoleReverb,
	}, roles)
}

// TestInsistRefusesRatherThanSubstitute covers the gear being the point.
func (s *TranslatePublicTestSuite) TestInsistRefusesRatherThanSubstitute() {
	_, notes, err := translate.Translate(
		s.ask("gear:\n  - gear: a pedal nobody makes\n    role: drive\n    insist: true\n"),
		s.setup(""), s.deps)

	s.Require().ErrorIs(err, translate.ErrInsisted)
	s.Require().NotEmpty(notes.Unmet())
}

// TestAnAmbiguousNameSaysWhichModelsItFits covers the names that collide.
//
// 665 models share 469 names, so "Ampeg SVT" fits four. The note lists model
// identifiers rather than names, because the names are what collided.
func (s *TranslatePublicTestSuite) TestAnAmbiguousNameSaysWhichModelsItFits() {
	_, notes, err := translate.Translate(
		s.ask("gear:\n  - gear: Ampeg SVT\n    role: amp\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)

	var said string
	for _, note := range notes.Unmet() {
		said = note.Said
	}

	s.Require().Contains(said, "fits")
	s.Require().Contains(said, "HD2_", "identifiers, because the names collided")
}

// TestWhatItCannotAnswerItSays covers a genre and a player.
//
// A request carrying either is half answered, and the half that was not is
// the part somebody needs to know about.
func (s *TranslatePublicTestSuite) TestWhatItCannotAnswerItSays() {
	_, notes, err := translate.Translate(
		s.ask(`genre: punk
like:
  artist: Mike Dirnt
  recording: `+s.recording()+"\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)

	said := strings.Join(sayings(notes.Unmet()), " ")
	s.Require().Contains(said, "no records carry that genre yet")
	s.Require().Contains(said, "Mike Dirnt")
}

// TestNothingToBuildFromIsRefused covers an empty ask.
func (s *TranslatePublicTestSuite) TestNothingToBuildFromIsRefused() {
	_, _, err := translate.Translate(s.ask("words: [dark]\n"), s.setup(""), s.deps)

	s.Require().ErrorContains(err, "no chain to build")
}

// TestTheSetupDecidesTheInstrument covers what somebody plays.
func (s *TranslatePublicTestSuite) TestTheSetupDecidesTheInstrument() {
	tests := []struct {
		name string
		give string
		want rig.Instrument
	}{
		{
			name: "a bass",
			give: "instruments:\n  - gear: Fender Jazz Bass\n",
			want: rig.InstrumentBass,
		},
		{
			name: "a guitar",
			give: "instruments:\n  - gear: Fender Telecaster\n",
			want: rig.InstrumentGuitar,
		},
		{
			// The one marked default rather than the first listed.
			name: "several, one of them marked",
			give: "instruments:\n" +
				"  - gear: Fender Telecaster\n" +
				"  - gear: Fender Jazz Bass\n    default: true\n",
			want: rig.InstrumentBass,
		},
		{name: "nothing at all", give: "", want: rig.InstrumentBass},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, _, err := translate.Translate(
				s.ask("gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
				s.setup(tt.give), s.deps)

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Instrument)
		})
	}
}

// TestTheSubjectFollowsTheAsk covers what a rig is attributed to.
func (s *TranslatePublicTestSuite) TestTheSubjectFollowsTheAsk() {
	tests := []struct {
		name string
		give string
		kind rig.Kind
		who  string
	}{
		{
			name: "a player",
			give: "like:\n  artist: Mike Dirnt\n  band: Green Day\n",
			kind: rig.KindArtist, who: "Mike Dirnt",
		},
		{
			name: "a band",
			give: "like:\n  band: Green Day\n",
			kind: rig.KindBand, who: "Green Day",
		},
		{
			name: "a song",
			give: "like:\n  song: Longview\n",
			kind: rig.KindSong, who: "Longview",
		},
		{
			name: "a genre",
			give: "genre: punk\n",
			kind: rig.KindGenre, who: "punk",
		},
		{
			name: "nobody in particular",
			give: "",
			kind: rig.KindSound, who: "a sound somebody asked for",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, _, err := translate.Translate(
				s.ask(tt.give+"gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
				s.setup(""), s.deps)

			s.Require().NoError(err)
			s.Require().Equal(tt.kind, got.Subject.Kind)
			s.Require().Equal(tt.who, got.Subject.Name)
			s.Require().NotEmpty(got.ID)
		})
	}
}

// TestARecordingItCannotReadIsSaid covers a path that is not audio.
func (s *TranslatePublicTestSuite) TestARecordingItCannotReadIsSaid() {
	tests := []struct{ name, give, want string }{
		{name: "not there", give: "nowhere.wav", want: "cannot be read"},
		{
			name: "not audio",
			give: filepath.Join("..", "..", "..", "go.mod"),
			want: "is not audio",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, notes, err := translate.Translate(
				s.ask("like:\n  recording: "+tt.give+
					"\ngear:\n  - gear: LA Studio Comp\n    role: comp\n"),
				s.setup(""), s.deps)

			s.Require().NoError(err)
			s.Require().Contains(
				strings.Join(sayings(notes.Unmet()), " "), tt.want)
		})
	}
}

// TestANudgeHasNothingToMoveFrom covers the one kind of ask that needs a
// previous answer.
func (s *TranslatePublicTestSuite) TestANudgeHasNothingToMoveFrom() {
	_, notes, err := translate.Translate(
		s.ask("nudges:\n  - word: darker\ngear:\n  - gear: LA Studio Comp\n    role: comp\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)
	s.Require().Contains(strings.Join(sayings(notes.Unmet()), " "),
		"nothing to move from")
}

// TestAGearNameFittingManySaysSoWithoutListingThemAll covers the cap.
//
// Some names fit a dozen, and a note listing all of them is one nobody reads.
func (s *TranslatePublicTestSuite) TestAGearNameFittingManySaysSoWithoutListingThemAll() {
	_, notes, err := translate.Translate(
		s.ask("gear:\n  - gear: 4x12\n    role: cab\n"),
		s.setup(""), s.deps)

	s.Require().NoError(err)

	said := strings.Join(sayings(notes.Unmet()), " ")
	s.Require().Contains(said, "more")
}

// TestARoleNothingIsMeasuredForIsSaid covers asking for a kind of block that
// no reading covers.
func (s *TranslatePublicTestSuite) TestARoleNothingIsMeasuredForIsSaid() {
	empty := s.deps
	empty.Measured = measured.Library{
		Device: "HX Stomp",
		Blocks: map[string]measured.Block{
			"a": {ID: "a", Category: "cab"},
		},
	}

	_, notes, err := translate.Translate(
		s.ask("like:\n  recording: "+s.recording()+
			"\ngear:\n  - gear: LA Studio Comp\n    role: comp\n"),
		s.setup(""), empty)

	s.Require().NoError(err)
	s.Require().Contains(strings.Join(sayings(notes.Unmet()), " "),
		"nothing of that kind has been measured")
}

// TestGearWithNoRoleIsLookedForAnywhere covers an entry that names none.
func (s *TranslatePublicTestSuite) TestGearWithNoRoleIsLookedForAnywhere() {
	got, _, err := translate.Translate(
		s.ask("gear:\n  - gear: LA Studio Comp\n"), s.setup(""), s.deps)

	s.Require().NoError(err)
	s.Require().Equal(rig.RoleOther, got.Chain[0].Role,
		"it goes in as other, and the compiler places it")
}

// TestASetupNamingAnotherDeviceIsRefused is not a warning.
//
// Every block was measured on one piece of hardware, and a ranking built from
// those readings ranks that device's blocks. Run against another it answers
// confidently with models the device in hand may not even have.
func (s *TranslatePublicTestSuite) TestASetupNamingAnotherDeviceIsRefused() {
	_, notes, err := translate.Translate(
		s.ask("gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
		s.setup("device:\n  model: Helix Floor\n"), s.deps)

	s.Require().ErrorIs(err, translate.ErrWrongDevice)
	s.Require().Contains(strings.Join(sayings(notes.Unmet()), " "),
		"every block was measured on HX Stomp")
}

// TestASetupNamingTheSameDeviceIsFine covers the ordinary case.
func (s *TranslatePublicTestSuite) TestASetupNamingTheSameDeviceIsFine() {
	_, _, err := translate.Translate(
		s.ask("gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
		s.setup("device:\n  model: HX Stomp\n"), s.deps)

	s.Require().NoError(err)
}

// TestAnImpulseResponseNobodyLoadedIsNotChosen covers a block that carries an
// index rather than any audio.
//
// What is in that slot is whatever its owner put there, so a chain naming one
// sounds like something on the device it was built on and like nothing at all
// on anybody else's. The reading for one is of an empty slot.
func (s *TranslatePublicTestSuite) TestAnImpulseResponseNobodyLoadedIsNotChosen() {
	quiet := measured.Figures{Centroid: 111, Low: 99, Mid: 1}

	only := s.deps
	only.Measured = measured.Library{
		Device: "HX Stomp",
		Blocks: map[string]measured.Block{
			"HD2_ImpulseResponse1024": {
				ID: "HD2_ImpulseResponse1024", Name: "IR 1024",
				Category: "amp", Figures: quiet,
			},
		},
	}

	_, notes, err := translate.Translate(
		s.ask("like:\n  recording: "+s.recording()+
			"\ngear:\n  - gear: LA Studio Comp\n    role: comp\n"),
		s.setup(""), only)

	s.Require().NoError(err)
	s.Require().Contains(strings.Join(sayings(notes.Unmet()), " "),
		"nothing of that kind has been measured",
		"the only candidate was one nobody has loaded")
}

// TestAnImpulseResponseSomebodyOwnsIsChosen covers the setup unlocking it.
func (s *TranslatePublicTestSuite) TestAnImpulseResponseSomebodyOwnsIsChosen() {
	only := s.deps
	only.Measured = measured.Library{
		Device: "HX Stomp",
		Blocks: map[string]measured.Block{
			"HD2_ImpulseResponse1024": {
				ID: "HD2_ImpulseResponse1024", Name: "IR 1024",
				Category: "amp",
				Figures:  measured.Figures{Centroid: 140, Low: 93, Mid: 7},
			},
		},
	}

	got, _, err := translate.Translate(
		s.ask("like:\n  recording: "+s.recording()+"\n"),
		s.setup("owns:\n  - kind: ir\n    name: IR 1024\n    slot: 3\n"),
		only)

	s.Require().NoError(err)
	s.Require().Len(got.Chain, 1)
	s.Require().Equal("IR 1024", got.Chain[0].Gear)
}

// sayings is what a set of notes said.
func sayings(
	notes translate.Notes,
) []string {
	out := make([]string, 0, len(notes))
	for _, note := range notes {
		out = append(out, note.About+": "+note.Said)
	}

	return out
}

func TestTranslatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TranslatePublicTestSuite))
}
