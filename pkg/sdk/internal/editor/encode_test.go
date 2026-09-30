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

package editor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// EncodeTestSuite covers turning a preset back into what a device lays out.
type EncodeTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *EncodeTestSuite) SetupSuite() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	s.cat = cat
}

// capture returns one slot as an HX Stomp actually sent it.
func (s *EncodeTestSuite) capture(
	name string,
) []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "wire", "testdata", name))
	s.Require().NoError(err)

	return raw
}

// asPreset rebuilds what the reading commands produce for a device's answer.
func (s *EncodeTestSuite) asPreset(
	name string,
	got wire.DevicePreset,
) *preset.Document {
	c, err := Plan(name, got, s.cat)
	s.Require().NoError(err)

	doc, err := preset.Blank()
	s.Require().NoError(err)

	doc.Data.Device = s.cat.DeviceID
	s.Require().NoError(doc.SetSpec(c))

	// A cabinet an amp carries is not in the chain. A preset keeps it beside
	// the routing, which is where the reading path puts it and where the
	// writing path looks for it.
	cabs := map[string]json.RawMessage{}
	pairedCabs(cabs, got, s.cat)

	for key, body := range cabs {
		doc.Data.Tone[processorKey][strings.TrimPrefix(key, processorKey+".")] = body
	}

	return doc
}

// read decodes one capture the way a device's answer is read.
func (s *EncodeTestSuite) read(
	name string,
) wire.DevicePreset {
	got, err := wire.DecodePreset(s.capture(name))
	s.Require().NoError(err)

	return got
}

// withController builds a preset from a capture and hangs one processor's
// worth of assignments off it, the way compile.Controllers writes them.
func (s *EncodeTestSuite) withController(
	body string,
) *preset.Document {
	doc := s.asPreset("one", s.read("preset.bin"))

	doc.Data.Tone[controllerKey] = preset.Tone{
		processorKey: json.RawMessage(body),
	}

	return doc
}

// aParameter is a position on the first processor and a parameter the block
// there actually carries.
//
// Discovered rather than named, so the test does not depend on which capture it
// reads or on a model keeping a control across a firmware release.
func (s *EncodeTestSuite) aParameter(
	doc *preset.Document,
) (int, string) {
	c, err := doc.Spec()
	s.Require().NoError(err)

	for _, b := range c.Blocks {
		if b.DSP != 0 {
			continue
		}

		model, ok := s.cat.SymbolNumber(b.Model)
		if !ok {
			continue
		}

		sym, _ := s.cat.Symbol(model)

		blk, ok := s.cat.Block(b.Model)
		if !ok {
			continue
		}

		for _, name := range sym.Params {
			if _, carried := blk.Params[name]; carried {
				return b.Pos, name
			}
		}
	}

	s.Require().Fail("no block in this capture carries a parameter")

	return 0, ""
}

// blockAt is the block a preset stores at a position on the first processor.
func (s *EncodeTestSuite) blockAt(
	doc *preset.Document,
	pos int,
) plan.Block {
	c, err := doc.Spec()
	s.Require().NoError(err)

	got, ok := plan.BlockAt(c.Blocks, 0, pos)
	s.Require().True(ok)

	return got
}

// model is the model table number of the block at a position.
func (s *EncodeTestSuite) model(
	doc *preset.Document,
	pos int,
) int {
	got, ok := s.cat.SymbolNumber(s.blockAt(doc, pos).Model)
	s.Require().True(ok)

	return got
}

