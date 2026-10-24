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
	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// LevelTestSuite covers making a build as loud as another rig.
//
// Found by playing two rigs built from the same corpus: one read as quiet, and
// the cause was one block. Across the 133 cabinets measured here the readings
// run 11.3dB apart, and the quieter of two presets loses an A/B almost
// regardless of its tone.
type LevelTestSuite struct {
	suite.Suite
}

// cat is an amplifier with a channel volume and two cabinets 2.9dB apart,
// which is the gap that was actually found.
func (s *LevelTestSuite) cat() *catalog.Catalog {
	return &catalog.Catalog{
		Device: "HX Stomp",
		Blocks: map[catalog.ModelID]catalog.Block{
			"CabUnmeasured": {ID: "CabUnmeasured", Category: catalog.CategoryCab},
			"AmpTest": {
				ID: "AmpTest", Category: catalog.CategoryAmp,
				Params: map[string]catalog.Param{
					"ChVol": {Key: "ChVol", Type: catalog.ParamFloat, Max: 1},
					"Drive": {Key: "Drive", Type: catalog.ParamFloat, Max: 1},
				},
			},
			"AmpNoLevel": {
				ID: "AmpNoLevel", Category: catalog.CategoryAmp,
				Params: map[string]catalog.Param{
					"Drive": {Key: "Drive", Type: catalog.ParamFloat, Max: 1},
				},
			},
			"CabQuiet": {ID: "CabQuiet", Category: catalog.CategoryCab},
			"CabLoud":  {ID: "CabLoud", Category: catalog.CategoryCab},
		},
	}
}

// lib is what the block library reads for those two cabinets.
func (s *LevelTestSuite) lib() measured.Library {
	return measured.Library{Blocks: map[string]measured.Block{
		"CabQuiet": {
			ID: "CabQuiet", Category: catalog.CategoryCab,
			Figures: measured.Figures{Level: -59.62},
		},
		"CabLoud": {
			ID: "CabLoud", Category: catalog.CategoryCab,
			Figures: measured.Figures{Level: -56.76},
		},
		"CabClipped": {
			ID: "CabClipped", Category: catalog.CategoryCab,
			Clipped: true, Figures: measured.Figures{Level: -1},
		},
	}}
}

// curves is a ChVol swept across its range, 6dB a quarter turn.
func (s *LevelTestSuite) curves() map[catalog.ModelID]measured.Curves {
	return map[catalog.ModelID]measured.Curves{
		"AmpTest": {Block: "AmpTest", Controls: map[string]measured.Curve{
			"ChVol": {
				Control: "ChVol", Settled: -20, SilentBelow: -50,
				Noise: map[audio.Figure]float64{audio.KeyLevel: 0.05},
				Points: []measured.Point{
					{Value: 0.50, Figures: measured.Figures{Level: -30}},
					{Value: 0.75, Figures: measured.Figures{Level: -24}},
					{Value: 1.00, Figures: measured.Figures{Level: -18}},
				},
			},
		}},
	}
}

// chain is a quiet cabinet behind an amplifier with its ChVol half up.
func (s *LevelTestSuite) chain() plan.Plan {
	return plan.Plan{Blocks: []plan.Block{
		{Model: "AmpTest", Pos: 0, Enabled: true, Params: plan.Params{
			"ChVol": catalog.Float(0.5), "Drive": catalog.Float(0.5),
		}},
		{Model: "CabQuiet", Pos: 1, Enabled: true},
	}}
}

