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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/plan"
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

// TestLoad reads a plan off a reader.
func (s *LoadPublicTestSuite) TestLoad() {
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
}

// TestWriteReadsBack covers a plan surviving the round trip.
//
// The point of writing one at all: `presets show` hands somebody a plan, and
// what they hand back has to be the same plan.
func (s *LoadPublicTestSuite) TestWriteReadsBack() {
	first, err := plan.Load(strings.NewReader(smallest +
		"rig: mike-dirnt\ntarget:\n  device: HX Stomp\n"))
	s.Require().NoError(err)

	var out bytes.Buffer
	s.Require().NoError(plan.Write(&out, first))

	again, err := plan.Load(bytes.NewReader(out.Bytes()))
	s.Require().NoError(err)

	s.Require().Equal(first, again)
}

// TestWriteReportsAWriterThatFails covers the error nothing else would.
func (s *LoadPublicTestSuite) TestWriteReportsAWriterThatFails() {
	err := plan.Write(deaf{}, plan.Plan{Name: "x", Blocks: []plan.Block{{}}})

	s.Require().Error(err)
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
