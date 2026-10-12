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

package presets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/internal/fileslots"
	"github.com/retr0h/toneharness/pkg/sdk/internal/presets"
	presetmocks "github.com/retr0h/toneharness/pkg/sdk/internal/presets/mocks"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/slot"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

type CompilePublicTestSuite struct {
	suite.Suite
}

// slotFixture is a file the slot flows test against, which a rig is exported
// out of here.
func slotFixture(
	name string,
) string {
	return filepath.Join("..", "fileslots", "testdata", name)
}

// planFixture is a plan written by hand, for what only a plan carries.
func planFixture(
	name string,
) string {
	return filepath.Join("testdata", "plans", name)
}

// catalogs hands over the catalog at path, however often it is asked.
func (s *CompilePublicTestSuite) catalogs(
	path string,
) *presetmocks.MockCatalogs {
	c := presetmocks.NewMockCatalogs(gomock.NewController(s.T()))
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) { return catalog.Open(path) },
	).AnyTimes()

	return c
}

// handWritten returns a rig nobody lifted from a preset: gear and nothing
// else, which is what somebody typing one produces.
func (s *CompilePublicTestSuite) handWritten(
	dir string,
) string {
	out := filepath.Join(dir, "typed.yaml")

	s.Require().NoError(os.WriteFile(out, []byte(`schema: ToneSpec
id: typed
rig:
  instrument: bass
  chain:
    - { role: amp, gear: Ampeg SVT }
`), 0o600))

	return out
}

// exported writes a slot out as a rig and returns where it went.
func (s *CompilePublicTestSuite) exported(
	dir string,
) string {
	out := filepath.Join(dir, "rig.yaml")

	_, err := (&fileslots.Flows{Catalogs: s.catalogs(slotFixture("catalog.json"))}).
		Export(context.Background(),
			slotFixture("setlist.hls"), slot.Address{}, out, result.FormatRig,
			result.ReplaceExisting)
	s.Require().NoError(err)

	return out
}

// exportedPlan writes a slot out as the plan that realises it on this device
// and returns where it went.
func (s *CompilePublicTestSuite) exportedPlan(
	dir string,
) string {
	read, err := (&fileslots.Flows{Catalogs: s.catalogs(slotFixture("catalog.json"))}).
		Show(context.Background(), slotFixture("setlist.hls"), slot.Address{})
	s.Require().NoError(err)

	out := filepath.Join(dir, "plan.yaml")

	var buf bytes.Buffer

	s.Require().NoError(plan.Write(&buf, read.Plan))
	s.Require().NoError(os.WriteFile(out, buf.Bytes(), 0o600))

	return out
}

// unknownGear writes a valid rig naming gear no catalog carries.
func (s *CompilePublicTestSuite) unknownGear(
	dir string,
) string {
	path := filepath.Join(dir, "unknown.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(
		"schema: ToneSpec\nid: unknown\nrig:\n"+
			"  instrument: guitar\n  chain:\n    - {role: amp, gear: Nonesuch 900}\n"),
		0o600))

	return path
}

// emptyChain writes a rig with no chain, which the contract refuses.
// nameless writes a rig whose chain entry names no gear.
//
// An empty chain used to be the handiest invalid rig and is a valid one now: a
// preset that makes no sound is one somebody meant. A chain entry with nothing in
// its gear field is still nothing anybody can build.
func (s *CompilePublicTestSuite) nameless(
	dir string,
) string {
	path := filepath.Join(dir, "bad.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(
		"schema: ToneSpec\nid: x\nrig:\n"+
			"  instrument: bass\n  chain:\n    - role: amp\n      gear: \"\"\n"), 0o600))

	return path
}

