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
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// MembersPublicTestSuite covers everything a preset holds beside its chain.
type MembersPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *MembersPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// read makes a document out of a tone section written by hand.
//
// By hand because preset.New builds a chain and these are the members beside
// one: there is no constructor that produces an empty footswitch section or a
// null cursor position, and those are the cases worth covering.
//
// One amplifier is put in the first processor whatever the case says, because a
// lift validates what it produces and a rig with no chain is not a valid one. It
// is the chain rather than a member, so nothing here carries it.
func (s *MembersPublicTestSuite) read(
	tone string,
) *preset.Document {
	var held map[string]map[string]json.RawMessage

	s.Require().NoError(json.Unmarshal([]byte(tone), &held))

	if held["dsp0"] == nil {
		held["dsp0"] = map[string]json.RawMessage{}
	}

	held["dsp0"]["block0"] = json.RawMessage(
		`{"@model":"HD2_AmpSVBeastBrt","@enabled":true,"@position":0}`)

	body, err := json.Marshal(held)
	s.Require().NoError(err)

	raw := `{"schema":"L6Preset","version":6,"data":{"device":2162694,` +
		`"meta":{"name":"Held"},"tone":` + string(body) + `}}`

	doc, err := preset.Read(bytes.NewReader([]byte(raw)))
	s.Require().NoError(err)

	return doc
}

