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
	"strconv"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// stomped is the kinds of block a player turns on and off with a foot, in the
// order a chain holds them.
//
// Not every block is worth a switch, and the ones that are not are the point
// of the list. An amplifier and a cabinet are the sound rather than an effect
// on it: bypassing either mid-song is not a thing anybody does, and a switch
// spent on one is a switch not spent on the drive. A compressor on a bass is
// set once and left, which is why it is absent here and why every chain the
// corpus builds carries one without anybody reaching for it.
//
// What is left is what a pedalboard would have had on the floor.
var stomped = map[catalog.Category]bool{
	catalog.CategoryDrive:  true,
	catalog.CategoryFilter: true,
	catalog.CategoryMod:    true,
	catalog.CategoryDelay:  true,
	catalog.CategoryReverb: true,
	catalog.CategoryPitch:  true,
	catalog.CategoryEQ:     true,
}

// palette is the colours a switch is given, in the order they are handed out.
//
// Unique per switch rather than per kind of block, so a board where every
// switch is on is a board where no two lights look alike.
//
// The device names twelve and the catalog reads them off it in the order it
// numbers them, which is where `auto color` being 0, red 2 and off 11 come
// from. Three carry no hue: `auto` follows the block, and white and off are
// not colours to tell apart. Of the nine that remain, dark orange and light
// orange sit 15 degrees apart on the hue circle and no player can tell them
// from each other, so one of them is dropped. Two colours that look the same
// cannot both be distinct, and keeping both only guarantees a board that has
// a pair of them somewhere.
//
// The eight left are ordered to make the worst pair as good as possible rather
// than by stepping the circle greedily. Stepping it farthest-first reads well
// and ends badly: it spends the far colours early and leaves the neighbours to
// collide at the end, which is how the old order put light orange next to dark
// orange on switches 8 and 9, 15 degrees apart.
//
// Ordered this way, measured in degrees of hue:
//
//	the closest pair among the first three  100   an HX Stomp shows three
//	the closest pair among all eight         20   and they are five apart
//	the closest pair on adjacent switches   100
//
// Red stays first because it is the one somebody expects on FS1, and it costs
// 5 degrees against the best order that does not: red, green and blue are the
// three an HX Stomp shows, which is as legible as three lights get.
//
// **This is a visual choice. The hue figures are read off a colour wheel and
// not off the pedal**, so they rank the orderings against each other rather
// than describing what the LEDs emit. Anybody who dislikes the result says so
// in the rig, where a footswitch somebody wrote outranks this entirely.
//
// What it gives up is worth naming. `auto` lights by what the block is, so an
// untouched preset shows red for an amp, amber for drive and blue for
// modulation, and a player learns that one mapping across every preset they
// own. This trades it for a board legible at arm's length, which is the thing
// to want when the switches are three feet away and all of them are lit.
var palette = []string{
	"red", "green", "blue", "light orange",
	"violet", "yellow", "turquoise", "pink",
}

// printed is what the pedal prints under a switch.
//
// **The pedal has room for about six characters and the gear's own name is up
// to thirty.** FS1 on a built preset read "Ampeg Scrambler Bass Overdrive",
// which is a switch that works and cannot be read from three feet away. A
// device keeps two labels per assignment and uses them differently: HX Edit's
// own presets put the gear in `@fs_label` ("Clawthorn Drive", "Bass Octaver")
// and what the switch prints in `@fs_customlabel` ("FUZZ", "Oct -1"). This
// tool sent the gear name as both.
//
// The kind of block, which is the role the rig already states and the same
// word the catalog files it under. Upper case, the way a label somebody wrote
// by hand reads: DRIVE, EQ, MOD, DELAY, REVERB, PITCH, FILTER. Every one of
// the seven fits, the longest being six.
//
// **Measured against the alternative, which was the gear's last distinctive
// word.** Across the eleven stompable blocks the shipped rigs name, that gives
// Overdrive, MT-2, 250, Pi, Distortion, GT2, Booster, EQ, EQ, Tilt and III.
// Four of those say nothing about what the switch does, two collide, and
// Distortion and Overdrive are still nine and ten characters. The role's
// failure is milder and fixable: two blocks of a kind read alike, and the
// colours are already distinct per switch.
//
// So the second of a kind is numbered. DRIVE then DRIVE 2, which is seven
// characters and the only thing a foot needs to tell them apart.
//
// Nothing here invents an abbreviation per block. That is a table of 661
// entries nobody has written, and a person who wants FUZZ on their fuzz says
// so in the rig, which outranks this.
func printed(
	of catalog.Category,
	already []rig.Footswitch,
) string {
	label := strings.ToUpper(string(of))

	// Counted over what this has handed out rather than over the chain,
	// because a block past the last switch gets no label to collide with.
	same := 0

	for _, fs := range already {
		if fs.Label != nil && strings.HasPrefix(*fs.Label, label) {
			same++
		}
	}

	if same == 0 {
		return label
	}

	return label + " " + strconv.Itoa(same+1)
}

