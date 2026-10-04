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
	"errors"
	"maps"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// NamesTestSuite covers holding the catalog's parameter order to the device.
type NamesTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockPedal
	cat   *catalog.Catalog
}

func (s *NamesTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockPedal(s.ctrl)
	// Every measuring command asks the client which models the attached
	// device has, rather than reading the built-in HX Stomp catalog, so that
	// --catalog and --device reach them.
	s.pedal.EXPECT().Catalog(gomock.Any()).
		DoAndReturn(func(context.Context) (*catalog.Catalog, error) {
			return catalog.BuiltIn()
		}).AnyTimes()

	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	s.cat = cat

	// The preset read back and rebuilt: every command here takes its chain off
	// the measuring loop before playing it, sending the output to USB rather
	// than to the socket the measuring lead comes from. What that rewrite does
	// is offTheLoop's own test, so it is answered here rather than asserted.
	s.pedal.EXPECT().
		PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
}

// device answers as a pedal whose wire order is the one given.
//
// Each probe plays the preset, reads the parameters, moves one index and
// reads them again. What moves is whatever sits at that index in order.
func (s *NamesTestSuite) device(
	order []string,
) {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			writeBlank(s.T(), in.Out)

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	held := plan.Params{}
	for _, name := range order {
		held[name] = catalog.Float(0.5)
	}

	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, at sdk.Address, v float32) error {
			if at.Param >= len(order) {
				return errors.New("error -3")
			}

			held[order[at.Param]] = catalog.Float(float64(v))

			return nil
		}).AnyTimes()

	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, sdk.Format) (sdk.Reading, error) {
			now := plan.Params{}
			maps.Copy(now, held)

			return probedReading(now), nil
		}).AnyTimes()
}

// probedReading is a device read of a chain holding one block.
//
// Both layers, because a read populates both. A probe reads only the plan's
// parameters, so the model names the block a plan must carry and nothing
// compares it.
func probedReading(
	held plan.Params,
) sdk.Reading {
	return sdk.Reading{
		ID: "probed",
		Rig: rig.Spec{
			Instrument: rig.InstrumentBass,
			Chain:      []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "an amplifier"}},
		},
		Plan: plan.Plan{
			Name: "probed",
			Blocks: []plan.Block{{
				Model: "HD2_AmpUSDripmanNorm", Params: held, Enabled: true,
			}},
		},
	}
}

// run checks one model and returns what was said.
func (s *NamesTestSuite) run(
	model string,
) (string, error) {
	var buf bytes.Buffer

	err := MeasureNames(context.Background(), &buf, NamesOptions{
		Client: s.pedal, Model: model,
	})

	return buf.String(), err
}

