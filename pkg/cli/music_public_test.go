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
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
	sdk "github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// MusicPublicTestSuite covers the tables that say what the music corpus holds.
type MusicPublicTestSuite struct {
	suite.Suite
}

// wanted is one string a table expects, and why it matters when it is missing.
//
// Carried rather than asserted bare, because the reason is what a failure has
// to say: "does not contain none named" is a diff, and "a player in no band
// says so" is the rule that broke.
type wanted struct {
	text string
	why  string
}

// holds checks what a table drew, and what it must not say.
func (s *MusicPublicTestSuite) holds(
	got string,
	want, not []wanted,
) {
	for _, w := range want {
		s.Require().Contains(got, w.text, w.why)
	}

	for _, w := range not {
		s.Require().NotContains(got, w.text, w.why)
	}
}

// players draws a set of players and hands back what was written.
func (s *MusicPublicTestSuite) players(
	of []sdk.MusicPlayer,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.MusicPlayers(&buf, of))

	return buf.String()
}

// groups draws a set of genres or bands.
func (s *MusicPublicTestSuite) groups(
	of []sdk.MusicGroup,
	kind string,
	threshold bool,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.MusicGroups(&buf, of, kind, threshold))

	return buf.String()
}

// records draws a set of records.
func (s *MusicPublicTestSuite) records(
	of []sdk.MusicRecord,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.MusicRecords(&buf, of))

	return buf.String()
}

// measured draws a set of measured genres.
func (s *MusicPublicTestSuite) measured(
	of []sdk.MeasuredGenre,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.MeasuredGenres(&buf, of))

	return buf.String()
}

// TestMusicPlayers covers the table somebody growing a corpus reads.
func (s *MusicPublicTestSuite) TestMusicPlayers() {
	for _, tt := range []struct {
		name string
		of   []sdk.MusicPlayer
		want []wanted
		not  []wanted
	}{
		{
			name: "what is missing is named",
			of: []sdk.MusicPlayer{
				{
					Instrument: "bass", ID: "mike-dirnt", Artist: "Mike Dirnt", Records: 3,
					Bands: []string{"Green Day"}, Genres: []string{"pop-punk", "punk"},
				},
				// No band and no genre, which is every record written before
				// those fields existed.
				{Instrument: "bass", ID: "flea", Artist: "Flea", Records: 3, Untagged: 3},
			},
			want: []wanted{
				{text: "mike-dirnt"},
				{text: "Green Day"},
				{text: "pop-punk, punk"},
				{text: "none named", why: "a player in no band says so"},
				{text: "none tagged"},
				{text: "3 untagged", why: "and how many records that is"},
				// One of the two has every record tagged, so the detail line
				// says so rather than claiming the corpus is finished.
				{text: "1 with every record tagged"},
			},
		},
		{
			// The absence is the point, so that is what is marked and what the
			// detail line counts. It is the number somebody wants when a genre
			// will not build.
			name: "who has no rig is marked, and counted",
			of: []sdk.MusicPlayer{
				{ID: "mike-dirnt", Records: 3, Genres: []string{"punk"}, Rig: true},
				{ID: "cone-mccaslin", Records: 3, Genres: []string{"punk"}},
			},
			want: []wanted{
				{text: "none", why: "the player with no rig is marked"},
				{text: "1 with no rig"},
			},
		},
		{
			name: "every player has gear",
			of: []sdk.MusicPlayer{
				{ID: "a", Records: 1, Genres: []string{"punk"}, Rig: true},
			},
			want: []wanted{{text: "all with gear"}},
		},
		{
			name: "every player tagged",
			of: []sdk.MusicPlayer{
				{ID: "a", Records: 1, Genres: []string{"punk"}},
			},
			want: []wanted{{text: "every record carrying a genre"}},
		},
		{
			name: "one of something reads as one",
			of: []sdk.MusicPlayer{
				{ID: "a", Records: 1, Genres: []string{"punk"}},
			},
			want: []wanted{{text: "1 player"}},
			not:  []wanted{{text: "1 players"}},
		},
		{
			name: "an empty corpus reaching the table",
			want: []wanted{{text: "no players in this corpus"}},
		},
	} {
		s.Run(tt.name, func() {
			s.holds(s.players(tt.of), tt.want, tt.not)
		})
	}
}

