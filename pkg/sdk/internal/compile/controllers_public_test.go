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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// ControllersPublicTestSuite covers what moves while somebody plays reaching
// a built preset.
type ControllersPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *ControllersPublicTestSuite) SetupSuite() { s.cat = loadCatalog(&s.Suite) }

// sweep returns a pointer to one end of a controller's travel.
func sweep(
	v float32,
) *float32 {
	return &v
}

// TestControllers covers what a preset ends up holding.
func (s *ControllersPublicTestSuite) TestControllers() {
	amp := plan.Block{Model: "HD2_AmpSVBeastNrm", Pos: 1}

	tests := []struct {
		name     string
		blocks   []plan.Block
		control  []rig.Controller
		existing string
		want     string
	}{
		{
			// Left as it was, rather than emptied: a plan that says nothing
			// about what moves is not a plan saying nothing moves.
			name:     "a plan that assigns nothing",
			blocks:   []plan.Block{amp},
			existing: `{"dsp0":{"block0":{"Pedal":{"@controller":1,"@min":0,"@max":1}}}}`,
			want:     `{"dsp0":{"block0":{"Pedal":{"@controller":1,"@min":0,"@max":1}}}}`,
		},
		{
			// The expression pedal on the amplifier's drive, between two
			// settings neither of which is silence.
			name:   "a pedal on a knob",
			blocks: []plan.Block{amp},
			control: []rig.Controller{{
				Controller: 2, Block: 1, Parameter: "Drive",
				Min: sweep(0.3), Max: sweep(0.85), NoSnapshot: &yes,
			}},
			want: `{"dsp0":{"block1":{"Drive":` +
				`{"@controller":2,"@max":0.85,"@min":0.3,"@snapshot_disable":true}}}}`,
		},
		{
			// A plan that names no ends gets the control's own, which is the
			// whole of it and invents nothing.
			name:    "a pedal with no ends stated",
			blocks:  []plan.Block{amp},
			control: []rig.Controller{{Controller: 1, Block: 1, Parameter: "Interval"}},
			want: `{"dsp0":{"block1":{"Interval":` +
				`{"@controller":1,"@max":24,"@min":-24}}}}`,
		},
		{
			// A model from newer firmware than the catalog was built from is
			// still a model the device has. Nothing states its range, so the
			// assignment takes the whole of a normal control.
			name:    "a model this catalog does not carry",
			blocks:  []plan.Block{{Model: "HD2_FromNewerFirmware", Pos: 0}},
			control: []rig.Controller{{Controller: 2, Block: 0, Parameter: "Drive"}},
			want:    `{"dsp0":{"block0":{"Drive":{"@controller":2,"@max":1,"@min":0}}}}`,
		},
		{
			// A chain the device split across two processors lands on the
			// block the plan named, on the processor it named it on.
			name:   "a block on the second processor",
			blocks: []plan.Block{{Model: "HD2_AmpSVBeastNrm", DSP: 1, Pos: 4}},
			control: []rig.Controller{
				{Controller: 2, Block: 4, Path: &second, Parameter: "Drive"},
			},
			want: `{"dsp1":{"block4":{"Drive":{"@controller":2,"@max":1,"@min":0}}}}`,
		},
		{
			// Both paths count from zero, so a plan that says nothing about
			// which one means the first. The block on the second is not it.
			name:     "a position on a processor the plan did not name",
			blocks:   []plan.Block{{Model: "HD2_AmpSVBeastNrm", DSP: 1, Pos: 4}},
			existing: `{}`,
			control:  []rig.Controller{{Controller: 2, Block: 4, Parameter: "Drive"}},
			want:     `{}`,
		},
		{
			// What the preset underneath came with is not what the plan says
			// moves.
			name:     "written over assignments the preset came with",
			blocks:   []plan.Block{amp},
			existing: `{"dsp0":{"block0":{"Pedal":{"@controller":1,"@min":0,"@max":1}}}}`,
			control:  []rig.Controller{{Controller: 2, Block: 1, Parameter: "Drive"}},
			want:     `{"dsp0":{"block1":{"Drive":{"@controller":2,"@max":1,"@min":0}}}}`,
		},
		{
			// Refused by check long before this, so reaching here means the
			// chain moved underneath the assignment. Writing it onto
			// whatever sits at that position now would be worse than
			// dropping it.
			name:     "a position the chain no longer has",
			blocks:   []plan.Block{amp},
			existing: `{}`,
			control:  []rig.Controller{{Controller: 2, Block: 7, Parameter: "Drive"}},
			want:     `{}`,
		},
		{
			// The same, for a chain the device renumbered across two
			// processors: the position exists and the block at it is a
			// different one.
			name:     "a control the block at that position does not have",
			blocks:   []plan.Block{amp},
			existing: `{}`,
			control:  []rig.Controller{{Controller: 2, Block: 1, Parameter: "Warp"}},
			want:     `{}`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := preset.Blank()
			s.Require().NoError(err)

			if tt.existing != "" {
				var entry preset.Tone
				s.Require().NoError(json.Unmarshal([]byte(tt.existing), &entry))

				doc.Data.Tone["controller"] = entry
			}

			made := plan.Plan{Controllers: tt.control}

			compile.Controllers(doc, made, tt.blocks, s.cat)

			body, err := json.Marshal(doc.Data.Tone["controller"])
			s.Require().NoError(err)
			s.Require().JSONEq(tt.want, string(body))
		})
	}
}

// yes is a plan saying a field is true.
var yes = true

// second is the other processor, on a device that has one.
var second = 1

// TestLowerWritesWhatMoves covers the assignments reaching a compiled preset.
func (s *ControllersPublicTestSuite) TestLowerWritesWhatMoves() {
	made := realised(&s.Suite, bassRig("Ampeg SVT", ""), s.cat)
	made.Controllers = []rig.Controller{
		{Controller: 2, Block: 0, Parameter: "Drive", Min: sweep(0.3), Max: sweep(0.85)},
	}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().NoError(compile.Lower(doc, made, s.cat))

	body, err := json.Marshal(doc.Data.Tone["controller"])
	s.Require().NoError(err)
	s.Require().JSONEq(
		`{"dsp0":{"block0":{"Drive":{"@controller":2,"@max":0.85,"@min":0.3}}}}`,
		string(body))
}

// TestLowerRefusesAnAssignmentItCannotMake covers the check that runs before
// anything is written, so a plan naming a block its own chain does not have
// fails rather than building a preset with the pedal on nothing.
func (s *ControllersPublicTestSuite) TestLowerRefusesAnAssignmentItCannotMake() {
	made := realised(&s.Suite, bassRig("Ampeg SVT", ""), s.cat)
	made.Controllers = []rig.Controller{
		{Controller: 2, Block: 9, Parameter: "Drive"},
	}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().ErrorIs(compile.Lower(doc, made, s.cat), compile.ErrNoSuchBlock)
	s.Require().NotContains(doc.Data.Tone, "controller")
}

// TestLowerRefusesAProcessorTheDeviceDoesNotHave covers a plan naming the
// second path on a chain that has one.
func (s *ControllersPublicTestSuite) TestLowerRefusesAProcessorTheDeviceDoesNotHave() {
	made := realised(&s.Suite, bassRig("Ampeg SVT", ""), s.cat)
	made.Controllers = []rig.Controller{
		{Controller: 2, Block: 0, Path: &second, Parameter: "Drive"},
	}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().ErrorIs(compile.Lower(doc, made, s.cat), compile.ErrNoSuchBlock)
}

func TestControllersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ControllersPublicTestSuite))
}
