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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// ManifestPublicTestSuite covers the record of what a corpus was.
type ManifestPublicTestSuite struct {
	suite.Suite
}

// read parses a manifest the way a caller would.
func (s *ManifestPublicTestSuite) read(
	doc string,
) audio.Manifest {
	got, err := audio.ReadManifest(strings.NewReader(doc))
	s.Require().NoError(err)

	return got
}

// full is a manifest with something in every field.
const full = `
artist: Mike Dirnt
tracks:
  - track: longview
    url: https://open.spotify.com/track/abc
    year: 1994
    source: https://www.youtube.com/watch?v=abc
    at: "1:20-1:45"
    note: the bass carries the verse alone
  - track: basket-case
    url: https://open.spotify.com/track/def
    year: 1994
`

// TestItReadsWhatWasMeasured covers the whole shape.
func (s *ManifestPublicTestSuite) TestItReadsWhatWasMeasured() {
	got := s.read(full)

	s.Require().Equal("Mike Dirnt", got.Artist)
	s.Require().Len(got.Tracks, 2)

	s.Require().Equal("longview", got.Tracks[0].Track)
	s.Require().Equal("https://open.spotify.com/track/abc", got.Tracks[0].URL)
	s.Require().Equal("https://www.youtube.com/watch?v=abc", got.Tracks[0].Source)
	s.Require().Equal("1:20-1:45", got.Tracks[0].At)
	s.Require().Equal("the bass carries the verse alone", got.Tracks[0].Note)
}

// TestASourceIsOptional covers the ordinary record, which needs no fallback.
//
// Most links download from the url alone. A source is only written down when
// that failed once and somebody found what did work.
func (s *ManifestPublicTestSuite) TestASourceIsOptional() {
	got := s.read(full)

	s.Require().Empty(got.Tracks[1].Source)
}

