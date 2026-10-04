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
	"math"
	"os"
	"path/filepath"
	"testing"

	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// CorpusPublicTestSuite covers measuring several players and comparing them.
type CorpusPublicTestSuite struct {
	suite.Suite

	root string
}

func (s *CorpusPublicTestSuite) SetupTest() {
	s.root = s.T().TempDir()
}

// record writes one of a player's recordings.
func (s *CorpusPublicTestSuite) record(
	player, track string,
	samples []float64,
) {
	full := filepath.Join(s.root, player, "stems", "htdemucs", track, "bass.wav")

	s.Require().NoError(os.MkdirAll(filepath.Dir(full), 0o750))

	f, err := os.Create(full) //nolint:gosec // a path this test chose
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

// corpus reads the tree the way the command does.
func (s *CorpusPublicTestSuite) corpus() []audio.Player {
	got, err := audio.Corpus(os.DirFS(s.root), ".")
	s.Require().NoError(err)

	return got
}

// TestOnePlayerEarnsNothing covers why this exists.
//
// A word is earned by sitting clear of the other players. One player is clear
// of nobody, however far from the middle their records sit.
func (s *CorpusPublicTestSuite) TestOnePlayerEarnsNothing() {
	for _, track := range []string{"one", "two", "three"} {
		s.record("mike-dirnt", track, audio.Sine(800, 1, rate, 0.8))
	}

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal("mike-dirnt", got[0].ID)
	s.Require().Equal(3, got[0].Records)
	s.Require().Empty(got[0].Terms)
}

// TestAPlayerClearOfTheRestEarnsAWord covers the comparison doing its job.
//
// Three players low and one high, so the high one is the only one whose whole
// range sits above the others.
func (s *CorpusPublicTestSuite) TestAPlayerClearOfTheRestEarnsAWord() {
	for _, player := range []string{"low-one", "low-two", "low-three"} {
		for _, track := range []string{"one", "two", "three"} {
			s.record(player, track, audio.Sine(80, 1, rate, 0.8))
		}
	}

	for _, track := range []string{"one", "two", "three"} {
		s.record("bright-one", track, audio.Sine(2000, 1, rate, 0.8))
	}

	got := s.corpus()
	s.Require().Len(got, 4)

	terms := map[string][]string{}

	for _, p := range got {
		for _, t := range p.Terms {
			terms[p.ID] = append(terms[p.ID], t.Term)
		}
	}

	s.Require().Contains(terms["bright-one"], "bright")

	// The three at the bottom are identical, so which of them the quartile
	// lands on is decided by noise. What matters is that none of them is the
	// bright one.
	for _, player := range []string{"low-one", "low-two", "low-three"} {
		s.Require().NotContains(terms[player], "bright")
	}
}

// TestAPlayerIsComparedAgainstEverybodyElse covers a player not being
// compared against themselves.
func (s *CorpusPublicTestSuite) TestAPlayerIsComparedAgainstEverybodyElse() {
	for _, player := range []string{"one", "two"} {
		s.record(player, "track", audio.Sine(110, 1, rate, 0.8))
	}

	got := s.corpus()
	s.Require().Len(got, 2)

	for _, p := range got {
		for _, t := range p.Terms {
			s.Require().Equal(2, t.Of,
				"both players, counted once each")
		}
	}
}

// TestAWordSaysHowFarPastTheLineItSits covers the margin, which is what
// tells a word that will never flip from one that flips on the next player.
func (s *CorpusPublicTestSuite) TestAWordSaysHowFarPastTheLineItSits() {
	for _, player := range []string{"low-one", "low-two", "low-three"} {
		for _, track := range []string{"one", "two", "three"} {
			s.record(player, track, audio.Sine(80, 1, rate, 0.8))
		}
	}

	for _, track := range []string{"one", "two", "three"} {
		s.record("bright-one", track, audio.Sine(2000, 1, rate, 0.8))
	}

	for _, p := range s.corpus() {
		for _, t := range p.Terms {
			if t.Term == "bright" {
				s.Require().Positive(t.Margin,
					"a word is earned by clearing a line, so it clears it by something")
				s.Require().Greater(t.Mine, t.Others)

				return
			}
		}
	}

	s.Require().Fail("nobody earned bright")
}

// manifest writes one player's manifest, naming the records given.
func (s *CorpusPublicTestSuite) manifest(
	player string,
	tracks ...string,
) {
	body := "artist: " + player + "\ntracks:\n"
	for _, t := range tracks {
		body += "  - track: " + t +
			"\n    url: https://open.spotify.com/track/abc\n    year: 1994\n"
	}

	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, player, "corpus.yaml"), []byte(body), 0o600))
}

