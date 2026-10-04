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
package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// ResolvedPublicTestSuite covers the table a resolve prints.
type ResolvedPublicTestSuite struct {
	suite.Suite
}

// TestResolved covers Resolved, which says what a rig resolved to and how many
// controls each block carries.
//
// One method and one table, so a case is a row rather than a file.
func (s *ResolvedPublicTestSuite) TestResolved() {
	for _, tt := range []struct {
		name string
		of   result.Made
		want []string
	}{
		{
			// The count is the point: a rig that said one word about an
			// amplifier resolves to a block with a dozen controls set, and
			// seeing that is how somebody knows the document describes the
			// preset rather than gesturing at it.
			name: "a chain and how many controls each block carries",
			of: result.Made{
				Path: "/tmp/punk.yaml",
				Plan: plan.Plan{Blocks: []plan.Block{
					{
						Model: "HD2_AmpSVBeastBrt", DSP: 0, Pos: 1,
						Params: plan.Params{
							"Drive":  catalog.Float(0.6),
							"Bass":   catalog.Float(0.41),
							"BiasX":  catalog.Float(0.5),
							"Master": catalog.Float(1),
						},
					},
					{Model: "HD2_Cab8x10SVBeast", DSP: 0, Pos: 2},
				}},
			},
			want: []string{
				"HD2_AmpSVBeastBrt", "HD2_Cab8x10SVBeast",
				"4", "/tmp/punk.yaml", "CONTROLS",
			},
		},
		{
			// A rig naming no gear, which is a rig nothing can be built from.
			name: "nothing resolved",
			of:   result.Made{Path: "/tmp/empty.yaml"},
			want: []string{"the rig resolved to no blocks"},
		},
	} {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			s.Require().NoError(cli.Resolved(&buf, tt.of))

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

func TestResolvedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ResolvedPublicTestSuite))
}
