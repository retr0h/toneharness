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
	"math"
	"os"
	"path/filepath"
	"testing"

	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// GenrePublicTestSuite covers the rule a genre is measured by.
//
// Against a built corpus rather than audio. What is under test is the
// comparison, and feeding it real recordings would make the assertion depend on
// whichever records somebody happened to download.
type GenrePublicTestSuite struct {
	suite.Suite
}

// sitting is a measurement placed at one value on every axis.
//
// The three axes a term can be earned on all read the median of a Spread, so
// setting Low, Mid and High to the same number puts the group at a known point
// with no width. Width is what the genre rule deliberately ignores.
func sitting(
	tracks int,
	mid, centroid, harmonics float64,
) audio.Across {
	at := func(v float64) audio.Spread {
		return audio.Spread{Low: v, Mid: v, High: v}
	}

	return audio.Across{
		Tracks:    tracks,
		Mid:       at(mid),
		Centroid:  at(centroid),
		Harmonics: at(harmonics),
	}
}

// pack is four players sitting close together, which is what a genre stands
// against.
func pack() map[string]audio.Across {
	return map[string]audio.Across{
		"a": sitting(3, 0.20, 150, 0.30),
		"b": sitting(3, 0.22, 160, 0.32),
		"c": sitting(3, 0.24, 170, 0.34),
		"d": sitting(3, 0.26, 180, 0.36),
	}
}

// TestAWideSpreadStillEarnsATerm is the whole reason this rule exists.
//
// Derive wants the entire spread outside the middle half of the others, and a
// genre pools several players so its spread is far too wide for that. This
// reads the middle, so a wide group still earns what its centre says.
func (s *GenrePublicTestSuite) TestAWideSpreadStillEarnsATerm() {
	wide := audio.Across{
		Tracks: 9,
		// A median well below the pack, with a spread running right through it.
		Mid:       audio.Spread{Low: 0.01, Mid: 0.05, High: 0.40},
		Centroid:  audio.Spread{Low: 100, Mid: 165, High: 250},
		Harmonics: audio.Spread{Low: 0.10, Mid: 0.33, High: 0.60},
	}

	got := audio.Displaced(wide, pack())
	s.Require().Len(got, 1, "only the mid band's median sits outside")
	s.Require().Equal("scooped", got[0].Term)

	// And Derive refuses the same group, which is the comparison worth making.
	s.Require().Empty(audio.Derive(wide, pack()),
		"the whole-spread rule cannot serve a pooled group")
}

// TestAboveThePackEarnsTheOtherWord covers the upper side.
func (s *GenrePublicTestSuite) TestAboveThePackEarnsTheOtherWord() {
	got := audio.Displaced(sitting(9, 0.40, 300, 0.80), pack())

	terms := map[string]bool{}
	for _, t := range got {
		terms[t.Term] = true
	}

	s.Require().True(terms["mid-forward"])
	s.Require().True(terms["bright"])
	s.Require().True(terms["saturated"])
}

// TestSittingInsideThePackEarnsNothing is the ordinary answer.
//
// It is also the real result for punk on this corpus: the most records and the
// most players of the three genres, and inside the middle half on every axis.
// A genre clearing the record threshold is not a genre that sounds like
// anything in particular.
func (s *GenrePublicTestSuite) TestSittingInsideThePackEarnsNothing() {
	s.Require().Empty(audio.Displaced(sitting(12, 0.23, 165, 0.33), pack()))
}

// TestTheMarginIsHowFarPastTheLine covers what tells a strong word from a weak
// one.
//
// Two words that read alike are not alike. The pack sits at 0.20, 0.22, 0.24 and
// 0.26, so its upper quartile interpolates to 0.245: a median of 0.26 is 0.015
// past the line and one more player could take it away, where 0.60 could not.
func (s *GenrePublicTestSuite) TestTheMarginIsHowFarPastTheLine() {
	weak := audio.Displaced(sitting(9, 0.26, 165, 0.33), pack())
	s.Require().Len(weak, 1)
	s.Require().InDelta(0.015, weak[0].Margin, 0.001)

	strong := audio.Displaced(sitting(9, 0.60, 165, 0.33), pack())
	s.Require().Len(strong, 1)
	s.Require().Greater(strong[0].Margin, weak[0].Margin)
}