// tagged writes a manifest whose records all carry the same genres, which is
// what puts a player in one.
func (s *CorpusPublicTestSuite) tagged(
	player string,
	genres string,
	tracks ...string,
) {
	body := "artist: " + player + "\ntracks:\n"
	for _, t := range tracks {
		body += "  - track: " + t +
			"\n    url: https://open.spotify.com/track/abc\n    year: 1994\n" +
			"    genres: [" + genres + "]\n    genres_by: person\n"
	}

	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, player, "corpus.yaml"), []byte(body), 0o600))
}

// TestAPlayerIsAlsoComparedInsideTheirGenre covers the second comparison, which
// answers a different question from the first.
//
// One method and one table, so a case is a row rather than a file.
func (s *CorpusPublicTestSuite) TestAPlayerIsAlsoComparedInsideTheirGenre() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The case the whole thing exists for.
			//
			// A player in the middle of the corpus can sit at the edge of their
			// own genre, and that is the claim somebody building that sound
			// wants. Three loud players make a genre whose middle is high, and
			// the quiet one in it is dark for the genre while sitting in the
			// middle of everybody.
			name: "a player flat against the corpus is placed inside their genre",
			then: func() {
				for _, p := range []string{"loud-one", "loud-two", "loud-three"} {
					for _, t := range []string{"one", "two", "three"} {
						s.record(p, t, audio.Sine(800, 1, rate, 0.8))
					}

					s.tagged(p, "punk", "one", "two", "three")
				}

				for _, t := range []string{"one", "two", "three"} {
					s.record("quiet", t, audio.Sine(300, 1, rate, 0.8))
				}

				s.tagged("quiet", "punk", "one", "two", "three")

				// Two players of another genre, between the two groups, so the
				// corpus-wide middle sits where nobody is clear of it.
				for _, p := range []string{"other-one", "other-two"} {
					for _, t := range []string{"one", "two", "three"} {
						s.record(p, t, audio.Sine(500, 1, rate, 0.8))
					}

					s.tagged(p, "jazz", "one", "two", "three")
				}

				got := s.byID()

				within := got["quiet"].Within
				s.Require().Len(within, 1, "one genre, so one comparison")
				s.Require().Equal("punk", within[0].Genre)
				s.Require().Equal(4, within[0].Of, "the player and their three peers")
				s.Require().NotEmpty(within[0].Terms,
					"clear of the punk players, whatever the corpus says")
			},
		},
		{
			// A genre too small to compare inside.
			//
			// Two players is one other, and a word against one person says they
			// differ rather than anything about the genre. The corpus-wide
			// comparison still happens: this is an extra answer, never a
			// replacement.
			name: "a genre under the threshold is not compared inside",
			then: func() {
				for _, p := range []string{"one-of-two", "two-of-two"} {
					for _, t := range []string{"one", "two", "three"} {
						s.record(p, t, audio.Sine(800, 1, rate, 0.8))
					}

					s.tagged(p, "rare", "one", "two", "three")
				}

				for _, t := range []string{"one", "two", "three"} {
					s.record("elsewhere", t, audio.Sine(80, 1, rate, 0.8))
				}

				s.tagged("elsewhere", "common", "one", "two", "three")

				got := s.byID()
				s.Require().Empty(got["one-of-two"].Within,
					"two players is one other, which is not a genre")
			},
		},
		{
			// An untagged player.
			//
			// Nothing to compare inside, and the corpus-wide answer is
			// untouched. A manifest is optional and so is a genre on it.
			name: "a player with no genre has nothing to be compared inside",
			then: func() {
				for _, p := range []string{"low-one", "low-two", "low-three"} {
					for _, t := range []string{"one", "two", "three"} {
						s.record(p, t, audio.Sine(80, 1, rate, 0.8))
					}
				}

				for _, t := range []string{"one", "two", "three"} {
					s.record("high", t, audio.Sine(800, 1, rate, 0.8))
				}

				got := s.byID()
				s.Require().Empty(got["high"].Within)
				s.Require().NotEmpty(got["high"].Terms,
					"the comparison against everybody is unaffected")
			},
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()
			tt.then()
		})
	}
}

