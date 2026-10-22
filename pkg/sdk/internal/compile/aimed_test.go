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

package compile

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// AimedTestSuite covers setting controls from a sweep toward a target.
//
// What task 268 was about. All 32 artist rigs named the amplifier a source
// puts somebody on and then carried the catalog's defaults on every control:
// `dee-dee-ramone` sat at Bass 0.41, Bias 0.74, ChVol 0.8 and Drive 0.6, which
// is the factory setting to the digit, while his three records sat on the
// document unread.
type AimedTestSuite struct {
	suite.Suite
}

// aimedAmp is a bass amplifier with a float, an integer and a boolean control,
// which is the spread the swept drives hold: 163 floats, 20 integers and 16
// booleans.
func (s *AimedTestSuite) aimedAmp() catalog.Block {
	return catalog.Block{
		ID: "HD2_AmpTestBass", Category: catalog.CategoryAmp,
		Subcategory: "Bass",
		Params: map[string]catalog.Param{
			"Drive":   {Type: catalog.ParamFloat, Min: 0, Max: 1},
			"MidFreq": {Type: catalog.ParamInt, Min: 0, Max: 4},
			"Bright":  {Type: catalog.ParamBool},
		},
	}
}

// aimedCurves is a sweep of that amplifier, plus a control it does not carry.
//
// Every curve runs the mid band from 10 to 40 against a noise floor of 0.1, so
// each one moves something and the nearest position is unambiguous. `Settled`
// is where every point sits, so the level window lets all of them through.
func (s *AimedTestSuite) aimedCurves() measured.Curves {
	at := func(kind string, low, high float64) measured.Curve {
		return measured.Curve{
			Kind:    kind,
			Settled: -20,
			Noise:   map[audio.Figure]float64{audio.KeyMid: 0.1},
			Points: []measured.Point{
				{Value: low, Figures: measured.Figures{Mid: 10, Level: -20}},
				{Value: high, Figures: measured.Figures{Mid: 40, Level: -20}},
			},
		}
	}

	return measured.Curves{Controls: map[string]measured.Curve{
		"Drive":   at("float", 0.25, 0.828986013986014),
		"MidFreq": at("int", 0, 3),
		"Bright":  at("bool", 0, 1),
		// A control the sweep read and this model does not carry, which is a
		// reading about a block this is not.
		"Elsewhere": at("float", 0, 1),
	}}
}

// TestAimed covers which blocks are aimed and which are left alone.
func (s *AimedTestSuite) TestAimed() {
	// 40 in the mid band, which is the high end of every curve, so a control
	// that is aimed lands on its upper position.
	high := measured.Figures{Mid: 40}

	cab := catalog.Block{
		ID: "HD2_CabTest", Category: catalog.CategoryCab,
		Params: map[string]catalog.Param{
			"LowCut": {Type: catalog.ParamFloat, Min: 20, Max: 500},
		},
	}

	tests := []struct {
		name   string
		blocks []catalog.Block
		built  plan.Plan
		intent Intent
		// wants is what each control must read afterwards, by block index.
		wants []plan.Params
	}{
		{
			// The ordinary answer for 11 of the 43 players nobody has
			// measured: the rig keeps the controls its sources justify.
			name:   "a rig nobody has records for is left alone",
			blocks: []catalog.Block{s.aimedAmp()},
			built: plan.Plan{Blocks: []plan.Block{{
				Model:  "HD2_AmpTestBass",
				Params: plan.Params{"Drive": catalog.Float(0.6)},
			}}},
			intent: Intent{Curves: s.sweptAs(s.aimedCurves())},
			wants:  []plan.Params{{"Drive": catalog.Float(0.6)}},
		},
		{
			// A release, where only the fingerprints are packed into the
			// binary and no curves ship beside them.
			name:   "no sweeps offered leaves every control where it was",
			blocks: []catalog.Block{s.aimedAmp()},
			built: plan.Plan{Blocks: []plan.Block{{
				Model:  "HD2_AmpTestBass",
				Params: plan.Params{"Drive": catalog.Float(0.6)},
			}}},
			intent: Intent{Aim: &high},
			wants:  []plan.Params{{"Drive": catalog.Float(0.6)}},
		},
		{
			// 436 of this device's 614 blocks, which the campaign never
			// reached.
			name:   "a block the campaign never swept keeps its default",
			blocks: []catalog.Block{s.aimedAmp()},
			built: plan.Plan{Blocks: []plan.Block{{
				Model:  "HD2_AmpTestBass",
				Params: plan.Params{"Drive": catalog.Float(0.6)},
			}}},
			intent: Intent{
				Aim: &high,
				Curves: func(catalog.ModelID) (measured.Curves, bool) {
					return measured.Curves{}, false
				},
			},
			wants: []plan.Params{{"Drive": catalog.Float(0.6)}},
		},
		{
			// Task 255's exploit. Letting a sweep set a whole cabinet sent
			// LowCut to 79.9Hz, above the fundamental of a bass's open E.
			name:   "the cabinet is not aimed",
			blocks: []catalog.Block{cab},
			built: plan.Plan{Blocks: []plan.Block{{
				Model:  "HD2_CabTest",
				Params: plan.Params{"LowCut": catalog.Float(37)},
			}}},
			intent: Intent{Aim: &high, Curves: func(
				catalog.ModelID,
			) (measured.Curves, bool) {
				return measured.Curves{Controls: map[string]measured.Curve{
					"LowCut": {
						Kind:    "float",
						Settled: -20,
						Noise:   map[audio.Figure]float64{audio.KeyMid: 0.1},
						Points: []measured.Point{
							{Value: 37, Figures: measured.Figures{
								Mid: 10, Level: -20,
							}},
							{Value: 79.9, Figures: measured.Figures{
								Mid: 40, Level: -20,
							}},
						},
					},
				}}, true
			}},
			wants: []plan.Params{{"LowCut": catalog.Float(37)}},
		},
		{
			// The whole point. Every control the model carries and the sweep
			// read moves to the position nearest those records, in its own
			// kind, and the one the model does not carry is not written.
			name:   "a swept block is set to what the records measure as",
			blocks: []catalog.Block{s.aimedAmp()},
			built: plan.Plan{Blocks: []plan.Block{{
				Model: "HD2_AmpTestBass",
				Params: plan.Params{
					"Drive":   catalog.Float(0.6),
					"MidFreq": catalog.Int(2),
					"Bright":  catalog.Bool(false),
				},
			}}},
			intent: Intent{Aim: &high, Curves: s.sweptAs(s.aimedCurves())},
			wants: []plan.Params{{
				// Rounded to a thousandth of the control's travel: the digits
				// past that are interpolation between two swept positions
				// rather than anything measured.
				"Drive":   catalog.Float(0.829),
				"MidFreq": catalog.Int(3),
				"Bright":  catalog.Bool(true),
			}},
		},
		{
			// A chain shorter than the blocks it was built from, which is not
			// a state a resolve reaches and is a panic if it ever does.
			name:   "more blocks than the plan holds stops at the plan",
			blocks: []catalog.Block{s.aimedAmp(), s.aimedAmp()},
			built: plan.Plan{Blocks: []plan.Block{{
				Model:  "HD2_AmpTestBass",
				Params: plan.Params{"Drive": catalog.Float(0.6)},
			}}},
			intent: Intent{Aim: &high, Curves: s.sweptAs(s.aimedCurves())},
			wants:  []plan.Params{{"Drive": catalog.Float(0.829)}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			aimed(tt.intent, tt.blocks, tt.built, "bass")

			for i, want := range tt.wants {
				for name, at := range want {
					s.Require().Equal(at, tt.built.Blocks[i].Params[name],
						"block %d's %s", i, name)
				}
			}
		})
	}
}

