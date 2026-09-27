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

package musicview_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/musicview"
)

// MusicviewPublicTestSuite covers what the corpus says about itself.
type MusicviewPublicTestSuite struct {
	suite.Suite
}

// corpus is a tree of manifests, built in memory so a case reads beside its
// assertion rather than in a testdata directory.
func corpus(
	files map[string]string,
) fstest.MapFS {
	out := fstest.MapFS{}
	for at, body := range files {
		out[at] = &fstest.MapFile{Data: []byte(body)}
	}

	return out
}

// one is a manifest naming a single record.
func one(
	artist, track, band, genres, by string,
) string {
	out := "artist: " + artist + "\ntracks:\n  - track: " + track +
		"\n    url: https://open.spotify.com/track/x\n    year: 1994\n"

	if band != "" {
		out += "    band: " + band + "\n"
	}

	if genres != "" {
		out += "    genres: [" + genres + "]\n    genres_by: " + by + "\n"
	}

	return out
}

// TestPlayers covers what each player's manifest says.
func (s *MusicviewPublicTestSuite) TestPlayers() {
	fsys := corpus(map[string]string{
		"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
			"punk, pop-punk", "llm"),
		// A second record with no genre, so the untagged count has something
		// to report. A record naming none counts towards no genre at all.
		"flea/corpus.yaml": one("Flea", "aeroplane", "", "", "") +
			"  - track: ethiopia\n    url: https://open.spotify.com/track/y\n" +
			"    year: 2016\n",
		// A directory with no manifest, which is a player whose records
		// somebody is still choosing.
		"ben-shepherd/notes.txt": "nothing yet",
	})

	got, err := musicview.Players(fsys, ".")
	s.Require().NoError(err)
	s.Require().Len(got, 2, "the directory with no manifest is skipped")

	s.Require().Equal("flea", got[0].ID)
	s.Require().Equal(2, got[0].Records)
	s.Require().Empty(got[0].Genres)
	s.Require().Equal(2, got[0].Untagged)

	s.Require().Equal("mike-dirnt", got[1].ID)
	s.Require().Equal("Mike Dirnt", got[1].Artist)
	s.Require().Equal([]string{"Green Day"}, got[1].Bands)
	s.Require().Equal([]string{"pop-punk", "punk"}, got[1].Genres, "in order")
	s.Require().Zero(got[1].Untagged)
}

// TestGenresReportsWhatEachIsShortOf is the number somebody acts on.
//
// "Not usable" says nothing about what to do next. Two records short of eight,
// or one player short of three, says which.
func (s *MusicviewPublicTestSuite) TestGenresReportsWhatEachIsShortOf() {
	// Six punk records from two players: short on both counts.
	fsys := corpus(map[string]string{
		"mike-dirnt/corpus.yaml":   many("Mike Dirnt", "punk", 3),
		"matt-freeman/corpus.yaml": many("Matt Freeman", "punk", 3),
	})

	got, err := musicview.Genres(fsys, ".")
	s.Require().NoError(err)
	s.Require().Len(got, 1)

	s.Require().Equal("punk", got[0].Slug)
	s.Require().Equal(6, got[0].Records)
	s.Require().Equal(2, got[0].Artists)
	s.Require().False(got[0].Usable)
	s.Require().Equal(2, got[0].ShortRecords)
	s.Require().Equal(1, got[0].ShortArtists)

	// Every record here was tagged by a model, so a genre can clear the
	// threshold with nobody having checked any of it.
	s.Require().Equal(6, got[0].Unsighted)
}

// TestGenresCountsAUsableOne covers the threshold being met.
func (s *MusicviewPublicTestSuite) TestGenresCountsAUsableOne() {
	fsys := corpus(map[string]string{
		"a/corpus.yaml": many("A", "punk", 3),
		"b/corpus.yaml": many("B", "punk", 3),
		"c/corpus.yaml": many("C", "punk", 2),
	})

	got, err := musicview.Genres(fsys, ".")
	s.Require().NoError(err)
	s.Require().Len(got, 1)

	s.Require().Equal(8, got[0].Records)
	s.Require().Equal(3, got[0].Artists)
	s.Require().True(got[0].Usable, "eight records from three players")
	s.Require().Zero(got[0].ShortRecords)
	s.Require().Zero(got[0].ShortArtists)
}