// Assign gives a switch to each block worth stomping, in chain order.
//
// A built preset had none. `plan.Footswitches` was only ever filled by a lift,
// which is to say by reading a preset somebody else had already set up, so
// everything this project built arrived with its switches unassigned: the
// blocks were there, in the right order, at the right values, and nothing on
// the pedal turned any of them off.
//
// Chain order rather than importance, because the chain is the order they sit
// in on the floor and a player looking down expects the first switch to be the
// first pedal. Importance would be this deciding something musical.
//
// Capped by what the device has: an HX Stomp's three against a Helix Floor's
// ten, from [plan.Limits]. Blocks past the last switch keep working and are
// simply not assignable, which is what happens on the hardware too.
//
// The label is what the switch does and the gear is what the block is, which
// is the distinction a device already makes and this collapsed: see [printed].
//
// A person's to change either way. A rig's own `footswitch:` outranks all of
// this, so what is decided here is only the default.
func Assign(
	built plan.Plan,
	cat *catalog.Catalog,
	switches int,
) []rig.Footswitch {
	if switches <= 0 || cat == nil {
		return nil
	}

	out := []rig.Footswitch(nil)

	for i, b := range built.Blocks {
		// Through the catalog rather than off the plan, because a plan holds a
		// model identifier and the two questions here — what kind of block is
		// this, and what is it called — are both the catalog's.
		held, known := cat.Block(b.Model)
		if !known || !stomped[held.Category] {
			continue
		}

		if len(out) >= switches {
			break
		}

		at := len(out) + 1
		name := held.Gear()
		label := printed(held.Category, out)
		on := i

		// Round the palette rather than stopping at its end, so a board with
		// more switches than colours repeats the sequence instead of leaving
		// the rest dark. Eight colours against a Helix Floor's ten switches,
		// so the ninth and tenth repeat the first two: a repeat between
		// switches that far apart is worth less than a pair of oranges
		// nobody can tell apart, which is what carrying a ninth colour cost.
		colour := palette[len(out)%len(palette)]

		// The three a device writes on every assignment and this did not.
		//
		// `fieldsOf` has always rendered them and they were never set, so each
		// switch reached the pedal carrying an index, a label and a colour and
		// none of `@fs_primary`, `@fs_enabled` or `@fs_momentary`. A preset
		// HX Edit writes carries all three on all of its assignments, and
		// without them the device stored the section and the switch did
		// nothing.
		//
		// Primary is what the switch shows when more than one block sits on
		// it, and with one block each it is simply true. Enabled is whether
		// the switch acts on the block, which an untouched assignment leaves
		// false the way the pedal does. Momentary latches rather than holding
		// while pressed, which is what a player expects of a pedal they step
		// off.
		primary, enabled, momentary := true, false, false

		out = append(out, rig.Footswitch{
			Switch:    &at,
			Block:     &on,
			Label:     &label,
			Gear:      &name,
			Led:       &colour,
			Primary:   &primary,
			Enabled:   &enabled,
			Momentary: &momentary,
		})
	}

	return out
}
