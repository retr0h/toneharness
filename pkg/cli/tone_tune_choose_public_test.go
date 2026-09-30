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

// ChoosePublicTestSuite covers the list half of the tuning loop.
//
// The dials are solved together and a list is compared one setting at a time,
// and what is under test here is the comparing: which settings are read, which
// are thrown away because their figures describe something other than the
// setting, and which one the chain is left on.
type ChoosePublicTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockTuner
	genre *mocks.MockGenres
	dry   string
}

func (s *ChoosePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockTuner(s.ctrl)
	s.genre = mocks.NewMockGenres(s.ctrl)
	s.dry = "../../resources/dry/bass-di.wav"
}

// cab is a real model whose Mic is a list of twelve, which is the control this
// whole piece of work exists for.
const cab = catalog.ModelID("HD2_CabMicIr_2x15Brute")

func (s *ChoosePublicTestSuite) catalog() *catalog.Catalog {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	return cat
}

func (s *ChoosePublicTestSuite) opts() TuneOptions {
	return TuneOptions{
		Client: s.pedal, Genres: s.genre, Bench: bench{},
		ID: "matt-freeman", Genre: "punk", Corpus: "resources/music/bass",
		Dry: s.dry, Seconds: 0.1, Takes: 2, Passes: 2, Tries: 2, Nudge: 0.1,
	}
}

// aims is a target on the centroid with a tolerance of one, so a residual reads
// as the distance itself.
func (s *ChoosePublicTestSuite) aims(
	want float64,
) map[audio.Figure]solve.Aim {
	return map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: want, Tol: 1}}
}

// mic is the chain's one list, addressed the way a preset records it.
func (s *ChoosePublicTestSuite) mic() solve.Choice {
	got := listsOf(plan.Plan{Blocks: []plan.Block{{
		Model: cab, Pos: 0, Enabled: true,
		Params: plan.Params{"Mic": catalog.Int(4)},
	}}}, s.catalog())

	s.Require().Len(got, 1)

	return got[0]
}

// TestListsOfFindsTheListsAndLeavesTheDialsAlone is the counterpart to knobsOf.
//
// A cabinet carries both kinds, so the same block has to answer one function
// with its dials and the other with its lists, and neither may see the other's
// controls. A list in the matrix is a row built from a slope that does not
// exist; a dial in the comparison is twelve readings of something that has a
// slope and did not need them.
func (s *ChoosePublicTestSuite) TestListsOfFindsTheListsAndLeavesTheDialsAlone() {
	made := plan.Plan{Blocks: []plan.Block{{
		Model: cab, Pos: 0, Enabled: true,
		Params: plan.Params{"Mic": catalog.Int(4)},
	}}}

	lists := listsOf(made, s.catalog())
	s.Require().Len(lists, 1)

	one := lists[0]
	s.Require().Equal("Mic", one.Setting)
	s.Require().Equal(0, one.Block)
	s.Require().Equal(4, one.At, "where the compiler left it")
	s.Require().Len(one.Options, 12, "twelve microphones, every one of them")
	s.Require().Equal(0, one.Options[0])
	s.Require().Equal(11, one.Options[11])

	for _, k := range knobsOf(made, s.catalog()) {
		s.Require().NotEqual("Mic", k.Setting,
			"a list has no slope, so the solver may not be handed one")
	}
}

// TestListsOfFindsASwitchAsAChoiceOfTwo covers the kind that used to be skipped.
//
// A switch is a choice of two and is compared on the same machinery: the
// ranking does not care how many settings there are. It carries Flip, because
// the device does not coerce and setting one takes the other wire call.
//
// An SVT 4 Pro is the case worth having. It carries four switches beside its
// dials, and one of them is a Bright that moves a bass chain further than
// several of the knobs the solver was already spending readings on.
func (s *ChoosePublicTestSuite) TestListsOfFindsASwitchAsAChoiceOfTwo() {
	made := plan.Plan{Blocks: []plan.Block{{
		Model: catalog.ModelID("HD2_PreampSVT4Pro"), Pos: 0, Enabled: true,
		Params: plan.Params{"Bright": catalog.Bool(true)},
	}}}

	lists := listsOf(made, s.catalog())

	var bright solve.Choice

	for _, c := range lists {
		if c.Setting == "Bright" {
			bright = c
		}
	}

	s.Require().Equal("Bright", bright.Setting, "a switch is compared")
	s.Require().True(bright.Flip)
	s.Require().Equal([]int{0, 1}, bright.Options,
		"off and on, because false and true is not a range to step through")
	s.Require().Equal(1, bright.At, "where the compiler left it")

	for _, k := range knobsOf(made, s.catalog()) {
		s.Require().NotEqual("Bright", k.Setting,
			"a switch has no slope, so the solver may not be handed one")
	}
}