// TestBandsCarryNoThreshold covers a band never reading as unsighted.
//
// Nobody labels a band, so the provenance count that means something for a
// genre must not follow it across.
func (s *MusicviewPublicTestSuite) TestBandsCarryNoThreshold() {
	fsys := corpus(map[string]string{
		"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
			"punk", "llm"),
		"other/corpus.yaml": one("Other", "song", "Green Day", "punk", "llm"),
	})

	got, err := musicview.Bands(fsys, ".")
	s.Require().NoError(err)
	s.Require().Len(got, 1)

	s.Require().Equal("Green Day", got[0].Name)
	s.Require().Equal(2, got[0].Records)
	s.Require().Equal(2, got[0].Artists)
	s.Require().Zero(got[0].Unsighted, "a band is nobody's label")
}

// TestRecordsSaysWhichAreSeparated is what the manifest cannot say.
//
// A record named with no stems beside it is measured by nothing, and the
// manifest looks complete either way.
func (s *MusicviewPublicTestSuite) TestRecordsSaysWhichAreSeparated() {
	fsys := corpus(map[string]string{
		"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
			"punk", "person") +
			"  - track: holiday\n    url: https://open.spotify.com/track/y\n" +
			"    year: 2004\n",
		"mike-dirnt/stems/htdemucs/longview/bass.wav": "",
	})

	got, err := musicview.Records(fsys, ".")
	s.Require().NoError(err)
	s.Require().Len(got, 2)

	s.Require().Equal("holiday", got[0].Track)
	s.Require().False(got[0].Separated)

	s.Require().Equal("longview", got[1].Track)
	s.Require().True(got[1].Separated)
	s.Require().Equal("person", got[1].DecidedBy)
	s.Require().Equal("Green Day", got[1].Band)
}

// TestAGuitarStemCounts covers the other separator model.
func (s *MusicviewPublicTestSuite) TestAGuitarStemCounts() {
	fsys := corpus(map[string]string{
		"a/corpus.yaml":                       one("A", "song", "", "", ""),
		"a/stems/htdemucs_6s/song/guitar.wav": "",
	})

	got, err := musicview.Records(fsys, ".")
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Require().True(got[0].Separated)
}

// TestAnEmptyCorpusIsRefused covers the path that points at nothing.
//
// Refused rather than answered empty, because the ordinary cause is a wrong
// path and an empty table reads as a corpus that exists and holds nothing.
func (s *MusicviewPublicTestSuite) TestAnEmptyCorpusIsRefused() {
	fsys := corpus(map[string]string{"notes.txt": "no players here"})

	for _, tt := range []struct {
		name string
		call func() error
	}{
		{"players", func() error { _, err := musicview.Players(fsys, "."); return err }},
		{"genres", func() error { _, err := musicview.Genres(fsys, "."); return err }},
		{"bands", func() error { _, err := musicview.Bands(fsys, "."); return err }},
		{"records", func() error { _, err := musicview.Records(fsys, "."); return err }},
	} {
		s.Run(tt.name, func() {
			err := tt.call()
			s.Require().ErrorIs(err, musicview.ErrNoCorpus)
		})
	}
}

// TestAnUnreadableManifestStops covers a typo failing loudly.
//
// The alternative is a genre counted short with nothing saying why, which is
// the failure the strict manifest reader exists to prevent.
func (s *MusicviewPublicTestSuite) TestAnUnreadableManifestStops() {
	fsys := corpus(map[string]string{
		"a/corpus.yaml": "artist: A\ntracks:\n  - trak: typo\n",
	})

	_, err := musicview.Players(fsys, ".")
	s.Require().ErrorContains(err, "a/corpus.yaml")
}

// TestATreeThatIsNotThere covers a corpus path nobody can read.
func (s *MusicviewPublicTestSuite) TestATreeThatIsNotThere() {
	_, err := musicview.Players(corpus(nil), "nowhere")
	s.Require().ErrorContains(err, "nowhere")
}

func TestMusicviewPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MusicviewPublicTestSuite))
}

// many is a manifest naming n records, all carrying one genre from a model.
func many(
	artist, genre string,
	n int,
) string {
	out := "artist: " + artist + "\ntracks:\n"
	for i := range n {
		out += fmt.Sprintf(
			"  - track: t%d\n    url: https://open.spotify.com/track/x%d\n"+
				"    year: 1994\n    genres: [%s]\n    genres_by: llm\n",
			i, i, genre)
	}

	return out
}