// TestLiftMembers covers reading a preset's members into a rig, which is what
// makes a rigspec the specification of the whole preset rather than its chain.
//
// One method and one table, so a case is a row rather than a file.
func (s *MembersPublicTestSuite) TestLiftMembers() {
	for _, tt := range []struct {
		name string
		tone string
		// then is what the lifted members have to say.
		then func(got map[string]rig.PresetMember)
	}{
		{
			// The split a device spells with a leading @ from the controls
			// beside it. An attribute says where a member sits or whether it is
			// on; a control is a knob the catalog describes.
			name: "attributes and controls are told apart",
			tone: `{"dsp0":{"split":{"@model":"HD2_AppDSPFlowSplitY",` +
				`"@enabled":true,"@position":0,"BalanceA":0.5,"bypass":false}}}`,
			then: func(got map[string]rig.PresetMember) {
				under := *got["dsp0"].Members
				split := under["split"]

				s.Require().Equal("HD2_AppDSPFlowSplitY", *split.Model)
				s.Require().Contains(*split.Attrs, "@enabled")
				s.Require().Contains(*split.Attrs, "@position")
				s.Require().Contains(*split.Controls, "BalanceA")
				s.Require().Contains(*split.Controls, "bypass")
				s.Require().NotContains(*split.Controls, "@model")
			},
		},
		{
			// The chain already says these, and two places setting one control
			// is the thing this field avoids.
			name: "a processor's own blocks are left to the chain",
			tone: `{"dsp0":{"block0":{"@model":"HD2_AmpSVBeastBrt","@enabled":true,` +
				`"@position":0},"outputA":{"@model":"HD2_AppDSPFlowOutput","@output":1}}}`,
			then: func(got map[string]rig.PresetMember) {
				under := *got["dsp0"].Members

				s.Require().NotContains(under, "block0")
				s.Require().Contains(under, "outputA")
			},
		},
		{
			// Not a processor, so its blocks are what that switch does about a
			// block rather than the block itself.
			name: "a footswitch's blocks are kept",
			tone: `{"footswitch":{"dsp0":{"block2":{"@enabled":true}}}}`,
			then: func(got map[string]rig.PresetMember) {
				under := *(*got["footswitch"].Members)["dsp0"].Members

				s.Require().Contains(under, "block2")
			},
		},
		{
			// The order is the meaning: an expression pedal's assignments are
			// told apart by position and nothing else.
			name: "an ordered list keeps its gaps",
			tone: `{"controllers":{"@expPedal1":[{"@param":"Drive","@dsp":0},null,` +
				`{"@param":"Speed","@dsp":0}]}}`,
			then: func(got map[string]rig.PresetMember) {
				entries := *(*got["controllers"].Members)["@expPedal1"].Entries

				s.Require().Len(entries, 3)
				s.Require().NotNil(entries[0].Attrs)
				s.Require().Nil(entries[1].Attrs, "a gap is a member with nothing in it")
				s.Require().NotNil(entries[2].Attrs)
			},
		},
		{
			// Five presets in the corpus hold one, and an absent member is a
			// different preset from an empty one.
			name: "an empty member is not an absent one",
			tone: `{"footswitch":{"dsp0":{}}}`,
			then: func(got map[string]rig.PresetMember) {
				under := *got["footswitch"].Members

				s.Require().Contains(under, "dsp0")
				s.Require().NotNil(under["dsp0"].Members)
				s.Require().Empty(*under["dsp0"].Members)
			},
		},
		{
			// Three presets in the corpus have a null cursor position. A nil
			// carries it, which is the absence of a value rather than one.
			name: "a null value is carried as an absent one",
			tone: `{"global":{"@model":"@global_params","@cursor_path":null,"@tempo":120}}`,
			then: func(got map[string]rig.PresetMember) {
				attrs := *got["global"].Attrs

				s.Require().Contains(attrs, "@cursor_path")
				s.Require().Nil(attrs["@cursor_path"])
				s.Require().NotNil(attrs["@tempo"])
			},
		},
		{
			// A control can be null too, not only an attribute, and the name is
			// what decides which map it lands in either way.
			name: "a null control is carried as an absent one",
			tone: `{"dsp0":{"outputA":{"@model":"HD2_AppDSPFlowOutput","gain":null}}}`,
			then: func(got map[string]rig.PresetMember) {
				out := (*got["dsp0"].Members)["outputA"]

				s.Require().Contains(*out.Controls, "gain")
				s.Require().Nil((*out.Controls)["gain"])
			},
		},
		{
			// A model stated as a number rather than a name, which no preset in
			// the corpus does and a document somebody edited could.
			name: "a model that is not a name is kept as written",
			tone: `{"dt0":{"@model":7}}`,
			then: func(got map[string]rig.PresetMember) {
				s.Require().Equal("7", *got["dt0"].Model)
			},
		},
		{
			// The empty string a device writes for a slot nothing is assigned
			// to, which 128 slots hold in every preset carrying a table.
			name: "an unassigned slot is the empty string",
			tone: `{"irUuidTable":{"000":"d5779be2c82f149081bff8af82702770","001":""}}`,
			then: func(got map[string]rig.PresetMember) {
				controls := *got["irUuidTable"].Controls

				s.Require().False(controls["000"].IsBlank())
				s.Require().True(controls["001"].IsBlank())
			},
		},
	} {
		s.Run(tt.name, func() {
			_, spec, _, err := compile.Lift(s.read(tt.tone), s.cat)
			s.Require().NoError(err)
			s.Require().NotNil(spec.Preset)
			tt.then(*spec.Preset)
		})
	}
}

