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
package sdk_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/config"
)

// ConfigFilePublicTestSuite covers the document that says what somebody has, and
// the command that writes one.
//
// Every field on a setup is optional, which makes the empty document valid and
// useless. So the starter file is the feature: what lands is the schema line,
// the version this build reads, and a commented line beside each field saying
// what it is for.
type ConfigFilePublicTestSuite struct {
	suite.Suite
}

// TestConfigPathIsUnderTheConfigDirectory covers where a setup lives.
//
// A setup is configuration rather than data or state, so it is the config
// directory, resolved the way rigs and backups resolve theirs: the XDG
// variable, and the documented fallback under the home directory where the
// environment says nothing.
func (s *ConfigFilePublicTestSuite) TestConfigPathIsUnderTheConfigDirectory() {
	for _, tt := range []struct {
		name string
		// config is what XDG_CONFIG_HOME is set to, or empty to unset it.
		config string
		// under is what the answer has to end with.
		under string
	}{
		{
			name:   "the config directory the environment names",
			config: filepath.Join("/tmp", "someone", "config"),
			under:  filepath.Join("/tmp", "someone", "config", "toneharness", "config.yaml"),
		},
		{
			name:  "the documented fallback where it names none",
			under: filepath.Join(".config", "toneharness", "config.yaml"),
		},
	} {
		s.Run(tt.name, func() {
			s.T().Setenv("XDG_CONFIG_HOME", tt.config)

			at, err := sdk.ConfigPath()

			s.Require().NoError(err)
			s.Require().True(strings.HasSuffix(at, tt.under),
				"%s does not end with %s", at, tt.under)
		})
	}
}

// TestStartConfigWritesOneThatValidates covers what a starter setup is for.
//
// Read back through the loader rather than only checked for bytes, because a
// starter file that does not validate is worse than none: it is one somebody
// edits for an hour before the first refusal, and the refusal then names
// their line rather than the one that shipped.
func (s *ConfigFilePublicTestSuite) TestStartConfigWritesOneThatValidates() {
	at := filepath.Join(s.T().TempDir(), "config.yaml")

	written, err := sdk.New().StartConfig(at)

	s.Require().NoError(err)
	s.Require().Equal(at, written.Path)
	s.Require().Equal(config.Name, written.Name)

	body, err := os.ReadFile(at)
	s.Require().NoError(err)

	// The version this build reads, rather than one typed into the template:
	// a starter file stating a version the embedded contract does not is one
	// that fails the moment it is read back.
	version, err := config.Version()
	s.Require().NoError(err)
	s.Require().Contains(string(body), `version: "`+version+`"`)

	// Every field somebody fills in is named, and commented out, so what
	// lands is valid on its own and becomes useful one line at a time.
	for _, field := range []string{
		"# device:", "# instruments:", "# slots:", "# favourites:",
		"# match_loudness:",
	} {
		s.Require().Contains(string(body), field, "the starter setup names no "+field)
	}

	held, err := sdk.New(sdk.WithConfig(at)).Setup()

	s.Require().NoError(err)
	s.Require().Nil(held.Slots, "a starter setup places no slots")
	s.Require().Nil(held.Favourites, "a starter setup lists no favourites")
}

// TestStartConfigDefaultsToWhereCommandsLook covers the path it takes when
// nobody names one, which is how it is meant to be run.
func (s *ConfigFilePublicTestSuite) TestStartConfigDefaultsToWhereCommandsLook() {
	home := s.T().TempDir()
	s.T().Setenv("XDG_CONFIG_HOME", home)

	written, err := sdk.New().StartConfig("")

	s.Require().NoError(err)
	s.Require().Equal(
		filepath.Join(home, "toneharness", "config.yaml"), written.Path)
}

// TestConfigPathReportsNoHomeDirectory covers the one thing that can stop a
// path being worked out.
//
// Unreachable in a correct build, which is why it is reachable from here: this
// branch decides whether somebody is told where to put a setup or told
// nothing, and a branch nothing can reach is one nobody has read.
func (s *ConfigFilePublicTestSuite) TestConfigPathReportsNoHomeDirectory() {
	s.T().Setenv("XDG_CONFIG_HOME", "")
	s.T().Cleanup(sdk.HomeDirWas(func() (string, error) {
		return "", errors.New("nobody lives here")
	}))

	_, err := sdk.ConfigPath()

	s.Require().ErrorContains(err, "finding where a setup lives")
	s.Require().ErrorContains(err, "nobody lives here")

	// And the same failure reaching StartConfig, which is where somebody
	// actually meets it: nowhere named means the default path, and there is
	// no default path to work out.
	_, err = sdk.New().StartConfig("")

	s.Require().ErrorContains(err, "nobody lives here")
}

