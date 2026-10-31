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
		// repeats says this board has more switches than the palette has
		// colours, so a colour is expected on two of them.
		repeats bool
	}{
		{
			// The ordinary chain the corpus builds, and the compressor in it
			// gets the switch.
			//
			// It used to get none, on the reasoning that a bass compressor is
			// set once. The person holding the pedal disagreed: two of seven
			// presets loaded on 2026-10-23 were a compressor, an amplifier
			// and a cabinet, and they reached the hardware with nothing on
			// any switch at all. A compressor is a pedal somebody steps on,
			// the corpus stomps 1,154 of them, and the amplifier and cabinet
			// are still the sound.
			name:     "the compressor in an ordinary chain gets a switch",
			models:   []catalog.ModelID{"comp", "amp", "cab"},
			switches: 3,
			labels:   []string{"COMP"},
			gear:     []string{"A Comp"},
			leds:     []string{"violet"},
			autos:    []int{0x080c00},
		},
		{
			// Chain order, which is the order they sit in on the floor.
			name:     "a compressor, a drive and an equaliser each get one, in chain order",
			models:   []catalog.ModelID{"comp", "drive", "amp", "cab", "eq"},
			switches: 3,
			labels:   []string{"COMP", "DRIVE", "EQ"},
			gear:     []string{"A Comp", "A Drive", "An EQ"},
			leds:     []string{"violet", "turquoise", "pink"},
			autos:    []int{0x080c00, 0x080600, 0x080c00},
		},
		{
			// A chain genuinely holding nothing to stomp, which is the case
			// the row above used to stand for.
			name:     "a chain with nothing worth a switch gets none",
			models:   []catalog.ModelID{"amp", "cab"},
			switches: 3,
		},
		{
			// More to stomp than the pedal has switches. The blocks still work
			// and simply are not assignable, which is what the hardware does.
			name:     "a device runs out of switches before the chain runs out",
			models:   []catalog.ModelID{"drive", "eq", "verb", "mod"},
			switches: 2,
			labels:   []string{"DRIVE", "EQ"},
			gear:     []string{"A Drive", "An EQ"},
			leds:     []string{"violet", "turquoise"},
		},
		{
			// Every switch a colour no other switch has, and none of them a
			// shade of its neighbour: the closest pair here is 90 degrees of
			// hue apart. See [palette].
			name:     "each switch is a colour no other switch has",
			models:   []catalog.ModelID{"drive", "eq", "verb", "mod"},
			switches: 4,
			labels:   []string{"DRIVE", "EQ", "REVERB", "MOD"},
			gear:     []string{"A Drive", "An EQ", "A Reverb", "A Mod"},
			leds:     []string{"violet", "turquoise", "pink", "blue"},
			autos:    []int{0x080600, 0x080c00, 0x100400, 0x00040d},
		},
		{
			// More switches than the palette has colours, which is a Helix
			// Floor's ten against five. The sequence repeats from the start
			// rather than leaving the rest dark.
			name: "a board with more switches than there are colours",
			models: []catalog.ModelID{
				"drive", "eq", "verb", "mod",
				"drive", "eq", "verb", "mod",
			},
			switches: 8,
			labels: []string{
				"DRIVE", "EQ", "REVERB", "MOD",
				"DRIVE 2", "EQ 2", "REVERB 2", "MOD 2",
			},
			gear: []string{
				"A Drive", "An EQ", "A Reverb", "A Mod",
				"A Drive", "An EQ", "A Reverb", "A Mod",
			},
			leds: []string{
				"violet", "turquoise", "pink", "blue", "light orange",
				"violet", "turquoise", "pink",
			},
			repeats: true,
		},
		{
			// The whole palette on one board, which is the only way to see
			// that it does what it claims: five colours, the closest pair of
			// them 90 degrees of hue apart, and none of the three every other
			// pedal on the floor uses.
			//
			// More switches than colours, so the sequence repeats rather than
			// leaving the rest dark: a board of eight on a palette of five
			// shows the first three twice. A repeat four switches apart is
			// worth less than a pair nobody can tell apart.
			name: "every colour in the palette, none a shade of its neighbour",
			models: []catalog.ModelID{
				"drive", "eq", "verb", "mod", "drive",
			},
			switches: 5,
			// And where a board carries two of a kind, the second is
			// numbered: a foot needs to tell them apart and the kind alone
			// cannot.
			labels: []string{"DRIVE", "EQ", "REVERB", "MOD", "DRIVE 2"},
			gear: []string{
				"A Drive", "An EQ", "A Reverb", "A Mod", "A Drive",
			},
			leds: []string{
				"violet", "turquoise", "pink", "blue", "light orange",
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

				// Every switch a colour no other switch has, up to the size
				// of the palette. Past that the sequence repeats rather than
				// leaving switches dark, which `repeats` marks: five colours
				// against a Helix Floor's ten switches means the sixth
				// switch shows the first colour again, and a repeat five
				// switches apart beats a dark one.
				if !tt.repeats {
					s.Require().False(seen[*f.Led],
						"%s is on two switches, so the board reads as one colour",
						*f.Led)
				}

				seen[*f.Led] = true
			}
		})
	}
}