// TestTooFewToStandAgainst covers the comparison having nothing behind it.
//
// One player is not a distribution, and the quartiles of one are that player.
// Refused rather than answered, the same way one artist can earn no word.
func (s *GenrePublicTestSuite) TestTooFewToStandAgainst() {
	only := map[string]audio.Across{"a": sitting(3, 0.20, 150, 0.30)}

	s.Require().Empty(audio.Displaced(sitting(9, 0.90, 400, 0.90), only),
		"one player to stand against earns nothing however far away")
	s.Require().Empty(audio.Displaced(sitting(9, 0.90, 400, 0.90), nil))
}

// TestAGroupWithNoRecords covers a genre nothing was measured for.
func (s *GenrePublicTestSuite) TestAGroupWithNoRecords() {
	s.Require().Empty(audio.Displaced(audio.Across{}, pack()))
}

// TestAPlayerWithNoRecordsIsNotAVote covers the others being filtered.
//
// An empty Across is a player nobody measured, and counting its zero as a
// position would drag every comparison toward it.
func (s *GenrePublicTestSuite) TestAPlayerWithNoRecordsIsNotAVote() {
	others := pack()
	others["silent"] = audio.Across{}

	got := audio.Displaced(sitting(9, 0.05, 165, 0.33), others)
	s.Require().Len(got, 1)
	s.Require().Equal("scooped", got[0].Term)

	// Five in the map, four of them real, and the count says what the word
	// stood against plus itself.
	s.Require().Equal(6, got[0].Of)
}

func TestGenrePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GenrePublicTestSuite))
}

// MeasuredGenresPublicTestSuite covers the walk that measures every genre.
//
// Synthetic audio rather than records, so the corpus is one this test decides.
// A bright genre and a dark one is enough to prove the join works: which
// records carry which tag, who made them, and who the comparison stands on.
type MeasuredGenresPublicTestSuite struct {
	suite.Suite

	root string
}

func (s *MeasuredGenresPublicTestSuite) SetupTest() {
	s.root = s.T().TempDir()
}

// player writes one player's manifest and the recordings it names.
//
// Under an instrument, because that is the shape the tree has and what
// Manifests reads the instrument from.
func (s *MeasuredGenresPublicTestSuite) player(
	id, genres string,
	hz float64,
	tracks ...string,
) {
	s.playerOn("bass", id, genres, hz, tracks...)
}

// playerOn is the same, under a named instrument's tree.
//
// The instrument is the directory above the player, which is how the manifests
// record it and the only place it is written down.
func (s *MeasuredGenresPublicTestSuite) playerOn(
	instrument, id, genres string,
	hz float64,
	tracks ...string,
) {
	dir := filepath.Join(s.root, instrument, id)

	body := "artist: " + id + "\ntracks:\n"

	for i, track := range tracks {
		body += fmt.Sprintf("  - track: %s\n    url: https://open.spotify.com/track/x%d\n"+
			"    year: 1994\n", track, i)

		if genres != "" {
			body += "    genres: [" + genres + "]\n    genres_by: llm\n"
		}

		at := filepath.Join(dir, "stems", "htdemucs", track, "bass.wav")
		s.Require().NoError(os.MkdirAll(filepath.Dir(at), 0o750))
		s.write(at, audio.Sine(hz, 1, rate, 0.8))
	}

	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "corpus.yaml"), []byte(body), 0o600))
}

// write puts samples on disk as a mono wav.
func (s *MeasuredGenresPublicTestSuite) write(
	at string,
	samples []float64,
) {
	f, err := os.Create(at) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	enc := wav.NewEncoder(f, rate, 16, 1, 1)

	scale := math.Pow(2, 15) - 1
	data := make([]int, 0, len(samples))

	for _, v := range samples {
		data = append(data, int(v*scale))
	}

	s.Require().NoError(enc.Write(&goaudio.IntBuffer{
		Format:         &goaudio.Format{NumChannels: 1, SampleRate: rate},
		Data:           data,
		SourceBitDepth: 16,
	}))
	s.Require().NoError(enc.Close())
	s.Require().NoError(f.Close())
}

