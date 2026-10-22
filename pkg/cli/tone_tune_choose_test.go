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
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// ChooseTestSuite covers the list half of the tuning loop.
//
// The dials are solved together and a list is compared one setting at a time,
// and what is under test here is the comparing: which settings are read, which
// are thrown away because their figures describe something other than the
// setting, and which one the chain is left on.
type ChooseTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockTuner
	genre *mocks.MockGenres
	dry   string
}

func (s *ChooseTestSuite) SetupTest() {
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
	s.dry = "../../resources/dry/bass-di.wav"
}

// cab is a real model whose Mic is a list of twelve, which is the control this
// whole piece of work exists for.
const cab = catalog.ModelID("HD2_CabMicIr_2x15Brute")

func (s *ChooseTestSuite) catalog() *catalog.Catalog {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	return cat
}

func (s *ChooseTestSuite) opts() TuneOptions {
	return TuneOptions{
		Client: s.pedal, Genres: s.genre, Bench: bench{},
		ID: "matt-freeman", Genre: "punk", Corpus: "resources/music/bass",
		Dry: s.dry, Seconds: 0.1, Takes: 2, Passes: 2, Tries: 2, Nudge: 0.1,
	}
}

// aims is a target on the centroid with a tolerance of one, so a residual reads
// as the distance itself.
func (s *ChooseTestSuite) aims(
	want float64,
) map[audio.Figure]solve.Aim {
	return map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: want, Tol: 1}}
}

// mic is the chain's one list, addressed the way a preset records it.
func (s *ChooseTestSuite) mic() solve.Choice {
	got := listsOf(plan.Plan{Blocks: []plan.Block{{
		Model: cab, Pos: 0, Enabled: true,
		Params: plan.Params{"Mic": catalog.Int(4)},
	}}}, s.catalog())

	s.Require().Len(got, 1)

	return got[0]
}

// svt is a real swept amplifier, and it holds both ends of task 253 on one
// block. Against the centroid and the three band shares its Bass reads 0.911
// straight and its Master 0.063, so one of them is a line a slope carries
// across and the other has a slope that is true only where it was read. It is
// also the amplifier whose Sag and Hum a solve took to the stops in task 217,
// and whose Master a pass took from 1.0 to 0.037.
const svt = catalog.ModelID("HD2_AmpSVBeastBrt")

// sweeps is the committed curves, which say how far a slope carries.
const sweeps = "../../resources/sweeps/hx-stomp"

