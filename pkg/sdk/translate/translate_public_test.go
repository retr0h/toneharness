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
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
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
	// A genre on every ask, because the contract requires one and most cases
	// below are about the rest of the document. One that names its own keeps it.
	head := "schema: ToneSpec\n"
	if !strings.Contains(body, "genre:") {
		// A genre the corpus has measured, so the ask resolves fully and the
		// cases below see only the notes they are about.
		head += "genre: [punk]\n"
	}

	spec, err := tone.Load(strings.NewReader(head + body))
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
	return filepath.Join("..", "..", "..", "resources", "dry", "bass-di-short.wav")
}

// TestTranslate covers Translate, which turns a request and a setup into a
// rig.
//
// One method and one table, so a case is a row rather than a file.
func (s *TranslatePublicTestSuite) TestTranslate() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The deterministic half.
			name: "named gear resolves to a model",
			then: func() {
				got, notes, err := translate.Translate(
					s.ask("gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)
				s.Require().NoError(rig.Validate(got))
				s.Require().Len(got.Chain, 1)
				s.Require().Equal("LA Studio Comp", got.Chain[0].Gear)
				s.Require().NotEmpty(notes)

				// Said rather than written into the entry. Which model a name resolves to
				// is the plan's answer, so the rig names the gear and the note names the
				// model it reached.
				s.Require().Contains(sayings(notes),
					"LA Studio Comp: resolved to HD2_CompressorLAStudioComp")
			},
		},
		{
			// The bug this exists for, and it was invisible because it was a
			// coin toss.
			//
			// 355 of this device's 661 models share a name with another of
			// their own category, and lookup returned the first one a Go map
			// yielded. The same ask compiled to three different cabinets
			// across twelve runs: HD2_Cab1x15TucknGo at 7.2 DSP with no
			// microphone list, and two HD2_CabMicIr models at 2.5 with one.
			// Which a chain got decided whether its most powerful control
			// existed, and it was decided by nothing.
			//
			// Run rather than asserted once, because a map order bug passes a
			// single run two times in three.
			name: "a gear name fitting several exactly answers the same way every time",
			then: func() {
				said := map[string]int{}

				for range 24 {
					_, notes, err := translate.Translate(
						s.ask("gear:\n  - gear: 1x15 Ampeg B-15\n    role: cab\n"),
						s.setup(""), s.deps)

					s.Require().NoError(err)

					for _, note := range notes {
						if note.About == "1x15 Ampeg B-15" {
							said[note.Said]++
						}
					}
				}

				s.Require().Len(said, 1, "one ask, one answer: %v", said)

				for got := range said {
					// The mic'd cabinet, chosen by family rather than by the alphabet. The
					// same speaker ships three times under one name and this is the one
					// carrying Mic, Angle and Position, at 2.5 DSP against 7.2.
					s.Require().Contains(got, "resolved to HD2_CabMicIr_1x15AmpegB15")

					// And the other two named, because "resolved to X" on its own reads as
					// the only answer when it was one of three.
					s.Require().Contains(got, "that name fits 3 exactly")
					s.Require().Contains(got, "HD2_Cab1x15TucknGo",
						"the legacy one is named as an alternative rather than chosen")
				}
			},
		},
		{
			// The half no reasoning about names can do.
			//
			// The recording is measured through the same figures every block
			// was, so the comparison is between two of the same kind of
			// thing, and the answer is the nearest of two hundred and twenty
			// four rather than whichever name somebody wrote down.
			name: "a recording chooses the amplifier",
			then: func() {
				got, notes, err := translate.Translate(
					s.ask("like:\n  recording: "+s.recording()+"\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)
				s.Require().NoError(rig.Validate(got))
				s.Require().Len(got.Chain, 1)
				s.Require().Equal(rig.RoleAmp, got.Chain[0].Role)
				s.Require().NotEmpty(got.Chain[0].Gear)

				var said string
				for _, note := range notes {
					if note.About == "amp" {
						said = note.Said
					}
				}

				s.Require().Contains(said, "closest of 224 measured",
					"the note says what it chose from and why")
			},
		},
		{
			// What makes a RigSpec worth sharing.
			name: "the same ask twice is the same rig",
			then: func() {
				body := "words:\n  - term: punchy\nlike:\n  recording: " + s.recording() + "\n"

				first, _, err := translate.Translate(s.ask(body), s.setup(""), s.deps)
				s.Require().NoError(err)

				for range 5 {
					again, _, err := translate.Translate(s.ask(body), s.setup(""), s.deps)

					s.Require().NoError(err)
					s.Require().Equal(first, again)
				}
			},
		},
		{
			// The line this does not cross.
			//
			// How it should sound is what somebody asked for, so it stays on
			// the ask and the rig says only which gear answered. A rig
			// carrying the words too would be a second place for the answer
			// to live, and turning them into knob positions here would be the
			// guessing the whole project removed: that happens once, in the
			// compiler, which is the only place the resolved chain exists.
			name: "words stay on the ask",
			then: func() {
				got, _, err := translate.Translate(
					s.ask(
						"words:\n  - term: punchy\n  - term: dark\ngear:\n  - gear: LA Studio Comp\n    role: comp\n",
					),
					s.setup(""),
					s.deps,
				)

				s.Require().NoError(err)
				s.Require().NotEmpty(got.Chain, "the gear still answered")

				built, err := json.Marshal(got)
				s.Require().NoError(err)
				s.Require().NotContains(string(built), "punchy")
				s.Require().NotContains(string(built), "dark")
			},
		},
		{
			// Gear listed in any order.
			//
			// A drive ahead of an amplifier is a different sound from one
			// behind it, and a request listing gear is not stating a signal
			// path.
			name: "a chain is ordered by signal path",
			then: func() {
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
			},
		},
		{
			// #128's honest half.
			//
			// Flatwounds against roundwounds is a larger difference than most
			// pedals make, and an amplifier chosen by measuring a record
			// played on flats is chosen against a spectrum nobody will
			// reproduce on rounds. Nothing here has measured what that does,
			// so the mismatch is reported rather than corrected for: applying
			// a number nobody measured is the guessing this project removed.
			name: "strings that do not match are reported",
			then: func() {
				tests := []struct {
					name  string
					ask   string
					setup string
					says  bool
				}{
					{
						name:  "flats on the record and rounds in the room",
						ask:   "played:\n  - gear: Fender Precision\n    strings: flat\n",
						setup: "instruments:\n  - gear: Fender Jazz\n    strings: round\n    default: true\n",
						says:  true,
					},
					{
						name:  "the same strings either side",
						ask:   "played:\n  - gear: Fender Precision\n    strings: round\n",
						setup: "instruments:\n  - gear: Fender Jazz\n    strings: round\n    default: true\n",
					},
					{
						// Unknown is not a mismatch. Saying nobody knows and then
						// reporting a difference against it would be inventing one.
						name:  "nobody knows what the record was played on",
						ask:   "played:\n  - gear: Fender Precision\n    strings: unknown\n",
						setup: "instruments:\n  - gear: Fender Jazz\n    strings: round\n    default: true\n",
					},
					{
						name:  "the request says nothing about strings",
						ask:   "played:\n  - gear: Fender Precision\n",
						setup: "instruments:\n  - gear: Fender Jazz\n    strings: round\n    default: true\n",
					},
					{
						name:  "the setup says nothing about strings",
						ask:   "played:\n  - gear: Fender Precision\n    strings: flat\n",
						setup: "instruments:\n  - gear: Fender Jazz\n    default: true\n",
					},
					{
						// A setup naming no default falls to the first instrument, which
						// is what somebody with one bass has written: `default: true` on
						// a list of one says nothing, so nobody types it.
						name:  "nothing marked default, so the first one answers",
						ask:   "played:\n  - gear: Fender Precision\n    strings: flat\n",
						setup: "instruments:\n  - gear: Fender Jazz\n    strings: round\n",
						says:  true,
					},
					{
						name:  "and the first one saying nothing is not a mismatch",
						ask:   "played:\n  - gear: Fender Precision\n    strings: flat\n",
						setup: "instruments:\n  - gear: Fender Jazz\n",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						_, notes, err := translate.Translate(
							s.ask(tt.ask+"gear:\n  - gear: Ampeg SVT\n    role: amp\n"),
							s.setup(tt.setup), s.deps)

						s.Require().NoError(err)

						var said bool
						for _, n := range notes {
							said = said || n.About == "strings"
						}

						s.Require().Equal(tt.says, said)
					})
				}
			},
		},
		{
			// A drive behind the amplifier.
			//
			// A chain is sorted into the ordinary signal path, because
			// listing gear is not stating one: a drive goes in front of the
			// amplifier whichever order somebody typed. That is right for the
			// common case and wrong for somebody who means it, and a drive
			// behind the amplifier is a known way to use one rather than a
			// mistake. Before this the only route was to build the rig and
			// edit a line, which is an edit nobody records the reason for.
			name: "a request may say where a block goes",
			then: func() {
				tests := []struct {
					name string
					ask  string
					want []rig.Role
				}{
					{
						// The ordinary case, unchanged: order asked for is ignored.
						name: "a drive nobody placed goes in front",
						ask: `gear:
  - gear: Minotaur
    role: drive
  - gear: Ampeg SVT
    role: amp
`,
						want: []rig.Role{rig.RoleDrive, rig.RoleAmp},
					},
					{
						name: "a drive asked to sit behind the amplifier does",
						ask: `gear:
  - gear: Minotaur
    role: drive
    after: amp
  - gear: Ampeg SVT
    role: amp
`,
						want: []rig.Role{rig.RoleAmp, rig.RoleDrive},
					},
					{
						// Named by role rather than by position, so it lands after the
						// amplifier and before what usually follows one.
						name: "and still before the reverb",
						ask: `gear:
  - gear: Glitz Reverb
    role: reverb
  - gear: Minotaur
    role: drive
    after: amp
  - gear: Ampeg SVT
    role: amp
`,
						want: []rig.Role{rig.RoleAmp, rig.RoleDrive, rig.RoleReverb},
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, notes, err := translate.Translate(
							s.ask(tt.ask), s.setup(""), s.deps)

						s.Require().NoError(err)
						s.Require().NoError(rig.Validate(got))

						roles := make([]rig.Role, 0, len(got.Chain))
						for _, entry := range got.Chain {
							roles = append(roles, entry.Role)
						}

						s.Require().Equal(tt.want, roles)

						// A placement somebody asked for is reported, because a chain in
						// an order nobody expected should say who chose it.
						if strings.Contains(tt.ask, "after:") {
							var said bool
							for _, n := range notes {
								said = said || strings.Contains(n.Said, "behind the amp")
							}

							s.Require().True(said, "the notes say it was asked for")
						}
					})
				}
			},
		},
		{
			// The rest of the signal path.
			//
			// The ordering test above walks the four roles a request usually
			// names. This walks the ones it rarely does, because a role
			// nothing places falls to the end of the chain, and falling to
			// the end is right for a role nobody has an opinion about and
			// wrong for a wah.
			//
			// A pitch block is the fallthrough on purpose: no fixed place in
			// a signal path is the honest answer for it, so it lands after
			// everything that has one, and this pins that rather than leaving
			// it to whatever a map iterates.
			name: "every role has a place in the chain",
			then: func() {
				got, _, err := translate.Translate(
					s.ask(`gear:
  - gear: Buzz Wave
    role: pitch
  - gear: Analog Echo
    role: delay
  - gear: Chorus
    role: mod
  - gear: Cali Q
    role: eq
  - gear: Comet Trails
    role: filter
  - gear: Chrome
    role: wah
`),
					s.setup(""), s.deps)

				s.Require().NoError(err)
				s.Require().NoError(rig.Validate(got))

				roles := make([]rig.Role, 0, len(got.Chain))
				for _, entry := range got.Chain {
					roles = append(roles, entry.Role)
				}

				s.Require().Equal([]rig.Role{
					rig.RoleWah, rig.RoleFilter, rig.RoleEQ,
					rig.RoleMod, rig.RoleDelay, rig.RolePitch,
				}, roles)
			},
		},
		{
			// The vague ask.
			//
			// "Punk bass" is a real request and there is nothing in it to
			// resolve, so it is refused with a sentinel rather than guessed
			// at. Which question to ask next depends on whether a person or
			// an agent is asking, so the SDK names the problem and the caller
			// names the next step.
			name: "a request with nothing in it is refused",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("genre: [punk]\ninstrument: bass\n"), s.setup(""), s.deps)

				s.Require().ErrorIs(err, translate.ErrNothingToBuildFrom)
				s.Require().NotEmpty(notes, "and it still says what it assumed on the way")
			},
		},
		{
			// How a note reads.
			//
			// Both of these are the same assumption about the same absent
			// setup, and one of them reported "could not", which told
			// somebody the tool had failed at something it had in fact
			// decided.
			name: "an assumption is something that happened",
			then: func() {
				_, notes, _ := translate.Translate(
					s.ask("like:\n  recording: "+s.recording()+"\n"), s.setup(""), s.deps)

				for _, about := range []string{"device", "instrument"} {
					s.Run(about, func() {
						for _, n := range notes {
							if n.About == about {
								s.Require().True(n.Honoured,
									"%q assumed a default, which is a thing it did", about)

								return
							}
						}

						s.Require().Fail("no note about " + about)
					})
				}
			},
		},
		{
			// An empty ask.
			name: "nothing to build from is refused",
			then: func() {
				_, _, err := translate.Translate(s.ask("words:\n  - term: dark\n"), s.setup(""), s.deps)

				s.Require().ErrorContains(err, "no chain to build")
			},
		},
		{
			// What somebody plays.
			name: "the setup decides the instrument",
			then: func() {
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
			},
		},
		{
			// What a rig is named after.
			//
			// Who the rig is for is the ask's to say and no longer sits on
			// the rig, so the identifier is what carries the attribution
			// across: it is how a rig is found again, and a rig named after
			// nobody could not be.
			name: "the identifier follows the ask",
			then: func() {
				tests := []struct {
					name string
					give string
					want string
				}{
					{
						name: "a player",
						give: "like:\n  artist: Mike Dirnt\n  band: Green Day\n",
						want: "mike-dirnt-green-day",
					},
					{
						name: "a band",
						give: "like:\n  band: Green Day\n",
						want: "green-day",
					},
					{
						name: "a song",
						give: "like:\n  song: Longview\n",
						want: "longview",
					},
					{
						// The genre is not in the name. Every ask carries one now, so it
						// would prefix every identifier in the repository with a word and
						// distinguish nothing.
						name: "a genre and nobody else",
						give: "genre: [punk]\n",
						want: "a-sound",
					},
					{
						name: "nobody in particular",
						give: "",
						want: "a-sound",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, _, err := translate.Translate(
							s.ask(tt.give+"gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
							s.setup(""), s.deps)

						s.Require().NoError(err)
						s.Require().Equal(tt.want, got.ID)
					})
				}
			},
		},
		{
			// An entry that names none.
			name: "gear with no role is looked for anywhere",
			then: func() {
				got, _, err := translate.Translate(
					s.ask("gear:\n  - gear: LA Studio Comp\n"), s.setup(""), s.deps)

				s.Require().NoError(err)
				s.Require().Equal(rig.RoleOther, got.Chain[0].Role,
					"it goes in as other, and the compiler places it")
			},
		},
		{
			// The ordinary case.
			name: "a setup naming the same device is fine",
			then: func() {
				_, _, err := translate.Translate(
					s.ask("gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
					s.setup("device:\n  model: HX Stomp\n"), s.deps)

				s.Require().NoError(err)
			},
		},
		{
			// The setup unlocking it.
			name: "an impulse response somebody owns is chosen",
			then: func() {
				only := s.deps
				only.Measured = measured.Library{
					Device: "HX Stomp",
					Blocks: map[string]measured.Block{
						"HD2_ImpulseResponse1024": {
							ID: "HD2_ImpulseResponse1024", Name: "IR 1024",
							Category: "amp",
							Figures:  measured.Figures{Centroid: 140, Level: -20, Low: 93, Mid: 7},
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
			},
		},
		{
			// Plays_into.
			//
			// A cabinet block simulates a speaker and an amplifier has one,
			// so a chain holding a cabinet played into an amplifier has both.
			// Said rather than changed: a cabinet block is how a chain is
			// made to sound like the record it came from, so somebody chasing
			// a record through their own amplifier wants both and is right
			// to, and the tool does not get to decide that.
			name: "two speakers in the path are counted",
			then: func() {
				chain := "gear:\n  - {gear: Ampeg SVT, role: amp}\n  - {gear: 8x10, role: cab}\n"

				for _, tt := range []struct {
					name string
					// chain overrides the one-cabinet chain above, for a row
					// about how many are counted.
					chain string
					into  string
					says  bool
					// held is what the note should name the speakers as.
					held string
				}{
					{name: "into the front of an amplifier", into: "plays_into: amp-front\n", says: true},
					{
						// Two cabinet blocks and an amplifier's own speaker is
						// three speakers in the path, so the note counts them
						// rather than naming one.
						name: "two cabinet blocks in the path are counted",
						chain: "gear:\n  - {gear: Ampeg SVT, role: amp}\n" +
							"  - {gear: 8x10, role: cab}\n  - {gear: 4x10, role: cab}\n",
						into: "plays_into: amp-front\n", says: true,
						held: "2 cabinet blocks",
					},
					{name: "into an amplifier's effects return", into: "plays_into: amp-return\n", says: true},
					{
						// A PA has no speaker of its own, so the cabinet block is the only
						// one in the path and there is nothing to say.
						name: "into a PA", into: "plays_into: pa\n",
					},
					{name: "into headphones", into: "plays_into: headphones\n"},
					{name: "a setup that does not say", into: ""},
				} {
					s.Run(tt.name, func() {
						use := chain
						if tt.chain != "" {
							use = tt.chain
						}

						_, notes, err := translate.Translate(
							s.ask(use), s.setup(tt.into), s.deps)

						s.Require().NoError(err)

						said := ""

						for _, n := range notes {
							if n.About == "plays_into" {
								said = n.Said

								s.Require().True(n.Honoured, "nothing was changed")
							}
						}

						if !tt.says {
							s.Require().Empty(said)

							return
						}

						held := tt.held
						if held == "" {
							held = "a cabinet block"
						}

						s.Require().Contains(said, held)
						s.Require().Contains(said, "speaker of its own")
					})
				}
			},
		},
		{
			// The other half of the count.
			//
			// Nothing to report: one speaker in the path is the amplifier's,
			// which is what somebody plugging into one already knows.
			name: "an amplifier with no cabinet says nothing",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("gear:\n  - {gear: Ampeg SVT, role: amp}\n"),
					s.setup("plays_into: amp-front\n"), s.deps)

				s.Require().NoError(err)

				for _, n := range notes {
					s.Require().NotEqual("plays_into", n.About)
				}
			},
		},
		{
			// The promise the contract had not kept.
			//
			// "a word that reaches no control cannot be aimed at and saying
			// so is better than accepting it and quietly doing nothing" — and
			// nothing said it. The check existed in compile and was reached
			// only by `presets make`, so a `tone build` took an ask saying
			// `sparkly`, resolved the chain, reported every other thing it
			// did, and never mentioned the word.
			name: "a word nothing aims at is said",
			then: func() {
				deps := s.deps
				deps.UnknownWords = func(words []string) []translate.UnknownWord {
					out := make([]translate.UnknownWord, 0, len(words))
					for _, w := range words {
						if w == "bright" {
							continue
						}

						near := []string(nil)
						if w == "pick attack audible" {
							near = []string{"audible-pick-attack"}
						}

						out = append(out, translate.UnknownWord{Term: w, Near: near})
					}

					return out
				}

				_, notes, err := translate.Translate(
					s.ask("schema: ToneSpec\ngenre: [punk]\n"+
						"gear:\n  - {gear: Ampeg SVT, role: amp}\n"+
						"words:\n  - term: bright\n  - term: sparkly\n"+
						"  - term: pick attack audible\n"),
					s.setup(""), deps)

				s.Require().NoError(err, "one adjective does not lose the record beside it")

				said := map[string]translate.Note{}
				for _, n := range notes {
					said[n.About] = n
				}

				s.Require().NotContains(said, "bright", "a word it carries says nothing")

				s.Require().Contains(said, "sparkly")
				s.Require().False(said["sparkly"].Honoured, "it reached no control")
				s.Require().NotContains(said["sparkly"].Said, "Did you mean",
					"nothing is close, so nothing is suggested")

				s.Require().Contains(said["pick attack audible"].Said,
					"Did you mean audible-pick-attack?",
					"offered rather than substituted: a guess that lands wrong aims the "+
						"answer somewhere nobody can see")
			},
		},
		{
			// A caller that wants only a chain.
			name: "nothing checking words says nothing",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("schema: ToneSpec\ngenre: [punk]\n"+
						"gear:\n  - {gear: Ampeg SVT, role: amp}\n"+
						"words:\n  - term: sparkly\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				for _, n := range notes {
					s.Require().NotEqual("sparkly", n.About)
				}
			},
		},
		{
			// The other collision.
			//
			// 108 names on this device are both a full amplifier and a
			// preamp, and they share a category, so a rig naming "Ampeg SVT"
			// could reach either. It means the amplifier. Until the family
			// decided it the answer was whichever sorted first, which
			// happened to be right because HD2_Amp precedes HD2_Preamp, and
			// right by the alphabet is not right by decision.
			name: "an amplifier is preferred to a preamp of the same name",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("gear:\n  - gear: A30 Fawn Nrm\n    role: amp\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				var said string

				for _, note := range notes {
					if note.About == "A30 Fawn Nrm" {
						said = note.Said
					}
				}

				s.Require().Contains(said, "HD2_AmpA30FawnNrm")
				s.Require().NotContains(said, "resolved to HD2_PreampA30FawnNrm")
			},
		},
		{
			// A bug this had for as long as `identify` had a slug rule of its
			// own.
			//
			// It mapped a space to a hyphen, deleted everything else and
			// never collapsed the runs that left behind, so "Earth, Wind &
			// Fire" became "earth-wind--fire". The contract's pattern for an
			// id is `^[a-z0-9]+(-[a-z0-9]+)*$`, which forbids two hyphens
			// together, so Translate resolved the entire chain, chose an
			// amplifier by measurement, and then refused to write the rig it
			// had just built. Any ampersand, comma or double space in a band,
			// artist or song did it.
			//
			// The rule is `slug.Of` now, which every other identifier in the
			// repository already used, including one 260 lines away in this
			// same file.
			name: "an ampersand in a name still validates",
			then: func() {
				for _, name := range []string{
					"Earth, Wind & Fire",
					"Sly & The Family Stone",
					"AC/DC",
					"Red  Hot  Chili Peppers",
					"Motley_Crue",
				} {
					s.Run(name, func() {
						out, _, err := translate.Translate(
							s.ask("like:\n  band: "+name+"\n  recording: "+s.recording()+"\n"),
							s.setup(""), s.deps)

						s.Require().NoError(err, "%q makes a rig the contract refuses", name)
						s.Require().NotContains(out.ID, "--",
							"two hyphens together are what the pattern forbids")
						s.Require().NoError(rig.Validate(out),
							"the rig it built has to satisfy the contract it is validated against")
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

// TestUnmet covers Unmet, which is the notes describing what could not be
// done.
//
// One method and one table, so a case is a row rather than a file.
func (s *TranslatePublicTestSuite) TestUnmet() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The gear being the point.
			name: "insist refuses rather than substitute",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("gear:\n  - gear: a pedal nobody makes\n    role: drive\n    insist: true\n"),
					s.setup(""), s.deps)

				s.Require().ErrorIs(err, translate.ErrInsisted)
				s.Require().NotEmpty(notes.Unmet())
			},
		},
		{
			// The names that collide.
			//
			// 661 models share 468 names, so "Ampeg SVT" fits four. The note
			// lists model identifiers rather than names, because the names
			// are what collided.
			name: "an ambiguous name says which models it fits",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("gear:\n  - gear: Ampeg SVT\n    role: amp\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				// Across every note rather than the last one. Each ask carries a genre now,
				// so which note comes last is a fact about the reporting order and not about
				// the ambiguous name this is looking for.
				var said string
				for _, note := range notes.Unmet() {
					said += note.Said + "\n"
				}

				s.Require().Contains(said, "fits")
				s.Require().Contains(said, "HD2_", "identifiers, because the names collided")
			},
		},
		{
			// A genre and a player.
			//
			// A request carrying either is half answered, and the half that
			// was not is the part somebody needs to know about.
			name: "what it cannot answer it says",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask(`genre: [punk]
like:
  artist: Mike Dirnt
  recording: `+s.recording()+"\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				said := strings.Join(sayings(notes.Unmet()), " ")

				// Punk is measured and clears the threshold, so the note says what it
				// measured as rather than that nobody has tagged anything, which is what it
				// used to say of every genre.
				//
				// It names a word now. Punk set none of the figures apart from the players
				// who avoid it until Alkaline Trio's records joined it, and then it earned
				// `clean`. What is asserted is that the note reports the measurement, not
				// which word came out of it.
				s.Require().Contains(said, "records from")
				s.Require().Contains(said, "measured across")
				s.Require().Contains(said, "Mike Dirnt")
			},
		},
		{
			// The one that would otherwise converge and be wrong. A bass
			// corpus puts every genre's centroid between 90 and 182Hz, and a
			// guitar chain solved against that is not near it: it is being
			// asked to sound like another instrument, and every dial would be
			// spent doing it. Which instrument the ask states is the only
			// thing that catches it, so it travels into the note.
			name: "the ask names one instrument and the genre is measured on another",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("instrument: guitar\ngenre: [punk]\nlike:\n  recording: "+
						s.recording()+"\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				said := strings.Join(sayings(notes.Unmet()), " ")
				s.Require().Contains(said, "is measured on bass")
				s.Require().Contains(said, "this asks for guitar")
				s.Require().Contains(said, "about an octave")
			},
		},
		{
			// The answer for a word with no records.
			name: "a genre nobody has tagged",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("genre: [sea-shanty]\nlike:\n  recording: "+s.recording()+"\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)
				s.Require().Contains(strings.Join(sayings(notes.Unmet()), " "),
					"no records carry that genre")
			},
		},
		{
			// The answer somebody asked for.
			//
			// Grunge is displaced on two axes against the players who play
			// none of it, so the note says what it measured as rather than
			// what it cannot do.
			name: "a genre that earns words",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("genre: [grunge]\nlike:\n  recording: "+s.recording()+"\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				said := strings.Join(sayings(notes.Unmet()), " ")
				s.Require().Contains(said, "measured across")
				s.Require().Contains(said, "scooped")
			},
		},
		{
			// A path that is not audio.
			name: "a recording it cannot read is said",
			then: func() {
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
			},
		},
		{
			// The one kind of ask that needs a previous answer.
			name: "a nudge has nothing to move from",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("nudges:\n  - word: darker\ngear:\n  - gear: LA Studio Comp\n    role: comp\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)
				s.Require().Contains(strings.Join(sayings(notes.Unmet()), " "),
					"nothing to move from")
			},
		},
		{
			// What a rebuild does with rounds of correction somebody already
			// made.
			//
			// Nothing, and says so. A correction's paths point into the plan
			// it was made against and this builds a new one, so the settings
			// those rounds arrived at are not in the answer. Left unsaid it
			// reads as a rebuild that carried them.
			//
			// The entry with no verdict is the one that matters: it has been
			// built and not yet listened to, which is the state a person
			// needs shown rather than left to be rediscovered.
			name: "a correction history is read out and not replayed",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("corrections:\n"+
						"  - ask: make it clunkier\n"+
						"    verdict: muddy now\n"+
						"  - ask: put the drive back\n"+
						"    changed:\n"+
						"      - {path: \"chain[1].settings.drive\", from: 0.58, to: 0.47}\n"+
						"gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				said := strings.Join(sayings(notes.Unmet()), " ")
				s.Require().Contains(said, "muddy now")
				s.Require().Contains(said, "does not replay it")
				s.Require().Contains(said, "nobody has said what it sounded like")
			},
		},
		{
			// The cap.
			//
			// Some names fit a dozen, and a note listing all of them is one
			// nobody reads.
			name: "a gear name fitting many says so without listing them all",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("gear:\n  - gear: 4x12\n    role: cab\n"),
					s.setup(""), s.deps)

				s.Require().NoError(err)

				said := strings.Join(sayings(notes.Unmet()), " ")
				s.Require().Contains(said, "more")
			},
		},
		{
			// Asking for a kind of block that no reading covers.
			name: "a role nothing is measured for is said",
			then: func() {
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
			},
		},
		{
			// Not a warning.
			//
			// Every block was measured on one piece of hardware, and a
			// ranking built from those readings ranks that device's blocks.
			// Run against another it answers confidently with models the
			// device in hand may not even have.
			name: "a setup naming another device is refused",
			then: func() {
				_, notes, err := translate.Translate(
					s.ask("gear:\n  - gear: LA Studio Comp\n    role: comp\n"),
					s.setup("device:\n  model: Helix Floor\n"), s.deps)

				s.Require().ErrorIs(err, translate.ErrWrongDevice)
				s.Require().Contains(strings.Join(sayings(notes.Unmet()), " "),
					"every block was measured on HX Stomp")
			},
		},
		{
			// A block that carries an index rather than any audio.
			//
			// What is in that slot is whatever its owner put there, so a
			// chain naming one sounds like something on the device it was
			// built on and like nothing at all on anybody else's. The reading
			// for one is of an empty slot.
			name: "an impulse response nobody loaded is not chosen",
			then: func() {
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
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestAGenreUnderTheThreshold covers the count being reported.
//
// Said as the two numbers rather than as a refusal, because how far short it is
// decides what to download next. Tested directly, because every genre measured
// into this binary clears the threshold and which ones do depends on whichever
// records somebody tagged.
func (s *TranslatePublicTestSuite) TestAGenreUnderTheThreshold() {
	got := translate.GenreNote("emo", audio.Genre{
		Name: "emo", Slug: "emo", Records: 3, Players: 1, Instrument: "bass",
	}, true, "bass")

	s.Require().Contains(got.Said, "3 records from 1 players")
	s.Require().Contains(got.Said, "under the eight from three")
	s.Require().Contains(got.Said, "wearing a genre's name")
}

// TestAGenreMeasuredOnAnotherInstrument is the one that would otherwise
// converge and be wrong.
//
// Every figure in this repository was measured on bass, which puts each genre's
// centroid between 90 and 182Hz. Aim a guitar chain at one of those and the
// solve does not fail: it spends every dial driving the chain an octave down,
// reports its tolerances met, and has answered a question nobody asked.
//
// So it is said before any of that happens, and it names both sides. A note
// saying only "out of reach" would send somebody looking at the gear.
func (s *TranslatePublicTestSuite) TestAGenreMeasuredOnAnotherInstrument() {
	got := translate.GenreNote("grunge", audio.Genre{
		Name: "grunge", Slug: "grunge", Records: 9, Players: 3,
		Instrument: "bass", Usable: true,
	}, true, "guitar")

	s.Require().Contains(got.Said, "measured on bass")
	s.Require().Contains(got.Said, "asks for guitar")
	s.Require().Contains(got.Said, "octave")
}

// TestAGenrePooledAcrossInstrumentsIsRefused covers figures describing nothing.
//
// Reported apart from the threshold, because the count is not what is wrong
// with it: nine records from three players is plenty, and a centre of gravity
// halfway between a bass and a guitar is not a target either.
func (s *TranslatePublicTestSuite) TestAGenrePooledAcrossInstrumentsIsRefused() {
	got := translate.GenreNote("punk", audio.Genre{
		Name: "punk", Slug: "punk", Records: 9, Players: 3,
	}, true, "bass")

	s.Require().Contains(got.Said, "not all played on one instrument")
	s.Require().Contains(got.Said, "9 records")
}

// TestAnAskNamingNoInstrumentClaimsNothing covers leaving it to the Setup.
//
// A request may say nothing about the instrument, and then there is no
// disagreement to report: the note says what the genre measured as, the same as
// it always did.
func (s *TranslatePublicTestSuite) TestAnAskNamingNoInstrumentClaimsNothing() {
	got := translate.GenreNote("grunge", audio.Genre{
		Name: "grunge", Slug: "grunge", Records: 9, Players: 3,
		Instrument: "bass", Usable: true,
		Terms: []audio.Derived{{Term: "scooped"}},
	}, true, "")

	s.Require().Contains(got.Said, "measured across")
	s.Require().NotContains(got.Said, "asks for")
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

// TestALegacyModelIsReachableByTheNameLine6GivesIt covers the other side of
// the family preference.
//
// The same cabinet ships three times under one display name, so the preference
// has to pick one and it picks the mic'd model. That would strand the legacy
// one if nothing else reached it. Line 6's own guide calls it `Legacy 1x15"
// Ampeg B-15` and the catalog records that as what it is based on, so a rig
// naming it gets it.
//
// It only works because the guide's two entries are now read separately. The
// Pilot's Guide prints its "Based On" header once per table and the cabinets
// run over four pages, so the extractor read the page with the header and
// dropped the three before it: the current models' rows were on those, the
// Legacy section's header was on the page after, and 58 mic'd cabinets ended up
// carrying their legacy twin's entry. All three then claimed to be Legacy and
// this name reached whichever the preference chose.
func (s *TranslatePublicTestSuite) TestALegacyModelIsReachableByTheNameLine6GivesIt() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	legacy, ok := cat.Blocks["HD2_Cab1x15TucknGo"]
	s.Require().True(ok)
	s.Require().Equal(`Legacy 1x15" Ampeg B-15`, legacy.BasedOn)

	for _, id := range []string{
		"HD2_CabMicIr_1x15AmpegB15", "HD2_CabMicIr_1x15AmpegB15WithPan",
	} {
		got, ok := cat.Blocks[catalog.ModelID(id)]
		s.Require().True(ok)
		s.Require().Equal(`1x15" Ampeg B-15`, got.BasedOn,
			"%s is that cabinet mic'd, not the legacy model", id)
	}
}

// TestNoMicdCabinetClaimsToBeALegacyModel holds the whole set, not one example.
func (s *TranslatePublicTestSuite) TestNoMicdCabinetClaimsToBeALegacyModel() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	var mics int

	for id, block := range cat.Blocks {
		if block.Family != "cabmicirs" && block.Family != "cabmicirswithpan" {
			continue
		}

		mics++

		s.Require().NotContains(block.BasedOn, "Legacy",
			"%s is a current model", id)
	}

	s.Require().Positive(mics, "the catalog carries mic'd cabinets at all")
}
