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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

type RigPublicTestSuite struct {
	suite.Suite
}

func (s *RigPublicTestSuite) TestSpecRoundTripsThroughJSON() {
	in := plan.Plan{
		Name: "Test Rig",
		Blocks: []plan.Block{
			{
				Model: "HD2_AmpTest",
				Params: map[string]catalog.ParamValue{
					"Gain": catalog.Float(0.5),
				},
				DSP:     0,
				Pos:     0,
				Enabled: true,
			},
		},
	}

	b, err := json.Marshal(in)
	s.Require().NoError(err)

	var out plan.Plan
	s.Require().NoError(json.Unmarshal(b, &out))
	s.Require().Equal(in, out)
}

// TestLimitsForNamesTheDevice covers each device's ceilings being its own.
func (s *RigPublicTestSuite) TestLimitsForNamesTheDevice() {
	tests := []struct {
		name   string
		device string
		blocks int
		paths  int
		why    string
	}{
		{
			name: "an HX Stomp", device: "HX Stomp", blocks: 8, paths: 1,
			why: "721 presets, the largest holding eight blocks, none on a second processor",
		},
		{
			name: "an HX Stomp XL", device: "HX Stomp XL", blocks: 8, paths: 1,
			why: "the same two chips in a bigger box",
		},
		{
			name: "a Helix Floor", device: "Helix Floor", blocks: 29, paths: 2,
			why: "1,698 presets, the largest holding 29 blocks, 1,219 on two processors",
		},
		{
			name: "a Helix LT", device: "Helix LT", blocks: 29, paths: 2,
			why: "the Helix Floor's processing in a smaller box",
		},
		{
			name: "a device named the way somebody typed it", device: "helix floor",
			blocks: 29, paths: 2, why: "a catalog's name is not case",
		},
		{
			name: "a device nothing here knows", device: "Pod Go", blocks: 8, paths: 1,
			why: "the smallest ceilings here, because a chain that fits them fits the rest",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			l := plan.LimitsFor(tt.device)

			s.Require().Equal(tt.blocks, l.MaxBlocks, tt.why)
			s.Require().Equal(tt.paths, l.Paths, tt.why)
			s.Require().InDelta(95.0, l.ChipCeiling, 1e-9)
		})
	}
}

func (s *RigPublicTestSuite) TestHXStompLimitsAreTheDocumentedCeilings() {
	l := plan.HXStompLimits()

	s.Require().Equal(8, l.MaxBlocks,
		"the corpus shows HX Stomp presets holding eight blocks")
	s.Require().Equal(1, l.Paths,
		"no HX Stomp preset in the corpus has a second signal path")
	s.Require().InDelta(95.0, l.ChipCeiling, 1e-9)
	s.Require().Greater(l.ChipCeiling, 1.0,
		"the ceiling is a percentage, matching how Line 6 states a block's cost")
}

// TestBlockStereo covers which of a model's two parameter orders addresses
// this instance.
//
// One method and one table, so a case is a row rather than a file.
func (s *RigPublicTestSuite) TestBlockStereo() {
	for _, tt := range []struct {
		name  string
		attrs map[string]json.RawMessage
		want  bool
	}{
		{
			// What a device writes for a block it runs in stereo.
			name:  "stereo when the preset says so",
			attrs: map[string]json.RawMessage{"@stereo": json.RawMessage(`true`)},
			want:  true,
		},
		{
			name:  "mono when the preset says so",
			attrs: map[string]json.RawMessage{"@stereo": json.RawMessage(`false`)},
			want:  false,
		},
		{
			// The common case, and the one that has to be mono: a device loads
			// a block mono unless a preset asks otherwise, so an absent
			// attribute is an answer rather than a gap.
			name:  "mono when the preset says nothing",
			attrs: nil,
			want:  false,
		},
		{
			// Rather than a panic or a true. A preset carrying something that
			// is not a boolean here is a preset nothing can trust about
			// stereo, and mono is what the device would have loaded.
			name:  "mono when the attribute is not a boolean",
			attrs: map[string]json.RawMessage{"@stereo": json.RawMessage(`"yes"`)},
			want:  false,
		},
	} {
		s.Run(tt.name, func() {
			got := plan.Block{Model: "HD2_Chorus", Attrs: tt.attrs}.Stereo()

			s.Require().Equal(tt.want, got)
		})
	}
}

func TestRigPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RigPublicTestSuite))
}