// TestPlacements covers Placements, which turns a preset into what a device
// lays out.
//
// One method and one table, so a case is a row rather than a file.
func (s *EncodeTestSuite) TestPlacements() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The claim this file exists for.
			//
			// Every capture is read the way `presets show` reads one, written
			// back the way an import writes one, and read again. A block that
			// comes back changed is a tone somebody would hear go wrong.
			name: "a preset survives going back to the device",
			then: func() {
				for _, name := range []string{"preset.bin", "switches.bin", "empty.bin"} {
					s.Run(name, func() {
						was, err := wire.DecodePreset(s.capture(name))
						s.Require().NoError(err)

						blocks, err := Placements(s.asPreset(name, was), s.cat)
						s.Require().NoError(err)

						out, err := wire.Blank()
						s.Require().NoError(err)
						s.Require().NoError(wire.PlaceAsWritten(out, blocks))

						back, err := wire.DecodePreset(out.Encode())
						s.Require().NoError(err)

						s.Require().Equal(was.Blocks, back.Blocks)
					})
				}
			},
		},
		{
			// The ways a preset can name something a device has no number
			// for.
			name: "placements of reports what it cannot write",
			then: func() {
				tests := []struct {
					name string
					cat  *catalog.Catalog
					tone map[string]json.RawMessage
					want string
				}{
					{
						name: "a catalog with no model table",
						cat:  &catalog.Catalog{},
						want: "go generate",
					},
					{
						name: "a model the table does not carry",
						tone: map[string]json.RawMessage{
							"block0": json.RawMessage(
								`{"@model":"HD2_NoSuchThing","@position":1,"@enabled":true}`),
						},
						want: "does not carry",
					},
					{
						name: "an amp naming a cabinet the preset does not hold",
						tone: map[string]json.RawMessage{
							"block0": json.RawMessage(
								`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
									`"@cab":"cab7"}`),
						},
						want: "which the preset does not hold",
					},
					{
						name: "a cabinet entry that is not an entry",
						tone: map[string]json.RawMessage{
							"block0": json.RawMessage(
								`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
									`"@cab":"cab0"}`),
							"cab0": json.RawMessage(`5`),
						},
						want: "reading cabinet",
					},
					{
						name: "a cabinet naming no model",
						tone: map[string]json.RawMessage{
							"block0": json.RawMessage(
								`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
									`"@cab":"cab0"}`),
							"cab0": json.RawMessage(`{"Level":1}`),
						},
						want: "names no model",
					},
					{
						name: "a cabinet the table does not carry",
						tone: map[string]json.RawMessage{
							"block0": json.RawMessage(
								`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
									`"@cab":"cab0"}`),
							"cab0": json.RawMessage(`{"@model":"HD2_NoSuchCab"}`),
						},
						want: "does not carry",
					},
					{
						name: "a chain that cannot be read at all",
						tone: map[string]json.RawMessage{
							"block0": json.RawMessage(`"not a block"`),
						},
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						cat := tt.cat
						if cat == nil {
							cat = s.cat
						}

						doc, err := preset.Blank()
						s.Require().NoError(err)

						for key, body := range tt.tone {
							doc.Data.Tone[processorKey][key] = body
						}

						_, err = Placements(doc, cat)

						s.Require().Error(err)

						if tt.want != "" {
							s.Require().Contains(err.Error(), tt.want)
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

// TestValuesOf covers valuesOf, which puts a block's parameters back in the
// order a device reads them.
//
// One method and one table, so a case is a row rather than a file.
func (s *EncodeTestSuite) TestValuesOf() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The typing JSON cannot carry.
			//
			// "e" is in the model table and not in the catalog's block, which
			// is the shape of a real mismatch rather than an artificial one:
			// a cabinet's symbol list ends in `IrData`, the catalog's block
			// does not carry it, and a device writes the entry without it.
			// Ninety-two cabinets are like that.
			//
			// This used to expect "e" to take its place holding a zero. That
			// was the bug: the extra value produces a chain entry a device
			// stores, hands back unchanged, and renders as an empty chain.
			name: "values follow the catalogs word",
			then: func() {
				sym := catalog.Symbol{Params: []string{"a", "b", "c", "d", "e"}}
				types := map[string]catalog.Param{
					"a": {Type: catalog.ParamFloat},
					"b": {Type: catalog.ParamInt},
					"c": {Type: catalog.ParamBool},
					"d": {Type: catalog.ParamEnum},
				}

				tests := []struct {
					name string
					// a model the table names but gives nothing to set.
					bare bool
					// a model the catalog carries no block for, so nothing says which of
					// its parameters a chain entry holds.
					untyped bool
					params  map[string]catalog.ParamValue
					want    []any
				}{
					{
						name: "a whole number the catalog calls a fraction",
						params: map[string]catalog.ParamValue{
							"a": catalog.Int(3),
							"b": catalog.Float(4),
							"c": catalog.Bool(true),
							"d": catalog.Float(2),
						},
						want: []any{float64(3), int64(4), true, int64(2)},
					},
					{
						// These four state no default, so there is nothing better to fall
						// back to. TestASettingFallsBackToTheCatalogsDefault covers the
						// case where there is.
						name:   "a preset that carries none of them, and a catalog with no defaults",
						params: nil,
						want:   []any{float64(0), int64(0), false, int64(0)},
					},
					{
						name: "values already of the kind the catalog states",
						params: map[string]catalog.ParamValue{
							"a": catalog.Float(0.25),
							"b": catalog.Int(7),
							"c": catalog.Bool(false),
							"d": catalog.Int(1),
							"e": catalog.Bool(true),
						},
						want: []any{0.25, int64(7), false, int64(1)},
					},
					{
						// Nothing to filter against, so the whole list travels: writing
						// a short entry is the worse guess of the two.
						name:    "a model the catalog has no block for",
						untyped: true,
						params: map[string]catalog.ParamValue{
							"a": catalog.Float(0.5),
						},
						want: []any{0.5, float64(0), float64(0), float64(0), float64(0)},
					},
					{name: "a model with no parameters at all", bare: true},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						if tt.bare {
							s.Require().Nil(valuesOf(catalog.Symbol{}, nil, nil, nil))

							return
						}

						have := types
						if tt.untyped {
							have = nil
						}

						s.Require().Equal(tt.want, valuesOf(sym, tt.params, have, nil))
					})
				}
			},
		},
		{
			// A parameter nobody set.
			//
			// The failure it exists for was silent and cost a night. A plan
			// built to measure one block names the model and nothing else,
			// every absent parameter went to the device as zero, and an amp
			// with its Master and channel volume at zero came back 50dB down.
			// That reads as a chain that loaded and passed no signal, which
			// is a different bug from the one it was.
			name: "a setting falls back to the catalogs default",
			then: func() {
				sym := catalog.Symbol{Params: []string{"Master", "ChVol", "MidFreq", "Bright"}}
				types := map[string]catalog.Param{
					"Master":  {Type: catalog.ParamFloat, Default: catalog.Float(1)},
					"ChVol":   {Type: catalog.ParamFloat, Default: catalog.Float(0.8)},
					"MidFreq": {Type: catalog.ParamInt, Default: catalog.Int(2)},
					"Bright":  {Type: catalog.ParamBool, Default: catalog.Bool(true)},
				}

				s.Run("nothing is set, so every default travels", func() {
					s.Require().Equal([]any{float64(1), 0.8, int64(2), true},
						valuesOf(sym, nil, types, nil))
				})

				s.Run("a value the chain carries beats the default", func() {
					s.Require().Equal([]any{0.25, 0.8, int64(2), true},
						valuesOf(sym, map[string]catalog.ParamValue{
							"Master": catalog.Float(0.25),
						}, types, nil))
				})

				s.Run("a zero somebody set is not an absent one", func() {
					s.Require().Equal([]any{float64(0), 0.8, int64(2), false},
						valuesOf(sym, map[string]catalog.ParamValue{
							"Master": catalog.Float(0),
							"Bright": catalog.Bool(false),
						}, types, nil),
						"a knob turned down is not a knob nobody mentioned")
				})
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestMicOf covers the value a cabinet sends past its named ones.
func (s *EncodeTestSuite) TestMicOf() {
	tests := []struct {
		name   string
		fields map[string]json.RawMessage
		want   []any
	}{
		{
			name:   "a cabinet naming its microphone",
			fields: map[string]json.RawMessage{cabMic: json.RawMessage(`11`)},
			want:   []any{int64(11)},
		},
		{
			name:   "one that names none",
			fields: nil,
		},
		{
			name:   "one whose microphone is not a value",
			fields: map[string]json.RawMessage{cabMic: json.RawMessage(`{}`)},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, micOf(tt.fields))
		})
	}
}