// TestStartConfigRefusesATemplateThatHasDrifted covers the guard between the
// contract and the file this writes.
//
// Checked before the file lands rather than after, so a starter setup that
// does not validate never reaches the disk. The version is what the template
// takes from the contract, so a version the contract's own pattern refuses is
// the drift this is for.
func (s *ConfigFilePublicTestSuite) TestStartConfigRefusesATemplateThatHasDrifted() {
	for _, tt := range []struct {
		name string
		// version is what the contract's version is made to answer.
		version string
		fails   error
		says    string
		// stillThere is false where nothing should have been written.
		at string
	}{
		{
			name:    "a version the contract has no pattern for",
			version: "not-a-version",
			says:    "the starter setup does not validate",
		},
		{
			name:  "a contract that will not say what version it is",
			fails: errors.New("the schema states no version"),
			says:  "writing a starter setup",
		},
	} {
		s.Run(tt.name, func() {
			s.T().Cleanup(sdk.ContractVersionWas(func() (string, error) {
				return tt.version, tt.fails
			}))

			at := filepath.Join(s.T().TempDir(), "config.yaml")

			_, err := sdk.New().StartConfig(at)

			s.Require().ErrorContains(err, tt.says)

			// Nothing on the disk, which is the point of checking first.
			_, statErr := os.Stat(at)
			s.Require().ErrorIs(statErr, os.ErrNotExist)
		})
	}
}

// TestStartConfigRefuses covers the answers that are not a file.
func (s *ConfigFilePublicTestSuite) TestStartConfigRefuses() {
	s.Run("a setup already there", func() {
		at := filepath.Join(s.T().TempDir(), "config.yaml")
		s.Require().NoError(os.WriteFile(at, []byte("mine\n"), 0o600))

		_, err := sdk.New().StartConfig(at)

		// Reported rather than replaced, and the whole file left alone. A
		// setup is hand-authored and the only record of decisions nothing
		// else keeps, so moving the old one aside is the person's call.
		s.Require().ErrorIs(err, sdk.ErrConfigExists)
		s.Require().ErrorContains(err, at)

		body, err := os.ReadFile(at)
		s.Require().NoError(err)
		s.Require().Equal("mine\n", string(body))
	})

	s.Run("a directory nothing may write into", func() {
		// The write fails rather than the directory: the parent is already
		// there so making it succeeds, and then nothing may put a file in it.
		dir := filepath.Join(s.T().TempDir(), "config")
		s.Require().NoError(os.Mkdir(dir, 0o500))

		// Put the permissions back, so the temporary directory can be
		// removed when the test ends.
		s.T().Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		at := filepath.Join(dir, "config.yaml")

		_, err := sdk.New().StartConfig(at)

		s.Require().ErrorContains(err, "writing "+at)
	})

	s.Run("nowhere to put it", func() {
		// A file where the directory has to go, so making the directory
		// fails rather than the write.
		blocked := filepath.Join(s.T().TempDir(), "in-the-way")
		s.Require().NoError(os.WriteFile(blocked, nil, 0o600))

		_, err := sdk.New().StartConfig(filepath.Join(blocked, "config.yaml"))

		s.Require().ErrorContains(err, "making somewhere for a setup")
	})
}

// TestConfigReadsBackWhatWasWritten covers the read the slot map and the
// favourites are asked for through.
func (s *ConfigFilePublicTestSuite) TestConfigReadsBackWhatWasWritten() {
	at := filepath.Join(s.T().TempDir(), "config.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(`schema: Config
version: "1.0.0"
favourites: ["11A", "19B"]
slots:
  - { at: 11A, rig: punk }
`), 0o600))

	held, err := sdk.New(sdk.WithConfig(at)).Setup()

	s.Require().NoError(err)
	s.Require().NotNil(held.Favourites)
	s.Require().Equal([]string{"11A", "19B"}, *held.Favourites)
	s.Require().NotNil(held.Slots)
	s.Require().Len(*held.Slots, 1)
	s.Require().Equal("punk", (*held.Slots)[0].Rig)
}

// TestConfigRefusesWhatCannotBeRead covers naming no setup and naming one that
// is not there.
func (s *ConfigFilePublicTestSuite) TestConfigRefusesWhatCannotBeRead() {
	for _, tt := range []struct {
		name string
		at   string
		says string
	}{
		{name: "no setup at all", says: "name one with --setup"},
		{name: "a setup that is not there", at: "no-such-setup.yaml", says: "no-such-setup.yaml"},
	} {
		s.Run(tt.name, func() {
			_, err := sdk.New(sdk.WithConfig(tt.at)).Setup()

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

func TestConfigFilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ConfigFilePublicTestSuite))
}
