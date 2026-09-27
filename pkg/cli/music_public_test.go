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

// TestPlayersNamesWhatIsMissing is what somebody growing a corpus reads.
func (s *MusicPublicTestSuite) TestPlayersNamesWhatIsMissing() {
	got := s.players([]sdk.MusicPlayer{
		{
			Instrument: "bass", ID: "mike-dirnt", Artist: "Mike Dirnt", Records: 3,
			Bands: []string{"Green Day"}, Genres: []string{"pop-punk", "punk"},
		},
		// No band and no genre, which is every record written before those
		// fields existed.
		{Instrument: "bass", ID: "flea", Artist: "Flea", Records: 3, Untagged: 3},
	})

	s.Require().Contains(got, "mike-dirnt")
	s.Require().Contains(got, "Green Day")
	s.Require().Contains(got, "pop-punk, punk")

	s.Require().Contains(got, "none named", "a player in no band says so")
	s.Require().Contains(got, "none tagged")
	s.Require().Contains(got, "3 untagged", "and how many records that is")

	// One of the two has every record tagged, so the detail line says so
	// rather than claiming the corpus is finished.
	s.Require().Contains(got, "1 with every record tagged")
}

// TestEveryPlayerTagged covers the other detail line.
func (s *MusicPublicTestSuite) TestEveryPlayerTagged() {
	got := s.players([]sdk.MusicPlayer{
		{ID: "a", Records: 1, Genres: []string{"punk"}},
	})

	s.Require().Contains(got, "every record carrying a genre")
}

// TestNoPlayers covers an empty corpus reaching the table.
func (s *MusicPublicTestSuite) TestNoPlayers() {
	s.Require().Contains(s.players(nil), "no players in this corpus")
}

// TestAGenreShortOfTheThresholdSaysByHowMuch is the number somebody acts on.
func (s *MusicPublicTestSuite) TestAGenreShortOfTheThresholdSaysByHowMuch() {
	got := s.groups([]sdk.MusicGroup{{
		Name: "punk", Slug: "punk", Records: 6, Artists: 2,
		Who:       []string{"Matt Freeman", "Mike Dirnt"},
		Unsighted: 6, ShortRecords: 2, ShortArtists: 1,
	}}, "genre", true)

	s.Require().Contains(got, "2 records short")
	s.Require().Contains(got, "1 player short")
	s.Require().Contains(got, "none", "nobody checked any of the six")
	s.Require().Contains(got, "Matt Freeman, Mike Dirnt", "named, not counted")
	s.Require().Contains(got, "0 worth aiming at")
}

// TestAUsableGenreReadsAsOne covers the threshold being met.
func (s *MusicPublicTestSuite) TestAUsableGenreReadsAsOne() {
	got := s.groups([]sdk.MusicGroup{{
		Name: "grunge", Slug: "grunge", Records: 9, Artists: 3,
		Who:    []string{"Ben Shepherd", "Jeff Ament", "Krist Novoselic"},
		Usable: true,
	}}, "genre", true)

	s.Require().Contains(got, "a genre")
	s.Require().Contains(got, "1 worth aiming at")
	s.Require().Contains(got, "all", "every record checked")
}

// TestAPartlyCheckedGenre covers the middle case.
//
// The one that matters most: enough records, and only some of them looked at.
func (s *MusicPublicTestSuite) TestAPartlyCheckedGenre() {
	got := s.groups([]sdk.MusicGroup{{
		Name: "punk", Slug: "punk", Records: 12, Artists: 4,
		Who: []string{"A", "B", "C", "D"}, Unsighted: 5, Usable: true,
	}}, "genre", true)

	s.Require().Contains(got, "7 of 12")
}

// TestBandsCarryNoThreshold covers the table without the genre columns.
func (s *MusicPublicTestSuite) TestBandsCarryNoThreshold() {
	got := s.groups([]sdk.MusicGroup{{
		Name: "Green Day", Slug: "green-day", Records: 3, Artists: 1,
		Who: []string{"Mike Dirnt"},
	}}, "band", false)

	s.Require().Contains(got, "Green Day")
	s.Require().Contains(got, "Mike Dirnt")
	s.Require().Contains(got, "two spellings of one band count once")

	s.Require().NotContains(got, "worth aiming at",
		"a band is not something to aim at")
	s.Require().NotContains(got, "CHECKED", "and nobody labels one")
}

// TestAGroupNobodyMade covers a name with no players behind it.
func (s *MusicPublicTestSuite) TestAGroupNobodyMade() {
	got := s.groups([]sdk.MusicGroup{{Name: "punk", Slug: "punk"}}, "genre", true)
	s.Require().Contains(got, "nobody")
}

// TestNoGroups covers the empty table, which names the kind asked for.
func (s *MusicPublicTestSuite) TestNoGroups() {
	s.Require().Contains(s.groups(nil, "genre", true), "no genres named")
	s.Require().Contains(s.groups(nil, "band", false), "no bands named")
}

// TestRecordsSaysWhichHaveNoStems is what the manifest cannot say.
func (s *MusicPublicTestSuite) TestRecordsSaysWhichHaveNoStems() {
	got := s.records([]sdk.MusicRecord{
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
	})

	s.Require().Contains(got, "longview")
	s.Require().Contains(got, "a person")
	s.Require().Contains(got, "a model")
	s.Require().Contains(got, "1 separated")
	s.Require().Contains(got, "3 records")
	s.Require().Contains(got, "separate it with")
}

// TestNoRecords covers the empty table.
func (s *MusicPublicTestSuite) TestNoRecords() {
	s.Require().Contains(s.records(nil), "no records in this corpus")
}

// TestOneOfSomethingReadsAsOne covers the count that does not take an s.
func (s *MusicPublicTestSuite) TestOneOfSomethingReadsAsOne() {
	got := s.players([]sdk.MusicPlayer{
		{ID: "a", Records: 1, Genres: []string{"punk"}},
	})

	s.Require().Contains(got, "1 player")
	s.Require().NotContains(got, "1 players")
}

func TestMusicPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MusicPublicTestSuite))
}

// measured draws a set of measured genres.
func (s *MusicPublicTestSuite) measured(
	of []sdk.MeasuredGenre,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.MeasuredGenres(&buf, of))

	return buf.String()
}

// TestMeasuredGenresSaysWhichSetSomethingApart covers the three states.
//
// The middle one is the finding: a genre can clear the record threshold and
// still sit inside the middle half on every axis, which is punk on this corpus.
func (s *MusicPublicTestSuite) TestMeasuredGenresSaysWhichSetSomethingApart() {
	got := s.measured([]sdk.MeasuredGenre{
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
	})

	s.Require().Contains(got, "a genre")
	s.Require().Contains(got, "scooped")
	s.Require().Contains(got, "(0.01 v 0.06)", "the figures behind the word")

	s.Require().Contains(got, "sets nothing apart", "measured, and nothing to aim at")
	s.Require().Contains(got, "not a genre yet", "and one that lacks the records")

	s.Require().Contains(got, "3 genres measured, 1 earning a word")
}

// TestNoGenreMeasured covers a corpus nobody has tagged.
func (s *MusicPublicTestSuite) TestNoGenreMeasured() {
	s.Require().Contains(s.measured(nil), "no records carry a genre")
}