// TestASwitchIsSetWithTheOtherCall is why Flip is carried at all.
//
// The device does not coerce. A switch sent the index 1 is refused with the
// same error it gives for a block that is not there, which reads as the address
// being wrong rather than the value, so the two cases cannot share a call.
func (s *ChoosePublicTestSuite) TestASwitchIsSetWithTheOtherCall() {
	ctrl := gomock.NewController(s.T())
	pedal := mocks.NewMockTuner(ctrl)

	pedal.EXPECT().
		Switch(gomock.Any(), sdk.Control(0, 3), true).
		Return(nil)

	s.Require().NoError(choose(context.Background(),
		TuneOptions{Client: pedal},
		solve.Choice{Block: 0, Param: 3, Flip: true, Options: []int{0, 1}}, 1))
}

// TestListsOfOnABlockTheCatalogDoesNotCarry covers a chain built elsewhere.
func (s *ChoosePublicTestSuite) TestListsOfOnABlockTheCatalogDoesNotCarry() {
	s.Require().Empty(listsOf(plan.Plan{Blocks: []plan.Block{{
		Model: catalog.ModelID("HD2_NotAModel"), Pos: 0,
	}}}, s.catalog()))
}

// TestComparedLeavesTheChainOnTheNearestSetting is the whole point.
//
// The bench answers a different centroid every reading, so the twelve settings
// read twelve different figures and exactly one of them is nearest the target.
// What is checked is the last Choose, because that is what the chain is playing
// when the dials are then solved.
func (s *ChoosePublicTestSuite) TestComparedLeavesTheChainOnTheNearestSetting() {
	var chosen []int

	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.Address, at int) error {
			chosen = append(chosen, at)

			return nil
		}).AnyTimes()

	// 117Hz at the first reading and climbing seven each time, so the setting
	// read eighth is the one nearest 166.
	ranked, named, err := compared(context.Background(), buffer(), s.opts(),
		&sloping{}, make([]float32, 64), []solve.Choice{s.mic()},
		s.aims(166), -200)

	s.Require().NoError(err)
	s.Require().Len(ranked, 1)
	s.Require().Len(named, 1)

	order := ranked[s.mic().Where()]
	s.Require().Len(order, 12)
	s.Require().Equal(order[0].Value, chosen[len(chosen)-1],
		"the chain is left on the nearest, not on the last one tried")

	for i := range len(order) - 1 {
		s.Require().LessOrEqual(order[i].Worst, order[i+1].Worst)
	}
}

// TestComparedReportsTheWinnerAndTheRunnerUp covers what a person reads.
//
// The gap between them is the useful number: nothing between them says this
// control does not matter for this target, and tolerances between them say it
// is the most important thing in the chain.
func (s *ChoosePublicTestSuite) TestComparedReportsTheWinnerAndTheRunnerUp() {
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	w := buffer()

	_, _, err := compared(context.Background(), w, s.opts(),
		&sloping{}, make([]float32, 64), []solve.Choice{s.mic()},
		s.aims(166), -200)

	s.Require().NoError(err)
	s.Require().Contains(w.String(), "Mic: 12 of 12 settings read")
	s.Require().Contains(w.String(), "next nearest is")
}

