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

package presets_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/config"
	"github.com/retr0h/toneharness/pkg/sdk/internal/presets"
)

// SpeakerPublicTestSuite covers the cabinet block coming out where somebody
// plays into a cabinet of their own.
//
// A cabinet block simulates a speaker and a real cabinet is one, so a chain
// carrying both puts a simulated speaker in front of a real one. The person
// who wrote `amp-return` has already said which they have.
type SpeakerPublicTestSuite struct {
	suite.Suite

	dir string
}

func (s *SpeakerPublicTestSuite) SetupTest() {
	s.dir = s.T().TempDir()
}

// built makes a preset from the shipped punk rig under the config given, and
// says which categories the chain came out holding.
func (s *SpeakerPublicTestSuite) built(
	body string,
) ([]catalog.Category, []string) {
	at := filepath.Join(s.dir, "config.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(body), 0o600))

	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	made, err := presets.Make(context.Background(), presets.MakeOptions{
		RigID:      "punk",
		ConfigPath: at,
		OutputPath: filepath.Join(s.dir, "out.hlx"),
	})
	s.Require().NoError(err)

	kinds := make([]catalog.Category, 0, len(made.Plan.Blocks))
	for _, b := range made.Plan.Blocks {
		held, known := cat.Block(b.Model)
		s.Require().True(known, string(b.Model))
		kinds = append(kinds, held.Category)
	}

	left := make([]string, 0, len(made.Omitted))
	for _, o := range made.Omitted {
		left = append(left, o.Name)
	}

	return kinds, left
}

// holds says whether the chain carries one of that kind.
func holds(
	kinds []catalog.Category,
	of catalog.Category,
) bool {
	for _, k := range kinds {
		if k == of {
			return true
		}
	}

	return false
}

// TestWhereItPlays covers each answer the field takes.
//
// One method and one table, so a case is a row rather than a file.
func (s *SpeakerPublicTestSuite) TestWhereItPlays() {
	const head = "schema: Config\nversion: \"1.1.0\"\n"

	for _, tt := range []struct {
		name string
		body string
		// cab says whether the chain should come out holding one.
		cab bool
	}{
		{
			// Nothing else is a speaker, so it has to be there.
			name: "headphones",
			body: head + "plays_into: headphones\n",
			cab:  true,
		},
		{
			// A PA, a full-range speaker or an interface: the same path with
			// the room left in.
			name: "a desk",
			body: head + "plays_into: pa\n",
			cab:  true,
		},
		{
			// The effects return, past the amplifier's own preamp and into
			// its speaker.
			name: "the return of an amplifier",
			body: head + "plays_into: amp-return\n",
		},
		{
			// The instrument input, so the amplifier's preamp is in front of
			// its speaker. A different question about the preamp and the same
			// one about the speaker.
			name: "the front of an amplifier",
			body: head + "plays_into: amp-front\n",
		},
		{
			// Saying nothing builds the cabinet, which is what a preset heard
			// on headphones or sent to a desk needs. A config is not a
			// declaration that somebody owns a stack.
			name: "a config that does not say",
			body: head,
			cab:  true,
		},
	} {
		s.Run(tt.name, func() {
			kinds, left := s.built(tt.body)

			s.Require().True(holds(kinds, catalog.CategoryAmp),
				"the amplifier stays either way")
			s.Require().Equal(tt.cab, holds(kinds, catalog.CategoryCab))

			if tt.cab {
				s.Require().Empty(left, "nothing was taken out")

				return
			}

			// Said rather than left to be noticed: a chain that came back
			// without the cabinet its own document names is one somebody
			// would otherwise go looking for.
			s.Require().Len(left, 1)
			s.Require().Contains(left[0], "8x10")
		})
	}
}

// TestTheGridHasNoHoleInIt covers the positions closing up behind the
// cabinet.
//
// A block's position is what a footswitch names it by, so a gap where the
// cabinet was would put every switch after it one place out — which is task
// 278 arrived at from the other direction.
func (s *SpeakerPublicTestSuite) TestTheGridHasNoHoleInIt() {
	at := filepath.Join(s.dir, "config.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(
		"schema: Config\nversion: \"1.1.0\"\nplays_into: amp-return\n"), 0o600))

	made, err := presets.Make(context.Background(), presets.MakeOptions{
		RigID:      "punk",
		ConfigPath: at,
		OutputPath: filepath.Join(s.dir, "out.hlx"),
	})
	s.Require().NoError(err)

	for i, b := range made.Plan.Blocks {
		s.Require().Equal(i, b.Pos, "block %d sits at %d", i, b.Pos)
	}
}

// TestASlotOverridesTheFile covers one slot differing from the rest, which
// is how the same rig is built twice.
//
// The file says headphones and the slot says it plays into an amplifier, so
// the slot wins — for that build and no other.
func (s *SpeakerPublicTestSuite) TestASlotOverridesTheFile() {
	at := filepath.Join(s.dir, "config.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(
		"schema: Config\nversion: \"1.1.0\"\nplays_into: headphones\n"), 0o600))

	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	into := config.AmpReturn

	for _, tt := range []struct {
		name string
		// override is what the slot says, or nil for a slot that says
		// nothing and takes the file's answer.
		override *config.PlaysInto
		cab      bool
	}{
		{name: "a slot that says nothing takes the file's answer", cab: true},
		{name: "a slot that plays into an amplifier", override: &into},
	} {
		s.Run(tt.name, func() {
			made, err := presets.Make(context.Background(), presets.MakeOptions{
				RigID:      "punk",
				ConfigPath: at,
				PlaysInto:  tt.override,
				OutputPath: filepath.Join(s.dir, "out.hlx"),
			})
			s.Require().NoError(err)

			held := false

			for _, b := range made.Plan.Blocks {
				if one, known := cat.Block(b.Model); known &&
					one.Category == catalog.CategoryCab {
					held = true
				}
			}

			s.Require().Equal(tt.cab, held)
		})
	}
}

// TestAChainWithNoCabinetIsLeftAlone covers a rig naming none, where there is
// nothing to take out and nothing to report.
//
// A chain of effects is one: no amplifier, so no cabinet and nothing for
// `impliedCab` to supply either.
func (s *SpeakerPublicTestSuite) TestAChainWithNoCabinetIsLeftAlone() {
	at := filepath.Join(s.dir, "config.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(
		"schema: Config\nversion: \"1.1.0\"\nplays_into: amp-return\n"), 0o600))

	rig := filepath.Join(s.dir, "effects.yaml")
	s.Require().NoError(os.WriteFile(rig, []byte(`schema: ToneSpec
version: "4.0.0"
id: just-a-drive
rig:
  instrument: bass
  chain:
    - role: drive
      gear: Klon Centaur
`), 0o600))

	made, err := presets.Make(context.Background(), presets.MakeOptions{
		RigPath:    rig,
		ConfigPath: at,
		OutputPath: filepath.Join(s.dir, "out.hlx"),
	})
	s.Require().NoError(err)
	s.Require().Empty(made.Omitted, "nothing to take out")
}

func TestSpeakerPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(SpeakerPublicTestSuite))
}