// TestGenresMeasured covers every case GenresMeasured answers.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuredGenresPublicTestSuite) TestGenresMeasured() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The join under test.
			name: "a genre stands on the players who do not play it",
			then: func() {
				// Three bright players tagged punk, and four dark ones tagged nothing.
				s.player("a", "punk", 3000, "t1", "t2", "t3")
				s.player("b", "punk", 3100, "t1", "t2", "t3")
				s.player("c", "punk", 3200, "t1", "t2", "t3")

				for i, id := range []string{"w", "x", "y", "z"} {
					s.player(id, "", 200+float64(i)*20, "t1", "t2", "t3")
				}

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				punk := got[0]
				s.Require().Equal("punk", punk.Slug)
				s.Require().Equal(9, punk.Records)
				s.Require().Equal(3, punk.Players)
				s.Require().Equal(4, punk.Against, "the four who play none of it")
				s.Require().True(punk.Usable, "nine records from three players")

				terms := map[string]bool{}
				for _, t := range punk.Terms {
					terms[t.Term] = true
				}

				s.Require().True(terms["bright"], "3kHz against 200Hz")
			},
		},
		{
			// Why the field exists.
			//
			// Without it a genre's figures carry no hint of what they
			// describe, and they do not read as wrong: this corpus is bass,
			// so every genre's centroid sits between 90 and 182Hz, and a
			// guitar chain solved against one of those converges, reports its
			// tolerances met, and has been asked to sound like another
			// instrument.
			name: "a genre records which instrument it was measured on",
			then: func() {
				s.player("a", "punk", 3000, "t1", "t2", "t3")
				s.player("b", "punk", 3100, "t1", "t2", "t3")
				s.player("c", "punk", 3200, "t1", "t2", "t3")

				for i, id := range []string{"w", "x"} {
					s.player(id, "", 200+float64(i)*20, "t1")
				}

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().Equal("bass", got[0].Instrument)
				s.Require().True(got[0].Usable)
			},
		},
		{
			// The mixed case.
			//
			// Its centre of gravity sits between the two and describes
			// neither. That is worse than too few records, because the
			// figures look ordinary: a genre half bass and half guitar reads
			// as a plausible middle nothing was played at.
			name: "a genre pooled across two instruments may not be aimed at",
			then: func() {
				s.playerOn("bass", "a", "punk", 100, "t1", "t2", "t3")
				s.playerOn("bass", "b", "punk", 110, "t1", "t2", "t3")
				s.playerOn("guitar", "c", "punk", 3000, "t1", "t2", "t3")

				for i, id := range []string{"w", "x"} {
					s.player(id, "", 200+float64(i)*20, "t1")
				}

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().Equal(9, got[0].Records, "it is still measured and reported")
				s.Require().Equal(3, got[0].Players)
				s.Require().Empty(got[0].Instrument, "no one instrument describes it")
				s.Require().False(got[0].Usable,
					"nine records from three players, and still nothing to aim at")
			},
		},
		{
			// The gap that reads as agreement.
			//
			// A player sitting directly under the corpus root rather than
			// under an instrument's tree has no instrument: the manifests
			// take it from the directory above the player, and there is none.
			// Treated as "nothing recorded yet", that player is absorbed into
			// whichever instrument the next record names, and a genre half
			// made of records nobody classified reads as pure bass and may be
			// aimed at.
			//
			// Not knowing is not agreeing, so it clears the answer the same
			// way a second instrument does.
			name: "a record naming no instrument is a disagreement",
			then: func() {
				// Two bass players, and one sitting at the root with no instrument above it.
				s.playerOn("bass", "a", "punk", 100, "t1", "t2", "t3")
				s.playerOn("bass", "b", "punk", 110, "t1", "t2", "t3")
				s.playerOn(".", "rootling", "punk", 120, "t1", "t2", "t3")

				for i, id := range []string{"w", "x"} {
					s.player(id, "", 200+float64(i)*20, "t1")
				}

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)

				var punk audio.Genre

				for _, g := range got {
					if g.Slug == "punk" {
						punk = g
					}
				}

				s.Require().Equal("punk", punk.Slug)
				s.Require().Empty(punk.Instrument,
					"one unclassified player means the genre names no instrument")
				s.Require().False(punk.Usable,
					"and nothing may aim at figures whose instrument is unknown")
			},
		},
		{
			// Reporting rather than skipping.
			//
			// Knowing how far three punk records sit from the rest is worth
			// seeing. What the threshold decides is whether anything may aim
			// at it.
			name: "a genre under the threshold is still measured",
			then: func() {
				s.player("a", "punk", 3000, "t1", "t2", "t3")

				for i, id := range []string{"w", "x", "y"} {
					s.player(id, "", 200+float64(i)*20, "t1")
				}

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				s.Require().Equal(3, got[0].Records)
				s.Require().Equal(1, got[0].Players)
				s.Require().False(got[0].Usable, "one player is that player, not a genre")
				s.Require().NotEmpty(got[0].Terms, "measured anyway, and reported")
			},
		},
		{
			// The manifest being the statement of what was measured.
			//
			// Its stems sit there because separating is expensive and nobody
			// deletes them, and a record dropped from a manifest is one
			// somebody decided not to measure.
			name: "a record the manifest does not name is not measured",
			then: func() {
				s.player("a", "punk", 3000, "t1", "t2")

				// A fourth recording on disk that the manifest never mentions.
				at := filepath.Join(s.root, "bass", "a", "stems", "htdemucs", "stray", "bass.wav")
				s.Require().NoError(os.MkdirAll(filepath.Dir(at), 0o750))
				s.write(at, audio.Sine(500, 1, rate, 0.8))

				for _, id := range []string{"w", "x"} {
					s.player(id, "", 200, "t1")
				}

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)
				s.Require().Equal(2, got[0].Records, "the stray is not one of them")
			},
		},
		{
			// A corpus nobody has tagged.
			name: "no genres at all",
			then: func() {
				s.player("a", "", 200, "t1")

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Empty(got)
			},
		},
		{
			// The failure being reported.
			name: "a recording that will not read",
			then: func() {
				s.player("a", "punk", 3000, "t1")

				at := filepath.Join(s.root, "bass", "a", "stems", "htdemucs", "t1", "bass.wav")
				s.Require().NoError(os.WriteFile(at, []byte("not a wav"), 0o600))

				_, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().Error(err)
			},
		},
		{
			// The walk finding nothing to measure.
			name: "a tree with no manifests",
			then: func() {
				_, err := audio.GenresMeasured(os.DirFS(s.root), "nowhere")
				s.Require().Error(err)
			},
		},
		{
			// The order two runs must agree on.
			name: "two genres sort by their slug",
			then: func() {
				s.player("a", "punk", 3000, "t1")
				s.player("b", "grunge", 2000, "t1")

				for _, id := range []string{"w", "x"} {
					s.player(id, "", 200, "t1")
				}

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Len(got, 2)
				s.Require().Equal("grunge", got[0].Slug)
				s.Require().Equal("punk", got[1].Slug)
			},
		},
		{
			// A player who cannot vote.
			//
			// The manifest names records and none of them were separated, so
			// there is nothing to measure. That is a player mid-setup rather
			// than a fault, and counting their empty measurement as a
			// position would drag the comparison toward zero.
			name: "a manifest naming nothing on disk is not a player",
			then: func() {
				s.player("a", "punk", 3000, "t1")

				for _, id := range []string{"w", "x"} {
					s.player(id, "", 200, "t1")
				}

				// A manifest with no stems beside it at all.
				dir := filepath.Join(s.root, "bass", "waiting")
				s.Require().NoError(os.MkdirAll(dir, 0o750))
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "corpus.yaml"),
					[]byte("artist: Waiting\ntracks:\n  - track: soon\n"+
						"    url: https://open.spotify.com/track/x\n    year: 1994\n"), 0o600))

				got, err := audio.GenresMeasured(os.DirFS(s.root), ".")
				s.Require().NoError(err)
				s.Require().Len(got, 1)
				s.Require().Equal(2, got[0].Against, "the two measured, not the one waiting")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// TestEveryShippedGenreNamesItsInstrument holds the generated file to the field.
