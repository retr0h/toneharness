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

package compile_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

type LiftPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *LiftPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// preset builds a document holding one block of the given model.
func (s *LiftPublicTestSuite) preset(
	name string,
	models ...catalog.ModelID,
) *preset.Document {
	blocks := make([]plan.Block, 0, len(models))
	for i, m := range models {
		blocks = append(blocks, plan.Block{Model: m, Pos: i, Enabled: true})
	}

	doc, err := preset.New(s.cat.DeviceID, plan.Plan{Name: name, Blocks: blocks})
	s.Require().NoError(err)

	return doc
}

// catalogOf builds a catalog holding nothing but the given blocks.
func (s *LiftPublicTestSuite) catalogOf(
	blocks map[catalog.ModelID]catalog.Block,
) *catalog.Catalog {
	if blocks == nil {
		return s.cat
	}

	return &catalog.Catalog{
		Device: "Test", DeviceID: s.cat.DeviceID, Blocks: blocks,
	}
}

// rigOf returns a valid rig naming one piece of gear.
func rigOf(
	gear string,
	inst rig.Instrument,
) rig.Spec {
	return rig.Spec{
		Instrument: inst,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: gear},
		},
	}
}

// TestLift covers Lift, which reads a preset into the rig it describes and
// the plan that realises it.
//
// One method and one table, so a case is a row rather than a file.
func (s *LiftPublicTestSuite) TestLift() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Reads a preset as a rig.
			name: "lift",
			then: func() {
				tests := []struct {
					name string
					// what the preset is called, which is where a rig's identifier comes
					// from.
					title  string
					model  catalog.ModelID
					blocks map[catalog.ModelID]catalog.Block
					// a document written by hand, for what preset.New cannot build.
					raw string
					// a preset holding no blocks at all.
					bare bool

					wantGear       string
					wantRole       rig.Role
					wantInstrument rig.Instrument
					wantID         string
					err            error
					errText        string
				}{
					{
						name:           "an amp emulating real gear, named the way a person would",
						model:          "HD2_AmpSVBeastNrm",
						wantGear:       "Ampeg SVT® (normal channel)",
						wantInstrument: rig.InstrumentBass,
					},
					{
						name:           "a Line 6 original, by its own name, since it emulates nothing",
						model:          "HD2_AmpLine6Litigator",
						wantGear:       "Line 6 Litigator",
						wantInstrument: rig.InstrumentGuitar,
					},
					{
						// A rig has to say what every block is, and "something this
						// device carries and we do not recognise" is a truthful answer.
						name:           "a model the catalog has never heard of, by identifier",
						model:          "HD2_NotInThisCatalog",
						wantGear:       "HD2_NotInThisCatalog",
						wantRole:       rig.RoleOther,
						wantInstrument: rig.InstrumentGuitar,
					},
					{
						name:           "a preset with no amp in it",
						model:          "HD2_DistMinotaur",
						wantInstrument: rig.InstrumentGuitar,
					},
					{
						// A handful of catalog entries carry an empty name and no gear,
						// so neither handle is available and the identifier is all there
						// is.
						name:  "a model with no name",
						model: "HD2_Nameless",
						blocks: map[catalog.ModelID]catalog.Block{
							"HD2_Nameless": {ID: "HD2_Nameless", Category: catalog.CategoryDrive},
						},
						wantGear: "HD2_Nameless",
					},
					{
						name:  "a category this project does not know",
						model: "HD2_Odd",
						blocks: map[catalog.ModelID]catalog.Block{
							"HD2_Odd": {
								ID: "HD2_Odd", Name: "Odd",
								Category: catalog.Category("nonsense"),
							},
						},
						wantRole: rig.RoleOther,
					},
					{
						name:   "a name that is already an identifier",
						title:  "Mike Dirnt",
						model:  "HD2_AmpSVBeastNrm",
						wantID: "mike-dirnt",
					},
					{
						name:   "a name carrying punctuation",
						title:  "CT-Blackend",
						model:  "HD2_AmpSVBeastNrm",
						wantID: "ct-blackend",
					},
					{
						name:   "a name somebody spaced out",
						title:  "  Lots   of   Space  ",
						model:  "HD2_AmpSVBeastNrm",
						wantID: "lots-of-space",
					},
					{
						name:   "a name of nothing but punctuation",
						title:  "!!!",
						model:  "HD2_AmpSVBeastNrm",
						wantID: "untitled",
					},
					{
						name:   "no name at all",
						model:  "HD2_AmpSVBeastNrm",
						wantID: "untitled",
					},
					{name: "a preset holding no blocks", bare: true, err: rig.ErrInvalid},
					{
						name: "a document it cannot read",
						raw: `{"schema":"L6Preset","version":6,"data":{"device":2162694,` +
							`"meta":{"name":"Bad"},"tone":{"dspX":{"block0":{"@model":"x"}}}}}`,
						errText: "reading the chain",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						var doc *preset.Document

						switch {
						case tt.raw != "":
							var err error

							doc, err = preset.Read(bytes.NewReader([]byte(tt.raw)))
							s.Require().NoError(err)
						case tt.bare:
							var err error

							doc, err = preset.New(s.cat.DeviceID, plan.Plan{Name: "Empty"})
							s.Require().NoError(err)
						default:
							doc = s.preset(tt.title, tt.model)
						}

						gotID, got, _, err := compile.Lift(doc, s.catalogOf(tt.blocks))

						if tt.err != nil || tt.errText != "" {
							s.Require().Error(err)

							if tt.err != nil {
								s.Require().ErrorIs(err, tt.err)
							}

							if tt.errText != "" {
								s.Require().Contains(err.Error(), tt.errText)
							}

							return
						}

						s.Require().NoError(err)
						s.Require().NoError(rig.Validate(got), "a lifted rig must validate")

						if tt.wantGear != "" {
							s.Require().Equal(tt.wantGear, got.Chain[0].Gear)
						}

						if tt.wantRole != "" {
							s.Require().Equal(tt.wantRole, got.Chain[0].Role)
						}

						if tt.wantInstrument != "" {
							s.Require().Equal(tt.wantInstrument, got.Instrument)
						}

						if tt.wantID != "" {
							s.Require().Equal(tt.wantID, gotID)
						}
					})
				}
			},
		},
		{
			// Writes a plan into a preset.
			name: "lower",
			then: func() {
				tests := []struct {
					name string
					made plan.Plan
					// a plan lifted off a preset of this model, rather than one written
					// out here.
					from catalog.ModelID

					wantModel catalog.ModelID
					// -1 asserts every knob is set, a positive number asserts how many.
					exact   int
					types   map[string]catalog.ParamType
					ints    map[string]int64
					errText string
				}{
					{
						name: "a plan stating parameters, which are the whole truth",
						made: ampPlan("exact", plan.Params{
							"Drive": catalog.Float(0.8), "MidFreq": catalog.Int(2),
							"Bright": catalog.Bool(true), "Voicing": catalog.Enum("Modern"),
						}),
						exact: 4,
						// A switch stays a switch: a device given 1.5 for a
						// three-position control refuses the preset rather than rounding.
						types: map[string]catalog.ParamType{
							"MidFreq": catalog.ParamInt,
							"Bright":  catalog.ParamBool,
							"Voicing": catalog.ParamEnum,
						},
					},
					{
						// Written as stated. Nudging by a half and truncating cuts
						// toward zero, so this arrived as -11: an octave down turned
						// into a major seventh, in a preset nobody would think to check.
						name:  "a parameter somebody set below nothing",
						made:  ampPlan("octave", plan.Params{"MidFreq": catalog.Int(-12)}),
						types: map[string]catalog.ParamType{"MidFreq": catalog.ParamInt},
						ints:  map[string]int64{"MidFreq": -12},
					},
					{
						// "Ampeg SVT" matches both channels. The recorded identifier is
						// what makes a lifted plan rebuild into the preset it came from.
						name:      "the exact model, over the name it shares",
						from:      "HD2_AmpSVBeastBrt",
						wantModel: "HD2_AmpSVBeastBrt",
					},
					{
						// Lowering asks the same question about what a plan claims beside
						// its chain, so a colour this device cannot light fails here
						// rather than reaching a preset.
						name:    "a colour the device does not have",
						made:    withSwitch(ampPlan("lit", nil), "chartruse"),
						errText: "footswitches[0].led",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						doc := s.preset("Test", "HD2_AmpSVBeastNrm")

						made := tt.made

						if tt.from != "" {
							var err error

							_, _, made, err = compile.Lift(s.preset("Test", tt.from), s.cat)
							s.Require().NoError(err)
						}

						err := compile.Lower(doc, made, s.cat)

						if tt.errText != "" {
							s.Require().Error(err)
							s.Require().Contains(err.Error(), tt.errText)

							return
						}

						s.Require().NoError(err)

						c, err := doc.Spec()
						s.Require().NoError(err)

						if tt.wantModel != "" {
							s.Require().Equal(tt.wantModel, c.Blocks[0].Model)
						}

						switch {
						case tt.exact < 0:
							s.Require().NotEmpty(c.Blocks[0].Params, "every knob is set")
						case tt.exact > 0:
							s.Require().Len(c.Blocks[0].Params, tt.exact)
						}

						for key, want := range tt.types {
							s.Require().Equal(want, c.Blocks[0].Params[key].Type())
						}

						for key, want := range tt.ints {
							got, ok := c.Blocks[0].Params[key].Int()
							s.Require().True(ok)
							s.Require().Equal(want, got, "%s", key)
						}
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

// withSwitch puts one footswitch on a plan, lit the given colour.
func withSwitch(
	made plan.Plan,
	led string,
) plan.Plan {
	made.Footswitches = []rig.Footswitch{{Led: &led}}

	return made
}

// substituted says what to put in place of the gear a rig names.
func substituted(
	spec rig.Spec,
	instead string,
) rig.Spec {
	spec.Chain[0].Substitute = &rig.Substitute{Gear: instead}

	return spec
}

// ampPlan is a plan holding the bass amplifier, set as stated.
func ampPlan(
	name string,
	params plan.Params,
) plan.Plan {
	return plan.Plan{
		Name: name,
		Blocks: []plan.Block{
			{Model: "HD2_AmpSVBeastNrm", Params: params, Enabled: true},
		},
	}
}

// TestRealise fits a rig to the device a catalog describes.
func (s *LiftPublicTestSuite) TestRealise() {
	tests := []struct {
		name string
		// id is what the plan is asked to name, which is the document's rather
		// than the gear's: a rig is gear, and what it is called is said once at
		// the top of the document that holds it.
		id     string
		spec   rig.Spec
		blocks map[catalog.ModelID]catalog.Block

		wantModel  catalog.ModelID
		wantParams []string
		absent     []string
		// -1 asserts every knob is set, a positive number asserts how many.
		exact   int
		err     error
		errText string
	}{
		{name: "a rig that is not one", err: rig.ErrInvalid},
		{
			name:    "gear nothing on this device models",
			spec:    rigOf("Nonesuch 900", rig.InstrumentGuitar),
			errText: "emulates \"Nonesuch 900\"",
		},
		{
			// The rig names what was really played and says what this device
			// should put there, so fitting it lands on the stand-in.
			name: "gear nothing models, with a stand-in the rig names",
			spec: substituted(
				rigOf("Nonesuch 900", rig.InstrumentBass),
				"Ampeg SVT (normal"),
			wantModel: "HD2_AmpSVBeastNrm",
			exact:     -1,
		},
		{
			name: "a stand-in nothing models either",
			spec: substituted(
				rigOf("Nonesuch 900", rig.InstrumentBass),
				"Also Nonesuch"),
			errText: `"Also Nonesuch" stands in for "Nonesuch 900"`,
		},
		{
			// A rig describes gear rather than a block, so every knob gets
			// Line 6's own default, which is never invalid.
			name:  "a rig naming gear and nothing else",
			spec:  rigOf("Ampeg SVT (normal", rig.InstrumentBass),
			exact: -1,
		},
		{
			// A parameter Line 6 state no default for has no kind, and
			// writing a value with no kind produces a preset the device
			// rejects.
			name: "a parameter with no stated default",
			spec: rigOf("Half A Thing", rig.InstrumentGuitar),
			blocks: map[catalog.ModelID]catalog.Block{
				"HD2_Half": {
					ID: "HD2_Half", Name: "Half", BasedOn: "Half A Thing",
					Category: catalog.CategoryAmp,
					Params: map[string]catalog.Param{
						"Drive":   {Type: catalog.ParamFloat, Default: catalog.Float(0.5)},
						"Missing": {Type: catalog.ParamFloat},
					},
				},
			},
			wantParams: []string{"Drive"},
			absent:     []string{"Missing"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			id := tt.id
			if id == "" {
				id = "a-rig"
			}

			made, err := compile.Realise(id, tt.spec, s.catalogOf(tt.blocks))

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(id, made.Rig, "a plan names the rig it realises")

			if tt.wantModel != "" {
				s.Require().Equal(tt.wantModel, made.Blocks[0].Model)
			}

			switch {
			case tt.exact < 0:
				s.Require().NotEmpty(made.Blocks[0].Params, "every knob is set")
			case tt.exact > 0:
				s.Require().Len(made.Blocks[0].Params, tt.exact)
			}

			for _, want := range tt.wantParams {
				s.Require().Contains(made.Blocks[0].Params, want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(made.Blocks[0].Params, unwanted)
			}
		})
	}
}

// TestRealisePicksTheSameModelEveryTime is a property of the resolver rather
// than a case of the call.
//
// Fitting used to range a map and take the first name that matched, so one
// rig became a different preset each run: six compiles of this one named an
// amplifier, a preamp, the bright channel and twice a cabinet. A plan lifted
// off a device was unaffected, because it carries the model identifier, which
// is why nothing caught it.
func (s *LiftPublicTestSuite) TestRealisePicksTheSameModelEveryTime() {
	spec := rigOf("Ampeg SVT", rig.InstrumentBass)

	var first catalog.ModelID

	for range 20 {
		made, err := compile.Realise("a-rig", spec, s.cat)
		s.Require().NoError(err)
		s.Require().NotEmpty(made.Blocks)

		if first == "" {
			first = made.Blocks[0].Model
		}

		s.Require().Equal(first, made.Blocks[0].Model, "the same rig, a different model")
	}

	// And an amplifier, because the role is half the question.
	b, ok := s.cat.Block(first)
	s.Require().True(ok)
	s.Require().Equal(catalog.CategoryAmp, b.Category)
}

func TestLiftPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LiftPublicTestSuite))
}
