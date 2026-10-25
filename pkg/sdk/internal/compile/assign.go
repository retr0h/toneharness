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
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// palette is the colours a switch is given, in the order they are handed out.
//
// Unique per switch rather than per kind of block, so a board where every
// switch is on is a board where no two lights look alike.
//
// The device names twelve and the catalog reads them off it in the order it
// numbers them, which is where `auto color` being 0, red 2 and off 11 come
// from. Three carry no hue: `auto` follows the block, and white and off are
// not colours to tell apart. Dark orange and light orange sit 15 degrees
// apart and no player can tell them from each other, so one of them goes:
// two colours that look the same cannot both be distinct, and keeping both
// only guarantees a board with a pair of them somewhere.
//
// **Red, yellow and green are left out, and that is the owner's call rather
// than a measurement.** They are what every pedal on the floor already uses,
// which makes a board of them look like every other board. What is left is
// the part of the circle nothing else claims:
//
//	violet         270 degrees
//	turquoise      174
//	pink           330
//	blue           240
//	light orange    30
//
// Ordered so no two handed out next to each other are close: the smallest gap
// between adjacent entries is 90 degrees, and round the wrap it is 120. An HX
// Stomp shows three at a time, so what matters is the first three and the
// worst pair among them is 96.
//
// **This is a visual choice. The hue figures are read off a colour wheel and
// not off the pedal**, so they rank orderings against each other rather than
// describing what the LEDs emit. Anybody who wants red back says so in the
// rig, where a footswitch somebody wrote outranks this entirely.
//
// What it gives up is worth naming. `auto` lights by what the block is, so an
// untouched preset shows red for an amp, amber for drive and blue for
// modulation, and a player learns that one mapping across every preset they
// own. This trades it for a board that does not look like the shop floor,
// which is the thing to want when they are all yours.
var palette = []string{
	"violet", "turquoise", "pink", "blue", "light orange",
}

// automatic is the colour a switch lights when nobody chose one, by the kind
// of block and whether it is on.
//
// `@fs_ledcolor` beside the chosen colour, and not an index into the device's
// own list: it is a packed 24-bit value, and the off state is the same hue at
// about a sixteenth of the brightness. A drive reads 0xd0af00 on and 0x080600
// off, modulation 0x0032ff and 0x00040d. The write path sent 0, because
// nothing here had the table.
//
// **Mined from the preset corpus rather than guessed or read off a wheel.**
// Across the 4,426 presets in `resources/schemas/corpus/`, 5,819 footswitch
// assignments in these seven categories carry both a colour and a block whose
// state is known, and 99.2% of them agree with the pair below. Per category
// and per state rather than per model: all 313 models that appear resolve to
// one of these, which is why a table of fourteen numbers covers them.
//
//	category   on          off        agreeing
//	drive      0xd0af00    0x080600   94.4% of 501 on, 99.7% of 737 off
//	filter     0x3200ff    0x03000b   100% of 22, 100% of 47
//	mod        0x0032ff    0x00040d   98.7% of 310, 99.5% of 780
//	delay      0x06ff00    0x010900   99.5% of 604, 100% of 843
//	reverb     0xff2d00    0x100400   100% of 819, 98.0% of 249
//	pitch      0x3200ff    0x03000b   100% of 112, 100% of 256
//	eq         0x84ff00    0x080c00   100% of 228, 99.7% of 311
//
// Three pairs repeat, which is the device colouring by what a block does
// rather than by which block it is: a filter, a pitch shifter and a wah share
// one, and so do a compressor, an equaliser and a gate.
//
// **Nothing visible changes today**, which is why task 264 called this not
// urgent. `Assign` always chooses a colour, and a chosen one wins. What it
// buys is a preset that says what the pedal would say: a rig asking for `auto`
// gets the light the block would have lit, and the field stops carrying a zero
// that is not a colour.
//
// Only the seven a switch is given. A utility block never gets one, and its
// readings are the one category that does not settle: 462860 on 58.5% of 573.
var automatic = map[catalog.Category]struct{ on, off int }{
	catalog.CategoryDrive:  {0xd0af00, 0x080600},
	catalog.CategoryFilter: {0x3200ff, 0x03000b},
	catalog.CategoryMod:    {0x0032ff, 0x00040d},
	catalog.CategoryDelay:  {0x06ff00, 0x010900},
	catalog.CategoryReverb: {0xff2d00, 0x100400},
	catalog.CategoryPitch:  {0x3200ff, 0x03000b},
	catalog.CategoryEQ:     {0x84ff00, 0x080c00},
	// Mined the same way and from the same corpus as the seven above, which
	// is how its agreement can be stated beside theirs: on 0x84ff00 for
	// 875 of 888, off 0x080c00 for 264 of 266. The method reproduces every
	// one of the other seven exactly, which is what makes this row worth as
	// much as they are.
	//
	// The same pair as an equaliser, and that is the device's answer rather
	// than a gap in the mining: Line 6 light both green, and a gate reads
	// green too. A switch is told from its neighbour by the colour somebody
	// chose, which `palette` hands out per switch rather than per kind.
	catalog.CategoryComp: {0x84ff00, 0x080c00},
}