//
// The check is here rather than left to the reader because an unnamed instrument
// and a genre pooled across two are the same empty string, and a genres.json
// generated before the field existed would read as the second. That turns a
// stale file into a wrong refusal with a confident explanation, so this fails
// instead: regenerate with `go generate ./pkg/sdk/audio`.
func (s *MeasuredGenresPublicTestSuite) TestEveryShippedGenreNamesItsInstrument() {
	all, err := audio.Shipped()
	s.Require().NoError(err)
	s.Require().NotEmpty(all)

	for _, g := range all {
		s.Require().NotEmpty(g.Instrument,
			"%s names no instrument, so the shipped file predates the field",
			g.Slug)
	}
}

func TestMeasuredGenresPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasuredGenresPublicTestSuite))
}

// TestAnAxisNobodyHasAReadingFor covers a figure the others cannot supply.
//
// Transient and decay are absent from audio that never rises or never falls
// that far, so an axis can have fewer positions behind it than there are
// players. Fewer than two is not a distribution and earns nothing.
func (s *GenrePublicTestSuite) TestAnAxisNobodyHasAReadingFor() {
	// Two players, one of which has no recordings at all, so every axis has a
	// single position behind it.
	others := map[string]audio.Across{
		"a":      sitting(3, 0.20, 150, 0.30),
		"silent": {},
	}

	s.Require().Empty(audio.Displaced(sitting(9, 0.90, 400, 0.90), others),
		"one position is not a middle half")
}

