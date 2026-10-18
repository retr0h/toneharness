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
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
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
		"\n    url: https://open.spotify.com/track/x\n    year: 1994\n    seconds: 180\n"

	if band != "" {
		out += "    band: " + band + "\n"
	}

	if genres != "" {
		out += "    genres: [" + genres + "]\n    genres_by: " + by + "\n"
	}

	return out
}

// TestPlayers covers Players, which is every player the corpus names, with
// what their records say.
//
// One method and one table, so a case is a row rather than a file.
func (s *MusicviewPublicTestSuite) TestPlayers() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// What each player's manifest says.
			name: "players",
			then: func() {
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
						"punk, pop-punk", "llm"),
					// A second record with no genre, so the untagged count has something
					// to report. A record naming none counts towards no genre at all.
					"flea/corpus.yaml": one("Flea", "aeroplane", "", "", "") +
						"  - track: ethiopia\n    url: https://open.spotify.com/track/y\n" +
						"    year: 2016\n    seconds: 180\n",
					// A directory with no manifest, which is a player whose records
					// somebody is still choosing.
					"ben-shepherd/notes.txt": "nothing yet",
				})

				got, err := musicview.Players(fsys, ".", nil)
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

				s.Require().False(got[0].Rig, "nothing was handed in, so nobody has gear")
				s.Require().False(got[1].Rig)
			},
		},
		{
			// The join a genre that will not build turns on.
			//
			// A player with records and no rig still earns their genres
			// words, so the genre reads as usable and then has nothing to
			// build from. Five players sat like that until somebody asked why
			// a build failed.
			name: "players say who has no rig",
			then: func() {
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml":    many("Mike Dirnt", "punk", 1),
					"cone-mccaslin/corpus.yaml": many("Cone McCaslin", "punk", 1),
				})

				got, err := musicview.Players(fsys, ".", map[string]bool{"mike-dirnt": true})
				s.Require().NoError(err)
				s.Require().Len(got, 2)

				s.Require().Equal("cone-mccaslin", got[0].ID)
				s.Require().False(got[0].Rig, "records and nothing to build them with")

				s.Require().Equal("mike-dirnt", got[1].ID)
				s.Require().True(got[1].Rig)
			},
		},
		{
			// A typo failing loudly.
			//
			// The alternative is a genre counted short with nothing saying
			// why, which is the failure the strict manifest reader exists to
			// prevent.
			name: "an unreadable manifest stops",
			then: func() {
				fsys := corpus(map[string]string{
					"a/corpus.yaml": "artist: A\ntracks:\n  - trak: typo\n",
				})

				_, err := musicview.Players(fsys, ".", nil)
				s.Require().ErrorContains(err, "a/corpus.yaml")
			},
		},
		{
			// A corpus path nobody can read.
			name: "a tree that is not there",
			then: func() {
				_, err := musicview.Players(corpus(nil), "nowhere", nil)
				s.Require().ErrorContains(err, "nowhere")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestGenres covers Genres, which is every genre the corpus names, and how
// far each is from usable.
//
// One method and one table, so a case is a row rather than a file.
func (s *MusicviewPublicTestSuite) TestGenres() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A genre that is hollow.
			//
			// Joined on the directory rather than on the name a group
			// reports, because those differ: a group names "Mike Dirnt" and a
			// rig is mike-dirnt. Counting on the name would report every
			// genre as having gear for nobody.
			name: "genres count their geared players",
			then: func() {
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml":    many("Mike Dirnt", "punk", 3),
					"cone-mccaslin/corpus.yaml": many("Cone McCaslin", "punk", 3),
					"ben-shepherd/corpus.yaml":  many("Ben Shepherd", "grunge", 3),
				})

				got, err := musicview.Genres(fsys, ".", map[string]bool{"mike-dirnt": true})
				s.Require().NoError(err)
				s.Require().Len(got, 2)

				s.Require().Equal("grunge", got[0].Slug)
				s.Require().Equal(1, got[0].Artists)
				s.Require().Zero(got[0].Geared, "a genre earning words nothing can build")

				s.Require().Equal("punk", got[1].Slug)
				s.Require().Equal(2, got[1].Artists)
				s.Require().Equal(1, got[1].Geared, "one of the two has gear")
			},
		},
		{
			// The number somebody acts on.
			//
			// "Not usable" says nothing about what to do next. Two records
			// short of eight, or one player short of three, says which.
			name: "genres reports what each is short of",
			then: func() {
				// Six punk records from two players: short on both counts.
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml":   many("Mike Dirnt", "punk", 3),
					"matt-freeman/corpus.yaml": many("Matt Freeman", "punk", 3),
				})

				got, err := musicview.Genres(fsys, ".", nil)
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
			},
		},
		{
			// The threshold being met.
			name: "genres counts a usable one",
			then: func() {
				fsys := corpus(map[string]string{
					"a/corpus.yaml": many("A", "punk", 3),
					"b/corpus.yaml": many("B", "punk", 3),
					"c/corpus.yaml": many("C", "punk", 2),
				})

				got, err := musicview.Genres(fsys, ".", nil)
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().Equal(8, got[0].Records)
				s.Require().Equal(3, got[0].Artists)
				s.Require().True(got[0].Usable, "eight records from three players")
				s.Require().Zero(got[0].ShortRecords)
				s.Require().Zero(got[0].ShortArtists)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestBands covers Bands, which is every band the corpus names.
//
// One method and one table, so a case is a row rather than a file.
func (s *MusicviewPublicTestSuite) TestBands() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A band never reading as unsighted.
			//
			// Nobody labels a band, so the provenance count that means
			// something for a genre must not follow it across.
			name: "bands carry no threshold",
			then: func() {
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
			},
		},
		{
			// The path that points at nothing.
			//
			// Refused rather than answered empty, because the ordinary cause
			// is a wrong path and an empty table reads as a corpus that
			// exists and holds nothing.
			name: "an empty corpus is refused",
			then: func() {
				fsys := corpus(map[string]string{"notes.txt": "no players here"})

				for _, tt := range []struct {
					name string
					call func() error
				}{
					{"players", func() error { _, err := musicview.Players(fsys, ".", nil); return err }},
					{"genres", func() error { _, err := musicview.Genres(fsys, ".", nil); return err }},
					{"bands", func() error { _, err := musicview.Bands(fsys, "."); return err }},
					{"records", func() error { _, err := musicview.Records(fsys, "."); return err }},
				} {
					s.Run(tt.name, func() {
						err := tt.call()
						s.Require().ErrorIs(err, musicview.ErrNoCorpus)
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

// TestRecords covers Records, which is every recording the corpus names, and
// whether it has stems yet.
//
// One method and one table, so a case is a row rather than a file.
// wav is a WAV file's bytes, holding seconds of sound after a head of quiet.
//
// Written to a real file and read back, because the encoder needs somewhere to
// seek and a corpus in this package's tests is a map of bytes.
func (s *MusicviewPublicTestSuite) wav(
	quiet, sound float64,
) string {
	const rate = 48000

	samples := make([]float32, 0, int((quiet+sound)*rate))
	for range int(quiet * rate) {
		samples = append(samples, 0)
	}

	for i := range int(sound * rate) {
		samples = append(samples, float32(0.8*math.Sin(2*math.Pi*110*float64(i)/rate)))
	}

	at := filepath.Join(s.T().TempDir(), "stem.wav")
	s.Require().NoError(audio.Write(at, samples, rate))

	raw, err := os.ReadFile(at) //nolint:gosec // a file this test just wrote
	s.Require().NoError(err)

	return string(raw)
}

// TestLengths covers holding a separated recording to the length its link
// states.
//
// One method and one table, so a case is a row rather than a file.
func (s *MusicviewPublicTestSuite) TestLengths() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary answer, and what `off` is measured from. The stem
			// carries two seconds of quiet before the sound, which is where
			// the instrument comes in rather than padding, so it changes the
			// sounding part and not the comparison.
			name: "a stem is held to its own length, not its sounding part",
			then: func() {
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
						"punk", "person"),
					"mike-dirnt/stems/htdemucs/longview/bass.wav": s.wav(2, 3),
				})

				got, err := musicview.Lengths(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().True(got[0].Read)
				s.Require().Equal(180, got[0].States)
				s.Require().InDelta(5.0, got[0].Seconds, 0.05)
				s.Require().InDelta(3.0, got[0].Sounding, 0.1)
				s.Require().InDelta(5.0-180, got[0].Off, 0.05)
			},
		},
		{
			// A record between a fetch and a separation. Reported rather than
			// skipped, because that is an ordinary state and not a mismatch.
			name: "a record nobody has separated says so",
			then: func() {
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
						"punk", "person"),
				})

				got, err := musicview.Lengths(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().False(got[0].Read)
				s.Require().Contains(got[0].Why, "separated")
				s.Require().Zero(got[0].Seconds)
			},
		},
		{
			// A directory the separator made and did not fill, or a file that
			// is not audio. Both are reported against the record rather than
			// failing the whole run, because one bad file would otherwise
			// hide every good one.
			name: "a stem that will not decode is reported and the rest still read",
			then: func() {
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
						"punk", "person") +
						"  - track: holiday\n    url: https://open.spotify.com/track/y\n" +
						"    year: 2004\n    seconds: 232\n",
					"mike-dirnt/stems/htdemucs/longview/bass.wav": "not a wav at all",
					"mike-dirnt/stems/htdemucs/holiday/bass.wav":  s.wav(0, 1),
				})

				got, err := musicview.Lengths(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 2)

				s.Require().Equal("holiday", got[0].Track)
				s.Require().True(got[0].Read)

				s.Require().Equal("longview", got[1].Track)
				s.Require().False(got[1].Read)
				s.Require().Contains(got[1].Why, "longview")
			},
		},
		{
			// A guitar corpus, which separates with a different model into a
			// different part. The join has to find either.
			name: "a guitar stem is found too",
			then: func() {
				fsys := corpus(map[string]string{
					"a/corpus.yaml":                       one("A", "song", "B", "rock", "person"),
					"a/stems/htdemucs_6s/song/guitar.wav": s.wav(0, 2),
				})

				got, err := musicview.Lengths(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)
				s.Require().True(got[0].Read)
			},
		},
		{
			// Sorted the way the other tables are, so two runs read the same
			// and a diff between them is a change rather than a reshuffle.
			name: "sorted by instrument, then player, then track",
			then: func() {
				fsys := corpus(map[string]string{
					"bass/b-player/corpus.yaml": one("B", "song", "Band", "rock", "person"),
					"bass/a-player/corpus.yaml": one("A", "zzz", "Band", "rock", "person") +
						"  - track: aaa\n    url: https://open.spotify.com/track/y\n" +
						"    year: 2004\n    seconds: 100\n",
					"guitar/a-player/corpus.yaml": one("A", "song", "Band", "rock", "person"),
				})

				got, err := musicview.Lengths(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 4)

				s.Require().Equal([]string{"bass", "bass", "bass", "guitar"},
					[]string{
						got[0].Instrument, got[1].Instrument,
						got[2].Instrument, got[3].Instrument,
					})
				s.Require().Equal([]string{"aaa", "zzz", "song", "song"},
					[]string{got[0].Track, got[1].Track, got[2].Track, got[3].Track})
			},
		},
		{
			// A filesystem whose files cannot seek. A WAV header is read
			// before its samples are, so one that cannot is refused by name
			// rather than decoded into nonsense.
			name: "a stem that cannot be seeked is refused by name",
			then: func() {
				fsys := unseekable{corpus(map[string]string{
					"a/corpus.yaml":                  one("A", "song", "B", "rock", "person"),
					"a/stems/htdemucs/song/bass.wav": s.wav(0, 1),
				})}

				got, err := musicview.Lengths(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().False(got[0].Read)
				s.Require().Contains(got[0].Why, "bass.wav")
			},
		},
		{
			// A stem the walk can see and cannot read, which is a permission
			// or a vanished file between the two calls. Reported against the
			// record rather than stopping the run.
			name: "a stem that will not open is reported",
			then: func() {
				fsys := unopenable{corpus(map[string]string{
					"a/corpus.yaml":                  one("A", "song", "B", "rock", "person"),
					"a/stems/htdemucs/song/bass.wav": s.wav(0, 1),
				})}

				got, err := musicview.Lengths(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().False(got[0].Read)
				s.Require().Contains(got[0].Why, "opening")
			},
		},
		{
			name: "a tree with no manifests",
			then: func() {
				_, err := musicview.Lengths(corpus(map[string]string{}), ".")
				s.Require().ErrorIs(err, musicview.ErrNoCorpus)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// unseekable is a filesystem whose files can be read and not seeked.
//
// fstest.MapFS hands back files that seek, and so does a directory on disk, so
// this is the only way to reach what happens to one that does not.
type unseekable struct {
	fs.FS
}

func (u unseekable) Open(
	name string,
) (fs.File, error) {
	f, err := u.FS.Open(name)
	if err != nil {
		return nil, err
	}

	// The audio only. A directory still has to list itself, which is how the
	// manifests are found in the first place.
	if !strings.HasSuffix(name, ".wav") {
		return f, nil
	}

	return readOnly{f}, nil
}

// unopenable is a filesystem that admits a stem exists and will not hand it
// over, which is a permission or a file that went away between the two calls.
type unopenable struct {
	fstest.MapFS
}

// Stat answers, so the walk finds the stem. Without this the open below is
// never reached: fs.Stat falls back to opening a file on a filesystem that
// does not implement it, and the record would read as one nobody separated.
func (u unopenable) Stat(
	name string,
) (fs.FileInfo, error) {
	return u.MapFS.Stat(name)
}

func (u unopenable) Open(
	name string,
) (fs.File, error) {
	if strings.HasSuffix(name, ".wav") {
		return nil, fs.ErrPermission
	}

	return u.MapFS.Open(name)
}

// readOnly hides whatever else the file underneath can do.
type readOnly struct {
	fs.File
}

func (r readOnly) Read(
	p []byte,
) (int, error) {
	return r.File.Read(p)
}

func (s *MusicviewPublicTestSuite) TestRecords() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// What the manifest cannot say.
			//
			// A record named with no stems beside it is measured by nothing,
			// and the manifest looks complete either way.
			name: "records says which are separated",
			then: func() {
				fsys := corpus(map[string]string{
					"mike-dirnt/corpus.yaml": one("Mike Dirnt", "longview", "Green Day",
						"punk", "person") +
						"  - track: holiday\n    url: https://open.spotify.com/track/y\n" +
						"    year: 2004\n    seconds: 180\n",
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
			},
		},
		{
			// The other separator model.
			name: "a guitar stem counts",
			then: func() {
				fsys := corpus(map[string]string{
					"a/corpus.yaml":                       one("A", "song", "", "", ""),
					"a/stems/htdemucs_6s/song/guitar.wav": "",
				})

				got, err := musicview.Records(fsys, ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)
				s.Require().True(got[0].Separated)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
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
				"    year: 1994\n    seconds: 180\n    genres: [%s]\n    genres_by: llm\n",
			i, i, genre)
	}

	return out
}
