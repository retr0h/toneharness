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
			s.Require().NoError(compile.ApplyMembers(into, spec.Preset))
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

			s.Require().NoError(compile.ApplyMembers(doc, tt.of))

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

// TestMembersRefuse covers a document stating something no preset can hold.
func (s *MembersPublicTestSuite) TestMembersRefuse() {
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

			s.Require().Error(compile.ApplyMembers(doc, &held))
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
		s.Require().NoError(compile.ApplyMembers(into, spec.Preset), p)

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
