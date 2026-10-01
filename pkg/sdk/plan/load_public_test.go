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

package plan_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

type LoadPublicTestSuite struct {
	suite.Suite
}

const smallest = `
name: b15-eras
blocks:
  - model: HD2_AmpTucknGo
    dsp: 0
    pos: 0
    enabled: true
`

// TestLoad covers Load, which reads a plan, refusing any field it does not
// know.
//
// One method and one table, so a case is a row rather than a file.
func (s *LoadPublicTestSuite) TestLoad() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Reads a plan off a reader.
			name: "load",
			then: func() {
				tests := []struct {
					name    string
					in      string
					errText string
				}{
					{name: "the smallest plan there is", in: smallest},
					{
						// The reason Load is strict. A plan is machine-written, so the case
						// worth guarding is somebody exporting one and editing it by hand.
						name:    "a field nobody spelled right",
						in:      smallest + "targt:\n  device: HX Stomp\n",
						errText: "targt",
					},
					{
						name:    "a misspelt key inside a block",
						in:      "name: x\nblocks:\n  - model: HD2_AmpTucknGo\n    poss: 0\n",
						errText: "poss",
					},
					{
						name:    "a plan holding no blocks",
						in:      "name: x\nblocks: []\n",
						errText: "holds no blocks",
					},
					{
						name:    "something that is not YAML",
						in:      "\tnope: [",
						errText: "not a readable plan",
					},
					{
						// The guard that used to live on rig.Load, moved here with the
						// integers it protects. JSON Schema calls this an integer and Go's
						// int cannot hold it, and left unchecked it zeroed the field and
						// reported nothing.
						name: "a number larger than the type that holds it",
						in: smallest +
							"footswitches:\n  - {switch: 99999999999999999999, block: 1}\n",
						errText: "of type int",
					},
					{
						// Everything a lifted preset carries, so the fields that moved off
						// a rig are read here.
						name: "a plan carrying what a device arrived with",
						in: smallest + "rig: mike-dirnt\n" +
							"target:\n  device: HX Stomp\n" +
							"snapshots:\n  - name: SNAPSHOT 1\n" +
							"footswitches:\n  - {switch: 2, block: 1, label: Drive}\n" +
							"controllers:\n  - {controller: 2, block: 1, parameter: Drive}\n",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, err := plan.Load(strings.NewReader(tt.in))

						if tt.errText != "" {
							s.Require().Error(err)
							s.Require().ErrorIs(err, plan.ErrNotAPlan)
							s.Require().Contains(err.Error(), tt.errText)

							return
						}

						s.Require().NoError(err)
						s.Require().Len(got.Blocks, 1)
						s.Require().Equal("HD2_AmpTucknGo", string(got.Blocks[0].Model))
					})
				}
			},
		},
		{
			// The plan a device read produces.
			//
			// Off disk rather than built here, because the shape that broke
			// this is one nothing in this package assembles: the routing,
			// footswitches and snapshots a device wraps a chain in, every one
			// of them holding raw JSON.
			name: "load reads a lifted preset",
			then: func() {
				at := filepath.Join("..", "..", "..", "resources", "reference", "plan", "dir-angl-meteor.yaml")

				f, err := os.Open(at) //nolint:gosec // a path this test chose
				s.Require().NoError(err)

				defer func() { s.Require().NoError(f.Close()) }()

				first, err := plan.Load(f)
				s.Require().NoError(err)

				// The knobs and the attributes, which is what a lifted preset is mostly
				// made of and what a decode reading none of them still called a success.
				s.Require().Len(first.Blocks, 8)

				for _, block := range first.Blocks {
					s.Require().NotEmpty(block.Params, block.Model)
					s.Require().NotEmpty(block.Attrs, block.Model)

					for key, value := range block.Params {
						s.Require().NotEmpty(value.Type(), key)
					}
				}

				s.Require().Len(first.Snapshots, 3)
				s.Require().Len(first.Footswitches, 3)
				s.Require().NotNil(first.Device)
				s.Require().NotNil(first.Target)

				var out bytes.Buffer
				s.Require().NoError(plan.Write(&out, first))

				again, err := plan.Load(bytes.NewReader(out.Bytes()))
				s.Require().NoError(err)

				s.Require().Equal(first, again)
			},
		},
		{
			// Reads the fixture the coverage test counts fields in.
			//
			// Two tests over one file, because they ask different things:
			// that one is about whether a field has been written down, this
			// one about whether writing it down produces a plan. A fixture
			// nothing parses would satisfy the first and mean nothing.
			name: "load reads every field this format models",
			then: func() {
				f, err := os.Open(filepath.Join("testdata", "everything.yaml"))
				s.Require().NoError(err)

				defer func() { _ = f.Close() }()

				got, err := plan.Load(f)
				s.Require().NoError(err)

				// Each of the five that moved off a rig, since those are the ones no test
				// reached while they were declared in a contract nothing referenced.
				s.Require().NotEmpty(got.Blocks[0].Params)
				s.Require().NotEmpty(got.Blocks[0].Attrs)
				s.Require().Len(got.Snapshots, 1)
				s.Require().Len(got.Footswitches, 1)
				s.Require().Len(got.Controllers, 1)
				s.Require().NotNil(got.Device)
				s.Require().NotNil(got.Target)

				// The awkward one: what a device stored under a footswitch that this
				// format does not model, kept rather than dropped.
				s.Require().NotNil(got.Footswitches[0].Rest)

				var back strings.Builder
				s.Require().NoError(plan.Write(&back, got))

				again, err := plan.Load(strings.NewReader(back.String()))
				s.Require().NoError(err)
				s.Require().Equal(got, again)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestWrite covers Write, which writes a plan out as YAML.
//
// One method and one table, so a case is a row rather than a file.
func (s *LoadPublicTestSuite) TestWrite() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A plan surviving the round trip.
			//
			// The point of writing one at all: `presets show` hands somebody
			// a plan, and what they hand back has to be the same plan.
			name: "write reads back",
			then: func() {
				first, err := plan.Load(strings.NewReader(smallest +
					"rig: mike-dirnt\ntarget:\n  device: HX Stomp\n"))
				s.Require().NoError(err)

				var out bytes.Buffer
				s.Require().NoError(plan.Write(&out, first))

				again, err := plan.Load(bytes.NewReader(out.Bytes()))
				s.Require().NoError(err)

				s.Require().Equal(first, again)
			},
		},
		{
			// A knob surviving the round trip.
			//
			// The assertion whose absence let a plan be lossy for months. A
			// ParamValue keeps its kind in unexported fields and an attribute
			// is raw JSON, so neither is reachable by a decoder working off
			// the struct: the only caller that wrote a plan wrote blocks with
			// no params and no attrs, and nothing noticed that everything
			// else came back empty.
			name: "write reads back every parameter kind",
			then: func() {
				first := plan.Plan{
					Name: "every-kind",
					Blocks: []plan.Block{{
						Model:   "HD2_AmpTucknGo",
						Enabled: true,
						Params: plan.Params{
							"Drive":       catalog.Float(0.3500000238418579),
							"HighCut":     catalog.Int(8000),
							"MidBoost":    catalog.Bool(false),
							"SyncSelect1": catalog.Enum("Quarter"),
						},
						Attrs: map[string]json.RawMessage{
							"@no_snapshot_bypass": json.RawMessage(`false`),
							"@position":           json.RawMessage(`2`),
							"@type":               json.RawMessage(`1`),
						},
					}},
				}

				var out bytes.Buffer
				s.Require().NoError(plan.Write(&out, first))

				again, err := plan.Load(bytes.NewReader(out.Bytes()))
				s.Require().NoError(err)

				s.Require().Equal(first, again)
			},
		},
		{
			// The error nothing else would.
			name: "write reports a writer that fails",
			then: func() {
				err := plan.Write(deaf{}, plan.Plan{Name: "x", Blocks: []plan.Block{{}}})

				s.Require().Error(err)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// deaf is a writer that refuses everything.
type deaf struct{}

func (deaf) Write(
	_ []byte,
) (int, error) {
	return 0, errNope
}

var errNope = errors.New("nope")

func TestLoadPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LoadPublicTestSuite))
}
