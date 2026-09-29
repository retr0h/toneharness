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
	"encoding/json"
	"errors"
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

// HeadroomPublicTestSuite covers lowering the one gain that is not the tone.
type HeadroomPublicTestSuite struct {
	suite.Suite
}

// routed is a plan carrying the output entry a preset arrives with.
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

// TestItSetsTheOutputGainAndLeavesEverythingElse is the whole job.
//
// The output block sits after the chain, so turning it down lowers what
// reaches the quarter-inch socket the measuring lead comes from without
// touching the tone. The amplifier's ChVol and Master are the tone, and
// lowering those would have the solver solve for a different sound.
func (s *HeadroomPublicTestSuite) TestItSetsTheOutputGainAndLeavesEverythingElse() {
	got, err := withGain(routed(), -30)
	s.Require().NoError(err)

	var fields map[string]any
	s.Require().NoError(
		json.Unmarshal((*got.Device.Routing)[outputSlot], &fields))

	s.Require().InDelta(-30, fields[gainKey], 0.001)
	s.Require().InDelta(0.5, fields["pan"], 0.001, "the rest is left alone")
	s.Require().Equal("HelixStomp_AppDSPFlowOutputMain", fields["@model"])
	s.Require().Contains(*got.Device.Routing, "dsp0.outputB",
		"the other entries travel")
}

// TestThePlanHandedOverIsNotChanged covers the copy.
//
// A caller that keeps the plan it built still holds what it built, because a
// plan is written out and compared and a routing map shared between two of
// them makes the comparison meaningless.
func (s *HeadroomPublicTestSuite) TestThePlanHandedOverIsNotChanged() {
	was := routed()

	_, err := withGain(was, -30)
	s.Require().NoError(err)

	var fields map[string]any
	s.Require().NoError(
		json.Unmarshal((*was.Device.Routing)[outputSlot], &fields))

	s.Require().InDelta(0, fields[gainKey], 0.001)
}

// TestAPlanWithNoOutputIsRefused covers having nothing to turn down.
//
// Said rather than passed through. A plan compiled from a rig carries no
// routing at all, because the inputs and outputs come from the blank template
// while the preset is written, and returning the loud one silently is how a
// guard comes to pass on a chain nobody protected.
func (s *HeadroomPublicTestSuite) TestAPlanWithNoOutputIsRefused() {
	_, err := withGain(plan.Plan{}, -30)
	s.Require().ErrorIs(err, ErrNoOutput)

	routing := map[string]json.RawMessage{"dsp0.inputA": json.RawMessage(`{}`)}

	_, err = withGain(
		plan.Plan{Device: &rig.DeviceState{Routing: &routing}}, -30)
	s.Require().ErrorIs(err, ErrNoOutput)
}

// TestAnOutputEntryThatWillNotDecode covers a plan somebody broke.
func (s *HeadroomPublicTestSuite) TestAnOutputEntryThatWillNotDecode() {
	routing := map[string]json.RawMessage{outputSlot: json.RawMessage(`{`)}

	_, err := withGain(
		plan.Plan{Device: &rig.DeviceState{Routing: &routing}}, -30)

	s.Require().ErrorContains(err, outputSlot)
}

func TestHeadroomPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HeadroomPublicTestSuite))
}

// TestNoHeadroomLeavesThePresetAlone covers asking for nothing.
func (s *HeadroomPublicTestSuite) TestNoHeadroomLeavesThePresetAlone() {
	ctrl := gomock.NewController(s.T())
	pedal := mocks.NewMockTuner(ctrl)

	// No PresetFile and no Compile expected: asking for no headroom must not
	// read or write anything.
	got, err := quieter(context.Background(),
		TuneOptions{Client: pedal, ID: "matt-freeman"}, "already.hlx", 0)

	s.Require().NoError(err)
	s.Require().Equal("already.hlx", got)
}

// TestItReadsThePresetBackBecauseARigCarriesNoRouting is the reason for the
// round trip.
//
// A plan compiled from a rig has no routing at all: the inputs and outputs
// come from the blank template while the preset is written. The preset has
// them, so reading it back is how the output entry arrives complete, with its
// model and its output already set.
func (s *HeadroomPublicTestSuite) TestItReadsThePresetBackBecauseARigCarriesNoRouting() {
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
		TuneOptions{Client: pedal, ID: "matt-freeman"}, "already.hlx", -30)

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
}

// TestAPresetThatWillNotReadBack covers the read failing.
func (s *HeadroomPublicTestSuite) TestAPresetThatWillNotReadBack() {
	ctrl := gomock.NewController(s.T())
	pedal := mocks.NewMockTuner(ctrl)

	wanted := errors.New("not a preset")

	pedal.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{}, wanted)

	_, err := quieter(context.Background(),
		TuneOptions{Client: pedal, ID: "matt-freeman"}, "already.hlx", -30)

	s.Require().ErrorIs(err, wanted)
}

// TestAPresetWithNoOutputToTurnDown covers a chain nobody can protect.
func (s *HeadroomPublicTestSuite) TestAPresetWithNoOutputToTurnDown() {
	ctrl := gomock.NewController(s.T())
	pedal := mocks.NewMockTuner(ctrl)

	pedal.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: plan.Plan{}}, nil)

	_, err := quieter(context.Background(),
		TuneOptions{Client: pedal, ID: "matt-freeman"}, "already.hlx", -30)

	s.Require().ErrorIs(err, ErrNoOutput)
}

// TestTheQuietPresetThatWillNotCompile covers the write failing.
func (s *HeadroomPublicTestSuite) TestTheQuietPresetThatWillNotCompile() {
	ctrl := gomock.NewController(s.T())
	pedal := mocks.NewMockTuner(ctrl)

	wanted := errors.New("the catalog does not carry that")

	pedal.EXPECT().PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routed()}, nil)
	pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, wanted)

	_, err := quieter(context.Background(),
		TuneOptions{Client: pedal, ID: "matt-freeman"}, "already.hlx", -30)

	s.Require().ErrorIs(err, wanted)
}
