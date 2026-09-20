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
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/cli/internal/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
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

	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	s.cat = cat
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
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	held := map[string]any{}
	for _, name := range order {
		held[name] = 0.5
	}

	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, at sdk.Address, v float32) error {
			if at.Param >= len(order) {
				return errors.New("error -3")
			}

			held[order[at.Param]] = float64(v)

			return nil
		}).AnyTimes()

	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, sdk.Format) (sdk.Reading, error) {
			now := map[string]any{}
			for k, v := range held {
				now[k] = v
			}

			return sdk.Reading{Rig: rig.Spec{
				Schema: rig.SchemaName, ID: "probed",
				Subject:    rig.Subject{Kind: rig.KindSound, Name: "one block"},
				Instrument: rig.InstrumentBass,
				Chain: []rig.ChainEntry{{
					Role: rig.RoleAmp, Gear: "an amplifier", Params: &now,
				}},
			}}, nil
		}).AnyTimes()
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

// TestNamesAgreeWithTheCatalog covers hardware that matches the claim.
func (s *NamesTestSuite) TestNamesAgreeWithTheCatalog() {
	model := "HD2_AmpUSDripmanNorm"
	s.device(WireOrder(s.cat, model))

	said, err := s.run(model)

	s.Require().NoError(err)
	s.Require().Contains(said, "agree on every index")
	s.Require().Contains(said, "Norm Drive")
}

// TestNamesSayWhenTheyDisagree is the whole reason this exists.
//
// Every curve is filed under a parameter's position, and counting down a
// sorted printout instead of the wire order mislabels all of them while the
// numbers stay entirely plausible.
func (s *NamesTestSuite) TestNamesSayWhenTheyDisagree() {
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
}

// TestASwitchIsNotADisagreement is the regression this type exists for.
//
// A switch refuses a number on a dial, which says nothing about whether the
// catalog has its name in the right place. Counted as a mismatch it reported
// every curve for the US Dripman as filed under the wrong control, while all
// eleven indexes that could actually be tested had matched. Somebody reading
// that would have thrown away a correct catalog, or re-run an hour of sweeps
// to fix nothing.
func (s *NamesTestSuite) TestASwitchIsNotADisagreement() {
	model := "HD2_AmpUSDripmanNorm"
	order := WireOrder(s.cat, model)

	// Bright, which the device will not take a float for.
	s.refuse(order, "Bright")

	said, err := s.run(model)

	s.Require().NoError(err)
	s.Require().Contains(said, "refused a number on a dial")
	s.Require().Contains(said, "agree on every index this could")
	s.Require().Contains(said, "1 could not be tested")
	s.Require().NotContains(said, "they disagree")
}

// refuse is a device that takes every index but one, which is a switch.
func (s *NamesTestSuite) refuse(
	order []string,
	switched string,
) {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	held := map[string]any{}
	for _, name := range order {
		held[name] = 0.5
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

			held[order[at.Param]] = float64(v)

			return nil
		}).AnyTimes()

	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, sdk.Format) (sdk.Reading, error) {
			now := map[string]any{}
			for k, v := range held {
				now[k] = v
			}

			return sdk.Reading{Rig: rig.Spec{
				Schema: rig.SchemaName, ID: "probed",
				Subject:    rig.Subject{Kind: rig.KindSound, Name: "one block"},
				Instrument: rig.InstrumentBass,
				Chain: []rig.ChainEntry{{
					Role: rig.RoleAmp, Gear: "an amplifier", Params: &now,
				}},
			}}, nil
		}).AnyTimes()
}

// TestNamesReportAnIndexThatReachesNothing covers a parameter past the list.
func (s *NamesTestSuite) TestNamesReportAnIndexThatReachesNothing() {
	model := "HD2_AmpUSDripmanNorm"
	s.device([]string{"Norm Drive"})

	said, err := s.run(model)

	s.Require().NoError(err)
	s.Require().Contains(said, "refused a number on a dial")
}

// TestNamesRefuseAModelNobodyHas covers a name that is not a block.
func (s *NamesTestSuite) TestNamesRefuseAModelNobodyHas() {
	_, err := s.run("nothing calls itself this")

	s.Require().ErrorContains(err, "the catalog has no")
}

// TestNamesReportAPresetItCannotBuild covers the compiler failing.
func (s *NamesTestSuite) TestNamesReportAPresetItCannotBuild() {
	s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, errors.New("will not build")).AnyTimes()

	_, err := s.run("HD2_AmpUSDripmanNorm")

	s.Require().Error(err)
}

// TestNamesReportADeviceThatWillNotAnswer covers a read failing.
func (s *NamesTestSuite) TestNamesReportADeviceThatWillNotAnswer() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{}, errors.New("usb: gone")).AnyTimes()

	_, err := s.run("HD2_AmpUSDripmanNorm")

	s.Require().ErrorContains(err, "usb: gone")
}

// TestNamesReportAChainWithNothingInIt covers a device playing an empty
// preset.
//
// It is the state a broken encoder left every written preset in for an
// evening, and the four structural slots still answer, so it reads as a
// device that is fine.
func (s *NamesTestSuite) TestNamesReportAChainWithNothingInIt() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Rig: rig.Spec{}}, nil).AnyTimes()

	_, err := s.run("HD2_AmpUSDripmanNorm")

	s.Require().ErrorContains(err, "is not one of them")
}

// TestNamesReportAPedalThatWillNotLoadThePreset covers Play failing.
func (s *NamesTestSuite) TestNamesReportAPedalThatWillNotLoadThePreset() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
		Return(errors.New("the device said no")).AnyTimes()

	_, err := s.run("HD2_AmpUSDripmanNorm")

	s.Require().ErrorContains(err, "the device said no")
}

func TestNamesTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NamesTestSuite))
}