// TestClassOf covers what a block tells the device it is.
func (s *EncodeTestSuite) TestClassOf() {
	tests := []struct {
		name   string
		model  catalog.ModelID
		paired bool
		want   int
	}{
		{
			name:  "an amp on its own",
			model: "HD2_AmpTucknGo",
			want:  wire.ClassAmp,
		},
		{
			name:   "an amp carrying its cabinet",
			model:  "HD2_AmpTucknGo",
			paired: true,
			want:   wire.ClassAmpCab,
		},
		{
			name:  "a cabinet",
			model: "HD2_Cab1x15TucknGo",
			want:  wire.ClassCab,
		},
		{
			name:  "an effect",
			model: "HD2_DistTeemah",
			want:  wire.ClassEffect,
		},
		{
			name:  "a model the catalog does not carry",
			model: "HD2_NoSuchThing",
			want:  wire.ClassEffect,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, classOf(tt.model, s.cat, tt.paired))
		})
	}
}

// TestCabNameOf covers both places a preset can name a block's cabinet.
func (s *EncodeTestSuite) TestCabNameOf() {
	tests := []struct {
		name  string
		block plan.Block
		want  string
		found bool
	}{
		{
			name: "a chain built here, where it is a parameter",
			block: plan.Block{
				Params: plan.Params{attrCab: catalog.Enum("cab0")},
			},
			want:  "cab0",
			found: true,
		},
		{
			name: "one read back out of a preset, where it is an attribute",
			block: plan.Block{
				Attrs: map[string]json.RawMessage{attrCab: json.RawMessage(`"cab1"`)},
			},
			want:  "cab1",
			found: true,
		},
		{
			name:  "a block carrying no cabinet",
			block: plan.Block{},
		},
		{
			name: "one whose cabinet is not a name",
			block: plan.Block{
				Attrs: map[string]json.RawMessage{attrCab: json.RawMessage(`7`)},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := cabNameOf(tt.block)

			s.Require().Equal(tt.found, ok)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestParamsOfJSON covers reading a cabinet entry's settings.
func (s *EncodeTestSuite) TestParamsOfJSON() {
	sym := catalog.Symbol{Params: []string{"Level", "LowCut"}}

	tests := []struct {
		name   string
		fields map[string]json.RawMessage
		want   map[string]catalog.ParamValue
	}{
		{
			name: "an entry carrying both",
			fields: map[string]json.RawMessage{
				"Level":  json.RawMessage(`0.5`),
				"LowCut": json.RawMessage(`20`),
			},
			want: map[string]catalog.ParamValue{
				"Level":  catalog.Float(0.5),
				"LowCut": catalog.Int(20),
			},
		},
		{
			name:   "one carrying neither",
			fields: nil,
			want:   map[string]catalog.ParamValue{},
		},
		{
			name: "one whose value is not a value",
			fields: map[string]json.RawMessage{
				"Level": json.RawMessage(`{}`),
			},
			want: map[string]catalog.ParamValue{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, paramsOfJSON(sym, tt.fields))
		})
	}
}

// TestTypesOf covers what the catalog says a model's parameters hold.
func (s *EncodeTestSuite) TestTypesOf() {
	s.Require().NotEmpty(typesOf("HD2_AmpTucknGo", s.cat))
	s.Require().Nil(typesOf("HD2_NoSuchThing", s.cat))
}

func TestEncodeTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EncodeTestSuite))
}

