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

package audio_test

import (
	"fmt"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

type FingerprintPublicTestSuite struct {
	suite.Suite
}

// one is a manifest body holding a single record, which only has to parse as far
// as the walk cares: CorpusSHA hashes bytes and never reads a manifest's fields.
func one(
	artist string,
) string {
	return "artist: " + artist + "\ntracks:\n" + record("t", 180)
}

// two is the same manifest with a second record added, for the case where a
// manifest is edited in place.
func two(
	artist string,
) string {
	return one(artist) + record("u", 200)
}

// record is one entry, with the three fields a manifest is refused without.
func record(
	track string,
	seconds int,
) string {
	return fmt.Sprintf(
		"  - track: %s\n    url: https://open.spotify.com/track/%s\n"+
			"    year: 1994\n    seconds: %d\n",
		track, track, seconds)
}

// said wraps a manifest body the way fstest.MapFS wants it, so a tree of them
// reads as a list of paths rather than as a wall of conversions.
func said(
	body string,
) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(body)}
}

// TestCorpusSHA covers the fingerprint, which is what lets a packed measurement
// say which corpus it came from.
func (s *FingerprintPublicTestSuite) TestCorpusSHA() {
	base := fstest.MapFS{
		"bass/dirnt/corpus.yaml":    said(one("Mike Dirnt")),
		"bass/shepherd/corpus.yaml": said(one("Ben Shepherd")),
	}

	first, err := audio.CorpusSHA(base, ".")
	s.Require().NoError(err)

	tests := []struct {
		name string
		tree fstest.MapFS
		same bool
		why  string
	}{
		{
			name: "the same manifests fingerprint the same",
			tree: fstest.MapFS{
				"bass/dirnt/corpus.yaml":    said(one("Mike Dirnt")),
				"bass/shepherd/corpus.yaml": said(one("Ben Shepherd")),
			},
			same: true,
			why:  "nothing about the records changed",
		},
		{
			name: "a record added changes it",
			tree: fstest.MapFS{
				"bass/dirnt/corpus.yaml":    said(two("Mike Dirnt")),
				"bass/shepherd/corpus.yaml": said(one("Ben Shepherd")),
			},
			why: "a record added is a different corpus to measure over",
		},
		{
			name: "a player moved changes it, though no file's contents did",
			tree: fstest.MapFS{
				"guitar/dirnt/corpus.yaml":  said(one("Mike Dirnt")),
				"bass/shepherd/corpus.yaml": said(one("Ben Shepherd")),
			},
			why: "the instrument is the directory, so moving one re-files " +
				"every record it holds",
		},
		{
			name: "a player removed changes it",
			tree: fstest.MapFS{
				"bass/dirnt/corpus.yaml": said(one("Mike Dirnt")),
			},
			why: "a genre measured over fewer players is a different figure",
		},
		{
			name: "anything that is not a manifest is ignored",
			tree: fstest.MapFS{
				"bass/dirnt/corpus.yaml":    said(one("Mike Dirnt")),
				"bass/shepherd/corpus.yaml": said(one("Ben Shepherd")),
				// The audio and the stems. Neither ships, and two machines hold
				// different downloads of the same track, so a fingerprint they
				// moved would say nothing anybody could act on.
				"bass/dirnt/t.mp3":              said("not audio"),
				"bass/dirnt/stems/h/t/bass.wav": said("nor this"),
				"README.md":                     said("what is here"),
			},
			same: true,
			why:  "it fingerprints what ships, which is the manifests",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := audio.CorpusSHA(tt.tree, ".")
			s.Require().NoError(err)

			if tt.same {
				s.Require().Equal(first, got, tt.why)

				return
			}

			s.Require().NotEqual(first, got, tt.why)
		})
	}
}

// TestATreeThatIsNotThere is the one error this can report.
func (s *FingerprintPublicTestSuite) TestATreeThatIsNotThere() {
	_, err := audio.CorpusSHA(fstest.MapFS{}, "nowhere")
	s.Require().ErrorContains(err, "walking nowhere")
}

// TestTheShippedFiguresNameTheCorpusTheyWereMeasuredFrom is the check nothing
// had, and the reason this file exists.
//
// The audio is not in the repository and never will be, so no test can
// re-derive a figure in data/genres.json: the recordings are somebody else's.
// The manifests are committed, which makes this one claim checkable — that the
// packed figures were measured over the records this checkout holds — and it
// needs no audio, so it runs in continuous integration where the audio never is.
//
// **What it catches.** A manifest edited without regenerating. finishing.md
// calls step 5 the one that gets forgotten and says forgetting it is invisible:
// the measuring commands print the new numbers while the embedded file still
// holds the old ones, so a build aims where the corpus used to be. Nothing said
// so until now, because a pack and a manifest set that had drifted apart agreed
// with each other perfectly.
//
// **What it does not catch**, and no fingerprint can: a branch that is behind on
// the manifests. Its pack matches its own manifests, so this passes and the
// figures are still measured over a record somebody else has already removed.
// That is a fact about the branch rather than about the tree, so only git knows
// it. One happened: a branch missing a merge measured against a stem reading
// 947 Hz when nothing else in the corpus reads above 264 Hz, every check agreed,
// and an unrelated conflict is the only reason it was caught.
//
// Fixing it: run `just generate` on a machine holding the audio, which is the
// only thing that can re-measure, and commit what it writes.
func (s *FingerprintPublicTestSuite) TestTheShippedFiguresNameTheCorpusTheyWereMeasuredFrom() {
	want, err := audio.CorpusSHA(os.DirFS(audio.CorpusRelative), ".")
	s.Require().NoError(err)

	s.Require().Equal(want, audio.ShippedCorpus(),
		"data/genres.json was measured over a different set of manifests than "+
			"the ones committed here: re-run `just generate` on a machine "+
			"holding the audio and commit what it writes")
}

func TestFingerprintPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FingerprintPublicTestSuite))
}

// unreadable lists a tree faithfully and refuses to open one file in it.
//
// Which is the state a manifest deleted between the walk and the read leaves
// behind, and the one error [audio.CorpusSHA] reports that a well-formed tree
// cannot produce.
type unreadable struct {
	fs.FS
	at string
}

// Open refuses the one path, and answers for every other.
func (u unreadable) Open(
	name string,
) (fs.File, error) {
	if name == u.at {
		return nil, fs.ErrPermission
	}

	return u.FS.Open(name)
}

// ReadDir delegates, so the walk still finds the file it cannot then read.
//
// Deliberately not a ReadFileFS: fs.ReadFile prefers that interface where a
// filesystem offers one, and fstest.MapFS does, so the refusal above would
// never be consulted.
func (u unreadable) ReadDir(
	name string,
) ([]fs.DirEntry, error) {
	return fs.ReadDir(u.FS, name)
}

// TestAManifestTheWalkFindsAndCannotRead is the second of the two errors.
func (s *FingerprintPublicTestSuite) TestAManifestTheWalkFindsAndCannotRead() {
	tree := fstest.MapFS{
		"bass/dirnt/corpus.yaml": said(one("Mike Dirnt")),
	}

	_, err := audio.CorpusSHA(
		unreadable{FS: tree, at: "bass/dirnt/corpus.yaml"}, ".")
	s.Require().ErrorContains(err, "reading bass/dirnt/corpus.yaml")
}
