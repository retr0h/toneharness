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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// CompilePublicTestSuite covers the package's work reached as a value.
//
// Each method is the package-level function of the same name, so what is
// asserted here is that calling it through the type and calling it directly
// agree. The behaviour itself is covered by the function's own suite.
type CompilePublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *CompilePublicTestSuite) SetupSuite() { s.cat = loadCatalog(&s.Suite) }

// TestNew covers what a caller is handed.
func (s *CompilePublicTestSuite) TestNew() {
	s.Require().NotNil(compile.New())
}

// TestLift covers reading a preset into a rig through the type.
func (s *CompilePublicTestSuite) TestLift() {
	tests := []struct {
		name string
		doc  func() *preset.Document
	}{
		{name: "a blank preset", doc: func() *preset.Document {
			doc, _ := preset.Blank()

			return doc
		}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := tt.doc()
			want, wantPlan, wantErr := compile.Lift(doc, s.cat)
			got, made, err := compile.New().Lift(doc, s.cat)

			s.Require().Equal(wantErr == nil, err == nil)
			s.Require().Equal(want, got)
			// Both halves through the type, because a lift answers with two
			// documents and a method that dropped one would pass a test for the
			// other.
			s.Require().Equal(wantPlan, made)
		})
	}
}

// TestLower covers writing a plan back into a preset through the type.
func (s *CompilePublicTestSuite) TestLower() {
	tests := []struct {
		name string
		made plan.Plan
	}{
		{
			name: "a plan holding an amplifier",
			made: plan.Plan{Name: "test", Blocks: []plan.Block{
				{Model: "HD2_AmpSVBeastNrm", Enabled: true},
			}},
		},
		{name: "a plan holding nothing", made: plan.Plan{}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			first, _ := preset.Blank()
			second, _ := preset.Blank()

			want := compile.Lower(first, tt.made, s.cat)
			got := compile.New().Lower(second, tt.made, s.cat)

			s.Require().Equal(want == nil, got == nil)
			s.Require().Equal(first, second)
		})
	}
}

// TestRealise covers fitting a rig to a device through the type.
func (s *CompilePublicTestSuite) TestRealise() {
	tests := []struct {
		name string
		spec rig.Spec
	}{
		{name: "a rig naming an amplifier", spec: bassRig("Ampeg SVT", "")},
		{name: "a rig naming gear no model emulates", spec: bassRig("Nothing At All", "")},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			want, wantErr := compile.Realise(tt.spec, s.cat)
			got, err := compile.New().Realise(tt.spec, s.cat)

			s.Require().Equal(wantErr == nil, err == nil)
			s.Require().Equal(want, got)
		})
	}
}

// TestSections covers writing song sections into snapshots through the type.
func (s *CompilePublicTestSuite) TestSections() {
	blocks := []plan.Block{{Model: "HD2_AmpSVBeastNrm", Enabled: true}}
	amp := []rig.Role{rig.RoleAmp}
	drive := []rig.Role{rig.RoleDrive}

	tests := []struct {
		name string
		spec rig.Spec
	}{
		{
			name: "a section the chain can play",
			spec: rig.Spec{Sections: &[]rig.Section{{Name: "Verse", Bypass: &amp}}},
		},
		{
			name: "a section naming a role the chain lacks",
			spec: rig.Spec{Sections: &[]rig.Section{{Name: "Chorus", Play: &drive}}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			first, _ := preset.Blank()
			second, _ := preset.Blank()

			want := compile.Sections(first, tt.spec, plan.Plan{}, blocks, s.cat)
			got := compile.New().Sections(second, tt.spec, plan.Plan{}, blocks, s.cat)

			s.Require().Equal(want == nil, got == nil)
			s.Require().Equal(first, second)
		})
	}
}