// TestLevel covers what it moves, and every way it moves nothing.
func (s *LevelTestSuite) TestLevel() {
	for _, tt := range []struct {
		name   string
		built  plan.Plan
		want   float64
		curves map[catalog.ModelID]measured.Curves

		param   string
		to      float64
		matches float64
		because string
	}{
		{
			// Six decibels up is a quarter turn on this curve, which is a
			// position the sweep actually measured rather than one a slope
			// extrapolated to.
			name:    "a chain asked to come up",
			built:   s.chain(),
			want:    -53.62,
			curves:  s.curves(),
			param:   "ChVol",
			to:      0.75,
			matches: -53.62,
		},
		{
			// And down, which is the same arithmetic with the sign flipped.
			name: "a chain asked to come down",
			built: plan.Plan{Blocks: []plan.Block{
				{Model: "AmpTest", Pos: 0, Enabled: true, Params: plan.Params{
					"ChVol": catalog.Float(1), "Drive": catalog.Float(0.5),
				}},
				{Model: "CabQuiet", Pos: 1, Enabled: true},
			}},
			want:    -65.62,
			curves:  s.curves(),
			param:   "ChVol",
			to:      0.75,
			matches: -65.62,
		},
		{
			// Below what anybody reliably hears on a broadband signal. A build
			// reporting a control moved a third of a decibel has spent
			// somebody's attention on nothing.
			name:    "a chain already where it was asked to be",
			built:   s.chain(),
			want:    -59.2,
			curves:  s.curves(),
			because: "its cabinet is already within 1dB of that one's",
		},
		{
			// The common answer on a release: the curves are not packed into
			// the binary.
			name:    "an amplifier nobody has swept",
			built:   s.chain(),
			want:    -53.62,
			because: "AmpTest has no committed sweep",
		},
		{
			// Every figure here was measured through a cabinet, so a chain
			// without one is a chain nothing has a reading for. A guess would
			// be worse than an answer of no.
			name: "a chain with no cabinet",
			built: plan.Plan{Blocks: []plan.Block{
				{Model: "AmpTest", Pos: 0, Enabled: true, Params: plan.Params{
					"ChVol": catalog.Float(0.5),
				}},
			}},
			want:    -53.62,
			curves:  s.curves(),
			because: "nothing in the chain has a measured level",
		},
		{
			// A clipped reading is a reading of the converters, so it is not a
			// level to aim from.
			name: "a cabinet whose reading clipped",
			built: plan.Plan{Blocks: []plan.Block{
				{Model: "AmpTest", Pos: 0, Enabled: true, Params: plan.Params{
					"ChVol": catalog.Float(0.5),
				}},
				{Model: "CabClipped", Pos: 1, Enabled: true},
			}},
			want:    -53.62,
			curves:  s.curves(),
			because: "nothing in the chain has a measured level",
		},
		{
			// A cabinet the library has never read. Nothing is the right
			// answer: a level aimed from a figure nobody took is a level
			// nobody can check.
			name: "a cabinet nobody has measured",
			built: plan.Plan{Blocks: []plan.Block{
				{Model: "AmpTest", Pos: 0, Enabled: true, Params: plan.Params{
					"ChVol": catalog.Float(0.5),
				}},
				{Model: "CabUnmeasured", Pos: 1, Enabled: true},
			}},
			want:    -53.62,
			curves:  s.curves(),
			because: "nothing in the chain has a measured level",
		},
		{
			// Drive is not a level control here and must not become one. The
			// sweep says turning it down is marginally louder on one
			// amplifier, so anybody reasoning from the name gets the sign
			// wrong.
			name: "an amplifier with no level control",
			built: plan.Plan{Blocks: []plan.Block{
				{Model: "AmpNoLevel", Pos: 0, Enabled: true, Params: plan.Params{
					"Drive": catalog.Float(0.5),
				}},
				{Model: "CabQuiet", Pos: 1, Enabled: true},
			}},
			want:    -53.62,
			curves:  s.curves(),
			because: "no amplifier in the chain has a level control",
		},
		{
			// A position at the converters' ceiling reads the converters
			// rather than the amplifier, so it is not a position to land on.
			// The one below it answers instead.
			name:  "a curve whose top position is at the ceiling",
			built: s.chain(),
			want:  -47.62,
			curves: map[catalog.ModelID]measured.Curves{
				"AmpTest": {Controls: map[string]measured.Curve{
					"ChVol": func() measured.Curve {
						c := s.curves()["AmpTest"].Controls["ChVol"]
						c.Points[2].Level = -1

						return c
					}(),
				}},
			},
			param:   "ChVol",
			to:      0.75,
			matches: -53.62,
		},
		{
			// And one below the curve's own floor reads hiss. The curve says
			// where that is, so there is no threshold to pick here.
			name:  "a curve whose top position is below its floor",
			built: s.chain(),
			want:  -47.62,
			curves: map[catalog.ModelID]measured.Curves{
				"AmpTest": {Controls: map[string]measured.Curve{
					"ChVol": func() measured.Curve {
						c := s.curves()["AmpTest"].Controls["ChVol"]
						c.Points[2].Level = -60

						return c
					}(),
				}},
			},
			param:   "ChVol",
			to:      0.75,
			matches: -53.62,
		},
		{
			// The sweep flagged it, which the measured ceiling is a floor
			// under rather than a replacement for.
			name:  "a position the sweep flagged as clipped",
			built: s.chain(),
			want:  -47.62,
			curves: map[catalog.ModelID]measured.Curves{
				"AmpTest": {Controls: map[string]measured.Curve{
					"ChVol": func() measured.Curve {
						c := s.curves()["AmpTest"].Controls["ChVol"]
						c.ClippedAt = []float64{1}

						return c
					}(),
				}},
			},
			param:   "ChVol",
			to:      0.75,
			matches: -53.62,
		},
		{
			name:  "a position the sweep flagged as muted",
			built: s.chain(),
			want:  -47.62,
			curves: map[catalog.ModelID]measured.Curves{
				"AmpTest": {Controls: map[string]measured.Curve{
					"ChVol": func() measured.Curve {
						c := s.curves()["AmpTest"].Controls["ChVol"]
						c.MutedAt = []float64{1}

						return c
					}(),
				}},
			},
			param:   "ChVol",
			to:      0.75,
			matches: -53.62,
		},
		{
			// Every position unusable, so the curve says nothing about where
			// the control sits and nothing can be read off it.
			name:  "a curve with no usable position at all",
			built: s.chain(),
			want:  -47.62,
			curves: map[catalog.ModelID]measured.Curves{
				"AmpTest": {Controls: map[string]measured.Curve{
					"ChVol": func() measured.Curve {
						c := s.curves()["AmpTest"].Controls["ChVol"]
						for i := range c.Points {
							c.Points[i].Level = -1
						}

						return c
					}(),
				}},
			},
			because: "does not reach",
		},
		{
			// One position, so there is nowhere to move to that lands any
			// nearer than staying put. A control turned for the report's
			// benefit is not a move.
			name:  "a curve with only the position it already sits at",
			built: s.chain(),
			want:  -47.62,
			curves: map[catalog.ModelID]measured.Curves{
				"AmpTest": {Controls: map[string]measured.Curve{
					"ChVol": {
						Control: "ChVol", Settled: -20, SilentBelow: -50,
						Points: []measured.Point{
							{Value: 0.5, Figures: measured.Figures{Level: -30}},
						},
					},
				}},
			},
			because: "does not reach",
		},
		{
			// Asked for more than the control holds. Nothing moves rather than
			// the control going to its stop and the report claiming a level it
			// did not reach.
			name:    "a level further than the control reaches",
			built:   s.chain(),
			want:    -20,
			curves:  s.curves(),
			param:   "ChVol",
			to:      1,
			matches: -47.62,
		},
	} {
		s.Run(tt.name, func() {
			got, at := Level(tt.built, s.cat(), tt.want, s.lib(), tt.curves)

			s.Require().InDelta(tt.want, at.Wanted, 1e-9)

			if tt.because != "" {
				s.Require().Contains(at.Because, tt.because)
				s.Require().Empty(at.Param)

				return
			}

			s.Require().Empty(at.Because)
			s.Require().Equal(tt.param, at.Param)

			held, _ := got.Blocks[0].Params[tt.param].Float()
			s.Require().InDelta(tt.to, held, 1e-9)
			s.Require().InDelta(tt.to, at.To, 1e-9)
			s.Require().InDelta(tt.matches, at.Matches, 1e-6)
		})
	}
}

func TestLevelTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LevelTestSuite))
}

// TestCompilerLevel covers the method the Compiler value carries.
//
// The type exists so a caller can say what it depends on and stand something
// else in its place, which a package-level function does not allow, so every
// method has to actually reach its function.
func (s *LevelTestSuite) TestCompilerLevel() {
	got, at := New().Level(s.chain(), s.cat(), -53.62, s.lib(), s.curves())

	s.Require().Equal("ChVol", at.Param)

	held, _ := got.Blocks[0].Params["ChVol"].Float()
	s.Require().InDelta(0.75, held, 1e-9)
}

// TestAsked covers whether a Setup stated a level at all.
func (s *LevelTestSuite) TestAsked() {
	for _, tt := range []struct {
		name string
		in   result.Levelled
		want bool
	}{
		{
			name: "a level that was asked for",
			in:   result.Levelled{Wanted: -56.76, Was: -59.62},
			want: true,
		},
		{
			// A build that reported where it comes out without being given a
			// target is still a build that has something to say.
			name: "a reading with no target",
			in:   result.Levelled{Was: -59.62},
			want: true,
		},
		{
			// A Setup that said nothing leaves both at zero, and a target of
			// zero would be a chain asked for full scale, which nobody asks
			// for.
			name: "nothing asked and nothing read",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Asked())
		})
	}
}