// lit is the automatic colour for one block in one state.
//
// Zero for a category nothing measured, which leaves the field out rather than
// writing a number nobody read.
func lit(
	of catalog.Category,
	on bool,
) int {
	held, known := automatic[of]
	if !known {
		return 0
	}

	if on {
		return held.on
	}

	return held.off
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

	// Blocks that arrived bypassed first, then the rest, each in chain order.
	//
	// **A switch belongs to an effect somebody would step on, not to the
	// sound.** A bypassed block is an offer: it is there to be brought in, so
	// a switch is the only way to reach it and is what it was put there for.
	// A running block is part of how the preset sounds, and spending one of
	// three switches on it means an offer goes unreachable.
	//
	// `pizzazz` has argued this from the corpus all along, about the very
	// block that made it obvious: "A compressor is in 85% of chains and
	// running in nearly all of them, so it is the sound rather than an effect;
	// a wah is in 5.7% and running in none, which is a footswitch effect
	// sitting there waiting to be stepped on." Nothing here read that, so a
	// built punk preset spent its middle switch on a compressor that was
	// already on while the reverb beside it had no switch at all.
	//
	// Running blocks are kept as candidates rather than refused, because a
	// chain with nothing bypassed should still be stompable: turning a drive
	// that is on back off is a thing a player does. They just go last.
	ordered := make([]int, 0, len(built.Blocks))

	for i, b := range built.Blocks {
		if !b.Enabled {
			ordered = append(ordered, i)
		}
	}

	for i, b := range built.Blocks {
		if b.Enabled {
			ordered = append(ordered, i)
		}
	}

	// Where in the palette this preset starts.
	//
	// **Rotated by the preset's own name, because the alternative was every
	// preset looking the same.** Walking the palette from nought meant FS1 was
	// the first colour, FS2 the second and FS3 the third on all forty-three
	// rigs on this pedal: three identical lights whatever the preset, which
	// tells a player nothing and was the first thing anybody noticed about
	// them.
	//
	// The order through the palette is kept rather than shuffled, so adjacent
	// switches stay far apart in hue the way the palette was chosen to be.
	// Deterministic, so a rebuild is byte-identical and two people building
	// one rig get the same pedal.
	//
	// A colour is not a claim about the sound, which is what makes an
	// arbitrary choice honest here: nothing downstream reads it and a player
	// reassigns it in seconds.
	start := 0

	if built.Name != "" {
		sum := fnv.New32a()
		_, _ = sum.Write([]byte(built.Name))

		start = int(sum.Sum32() % uint32(len(palette)))
	}

	for _, i := range ordered {
		b := built.Blocks[i]

		// Through the catalog rather than off the plan, because a plan holds a
		// model identifier and the two questions here — what kind of block is
		// this, and what is it called — are both the catalog's.
		held, known := cat.Block(b.Model)
		if !known || !catalog.Stompable(held.Category) {
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
		colour := palette[(start+len(out))%len(palette)]

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
		// **Enabled mirrors the block, and was a constant false.** Task 278.
		//
		// It is the switch's own on-off state, which the pedal applies to the
		// block when the preset loads, so it has to agree with the block's
		// own bypass or the two fight: the first press reconciles them, the
		// second leaves them disagreeing again, and the third settles it.
		// After that the switch works for good, which is what made it look
		// like a decode fault.
		//
		// Read off one HX Stomp with three switches, where the two blocks
		// that arrived bypassed worked from the first press and the one that
		// arrived running took three. Every one of them had `false` here.
		//
		// A constant `true` is not the fix either, and was tried: it turns
		// every block on when the preset loads, which for a block offered
		// bypassed throws away the offer. Mirroring is the only answer that
		// is right for both.
		primary, momentary := true, false
		enabled := b.Enabled

		// The colour the switch would light with nobody choosing, which is
		// the field the device keeps beside the chosen one. See [automatic].
		//
		// The block's own state, which is what it was already being passed
		// now that the two agree, and which is the honest source: this is
		// whether the block is passing signal rather than anything about the
		// switch.
		auto := lit(held.Category, b.Enabled)

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

		// Last, because a category nothing measured leaves it out rather than
		// writing a zero, and a zero in this field is not a colour.
		if auto != 0 {
			out[len(out)-1].Colour = &auto
		}
	}

	return out
}
