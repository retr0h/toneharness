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
package rigs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/rigs"
	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// BackingPublicTestSuite covers holding a rig's records to the era it claims.
type BackingPublicTestSuite struct {
	suite.Suite
}

// read joins the fixture rigs to the fixture corpus.
func (s *BackingPublicTestSuite) read() map[string]result.Backing {
	got, err := rigs.Backing(
		rigs.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
		filepath.Join("testdata", "backing", "music"),
	)
	s.Require().NoError(err)

	out := map[string]result.Backing{}
	for _, b := range got {
		out[b.ID] = b
	}

	return out
}

// TestBacking covers Backing, which reads which records back each rig, and
// holds them to its era.
//
// One method and one table, so a case is a row rather than a file.
func (s *BackingPublicTestSuite) TestBacking() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The case nobody has to act on.
			name: "records inside the era",
			then: func() {
				got := s.read()["in-era"]

				s.Require().True(got.Stated())
				s.Require().Len(got.Records, 2)
				s.Require().Zero(got.Outside())

				for _, r := range got.Records {
					s.Require().False(r.Outside)
				}
			},
		},
		{
			// The finding this exists for. The rig describes 2004 and one
			// record is from 1994, so the figures measured from it describe
			// gear the rig does not name.
			name: "a record outside the era",
			then: func() {
				got := s.read()["out-of-era"]

				s.Require().Equal(1, got.Outside())
				s.Require().True(got.Records[0].Outside, "1994 against a 2004 rig")
				s.Require().False(got.Records[1].Outside, "2004 against a 2004 rig")
			},
		},
		{
			// What cannot be checked, reported rather than passed over: a rig
			// with no years is a rig nothing can hold its records to, which is
			// worth seeing beside the ones that can.
			name: "a rig that states no era",
			then: func() {
				got := s.read()["no-years"]

				s.Require().False(got.Stated())
				s.Require().Zero(got.Outside(), "nothing to be outside of")
			},
		},
		{
			// The ordinary case. Most players have gear evidence long before
			// anybody owns their records, so a rig with no corpus is not a
			// fault.
			name: "a rig nobody has measured",
			then: func() {
				s.Require().Empty(s.read()["no-years"].Records)
			},
		},
		{
			// The join failing quietly. A rig reaches its records by the
			// directory carrying its identifier. A directory called anything
			// else reads exactly like a rig nobody has measured yet, so the
			// typo survives until something says which it is.
			name: "records no rig is named for",
			then: func() {
				got := s.read()["mccartney"]

				s.Require().True(got.NoRig)
				s.Require().NotEmpty(got.Records, "the records are there, and nobody claims them")
				s.Require().Zero(got.Outside(), "there is no era to be outside of")
			},
		},
		{
			name: "a rig is not its own orphan",
			then: func() {
				for _, id := range []string{"in-era", "out-of-era", "no-years"} {
					s.Require().False(s.read()[id].NoRig, id)
				}
			},
		},
		{
			// The join a genre needs. A rig for a person reaches its records
			// through the directory carrying its identifier. A genre has no
			// directory and never will: its records are other people's,
			// sitting under the players who made them, and the genre tag is
			// what joins them to it.
			//
			// Without this a genre rig reads "nothing measured for it", which
			// is the opposite of true, because measurement is its only
			// evidence.
			name: "a rig for a genre reaches its records by their tags",
			then: func() {
				got := s.read()["thrash"]

				s.Require().False(got.NoRig)
				s.Require().Len(got.Records, 2, "one record from each of two players")
				s.Require().False(got.Stated(), "a genre claims no years")
				s.Require().Zero(got.Outside())

				// Nothing in the corpus is called thrash, so a directory join
				// would have found none of these.
				s.Require().NoDirExists(filepath.Join("testdata", "backing", "music", "thrash"))
			},
		},
		{
			// The second kind of wrong-era mistake. The era check asks whether
			// the records were made when the gear was; this asks whether they
			// were made through it. Five of the nine rigs that ship measure a
			// signal that went to the desk, and every one of them ends in a
			// cabinet, so a figure read off those records was not shaped by
			// the box the preset builds.
			name: "a signal that never met a microphone",
			then: func() {
				got := s.read()["went-direct"]

				s.Require().Equal(2, got.Direct)
				s.Require().Equal(2, got.Captured)
				s.Require().Zero(got.Both)

				s.Require().Equal(1, got.Stage,
					"a rundown photographs a backline and the corpus measures records")
			},
		},
		{
			// The third answer. Jaco Pastorius took "a little bit of both, the
			// highs and lows", which is neither of the other two and must not
			// be counted as direct.
			name: "a direct and a microphone at once",
			then: func() {
				got := s.read()["took-both"]

				s.Require().Equal(1, got.Both)
				s.Require().Zero(got.Direct)
				s.Require().Equal(2, got.Captured,
					"the miked entry is established too, and says so")
			},
		},
		{
			// Silence, which is not the same as miked. A rig nobody has asked
			// the question of reads as miked unless the count of answers is
			// kept separately, and that would turn an open question into a
			// claim.
			name: "nobody established the room",
			then: func() {
				got := s.read()["in-era"]

				s.Require().Zero(got.Captured)
				s.Require().Zero(got.Direct)
				s.Require().Zero(got.Stage)
			},
		},
		{
			// The join `played.records` borrows, and the way it fails. An
			// instrument claims the records it made by their track names, the
			// same join the corpus directory makes. A name matching nothing
			// attributes a figure to nothing, and it reads exactly like an
			// instrument nobody has got to yet.
			name: "an instrument naming a record nobody has",
			then: func() {
				s.Require().Equal([]string{"a-track-nobody-has"}, s.read()["misnamed"].Misnamed)
			},
		},
		{
			name: "an instrument naming records that exist",
			then: func() {
				s.Require().Empty(s.read()["in-era"].Misnamed)
			},
		},
		{
			// A directory somebody made and has not filled, which claims
			// nothing and is nobody's problem.
			//
			// Built here rather than kept in testdata, because git does not
			// track an empty directory: as a fixture this passed locally and
			// never ran anywhere else, which is the kind of test that reports
			// coverage it does not have.
			name: "a directory with no manifest",
			then: func() {
				corpus := s.T().TempDir()

				s.Require().NoError(os.MkdirAll(filepath.Join(corpus, "empty-dir"), 0o750))
				s.Require().NoError(os.MkdirAll(filepath.Join(corpus, "mccartney"), 0o750))

				named, err := os.ReadFile(
					filepath.Join("testdata", "backing", "music", "mccartney", "corpus.yaml"))
				s.Require().NoError(err)
				s.Require().NoError(os.WriteFile(
					filepath.Join(corpus, "mccartney", "corpus.yaml"), named, 0o600))

				got, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "backing", "rigs")}, corpus)
				s.Require().NoError(err)

				seen := map[string]bool{}
				for _, b := range got {
					seen[b.ID] = true
				}

				s.Require().True(seen["mccartney"], "records nobody's rig is named for")
				s.Require().False(seen["empty-dir"], "nothing in it to report")
			},
		},
		{
			// A directory no rig claims whose manifest is broken, which is
			// reported rather than passed over.
			name: "an orphan manifest that will not read",
			then: func() {
				_, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
					filepath.Join("testdata", "backing", "brokenorphan"),
				)

				s.Require().Error(err)
			},
		},
		{
			// The corpus argument naming a file, which is caught reading the
			// rigs' own records.
			name: "a corpus directory that is a file",
			then: func() {
				_, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
					filepath.Join("testdata", "backing", "notadir", "in-era"),
				)

				s.Require().Error(err)
			},
		},
		{
			// The same argument reaching the scan for directories nobody
			// claims, which is the other way in.
			name: "a corpus that is a file with no rigs to read",
			then: func() {
				_, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "backing", "notadir")},
					filepath.Join("testdata", "backing", "notadir", "in-era"),
				)

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "in-era")
			},
		},
		{
			// The path being wrong.
			//
			// A directory nobody has is the same as a player nobody has
			// measured, so it reports rather than fails: the rigs still read.
			// The genre join answers the same way, and did not until it was
			// made to.
			name: "a corpus that is not there",
			then: func() {
				got, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
					filepath.Join("testdata", "backing", "nowhere"),
				)

				s.Require().NoError(err)
				s.Require().NotEmpty(got)

				for _, b := range got {
					s.Require().Empty(b.Records)
				}
			},
		},
		{
			// A corpus somebody broke.
			name: "a manifest that will not read",
			then: func() {
				_, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
					filepath.Join("testdata", "backing", "broken"),
				)

				s.Require().Error(err)
			},
		},
		{
			// Reading a corpus with no rigs to read it against.
			//
			// Not an empty answer: the records are there and nothing claims
			// them, which is the same thing as a misspelt directory and reads
			// the same way.
			name: "a rig directory nobody has",
			then: func() {
				got, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "nowhere")},
					filepath.Join("testdata", "backing", "music"),
				)

				s.Require().NoError(err)
				s.Require().NotEmpty(got)

				for _, b := range got {
					s.Require().True(b.NoRig, b.ID)
				}
			},
		},
		{
			// A corpus argument naming a file.
			name: "a corpus path that is not a directory",
			then: func() {
				_, err := rigs.Backing(
					rigs.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
					filepath.Join("testdata", "backing", "notadir"),
				)

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "in-era")
			},
		},
		{
			// A rig directory holding a broken file.
			//
			// A half-read knowledge base is worse than a clear complaint
			// about the file to fix, which is what Load does, and this
			// reports it rather than answering about the rigs that happened
			// to parse.
			name: "a rig that will not read",
			then: func() {
				_, err := rigs.Backing(
					rigs.Source{Dir: "testdata"},
					filepath.Join("testdata", "backing", "music"),
				)

				s.Require().Error(err)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestBackingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BackingPublicTestSuite))
}