// TestReadManifestRefusesWhatItCannotTrust covers every manifest that is not a manifest, and the two things one may say beyond its records.
//
// One method and one table, so a case is a row rather than a file.
func (s *ManifestPublicTestSuite) TestReadManifestRefusesWhatItCannotTrust() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// the link it stands in for.
			name: "a bad source is caught here",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    url: https://open.spotify.com/track/abc\n" +
						"    year: 1994\n    source: watch?v=abc\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "longview")
				s.Require().Contains(err.Error(), "source")
			},
		},
		{
			// measures nothing is worse than one that refuses.
			name: "a typo stops",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader("artist: x\ntrack:\n  - track: y\n"))

				s.Require().Error(err)
			},
		},
		{
			// which says nothing about which song was wrong.
			name: "a bad timestamp is caught here",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    url: https://open.spotify.com/track/abc\n" +
						"    year: 1994\n    at: \"about a minute in\"\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "longview")
				s.Require().Contains(err.Error(), "timestamp")
			},
		},
		{
			// evidence nobody can trace is an assertion with numbers on it. Three players
			// carried three tracks each with no links between them before this was
			// refused, and nothing said so.
			name: "a record with no link is refused",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    year: 1994\n    note: no link\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "longview")
				s.Require().Contains(err.Error(), "no url")
			},
		},
		{
			// measures another rig. Without the year nothing can say so, and four of the
			// nine rigs here turned out to be measuring records from the wrong decade.
			name: "a record with no year is refused",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    url: https://open.spotify.com/track/abc\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "longview")
				s.Require().Contains(err.Error(), "no year")
			},
		},
		{
			// take apart from the live one and the remaster. Anything else names a file,
			// and the mistake it prevents is invisible once the audio is on disk and
			// measuring fine.
			name: "a url somewhere else is refused",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    year: 1994\n" +
						"    url: https://example.com/longview.mp3\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "longview")
				s.Require().Contains(err.Error(), "Spotify")
			},
		},
		{
			// same song is up there as the album take, a live take and three lyric
			// videos.
			name: "a you tube url is refused",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    year: 1994\n" +
						"    url: https://www.youtube.com/watch?v=abc\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "Spotify")
			},
		},
		{
			name: "a source somewhere else is refused",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    year: 1994\n" +
						"    url: https://open.spotify.com/track/abc\n" +
						"    source: https://example.com/longview.mp3\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "longview")
				s.Require().Contains(err.Error(), "YouTube")
			},
		},
		{
			// malformed host walks straight past it: "http://[::1" is missing the bracket
			// that closes an IPv6 address. It reaches the host check, which cannot parse
			// it and therefore cannot match it against anything.
			name: "a url the parser cannot read is refused",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    year: 1994\n" +
						"    url: \"http://[::1\"\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "Spotify")
			},
		},
		{
			name: "a bad link is caught here",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - track: longview\n    year: 1994\n    url: spotify:track:abc\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "longview")
			},
		},
		{
			name: "an entry with no track stops",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader(
					"tracks:\n  - url: https://example.com/a\n"))

				s.Require().Error(err)
			},
		},
		{
			name: "it is not yaml",
			then: func() {
				_, err := audio.ReadManifest(strings.NewReader("\tnot: [a manifest"))

				s.Require().Error(err)
			},
		},
		{
			name: "a manifest may carry genres",
			then: func() {
				m, err := audio.ReadManifest(strings.NewReader(`
artist: Mike Dirnt
tracks:
  - track: longview
    url: https://open.spotify.com/track/x
    year: 1994
    genres: [punk, pop-punk]
    genres_by: person
`))

				s.Require().NoError(err)
				s.Require().Equal([]string{"punk", "pop-punk"}, m.Tracks[0].Genres)
				s.Require().Equal(audio.ByPerson, m.Tracks[0].GenresBy)
			},
		},
		{
			// word in a list, and the review list exists to tell them apart.
			name: "a genre says who decided it",
			then: func() {
				tests := []struct {
					name string
					body string
					want string
				}{
					{
						name: "genres with nobody behind them",
						body: "    genres: [punk]\n",
						want: "names genres and no genres_by",
					},
					{
						name: "a decider with nothing decided",
						body: "    genres_by: llm\n",
						want: "names no genres for it to have decided",
					},
					{
						name: "a decider that is neither",
						body: "    genres: [punk]\n    genres_by: vibes\n",
						want: `has a genres_by of "vibes"`,
					},
					{
						name: "neither, which is every record written before the field",
						body: "",
						want: "",
					},
					{
						name: "a model's guess",
						body: "    genres: [punk]\n    genres_by: llm\n",
						want: "",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						_, err := audio.ReadManifest(strings.NewReader(`
artist: Mike Dirnt
tracks:
  - track: longview
    url: https://open.spotify.com/track/x
    year: 1994
` + tt.body))

						if tt.want == "" {
							s.Require().NoError(err)

							return
						}

						s.Require().ErrorContains(err, tt.want)
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestItNamesNoFiles is the point of a manifest rather than a directory
// listing.
//
// The audio is somebody else's and cannot be committed. What can is the
// record of which songs were measured, and that record is worth nothing if it
// only works on the machine that holds them.
func (s *ManifestPublicTestSuite) TestItNamesNoFiles() {
	for _, rec := range s.read(full).Tracks {
		s.Require().NotContains(rec.URL, "/Users/")
		s.Require().NotContains(rec.URL, ".wav")
	}
}

// TestATypoStops covers a field nobody meant to write.
//

// TestABadTimestampIsCaughtHere covers the check happening where it can be
// acted on.
//

// TestARecordWithNoLinkIsRefused covers the reason a manifest exists.
//

// TestARecordWithNoYearIsRefused covers holding a record to an era.
//

// TestAUrlSomewhereElseIsRefused covers the link naming a recording rather
// than a copy of one.
//

// TestAYouTubeUrlIsRefused covers the fallback being offered as the evidence.
//

// TestASourceFromYouTubeIsAccepted covers the ordinary fallback.
func (s *ManifestPublicTestSuite) TestASourceFromYouTubeIsAccepted() {
	got := s.read(
		"tracks:\n  - track: longview\n    year: 1994\n" +
			"    url: https://open.spotify.com/track/abc\n" +
			"    source: https://www.youtube.com/watch?v=abc\n")

	s.Require().Equal("https://www.youtube.com/watch?v=abc", got.Tracks[0].Source)
}

// TestAUrlTheParserCannotReadIsRefused covers a link that looks like one and
// is not.
//

// TestJoinAttachesTheSourceToEveryReading covers matching readings to the records they came from.
//
// One method and one table, so a case is a row rather than a file.
func (s *ManifestPublicTestSuite) TestJoinAttachesTheSourceToEveryReading() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "join attaches the source",
			then: func() {
				got := s.read(full).Join([]audio.Named{
					{Name: "longview"},
					{Name: "brain-stew"},
				})

				s.Require().Equal("https://open.spotify.com/track/abc", got[0].Source.URL)
				s.Require().Equal("1:20-1:45", got[0].Source.At)

				s.Require().Empty(got[1].Source.URL,
					"a measurement the manifest does not mention is still a measurement")
			},
		},
		{
			name: "join keeps everything",
			then: func() {
				in := []audio.Named{{Name: "longview"}, {Name: "nothing-named-this"}}

				s.Require().Len(s.read(full).Join(in), len(in))
			},
		},
		{
			name: "join ignores case",
			then: func() {
				got := s.read(
					"tracks:\n  - track: LongView\n    year: 1994\n    url: https://open.spotify.com/track/ghi\n",
				).
					Join([]audio.Named{{Name: "longview"}})

				s.Require().Equal("https://open.spotify.com/track/ghi", got[0].Source.URL)
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestUnmatchedReportsWhatDoesNotLineUp covers a manifest and a tree that disagree.
//
// One method and one table, so a case is a row rather than a file.
func (s *ManifestPublicTestSuite) TestUnmatchedReportsWhatDoesNotLineUp() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "unmatched reports both directions",
			then: func() {
				missing, unnamed := s.read(full).Unmatched([]audio.Named{
					{Name: "longview"},
					{Name: "brain-stew"},
				})

				s.Require().Equal([]string{"basket-case"}, missing,
					"named in the manifest, nothing measured")
				s.Require().Equal([]string{"brain-stew"}, unnamed,
					"measured, and its evidence will go out with no link")
			},
		},
		{
			name: "unmatched is quiet when they agree",
			then: func() {
				missing, unnamed := s.read(full).Unmatched([]audio.Named{
					{Name: "longview"},
					{Name: "basket-case"},
				})

				s.Require().Empty(missing)
				s.Require().Empty(unnamed)
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestGenresCountsRecordsAndPlayers covers what decides whether a genre can be
// aimed at.
//
// TestGenresCountsWhatWasMeasured covers pooling readings by genre.
//
// One method and one table, so a case is a row rather than a file.
func (s *ManifestPublicTestSuite) TestGenresCountsWhatWasMeasured() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// sound wearing its name, and the figures cannot tell those apart, so the count
			// is the only thing that can.
			name: "genres counts records and players",
			then: func() {
				of := func(artist string, genres ...[]string) audio.Manifest {
					m := audio.Manifest{Artist: artist}
					for i, g := range genres {
						m.Tracks = append(m.Tracks, audio.Record{
							Track: fmt.Sprintf("t%d", i), Genres: g,
						})
					}

					return m
				}

				punk := []string{"punk", "pop-punk"}

				all := []audio.Manifest{
					of("A", punk, punk, punk),
					of("B", punk, punk, punk),
					of("C", punk, punk, []string{"grunge"}),
					of("D", []string{"grunge"}),
				}

				got := audio.Genres(all)

				// Name order, so two runs read the same.
				names := make([]string, 0, len(got))
				for _, g := range got {
					names = append(names, g.Slug)
				}

				s.Require().Equal([]string{"grunge", "pop-punk", "punk"}, names)

				by := map[string]audio.Grouped{}
				for _, g := range got {
					by[g.Slug] = g
				}

				s.Require().Equal(8, by["punk"].Records)
				s.Require().Equal(3, by["punk"].Artists)
				s.Require().True(by["punk"].Usable(), "eight records from three players")

				// Two records from two players. Enough players, not enough records.
				s.Require().Equal(2, by["grunge"].Records)
				s.Require().Equal(2, by["grunge"].Artists)
				s.Require().False(by["grunge"].Usable())
			},
		},
		{
			// than deducted: the count is still eight, and somebody should know none of it
			// was checked.
			name: "a models genres are counted",
			then: func() {
				all := []audio.Manifest{
					{Artist: "Mike Dirnt", Tracks: []audio.Record{
						{Track: "a", Genres: []string{"punk"}, GenresBy: audio.ByModel},
						{Track: "b", Genres: []string{"punk"}, GenresBy: audio.ByPerson},
					}},
					{Artist: "Matt Freeman", Tracks: []audio.Record{
						{Track: "c", Genres: []string{"punk"}, GenresBy: audio.ByModel},
						// A band carries no provenance, so it never counts as unsighted.
						{
							Track: "d", Band: "Rancid", GenresBy: audio.ByModel,
							Genres: []string{"punk"},
						},
					}},
				}

				got := audio.Genres(all)
				s.Require().Len(got, 1)
				s.Require().Equal(4, got[0].Records)
				s.Require().Equal(3, got[0].Unsighted, "one of the four was checked")

				bands := audio.Bands(all)
				s.Require().Len(bands, 1)
				s.Require().Zero(bands[0].Unsighted, "a band is nobody's label")
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestAGenreFromOneBandIsNotUsable is the case the threshold exists for.
func (s *ManifestPublicTestSuite) TestAGenreFromOneBandIsNotUsable() {
	alone := audio.Manifest{Artist: "Green Day"}
	for i := range 9 {
		alone.Tracks = append(alone.Tracks, audio.Record{
			Track: fmt.Sprintf("t%d", i), Genres: []string{"punk"},
		})
	}

	got := audio.Genres([]audio.Manifest{alone})
	s.Require().Len(got, 1)

	// Nine records is past the record threshold and still one band.
	s.Require().Equal(9, got[0].Records)
	s.Require().Equal(1, got[0].Artists)
	s.Require().False(got[0].Usable(),
		"nine records by one band is that band, not a genre")
}

// TestAGenreSaysWhoDecidedIt covers the provenance rule.
//

// TestAModelsGenresAreCounted covers what the review list reads.
//

// TestBandsGroupsOnTheSlug covers two spellings of one band counting once.
//
// TestBandsGroupsPlayersByTheirBand covers grouping on the slug, and the band nobody named.
//
// One method and one table, so a case is a row rather than a file.
func (s *ManifestPublicTestSuite) TestBandsGroupsPlayersByTheirBand() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// bands. Reported under the spelling first seen, because somebody reading the
			// answer wants the band and not the slug.
			name: "bands groups on the slug",
			then: func() {
				all := []audio.Manifest{
					{Artist: "Duff McKagan", Tracks: []audio.Record{
						{Track: "a", Band: "Guns N' Roses"},
						{Track: "b", Band: "Guns n Roses"},
					}},
					{Artist: "Somebody Else", Tracks: []audio.Record{
						{Track: "c", Band: "Guns N' Roses"},
						// No band at all, which is ordinary: a session player's record
						// belongs to whoever made it and often to no band.
						{Track: "d"},
					}},
				}

				got := audio.Bands(all)
				s.Require().Len(got, 1, "three records, one band, two spellings")

				s.Require().Equal("guns-n-roses", got[0].Slug)
				s.Require().Equal("Guns N' Roses", got[0].Name)
				s.Require().Equal(3, got[0].Records)
				s.Require().Equal(2, got[0].Artists)
			},
		},
		{
			// has to answer nothing rather than an empty band.
			name: "a band is optional",
			then: func() {
				s.Require().Empty(audio.Bands([]audio.Manifest{s.read(full)}))
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestABandIsOptional covers the corpus as it stands.
//

func TestManifestPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ManifestPublicTestSuite))
}
