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
	"os"
	"path/filepath"
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

// ControlsRunTestSuite covers sweeping one block's controls, without the
// block or the hardware.
type ControlsRunTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockPedal
	out   string
	dry   string
}

func (s *ControlsRunTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockPedal(s.ctrl)
	s.out = filepath.Join(s.T().TempDir(), "curves.json")
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di.wav")
}

// ready makes the pedal answer everything a sweep asks of it.
func (s *ControlsRunTestSuite) ready() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		Return(cabReading(nil), nil).AnyTimes()
}

// cabReading is a device read of a chain holding the cabinet alone.
//
// Both layers, because a read populates both: the rig is what travels with the
// curves, and the probe reads its parameters out of the plan.
func cabReading(
	held plan.Params,
) sdk.Reading {
	return sdk.Reading{
		Rig: rig.Spec{
			Schema: rig.SchemaName, ID: "measured",
			Instrument: rig.InstrumentBass,
			Chain:      []rig.ChainEntry{{Role: rig.RoleCab, Gear: "2x15 Brute"}},
		},
		Plan: plan.Plan{
			Name: "measured",
			Blocks: []plan.Block{{
				Model: "HD2_CabMicIr_2x15Brute", Params: held, Enabled: true,
			}},
		},
	}
}

// eqReading is a device read of a chain holding the equaliser alone.
func eqReading(
	held plan.Params,
) sdk.Reading {
	return sdk.Reading{
		Rig: rig.Spec{
			Schema: rig.SchemaName, ID: "probed",
			Instrument: rig.InstrumentBass,
			Chain:      []rig.ChainEntry{{Role: rig.RoleEQ, Gear: "Simple EQ"}},
		},
		Plan: plan.Plan{
			Name: "probed",
			Blocks: []plan.Block{{
				Model: "HD2_EQSimple3Band", Params: held, Enabled: true,
			}},
		},
	}
}

// moves is a device that reveals a wire order by answering the probe.
//
// Each index moves the parameter at that position, which is what discovering an
// order reads back. Self-contained rather than layered on ready(), because
// gomock takes the first matching expectation and ready()'s Turn does nothing,
// so a probe behind it would read every index as reaching no parameter.
func (s *ControlsRunTestSuite) moves(
	order []string,
) {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	held := plan.Params{}
	for _, name := range order {
		held[name] = catalog.Float(0.5)
	}

	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, at sdk.Address, v float32) error {
			if at.Param < len(order) {
				held[order[at.Param]] = catalog.Float(float64(v))
			}

			return nil
		}).AnyTimes()

	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, sdk.Format) (sdk.Reading, error) {
			now := plan.Params{}
			maps.Copy(now, held)

			return eqReading(now), nil
		}).AnyTimes()
}

// run sweeps a model and returns what landed on disk.
func (s *ControlsRunTestSuite) run(
	model string,
	b sdk.Bench,
) (measured.Curves, string, error) {
	var buf bytes.Buffer

	err := MeasureControls(context.Background(), &buf, ControlsOptions{
		Client: s.pedal, Model: model, Dry: s.dry, Out: s.out,
		Seconds: 1, Points: 3, Takes: 2, Bench: b,
	})
	if err != nil {
		return measured.Curves{}, buf.String(), err
	}

	f, err := os.Open(s.out)
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	// A run where nothing carried signal writes a document naming no
	// controls, which LoadCurves refuses on purpose. That is the answer here
	// rather than a failure, so it comes back empty.
	got, err := measured.LoadCurves(f)
	if err != nil {
		return measured.Curves{}, buf.String(), nil
	}

	return got, buf.String(), nil
}

// TestControlsAsksTheDeviceWhenTheCatalogClaimsNoOrder covers the equalisers.
//
// Line 6 ship no symbol list for one, so the catalog names no wire order and a
// sweep used to refuse outright. That left the blocks whose fingerprints say
// nothing as the blocks nothing described at all: an equaliser measures as the
// baseline at its defaults, because flat is what it ships at, so its curves are
// the only thing that says what it does.
//
// Discovered rather than handed over on a flag. An order somebody types is an
// order somebody can get wrong, and a curve filed under the wrong control is a
// plausible number about a different knob.
func (s *ControlsRunTestSuite) TestControlsAsksTheDeviceWhenTheCatalogClaimsNoOrder() {
	s.moves([]string{"LowGain", "MidFreq", "MidGain", "HighGain", "Level"})

	got, said, err := s.run("HD2_EQSimple3Band", bench{})

	s.Require().NoError(err)
	s.Require().Contains(said, "claims no wire order")
	s.Require().True(got.Probed,
		"the file says the order came from the device, not the catalog")
	s.Require().Contains(got.Controls, "MidGain",
		"a control the catalog never named was found and swept")
}