// TestReadingsThrowsAwayASettingWhoseFiguresAreNotOfTheSetting is the guard.
//
// Three ways a reading describes something other than the microphone, and the
// clipping one is the dangerous one because it reads as a finding: one
// microphone of a cabinet's twelve clipped and read a centroid of 4,471Hz where
// the other eleven sat between 126 and 147. Scored, it wins every target asking
// for a bright sound, and the answer is a cabinet nobody would have chosen.
func (s *ChoosePublicTestSuite) TestReadingsThrowsAwayASettingWhoseFiguresAreNotOfTheSetting() {
	tests := []struct {
		name  string
		bench sdk.Bench
		says  string
	}{
		{"silent", bench{quiet: true}, "silent, so its figures are of the noise"},
		{
			"clipped",
			bench{clipped: true},
			"clipped, so its figures are the converters'",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil).AnyTimes()

			w := buffer()

			got, err := readings(context.Background(), w, s.opts(), tt.bench,
				make([]float32, 64), s.mic(), 0)

			s.Require().NoError(err)
			s.Require().Empty(got, "no setting of it produced a figure about it")
			s.Require().Contains(w.String(), tt.says)
		})
	}
}

// TestReadingsCarriesOnPastASettingTheDeviceRefuses covers a refusal.
//
// Reported and skipped rather than fatal. A device refusing one index says
// nothing about the other eleven, and giving up on the whole comparison would
// throw away eleven readings already paid for.
func (s *ChoosePublicTestSuite) TestReadingsCarriesOnPastASettingTheDeviceRefuses() {
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), 0).
		Return(errors.New("that index is not one"))
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	w := buffer()

	got, err := readings(context.Background(), w, s.opts(), &sloping{},
		make([]float32, 64), s.mic(), -200)

	s.Require().NoError(err)
	s.Require().Len(got, 11, "eleven of the twelve still answered")
	s.Require().NotContains(got, 0)
	s.Require().Contains(w.String(), "Mic 0   refused")
}

// TestComparedLeavesAListItCouldNotReadWhereItWas covers every setting failing.
//
// Said rather than silent, and the chain left alone rather than put on setting
// zero. A control nothing could read is not a control that reads best at its
// first setting.
func (s *ChoosePublicTestSuite) TestComparedLeavesAListItCouldNotReadWhereItWas() {
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	w := buffer()

	ranked, _, err := compared(context.Background(), w, s.opts(),
		bench{quiet: true}, make([]float32, 64), []solve.Choice{s.mic()},
		s.aims(166), 0)

	s.Require().NoError(err)
	s.Require().Empty(ranked)
	s.Require().Contains(w.String(), "left where the compiler put it")
}

// TestReadingsGivesUpWhenTheBenchItselfFails covers the one fatal case.
//
// A bench that cannot hand a signal back is not a setting that read badly. It
// is the loop losing its ability to measure anything, so carrying on would
// rank twelve settings on nothing.
func (s *ChoosePublicTestSuite) TestReadingsGivesUpWhenTheBenchItselfFails() {
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	wanted := errors.New("the interface went away")

	_, err := readings(context.Background(), buffer(), s.opts(),
		bench{err: wanted}, make([]float32, 64), s.mic(), -200)

	s.Require().ErrorIs(err, wanted)
}

// TestAttemptWithNoListsIsJustTheSolve covers a chain of dials only.
//
// Most chains. An amplifier into a cabinet of floats has nothing to compare, so
// it must not pay for the comparison or print a heading about it.
func (s *ChoosePublicTestSuite) TestAttemptWithNoListsIsJustTheSolve() {
	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

	w := buffer()

	// A tolerance wider than the bench can read, so the chain arrives on the
	// first pass and no slope is spent.
	arrives := map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 0, Tol: 100000}}

	_, err := attempt(context.Background(), w, s.opts(), bench{},
		make([]float32, 64), "preset.hlx", []solve.Knob{knob}, nil, arrives, -200)

	s.Require().NoError(err)
	s.Require().NotContains(w.String(), "comparing")
}

