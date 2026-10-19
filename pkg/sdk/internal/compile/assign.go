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
// Unique per switch rather than per kind of block, and ordered so that two
// switches next to each other are never a shade apart. The device offers
// twelve: `auto color`, white, red, dark orange, light orange, yellow, green,
// turquoise, blue, violet, pink and off. Three of those are not colours — auto
// follows the block, and off and white carry no hue — so the eight that do are
// ordered by stepping as far around the hue circle as the remaining choices
// allow, farthest-first: red, turquoise, violet, yellow, green, blue, pink,
// and the oranges last because each sits beside red or yellow.
//
// **This is a visual choice and nothing measured it.** It is not a claim about
// the sound, and anybody who dislikes it can say so in the rig, where a
// footswitch somebody wrote outranks this entirely.
//
// What it gives up is worth naming: `auto` lights by what the block is, so an
// untouched preset shows red for an amp, amber for drive and blue for
// modulation, and a player learns the mapping across every preset they own.
// This trades that for a board where no two switches look alike, which is the
// thing to want when the switches are three feet away and all of them are on.
var palette = []string{
	"red", "turquoise", "violet", "yellow",
	"green", "blue", "pink", "light orange", "dark orange",
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
// The label is the gear's own name and the colour is left unset, which the
// contract calls `auto`: the light then follows the block, so a drive lights
// amber and a reverb blue without this choosing. Both are a person's to change
// and neither is something a build knows better.
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
		on := i

		// Round the palette rather than stopping at its end, so a board with
		// more switches than colours repeats the sequence instead of leaving
		// the rest dark. Nine colours against a Helix Floor's ten switches, so
		// only the tenth ever repeats, and it repeats the one furthest from it.
		colour := palette[len(out)%len(palette)]

		out = append(out, rig.Footswitch{
			Switch: &at,
			Block:  &on,
			Label:  &name,
			Gear:   &name,
			Led:    &colour,
		})
	}

	return out
}