// TestAssignOffersBeforeTheSound covers which blocks get a switch when some
// of them arrive running, what each switch's own state is set to, and where in
// the palette a preset starts.
//
// Task 278's second half. Everything [AssignTestSuite.TestAssign] covers is a
// chain where every block arrives bypassed, which is the case that cannot tell
// any of these three apart.
//
// One method and one table, so a case is a row rather than a file.
func (s *AssignTestSuite) TestAssignOffersBeforeTheSound() {
	for _, tt := range []struct {
		name string
		// on names which of the models arrive running, by position.
		models   []catalog.ModelID
		on       []bool
		switches int
		// preset is the plan's name, which the palette is rotated by.
		preset string
		// labels is what the pedal prints, in switch order.
		labels []string
		// enabled is each switch's own state, which has to agree with the
		// block's or the two fight and the switch takes three presses.
		enabled []bool
		// leds is the colour each switch gets.
		leds []string
	}{
		{
			// **The compressor is the sound and the reverb is an offer.**
			// Spending a switch on a compressor that is already on, while the
			// reverb beside it has none, is what a built punk preset did: the
			// middle switch took three presses and the reverb was
			// unreachable.
			name:     "a running compressor goes behind a bypassed reverb",
			models:   []catalog.ModelID{"comp", "amp", "cab", "verb"},
			on:       []bool{true, true, true, false},
			switches: 2,
			labels:   []string{"REVERB", "COMP"},
			enabled:  []bool{false, true},
		},
		{
			// Chain order inside each group, so a player looking down still
			// reads the offers left to right.
			name:     "bypassed blocks keep chain order among themselves",
			models:   []catalog.ModelID{"drive", "comp", "eq", "verb"},
			on:       []bool{false, true, false, false},
			switches: 4,
			labels:   []string{"DRIVE", "EQ", "REVERB", "COMP"},
			enabled:  []bool{false, false, false, true},
		},
		{
			// A chain with nothing bypassed is still stompable. Turning a
			// drive that is on back off is a thing a player does, so running
			// blocks are candidates rather than refused.
			name:     "a chain where everything is running",
			models:   []catalog.ModelID{"drive", "amp", "cab"},
			on:       []bool{true, true, true},
			switches: 2,
			labels:   []string{"DRIVE"},
			enabled:  []bool{true},
		},
		{
			// Two presets, two starting points in the palette. Walking from
			// nought meant FS1 was the first colour on all forty-three rigs
			// on one pedal: three identical lights whatever the preset.
			name:     "a named preset starts somewhere of its own",
			models:   []catalog.ModelID{"drive", "eq"},
			on:       []bool{false, false},
			switches: 2,
			preset:   "punk",
			labels:   []string{"DRIVE", "EQ"},
			enabled:  []bool{false, false},
			leds:     []string{"turquoise", "pink"},
		},
		{
			name:     "and another lands elsewhere",
			models:   []catalog.ModelID{"drive", "eq"},
			on:       []bool{false, false},
			switches: 2,
			preset:   "grunge",
			labels:   []string{"DRIVE", "EQ"},
			enabled:  []bool{false, false},
			leds:     []string{"pink", "blue"},
		},
	} {
		s.Run(tt.name, func() {
			built := s.chain(tt.models...)
			built.Name = tt.preset

			for i := range built.Blocks {
				built.Blocks[i].Pos = i
				built.Blocks[i].Enabled = tt.on[i]
			}

			got := Assign(built, s.cat(), tt.switches)

			s.Require().Len(got, len(tt.labels))

			for i, f := range got {
				s.Require().Equal(tt.labels[i], *f.Label, "switch %d", i+1)

				// The switch's own state against the block's. These have to
				// agree: the pedal applies this to the block when the preset
				// loads, so a disagreement is reconciled by the first press,
				// left disagreeing by the second and settled by the third.
				s.Require().NotNil(f.Enabled, "switch %d says nothing", i+1)
				s.Require().Equal(tt.enabled[i], *f.Enabled,
					"switch %d does not mirror its block", i+1)

				if tt.leds != nil {
					s.Require().Equal(tt.leds[i], *f.Led, "switch %d colour", i+1)
				}
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

// TestEveryStompableKindHasAColour covers the two lists agreeing.
//
// A switch written with no automatic colour sends a zero, and a zero is not a
// colour: on an attached HX Stomp it broke the whole footswitch section, not
// only the switch that carried it. That is how a wah being added to
// `catalog.Stompable` without a row in `automatic` reached 44 of the 86 presets
// one setup builds.
//
// Held here rather than trusted to review, because the two lists live in
// different packages and nothing else reads them together.
func (s *AssignTestSuite) TestEveryStompableKindHasAColour() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	// Every category the device actually models, so a kind added to the
	// catalog is covered here without anybody remembering to add it.
	kinds := map[catalog.Category]bool{}
	for _, b := range cat.Blocks {
		if catalog.Stompable(b.Category) {
			kinds[b.Category] = true
		}
	}

	s.Require().NotEmpty(kinds)

	for kind := range kinds {
		s.Run(string(kind), func() {
			on, off := lit(kind, true), lit(kind, false)

			s.Require().NotZero(on,
				"a switch on a running %s would send a zero, which is not a colour", kind)
			s.Require().NotZero(off,
				"a switch on a bypassed %s would send a zero, which is not a colour", kind)
		})
	}
}

func TestAssignTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AssignTestSuite))
}