// TestWireOrder covers the order the device writes a block's parameters in.
//
// One method and one table, so a case is a row rather than a file.
func (s *NamesTestSuite) TestWireOrder() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Hardware that matches the claim.
			name: "names agree with the catalog",
			then: func() {
				model := "HD2_AmpUSDripmanNorm"
				s.device(WireOrder(s.cat, model))

				said, err := s.run(model)

				s.Require().NoError(err)
				s.Require().Contains(said, "agree on every index")
				s.Require().Contains(said, "Norm Drive")
			},
		},
		{
			// The whole reason this exists.
			//
			// Every curve is filed under a parameter's position, and counting down a
			// sorted printout instead of the wire order mislabels all of them while the
			// numbers stay entirely plausible.
			name: "names say when they disagree",
			then: func() {
				model := "HD2_AmpUSDripmanNorm"

				// The order `catalog show` prints, which is sorted for a reader and is
				// not what the device sends.
				s.device([]string{
					"Bass", "Bias", "BiasX", "Bright", "ChVol", "Hum",
					"Master", "Mid", "Norm Drive", "Ripple", "Sag", "Treble",
				})

				said, err := s.run(model)

				s.Require().NoError(err)
				s.Require().Contains(said, "they disagree")
				s.Require().Contains(said, "filed under the wrong control")
			},
		},
		{
			// The regression this type exists for.
			//
			// A switch refuses a number on a dial, which says nothing about whether the
			// catalog has its name in the right place. Counted as a mismatch it reported
			// every curve for the US Dripman as filed under the wrong control, while all
			// eleven indexes that could actually be tested had matched. Somebody reading
			// that would have thrown away a correct catalog, or re-run an hour of sweeps
			// to fix nothing.
			name: "a switch is not a disagreement",
			then: func() {
				model := "HD2_AmpUSDripmanNorm"
				order := WireOrder(s.cat, model)

				// Bright, which the device will not take a float for.
				s.refuse(order, "Bright")

				said, err := s.run(model)

				s.Require().NoError(err)
				s.Require().Contains(said, "refused a number on a dial")
				s.Require().Contains(said, "claims and this could test")
				s.Require().Contains(said, "0 unclaimed and 1 untestable")
				s.Require().NotContains(said, "they disagree")
			},
		},
		{
			// The second regression of this
			// shape, and the one that mattered for the equalisers.
			//
			// Line 6 ship no symbol list for an equaliser, so the catalog claims no order
			// for one at all. Comparing an absent claim against the parameter that moved
			// reported that they disagree and that every curve was misfiled, for a block
			// the catalog had never made a claim about. The device is the only thing that
			// knows the order, and discovering it is the whole point of running this.
			name: "an unclaimed order is not a disagreement",
			then: func() {
				model := "HD2_EQSimple3Band"

				// A device that answers, against a catalog carrying no symbols: what the
				// device moved is the only record there is.
				s.device([]string{"LowGain", "MidFreq", "MidGain", "HighGain", "Level"})

				said, err := s.run(model)

				s.Require().NoError(err)
				s.Require().Contains(said, "claims no order for this block")
				s.Require().Contains(said, "LowGain")
				s.Require().NotContains(said, "they disagree")
			},
		},
		{
			// A parameter past the list.
			name: "names report an index that reaches nothing",
			then: func() {
				model := "HD2_AmpUSDripmanNorm"
				s.device([]string{"Norm Drive"})

				said, err := s.run(model)

				s.Require().NoError(err)
				s.Require().Contains(said, "refused a number on a dial")
			},
		},
		{
			// A name that is not a block.
			name: "names refuse a model nobody has",
			then: func() {
				_, err := s.run("nothing calls itself this")

				s.Require().ErrorContains(err, "the catalog has no")
			},
		},
		{
			// The compiler failing.
			name: "names report a preset it cannot build",
			then: func() {
				s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
					Return(sdk.Built{}, errors.New("will not build")).AnyTimes()

				_, err := s.run("HD2_AmpUSDripmanNorm")

				s.Require().Error(err)
			},
		},
		{
			// A read failing.
			name: "names report a device that will not answer",
			then: func() {
				s.pedal.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
						writeBlank(s.T(), in.Out)

						return sdk.Built{}, nil
					}).AnyTimes()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
					Return(sdk.Reading{}, errors.New("usb: gone")).AnyTimes()

				_, err := s.run("HD2_AmpUSDripmanNorm")

				s.Require().ErrorContains(err, "usb: gone")
			},
		},
		{
			// A device playing an empty
			// preset.
			//
			// It is the state a broken encoder left every written preset in for an
			// evening, and the four structural slots still answer, so it reads as a
			// device that is fine.
			name: "names report a chain with nothing in it",
			then: func() {
				s.pedal.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
						writeBlank(s.T(), in.Out)

						return sdk.Built{}, nil
					}).AnyTimes()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
					Return(sdk.Reading{Plan: plan.Plan{}}, nil).AnyTimes()

				_, err := s.run("HD2_AmpUSDripmanNorm")

				s.Require().ErrorContains(err, "is not one of them")
			},
		},
		{
			// Play failing.
			name: "names report a pedal that will not load the preset",
			then: func() {
				s.pedal.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
						writeBlank(s.T(), in.Out)

						return sdk.Built{}, nil
					}).AnyTimes()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(errors.New("the device said no")).AnyTimes()

				_, err := s.run("HD2_AmpUSDripmanNorm")

				s.Require().ErrorContains(err, "the device said no")
			},
		},
		{
			// --device naming a pedal this
			// binary ships no catalog for.
			//
			// First, because the model a probe is asked about is looked up in it and a
			// binary that cannot say which models exist has nothing to probe.
			name: "names reports a catalog it cannot read",
			then: func() {
				ctrl := gomock.NewController(s.T())
				pedal := mocks.NewMockPedal(ctrl)
				pedal.EXPECT().Catalog(gomock.Any()).
					Return(nil, errors.New("no catalog for that pedal")).AnyTimes()

				err := MeasureNames(context.Background(), buffer(), NamesOptions{
					Client: pedal, Model: "HD2_CabMicIr_2x15Brute",
				})

				s.Require().ErrorContains(err, "no catalog for that pedal")
			},
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			tt.then()
		})
	}
}

// refuse is a device that takes every index but one, which is a switch.
func (s *NamesTestSuite) refuse(
	order []string,
	switched string,
) {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			writeBlank(s.T(), in.Out)

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	held := plan.Params{}
	for _, name := range order {
		held[name] = catalog.Float(0.5)
	}

	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, at sdk.Address, v float32) error {
			if at.Param >= len(order) {
				return errors.New("error -3")
			}

			if order[at.Param] == switched {
				return errors.New("error -3")
			}

			held[order[at.Param]] = catalog.Float(float64(v))

			return nil
		}).AnyTimes()

	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, sdk.Format) (sdk.Reading, error) {
			now := plan.Params{}
			maps.Copy(now, held)

			return probedReading(now), nil
		}).AnyTimes()
}

func TestNamesTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NamesTestSuite))
}