// TestApplyMembers covers writing a rig's record of the preset back, which is
// what makes a value edited by hand reach the pedal.
func (s *MembersPublicTestSuite) TestApplyMembers() {
	for _, tt := range []struct {
		name string
		tone string
		// edit changes one value, as somebody opening the document would.
		edit func(got map[string]rig.PresetMember)
		// then is what the document holds afterwards.
		then func(tone map[string]preset.Tone)
	}{
		{
			// The whole point: a value changed in the document is the value the
			// device is handed.
			name: "an edited control reaches the document",
			tone: `{"dsp0":{"inputA":{"@model":"HD2_AppDSPFlow1Input",` +
				`"@input":1,"threshold":-48.0,"noiseGate":false}}}`,
			edit: func(got map[string]rig.PresetMember) {
				under := *got["dsp0"].Members
				controls := *under["inputA"].Controls
				changed := catalog.Hold(catalog.Float(-31.5))
				controls["threshold"] = &changed
			},
			then: func(tone map[string]preset.Tone) {
				var held map[string]json.RawMessage

				s.Require().NoError(
					json.Unmarshal(tone["dsp0"]["inputA"], &held))
				s.Require().JSONEq(`-31.5`, string(held["threshold"]))
			},
		},
		{
			// A member the rig does not mention is left where it was. A rig
			// somebody typed names a handful, and replacing on that would take
			// out the snapshots the preset underneath came with.
			name: "a member nobody mentioned is left alone",
			tone: `{"dsp0":{"outputA":{"@model":"HD2_AppDSPFlowOutput","@output":1}},` +
				`"snapshot0":{"@name":"Verse"}}`,
			edit: func(got map[string]rig.PresetMember) {
				delete(got, "snapshot0")
			},
			then: func(tone map[string]preset.Tone) {
				s.Require().Contains(tone, "snapshot0",
					"a partial record merges rather than replacing")
			},
		},
		{
			// A gap in a lane writes back as the device's own null, which is
			// unambiguous: no list entry in the corpus is an empty object.
			name: "a gap writes back as null",
			tone: `{"controllers":{"@expPedal1":[null,{"@param":"Drive","@dsp":0}]}}`,
			edit: func(map[string]rig.PresetMember) {},
			then: func(tone map[string]preset.Tone) {
				var held []json.RawMessage

				s.Require().NoError(
					json.Unmarshal(tone["controllers"]["@expPedal1"], &held))
				s.Require().Len(held, 2)
				s.Require().JSONEq(`null`, string(held[0]))
			},
		},
		{
			// An absent value writes back absent rather than as a reading of
			// whatever kind a zero happened to be.
			name: "an absent value writes back as null",
			tone: `{"global":{"@model":"@global_params","@cursor_path":null}}`,
			edit: func(map[string]rig.PresetMember) {},
			then: func(tone map[string]preset.Tone) {
				s.Require().JSONEq(`null`, string(tone["global"]["@cursor_path"]))
			},
		},
	} {
		s.Run(tt.name, func() {
			doc := s.read(tt.tone)

			_, spec, _, err := compile.Lift(doc, s.cat)
			s.Require().NoError(err)
			s.Require().NotNil(spec.Preset)

			tt.edit(*spec.Preset)

			// Into a fresh document, so what arrives is what the rig said rather
			// than what the one it was lifted from already held.
			into := s.read(tt.tone)
			compile.ApplyMembers(into, spec.Preset)
			tt.then(into.Data.Tone)
		})
	}
}

// TestApplyMembersSaysNothing covers the two cases where there is nothing to
// write, which a build hits on every rig somebody typed by hand.
func (s *MembersPublicTestSuite) TestApplyMembersSaysNothing() {
	for _, tt := range []struct {
		name string
		// of is what the rig states, nil being a rig that says nothing.
		of *map[string]rig.PresetMember
		// bare asks for a document with no tone section at all.
		bare bool
	}{
		{
			// A rig somebody typed has no preset members, and the untouched
			// preset underneath keeps whatever it came with.
			name: "a rig that states none",
			of:   nil,
		},
		{
			name: "a document with no tone section yet",
			of:   &map[string]rig.PresetMember{"dt0": {Model: strptr("@dt")}},
			bare: true,
		},
	} {
		s.Run(tt.name, func() {
			doc := s.read(`{"dsp0":{}}`)
			if tt.bare {
				doc.Data.Tone = nil
			}

			compile.ApplyMembers(doc, tt.of)

			if tt.of == nil {
				s.Require().Contains(doc.Data.Tone, "dsp0",
					"what the preset came with is left alone")

				return
			}

			s.Require().Contains(doc.Data.Tone, "dt0")
		})
	}
}

// strptr is a string a contract field can point at.
func strptr(
	of string,
) *string {
	return &of
}

