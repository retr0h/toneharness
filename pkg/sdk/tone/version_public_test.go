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

package tone_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"

	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// The contract as it last changed: the version somebody decided on, and the
// bytes that version describes.
//
// A pair, kept by hand on purpose. Nothing regenerates them, because a
// generator would move the hash on its own and the whole point is that
// changing the contract costs a decision about which number moves.
const (
	contractVersion = "4.2.0"
	contractSHA     = "71cd799bc361c98be2c5b8cb21d99872ecc444a391d2c2b0b7108ca151fb2830"
)

// semver is major.minor.patch and nothing else. No build metadata and no
// pre-release: this is one file in one repository, not a published package.
var semver = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// VersionPublicTestSuite holds the contract to saying which version of itself
// it is.
//
// Every document in the marketplace, every Go type downstream and every
// `description:` an agent reads as grammar comes out of one file. A reader
// that cannot tell which version of it they have cannot tell whether a
// document predates a field or breaks a rule that was added after it: both
// read as a document that is simply wrong.
type VersionPublicTestSuite struct {
	suite.Suite
}

// version reads what the contract says its version is.
func (s *VersionPublicTestSuite) version() string {
	var doc struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}

	s.Require().NoError(yaml.Unmarshal(tone.Schema, &doc))

	return doc.Info.Version
}

// TestTheContractCarriesASemanticVersion covers the version the contract
// states being one.
func (s *VersionPublicTestSuite) TestTheContractCarriesASemanticVersion() {
	got := s.version()

	s.Require().NotEmpty(got, "the contract states no version at all")
	s.Require().Regexp(
		semver, got,
		"info.version is %q, which is not major.minor.patch", got,
	)
}

// TestTheContractsVersionMovedWithIt covers the version being bumped whenever
// the contract changes.
//
// The failure this exists for is silent: a field's `description:` is the
// grammar for that field, so a sentence edited there changes what the tool
// refuses, and a document written yesterday against the old sentence is
// refused today with nothing saying why. The version is what tells those two
// cases apart, and a version nobody bumps says nothing.
//
// What moves which number is in write-a-spec's versions.md. Briefly: a field
// or value that used to validate and no longer does is major; anything added
// is minor; wording that changes nothing about what is accepted is patch.
func (s *VersionPublicTestSuite) TestTheContractsVersionMovedWithIt() {
	sum := sha256.Sum256(tone.Schema)
	got := hex.EncodeToString(sum[:])

	s.Require().Equal(
		contractSHA, got,
		"the contract changed. Decide which number moves (write-a-spec's "+
			"versions.md says how), set info.version, then record the pair "+
			"in this file: contractVersion %q, contractSHA %q",
		s.version(), got,
	)

	s.Require().Equal(
		contractVersion, s.version(),
		"the contract's bytes are the ones recorded here but info.version "+
			"reads %q against the recorded %q. One of the two was edited "+
			"without the other",
		s.version(), contractVersion,
	)
}

// TestADocumentStatesWhichContractItWasWrittenAgainst covers the version a
// document carries, and what a reader does with it.
//
// One method and one table, so a case is a row rather than a file.
func (s *VersionPublicTestSuite) TestADocumentStatesWhichContractItWasWrittenAgainst() {
	mine, err := tone.Version()
	s.Require().NoError(err)

	// Whatever the contract is at, and the three versions around it.
	parts := strings.Split(mine, ".")
	s.Require().Len(parts, 3)

	bump := func(at int, by int) string {
		n, err := strconv.Atoi(parts[at])
		s.Require().NoError(err)

		out := append([]string(nil), parts...)
		out[at] = strconv.Itoa(n + by)

		return strings.Join(out, ".")
	}

	document := func(version string) string {
		return fmt.Sprintf(`schema: ToneSpec
version: %q
id: whoever
rig:
  instrument: bass
  chain:
    - {role: amp, gear: Ampeg SVT}
`, version)
	}

	for _, tt := range []struct {
		name    string
		version string
		refused string
	}{
		{
			name:    "the version this binary reads",
			version: mine,
		},
		{
			// Minors add, so a document written against a later one is a
			// document using fields this reader knows and not using the new
			// ones. Nothing about it is unsafe to read.
			name:    "a later minor",
			version: bump(1, 1),
		},
		{
			name:    "a later patch",
			version: bump(2, 1),
		},
		{
			// The whole point of the field. A major means something that used
			// to validate no longer does, so what the fields mean cannot be
			// assumed from their names.
			name:    "another major is refused, and says both versions",
			version: bump(0, 1),
			refused: bump(0, 1),
		},
		{
			name:    "an earlier major is refused too",
			version: "1.0.0",
			refused: "1.0.0",
		},
	} {
		s.Run(tt.name, func() {
			_, err := tone.Load(strings.NewReader(document(tt.version)))

			if tt.refused == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tone.ErrInvalid)
			s.Require().ErrorContains(err, tt.refused)
			s.Require().ErrorContains(err, mine)
			s.Require().ErrorContains(err, "version")
		})
	}
}

