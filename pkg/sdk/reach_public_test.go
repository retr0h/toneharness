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

package sdk_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk"
)

// ReachPublicTestSuite covers answering whether a chain can meet a target from
// readings already committed, with nothing attached.
type ReachPublicTestSuite struct {
	suite.Suite

	corpus string
}

// SetupSuite builds the corpus once.
//
// Once rather than per test, because measuring one is reading audio and nine
// tests each reading it took five minutes where one reading takes thirty
// seconds. The tree is built rather than borrowed from resources/music/ so the
// assertions do not depend on whichever records somebody has on disk.
func (s *ReachPublicTestSuite) SetupSuite() {
	root := s.T().TempDir()

	// Three records over two players, each a different pitch, because a genre
	// is a distribution: one record has no spread on any axis and Aims drops
	// an axis nothing can say it arrived at. Measured from one record, punk
	// measures as nothing at all.
	for i, who := range []string{"a", "a", "b"} {
		track := fmt.Sprintf("t%d", i)
		stem := filepath.Join(root, who, "stems", "htdemucs", track)

		s.Require().NoError(os.MkdirAll(stem, 0o750))
		s.Require().NoError(os.WriteFile(filepath.Join(stem, "bass.wav"),
			recorded(60+40*float64(i)), 0o600))
	}

	for _, who := range []string{"a", "b"} {
		var manifest bytes.Buffer

		_, _ = fmt.Fprintf(&manifest, "artist: %s\ntracks:\n", who)

		for i := range 3 {
			if (who == "a") != (i < 2) {
				continue
			}

			_, _ = fmt.Fprintf(&manifest,
				"  - track: t%d\n    url: https://open.spotify.com/track/x%d\n"+
					"    year: 1994\n    genres: [punk]\n    genres_by: llm\n", i, i)
		}

		s.Require().NoError(os.WriteFile(
			filepath.Join(root, who, "corpus.yaml"), manifest.Bytes(), 0o600))
	}

	s.corpus = root
}

// recorded is a short recording at one pitch, as a 16-bit mono WAV.
//
// Synthesised rather than borrowed from resources/music/, so the assertions do
// not depend on whichever records somebody has on disk, and short so that
// measuring three of them is seconds rather than minutes.
func recorded(
	hz float64,
) []byte {
	const (
		rate    = 48000
		seconds = 2
		samples = rate * seconds
	)

	var body bytes.Buffer

	for i := range samples {
		at := float64(i) / rate

		// A fundamental with a little of its third, so the spectrum has
		// somewhere for a centroid to sit, and an envelope so notes start and
		// stop and a transient has something to read.
		v := 0.4*math.Sin(2*math.Pi*hz*at) + 0.1*math.Sin(2*math.Pi*hz*3*at)
		v *= math.Exp(-4 * math.Mod(at, 0.5))

		_ = binary.Write(&body, binary.LittleEndian, int16(v*32000))
	}

	var out bytes.Buffer

	out.WriteString("RIFF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(36+body.Len()))
	out.WriteString("WAVEfmt ")
	_ = binary.Write(&out, binary.LittleEndian, uint32(16))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint32(rate))
	_ = binary.Write(&out, binary.LittleEndian, uint32(rate*2))
	_ = binary.Write(&out, binary.LittleEndian, uint16(2))
	_ = binary.Write(&out, binary.LittleEndian, uint16(16))
	out.WriteString("data")
	_ = binary.Write(&out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())

	return out.Bytes()
}

func (s *ReachPublicTestSuite) at() (string, string) {
	return s.corpus,
		filepath.Join("..", "..", "resources", "sweeps", "hx-stomp")
}

func (s *ReachPublicTestSuite) ask() sdk.ReachAsk {
	corpus, sweeps := s.at()

	return sdk.ReachAsk{
		RigID: "matt-freeman", Genre: "punk", Corpus: corpus, Sweeps: sweeps,
	}
}

// TestItAnswersFromReadingsAlreadyTaken is the whole point.
//
// No device, no bench, no signal. Tune answers the same question with five
// minutes of real-time audio and a pedal held throughout; every number that
// answer rests on is already committed under resources/sweeps/.
func (s *ReachPublicTestSuite) TestItAnswersFromReadingsAlreadyTaken() {
	got, err := sdk.New().Reach(context.Background(), s.ask())

	s.Require().NoError(err)
	s.Require().Equal("matt-freeman", got.Rig)
	s.Require().Positive(got.Readings)
	s.Require().NotEmpty(got.Axes)

	// Worst first, so the axis that decides the run reads first.
	for i := range len(got.Axes) - 1 {
		if got.Axes[i].Within == got.Axes[i+1].Within {
			s.Require().GreaterOrEqual(got.Axes[i].Gap, got.Axes[i+1].Gap)
		}
	}
}

// TestABandShareIsComparedInTheTargetsUnits is the mistake that hides.
//
// A sweep reports a band share as a percentage and a corpus reports the same
// share as a fraction, so a range converted at one scale and compared at the
// other is out by a hundred. Out by a hundred on a band share reads as a
// perfectly ordinary number, and the axis quietly becomes unreachable.
func (s *ReachPublicTestSuite) TestABandShareIsComparedInTheTargetsUnits() {
	got, err := sdk.New().Reach(context.Background(), s.ask())
	s.Require().NoError(err)

	for _, v := range got.Axes {
		if v.Figure == "low" {
			s.Require().Less(v.From, 1.0,
				"a band share is a fraction here, not a percentage")

			return
		}
	}

	s.Require().Fail("the target named no low band")
}