// pastItsRange writes a rig whose control is set well outside what the catalog
// says the control can take.
//
// Reachable only since compiling started using the values a document states: when
// every block came back at the catalog's defaults there was no way for a document
// to put a bad one into the plan, and `plan.Validate` had nothing to catch.
func (s *CompilePublicTestSuite) pastItsRange(
	dir string,
) string {
	path := filepath.Join(dir, "past.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(
		"schema: ToneSpec\nid: past\nrig:\n"+
			"  instrument: bass\n  chain:\n    - role: amp\n"+
			"      gear: Ampeg SVT\n      controls: {Drive: \"8.0\"}\n"), 0o600))

	return path
}

// TestCompile covers turning a rig into a preset.
func (s *CompilePublicTestSuite) TestCompile() {
	tests := []struct {
		name string
		ctx  context.Context
		// which rig to build: one exported from a slot unless a case says
		// otherwise.
		rig string
		// which plan to build, for a case about what only a plan carries.
		// Exactly one of the two reaches the build, unless both says
		// otherwise.
		plan string
		both bool
		// a preset to write the chain into, rather than an untouched one.
		template string
		catalog  string
		out      string

		// how many blocks the built preset must report.
		blocks int
		// what the written preset must say, and must not.
		contains []string
		absent   []string
		// the chain must come back out of what was written.
		loadable bool

		// a compiler that leaves the preset unable to encode.
		unencodable bool

		err     error
		errText string
	}{
		{
			name:     "a rig lifted off a slot",
			blocks:   3,
			loadable: true,
		},
		{
			// A device expects inputs, outputs, a split and a join around a
			// chain. 98.6% of real presets carry them, and one assembled from
			// nothing carries none, so a compiled preset is written into an
			// untouched one.
			name:     "a rig somebody typed",
			rig:      "hand-written",
			contains: []string{"inputA", "outputA", "split", "join", "snapshot0"},
		},
		{
			// A lifted plan carries what the preset it came from carried, and
			// that wins over whatever the preset being written into holds.
			// Otherwise a plan shared with somebody else would rebuild with a
			// stranger's routing. The rig is the portable layer and states
			// none of it, so only a plan can make this claim.
			name:     "a lifted plan, written into somebody else's preset",
			plan:     "exported",
			template: slotFixture("preset.hlx"),
			absent:   []string{"controller"},
		},
		{
			name:     "a typed rig, written into a template",
			rig:      "hand-written",
			template: slotFixture("preset.hlx"),
			// A rig nobody lifted carries no state, so the template's is
			// kept.
			contains: []string{"controller"},
		},
		{
			// The pedal under somebody's foot is part of the plan, and a
			// preset built without it is one where the pedal does nothing.
			name:     "a plan with a pedal on a knob",
			plan:     planFixture("with-pedal.yaml"),
			loadable: true,
			contains: []string{`"@controller": 2`, `"@max": 0.85`},
		},
		{
			// What the pedal prints under a switch is a decision somebody
			// made once and reads every time they play. A built preset that
			// dropped it would give two different pedals from one plan.
			name:     "a plan with a label under a switch",
			plan:     planFixture("with-switch.yaml"),
			loadable: true,
			contains: []string{`"@fs_label": "Chunk"`},
		},
		{name: "a rig that is not there", rig: slotFixture("nope.yaml"), errText: "opening"},
		{
			// Refused before anything is written, because the device rejects the
			// whole preset over one value past a control's end and finding that out
			// from the pedal is worse than from a message naming the control.
			name:    "a control set past what it can take",
			rig:     "past-its-range",
			errText: "will not load",
		},
		{
			// A plan is written by a driver and edited by hand, so a knob no
			// model carries reaches here. A pedal moving whatever happens to
			// sit at that position is worse than a refusal.
			name:    "a plan whose pedal moves a knob the model does not have",
			plan:    planFixture("bad-pedal.yaml"),
			err:     compile.ErrNoSuchValue,
			errText: "Nonesuch",
		},
		{name: "a plan that is not there", plan: planFixture("nope.yaml"), errText: "opening"},
		{
			name:    "a file that is not a plan",
			plan:    slotFixture("setlist.hls"),
			err:     plan.ErrNotAPlan,
			errText: "not a readable plan",
		},
		{
			// A rig is realised on the way through and a plan already is, so
			// there is no answer to being handed both, and none to neither.
			name: "neither a rig nor a plan",
			rig:  "none",
			err:  presets.ErrOnePath,
		},
		{
			name: "a rig and a plan at once",
			plan: planFixture("with-pedal.yaml"),
			both: true,
			err:  presets.ErrOnePath,
		},
		{
			name:    "a caller who stopped waiting for a plan",
			ctx:     cancelledContext(),
			plan:    planFixture("with-pedal.yaml"),
			errText: context.Canceled.Error(),
		},
		{
			name:    "a file that is not a document",
			rig:     slotFixture("setlist.hls"),
			errText: "not a valid ToneSpec",
		},
		{
			name:    "a rig that does not meet its own contract",
			rig:     "nameless gear",
			err:     tone.ErrInvalid,
			errText: "gear",
		},
		{
			name:    "a rig naming gear this device does not model",
			rig:     "unknown gear",
			errText: "emulates \"Nonesuch 900\"",
		},
		{
			name:     "a template that is not there",
			template: slotFixture("nope.hlx"),
			errText:  "opening",
		},
		{
			name:     "a template that is not a preset",
			template: slotFixture("notapreset.hlx"),
			errText:  "reading",
		},
		{
			name:    "a catalog that is not there",
			catalog: slotFixture("nope.json"),
			errText: "catalog",
		},
		{
			// Reported, rather than a file holding nothing where a preset
			// was meant to be.
			name:        "a preset that will not encode",
			unencodable: true,
			errText:     "encoding preset",
		},
		{
			name:    "a destination directory that is not there",
			out:     filepath.Join("no", "out.hlx"),
			errText: "writing",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelledContext(),
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "out.hlx")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			var rigPath, planPath string

			switch {
			case tt.rig == "none":
			case tt.both:
				planPath, rigPath = tt.plan, s.handWritten(dir)
			case tt.plan == "exported":
				planPath = s.exportedPlan(dir)
			case tt.plan != "":
				planPath = tt.plan
			case tt.rig == "":
				rigPath = s.exported(dir)
			case tt.rig == "hand-written":
				rigPath = s.handWritten(dir)
			case tt.rig == "unknown gear":
				rigPath = s.unknownGear(dir)
			case tt.rig == "nameless gear":
				rigPath = s.nameless(dir)
			case tt.rig == "past-its-range":
				rigPath = s.pastItsRange(dir)
			default:
				rigPath = tt.rig
			}

			cat := slotFixture("catalog.json")
			if tt.catalog != "" {
				cat = tt.catalog
			}

			deps := presets.Deps{Catalogs: s.catalogs(cat)}

			if tt.unencodable {
				compiler := presetmocks.NewMockCompiler(gomock.NewController(s.T()))
				compiler.EXPECT().Realise(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(compile.New().Realise)
				compiler.EXPECT().
					Moves(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(compile.New().Moves)
				compiler.EXPECT().
					Lower(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(doc *preset.Document, _ plan.Plan, _ *catalog.Catalog) error {
						doc.Meta = json.RawMessage("{")

						return nil
					})

				deps.Compiler = compiler
			}

			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := presets.Compile(ctx, presets.CompileOptions{
				Deps:         deps,
				RigPath:      rigPath,
				PlanPath:     planPath,
				TemplatePath: tt.template,
				OutputPath:   out,
			})

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)
				s.Require().NoFileExists(out)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().ErrorContains(err, tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().NotEmpty(got.Name)

			if tt.blocks != 0 {
				s.Require().Equal(tt.blocks, got.Blocks)
			}

			raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(string(raw), want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(string(raw), unwanted)
			}

			if !tt.loadable {
				return
			}

			doc, err := preset.Read(bytes.NewReader(raw))
			s.Require().NoError(err)

			c, err := doc.Spec()
			s.Require().NoError(err)
			s.Require().NotEmpty(c.Blocks)
		})
	}
}

func TestCompilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CompilePublicTestSuite))
}