// TestADocumentWithNoVersionIsRefused covers the field being required.
//
// The failure it prevents: a document that says nothing about which rules it
// was written under reads exactly like one written against these rules and
// wrong, and nothing can tell them apart.
func (s *VersionPublicTestSuite) TestADocumentWithNoVersionIsRefused() {
	for _, tt := range []struct {
		name string
		raw  string
	}{
		{
			name: "a ToneSpec",
			raw: `schema: ToneSpec
id: whoever
rig:
  instrument: bass
  chain:
    - {role: amp, gear: Ampeg SVT}
`,
		},
		{
			name: "a Setup",
			raw:  "schema: Setup\n",
		},
	} {
		s.Run(tt.name, func() {
			var err error

			if strings.Contains(tt.raw, "ToneSpec") {
				_, err = tone.Load(strings.NewReader(tt.raw))
			} else {
				_, err = tone.LoadSetup(strings.NewReader(tt.raw))
			}

			s.Require().ErrorIs(err, tone.ErrInvalid)
			s.Require().ErrorContains(err, "version")
		})
	}
}

// TestVersionOfSaysWhenThereIsNoVersionToFind covers reading a contract that
// does not say which version it is.
//
// Neither case is reachable through the embedded contract, which parses and
// states 4.0.0. Both describe a binary built without it, and the answer that
// matters is that nothing guesses: a reader with no version of its own cannot
// hold a document to one, so it refuses rather than reading the document
// against a version nobody knows.
//
// One method and one table, so a case is a row rather than a file.
func (s *VersionPublicTestSuite) TestVersionOfSaysWhenThereIsNoVersionToFind() {
	for _, tt := range []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "the contract this binary carries",
			raw:  string(tone.Schema),
		},
		{
			name: "a schema stating no version",
			raw:  "openapi: 3.0.3\ninfo:\n  title: ToneSpec\n",
			want: "states no version",
		},
		{
			name: "an info block with an empty one",
			raw:  "info:\n  version: \"\"\n",
			want: "states no version",
		},
		{
			name: "something that is not a schema at all",
			raw:  "\tnot: [yaml",
			want: "reading the ToneSpec schema's version",
		},
	} {
		s.Run(tt.name, func() {
			got, err := tone.VersionOf([]byte(tt.raw))

			if tt.want == "" {
				s.Require().NoError(err)
				s.Require().Regexp(semver, got)

				return
			}

			s.Require().ErrorContains(err, tt.want)
			s.Require().Empty(got)
		})
	}
}

// TestSameContractAndMajorAnswerWhatNoDocumentCanAsk covers the two branches a
// document cannot reach, because the schema refuses what would reach them.
//
// One method and one table, so a case is a row rather than a file.
func (s *VersionPublicTestSuite) TestSameContractAndMajorAnswerWhatNoDocumentCanAsk() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A version with no dot in it. The pattern on the field refuses
			// one, so this is the answer for a caller that is not a document.
			name: "a version that is not major.minor.patch",
			then: func() {
				s.Require().Equal("3", tone.Major("3"))
				s.Require().Equal("", tone.Major(""))
			},
		},
		{
			// The embedded contract cannot fail to parse, so this is what a
			// binary built without it would do: refuse the document rather
			// than read it against a version nobody knows.
			name: "no version to compare against",
			then: func() {
				was := tone.Version
				tone.Version = func() (string, error) {
					return "", errors.New("built without the contract")
				}

				defer func() { tone.Version = was }()

				s.Require().ErrorContains(
					tone.SameContract(
						map[string]any{"version": "4.0.0"}, "ToneSpec",
					),
					"built without the contract",
				)
			},
		},
		{
			name: "something that is not a set of fields at all",
			then: func() {
				s.Require().NoError(tone.SameContract("not a document", "ToneSpec"))
			},
		},
		{
			name: "a version that is not a string",
			then: func() {
				s.Require().NoError(tone.SameContract(
					map[string]any{"version": 3}, "ToneSpec",
				))
			},
		},
	} {
		s.Run(tt.name, tt.then)
	}
}

func TestVersionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VersionPublicTestSuite))
}
