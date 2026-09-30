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
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// ManifestsPublicTestSuite covers reading every manifest under a tree.
type ManifestsPublicTestSuite struct {
	suite.Suite
}

// tree builds a corpus in memory, so a case reads beside its assertion.
func tree(
	files map[string]string,
) fstest.MapFS {
	out := fstest.MapFS{}
	for at, body := range files {
		out[at] = &fstest.MapFile{Data: []byte(body)}
	}

	return out
}

// TestManifests covers Manifests, which reads every manifest under a tree,
// wherever it sits in it.
//
// One method and one table, so a case is a row rather than a file.
func (s *ManifestsPublicTestSuite) TestManifests() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Why this walks rather than reads a directory.
			//
			// One root answers for every instrument at once, so a listing can
			// say what the whole tree holds instead of being told which
			// instrument to look at.
			name: "manifests finds them at either depth",
			then: func() {
				fsys := tree(map[string]string{
					"bass/mike-dirnt/corpus.yaml": "artist: Mike Dirnt\ntracks:\n  - track: t\n" +
						"    url: https://open.spotify.com/track/x\n    year: 1994\n",
					"guitar/somebody/corpus.yaml": "artist: Somebody\ntracks:\n  - track: u\n" +
						"    url: https://open.spotify.com/track/y\n    year: 1999\n",
				})

				got, err := audio.Manifests(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 2)

				// Sorted by instrument then player, so two runs read the same.
				s.Require().Equal("bass", got[0].Instrument)
				s.Require().Equal("mike-dirnt", got[0].ID)
				s.Require().Equal("Mike Dirnt", got[0].Artist)
				s.Require().Len(got[0].Tracks, 1)

				s.Require().Equal("guitar", got[1].Instrument)
				s.Require().Equal("somebody", got[1].ID)
			},
		},
		{
			// The shallower tree.
			//
			// The instrument is empty rather than guessed: the directory
			// above the player is the root itself, and nothing in the tree
			// says what it holds.
			name: "manifests pointed at one instrument",
			then: func() {
				fsys := tree(map[string]string{
					"mike-dirnt/corpus.yaml": "artist: Mike Dirnt\ntracks:\n  - track: t\n" +
						"    url: https://open.spotify.com/track/x\n    year: 1994\n",
				})

				got, err := audio.Manifests(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)
				s.Require().Empty(got[0].Instrument)
				s.Require().Equal("mike-dirnt", got[0].ID)
			},
		},
		{
			// The ordinary case.
			//
			// A player whose records somebody is still choosing has a
			// directory before it has a manifest, so this is not a fault.
			name: "a directory with no manifest is skipped",
			then: func() {
				fsys := tree(map[string]string{
					"bass/mike-dirnt/corpus.yaml": "artist: Mike Dirnt\ntracks:\n  - track: t\n" +
						"    url: https://open.spotify.com/track/x\n    year: 1994\n",
					"bass/ben-shepherd/notes.txt": "nothing yet",
				})

				got, err := audio.Manifests(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)
				s.Require().Equal("mike-dirnt", got[0].ID)
			},
		},
		{
			// Why this refuses rather than skips.
			//
			// A manifest with a typo counted short would leave a genre short
			// with nothing saying why, so the path is in the error.
			name: "an unreadable manifest names itself",
			then: func() {
				fsys := tree(map[string]string{
					"bass/a/corpus.yaml": "artist: A\ntracks:\n  - trak: typo\n",
				})

				_, err := audio.Manifests(fsys, ".")
				s.Require().ErrorContains(err, "bass/a/corpus.yaml")
			},
		},
		{
			// A path nobody can walk.
			name: "a tree that is not there",
			then: func() {
				_, err := audio.Manifests(tree(nil), "nowhere")
				s.Require().Error(err)
			},
		},
		{
			// A walk that finds nothing.
			//
			// Not an error here. Whether nothing is worth refusing depends on
			// what asked, so the caller decides rather than this.
			name: "an empty tree reads as empty",
			then: func() {
				got, err := audio.Manifests(tree(map[string]string{"notes.txt": "x"}), ".")
				s.Require().NoError(err)
				s.Require().Empty(got)
			},
		},
		{
			// The ordinary corpus.
			//
			// Every player in resources/music/bass shares an instrument, so
			// this is the comparison that actually runs rather than the
			// instrument one.
			name: "two players on one instrument sort by i d",
			then: func() {
				fsys := tree(map[string]string{
					"bass/mike-dirnt/corpus.yaml": "artist: Mike Dirnt\ntracks:\n  - track: t\n" +
						"    url: https://open.spotify.com/track/x\n    year: 1994\n",
					"bass/ben-shepherd/corpus.yaml": "artist: Ben Shepherd\ntracks:\n  - track: u\n" +
						"    url: https://open.spotify.com/track/y\n    year: 1991\n",
				})

				got, err := audio.Manifests(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 2)
				s.Require().Equal("ben-shepherd", got[0].ID)
				s.Require().Equal("mike-dirnt", got[1].ID)
			},
		},
		{
			// The unreadable file.
			name: "a manifest that will not open names itself",
			then: func() {
				fsys := tree(map[string]string{
					"bass/a/corpus.yaml": "artist: A\ntracks:\n  - track: t\n" +
						"    url: https://open.spotify.com/track/x\n    year: 1994\n",
				})

				_, err := audio.Manifests(refusing{fsys}, ".")
				s.Require().ErrorContains(err, "bass/a/corpus.yaml")
				s.Require().ErrorIs(err, fs.ErrPermission)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestGroupingKeepsWhatCountsAndDropsTheRest covers the hand-off to Genres.
//
// The directory is what joins a corpus to a rig, and grouping counts players by
// the name the manifest gives, so the two must not be confused.
func (s *ManifestsPublicTestSuite) TestGroupingKeepsWhatCountsAndDropsTheRest() {
	held := []audio.Held{
		{Instrument: "bass", ID: "mike-dirnt", Artist: "Mike Dirnt", Tracks: []audio.Record{
			{Track: "t", Genres: []string{"punk"}, GenresBy: audio.ByModel},
		}},
	}

	got := audio.Grouping(held)
	s.Require().Len(got, 1)
	s.Require().Equal("Mike Dirnt", got[0].Artist)
	s.Require().Len(got[0].Tracks, 1)

	// And it feeds the counting, which is the only reason it exists.
	genres := audio.Genres(got)
	s.Require().Len(genres, 1)
	s.Require().Equal([]string{"Mike Dirnt"}, genres[0].Who)
}

func TestManifestsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ManifestsPublicTestSuite))
}

// refusing is a tree that lists a manifest and then will not open it.
//
// A real one: a file the walk can see and the process cannot read. Nothing
// else reaches that branch, because a manifest fstest holds it also opens.
type refusing struct{ fs.FS }

func (r refusing) Open(
	name string,
) (fs.File, error) {
	if strings.HasSuffix(name, "corpus.yaml") {
		return nil, fs.ErrPermission
	}

	return r.FS.Open(name)
}