// ShippedPublicTestSuite covers the measured genres the binary carries.
//
// Against what is actually committed rather than a fixture, because the point
// of the file is that a binary built with no audio still answers. A fixture
// would prove the parser works and say nothing about the shipped data.
type ShippedPublicTestSuite struct {
	suite.Suite
}

// TestShippedParses covers the committed file being readable.
func (s *ShippedPublicTestSuite) TestShippedParses() {
	got, err := audio.Shipped()
	s.Require().NoError(err)

	// Parsed twice through the same once, which is the point of caching it.
	again, err := audio.Shipped()
	s.Require().NoError(err)
	s.Require().Equal(got, again)

	for _, g := range got {
		s.Require().NotEmpty(g.Slug, "a genre with no slug matches no request")
		s.Require().Positive(g.Records)
		s.Require().Positive(g.Players)
	}
}

// TestShippedGenreFindsOneBySlug covers the lookup a request goes through.
func (s *ShippedPublicTestSuite) TestShippedGenreFindsOneBySlug() {
	all, err := audio.Shipped()
	s.Require().NoError(err)

	if len(all) == 0 {
		s.T().Skip("no genres measured into this binary")
	}

	got, ok := audio.ShippedGenre(all[0].Slug)
	s.Require().True(ok)
	s.Require().Equal(all[0].Slug, got.Slug)
}

// TestAGenreNothingMeasured is the answer for a word nobody has records for.
//
// Not an error. A request may name anything, and the honest answer is that
// nothing was measured rather than that something went wrong.
func (s *ShippedPublicTestSuite) TestAGenreNothingMeasured() {
	got, ok := audio.ShippedGenre("no-such-genre")
	s.Require().False(ok)
	s.Require().Zero(got.Records)
}

func TestShippedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShippedPublicTestSuite))
}

// TestAFileThatWillNotParse covers the one failure the reader can have.
//
// The committed file always parses and go:embed refuses to compile without it,
// so nothing else reaches this. Tested through the unpacking rather than the
// embed, because faking the embed would test nothing the compiler does not
// already guarantee.
func (s *ShippedPublicTestSuite) TestAFileThatWillNotParse() {
	_, err := audio.UnpackGenres([]byte("not json"))
	s.Require().ErrorContains(err, "reading the measured genres")
}