// swept is an amplifier whose channel volume has been measured, with its
// fingerprint taken at the catalog's default of 0.5.
//
// Two positions either side of that default and a decibel apiece, so a
// correction is readable rather than inferred.
func (s *LevelTestSuite) swept() (catalog.Catalog, measured.Curves) {
	cat := catalog.Catalog{
		Device: "HX Stomp",
		Blocks: map[catalog.ModelID]catalog.Block{
			"AmpSwept": {
				ID: "AmpSwept", Category: catalog.CategoryAmp,
				Params: map[string]catalog.Param{
					"ChVol": {
						Key: "ChVol", Type: catalog.ParamFloat,
						Min: 0, Max: 1, Default: catalog.Float(0.5),
					},
				},
			},
			"AmpNoDefault": {
				ID: "AmpNoDefault", Category: catalog.CategoryAmp,
				Params: map[string]catalog.Param{
					"ChVol": {Key: "ChVol", Type: catalog.ParamFloat, Min: 0, Max: 1},
				},
			},
			"CabAny": {ID: "CabAny", Category: catalog.CategoryCab},
		},
	}

	return cat, measured.Curves{
		Controls: map[string]measured.Curve{"ChVol": {
			Control: "ChVol", Kind: "float", Settled: -20, SilentBelow: -50,
			Points: []measured.Point{
				{Value: 0, Figures: measured.Figures{Level: -26}},
				{Value: 0.5, Figures: measured.Figures{Level: -20}},
				{Value: 1, Figures: measured.Figures{Level: -14}},
			},
		}},
	}
}

