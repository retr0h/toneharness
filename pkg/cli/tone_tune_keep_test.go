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
package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// KeepTestSuite covers keeping what the chain played, as files somebody can
// listen to.
//
// The loop answers in ten figures and reduces everything else away. Ten figures
// can every one sit inside tolerance while the sound is plainly wrong, and the
// first run that kept the audio proved it: the figures said the loop was
// oscillating and the recording was a flat noise floor, which is the opposite
// fault with the opposite fix.
type KeepTestSuite struct {
	suite.Suite
}

// opts is a run that keeps nothing, which every row turns on for itself.
func (s *KeepTestSuite) opts() TuneOptions {
	return TuneOptions{Player: "mike-dirnt", Corpus: "resources/music/bass"}
}

// TestReturned covers telling a dead loop from a chain.
//
// One method and one table, so a case is a row rather than a file.
func (s *KeepTestSuite) TestReturned() {
	for _, tt := range []struct {
		name string
		sent float64
		got  float64
		dead bool
	}{
		{
			// The working loop, measured: a bass reference in, a chain back.
			name: "a loop that carries",
			sent: -13, got: -28.9,
		},
		{
			// The dead one, measured on the same rig an hour later.
			name: "a loop that returns nothing",
			sent: -16.1, got: -81.9, dead: true,
		},
		{
			// A chain may be louder than what it was handed, which an amplifier
			// certainly is, and that is not a fault.
			name: "a chain louder than the signal",
			sent: -30, got: -10,
		},
		{
			// Exactly at the line is not past it.
			name: "the width of the threshold itself",
			sent: 0, got: -silent,
		},
	} {
		s.Run(tt.name, func() {
			say, dead := Returned(tt.sent, tt.got)

			s.Require().Equal(tt.dead, dead)

			if !tt.dead {
				s.Require().Empty(say)

				return
			}

			s.Require().Contains(say, "Nothing is returning")
			s.Require().Contains(say, "of loss and not a chain")
		})
	}
}