// TestSet covers writing one swept position in its control's own kind.
//
// Separately from aimed, because the kinds are what a document refuses: the
// Kinky Boost's `Bright` is a boolean and a preset carrying 1.0 on it will not
// load, which is how 4 of the 11 genre rigs stopped compiling.
func (s *AimedTestSuite) TestSet() {
	blk := s.aimedAmp()

	tests := []struct {
		name    string
		control string
		at      measured.Setting
		// want is what the control must read, and nil means not written.
		want *catalog.ParamValue
	}{
		{
			name:    "a float is written as a float, rounded",
			control: "Drive",
			at:      measured.Setting{Value: 0.828986013986014, Kind: "float"},
			want:    s.value(catalog.Float(0.829)),
		},
		{
			name:    "an integer is written as an integer",
			control: "MidFreq",
			at:      measured.Setting{Value: 3, Kind: "int"},
			want:    s.value(catalog.Int(3)),
		},
		{
			name:    "a boolean is written as a boolean",
			control: "Bright",
			at:      measured.Setting{Value: 1, Kind: "bool"},
			want:    s.value(catalog.Bool(true)),
		},
		{
			name:    "a boolean at zero is false rather than absent",
			control: "Bright",
			at:      measured.Setting{Value: 0, Kind: "bool"},
			want:    s.value(catalog.Bool(false)),
		},
		{
			// A kind no sweep records, which would otherwise be dropped
			// silently. A number is the thing a control almost always is.
			name:    "a kind nothing recorded is written as a number",
			control: "Drive",
			at:      measured.Setting{Value: 0.5},
			want:    s.value(catalog.Float(0.5)),
		},
		{
			name:    "a control the model does not carry is not written",
			control: "Elsewhere",
			at:      measured.Setting{Value: 1, Kind: "float"},
			want:    nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			params := plan.Params{}

			set(params, blk, tt.control, tt.at)

			if tt.want == nil {
				s.Require().NotContains(params, tt.control)

				return
			}

			s.Require().Equal(*tt.want, params[tt.control])
		})
	}
}

// sweptAs answers with the same curves for every model asked about.
func (s *AimedTestSuite) sweptAs(
	curves measured.Curves,
) func(catalog.ModelID) (measured.Curves, bool) {
	return func(catalog.ModelID) (measured.Curves, bool) {
		return curves, true
	}
}

// value is a control reading as something a table may compare against nil.
func (s *AimedTestSuite) value(
	at catalog.ParamValue,
) *catalog.ParamValue {
	return &at
}

func TestAimedTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(AimedTestSuite))
}