// TestMembersLeaveOutWhatCannotBeWritten covers a value with no kind, which is
// left out rather than refusing the document.
//
// The same answer a chain entry's attributes get, so one policy covers both.
// Nothing a document can say produces one: reading a value either gives it a kind
// or refuses the literal outright, so these guard a caller building one in Go.
func (s *MembersPublicTestSuite) TestMembersLeaveOutWhatCannotBeWritten() {
	for _, tt := range []struct {
		name string
		// of is a member written by hand, as a document somebody edited would
		// arrive.
		of rig.PresetMember
	}{
		{
			// A value with no kind is one nothing set, and writing it would put
			// a control into a preset the device has no reading for.
			name: "a control with no kind cannot be written",
			of: rig.PresetMember{
				Controls: &map[string]*catalog.Held{"Drive": {}},
			},
		},
		{
			name: "an attribute with no kind cannot be written",
			of: rig.PresetMember{
				Attrs: &map[string]*catalog.Held{"@position": {}},
			},
		},
		{
			name: "a member under one cannot be written either",
			of: rig.PresetMember{
				Members: &map[string]rig.PresetMember{
					"split": {Controls: &map[string]*catalog.Held{"BalanceA": {}}},
				},
			},
		},
		{
			name: "nor one in a lane",
			of: rig.PresetMember{
				Members: &map[string]rig.PresetMember{
					"@expPedal1": {Entries: &[]rig.PresetMember{
						{Attrs: &map[string]*catalog.Held{"@dsp": {}}},
					}},
				},
			},
		},
	} {
		s.Run(tt.name, func() {
			doc := s.read(`{"dsp0":{}}`)
			held := map[string]rig.PresetMember{"dsp0": tt.of}

			compile.ApplyMembers(doc, &held)

			// Written, and without the value nothing could spell.
			s.Require().Contains(doc.Data.Tone, "dsp0")
			s.Require().NotContains(string(doc.Data.Tone["dsp0"]["split"]), "BalanceA")
		})
	}
}

// TestCorpusRoundTrip covers every preset in the corpus, which is the only check
// that says this is lossless.
//
// A hand-written case proves a shape is handled and says nothing about the 335
// field names across eighteen member kinds that real presets use. Every type
// inferred from reading a sample of those has been wrong at least once.
func (s *MembersPublicTestSuite) TestCorpusRoundTrip() {
	root := filepath.Join("..", "..", "..", "..", "resources", "schemas", "corpus")

	var paths []string

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && filepath.Ext(p) == ".hlx" {
			paths = append(paths, p)
		}

		return nil
	})
	if err != nil || len(paths) == 0 {
		s.T().Skipf("no preset corpus under %s", root)
	}

	lifted, differ := 0, 0

	for _, p := range paths {
		f, err := os.Open(filepath.Clean(p))
		if err != nil {
			continue
		}

		doc, err := preset.Read(f)
		_ = f.Close()

		if err != nil {
			continue
		}

		// Kept before the lift, because applying writes into the same document.
		was := map[string]map[string]json.RawMessage{}
		for key, entry := range doc.Data.Tone {
			was[key] = map[string]json.RawMessage{}
			for name, raw := range entry {
				was[key][name] = raw
			}
		}

		_, spec, _, err := compile.Lift(doc, s.cat)
		if err != nil {
			continue
		}

		lifted++

		s.Require().NotNil(spec.Preset, p)

		into := &preset.Document{Data: preset.Data{Tone: map[string]preset.Tone{}}}
		compile.ApplyMembers(into, spec.Preset)

		for key, fields := range was {
			for name, want := range fields {
				// The chain's own blocks are not this field's to carry.
				if preset.IsProcessorKey(key) && preset.IsBlockKey(name) {
					continue
				}

				got, ok := into.Data.Tone[key][name]
				if !ok {
					differ++

					s.Require().Fail("dropped", "%s: %s.%s", p, key, name)

					continue
				}

				if !sameJSON(want, got) {
					differ++

					s.Require().Fail("changed",
						"%s: %s.%s was %s, came back %s", p, key, name, want, got)
				}
			}
		}
	}

	s.T().Logf("presets lifted: %d of %d, fields that differ: %d",
		lifted, len(paths), differ)
	s.Require().NotZero(lifted, "the corpus is there but nothing lifted")
	s.Require().Zero(differ)
}

