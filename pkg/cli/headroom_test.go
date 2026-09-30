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
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"

	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// HeadroomPublicTestSuite covers taking a preset off the measuring loop.
type HeadroomPublicTestSuite struct {
	suite.Suite
}

// routed is a plan carrying the output entry a preset arrives with.
//
// `"@output":1` is what the blank template gives it, and entry 1 is
// `Multi (1/4", XLR, Digital, USB 1/2)`, which drives the socket the measuring
// lead comes from. So this is the plan that feeds itself, which is the one
// worth testing against.
func routed() plan.Plan {
	entry := json.RawMessage(`{"@model":"HelixStomp_AppDSPFlowOutputMain",` +
		`"@output":1,"pan":0.5,"gain":0}`)

	routing := map[string]json.RawMessage{
		outputSlot:     entry,
		"dsp0.outputB": json.RawMessage(`{"@output":0}`),
	}

	return plan.Plan{
		Name: "scratch",
		Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true,
		}},
		Device: &rig.DeviceState{Routing: &routing},
	}
}

// TestItSendsTheChainOffTheLoopAndLeavesEverythingElse is the whole job.
//
// The destination is the half that matters. A chain sent to Multi arrives back
// at its own input down the measuring lead, and a high-gain amplifier has
// enough of its own gain to keep that oscillating however far the output is
// turned down. Sending to USB 1/2 alone reaches the computer without reaching
// the socket.
//
// TestOffTheLoop covers offTheLoop, which is the plan with its output entry
// sent somewhere the measuring.
//
// One method and one table, so a case is a row rather than a file.
func (s *HeadroomPublicTestSuite) TestOffTheLoop() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// turning it down lowers the level without touching the tone. The amplifier's
			// ChVol and Master are the tone, and lowering those would have the solver
			// solve for a different sound.
			name: "it sends the chain off the loop and leaves everything else",
			then: func() {
				got, err := offTheLoop(routed(), 10, -30)
				s.Require().NoError(err)

				var fields map[string]any
				s.Require().NoError(
					json.Unmarshal((*got.Device.Routing)[outputSlot], &fields))

				s.Require().InDelta(10, fields[sendKey], 0.001, "off the loop")
				s.Require().InDelta(-30, fields[gainKey], 0.001)
				s.Require().InDelta(0.5, fields["pan"], 0.001, "the rest is left alone")
				s.Require().Equal("HelixStomp_AppDSPFlowOutputMain", fields["@model"])
				s.Require().Contains(*got.Device.Routing, "dsp0.outputB",
					"the other entries travel")
			},
		},
		{
			// plan is written out and compared and a routing map shared between two of
			// them makes the comparison meaningless.
			name: "the plan handed over is not changed",
			then: func() {
				was := routed()

				_, err := offTheLoop(was, 10, -30)
				s.Require().NoError(err)

				var fields map[string]any
				s.Require().NoError(
					json.Unmarshal((*was.Device.Routing)[outputSlot], &fields))

				s.Require().InDelta(0, fields[gainKey], 0.001)
				s.Require().InDelta(1, fields[sendKey], 0.001)
			},
		},
		{
			// that feeds itself. While zero returned the preset as built, `--headroom 0`
			// left it sending to the socket the lead comes from, so the one setting that
			// asked for an honest level got the least honest reading.
			name: "no headroom still comes off the loop",
			then: func() {
				got, err := offTheLoop(routed(), 10, 0)
				s.Require().NoError(err)

				var fields map[string]any
				s.Require().NoError(
					json.Unmarshal((*got.Device.Routing)[outputSlot], &fields))

				s.Require().InDelta(10, fields[sendKey], 0.001)
				s.Require().InDelta(0, fields[gainKey], 0.001, "the gain is left as it was")
			},
		},
		{
			// routing at all, because the inputs and outputs come from the blank template
			// while the preset is written, and returning the loud one silently is how a
			// guard comes to pass on a chain nobody protected.
			name: "off the loop refuses what it cannot rewrite",
			then: func() {
				entry := func(body string) plan.Plan {
					routing := map[string]json.RawMessage{outputSlot: json.RawMessage(body)}

					return plan.Plan{Device: &rig.DeviceState{Routing: &routing}}
				}

				for _, tt := range []struct {
					name string
					give plan.Plan
					by   float64
					is   error
					says string
				}{
					{
						name: "it carries no routing at all",
						give: plan.Plan{},
						by:   -30,
						is:   ErrNoOutput,
					},
					{
						name: "it carries routing with no output entry",
						give: func() plan.Plan {
							routing := map[string]json.RawMessage{
								"dsp0.inputA": json.RawMessage(`{}`),
							}

							return plan.Plan{Device: &rig.DeviceState{Routing: &routing}}
						}(),
						by: -30,
						is: ErrNoOutput,
					},
					{
						name: "the output entry will not decode",
						give: entry(`{`),
						by:   -30,
						says: outputSlot,
					},
					{
						// JSON has no way to spell a NaN, so an entry built from one
						// would be written as a preset nothing can read. It is refused
						// where it is encoded rather than where it is typed, because that
						// is the one place every caller passes through.
						name: "the gain is not a number",
						give: routed(),
						by:   math.NaN(),
						says: gainKey,
					},
				} {
					s.Run(tt.name, func() {
						_, err := offTheLoop(tt.give, 10, tt.by)

						if tt.is != nil {
							s.Require().ErrorIs(err, tt.is)

							return
						}

						s.Require().ErrorContains(err, tt.says)
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestThePlanHandedOverIsNotChanged covers the copy.
//

// TestTheDestinationComesFromThePresetsOwnDevice covers asking the right device.
//
// Every catalog shipped today puts `USB 1/2` at 10, so this asserts a habit
// rather than a difference: each case is checked against what that device's own
// list says, so the day one of them moves it, this passes and the measuring rig
// keeps working. Asserting 10 four times would pass that day too, and be wrong.
func (s *HeadroomPublicTestSuite) TestTheDestinationComesFromThePresetsOwnDevice() {
	for _, tt := range []struct {
		name   string
		device *int
	}{
		{name: "an HX Stomp", device: ptr(catalog.HXStomp)},
		{name: "a Helix LT", device: ptr(catalog.HelixLT)},
		{name: "a preset naming no device", device: nil},
		{name: "a device no catalog ships for", device: ptr(1)},
	} {
		s.Run(tt.name, func() {
			made := routed()
			made.Device.Id = tt.device

			to, err := sendTo(made)
			s.Require().NoError(err)

			want, err := catalogFor(made)
			s.Require().NoError(err)

			at, ok := want.DestinationAt("USB 1/2")
			s.Require().True(ok)
			s.Require().Equal(at, to,
				"the number this device files it under, not another's")
		})
	}
}

// ptr is an address for a literal, which a preset's device id wants.
func ptr[T any](
	v T,
) *T {
	return &v
}

// TestNoHeadroomStillComesOffTheLoop is why zero no longer means untouched.
//

// TestOffTheLoopRefusesWhatItCannotRewrite covers every plan that cannot come
// off the measuring loop.
//

func TestHeadroomPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HeadroomPublicTestSuite))
}

// TestNoHeadroomStillRewritesThePreset covers asking for no headroom.
//
// TestQuieter covers quieter, which writes the plan again off the measuring
// loop, and returns the preset.
//
// One method and one table, so a case is a row rather than a file.
func (s *HeadroomPublicTestSuite) TestQuieter() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// opens the loop and it has to be set whatever the gain is, so the preset is
			// rebuilt: `--headroom 0` used to be the one setting that left a chain
			// measuring itself.
			name: "no headroom still rewrites the preset",
			then: func() {
				ctrl := gomock.NewController(s.T())
				pedal := mocks.NewMockTuner(ctrl)
				work := s.T().TempDir()

				pedal.EXPECT().
					PresetFile(gomock.Any(), "already.hlx").
					Return(sdk.Reading{Plan: routed()}, nil)
				pedal.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					Return(sdk.Built{}, nil)

				got, err := quieter(context.Background(),
					pedal, "already.hlx", work, "matt-freeman", 0)

				s.Require().NoError(err)
				s.Require().NotEqual("already.hlx", got)
			},
		},
		{
			// come from the blank template while the preset is written. The preset has
			// them, so reading it back is how the output entry arrives complete, with its
			// model and its output already set.
			name: "it reads the preset back because a rig carries no routing",
			then: func() {
				ctrl := gomock.NewController(s.T())
				pedal := mocks.NewMockTuner(ctrl)

				pedal.EXPECT().PresetFile(gomock.Any(), "already.hlx").
					Return(sdk.Reading{Plan: routed()}, nil)

				var built sdk.Compile

				pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
						built = in

						return sdk.Built{}, nil
					})

				got, err := quieter(context.Background(),
					pedal, "already.hlx", s.T().TempDir(), "matt-freeman", -30)

				s.Require().NoError(err)
				s.Require().NotEqual("already.hlx", got, "it plays the quiet one")
				s.Require().Equal(got, built.Out)
				s.Require().NotEmpty(built.Plan, "compiled from a plan, not from the rig")

				// And the plan it wrote carries the gain.
				f, err := os.Open(built.Plan)
				s.Require().NoError(err)

				defer func() { _ = f.Close() }()

				made, err := plan.Load(f)
				s.Require().NoError(err)

				var fields map[string]any
				s.Require().NoError(
					json.Unmarshal((*made.Device.Routing)[outputSlot], &fields))
				s.Require().InDelta(-30, fields[gainKey], 0.001)
			},
		},
		{
			// output entry, writing the result, compiling it. The chain stays on the
			// measuring loop in all four, which is why none of them is a warning.
			name: "quieter refuses what it cannot take off the loop",
			then: func() {
				read := errors.New("not a preset")
				built := errors.New("the catalog does not carry that")

				for _, tt := range []struct {
					name  string
					pedal func(*mocks.MockTuner)
					// work answers the scratch directory the rewrite is written to, so a
					// row can hand over one nothing may write into.
					work func() string
					is   error
					says string
				}{
					{
						name: "the preset will not read back",
						pedal: func(p *mocks.MockTuner) {
							p.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
								Return(sdk.Reading{}, read)
						},
						is: read,
					},
					{
						name: "the preset has no output to turn down",
						pedal: func(p *mocks.MockTuner) {
							p.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
								Return(sdk.Reading{Plan: plan.Plan{}}, nil)
						},
						is: ErrNoOutput,
					},
					{
						name: "there is nowhere to write the rewritten preset",
						pedal: func(p *mocks.MockTuner) {
							p.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
								Return(sdk.Reading{Plan: routed()}, nil)
						},
						work: func() string {
							held := s.T().TempDir()
							s.Require().NoError(os.Chmod(held, 0o500))
							s.T().Cleanup(func() {
								s.Require().NoError(os.Chmod(held, 0o700))
							})

							return held
						},
						says: "matt-freeman.headroom.yaml",
					},
					{
						name: "the rewritten preset will not compile",
						pedal: func(p *mocks.MockTuner) {
							p.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
								Return(sdk.Reading{Plan: routed()}, nil)
							p.EXPECT().Compile(gomock.Any(), gomock.Any()).
								Return(sdk.Built{}, built)
						},
						is: built,
					},
				} {
					s.Run(tt.name, func() {
						pedal := mocks.NewMockTuner(gomock.NewController(s.T()))
						tt.pedal(pedal)

						work := s.T().TempDir()
						if tt.work != nil {
							work = tt.work()
						}

						_, err := quieter(context.Background(),
							pedal, "already.hlx", work, "matt-freeman", -30)

						if tt.is != nil {
							s.Require().ErrorIs(err, tt.is)

							return
						}

						s.Require().ErrorContains(err, tt.says)
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestItReadsThePresetBackBecauseARigCarriesNoRouting is the reason for the
// round trip.
//

// TestQuieterRefusesWhatItCannotTakeOffTheLoop covers every way the rewrite
// fails.
//

// TestItWarnsWhenItCannotPutTheOutputBack covers the silent wrong artifact.
//
// The plan is still returned, because every dial the solve found is in it and
// throwing a five-minute run away over one routing entry is the worse failure.
// But it goes back carrying the measuring rig's own output, so it is silent at
// the quarter-inch socket, and a person not told that finds out by plugging in.
func (s *HeadroomPublicTestSuite) TestItWarnsWhenItCannotPutTheOutputBack() {
	measuring := routed()

	for _, tt := range []struct {
		name string
		give sdk.Reading
		err  error
		says string
	}{
		{
			name: "the compiled preset will not read back",
			err:  errors.New("gone"),
			says: "could not be read back",
		},
		{
			name: "the compiled preset carries no routing",
			give: sdk.Reading{Plan: plan.Plan{}},
			says: "carries no routing",
		},
		{
			name: "the compiled preset carries no output entry",
			give: sdk.Reading{Plan: plan.Plan{Device: &rig.DeviceState{
				Routing: &map[string]json.RawMessage{
					"dsp0.inputA": json.RawMessage(`{}`),
				},
			}}},
			says: "carries no " + outputSlot,
		},
	} {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			pedal := mocks.NewMockTuner(ctrl)

			pedal.EXPECT().
				PresetFile(gomock.Any(), "as-built.hlx").
				Return(tt.give, tt.err)

			var buf bytes.Buffer

			got, err := asCompiled(
				context.Background(), &buf, pedal, measuring, "as-built.hlx")
			s.Require().NoError(err)

			said := buf.String()
			s.Require().Contains(said, "[warn]")
			s.Require().Contains(said, tt.says)
			s.Require().Contains(said, "quarter-inch")
			s.Require().Contains(said, "dials the solve found are unaffected",
				"it says what is still good, or somebody throws the plan away")

			// The plan comes back, measuring entry and all.
			var fields map[string]any
			s.Require().NoError(
				json.Unmarshal((*got.Device.Routing)[outputSlot], &fields))
			s.Require().InDelta(1, fields[sendKey], 0.001)
		})
	}
}