// TestMusicGroups covers the genre and band tables, which are one table asked
// two ways.
func (s *MusicPublicTestSuite) TestMusicGroups() {
	for _, tt := range []struct {
		name      string
		of        []sdk.MusicGroup
		kind      string
		threshold bool
		want      []wanted
		not       []wanted
	}{
		{
			// A genre that earns words and then cannot be built. Grunge did
			// exactly this: nine records from three players, the threshold
			// met, and gear for none of them.
			name: "a genre says when gear is missing",
			of: []sdk.MusicGroup{
				{Name: "grunge", Slug: "grunge", Records: 9, Artists: 3, Usable: true},
				{
					Name: "punk", Slug: "punk", Records: 12, Artists: 4,
					Usable: true, Geared: 3,
				},
			},
			kind: "genre", threshold: true,
			want: []wanted{
				{text: "GEARED"},
				{text: "0 of 3", why: "grunge has gear for nobody"},
				{text: "3 of 4"},
				{text: "1 with gear for nobody"},
			},
		},
		{
			name: "a genre short of the threshold says by how much",
			of: []sdk.MusicGroup{{
				Name: "punk", Slug: "punk", Records: 6, Artists: 2,
				Who:       []string{"Matt Freeman", "Mike Dirnt"},
				Unsighted: 6, ShortRecords: 2, ShortArtists: 1,
			}},
			kind: "genre", threshold: true,
			want: []wanted{
				{text: "2 records short"},
				{text: "1 player short"},
				{text: "none", why: "nobody checked any of the six"},
				{text: "Matt Freeman, Mike Dirnt", why: "named, not counted"},
				{text: "0 worth aiming at"},
			},
		},
		{
			name: "a usable genre reads as one",
			of: []sdk.MusicGroup{{
				Name: "grunge", Slug: "grunge", Records: 9, Artists: 3,
				Who:    []string{"Ben Shepherd", "Jeff Ament", "Krist Novoselic"},
				Usable: true,
			}},
			kind: "genre", threshold: true,
			want: []wanted{
				{text: "a genre"},
				{text: "1 worth aiming at"},
				{text: "all", why: "every record checked"},
			},
		},
		{
			// The one that matters most: enough records, and only some of them
			// looked at.
			name: "a partly checked genre",
			of: []sdk.MusicGroup{{
				Name: "punk", Slug: "punk", Records: 12, Artists: 4,
				Who: []string{"A", "B", "C", "D"}, Unsighted: 5, Usable: true,
			}},
			kind: "genre", threshold: true,
			want: []wanted{{text: "7 of 12"}},
		},
		{
			name: "a band carries no threshold",
			of: []sdk.MusicGroup{{
				Name: "Green Day", Slug: "green-day", Records: 3, Artists: 1,
				Who: []string{"Mike Dirnt"},
			}},
			kind: "band", threshold: false,
			want: []wanted{
				{text: "Green Day"},
				{text: "Mike Dirnt"},
				{text: "two spellings of one band count once"},
			},
			not: []wanted{
				{text: "worth aiming at", why: "a band is not something to aim at"},
				{text: "CHECKED", why: "and nobody labels one"},
			},
		},
		{
			name: "a group nobody made",
			of:   []sdk.MusicGroup{{Name: "punk", Slug: "punk"}},
			kind: "genre", threshold: true,
			want: []wanted{{text: "nobody"}},
		},
		{
			name: "no genres named",
			kind: "genre", threshold: true,
			want: []wanted{{text: "no genres named"}},
		},
		{
			name: "no bands named",
			kind: "band", threshold: false,
			want: []wanted{{text: "no bands named"}},
		},
	} {
		s.Run(tt.name, func() {
			s.holds(s.groups(tt.of, tt.kind, tt.threshold), tt.want, tt.not)
		})
	}
}

// TestMusicRecords covers the table that says what the manifest cannot.
func (s *MusicPublicTestSuite) TestMusicRecords() {
	for _, tt := range []struct {
		name string
		of   []sdk.MusicRecord
		want []wanted
		not  []wanted
	}{
		{
			name: "which records have no stems",
			of: []sdk.MusicRecord{
				{
					Player: "mike-dirnt", Track: "longview", Year: 1994,
					Band: "Green Day", Genres: []string{"punk"},
					DecidedBy: "person", Separated: true,
				},
				{
					Player: "mike-dirnt", Track: "holiday", Year: 2004,
					Band: "Green Day", Genres: []string{"punk"},
					DecidedBy: "llm", Separated: false,
				},
				// Written before the fields existed: no band, no genre, nobody
				// deciding.
				{Player: "flea", Track: "aeroplane", Year: 1995},
			},
			want: []wanted{
				{text: "longview"},
				{text: "a person"},
				{text: "a model"},
				{text: "1 separated"},
				{text: "3 records"},
				{text: "separate it with"},
			},
		},
		{
			name: "an empty table",
			want: []wanted{{text: "no records in this corpus"}},
		},
	} {
		s.Run(tt.name, func() {
			s.holds(s.records(tt.of), tt.want, tt.not)
		})
	}
}

// TestMeasuredGenres covers the three states a measured genre can be in.
func (s *MusicPublicTestSuite) TestMeasuredGenres() {
	for _, tt := range []struct {
		name string
		of   []sdk.MeasuredGenre
		want []wanted
		not  []wanted
	}{
		{
			// The middle state is the finding: a genre can clear the record
			// threshold and still sit inside the middle half on every axis,
			// which is punk on this corpus.
			name: "which genres set something apart",
			of: []sdk.MeasuredGenre{
				{
					Name: "grunge", Slug: "grunge", Records: 9, Players: 3,
					Against: 12, Usable: true,
					Terms: []audio.Derived{
						{Term: "scooped", Key: audio.KeyMid, Mine: 0.01, Others: 0.06},
						{Term: "clean", Key: audio.KeyHarmonics, Mine: 0.13, Others: 0.24},
					},
				},
				{
					Name: "punk", Slug: "punk", Records: 12, Players: 4,
					Against: 11, Usable: true,
				},
				{
					Name: "emo", Slug: "emo", Records: 3, Players: 1, Against: 14,
				},
			},
			want: []wanted{
				{text: "a genre"},
				{text: "scooped"},
				{text: "(0.01 v 0.06)", why: "the figures behind the word"},
				{text: "sets nothing apart", why: "measured, and nothing to aim at"},
				{text: "not a genre yet", why: "and one that lacks the records"},
				{text: "3 genres measured, 1 earning a word"},
			},
		},
		{
			name: "a corpus nobody has tagged",
			want: []wanted{{text: "no records carry a genre"}},
		},
	} {
		s.Run(tt.name, func() {
			s.holds(s.measured(tt.of), tt.want, tt.not)
		})
	}
}

func TestMusicPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MusicPublicTestSuite))
}