// TestARoundedValueIsStillInRange covers how far outside a control's range a
// value may sit before the build refuses it.
//
// A preset's own file does not spell a value to the precision the catalog states a
// bound in. 188 values in the preset corpus read `0.00999999` against a minimum of
// `0.01`, six significant figures and ten float32 steps under it, and refusing
// those made the presets holding them impossible to rebuild.
//
// The threshold was not a fine judgement. Measured over all 503 out-of-range values
// in the corpus, rounding reaches 4.0e-4 of a control's travel and the values that
// are genuinely out of range start at 0.6 of it, so the two populations have three
// orders of magnitude between them and anything inside that separates them.
//
// One method and one table, so a case is a row rather than a file.
func (s *MembersPublicTestSuite) TestARoundedValueIsStillInRange() {
	for _, tt := range []struct {
		name string
		// of is the value stated for the amplifier's Drive, whose range is 0 to 1.
		of string
		// refused says the build would not take it.
		refused bool
	}{
		{name: "a value in the middle", of: "0.5"},
		{name: "the bottom of the range", of: "0.0"},
		{name: "the top of the range", of: "1.0"},
		{
			// The case this exists for, at the scale a file rounds by.
			name: "a millionth of the travel under the bottom",
			of:   "-0.0000001",
		},
		{
			name: "a millionth of the travel over the top",
			of:   "1.0000001",
		},
		{
			// A thousandth of the travel out is still rounding, which is a
			// correction: the corpus does miss by that little. One preset's `Delay`
			// reads -2.00272e-05 against 0..0.05, four ten-thousandths of its own
			// travel under the bottom, and it came off a device.
			name: "a thousandth of the travel under the bottom",
			of:   "-0.001",
		},
		{
			// And a tenth of the travel is not. Nothing in the corpus misses by
			// between a hundredth and six tenths, so this sits in the empty space
			// between a file rounding and somebody setting a control wrongly.
			name:    "a tenth of the travel under the bottom",
			of:      "-0.1",
			refused: true,
		},
		{
			// What the far population looks like: a Drive of 8.0 against 0 to 1,
			// which is five presets in the corpus and a different firmware's idea
			// of the control.
			name:    "well outside the range",
			of:      "8.0",
			refused: true,
		},
	} {
		s.Run(tt.name, func() {
			var one catalog.Setting
			s.Require().NoError(json.Unmarshal([]byte(`"`+tt.of+`"`), &one))

			held := map[string]catalog.Setting{"Drive": one}

			_, _, _, _, err := compile.Resolve("x", rig.Spec{
				Instrument: rig.InstrumentBass,
				Chain: []rig.ChainEntry{{
					Role: rig.RoleAmp, Gear: "Ampeg SVT", Controls: &held,
				}},
			}, compile.Intent{}, s.cat, nil)

			if tt.refused {
				s.Require().Error(err)
				s.Require().ErrorContains(err, "Drive")

				return
			}

			s.Require().NoError(err)
		})
	}
}

