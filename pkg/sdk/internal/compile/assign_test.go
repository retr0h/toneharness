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

package compile

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// AssignTestSuite covers giving a switch to each block worth stomping.
type AssignTestSuite struct {
	suite.Suite
}

// cat is a catalog holding one block of each kind this cares about.
func (s *AssignTestSuite) cat() *catalog.Catalog {
	return &catalog.Catalog{Blocks: map[catalog.ModelID]catalog.Block{
		"comp":  {ID: "comp", Name: "A Comp", Category: catalog.CategoryComp},
		"drive": {ID: "drive", Name: "A Drive", Category: catalog.CategoryDrive},
		"amp":   {ID: "amp", Name: "An Amp", Category: catalog.CategoryAmp},
		"cab":   {ID: "cab", Name: "A Cab", Category: catalog.CategoryCab},
		"eq":    {ID: "eq", Name: "An EQ", Category: catalog.CategoryEQ},
		"verb":  {ID: "verb", Name: "A Reverb", Category: catalog.CategoryReverb},
		"mod":   {ID: "mod", Name: "A Mod", Category: catalog.CategoryMod},
	}}
}

// chain is a plan over the models named, in that order.
func (s *AssignTestSuite) chain(
	models ...catalog.ModelID,
) plan.Plan {
	out := plan.Plan{}
	for _, m := range models {
		out.Blocks = append(out.Blocks, plan.Block{Model: m})
	}

	return out
}

// TestAssign covers which blocks get a switch, and what colour.
func (s *AssignTestSuite) TestAssign() {
	tests := []struct {
		name     string
		models   []catalog.ModelID
		switches int
		// labels is what each switch ends up on, in switch order.
		labels []string
		// leds is the colour each gets, and no two may match.
		leds []string
	}{
		{
			// The ordinary chain the corpus builds. Nothing in it is stomped,
			// because an amplifier and a cabinet are the sound and a bass
			// compressor is set once: a switch on any of them is a switch
			// wasted.
			name:     "a chain with nothing worth a switch gets none",
			models:   []catalog.ModelID{"comp", "amp", "cab"},
			switches: 3,
		},
		{
			name:     "a drive and an equaliser each get one, in chain order",
			models:   []catalog.ModelID{"comp", "drive", "amp", "cab", "eq"},
			switches: 3,
			labels:   []string{"A Drive", "An EQ"},
			leds:     []string{"red", "green"},
		},
		{
			// More to stomp than the pedal has switches. The blocks still work
			// and simply are not assignable, which is what the hardware does.
			name:     "a device runs out of switches before the chain runs out",
			models:   []catalog.ModelID{"drive", "eq", "verb", "mod"},
			switches: 2,
			labels:   []string{"A Drive", "An EQ"},
			leds:     []string{"red", "green"},
		},
		{
			// Every switch a colour no other switch has, and none of them a
			// shade of its neighbour: the closest pair here is 100 degrees of
			// hue apart. See [palette].
			name:     "each switch is a colour no other switch has",
			models:   []catalog.ModelID{"drive", "eq", "verb", "mod"},
			switches: 4,
			labels:   []string{"A Drive", "An EQ", "A Reverb", "A Mod"},
			leds:     []string{"red", "green", "blue", "light orange"},
		},
		{
			// The whole palette on one board, which is the only way to see
			// that it does what it claims: eight switches, eight colours, and
			// the closest pair of them 100 degrees of hue apart. The old
			// order put light orange and dark orange side by side here, 15
			// degrees apart and indistinguishable on the pedal.
			name: "every colour in the palette, none a shade of its neighbour",
			models: []catalog.ModelID{
				"drive", "eq", "verb", "mod",
				"drive", "eq", "verb", "mod",
			},
			switches: 8,
			labels: []string{
				"A Drive", "An EQ", "A Reverb", "A Mod",
				"A Drive", "An EQ", "A Reverb", "A Mod",
			},
			leds: []string{
				"red", "green", "blue", "light orange",
				"violet", "yellow", "turquoise", "pink",
			},
		},
		{
			// A device reporting none, which is every device the limits table
			// does not know.
			name:     "no switches to give",
			models:   []catalog.ModelID{"drive"},
			switches: 0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := Assign(s.chain(tt.models...), s.cat(), tt.switches)

			s.Require().Len(got, len(tt.labels))

			seen := map[string]bool{}

			for i, f := range got {
				s.Require().Equal(i+1, *f.Switch, "switches count from one")
				s.Require().Equal(tt.labels[i], *f.Label)
				s.Require().Equal(tt.leds[i], *f.Led)

				s.Require().False(seen[*f.Led],
					"%s is on two switches, so the board reads as one colour",
					*f.Led)
				seen[*f.Led] = true
			}
		})
	}
}

// TestNoCatalog covers being handed nothing to look a model up in.
func (s *AssignTestSuite) TestNoCatalog() {
	s.Require().Empty(Assign(s.chain("drive"), nil, 3))
}

func TestAssignTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AssignTestSuite))
}
