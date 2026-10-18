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

package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

type GearPublicTestSuite struct {
	suite.Suite
}

// TestGear covers Gear, which is the only string a rig may name a block by.
//
// One method and one table, so a case is a row rather than a file.
func (s *GearPublicTestSuite) TestGear() {
	for _, tt := range []struct {
		name  string
		block catalog.Block
		want  string
	}{
		{
			name: "the gear it emulates, never Line 6's name for the model",
			block: catalog.Block{
				Name: "Cali 400 Ch1", BasedOn: "MESA/Boogie Bass 400+ (channel 1)",
			},
			want: "MESA/Boogie Bass 400+ (channel 1)",
		},
		{
			// The marks are in Line 6's data and in nobody's typing, and
			// their data is not consistent about them: the SVT's two channels
			// are spelled with and without one.
			name: "without the marks a manufacturer puts in",
			block: catalog.Block{
				Name: "Ampeg SVT Nrm", BasedOn: "Ampeg SVT® (normal channel)",
			},
			want: "Ampeg SVT (normal channel)",
		},
		{
			name: "a model whose own name is the gear's",
			block: catalog.Block{
				Name: "Ampeg B-15NF", BasedOn: "Ampeg B-15NF Portaflex®",
			},
			want: "Ampeg B-15NF Portaflex",
		},
		{
			// A Line 6 original emulates nothing, so "Line 6 Original" names
			// no gear and the model's own name is the only handle there is.
			name:  "a Line 6 original is named for itself",
			block: catalog.Block{Name: "Deluxe Comp", BasedOn: "Line 6 Original"},
			want:  "Deluxe Comp",
		},
		{
			name:  "a block Line 6 say nothing about",
			block: catalog.Block{Name: "Simple Delay"},
			want:  "Simple Delay",
		},
		{
			name:  "nothing to go on",
			block: catalog.Block{},
			want:  "",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.block.Gear())
		})
	}
}

func TestGearPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GearPublicTestSuite))
}