// TestMoves covers turning what a foot reaches into assignments, through the
// type.
//
// The same shape as TestSections above: the method is one line delegating to
// the function, so what is worth asserting is that it delegates and leaves the
// plan where the function would.
func (s *CompilePublicTestSuite) TestMoves() {
	blocks := []plan.Block{{Model: "HD2_AmpSVBeastNrm", Enabled: true}}

	moved := []rig.Move{{
		By: rig.MoveByExpression, Role: rig.RoleAmp, Setting: "Drive",
	}}

	tests := []struct {
		name string
		spec rig.Spec
	}{
		{name: "a rig naming nothing a foot reaches"},
		{
			name: "one that names a pedal and a dial",
			spec: rig.Spec{Moves: &moved},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			first, second := plan.Plan{Blocks: blocks}, plan.Plan{Blocks: blocks}

			want := compile.Moves(&first, tt.spec, blocks, s.cat)
			got := compile.New().Moves(&second, tt.spec, blocks, s.cat)

			s.Require().Equal(want == nil, got == nil)
			s.Require().Equal(first, second)
		})
	}
}

// TestControllers covers writing what moves through the type.
func (s *CompilePublicTestSuite) TestControllers() {
	blocks := []plan.Block{{Model: "HD2_AmpSVBeastNrm", Enabled: true}}

	tests := []struct {
		name string
		made plan.Plan
	}{
		{
			name: "an assignment the chain can make",
			made: plan.Plan{Controllers: []rig.Controller{
				{Controller: 2, Block: 0, Parameter: "Drive"},
			}},
		},
		{
			name: "a plan that assigns nothing",
			made: plan.Plan{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			first, _ := preset.Blank()
			second, _ := preset.Blank()

			compile.Controllers(first, tt.made, blocks, s.cat)
			compile.New().Controllers(second, tt.made, blocks, s.cat)

			s.Require().Equal(first, second)
		})
	}
}

// TestFootswitches covers writing what the pedal prints through the type.
func (s *CompilePublicTestSuite) TestFootswitches() {
	block, switched := 0, 1

	tests := []struct {
		name string
		made plan.Plan
	}{
		{
			name: "a switch on a block",
			made: plan.Plan{Footswitches: []rig.Footswitch{
				{Switch: &switched, Block: &block},
			}},
		},
		{
			name: "a plan with no switches set",
			made: plan.Plan{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			first, _ := preset.Blank()
			second, _ := preset.Blank()

			compile.Footswitches(first, tt.made, s.cat)
			compile.New().Footswitches(second, tt.made, s.cat)

			s.Require().Equal(first, second)
		})
	}
}

// TestResolve covers turning a rig into a chain through the type.
func (s *CompilePublicTestSuite) TestResolve() {
	tests := []struct {
		name string
		spec rig.Spec
	}{
		{name: "a rig naming an amplifier", spec: bassRig("Ampeg SVT", "")},
		{name: "a rig naming gear no model emulates", spec: bassRig("Nothing At All", "")},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			want, wantAdded, _, _, wantErr := compile.Resolve(
				tt.spec, compile.Intent{}, s.cat, nil)
			got, gotAdded, _, _, err := compile.New().Resolve(tt.spec, compile.Intent{}, s.cat, nil)

			s.Require().Equal(wantErr == nil, err == nil)
			s.Require().Equal(want, got)
			s.Require().Equal(wantAdded, gotAdded)
		})
	}
}

// TestFit covers dropping what a device has no room for, through the type.
func (s *CompilePublicTestSuite) TestFit() {
	built, _, _, _, err := compile.Resolve(bassRig("Ampeg SVT", ""), compile.Intent{}, s.cat, nil)
	s.Require().NoError(err)

	tests := []struct {
		name string
		lim  plan.Limits
	}{
		{name: "room for everything", lim: twoChips(1)},
		{name: "room for nothing", lim: oneChip(0)},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(
				compile.Fit(built, s.cat, tt.lim),
				compile.New().Fit(built, s.cat, tt.lim))
		})
	}
}

func TestCompilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CompilePublicTestSuite))
}
