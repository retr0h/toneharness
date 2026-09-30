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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// MeasureRunTestSuite covers a campaign end to end, with a pedal that answers
// from memory and a bench that hands back a tone.
type MeasureRunTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockPedal
	out   string
	dry   string
}

func (s *MeasureRunTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockPedal(s.ctrl)
	// Every measuring command asks the client which models the attached
	// device has, rather than reading the built-in HX Stomp catalog, so that
	// --catalog and --device reach them.
	s.pedal.EXPECT().Catalog(gomock.Any()).
		DoAndReturn(func(context.Context) (*catalog.Catalog, error) {
			return catalog.BuiltIn()
		}).AnyTimes()
	s.out = filepath.Join(s.T().TempDir(), "measured.json")
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di-short.wav")

	// The preset read back and rebuilt: every command here takes its chain off
	// the measuring loop before playing it, sending the output to USB rather
	// than to the socket the measuring lead comes from. What that rewrite does
	// is offTheLoop's own test, so it is answered here rather than asserted.
	s.pedal.EXPECT().
		PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
}

// bench hands back a tone loud enough to measure.
type bench struct {
	quiet   bool
	clipped bool
	err     error
}

func (b bench) Through(
	_ context.Context,
	signal []float32,
) ([]float32, error) {
	if b.err != nil {
		return nil, b.err
	}

	amplitude := 0.2
	if b.quiet {
		amplitude = 0.000001
	}

	// Driven well past the ceiling and clamped there, which is what clipping
	// is: flat tops, an RMS close to one, and harmonics that were never in
	// the signal.
	if b.clipped {
		amplitude = 8
	}

	out := make([]float32, len(signal))
	for i := range out {
		v := amplitude * math.Sin(2*math.Pi*110*float64(i)/sdk.Rate)
		out[i] = float32(math.Max(-1, math.Min(1, v)))
	}

	return out, nil
}

func (bench) Name() string { return "a bench" }

// squealing is a loop oscillating: it answers with the same high tone whatever
// is put in, silence included.
//
// Which is the shape of the real fault. The measuring lead runs from the
// pedal's output back into its own input, so with enough gain around the loop
// the chain feeds itself, and what comes back is the loop's own note rather
// than the reference. Three of the device's amplifiers still did it at 30dB of
// headroom, reading between 92% and 95% of their energy above 2kHz where the
// median was 0.41%.
type squealing struct {
	// silences counts the readings taken with nothing put in, which is the
	// extra one a block the invariant cannot settle costs. A cabinet is a low
	// pass, so brighter than its input settles it outright and this stays at
	// nothing.
	silences *int
}

func (b squealing) Through(
	_ context.Context,
	signal []float32,
) ([]float32, error) {
	if b.silences != nil && nothingIn(signal) {
		*b.silences++
	}

	out := make([]float32, len(signal))

	// Well above the 2kHz the high band starts at, and the same whether the
	// caller sent music or nothing.
	for i := range out {
		out[i] = float32(0.3 * math.Sin(2*math.Pi*6000*float64(i)/sdk.Rate))
	}

	return out, nil
}

func (squealing) Name() string { return "a loop feeding itself" }

// nothingIn reports a signal with nothing in it, which is what a silence
// reading sends. Not `silent`, which is a level this package already names.
func nothingIn(
	signal []float32,
) bool {
	for _, v := range signal {
		if v != 0 {
			return false
		}
	}

	return len(signal) > 0
}

// routedPlan is a plan carrying the output entry a preset arrives with, which
// is what headroom needs something to change.
func routedPlan() plan.Plan {
	routing := map[string]json.RawMessage{
		"dsp0.outputA": json.RawMessage(
			`{"@model":"HelixStomp_AppDSPFlowOutputMain","@output":1,` +
				`"pan":0.5,"gain":0}`),
	}

	return plan.Plan{
		Name: "scratch",
		Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true,
		}},
		Device: &rig.DeviceState{Routing: &routing},
	}
}

// compiles makes the pedal build any preset it is asked for.
func (s *MeasureRunTestSuite) compiles() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()

	// And the read back that headroom does, which takes the routing off the
	// preset because a plan compiled from a rig carries none.
	s.pedal.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routedPlan()}, nil).AnyTimes()
}