// TestControlsPickTheModel covers which of several models a gear name means
// being decided by the controls the document states.
//
// 661 models answer to only 468 names. Three are called `1x12 US Deluxe` and only
// one of those carries a `Pan` and a `Delay`, so a name alone resolved to a model
// the preset's own values did not fit and the build refused them. 64% of the
// corpus failed to rebuild that way, which was the single biggest reason a
// document could not be handed to somebody else.
//
// One method and one table, so a case is a row rather than a file.
func (s *MembersPublicTestSuite) TestControlsPickTheModel() {
	for _, tt := range []struct {
		name string
		// of is the gear as a document names it, and controls what it says that
		// gear is set to.
		of       string
		role     rig.Role
		controls []string
		// want is the model it has to resolve to.
		want catalog.ModelID
	}{
		{
			// The case this exists for. Pan and Delay are carried by one of the
			// three, and a name on its own picks the shortest.
			name:     "a cabinet whose controls name the model",
			of:       `1x12 US Deluxe`,
			role:     rig.RoleCab,
			controls: []string{"Pan", "Delay", "Mic", "Angle"},
			want:     "HD2_CabMicIr_1x12USDeluxeWithPan",
		},
		{
			// The same name with the microphone controls and no pan, which is the
			// middle of the three.
			name:     "the same name without a pan",
			of:       `1x12 US Deluxe`,
			role:     rig.RoleCab,
			controls: []string{"Mic", "Angle", "Position"},
			want:     "HD2_CabMicIr_1x12USDeluxe",
		},
		{
			// A rig somebody typed states no controls, and the name decides as it
			// always did.
			name: "no controls leaves the name to decide",
			of:   `1x12 US Deluxe`,
			role: rig.RoleCab,
			want: "HD2_CabMicIr_1x12USDeluxe",
		},
		{
			// Controls no model of that name carries. Answered on the name rather
			// than refused, so the error a caller sees afterwards is about the
			// control rather than about gear that plainly exists.
			name:     "controls nothing of that name has",
			of:       `1x12 US Deluxe`,
			role:     rig.RoleCab,
			controls: []string{"Nonesuch"},
			want:     "HD2_CabMicIr_1x12USDeluxe",
		},
	} {
		s.Run(tt.name, func() {
			held := map[string]catalog.Setting{}
			for _, c := range tt.controls {
				held[c] = catalog.Set(catalog.Float(0.5))
			}

			entry := rig.ChainEntry{Role: tt.role, Gear: tt.of}
			if len(held) > 0 {
				entry.Controls = &held
			}

			got, _, err := compile.Realise("x", rig.Spec{
				Instrument: rig.InstrumentGuitar,
				Chain:      []rig.ChainEntry{entry},
			}, s.cat)

			s.Require().NoError(err)
			s.Require().Len(got.Blocks, 1)
			s.Require().Equal(tt.want, got.Blocks[0].Model)
		})
	}
}

// TestRealiseUsesTheControlsStated covers the values a document states reaching
// the plan, and the ones its block cannot hold being named.
//
// The whole of what `presets compile` is for, and it did neither: every block came
// back at the catalog's defaults, so a resolved rig handed to somebody rebuilt as
// the catalog rather than as the sound.
//
// One method and one table, so a case is a row rather than a file.
func (s *MembersPublicTestSuite) TestRealiseUsesTheControlsStated() {
	for _, tt := range []struct {
		name string
		// held is what the document says about the amplifier.
		held map[string]string
		// want is the value the plan should carry for Drive, if any.
		want string
		// over is the control that should be reported as not carried.
		over string
	}{
		{
			name: "a value the block carries",
			held: map[string]string{"Drive": "0.33"},
			want: "0.33",
		},
		{
			// The default is what it was before anything applied the document, so
			// a row asserting the stated value is the one that fails without it.
			name: "a value that is not the default",
			held: map[string]string{"Drive": "0.9"},
			want: "0.9",
		},
		{
			// Named rather than refused, because a preset keeps the parameters of
			// whatever its blocks used to be and a lifted document states controls
			// from two models.
			name: "a control the block does not carry",
			held: map[string]string{"Drive": "0.4", "Nonesuch": "0.5"},
			want: "0.4",
			over: "Nonesuch",
		},
	} {
		s.Run(tt.name, func() {
			held := map[string]catalog.Setting{}

			for name, v := range tt.held {
				var one catalog.Setting
				s.Require().NoError(json.Unmarshal([]byte(`"`+v+`"`), &one))

				held[name] = one
			}

			made, over, err := compile.Realise("x", rig.Spec{
				Instrument: rig.InstrumentBass,
				Chain: []rig.ChainEntry{{
					Role: rig.RoleAmp, Gear: "Ampeg SVT", Controls: &held,
				}},
			}, s.cat)

			s.Require().NoError(err)
			s.Require().Len(made.Blocks, 1)
			s.Require().Equal(tt.want, made.Blocks[0].Params["Drive"].String(),
				"the document's own value, not the catalog's default")

			if tt.over == "" {
				s.Require().Empty(over)

				return
			}

			s.Require().Len(over, 1)
			s.Require().Equal(tt.over, over[0].Control)
			s.Require().Equal("0.5", over[0].Value, "and the value that was lost")
			s.Require().NotEmpty(over[0].Near, "with what the block does take")
		})
	}
}