// TestBackedUpTriesTheRunnerUpAndKeepsTheBetterOfTheTwo is the interleave.
//
// A setting that reads nearest before any dial has moved is not necessarily the
// one a solve can finish from, so the loop backs up. What it must not do is
// report the last attempt: the answer to a target is the best chain anybody
// demonstrated.
func (s *ChoosePublicTestSuite) TestBackedUpTriesTheRunnerUpAndKeepsTheBetterOfTheTwo() {
	var chosen []int

	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.Address, at int) error {
			chosen = append(chosen, at)

			return nil
		}).AnyTimes()

	mic := s.mic()
	knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

	// A first attempt that is four tolerances out, and one alternative.
	first := round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}}

	runners := []solve.Runner{{
		Where: mic.Where(), Control: mic.Control,
		Option: solve.Option{Value: 9, Worst: 5},
	}}

	w := buffer()

	got, err := backedUp(context.Background(), w, s.opts(), &sloping{},
		make([]float32, 64), "preset.hlx", []solve.Knob{knob},
		[]solve.Choice{mic}, s.aims(0), -200, runners, first)

	s.Require().NoError(err)
	s.Require().Equal([]int{9}, chosen, "the runner-up was actually applied")
	s.Require().Contains(w.String(), "Mic goes to 9")
	s.Require().InDelta(4, furthest(got.residual), 0.001,
		"a worse attempt does not replace a better one")
}

// TestBackedUpStopsAtTries covers the bound on the expensive half.
//
// Each attempt is a whole convergence, which on a chain of a dozen dials over
// three passes is around forty readings. Twelve of those is not a tuning
// session, it is an afternoon.
func (s *ChoosePublicTestSuite) TestBackedUpStopsAtTries() {
	var chosen []int

	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.Address, at int) error {
			chosen = append(chosen, at)

			return nil
		}).AnyTimes()

	mic := s.mic()
	knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

	runners := make([]solve.Runner, 0, 11)
	for at := range 11 {
		runners = append(runners, solve.Runner{
			Where: mic.Where(), Control: mic.Control,
			Option: solve.Option{Value: at, Worst: float64(at)},
		})
	}

	opts := s.opts()
	opts.Tries = 3

	_, err := backedUp(context.Background(), buffer(), opts, &sloping{},
		make([]float32, 64), "preset.hlx", []solve.Knob{knob},
		[]solve.Choice{mic}, s.aims(0), -200, runners,
		round{residual: map[audio.Figure]float64{audio.KeyCentroid: 400}})

	s.Require().NoError(err)
	s.Require().Len(chosen, 2, "three tries, one of which was already spent")
}

// TestBackedUpIgnoresARunnerNamingAControlTheChainDoesNotHave covers a stale
// ranking, which is a programming error rather than a device one and so is
// skipped rather than reported.
func (s *ChoosePublicTestSuite) TestBackedUpIgnoresARunnerNamingAControlTheChainDoesNotHave() {
	knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

	first := round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}}

	got, err := backedUp(context.Background(), buffer(), s.opts(), &sloping{},
		make([]float32, 64), "preset.hlx", []solve.Knob{knob}, nil,
		s.aims(0), -200,
		[]solve.Runner{{Where: solve.Where{Block: 9, Param: 9}}}, first)

	s.Require().NoError(err)
	s.Require().InDelta(4, furthest(got.residual), 0.001)
}

func TestChoosePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ChoosePublicTestSuite))
}

// TestComparedGivesUpWhenTheBenchDoes covers a loop that cannot measure.
func (s *ChoosePublicTestSuite) TestComparedGivesUpWhenTheBenchDoes() {
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	wanted := errors.New("the interface went away")

	_, _, err := compared(context.Background(), buffer(), s.opts(),
		bench{err: wanted}, make([]float32, 64), []solve.Choice{s.mic()},
		s.aims(166), -200)

	s.Require().ErrorIs(err, wanted)
}

// TestComparedGivesUpWhenTheWinnerWillNotApply covers the one refusal that is
// fatal.
//
// A setting refused while the comparison walks it is one fewer candidate and
// the other eleven still answer. The same refusal on the setting that won
// leaves the chain on whichever setting the walk happened to end on, which is
// not the answer and not a chain anybody asked for.
func (s *ChoosePublicTestSuite) TestComparedGivesUpWhenTheWinnerWillNotApply() {
	wanted := errors.New("no reply to opcode 30 within 6s")
	seen := 0

	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.Address, _ int) error {
			seen++

			// Every setting of the walk answers, and the thirteenth call is
			// the one putting the chain on the winner.
			if seen > 12 {
				return wanted
			}

			return nil
		}).AnyTimes()

	_, _, err := compared(context.Background(), buffer(), s.opts(),
		&sloping{}, make([]float32, 64), []solve.Choice{s.mic()},
		s.aims(166), -200)

	s.Require().ErrorIs(err, wanted)
}