// TestMeasureBlocks covers a whole campaign: the readings it takes, the
// backoff when the loop squeals, what it files as refused, and every way it
// stops before it has measured anything.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasureRunTestSuite) TestMeasureBlocks() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The backoff.
			//
			// The gain that makes the loop run away is the block's own, so
			// the headroom that stops it is per block. A bench answering the
			// same high tone whatever is put in is a loop feeding itself, and
			// the sweep takes the reading again with more headroom rather
			// than filing the squeal.
			//
			// A gate rather than a cabinet, which exercises the harder half.
			// A cabinet is a low pass and a reading brighter than the
			// reference settles it outright; a gate may legitimately be
			// brighter, so the only test left is whether the reading depends
			// on its input at all. Silence in, and an oscillation comes back
			// anyway.
			name: "a squealing loop is read again with more headroom",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var buf bytes.Buffer

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "gate",
					Seconds: 0.2, Bench: squealing{}, Headroom: -30,
				}))

				// Said and stepped, which is the whole behaviour: the reading is taken
				// again with more headroom rather than the squeal being filed.
				said := buf.String()
				s.Require().Contains(said, "read the loop rather than itself")
				s.Require().Contains(said, "again at -42dB")
				s.Require().Contains(said, "again at -54dB", "it keeps stepping")

				// And a block that never came clean is marked rather than filed. A
				// reading nobody can use is worse than a gap: a gap gets looked into.
				for _, got := range s.read().Blocks {
					s.Require().Contains(got.Refused, "read the loop rather than itself")
					s.Require().False(got.Measured())
				}
			},
		},
		{
			// The backoff staying out of the way.
			name: "a clean reading is not taken twice",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var buf bytes.Buffer

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{},
				}))

				s.Require().NotContains(buf.String(), "read the loop rather than itself")
			},
		},
		{
			// The instrument and the headroom.
			//
			// Every figure is a figure about one instrument through one loop,
			// and until these were recorded the only trace of either was the
			// reference's filename.
			name: "the readings say what they are about",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				s.Require().NoError(MeasureBlocks(context.Background(), &bytes.Buffer{},
					MeasureOptions{
						Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
						Seconds: 1, Bench: bench{}, Headroom: -30,
					}))

				lib := s.read()
				s.Require().Equal("bass", lib.Instrument)
				s.Require().InDelta(-30, lib.Headroom, 0.001)
			},
		},
		{
			// The half the invariant settles outright.
			//
			// A cabinet is a loudspeaker and a loudspeaker is a low pass, so
			// a reading brighter than the reference is a reading of something
			// else and there is nothing left to test. Only a block that may
			// legitimately brighten costs the extra reading.
			name: "a cabinet brighter than what it was given needs no silence reading",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var (
					buf      bytes.Buffer
					silences int
				)

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "cab",
					Seconds: 0.2, Bench: squealing{silences: &silences},
				}))

				s.Require().Contains(buf.String(), "read the loop rather than itself")

				// Not one. The message above is printed on both sides of this split, so
				// the silence readings are the only thing that tells them apart:
				// inverting the cabinet test leaves the message intact and this is what
				// notices.
				s.Require().NotEmpty(s.read().Blocks)
				s.Require().Zero(silences,
					"a loudspeaker cannot add high energy, so the invariant settles it "+
						"outright and nothing is played into silence")
			},
		},
		{
			// The other half.
			//
			// A gate may legitimately be brighter than what it was given, so
			// the invariant cannot settle it and the only question left is
			// whether the reading depends on its input at all. That costs a
			// second reading per block, which is what makes the cabinet case
			// worth having.
			// A block the loop cannot be asked about again is not a block
			// suspected of oscillating. The silence reading is what settles
			// it, so when the device will not load the block a second time
			// there is nothing to settle it with and the reading stands.
			name: "the device will not load a suspect block for its silence reading",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(nil).Times(2)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(errors.New("the device stopped taking the message")).
					AnyTimes()

				var (
					buf      bytes.Buffer
					silences int
				)

				s.Require().NoError(MeasureBlocks(
					context.Background(), &buf, MeasureOptions{
						Client: s.pedal, Dry: s.dry, Out: s.out,
						Category: "gate", Seconds: 0.2,
						Bench: squealing{silences: &silences},
					}))

				s.Require().NotEmpty(s.read().Blocks)
			},
		},
		{
			name: "a gate brighter than what it was given costs a silence reading",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var (
					buf      bytes.Buffer
					silences int
				)

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "gate",
					Seconds: 0.2, Bench: squealing{silences: &silences},
				}))

				s.Require().NotEmpty(s.read().Blocks)
				s.Require().Positive(silences,
					"the silence reading a cabinet does not need")
			},
		},
		{
			// The reading failing.
			name: "a bench that stops answering mid backoff",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				err := MeasureBlocks(context.Background(), &bytes.Buffer{}, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "gate",
					Seconds: 0.2, Bench: bench{err: errors.New("the interface went away")},
				})

				s.Require().ErrorContains(err, "the interface went away")
			},
		},
		{
			// A campaign that works.
			name: "blocks measures a category",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var buf bytes.Buffer

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{},
				}))

				lib := s.read()

				s.Require().Len(lib.Blocks, 8)
				s.Require().True(lib.Isolated)
				s.Require().NotEmpty(lib.Reference.SHA256)
				s.Require().Positive(lib.Baseline.Centroid,
					"the empty loop is measured first, or every block is a difference "+
						"from nothing")

				for _, block := range lib.Blocks {
					s.Require().Empty(block.Refused)
					s.Require().False(block.Clipped)
				}
			},
		},
		{
			// A reading at the converters' ceiling.
			//
			// A clipped spectrum is the clipping's rather than the block's,
			// so it reads as a bright block and would sit at the top of a
			// list of bright blocks.
			name: "blocks marks what clipped",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var buf bytes.Buffer

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{clipped: true},
				}))

				for _, block := range s.read().Blocks {
					s.Require().True(block.Clipped, "%s was not marked", block.ID)
					s.Require().False(block.Measured(), "a clipped reading is not usable")
				}

				s.Require().Contains(buf.String(), "CLIPPED")
			},
		},
		{
			// A device declining a preset.
			//
			// A refusal is a result rather than an end. Some of what the
			// catalog lists means nothing on its own, and a campaign has to
			// get past them to reach the rest.
			name: "blocks records what would not load",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(nil)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(errors.New("the device stopped taking the message")).AnyTimes()

				var buf bytes.Buffer

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{},
				}))

				for _, block := range s.read().Blocks {
					s.Require().NotEmpty(block.Refused)
				}

				s.Require().Contains(buf.String(), "refused")
			},
		},
		{
			// Why a long run survives a pedal that drops off.
			name: "blocks writes as it goes",
			then: func() {
				s.compiles()

				// The baseline, then one block, then the bench stops answering.
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var (
					buf   bytes.Buffer
					calls int
				)

				err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq", Seconds: 1,
					Bench: stopping{after: 3, calls: &calls},
				})

				s.Require().Error(err)

				lib := s.read()
				s.Require().NotEmpty(lib.Blocks,
					"what was measured before it stopped is on disk")
			},
		},
		{
			// A missing signal.
			name: "blocks reports a reference it cannot read",
			then: func() {
				var buf bytes.Buffer

				err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: "nowhere.wav", Out: s.out, Seconds: 1,
					Bench: bench{},
				})

				s.Require().ErrorContains(err, "nowhere.wav")
			},
		},
		{
			// The empty loop failing.
			//
			// Nothing below it would mean anything, so it stops rather than
			// measuring six hundred blocks against nothing.
			name: "blocks reports a baseline it cannot take",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(errors.New("no")).AnyTimes()

				var buf bytes.Buffer

				err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{},
				})

				s.Require().ErrorContains(err, "loading the empty loop")
			},
		},
		{
			// The compiler failing.
			name: "blocks reports a preset it cannot build",
			then: func() {
				s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
					Return(sdk.Built{}, errors.New("will not build")).AnyTimes()

				var buf bytes.Buffer

				err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{},
				})

				s.Require().ErrorContains(err, "building the empty loop")
			},
		},
		{
			// Read is the library a run wrote.
			// TestBlocksReportsACatalogItCannotRead covers --device naming a
			// pedal this binary ships no catalog for.
			//
			// Asked of the client rather than read from the built-in one, so
			// that --catalog and --device reach the measuring commands. A
			// campaign that cannot say which models exist has nothing to
			// measure.
			name: "blocks reports a catalog it cannot read",
			then: func() {
				ctrl := gomock.NewController(s.T())
				pedal := mocks.NewMockPedal(ctrl)
				pedal.EXPECT().Catalog(gomock.Any()).
					Return(nil, errors.New("no catalog for that pedal")).AnyTimes()

				var buf bytes.Buffer

				err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: pedal, Dry: s.dry, Out: s.out, Seconds: 1, Bench: bench{},
				})

				s.Require().ErrorContains(err, "no catalog for that pedal")
			},
		},
		{
			// A destination that is not there.
			//
			// The library is written after every block rather than at the
			// end, because a run that holds its results loses them all when
			// the pedal drops off the bus, which it has done. So the first
			// write is also the first chance to find out that the path is
			// wrong, and it is worth finding out then rather than after the
			// last block.
			name: "blocks reports somewhere it cannot write",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var buf bytes.Buffer

				err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Seconds: 1, Category: "eq",
					Out:   filepath.Join(s.T().TempDir(), "nowhere", "measured.json"),
					Bench: bench{},
				})

				s.Require().ErrorContains(err, "measured.json")
			},
		},
		{
			// --resume.
			//
			// A campaign is most of an hour and the pedal drops off the bus,
			// so the point is to not measure again what is already on disk.
			// The baseline comes back with it: a second baseline taken an
			// hour later is a different loop, and every block already in the
			// file was measured against the first one.
			name: "blocks resumes from what is already there",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var buf bytes.Buffer

				// One pass, then the same run again with --resume.
				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{},
				}))

				first := s.read()
				s.Require().NotEmpty(first.Blocks)

				var again bytes.Buffer

				s.Require().NoError(MeasureBlocks(context.Background(), &again, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{}, Resume: true,
				}))

				second := s.read()

				s.Require().Len(second.Blocks, len(first.Blocks))
				s.Require().InDelta(first.Baseline.Centroid, second.Baseline.Centroid, 0.001,
					"the baseline every block in the file was measured against")
			},
		},
		{
			// --resume with no file.
			//
			// Somebody who passes it on the first run of a campaign gets a
			// campaign rather than an error, because there is nothing in it
			// that is a problem.
			name: "blocks resuming from nothing measures everything",
			then: func() {
				s.compiles()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				var buf bytes.Buffer

				s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
					Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
					Seconds: 1, Bench: bench{}, Resume: true,
				}))

				s.Require().NotEmpty(s.read().Blocks)
			},
		},
		{
			// The five measuring commands all open the loop through benchFor,
			// and a run that cannot open it has nothing to measure.
			name: "no bench supplied and no interface of that name",
			then: func() {
				err := MeasureBlocks(context.Background(), &bytes.Buffer{},
					MeasureOptions{
						Client: s.pedal, Dry: s.dry, Out: s.out,
						Category: "eq", Seconds: 1,
						Hardware: "no such interface anybody owns",
					})

				s.Require().ErrorContains(err, "no such interface anybody owns")
			},
		},
		{
			// Every preset a campaign plays is written to a temporary
			// directory first, so a machine with nowhere to make one cannot
			// start. It is worth saying that before the pedal is asked for
			// anything.
			name: "nowhere to build the presets",
			then: func() {
				s.T().Setenv("TMPDIR",
					filepath.Join(s.T().TempDir(), "not a directory"))

				err := MeasureBlocks(context.Background(), &bytes.Buffer{},
					MeasureOptions{
						Client: s.pedal, Dry: s.dry, Out: s.out,
						Category: "eq", Seconds: 1, Bench: bench{},
					})

				s.Require().ErrorContains(err,
					"making somewhere to build presets")
			},
		},
		{
			// A block that never became a preset is filed with what went
			// wrong and the campaign carries on, the same as one the pedal
			// refused. The empty loop still has to build or there is nothing
			// to compare against, so this is the one case where the baseline
			// compiles and the blocks do not.
			name: "the compiler would not build a block's preset",
			then: func() {
				const empties = "HD2_EQSimple3Band"

				s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
					DoAndReturn(
						func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
							if !strings.Contains(in.Plan, empties) {
								return sdk.Built{},
									errors.New("the compiler would not have it")
							}

							s.Require().NoError(os.WriteFile(
								in.Out, []byte("a preset"), 0o600))

							return sdk.Built{}, nil
						}).AnyTimes()
				s.pedal.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
					Return(sdk.Reading{Plan: routedPlan()}, nil).AnyTimes()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				var buf bytes.Buffer

				s.Require().NoError(MeasureBlocks(
					context.Background(), &buf, MeasureOptions{
						Client: s.pedal, Dry: s.dry, Out: s.out,
						Category: "eq", Seconds: 1, Bench: bench{},
					}))

				refused := 0

				for _, block := range s.read().Blocks {
					if block.Refused != "" {
						refused++
					}
				}

				s.Require().NotZero(refused)
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

func (s *MeasureRunTestSuite) read() measured.Library {
	f, err := os.Open(s.out)
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	lib, err := measured.Load(f)
	s.Require().NoError(err)

	return lib
}

// turning is a bench that answers normally for a while and then differently.
//
// The noise floor is measured first, from the chain as it sits. A bench that
// was quiet or clipped from the very first reading would make that the
// settled level too, and nothing afterwards would stand out from it.
type turning struct {
	after int
	to    bench
	calls *int
}

func (b turning) Through(
	ctx context.Context,
	signal []float32,
) ([]float32, error) {
	*b.calls++

	if *b.calls > b.after {
		return b.to.Through(ctx, signal)
	}

	return bench{}.Through(ctx, signal)
}

func (turning) Name() string { return "a bench that changes" }

// stopping is a bench that answers a few times and then does not.
type stopping struct {
	after int
	calls *int
}

func (b stopping) Through(
	ctx context.Context,
	signal []float32,
) ([]float32, error) {
	*b.calls++

	if *b.calls > b.after {
		return nil, errors.New("stopped answering")
	}

	return bench{}.Through(ctx, signal)
}

func (stopping) Name() string { return "a bench that stops" }

func TestMeasureRunTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasureRunTestSuite))
}
