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
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// ReachPublicTestSuite covers asking how near a chain can get, from one pass.
//
// It reads the chain rather than resources/sweeps/, and that is the design
// rather than an implementation detail: every committed sweep was taken with
// its block alone, and an SV Beast with no cabinet has a median centroid of
// 8,139Hz where the chain reads about 144.
type ReachPublicTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockTuner
	genre *mocks.MockGenres
	dry   string
}

func (s *ReachPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockTuner(s.ctrl)
	// Every measuring command asks the client which models the attached
	// device has, rather than reading the built-in HX Stomp catalog, so that
	// --catalog and --device reach them.
	s.pedal.EXPECT().Catalog(gomock.Any()).
		DoAndReturn(func(context.Context) (*catalog.Catalog, error) {
			return catalog.BuiltIn()
		}).AnyTimes()
	s.genre = mocks.NewMockGenres(s.ctrl)
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di.wav")

	// The preset read back and rebuilt: every command here takes its chain off
	// the measuring loop before playing it, sending the output to USB rather
	// than to the socket the measuring lead comes from. What that rewrite does
	// is offTheLoop's own test, so it is answered here rather than asserted.
	s.pedal.EXPECT().
		PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, nil).AnyTimes()
}

func (s *ReachPublicTestSuite) opts() ReachOptions {
	return ReachOptions{
		Client: s.pedal, Genres: s.genre, Bench: &sloping{}, Dry: s.dry,
		ID: "matt-freeman", Genre: "punk", Corpus: "resources/music/bass",
		Seconds: 0.1, Takes: 2, Nudge: 0.1,
	}
}

// built makes the pedal answer everything one pass asks of it.
func (s *ReachPublicTestSuite) built() {
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true,
		}}}}, nil)
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
}

// wide is a target with room on every axis.
func (s *ReachPublicTestSuite) wide() []audio.Genre {
	band := audio.Spread{Low: -1000, Mid: 0, High: 1000}

	return []audio.Genre{{
		Name: "punk", Slug: "punk",
		Across: audio.Across{
			Tracks: 12, Low: band, Mid: band, High: band, Centroid: band,
		},
	}}
}

// TestItReadsTheChainRatherThanTheCommittedSweeps is the whole design.
func (s *ReachPublicTestSuite) TestItReadsTheChainRatherThanTheCommittedSweeps() {
	s.genre.EXPECT().
		MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.wide(), nil)
	s.built()

	w := buffer()

	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	said := w.String()
	s.Require().Contains(said, "dials through")
	s.Require().Contains(said, "the loop wanders")
	s.Require().Contains(said, "TOGETHER")
}

// TestNothingIsApplied is what separates this from a tuning pass.
//
// The chain is left where the compiler put it. Every control moved to read its
// slope is put back, so asking the question costs readings and changes nothing.
func (s *ReachPublicTestSuite) TestNothingIsApplied() {
	s.genre.EXPECT().
		MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.wide(), nil)
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true,
		}}}}, nil)
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	turned := map[sdk.Address]int{}

	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, a sdk.Address, _ float32) error {
			turned[a]++

			return nil
		}).AnyTimes()

	s.Require().NoError(Reach(context.Background(), buffer(), s.opts()))
	s.Require().NotEmpty(turned)

	// Two apiece and no more: moved to read a slope, moved back. A third would
	// be a solved step being applied, which is the tuning run this exists to
	// decide whether to spend.
	for a, n := range turned {
		s.Require().Equal(2, n, "%v was moved %d times", a, n)
	}
}

// TestAnAxisNothingCanCloseIsTheHeadline covers the answer that saves the
// afternoon.
func (s *ReachPublicTestSuite) TestAnAxisNothingCanCloseIsTheHeadline() {
	// A target far outside anything a chain of zero slopes can move to.
	s.genre.EXPECT().MeasuredGenres(gomock.Any(), gomock.Any()).
		Return([]audio.Genre{{
			Name: "punk", Slug: "punk",
			Across: audio.Across{
				Tracks:   12,
				Centroid: audio.Spread{Low: 900000, Mid: 1000000, High: 1100000},
			},
		}}, nil)
	s.built()

	w := buffer()

	err := Reach(context.Background(), w, s.opts())
	if err != nil {
		// A chain whose every slope against the target is zero is refused by
		// the solve rather than reported, which is its own honest answer.
		s.Require().ErrorIs(err, solve.ErrNoKnobs)

		return
	}

	s.Require().Contains(w.String(), "OUT OF REACH")
	s.Require().Contains(w.String(), "Change the chain, not the knobs")
}