// TestPlacedControllers covers PlacedControllers, which turns what a preset
// says moves into what a device stores.
//
// One method and one table, so a case is a row rather than a file.
func (s *EncodeTestSuite) TestPlacedControllers() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The bug this resolver was written with and then found by
			// putting it on hardware.
			//
			// A preset keeps what moves under `tone.controller`, and a plan
			// built from that preset carries its blocks and not its
			// assignments. Resolving off Spec therefore found nothing, wrote
			// an empty section, and the slot came back with no controllers at
			// all — which reads exactly like the bug it was meant to fix.
			name: "placed controllers reads the section rather than the plan",
			then: func() {
				pos, name := s.aParameter(s.withController(`{}`))

				doc := s.withController(fmt.Sprintf(`{"block%d":{%q:{
					"@controller": 2, "@min": 0.3, "@max": 0.85, "@snapshot_disable": true}}}`,
					pos, name))

				got, err := PlacedControllers(doc, s.cat)
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				one := got[0]
				s.Require().Equal(2, one.Controller)
				s.Require().Equal(pos, one.Block, "the position the preset stores it at")
				s.Require().InDelta(0.3, one.Min, 0.0001)
				s.Require().InDelta(0.85, one.Max, 0.0001)
				s.Require().True(one.NoSnapshot)

				// The parameter's place in the model's own list, which is the only thing a
				// device stores. Whatever number it is, it has to name the same control
				// again when the section is read back.
				sym, ok := s.cat.Symbol(s.model(doc, pos))
				s.Require().True(ok)
				s.Require().Equal(name, sym.Params[one.Param])
			},
		},
		{
			// The ends being left out.
			//
			// Zero and one are right for most parameters here and wrong for
			// every one measured in hertz or decibels, so an assignment that
			// names neither end gets the knob's own.
			name: "placed controllers falls back to the parameters own range",
			then: func() {
				pos, name := s.aParameter(s.withController(`{}`))

				doc := s.withController(
					fmt.Sprintf(`{"block%d":{%q:{"@controller": 2}}}`, pos, name))

				got, err := PlacedControllers(doc, s.cat)
				s.Require().NoError(err)
				s.Require().Len(got, 1)

				blk, ok := s.cat.Block(s.blockAt(doc, pos).Model)
				s.Require().True(ok)

				s.Require().InDelta(blk.Params[name].Min, got[0].Min, 0.0001)
				s.Require().InDelta(blk.Params[name].Max, got[0].Max, 0.0001)
			},
		},
		{
			// A hand-edited preset.
			//
			// Both paths into a preset have been through check, where a block
			// the chain does not have and a control the model does not carry
			// are reported with every other complaint about the rig. What is
			// left is somebody's own edit, and writing it would put a
			// controller on whatever happens to sit at that number.
			name: "placed controllers skips what does not line up",
			then: func() {
				tests := []struct {
					name string
					body string
				}{
					{"a block the chain does not have", `{"block9":{"Mix":{"@controller":2}}}`},
					{"a control the model does not carry", `{"block0":{"Nope":{"@controller":2}}}`},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, err := PlacedControllers(s.withController(tt.body), s.cat)

						s.Require().NoError(err)
						s.Require().Empty(got)
					})
				}

				// Named against a control the block really has, so it reaches the check
				// for the controller number rather than being skipped before it.
				s.Run("no controller number", func() {
					pos, name := s.aParameter(s.withController(`{}`))

					got, err := PlacedControllers(s.withController(
						fmt.Sprintf(`{"block%d":{%q:{"@min":0.3}}}`, pos, name)), s.cat)

					s.Require().NoError(err)
					s.Require().Empty(got)
				})

				// A model the catalog does not carry, which is a preset built against
				// another firmware release.
				s.Run("a model this catalog has no number for", func() {
					pos, name := s.aParameter(s.withController(`{}`))

					doc := s.withController(
						fmt.Sprintf(`{"block%d":{%q:{"@controller":2}}}`, pos, name))

					c, err := doc.Spec()
					s.Require().NoError(err)

					for i := range c.Blocks {
						if c.Blocks[i].DSP == 0 && c.Blocks[i].Pos == pos {
							c.Blocks[i].Model = "HD2_NoSuchModel"
						}
					}

					s.Require().NoError(doc.SetSpec(c))

					got, err := PlacedControllers(doc, s.cat)

					s.Require().NoError(err)
					s.Require().Empty(got)
				})
			},
		},
		{
			// A hand-edited file.
			//
			// Every one of these is somebody's own edit rather than anything
			// this writes, and each is reported rather than skipped: a key
			// that is not a number says the file is wrong, where an
			// assignment that does not line up only says the chain moved
			// under it.
			name: "placed controllers reports a section it cannot read",
			then: func() {
				tests := []struct {
					name string
					body string
					says string
				}{
					{"a processor body that is not an object", `["nope"]`, "says moves"},
					{"a block key that is not a number", `{"blockX":{"Mix":{"@controller":2}}}`, "blockX"},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						_, err := PlacedControllers(s.withController(tt.body), s.cat)

						s.Require().ErrorContains(err, tt.says)
					})
				}
			},
		},
		{
			// The plan failing.
			//
			// The assignments are read from the controller section and the
			// blocks they point at from the chain, so a chain that will not
			// parse stops this before any assignment is resolved.
			name: "placed controllers reports a chain it cannot read",
			then: func() {
				doc := s.withController(`{"block0":{"Mix":{"@controller":2}}}`)
				doc.Data.Tone["dspNope"] = preset.Tone{}

				_, err := PlacedControllers(doc, s.cat)

				s.Require().ErrorContains(err, "dspNope")
			},
		},
		{
			// A malformed section.
			name: "placed controllers reports a key it cannot read",
			then: func() {
				doc := s.withController(`{"block1":{"Drive":{"@controller":2}}}`)
				doc.Data.Tone[controllerKey]["dspX"] = doc.Data.Tone[controllerKey][processorKey]

				_, err := PlacedControllers(doc, s.cat)

				s.Require().ErrorContains(err, "dspX")
			},
		},
		{
			// The common case.
			name: "a preset that moves nothing resolves nothing",
			then: func() {
				got, err := PlacedControllers(s.asPreset("one", s.read("preset.bin")), s.cat)

				s.Require().NoError(err)
				s.Require().Empty(got)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}