// TestCarrying covers how far each dial may be moved on one reading of its
// slope.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestCarrying() {
	chain := func(model catalog.ModelID) plan.Plan {
		return plan.Plan{Blocks: []plan.Block{{
			Model: model, Pos: 0, Enabled: true,
			Params: plan.Params{
				"Bass": catalog.Float(0.5), "Master": catalog.Float(0.5),
			},
		}}}
	}

	// Every control of this amplifier runs 0 to 1, so a carry reads off as a
	// fraction of the range without any arithmetic.
	carryOf := func(made plan.Plan, name string, aims map[audio.Figure]solve.Aim,
		dir string, nudge float64,
	) float64 {
		knobs := carrying(knobsOf(made, s.catalog()), made, dir, nudge, aims)

		for _, k := range knobs {
			if k.Setting == name {
				return k.Carry
			}
		}

		s.Require().Fail("no " + name + " among the dials")

		return 0
	}

	// The centroid and the three band shares, which is what a bass target
	// names and the distribution the floor was sized against.
	spectral := map[audio.Figure]solve.Aim{
		audio.KeyCentroid: {Want: 150, Tol: 5},
		audio.KeyLow:      {Want: 95, Tol: 1},
		audio.KeyMid:      {Want: 4, Tol: 1},
		audio.KeyHigh:     {Want: 1, Tol: 1},
	}

	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary case, and what a release gets: only the
			// fingerprints are packed into the binary, so without a sweep
			// there is nothing that says how far a slope carries.
			name: "no sweeps named caps nothing",
			then: func() {
				s.Require().Zero(carryOf(
					chain(svt), "Master", spectral, "", 0.1))
			},
		},
		{
			// A block the campaign never reached, inside a tree that holds
			// other blocks. A missing file is nothing rather than an error.
			name: "a block nobody swept caps nothing",
			then: func() {
				made := chain("HD2_AmpArchetype")

				for _, k := range carrying(
					knobsOf(made, s.catalog()), made, sweeps, 0.1, spectral) {
					s.Require().Zero(k.Carry, k.Setting)
				}
			},
		},
		{
			// The floor. This amplifier's Master reads 0.063 straight, which
			// would cap it at 6.3% of its range: shorter than the step that
			// measured the slope, and a cap tighter than the evidence.
			name: "a control that is no line carries one nudge",
			then: func() {
				s.Require().InDelta(0.1, carryOf(
					chain(svt), "Master", spectral, sweeps, 0.1), 1e-9,
					"the nudge is the distance the slope was read over")
			},
		},
		{
			// And the floor is the caller's rather than this file's: a run
			// reading its slopes over a wider step may take them further.
			name: "a wider nudge carries further",
			then: func() {
				s.Require().InDelta(0.25, carryOf(
					chain(svt), "Master", spectral, sweeps, 0.25), 1e-9)
			},
		},
		{
			// The other side of the floor, on the same block. This
			// amplifier's Bass reads 0.911 straight against these axes: it
			// really is a line, so a slope read anywhere on it is true
			// everywhere and nearly the whole range carries.
			name: "a control that is a line carries almost its whole range",
			then: func() {
				got := carryOf(chain(svt), "Bass", spectral, sweeps, 0.1)

				s.Require().Greater(got, 0.9,
					"a line was capped as though it were a hairpin")
				s.Require().LessOrEqual(got, 1.0,
					"nothing carries past its own range")
			},
		},
		{
			// The axes decide, and they decide more than anything else here.
			// The same Bass that is a line for the band shares reads 0.162
			// straight for the decay, so changing the target takes the same
			// control from carrying nearly its whole range to carrying a
			// sixth of it. A figure almost nothing moves in a straight line
			// really does deserve the tight cap, and that is the finding
			// rather than a fault.
			name: "the axes being solved decide the distance",
			then: func() {
				s.Require().InDelta(0.162, carryOf(
					chain(svt), "Bass",
					map[audio.Figure]solve.Aim{
						audio.KeyDecay: {Want: 0.5, Tol: 0.01},
					}, sweeps, 0.1), 0.001)
			},
		},
		{
			// A control the sweep read and left no fit for, which is every
			// list: a line through twelve microphones is meaningless by
			// construction, so there is no distance to carry a slope over.
			// Unreachable through knobsOf, which hands the solver floats only,
			// and the guard is what keeps it that way.
			name: "a control with no fit carries nothing",
			then: func() {
				s.Require().Zero(carries(measured.Curves{
					Controls: map[string]measured.Curve{"Mic": {
						Spread: map[audio.Figure]float64{audio.KeyCentroid: 11},
					}},
				}, "Mic", 11, 0.1, []audio.Figure{audio.KeyCentroid}))
			},
		},
		{
			// A chain holding two of a kind, which is ordinary: the sweep is
			// read once and both blocks are capped from it.
			name: "two of a kind read one sweep",
			then: func() {
				twice := chain(svt)
				twice.Blocks = append(twice.Blocks, plan.Block{
					Model: svt, Pos: 1, Enabled: true,
					Params: plan.Params{
						"Bass": catalog.Float(0.5), "Master": catalog.Float(0.5),
					},
				})

				knobs := carrying(
					knobsOf(twice, s.catalog()), twice, sweeps, 0.1, spectral)

				seen := 0

				for _, k := range knobs {
					if k.Setting != "Master" {
						continue
					}

					seen++

					s.Require().InDelta(0.1, k.Carry, 1e-9,
						"block %d read a different sweep", k.Block)
				}

				s.Require().Equal(2, seen, "both blocks' Masters")
			},
		},
		{
			// A file that is there and will not parse, which is nothing
			// rather than an error: this must never stop a run that was about
			// to measure something.
			name: "a sweep that will not parse caps nothing",
			then: func() {
				dir := s.T().TempDir()
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, string(svt)+".json"),
					[]byte("{not json"), 0o600))

				made := chain(svt)

				for _, k := range carrying(
					knobsOf(made, s.catalog()), made, dir, 0.1, spectral) {
					s.Require().Zero(k.Carry, k.Setting)
				}
			},
		},
		{
			// A directory that holds no curves at all.
			name: "a sweeps path with nothing in it caps nothing",
			then: func() {
				made := chain(svt)

				for _, k := range carrying(knobsOf(made, s.catalog()), made,
					s.T().TempDir(), 0.1, spectral) {
					s.Require().Zero(k.Carry, k.Setting)
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestListsOf covers listsOf, which is every control in the chain that is
// compared rather than turned.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestListsOf() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The counterpart to knobsOf.
			//
			// A cabinet carries both kinds, so the same block has to answer
			// one function with its dials and the other with its lists, and
			// neither may see the other's controls. A list in the matrix is a
			// row built from a slope that does not exist; a dial in the
			// comparison is twelve readings of something that has a slope and
			// did not need them.
			name: "lists of finds the lists and leaves the dials alone",
			then: func() {
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
			},
		},
		{
			// The controls a solve may not turn.
			//
			// Modelled valve behaviour rather than tone: Sag is the supply
			// drooping, Bias and BiasX how hard the valves are run, Hum and
			// Ripple the mains in the signal. A band share closed by rebiasing
			// the output valves closes as well as one closed with the tone
			// stack, and only one of them is what anybody meant: a run took Sag
			// to 1.000 and Hum to 0.775 and the chain sounded worse than the rig
			// it started from.
			//
			// Left where the rig put them rather than dropped. A rig stating a
			// Sag still gets it; what cannot happen is the loop deciding one.
			name: "knobs of leaves the amplifier's valves alone",
			then: func() {
				amp := catalog.ModelID("HD2_AmpSVBeastBrt")
				made := plan.Plan{Blocks: []plan.Block{{
					Model: amp, Pos: 0, Enabled: true,
				}}}

				got := map[string]bool{}
				for _, k := range knobsOf(made, s.catalog()) {
					got[k.Setting] = true
				}

				for _, name := range []string{"Sag", "Bias", "BiasX", "Hum", "Ripple"} {
					s.Require().False(got[name],
						"%s is the amplifier's own behaviour, not a dial for a target", name)
				}

				// And the tone stack is still every bit of it, because taking
				// the wrong five away would leave a solve nothing to turn.
				for _, name := range []string{"Bass", "Mid", "Treble", "Drive", "ChVol", "Master"} {
					s.Require().True(got[name], "%s is a tone control and stays", name)
				}
			},
		},
		{
			// The kind that used to be skipped.
			//
			// A switch is a choice of two and is compared on the same
			// machinery: the ranking does not care how many settings there
			// are. It carries Flip, because the device does not coerce and
			// setting one takes the other wire call.
			//
			// An SVT 4 Pro is the case worth having. It carries four switches
			// beside its dials, and one of them is a Bright that moves a bass
			// chain further than several of the knobs the solver was already
			// spending readings on.
			name: "lists of finds a switch as a choice of two",
			then: func() {
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
			},
		},
		{
			// A chain built elsewhere.
			name: "lists of on a block the catalog does not carry",
			then: func() {
				s.Require().Empty(listsOf(plan.Plan{Blocks: []plan.Block{{
					Model: catalog.ModelID("HD2_NotAModel"), Pos: 0,
				}}}, s.catalog()))
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

// TestASwitchIsSetWithTheOtherCall is why Flip is carried at all.
//
// The device does not coerce. A switch sent the index 1 is refused with the
// same error it gives for a block that is not there, which reads as the address
// being wrong rather than the value, so the two cases cannot share a call.
func (s *ChooseTestSuite) TestASwitchIsSetWithTheOtherCall() {
	ctrl := gomock.NewController(s.T())
	pedal := mocks.NewMockTuner(ctrl)

	pedal.EXPECT().
		Switch(gomock.Any(), sdk.Control(0, 3), true).
		Return(nil)

	s.Require().NoError(choose(context.Background(),
		TuneOptions{Client: pedal},
		solve.Choice{Block: 0, Param: 3, Flip: true, Options: []int{0, 1}}, 1))
}

// TestCompared covers compared, which tries every setting of every list and
// leaves each on its nearest.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestCompared() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The whole point.
			//
			// The bench answers a different centroid every reading, so the
			// twelve settings read twelve different figures and exactly one
			// of them is nearest the target. What is checked is the last
			// Choose, because that is what the chain is playing when the
			// dials are then solved.
			name: "compared leaves the chain on the nearest setting",
			then: func() {
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
			},
		},
		{
			// What a person reads.
			//
			// The gap between them is the useful number: nothing between them
			// says this control does not matter for this target, and
			// tolerances between them say it is the most important thing in
			// the chain.
			name: "compared reports the winner and the runner up",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				w := buffer()

				_, _, err := compared(context.Background(), w, s.opts(),
					&sloping{}, make([]float32, 64), []solve.Choice{s.mic()},
					s.aims(166), -200)

				s.Require().NoError(err)
				s.Require().Contains(w.String(), "Mic: 12 of 12 settings read")
				s.Require().Contains(w.String(), "next nearest is")
			},
		},
		{
			// Every setting failing.
			//
			// Said rather than silent, and the chain left alone rather than
			// put on setting zero. A control nothing could read is not a
			// control that reads best at its first setting.
			name: "compared leaves a list it could not read where it was",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				w := buffer()

				ranked, _, err := compared(context.Background(), w, s.opts(),
					bench{quiet: true}, make([]float32, 64), []solve.Choice{s.mic()},
					s.aims(166), 0)

				s.Require().NoError(err)
				s.Require().Empty(ranked)
				s.Require().Contains(w.String(), "left where the compiler put it")
			},
		},
		{
			// A loop that cannot measure.
			name: "compared gives up when the bench does",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				wanted := errors.New("the interface went away")

				_, _, err := compared(context.Background(), buffer(), s.opts(),
					bench{err: wanted}, make([]float32, 64), []solve.Choice{s.mic()},
					s.aims(166), -200)

				s.Require().ErrorIs(err, wanted)
			},
		},
		{
			// The one refusal that is fatal.
			//
			// A setting refused while the comparison walks it is one fewer
			// candidate and the other eleven still answer. The same refusal
			// on the setting that won leaves the chain on whichever setting
			// the walk happened to end on, which is not the answer and not a
			// chain anybody asked for.
			name: "compared gives up when the winner will not apply",
			then: func() {
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

// TestReadings covers readings, which is what the chain measures at each
// setting of one list.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestReadings() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The guard.
			//
			// Three ways a reading describes something other than the
			// microphone, and the clipping one is the dangerous one because
			// it reads as a finding: one microphone of a cabinet's twelve
			// clipped and read a centroid of 4,471Hz where the other eleven
			// sat between 126 and 147. Scored, it wins every target asking
			// for a bright sound, and the answer is a cabinet nobody would
			// have chosen.
			name: "readings throws away a setting whose figures are not of the setting",
			then: func() {
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
			},
		},
		{
			// Affordable is a different question from believable.
			//
			// A setting loud enough to read is not a setting the target may pay
			// for. `MidFreq` going from 0 to 2 cost 7.8dB on hardware, which
			// nothing reported and nothing undid, and the pass after it put its
			// own dials back and left the chain at that reading.
			name: "readings throws away a setting that costs the level",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				w := buffer()

				// Settled at nothing, against a bench reading well under it: loud
				// enough to be believed and far past what may be spent.
				got, err := readings(context.Background(), w, s.opts(), bench{},
					make([]float32, 64), s.mic(), 0)

				s.Require().NoError(err)
				s.Require().Empty(got, "every setting cost more than it may")
				s.Require().Contains(w.String(), "of level, past the 6 a target may spend")
			},
		},
		{
			// A refusal.
			//
			// Reported and skipped rather than fatal. A device refusing one
			// index says nothing about the other eleven, and giving up on the
			// whole comparison would throw away eleven readings already paid
			// for.
			name: "readings carries on past a setting the device refuses",
			then: func() {
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
			},
		},
		{
			// The one fatal case.
			//
			// A bench that cannot hand a signal back is not a setting that
			// read badly. It is the loop losing its ability to measure
			// anything, so carrying on would rank twelve settings on nothing.
			name: "readings gives up when the bench itself fails",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				wanted := errors.New("the interface went away")

				_, err := readings(context.Background(), buffer(), s.opts(),
					bench{err: wanted}, make([]float32, 64), s.mic(), -200)

				s.Require().ErrorIs(err, wanted)
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

// TestAttempt covers attempt, which is the two halves interleaved: choose the
// lists, solve the dials, and back up to another setting if the solve fell
// short.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestAttempt() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A chain of dials only.
			//
			// Most chains. An amplifier into a cabinet of floats has nothing
			// to compare, so it must not pay for the comparison or print a
			// heading about it.
			name: "attempt with no lists is just the solve",
			then: func() {
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
			},
		},
		{
			// The whole interleave.
			//
			// The order is what is under test: the heading, then a list left
			// on a setting, then a solve, then a second solve from another
			// setting because the first fell short.
			name: "attempt compares then solves then backs up",
			then: func() {
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
			},
		},
		{
			// The first half failing.
			name: "attempt gives up if the comparison does",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				wanted := errors.New("the interface went away")

				_, err := attempt(context.Background(), buffer(), s.opts(),
					bench{err: wanted}, make([]float32, 64), "preset.hlx", nil,
					[]solve.Choice{s.mic()}, s.aims(0), -200)

				s.Require().ErrorIs(err, wanted)
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

// TestBackedUp covers backedUp, which tries the next-nearest settings,
// solving the dials from each.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChooseTestSuite) TestBackedUp() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The interleave.
			//
			// A setting that reads nearest before any dial has moved is not
			// necessarily the one a solve can finish from, so the loop backs
			// up. What it must not do is report the last attempt: the answer
			// to a target is the best chain anybody demonstrated.
			name: "backed up tries the runner up and keeps the better of the two",
			then: func() {
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
					[]solve.Choice{mic}, s.aims(0), -200, nil, runners, first)

				s.Require().NoError(err)
				s.Require().Equal([]int{9}, chosen, "the runner-up was actually applied")
				s.Require().Contains(w.String(), "Mic goes to 9")
				s.Require().InDelta(4, solve.Worst(got.residual), 0.001,
					"a worse attempt does not replace a better one")
			},
		},
		{
			// The bound on the expensive half.
			//
			// Each attempt is a whole convergence, which on a chain of a
			// dozen dials over three passes is around forty readings. Twelve
			// of those is not a tuning session, it is an afternoon.
			name: "backed up stops at tries",
			then: func() {
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
					[]solve.Choice{mic}, s.aims(0), -200, nil, runners,
					round{residual: map[audio.Figure]float64{audio.KeyCentroid: 400}})

				s.Require().NoError(err)
				s.Require().Len(chosen, 2, "three tries, one of which was already spent")
			},
		},
		{
			// A stale ranking, which is a programming error rather than a
			// device one and so is skipped rather than reported.
			name: "backed up ignores a runner naming a control the chain does not have",
			then: func() {
				knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

				first := round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}}

				got, err := backedUp(context.Background(), buffer(), s.opts(), &sloping{},
					make([]float32, 64), "preset.hlx", []solve.Knob{knob}, nil,
					s.aims(0), -200,
					nil,
					[]solve.Runner{{Where: solve.Where{Block: 9, Param: 9}}}, first)

				s.Require().NoError(err)
				s.Require().InDelta(4, solve.Worst(got.residual), 0.001)
			},
		},
		{
			// The happy exit.
			//
			// Every attempt past an arrival is a convergence spent improving
			// on a chain already inside its tolerances, which is a quarter of
			// an hour buying nothing.
			name: "backed up stops on the first attempt that arrives",
			then: func() {
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
					[]solve.Choice{mic}, arrives, -200, nil, runners,
					round{residual: map[audio.Figure]float64{audio.KeyCentroid: 400}})

				s.Require().NoError(err)
				s.Require().True(got.arrived)
				s.Require().Len(chosen, 1, "it stopped as soon as one arrived")
			},
		},
		{
			// The apply failing.
			name: "backed up reports a device that refuses the runner up",
			then: func() {
				wanted := errors.New("no reply to opcode 30 within 6s")

				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(wanted)

				mic := s.mic()

				_, err := backedUp(context.Background(), buffer(), s.opts(), &sloping{},
					make([]float32, 64), "preset.hlx", nil, []solve.Choice{mic},
					s.aims(0), -200,
					nil,
					[]solve.Runner{{Where: mic.Where(), Option: solve.Option{Value: 9}}},
					round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}})

				s.Require().ErrorIs(err, wanted)
			},
		},
		{
			// A setting that was affordable when it was read and is not any more.
			//
			// The comparison priced every setting against the chain it ran on, and
			// the dials have moved since. `MidFreq` going to 2 was inside the
			// tolerance when it was read and cost 6dB by the time a run backed up
			// onto it, which nothing noticed and nothing undid.
			name: "backed up puts a setting back when it now costs the level",
			then: func() {
				var chosen []int

				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ sdk.Address, at int) error {
						chosen = append(chosen, at)

						return nil
					}).AnyTimes()

				mic := s.mic()
				w := buffer()

				// Settled at nothing, against a bench reading well under it, so
				// whatever is chosen costs more than may be spent.
				_, err := backedUp(context.Background(), w, s.opts(), bench{},
					make([]float32, 64), "preset.hlx", nil,
					[]solve.Choice{mic}, s.aims(0), 0,
					map[solve.Where][]solve.Option{mic.Where(): {{Value: 3}}},
					[]solve.Runner{{Where: mic.Where(), Option: solve.Option{Value: 9}}},
					round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}})

				s.Require().NoError(err)
				s.Require().Contains(w.String(), "of level from here")
				s.Require().Equal([]int{9, 3}, chosen,
					"it tried the setting, then put the list back where it was")
			},
		},
		{
			// The convergence failing, one reading later.
			//
			// Its own row because backing up reads the chain before it solves, to
			// price the setting it just chose, so a bench that fails at once is
			// caught there rather than in the solve.
			name: "backed up reports a solve that cannot measure after the price",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				wanted := errors.New("the interface went away")
				mic := s.mic()

				_, err := backedUp(context.Background(), buffer(), s.opts(),
					&quits{after: 1, err: wanted}, make([]float32, 64), "preset.hlx",
					nil, []solve.Choice{mic}, s.aims(0), -200, nil,
					[]solve.Runner{{Where: mic.Where(), Option: solve.Option{Value: 9}}},
					round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}})

				s.Require().ErrorIs(err, wanted)
			},
		},
		{
			// The device refusing to put a setting back.
			//
			// Reported rather than swallowed, because the run would otherwise go on
			// believing the list is where it left it while the device holds the
			// setting that was just rejected for costing too much.
			name: "backed up reports a device that will not put a setting back",
			then: func() {
				wanted := errors.New("that index is not one")

				gomock.InOrder(
					s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), 9).
						Return(nil),
					s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), 3).
						Return(wanted),
				)

				mic := s.mic()

				_, err := backedUp(context.Background(), buffer(), s.opts(), bench{},
					make([]float32, 64), "preset.hlx", nil,
					[]solve.Choice{mic}, s.aims(0), 0,
					map[solve.Where][]solve.Option{mic.Where(): {{Value: 3}}},
					[]solve.Runner{{Where: mic.Where(), Option: solve.Option{Value: 9}}},
					round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}})

				s.Require().ErrorIs(err, wanted)
			},
		},
		{
			// The convergence failing.
			name: "backed up reports a solve that cannot measure",
			then: func() {
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				wanted := errors.New("the interface went away")
				mic := s.mic()

				_, err := backedUp(context.Background(), buffer(), s.opts(),
					bench{err: wanted}, make([]float32, 64), "preset.hlx", nil,
					[]solve.Choice{mic}, s.aims(0), -200,
					nil,
					[]solve.Runner{{Where: mic.Where(), Option: solve.Option{Value: 9}}},
					round{residual: map[audio.Figure]float64{audio.KeyCentroid: 4}})

				s.Require().ErrorIs(err, wanted)
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

func TestChooseTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ChooseTestSuite))
}