// TestKeepTakes covers writing the pair of readings out.
func (s *KeepTestSuite) TestKeepTakes() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Off unless asked for, because a campaign is hundreds of readings.
			name: "keeping nothing writes nothing",
			then: func() {
				dir := s.T().TempDir()
				at := filepath.Join(dir, "unused")

				s.Require().NoError(keepTakes(buffer(), s.opts(),
					make([]float32, 8), make([]float32, 8)))

				_, err := os.Stat(at)
				s.Require().Error(err, "nothing was written")
			},
		},
		{
			// Both halves, because one on its own says little: the dry file is
			// what every chain is measured with and the take is what this chain
			// did to it.
			name: "keeping writes what went in and what came back",
			then: func() {
				dir := s.T().TempDir()

				opts := s.opts()
				opts.Keep = filepath.Join(dir, "listen")

				w := buffer()

				s.Require().NoError(keepTakes(w, opts,
					make([]float32, 64), make([]float32, 64)))

				for _, name := range []string{"mike-dirnt-dry.wav", "mike-dirnt-chain.wav"} {
					_, err := os.Stat(filepath.Join(opts.Keep, name))
					s.Require().NoError(err, "%s is there", name)
					s.Require().Contains(w.String(), name)
				}
			},
		},
		{
			// A genre names itself, because there is no player to name.
			name: "a genre run names itself",
			then: func() {
				opts := TuneOptions{Genre: "punk", Keep: filepath.Join(s.T().TempDir(), "k")}

				s.Require().NoError(keepTakes(buffer(), opts,
					make([]float32, 8), make([]float32, 8)))

				_, err := os.Stat(filepath.Join(opts.Keep, "punk-dry.wav"))
				s.Require().NoError(err)
			},
		},
		{
			// Somewhere to put them that will not take a file.
			name: "keeping reports a take it cannot write",
			then: func() {
				opts := s.opts()
				opts.Keep = filepath.Join(s.T().TempDir(), "listen")

				// A directory standing where the first file has to go.
				s.Require().NoError(os.MkdirAll(
					filepath.Join(opts.Keep, "mike-dirnt-dry.wav"), 0o750))

				s.Require().Error(keepTakes(buffer(), opts,
					make([]float32, 8), make([]float32, 8)))
			},
		},
		{
			// Nowhere to put them.
			name: "keeping reports a directory it cannot make",
			then: func() {
				flat := filepath.Join(s.T().TempDir(), "file")
				s.Require().NoError(os.WriteFile(flat, []byte("x"), 0o600))

				opts := s.opts()
				opts.Keep = filepath.Join(flat, "under-a-file")

				s.Require().Error(keepTakes(buffer(), opts,
					make([]float32, 8), make([]float32, 8)))
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestDeadLoop covers reporting a loop that gave nothing back.
func (s *KeepTestSuite) TestDeadLoop() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The pair is kept first, because a refusal is when somebody most
			// wants to listen.
			name: "a dead loop keeps what it heard and says so",
			then: func() {
				opts := s.opts()
				opts.Keep = filepath.Join(s.T().TempDir(), "listen")

				err := deadLoop(buffer(), opts, make([]float32, 8),
					make([]float32, 8), "66dB of loss")

				s.Require().ErrorIs(err, ErrNoReturn)
				s.Require().ErrorContains(err, "66dB of loss")

				_, statErr := os.Stat(filepath.Join(opts.Keep, "mike-dirnt-chain.wav"))
				s.Require().NoError(statErr, "and the take is on disk")
			},
		},
		{
			// Nowhere to keep it is reported instead, because a caller told the
			// loop is dead and not told the audio was lost has been told half.
			name: "a dead loop reports audio it could not keep",
			then: func() {
				flat := filepath.Join(s.T().TempDir(), "file")
				s.Require().NoError(os.WriteFile(flat, []byte("x"), 0o600))

				opts := s.opts()
				opts.Keep = filepath.Join(flat, "under-a-file")

				err := deadLoop(buffer(), opts, make([]float32, 8),
					make([]float32, 8), "66dB of loss")

				s.Require().Error(err)
				s.Require().NotErrorIs(err, ErrNoReturn)
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestKeepPlayed covers the reading taken at the end of a run.
func (s *KeepTestSuite) TestKeepPlayed() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "keeping nothing takes no reading",
			then: func() {
				gone := errors.New("the bench would have been asked")

				s.Require().NoError(keepPlayed(context.Background(), buffer(),
					s.opts(), bench{err: gone}, make([]float32, 64)))
			},
		},
		{
			name: "keeping takes one more reading",
			then: func() {
				opts := s.opts()
				opts.Keep = filepath.Join(s.T().TempDir(), "listen")

				s.Require().NoError(keepPlayed(context.Background(), buffer(),
					opts, bench{}, make([]float32, 64)))

				_, err := os.Stat(filepath.Join(opts.Keep, "mike-dirnt-chain.wav"))
				s.Require().NoError(err)
			},
		},
		{
			name: "keeping reports a bench that will not answer",
			then: func() {
				gone := errors.New("the bench gave up")

				opts := s.opts()
				opts.Keep = filepath.Join(s.T().TempDir(), "listen")

				s.Require().ErrorIs(keepPlayed(context.Background(), buffer(),
					opts, bench{err: gone}, make([]float32, 64)), gone)
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

// corpusWithStems builds the layout a measured player has on disk.
//
// Built rather than pointed at the real one, because the recordings are not in
// this repository and never will be: they are somebody's records, the tree says
// so, and a test that needs them passes here and fails everywhere else.
func (s *KeepTestSuite) corpusWithStems(
	player string,
	records ...string,
) string {
	root := s.T().TempDir()

	for _, record := range records {
		at := filepath.Join(root, player, "stems", "htdemucs", record)
		s.Require().NoError(os.MkdirAll(at, 0o750))
		s.Require().NoError(os.WriteFile(
			filepath.Join(at, "bass.wav"), []byte("not really audio"), 0o600))
	}

	return root
}

// TestHeard covers naming the bass a target was measured from.
func (s *KeepTestSuite) TestHeard() {
	for _, tt := range []struct {
		name string
		// player and records are the tree that gets built, where there is one.
		player  string
		records []string
		// who and where are what the run was aimed at.
		who   string
		genre string
		says  string
	}{
		{
			// The point of the pair: something to hold the two takes against.
			name:    "a player's own stems",
			player:  "mike-dirnt",
			records: []string{"longview", "holiday"},
			who:     "mike-dirnt",
			says:    "bass.wav",
		},
		{
			// A genre is the spread across every player who plays it, so there
			// is no one recording it can be held against.
			name:    "a genre names nothing",
			player:  "mike-dirnt",
			records: []string{"longview"},
			genre:   "punk",
		},
		{
			name:    "a player nobody has measured",
			player:  "mike-dirnt",
			records: []string{"longview"},
			who:     "nobody",
		},
	} {
		s.Run(tt.name, func() {
			opts := TuneOptions{Player: tt.who, Genre: tt.genre}
			if tt.player != "" {
				opts.Corpus = s.corpusWithStems(tt.player, tt.records...)
			}

			w := buffer()

			heard(w, opts)

			if tt.says == "" {
				s.Require().Empty(w.String())

				return
			}

			s.Require().Contains(w.String(), tt.says)
			s.Require().Contains(w.String(), "was measured from")

			for _, record := range tt.records {
				s.Require().Contains(w.String(), record, "every record is named")
			}
		})
	}
}

// TestHeardWithNoCorpus covers a run that cannot look anywhere.
func (s *KeepTestSuite) TestHeardWithNoCorpus() {
	w := buffer()

	heard(w, TuneOptions{Player: "mike-dirnt"})

	s.Require().Empty(w.String())
}

func TestKeepTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KeepTestSuite))
}