// byID is the corpus keyed by player, for a case asserting about one of them.
func (s *CorpusPublicTestSuite) byID() map[string]audio.Player {
	out := map[string]audio.Player{}
	for _, p := range s.corpus() {
		out[p.ID] = p
	}

	return out
}

// TestOnlyWhatTheManifestNamesIsMeasured covers the manifest deciding what a
// player's figures are made of, rather than whatever is on disk.
//
// Records get replaced, and the workflow says to keep the audio of the ones
// taken out because separating it again costs minutes. Six were retired that
// way and every one of them was still in the figures until this: the counts
// read six where the manifest said three.
func (s *CorpusPublicTestSuite) TestOnlyWhatTheManifestNamesIsMeasured() {
	for _, track := range []string{"kept-one", "kept-two", "retired"} {
		s.record("mike-dirnt", track, audio.Sine(110, 1, rate, 0.8))
	}

	s.manifest("mike-dirnt", "kept-one", "kept-two")

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal(2, got[0].Records, "the retired record is still on disk")
}

// TestAPlayerWithNoManifestIsMeasuredAsFound covers a directory somebody is
// still assembling, where the manifest has not been written yet.
func (s *CorpusPublicTestSuite) TestAPlayerWithNoManifestIsMeasuredAsFound() {
	for _, track := range []string{"one", "two", "three"} {
		s.record("nobody", track, audio.Sine(110, 1, rate, 0.8))
	}

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal(3, got[0].Records)
}

// TestCorpus covers Corpus, which measures every player under a tree and
// derives what each one's figures say against the others.
//
// One method and one table, so a case is a row rather than a file.
func (s *CorpusPublicTestSuite) TestCorpus() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A corpus somebody broke, which stops the reading rather than
			// silently measuring the tree instead.
			name: "a manifest that will not read",
			then: func() {
				s.record("mike-dirnt", "one", audio.Sine(110, 1, rate, 0.8))
				s.Require().NoError(os.WriteFile(
					filepath.Join(s.root, "mike-dirnt", "corpus.yaml"),
					[]byte("tracks:\n  - track: one\n    note: a note: with a colon\n"), 0o600))

				_, err := audio.Corpus(os.DirFS(s.root), ".")

				s.Require().Error(err)
			},
		},
		{
			// A manifest that is there and will not open.
			//
			// Distinct from one that is absent, which measures the tree as
			// found: a manifest nobody can read is a statement of what to
			// measure that nobody can read, and measuring the tree instead
			// would quietly use records somebody took out.
			//
			// A symlink to itself rather than a file with its permissions
			// removed. Both fail to open; only one fails for every user, and
			// a test that skips itself for root is a test that does not run
			// where it matters.
			name: "a manifest that cannot be opened",
			then: func() {
				s.record("mike-dirnt", "one", audio.Sine(110, 1, rate, 0.8))

				at := filepath.Join(s.root, "mike-dirnt", "corpus.yaml")
				s.Require().NoError(os.Symlink("corpus.yaml", at))

				_, err := audio.Corpus(os.DirFS(s.root), ".")

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "mike-dirnt")
			},
		},
		{
			// The path being wrong.
			name: "a tree that is not there",
			then: func() {
				_, err := audio.Corpus(os.DirFS(s.root), "nowhere")

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "nowhere")
			},
		},
		{
			// A .wav that will not read.
			name: "a recording that is not one",
			then: func() {
				full := filepath.Join(s.root, "mike-dirnt", "broken.wav")
				s.Require().NoError(os.MkdirAll(filepath.Dir(full), 0o750))
				s.Require().NoError(os.WriteFile(full, []byte("not a wav"), 0o600))

				_, err := audio.Corpus(os.DirFS(s.root), ".")

				s.Require().Error(err)
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

// TestADirectoryWithNoRecordingsIsNotAPlayer covers a manifest waiting for
// audio somebody has not separated yet.
func (s *CorpusPublicTestSuite) TestADirectoryWithNoRecordingsIsNotAPlayer() {
	s.record("mike-dirnt", "longview", audio.Sine(110, 1, rate, 0.8))
	s.Require().NoError(os.MkdirAll(filepath.Join(s.root, "flea"), 0o750))
	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, "README.md"), []byte("not a player"), 0o600))

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal("mike-dirnt", got[0].ID)
}