// TestAChainNoBlockOfWhichHasBeenSwept covers having nothing to answer from.
//
// Refused rather than answered from nothing. A table of zeroes would read as a
// chain that can move nothing, which is a claim about the gear rather than
// about the readings.
func (s *ReachPublicTestSuite) TestAChainNoBlockOfWhichHasBeenSwept() {
	ask := s.ask()
	ask.Sweeps = s.T().TempDir()

	_, err := sdk.New().Reach(context.Background(), ask)
	s.Require().ErrorIs(err, sdk.ErrNoSweeps)
}

// TestABlockNobodyHasSweptIsNamed covers a partial answer.
func (s *ReachPublicTestSuite) TestABlockNobodyHasSweptIsNamed() {
	dir := s.T().TempDir()
	_, sweeps := s.at()

	// One block's readings and not the others', so the answer stands on
	// something while still being short.
	body, err := os.ReadFile(filepath.Join(sweeps, "HD2_AmpSVBeastBrt.json"))
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "HD2_AmpSVBeastBrt.json"), body, 0o600))

	ask := s.ask()
	ask.Sweeps = dir

	got, err := sdk.New().Reach(context.Background(), ask)

	s.Require().NoError(err)
	s.Require().NotEmpty(got.Unswept, "the cabinet has no readings here")
}

// TestASweepThatWillNotDecode covers a reading file somebody broke.
//
// Reported rather than skipped. A sweep that is there and unreadable is a
// different thing from one never taken, and treating it as missing would
// answer from a chain quietly short of a block.
func (s *ReachPublicTestSuite) TestASweepThatWillNotDecode() {
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "HD2_AmpSVBeastBrt.json"), []byte("{"), 0o600))

	ask := s.ask()
	ask.Sweeps = dir

	_, err := sdk.New().Reach(context.Background(), ask)
	s.Require().ErrorContains(err, "HD2_AmpSVBeastBrt.json")
}

// TestNoGenreIsNothingToAimAt covers a request aiming at nothing.
func (s *ReachPublicTestSuite) TestNoGenreIsNothingToAimAt() {
	ask := s.ask()
	ask.Genre = ""

	_, err := sdk.New().Reach(context.Background(), ask)
	s.Require().ErrorIs(err, sdk.ErrNothingToAimAt)
}

// TestAGenreNobodyTagged covers a target the corpus cannot place.
func (s *ReachPublicTestSuite) TestAGenreNobodyTagged() {
	ask := s.ask()
	ask.Genre = "skiffle"

	_, err := sdk.New().Reach(context.Background(), ask)
	s.Require().ErrorIs(err, sdk.ErrNothingToAimAt)
}

// TestACorpusThatIsNotThere covers a tree nobody has.
func (s *ReachPublicTestSuite) TestACorpusThatIsNotThere() {
	ask := s.ask()
	ask.Corpus = filepath.Join(s.T().TempDir(), "nowhere")

	_, err := sdk.New().Reach(context.Background(), ask)
	s.Require().Error(err)
}

// TestARigNobodyCurated covers gear the compiler cannot build.
func (s *ReachPublicTestSuite) TestARigNobodyCurated() {
	ask := s.ask()
	ask.RigID = "nobody-at-all"

	_, err := sdk.New().Reach(context.Background(), ask)
	s.Require().Error(err)
}

func TestReachPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReachPublicTestSuite))
}

// TestTheSweepsDefaultToTheCommittedTree covers naming neither tree.
//
// An agent over MCP names a rig and a genre and nothing else, so the defaults
// answer. The target comes from figures embedded in the binary; the readings
// do not, because they are a tree on disk and the default is repo-relative.
// Run from anywhere else they are not there, and the answer is an error rather
// than a table of zeroes.
func (s *ReachPublicTestSuite) TestTheSweepsDefaultToTheCommittedTree() {
	_, err := sdk.New().Reach(context.Background(), sdk.ReachAsk{
		RigID: "matt-freeman", Genre: "punk",
	})

	s.Require().ErrorIs(err, sdk.ErrNoSweeps,
		"the target resolved from the shipped figures and the readings did not")
}

// TestItReadsTheShippedFiguresWhenNobodyNamesACorpus is what makes it cheap
// enough to ask first.
//
// Measuring the corpus reads fifteen bass stems and takes 47 seconds, against
// milliseconds for everything else here. `just generate` already writes those
// figures into the binary, and a command whose whole point is costing less
// than the run it precedes cannot cost most of a minute.
func (s *ReachPublicTestSuite) TestItReadsTheShippedFiguresWhenNobodyNamesACorpus() {
	_, sweeps := s.at()

	started := time.Now()

	got, err := sdk.New().Reach(context.Background(), sdk.ReachAsk{
		RigID: "matt-freeman", Genre: "punk", Sweeps: sweeps,
	})

	s.Require().NoError(err)
	s.Require().NotEmpty(got.Axes)
	s.Require().Less(time.Since(started), 10*time.Second,
		"no audio was decoded, so this is arithmetic over committed readings")
}

// TestNamingACorpusMeasuresItRatherThanReadingTheShippedFigures covers the
// other half.
//
// A caller who named a tree means that tree. Answering from what was committed
// would ignore what they asked, and the figures would be of other records.
func (s *ReachPublicTestSuite) TestNamingACorpusMeasuresItRatherThanReadingTheShippedFigures() {
	ask := s.ask()
	ask.Corpus = filepath.Join(s.T().TempDir(), "nothing-here")

	_, err := sdk.New().Reach(context.Background(), ask)
	s.Require().Error(err,
		"the named tree is read even though punk is shipped with the binary")
}