// TestAttemptComparesThenSolvesThenBacksUp covers the whole interleave.
//
// The order is what is under test: the heading, then a list left on a setting,
// then a solve, then a second solve from another setting because the first fell
// short.
func (s *ChoosePublicTestSuite) TestAttemptComparesThenSolvesThenBacksUp() {
	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

	opts := s.opts()
	opts.Passes = 1

	w := buffer()

	_, err := attempt(context.Background(), w, opts, &sloping{},
		make([]float32, 64), "preset.hlx", []solve.Knob{knob},
		[]solve.Choice{s.mic()}, s.aims(0), -200)

	s.Require().NoError(err)
	s.Require().Contains(w.String(), "comparing 1 list(s)")
	s.Require().Contains(w.String(), "nearest is")
	s.Require().Contains(w.String(), "and the dials are solved again")
}

// TestAttemptGivesUpIfTheComparisonDoes covers the first half failing.
func (s *ChoosePublicTestSuite) TestAttemptGivesUpIfTheComparisonDoes() {
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	wanted := errors.New("the interface went away")

	_, err := attempt(context.Background(), buffer(), s.opts(),
		bench{err: wanted}, make([]float32, 64), "preset.hlx", nil,
		[]solve.Choice{s.mic()}, s.aims(0), -200)

	s.Require().ErrorIs(err, wanted)
}

// TestBackedUpStopsOnTheFirstAttemptThatArrives covers the happy exit.
//
// Every attempt past an arrival is a convergence spent improving on a chain
// already inside its tolerances, which is a quarter of an hour buying nothing.
func (s *ChoosePublicTestSuite) TestBackedUpStopsOnTheFirstAttemptThatArrives() {
	var chosen []int

	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.Address, at int) error {
			chosen = append(chosen, at)

			return nil
		}).AnyTimes()

	mic := s.mic()
	knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

	opts := s.opts()
	opts.Tries = 9

	// A tolerance wider than the bench can read, so the first backed-up
	// attempt arrives and the other eight are never spent.
	arrives := map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 0, Tol: 100000}}

	runners := make([]solve.Runner, 0, 3)
	for at := range 3 {
		runners = append(runners, solve.Runner{
			Where: mic.Where(), Control: mic.Control,
			Option: solve.Option{Value: at, Worst: float64(at)},
		})
	}

	got, err := backedUp(context.Background(), buffer(), opts, bench{},
		make([]float32, 64), "preset.hlx", []solve.Knob{knob},
		[]solve.Choice{mic}, arrives, -200, runners,
		round{residual: map[audio.Figure]float64{audio.KeyCentroid: 400}})

	s.Require().NoError(err)
	s.Require().True(got.arrived)
	s.Require().Len(chosen, 1, "it stopped as soon as one arrived")
}

// TestBackedUpReportsADeviceThatRefusesTheRunnerUp covers the apply failing.
func (s *ChoosePublicTestSuite) TestBackedUpReportsADeviceThatRefusesTheRunnerUp() {
	wanted := errors.New("no reply to opcode 30 within 6s")

	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(wanted)

	mic := s.mic()

	_, err := backedUp(context.Background(), buffer(), s.opts(), &sloping{},
		make([]float32, 64), "preset.hlx", nil, []solve.Choice{mic},
		s.aims(0), -200,
		[]solve.Runner{{Where: mic.Where(), Option: solve.Option{Value: 9}}},
		round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}})

	s.Require().ErrorIs(err, wanted)
}

// TestBackedUpReportsASolveThatCannotMeasure covers the convergence failing.
func (s *ChoosePublicTestSuite) TestBackedUpReportsASolveThatCannotMeasure() {
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	wanted := errors.New("the interface went away")
	mic := s.mic()

	_, err := backedUp(context.Background(), buffer(), s.opts(),
		bench{err: wanted}, make([]float32, 64), "preset.hlx", nil,
		[]solve.Choice{mic}, s.aims(0), -200,
		[]solve.Runner{{Where: mic.Where(), Option: solve.Option{Value: 9}}},
		round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}})

	s.Require().ErrorIs(err, wanted)
}