// TestTheOrderIsFixed covers two runs reading the same way.
func (s *CorpusPublicTestSuite) TestTheOrderIsFixed() {
	for _, player := range []string{"pino-palladino", "flea", "mike-dirnt"} {
		s.record(player, "track", audio.Sine(110, 1, rate, 0.8))
	}

	got := s.corpus()

	s.Require().Len(got, 3)
	s.Require().Equal("flea", got[0].ID)
	s.Require().Equal("mike-dirnt", got[1].ID)
	s.Require().Equal("pino-palladino", got[2].ID)
}

// TestScattered covers a player whose own records disagree with each other.
//
// The case it exists for is an artist measured across three albums and five
// years. Their figures are an average of several different sounds, so they earn
// nothing, and that silence reads exactly like a player who is unremarkable.
func (s *CorpusPublicTestSuite) TestScattered() {
	tests := []struct {
		name  string
		setup func()
		want  func(map[string]audio.Player)
	}{
		{
			// Three players holding still at different pitches, so the corpus has
			// a spread to be measured against and nobody is wider than it.
			name: "records that agree are not scattered",
			setup: func() {
				for i, player := range []string{"one", "two", "three"} {
					for _, track := range []string{"a", "b", "c"} {
						s.record(player, track, audio.Sine(float64(200+200*i), 1, rate, 0.8))
					}
				}
			},
			want: func(by map[string]audio.Player) {
				for _, p := range by {
					s.Require().Empty(p.Scattered, p.ID)
				}
			},
		},
		{
			// One player's three records at 100Hz, 1kHz and 3kHz: further apart
			// than the whole corpus of players is.
			name: "records that disagree are reported, with the ratio",
			setup: func() {
				for i, player := range []string{"one", "two", "three"} {
					for _, track := range []string{"a", "b", "c"} {
						s.record(player, track, audio.Sine(float64(200+100*i), 1, rate, 0.8))
					}
				}

				for i, track := range []string{"a", "b", "c"} {
					s.record("varied", track,
						audio.Sine([]float64{100, 1000, 3000}[i], 1, rate, 0.8))
				}
			},
			want: func(by map[string]audio.Player) {
				got := by["varied"].Scattered
				s.Require().NotEmpty(got)

				keys := make([]audio.Figure, 0, len(got))
				for _, d := range got {
					keys = append(keys, d.Key)

					s.Require().Greater(d.Own, d.Between, string(d.Key))
					s.Require().Greater(d.Times, 1.0, string(d.Key))
					s.Require().NotEmpty(d.Why, string(d.Key))
				}

				s.Require().Contains(keys, audio.KeyCentroid)

				for _, player := range []string{"one", "two", "three"} {
					s.Require().Empty(by[player].Scattered, player)
				}
			},
		},
		{
			// One record cannot disagree with anything, so it is never scattered
			// rather than always consistent.
			name: "one record is never scattered",
			setup: func() {
				for _, player := range []string{"one", "two", "three"} {
					for _, track := range []string{"a", "b", "c"} {
						s.record(player, track, audio.Sine(300, 1, rate, 0.8))
					}
				}

				s.record("alone", "only", audio.Sine(3000, 1, rate, 0.8))
			},
			want: func(by map[string]audio.Player) {
				s.Require().Equal(1, by["alone"].Records)
				s.Require().Empty(by["alone"].Scattered)
			},
		},
		{
			// Nobody to be measured against, so there is no spread to be wider
			// than and the question cannot be asked.
			name: "the only player in a corpus is not scattered",
			setup: func() {
				for i, track := range []string{"a", "b", "c"} {
					s.record("alone", track,
						audio.Sine([]float64{100, 1000, 3000}[i], 1, rate, 0.8))
				}
			},
			want: func(by map[string]audio.Player) {
				s.Require().Equal(3, by["alone"].Records)
				s.Require().Empty(by["alone"].Scattered)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()
			tc.want(s.byID())
		})
	}
}

func TestCorpusPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CorpusPublicTestSuite))
}