// TestNoGenreIsNoTarget covers a request aiming at nothing.
func (s *ReachPublicTestSuite) TestNoGenreIsNoTarget() {
	s.genre.EXPECT().
		MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.wide(), nil)

	opts := s.opts()
	opts.Genre = "skiffle"

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), opts), ErrNoTarget)
}

// TestAGenreThatMeasuresAsNothing covers a tag with no figures behind it.
func (s *ReachPublicTestSuite) TestAGenreThatMeasuresAsNothing() {
	s.genre.EXPECT().MeasuredGenres(gomock.Any(), gomock.Any()).
		Return([]audio.Genre{{Name: "punk", Slug: "punk"}}, nil)
	s.built()

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), s.opts()), ErrNoTarget)
}

// TestTheShippedFiguresAnswerWhenNobodyNamesACorpus covers the default.
//
// Measuring the corpus reads fifteen bass stems and takes most of a minute,
// and `just generate` already writes those figures into the binary.
func (s *ReachPublicTestSuite) TestTheShippedFiguresAnswerWhenNobodyNamesACorpus() {
	s.built()

	opts := s.opts()
	opts.Corpus = ""

	// No MeasuredGenres expectation: naming no tree must not read one.
	s.Require().NoError(Reach(context.Background(), buffer(), opts))
}

// TestAGenreNothingShippedMeasures covers a word the binary does not carry.
func (s *ReachPublicTestSuite) TestAGenreNothingShippedMeasures() {
	opts := s.opts()
	opts.Corpus = ""
	opts.Genre = "skiffle"

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), opts), ErrNoTarget)
}

// TestARigThatWillNotBuild covers gear the catalog cannot realise.
func (s *ReachPublicTestSuite) TestARigThatWillNotBuild() {
	wanted := errors.New("no such rig")

	s.genre.EXPECT().
		MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.wide(), nil)
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{}, wanted)

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), s.opts()), wanted)
}

// TestAChainWithNoDial covers gear the solver cannot touch.
func (s *ReachPublicTestSuite) TestAChainWithNoDial() {
	s.genre.EXPECT().
		MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.wide(), nil)
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_NotAModel"), Pos: 0,
		}}}}, nil)
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), s.opts()), solve.ErrNoKnobs)
}

// TestAReferenceThatIsNotThere covers a missing signal.
func (s *ReachPublicTestSuite) TestAReferenceThatIsNotThere() {
	s.genre.EXPECT().
		MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.wide(), nil)
	s.built()

	opts := s.opts()
	opts.Dry = filepath.Join(s.T().TempDir(), "nothing.wav")

	s.Require().Error(Reach(context.Background(), buffer(), opts))
}

// TestTheCorpusWillNotRead covers a tree that is not there.
func (s *ReachPublicTestSuite) TestTheCorpusWillNotRead() {
	wanted := errors.New("no such corpus")

	s.genre.EXPECT().
		MeasuredGenres(gomock.Any(), gomock.Any()).Return(nil, wanted)

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), s.opts()), wanted)
}

// TestSpansOfCountsTravelEachWaySeparately covers a control near a stop.
//
// A dial at the top of its range can only go down, so the reachable range is
// not symmetric about where it sits. Counting a full range both ways promises
// movement the chain does not have.
func (s *ReachPublicTestSuite) TestSpansOfCountsTravelEachWaySeparately() {
	atTop := solve.Knob{
		Block: 0, Param: 1, At: 1, Low: 0, High: 1,
		Slope: map[audio.Figure]float64{audio.KeyCentroid: 100},
	}

	got := spansOf([]solve.Knob{atTop},
		map[audio.Figure]float64{audio.KeyCentroid: 500})

	at := got[audio.KeyCentroid]
	s.Require().InDelta(500, at.High, 0.001, "it cannot go up")
	s.Require().InDelta(400, at.Low, 0.001, "it has a whole turn downward")
	s.Require().InDelta(100, at.Swing, 0.001)
}

// TestTheHeadlineNamesEveryAxisTheJointSolveMisses covers the third verdict.
func (s *ReachPublicTestSuite) TestTheHeadlineNamesEveryAxisTheJointSolveMisses() {
	w := buffer()

	verdict(w, []solve.Verdict{
		{Figure: audio.KeyHigh, Gap: 390, Shown: true, Within: true, Together: 45.2},
		{Figure: audio.KeyLow, Gap: 9, Shown: true, Within: true, Together: 0.9},
	}, solve.Result{})

	said := w.String()
	s.Require().Contains(said, "no one set of positions reaches them")
	s.Require().Contains(said, "high by 45.2")
	s.Require().NotContains(said, "low by")
}

