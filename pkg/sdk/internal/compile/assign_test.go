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
		// labels is what the pedal prints under each switch, in switch order.
		//
		// The kind of block rather than its name, which is the whole of task
		// 263: the pedal has room for about six characters and a gear name
		// runs to thirty, so FS1 on a built preset read "Ampeg Scrambler Bass
		// Overdrive".
		labels []string
		// gear is what each switch's block is called, which a device keeps
		// beside the label and this used to send as both.
		gear []string
		// leds is the colour each gets, and no two may match.
		leds []string
		// autos is the colour the switch would light with nobody choosing,
		// by the kind of block and bypassed, which is the state a switch
		// arrives in. See [automatic].
		autos []int
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
			labels:   []string{"DRIVE", "EQ"},
			gear:     []string{"A Drive", "An EQ"},
			leds:     []string{"red", "green"},
			autos:    []int{0x080600, 0x080c00},
		},
		{
			// More to stomp than the pedal has switches. The blocks still work
			// and simply are not assignable, which is what the hardware does.
			name:     "a device runs out of switches before the chain runs out",
			models:   []catalog.ModelID{"drive", "eq", "verb", "mod"},
			switches: 2,
			labels:   []string{"DRIVE", "EQ"},
			gear:     []string{"A Drive", "An EQ"},
			leds:     []string{"red", "green"},
		},
		{
			// Every switch a colour no other switch has, and none of them a
			// shade of its neighbour: the closest pair here is 100 degrees of
			// hue apart. See [palette].
			name:     "each switch is a colour no other switch has",
			models:   []catalog.ModelID{"drive", "eq", "verb", "mod"},
			switches: 4,
			labels:   []string{"DRIVE", "EQ", "REVERB", "MOD"},
			gear:     []string{"A Drive", "An EQ", "A Reverb", "A Mod"},
			leds:     []string{"red", "green", "blue", "light orange"},
			autos:    []int{0x080600, 0x080c00, 0x100400, 0x00040d},
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
			// And where a board carries two of a kind, the second is
			// numbered: a foot needs to tell them apart and the kind alone
			// cannot.
			labels: []string{
				"DRIVE", "EQ", "REVERB", "MOD",
				"DRIVE 2", "EQ 2", "REVERB 2", "MOD 2",
			},
			gear: []string{
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
				s.Require().Equal(tt.labels[i], *f.Label,
					"what the pedal prints")
				s.Require().Equal(tt.gear[i], *f.Gear, "what the block is")
				s.Require().Equal(tt.leds[i], *f.Led)

				if tt.autos != nil {
					s.Require().NotNil(f.Colour,
						"a switch with no automatic colour carries a zero, "+
							"which is not a colour")
					s.Require().Equal(tt.autos[i], *f.Colour,
						"the colour the block would light with")
				}

				// Every label short enough to read from three feet away,
				// which is what the switch is for.
				s.Require().LessOrEqual(len(*f.Label), 8,
					"%q will not fit under a switch", *f.Label)

				s.Require().False(seen[*f.Led],
					"%s is on two switches, so the board reads as one colour",
					*f.Led)
				seen[*f.Led] = true
			}
		})
	}
}

// TestLit covers the colour a switch lights with nobody choosing.
//
// Task 264. `@fs_ledcolor` is the block's own colour and was written as zero,
// which is not a colour. Mined from the preset corpus: 99.2% of the 5,819
// assignments in these categories agree with the table, which is why it is
// fourteen numbers rather than one per model.
//
// One method and one table, so a case is a row rather than a file.
func (s *AssignTestSuite) TestLit() {
	tests := []struct {
		name string
		of   catalog.Category
		on   bool
		want int
	}{
		{
			// The off state is the same hue at about a sixteenth of the
			// brightness, which is what the pedal does to a bypassed block.
			name: "a drive switched off",
			of:   catalog.CategoryDrive, want: 0x080600,
		},
		{
			name: "and the same drive on",
			of:   catalog.CategoryDrive, on: true, want: 0xd0af00,
		},
		{
			// Three pairs repeat across the seven, which is the device
			// colouring by what a block does rather than by which block it
			// is: a filter and a pitch shifter share one.
			name: "a filter and a pitch shifter share a colour",
			of:   catalog.CategoryFilter, on: true, want: 0x3200ff,
		},
		{
			name: "which is the pitch shifter's too",
			of:   catalog.CategoryPitch, on: true, want: 0x3200ff,
		},
		{
			// A utility block never gets a switch, and it is the one category
			// whose readings do not settle: 462860 on 58.5% of 573. Zero
			// leaves the field out rather than writing a number nobody read.
			name: "a category nothing measured claims nothing",
			of:   catalog.CategoryUtility,
		},
		{
			name: "nor does a category the catalog does not name",
			of:   "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, lit(tt.of, tt.on))
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
