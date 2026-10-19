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

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
	"github.com/retr0h/toneharness/pkg/sdk/tone"

	. "github.com/retr0h/toneharness/pkg/cli"
)

// RecordPublicTestSuite covers writing a round of tuning onto a document.
type RecordPublicTestSuite struct {
	suite.Suite

	dir string
}

func (s *RecordPublicTestSuite) SetupTest() {
	s.dir = s.T().TempDir()
}

// document writes a ToneSpec to a file and answers with its path.
func (s *RecordPublicTestSuite) document(
	body string,
) string {
	at := filepath.Join(s.dir, "rig.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(body), 0o600))

	return at
}

// a rig with an ask already on it, which is the ordinary case.
const asked = `schema: ToneSpec
version: 4.0.0
id: test-rig
ask:
  genre:
    - grunge
  instrument: bass
  words:
    - term: mid-forward
rig:
  instrument: bass
  chain:
    - role: amp
      gear: Ampeg SVT
`

// TestRecord covers appending a round of asking to a document.
//
// One method and one table, so a case is a row rather than a file. The round
// is the one input nothing can recover later: everything else here is a
// measurement or an assertion, and this is the record of somebody asking for
// something and something moving in answer.
//
// No row for a document with no ask at all, which is the branch that makes the
// section. It writes a document the contract then refuses, because an absent
// genre serialises as a present null, and a row asserting that would lock the
// bug in. The fixture below carries a genre for the same reason.
func (s *RecordPublicTestSuite) TestRecord() {
	for _, tt := range []struct {
		name string
		at   func() string
		says string
		then func(at string)
	}{
		{
			// The round lands on the ask, which is where a round of asking
			// belongs, and the verdict is left absent because nothing here
			// can hear.
			name: "a document with an ask already on it",
			at:   func() string { return s.document(asked) },
			then: func(at string) {
				f, err := os.Open(at)
				s.Require().NoError(err)

				defer func() { _ = f.Close() }()

				got, err := tone.Load(f)
				s.Require().NoError(err)
				s.Require().NotNil(got.Ask)
				s.Require().NotNil(got.Ask.Corrections)
				s.Require().Len(*got.Ask.Corrections, 1)
			},
		},
		{
			// Append-only, so a second round joins the first rather than
			// replacing it. The plan holds the current settings; this holds
			// why they are what they are, and a round that overwrote the last
			// would lose the history that is the whole point.
			name: "a second round joins the first",
			at: func() string {
				at := s.document(asked)
				s.Require().NoError(Record(at, "punk", nil, nil, true))

				return at
			},
			then: func(at string) {
				f, err := os.Open(at)
				s.Require().NoError(err)

				defer func() { _ = f.Close() }()

				got, err := tone.Load(f)
				s.Require().NoError(err)
				s.Require().Len(*got.Ask.Corrections, 2)
			},
		},
		{
			// The document is read and then rewritten in place, so a file
			// that opens and will not reopen for writing is the one failure
			// between the two. The round is lost and the document is left as
			// it was, which is the right outcome and worth saying out loud.
			name: "a document that cannot be written back",
			at: func() string {
				at := s.document(asked)
				s.Require().NoError(os.Chmod(at, 0o400))

				return at
			},
			says: "rig.yaml",
		},
		{
			name: "a path that is not a file",
			at:   func() string { return filepath.Join(s.dir, "nothing-here.yaml") },
			says: "nothing-here.yaml",
		},
		{
			// A round of tuning is worth nothing written onto a document
			// nothing can read, and saying which file beats saying which
			// field.
			name: "a document that is not a ToneSpec",
			at:   func() string { return s.document("this: is not a tonespec\n") },
			says: "rig.yaml",
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			at := tt.at()

			// A path into a plan has a grammar the contract enforces:
			// "blocks[1].params.Bass" parses and a knob with no Setting
			// produces one that does not.
			err := Record(at, "grunge",
				[]solve.Step{{
					Knob: solve.Knob{Block: 1, Param: 1, Setting: "Bass"},
					By:   0.1, To: 0.6,
				}},
				map[audio.Figure]float64{audio.KeyMid: 0.02}, false)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			tt.then(at)
		})
	}
}

func TestRecordPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RecordPublicTestSuite))
}