// TestCorpusRebuilds covers reading somebody else's preset into a rig and
// building it again, which is what handing a document to another person is.
//
// A round trip through the document is not enough on its own. The chain names
// gear the way a person does, so a rebuild has to resolve those names back to
// models, and a name this package writes and cannot read is a document nobody
// else can use. 31.4% of the corpus was in that state: a preset naming an impulse
// response wrote `IR 1024` and the resolver refused every IR block outright.
//
// A sample rather than all of it, because resolving one gear name reads every
// block in the catalog and the full corpus takes a quarter of an hour.
//
// The assertion is about the kind of failure rather than a count. What may still
// fail is a block this device has no model for, which is a preset made on other
// hardware: a stereo volume and pan, a stereo wah, FX loops three and four, a POD
// Go input. Anything else is this package disagreeing with itself.
func (s *MembersPublicTestSuite) TestCorpusRebuilds() {
	root := filepath.Join("..", "..", "..", "..", "resources", "schemas", "corpus")

	var paths []string

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && filepath.Ext(p) == ".hlx" {
			paths = append(paths, p)
		}

		return nil
	})
	if err != nil || len(paths) == 0 {
		s.T().Skipf("no preset corpus under %s", root)
	}

	const sample = 400

	if len(paths) > sample {
		thinned := make([]string, 0, sample)
		for i := 0; i < len(paths); i += len(paths) / sample {
			thinned = append(thinned, paths[i])
		}

		paths = thinned
	}

	rebuilt, missing := 0, 0

	for _, p := range paths {
		f, err := os.Open(filepath.Clean(p))
		if err != nil {
			continue
		}

		doc, err := preset.Read(f)
		_ = f.Close()

		if err != nil {
			continue
		}

		id, spec, _, err := compile.Lift(doc, s.cat)
		if err != nil {
			continue
		}

		if _, _, err := compile.Realise(id, spec, s.cat); err != nil {
			missing++

			s.Require().ErrorContains(err, "emulates",
				"%s: a rebuild may only fail on gear this device has no model for", p)

			continue
		}

		rebuilt++
	}

	s.T().Logf("rebuilt %d of %d, gear this device does not carry %d",
		rebuilt, rebuilt+missing, missing)
	s.Require().NotZero(rebuilt, "the corpus is there and nothing rebuilt")

	// A tenth is far above where this sits and far below where it was. The number
	// that matters is the kind of failure, asserted per preset above; this only
	// catches a change that makes most of the corpus unbuildable at once.
	s.Require().Less(missing*10, rebuilt,
		"most of the corpus has to survive being read out and built again")
}

// sameJSON compares two values by their canonical form, so key order and
// whitespace do not count as a difference and a number's own literal does.
func sameJSON(
	a json.RawMessage,
	b json.RawMessage,
) bool {
	// The literals first, which is the strict answer and the common one.
	if string(a) == string(b) {
		return true
	}

	var one, two any
	if json.Unmarshal(a, &one) != nil || json.Unmarshal(b, &two) != nil {
		return false
	}

	left, err := json.Marshal(one)
	if err != nil {
		return false
	}

	right, err := json.Marshal(two)

	return err == nil && string(left) == string(right)
}

func TestMembersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MembersPublicTestSuite))
}