// TestTurnedFrom covers how far an amplifier's level controls sit from where
// its fingerprint was taken.
//
// One method and one table, so a case is a row rather than a file.
func (s *LevelTestSuite) TestTurnedFrom() {
	cat, curve := s.swept()

	for _, tt := range []struct {
		name  string
		model catalog.ModelID
		at    plan.Params
		swept measured.Curves
		want  float64
	}{
		{
			// The case this was written for. An Ampeg SVT-4 Pro reached a
			// build with Ch Vol at 0.75 against a default of 1.0, and nothing
			// knew it was quieter than its own fingerprint said.
			name:  "below the default is quieter than the fingerprint",
			model: "AmpSwept",
			at:    plan.Params{"ChVol": catalog.Float(0)},
			swept: curve,
			want:  -6,
		},
		{
			name:  "above the default is louder",
			model: "AmpSwept",
			at:    plan.Params{"ChVol": catalog.Float(1)},
			swept: curve,
			want:  6,
		},
		{
			name:  "at the default is no correction at all",
			model: "AmpSwept",
			at:    plan.Params{"ChVol": catalog.Float(0.5)},
			swept: curve,
			want:  0,
		},
		{
			// A control the preset says nothing about is wherever the
			// catalog put it, which is where the fingerprint was taken.
			name:  "a control the plan does not state",
			model: "AmpSwept",
			at:    plan.Params{},
			swept: curve,
			want:  0,
		},
		{
			// Nothing to compare against, so nothing is claimed. The
			// cabinets then carry the whole comparison, which is what this
			// did before the amplifier was a term at all.
			name:  "an amplifier with no sweep",
			model: "AmpSwept",
			at:    plan.Params{"ChVol": catalog.Float(0)},
			swept: measured.Curves{},
			want:  0,
		},
		{
			name:  "a control whose default the catalog does not state",
			model: "AmpNoDefault",
			at:    plan.Params{"ChVol": catalog.Float(0)},
			swept: curve,
			want:  0,
		},
		{
			// A sweep whose every position is under its own silence floor,
			// which is a control that was measured through a chain carrying
			// nothing. Neither position can be read, so no correction is
			// claimed rather than one being guessed at.
			name:  "a sweep no position of which can be read",
			model: "AmpSwept",
			at:    plan.Params{"ChVol": catalog.Float(0)},
			swept: measured.Curves{Controls: map[string]measured.Curve{"ChVol": {
				Control: "ChVol", Settled: -20, SilentBelow: -10,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{Level: -90}},
					{Value: 1, Figures: measured.Figures{Level: -88}},
				},
			}}},
			want: 0,
		},
	} {
		s.Run(tt.name, func() {
			one := cat.Blocks[tt.model]
			got := turnedFrom(one, plan.Block{Model: tt.model, Params: tt.at}, tt.swept)

			s.Require().InDelta(tt.want, got, 1e-9)
		})
	}
}

// TestAmplifierLevel covers what the amplifier contributes to the comparison.
//
// One method and one table, so a case is a row rather than a file.
func (s *LevelTestSuite) TestAmplifierLevel() {
	cat, curve := s.swept()

	for _, tt := range []struct {
		name   string
		blocks []plan.Block
		curves map[catalog.ModelID]measured.Curves
		want   float64
	}{
		{
			name: "the amplifier's own controls are the term",
			blocks: []plan.Block{
				{Model: "CabAny"},
				{Model: "AmpSwept", Params: plan.Params{"ChVol": catalog.Float(0)}},
			},
			curves: map[catalog.ModelID]measured.Curves{"AmpSwept": curve},
			want:   -6,
		},
		{
			// A chain with no amplifier leaves the cabinets to carry it,
			// which two chains both missing this term still share.
			name:   "a chain with no amplifier contributes nothing",
			blocks: []plan.Block{{Model: "CabAny"}},
			curves: map[catalog.ModelID]measured.Curves{"AmpSwept": curve},
			want:   0,
		},
		{
			name: "a model the catalog does not carry",
			blocks: []plan.Block{
				{Model: "NothingCallsItselfThis", Params: plan.Params{}},
			},
			curves: map[catalog.ModelID]measured.Curves{},
			want:   0,
		},
	} {
		s.Run(tt.name, func() {
			got := amplifierLevel(
				plan.Plan{Blocks: tt.blocks}, &cat, tt.curves)

			s.Require().InDelta(tt.want, got, 1e-9)
		})
	}
}
