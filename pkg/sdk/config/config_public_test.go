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

package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/config"
)

// ConfigPublicTestSuite covers the application's own settings file.
//
// Its own contract since 2026-10-25, and these cases moved here with it. A rig
// is published and read by strangers against a version they can see; a
// preference never leaves the machine it is on. While the two shared one
// contract, adding a field here moved the version every shipped rig declared.
type ConfigPublicTestSuite struct {
	suite.Suite
}

// iotest is a reader that fails, for the path where the file cannot be read at
// all rather than cannot be understood.
type iotest struct{}

func (iotest) Read(
	[]byte,
) (int, error) {
	return 0, errors.New("boom")
}

// TestLoad covers reading a config and checking it against its own contract.
//
// One method and one table, so a case is a row rather than a file.
func (s *ConfigPublicTestSuite) TestLoad() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "what somebody has, read back",
			then: func() {
				held, err := config.Load(strings.NewReader(`
schema: Config
version: "1.0.0"
device:
  model: HX Stomp
instruments:
  - gear: Fender Jazz Bass
    strings: flat
owns:
  - kind: ir
    name: A cabinet somebody loaded
slots:
  - { at: 13C, rig: punk }
favourites: ["13C"]
match_loudness: punk
`))

				s.Require().NoError(err)
				s.Require().Equal("HX Stomp", held.Device.Model)
				s.Require().Equal(config.OwnedIR, (*held.Owns)[0].Kind)
				s.Require().Equal("13C", (*held.Slots)[0].At)
				s.Require().Equal([]string{"13C"}, *held.Favourites)
			},
		},
		{
			// Four spellings and nothing else, because the two amplifier
			// entries are different questions — the instrument input has the
			// amplifier's own preamp in front of its speaker, and the effects
			// return does not — and a free-string field would let somebody
			// write "amp" and mean either.
			name: "what the pedal is plugged into is optional and spelt",
			then: func() {
				held, err := config.Load(strings.NewReader(
					"schema: Config\nversion: \"1.0.0\"\n"))

				s.Require().NoError(err)
				s.Require().Nil(held.PlaysInto, "a config that does not say says nothing")

				for _, want := range []config.PlaysInto{
					config.PA, config.Headphones, config.AmpFront, config.AmpReturn,
				} {
					got, err := config.Load(strings.NewReader(
						"schema: Config\nversion: \"1.0.0\"\nplays_into: " +
							string(want) + "\n"))

					s.Require().NoError(err, string(want))
					s.Require().Equal(want, *got.PlaysInto)
				}

				_, err = config.Load(strings.NewReader(
					"schema: Config\nversion: \"1.0.0\"\nplays_into: amp\n"))

				s.Require().ErrorIs(err, config.ErrInvalid,
					"`amp` is two different paths, so the contract will not take it")
			},
		},
		{
			// The rig contract is a different document with a different
			// version, and saying one while being the other is a file in the
			// wrong place rather than a field to ignore.
			name: "a rig is not a config",
			then: func() {
				_, err := config.Load(strings.NewReader(
					"schema: ToneSpec\nversion: \"4.0.0\"\n"))

				s.Require().ErrorIs(err, config.ErrInvalid)
			},
		},
		{
			// Checked before anything decodes it, because decoding drops what
			// the generated types have no field for: a typo would be dropped
			// on the way in, the file would pass, and the line somebody wrote
			// would be gone with nothing said.
			name: "a field nobody spelt right",
			then: func() {
				_, err := config.Load(strings.NewReader(
					"schema: Config\nversion: \"1.0.0\"\nplays_intoo: pa\n"))

				s.Require().ErrorIs(err, config.ErrInvalid)
				s.Require().ErrorContains(err, "plays_intoo")
			},
		},
		{
			name: "a file that will not read at all",
			then: func() {
				_, err := config.Load(iotest{})

				s.Require().ErrorContains(err, "boom")
			},
		},
		{
			name: "a file that is not YAML",
			then: func() {
				_, err := config.Load(strings.NewReader("\tnot: [yaml"))

				s.Require().Error(err)
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestValidate covers a config assembled in Go rather than read off disk.
//
// Nothing in a generated struct stops a required field being empty or an
// enumeration carrying a word that is not in it, so one built in memory has
// never been checked.
func (s *ConfigPublicTestSuite) TestValidate() {
	s.Require().ErrorIs(config.Validate(config.Config{}), config.ErrInvalid,
		"a config saying nothing does not say what it is")

	held, err := config.Load(strings.NewReader(
		"schema: Config\nversion: \"1.0.0\"\nplays_into: headphones\n"))
	s.Require().NoError(err)
	s.Require().NoError(config.Validate(held))
}

// TestVersion covers the contract stating its own version, which is how a file
// says which rules it was written under.
func (s *ConfigPublicTestSuite) TestVersion() {
	got, err := config.Version()

	s.Require().NoError(err)
	s.Require().NotEmpty(got)
	s.Require().Regexp(`^\d+\.\d+\.\d+$`, got)
}

// TestTheConfigHoldsNothingAboutTheSound is the other half of the split.
//
// What somebody wants changes every request; what they own changes when they
// buy something. A genre or a word in here would be a preference about one
// sound pretending to be a preference about the person, and it would apply to
// every build they ever ran.
//
// The counterpart lives in the tone package, which holds the ask to the same
// line from the other side.
func (s *ConfigPublicTestSuite) TestTheConfigHoldsNothingAboutTheSound() {
	doc, err := openapi3.NewLoader().LoadFromData(config.Schema)
	s.Require().NoError(err)

	held := doc.Components.Schemas[config.Name].Value.Properties

	for _, field := range []string{"genre", "words", "nudges", "like", "chain", "ask", "rig"} {
		s.Require().NotContains(held, field,
			"%s belongs to the request, not to the person", field)
	}
}

func TestConfigPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(ConfigPublicTestSuite))
}