// TestControlsSweepsDialsAndLists covers a cabinet, which has both.
//
// Its Mic is twelve microphones rather than a dial, and every one of them is
// measured: nothing sits between two microphones, so asking for three evenly
// spaced positions across twelve would measure some twice and miss others.
func (s *ControlsRunTestSuite) TestControlsSweepsDialsAndLists() {
	s.ready()

	got, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})
	s.Require().NoError(err)

	s.Require().True(got.Isolated)
	s.Require().Equal(alone, got.Slot)
	s.Require().NotEmpty(got.Chain, "the rig it ran through travels with it")
	s.Require().NotEmpty(got.Reference.SHA256)

	mic := got.Controls["Mic"]
	s.Require().Equal("int", mic.Kind)
	s.Require().Len(mic.Points, 12, "every microphone, not three of them")
	s.Require().NotEmpty(mic.Spread)
	s.Require().Empty(mic.Fits, "a list of settings has no slope")

	dial := got.Controls["Distance"]
	s.Require().Equal("float", dial.Kind)
	s.Require().Len(dial.Points, 3)
	s.Require().NotEmpty(dial.Fits)
	s.Require().Empty(dial.Spread)
	s.Require().InDelta(1, dial.Span.Low, 0.001, "inches, from the catalog")
	s.Require().InDelta(12, dial.Span.High, 0.001)
	s.Require().True(dial.Span.FromCatalog)
}

// TestControlsRecordsWhatWentSilent covers a control that mutes the chain.
//
// Kept rather than dropped, because where a control mutes a chain is worth
// knowing, and left out of the arithmetic, because two takes of silence agree
// to the last digit and the centroid of hiss reads high.
func (s *ControlsRunTestSuite) TestControlsRecordsWhatWentSilent() {
	s.ready()

	var calls int

	// Only the first control, because each later one measures its own floor
	// from the chain as it then sits, and a bench that is quiet throughout
	// makes quiet the settled level.
	got, said, err := s.run("HD2_CabMicIr_2x15Brute",
		turning{after: 2, to: bench{quiet: true}, calls: &calls})

	s.Require().NoError(err)
	s.Require().Contains(said, "silent")
	s.Require().Contains(said, "no curve here",
		"a control with nothing to measure is said rather than invented")
	s.Require().NotContains(got.Controls, "Mic",
		"a control with one position that carried signal has no curve")
}

// TestControlsRecordsWhatClipped is the other end of the same problem.
//
// One microphone of a cabinet's twelve clips and read 4,471 Hz where the
// other eleven sat between 126 and 147. Left in, it is filed as the
// microphone choice moving the centre of gravity by 4,345 Hz, and the figure
// is the converters' ceiling.
func (s *ControlsRunTestSuite) TestControlsRecordsWhatClipped() {
	s.ready()

	var calls int

	got, said, err := s.run("HD2_CabMicIr_2x15Brute",
		turning{after: 2, to: bench{clipped: true}, calls: &calls})

	s.Require().NoError(err)
	s.Require().Contains(said, "clipped")
	s.Require().NotContains(got.Controls, "Mic",
		"every microphone hit the ceiling, so none of them is a reading")
}

// TestControlsRefusesWhenTheDeviceNamesNothingEither covers the last resort.
//
// An equaliser has no symbol entry, so the catalog cannot say which index is
// which and the device is asked instead. When the device answers nothing at any
// index there is no third place to look, and a sweep that went ahead would file
// every curve under a guess. That is the one case still refused.
//
// The refusal used to cover every unclaimed block, which left the equalisers
// undescribed: one measures as the baseline at its defaults, because flat is
// what it ships at, so its curves are the only thing that says what it does.
func (s *ControlsRunTestSuite) TestControlsRefusesWhenTheDeviceNamesNothingEither() {
	s.ready()

	_, _, err := s.run("HD2_EQSimple3Band", bench{})

	s.Require().ErrorContains(err, "named no parameter")
}

// TestControlsRefusesAModelNobodyHas covers a name that is not a block.
func (s *ControlsRunTestSuite) TestControlsRefusesAModelNobodyHas() {
	_, _, err := s.run("nothing calls itself this", bench{})

	s.Require().ErrorContains(err, "the catalog has no")
}

// TestControlsReportsAReferenceItCannotRead covers a missing signal.
func (s *ControlsRunTestSuite) TestControlsReportsAReferenceItCannotRead() {
	var buf bytes.Buffer

	err := MeasureControls(context.Background(), &buf, ControlsOptions{
		Client: s.pedal, Model: "HD2_CabMicIr_2x15Brute",
		Dry: "nowhere.wav", Out: s.out, Seconds: 1, Points: 3, Takes: 2,
		Bench: bench{},
	})

	s.Require().ErrorContains(err, "nowhere.wav")
}

// TestControlsReportsAChainItCannotBuild covers the compiler failing.
func (s *ControlsRunTestSuite) TestControlsReportsAChainItCannotBuild() {
	s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, errors.New("will not build")).AnyTimes()

	_, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

	s.Require().ErrorContains(err, "building a chain holding only")
}

// TestControlsCarriesOnPastAControlTheDeviceRefuses covers a declined move.
func (s *ControlsRunTestSuite) TestControlsCarriesOnPastAControlTheDeviceRefuses() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
		Return(cabReading(nil), nil).AnyTimes()
	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("error -3")).AnyTimes()
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("error -3")).AnyTimes()

	_, said, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

	s.Require().NoError(err)
	s.Require().Contains(said, "refused")
}

func TestControlsRunTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ControlsRunTestSuite))
}