// TestAChainWithNoSlopeOnAnyAxisSaysSo covers the sentence with a hole in it.
func (s *ReachPublicTestSuite) TestAChainWithNoSlopeOnAnyAxisSaysSo() {
	w := buffer()

	verdict(w, []solve.Verdict{
		{Figure: audio.KeyHigh, Gap: 390, Shown: true, Within: true},
	}, solve.Result{})

	s.Require().Contains(w.String(), "no dial in this")
	s.Require().NotContains(w.String(), "misses .")
}

// TestATargetNamingNoAxisThisChainReads covers an empty answer.
func (s *ReachPublicTestSuite) TestATargetNamingNoAxisThisChainReads() {
	w := buffer()

	verdict(w, nil, solve.Result{})

	s.Require().Contains(w.String(), "Nothing to aim at")
}

// TestOneSetOfPositionsReachingEverything covers the happy verdict.
func (s *ReachPublicTestSuite) TestOneSetOfPositionsReachingEverything() {
	w := buffer()

	verdict(w, []solve.Verdict{
		{Figure: audio.KeyHigh, Gap: 3, Shown: true, Within: true, Together: 0.4},
	}, solve.Result{Arrived: true})

	said := w.String()
	s.Require().Contains(said, "reaches all 1 axes at once")
	s.Require().Contains(said, "several")
	// Contains rather than an Index comparison: strings.Index answers -1 for
	// a string that is not there, and -1 is less than any length, so the
	// assertion it replaces held whatever the verdict said.
	s.Require().Contains(said, "Worth running")
}

// TestTheTableSaysWhatItReadsAndWhatItWants covers the columns that matter.
//
// A distance in tolerances is unreadable on its own, and hiding the readings
// behind it hid a fault for a whole session: "high, 390 out" sounds like a
// chain that needs tuning, and "reads 0.8433, wants 0" is an 8x10 cabinet
// passing 84% of its energy above 2kHz, which no cabinet does.
func (s *ReachPublicTestSuite) TestTheTableSaysWhatItReadsAndWhatItWants() {
	w := buffer()

	table(w, []solve.Verdict{
		{Figure: audio.KeyHigh, From: 0.8433, Want: 0, Gap: 390.6, Together: 45.9},
		{Figure: audio.KeyLow, From: 0.1543, Want: 0.97, Gap: 9.4, Shown: true, Within: true},
	})

	said := w.String()
	s.Require().Contains(said, "READS")
	s.Require().Contains(said, "WANTS")
	s.Require().Contains(said, "0.8433")
	s.Require().Contains(said, "down", "which way it has to move")
	s.Require().Contains(said, "up")
	s.Require().Contains(said, "OUT OF REACH")
}

// TestReachRefusesAReferenceForTheOtherInstrument covers the guard.
//
// Every figure measured by pushing a guitar recording through a bass rig
// describes the recording rather than the chain, so what comes back is the
// wrong instrument rather than the wrong settings. Refused before a reading is
// taken.
func (s *ReachPublicTestSuite) TestReachRefusesAReferenceForTheOtherInstrument() {
	s.built()
	s.genre.EXPECT().MeasuredGenres(gomock.Any(), gomock.Any()).
		Return(s.wide(), nil).AnyTimes()

	// The same recording under a name that says guitar, because a reference's
	// instrument is read off its filename: what is in the file is nobody's to
	// know and a path is what somebody typed.
	body, err := os.ReadFile(s.dry)
	s.Require().NoError(err)

	wrong := filepath.Join(s.T().TempDir(), "guitar-di.wav")
	s.Require().NoError(os.WriteFile(wrong, body, 0o600))

	opts := s.opts()
	opts.Dry = wrong

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), opts), ErrWrongInstrument)
}

// TestReachReportsHardwareItCannotOpen covers --hardware naming no device.
func (s *ReachPublicTestSuite) TestReachReportsHardwareItCannotOpen() {
	s.built()
	s.genre.EXPECT().MeasuredGenres(gomock.Any(), gomock.Any()).
		Return(s.wide(), nil).AnyTimes()

	opts := s.opts()
	opts.Hardware = "no such interface"
	// Nil, because a caller who supplies a bench owns its lifetime and
	// benchFor hands that one straight back without ever looking at the name.
	opts.Bench = nil

	s.Require().Error(Reach(context.Background(), buffer(), opts))
}

func TestReachPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReachPublicTestSuite))
}
