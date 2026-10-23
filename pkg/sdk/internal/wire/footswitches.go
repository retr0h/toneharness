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

package wire

import "github.com/vmihailenco/msgpack/v5"

// The keys an assignment and its body carry, beyond the ones preset.go names.
//
// Read off a device rather than guessed. `05C` on an HX Stomp carries three
// switches and its footswitch section decodes to the layout below, which the
// same preset exported from HX Edit agrees with field for field: body 8 is
// 14 for a block HX Edit calls `block5`, and body 6 is 525824 where HX Edit
// writes `@fs_ledcolor: 525824`.
//
// Two of them carry the same value on every assignment any device has written
// here and nothing says what they mean, so they are written as they were found.
// Naming them something they might not be would be worse than naming them
// after where they sit.
const (
	keyFsOrdinal   = 10
	keyFsMomentary = 12
	keyFsPrimary   = 13
	keyFsCustom    = 14
	keyFsAlways    = 15
	keyFsBodyOne   = 0
	keyFsBodyTwo   = 2
	keyFsAuto      = 6
	keyFsEnabled   = 7
)

// PlaceFootswitches writes what each switch on the pedal works on.
//
// The one thing a preset this project built never carried. `footswitchesOf`
// has always read them back and nothing wrote them, so a chain reached the
// pedal with its blocks in the right order at the right values and no switch
// turning any of them off. A preset written here and then exported from HX
// Edit had no footswitch section at all.
//
// **A switch names a block by its grid position, not by its place in the
// chain.** That is the whole trap: HX Edit's own file calls them `block0`,
// `block1` and `block5` for a chain of six, and the wire for the same preset
// says 1, 3 and 14. `Placement.Position` is the number to use and says so.
//
// An unused slot holds the section with its switches empty: five groups of
// nothing, where a Stomp has three switches and a Helix Floor ten. So this
// replaces a group rather than making one, and how many groups there are stays
// the device's to say.
//
// **Every group the chain does not name is emptied.** A preset compiled from a
// template inherits that template's whole footswitch section, and leaving an
// unnamed group alone kept whatever the template had on it: a chain with one
// drive on FS1 reached the pedal with a second switch carrying an empty name
// and pointing at a block the new chain never placed. A preset says what it
// holds and nothing it does not, the way [place] already empties every grid
// position the chain leaves out.
//
// Nothing here fails. A preset with no footswitch section has nothing to write
// into and a switch past what the device sent has nowhere to go, and both are
// left as they are: this assigns no block and moves no control, so there is no
// claim about the sound to get wrong.
func PlaceFootswitches(
	doc *Document,
	switches []DeviceFootswitch,
) {
	if len(switches) == 0 {
		return
	}

	body, ok := doc.Section(int8(keyFootswitch))
	if !ok {
		return
	}

	// By switch, because the wire holds one group per switch and a group holds
	// every assignment on it. Walking the assignments instead would write the
	// last one on a shared switch over the others.
	grouped := map[int][]DeviceFootswitch{}
	for _, fs := range switches {
		grouped[fs.Switch] = append(grouped[fs.Switch], fs)
	}

	// Every group the device sent, not only the ones a switch names, so a
	// template's assignments cannot outlive the chain they described. Walking
	// until a group is missing rather than to a count from the limits: the
	// section itself says how many there are, and a device that sends a
	// different number is answered on its own terms.
	for at := 0; ; at++ {
		if _, _, err := locate(body, path{keyFsPaths, at}); err != nil {
			break
		}

		// FS1 is 1 the way the pedal prints it, and the groups count from zero.
		body = placeGroup(body, at, grouped[at+1])
	}

	doc.SetSection(int8(keyFootswitch), body)
}

// placeGroup writes one switch's assignments over whatever that group held.
// An empty `on` empties the group, which is how a switch the chain does not
// name arrives unassigned rather than carrying what a template left there.
func placeGroup(
	body []byte,
	at int,
	on []DeviceFootswitch,
) []byte {
	if at < 0 {
		return body
	}

	start, end, err := locate(body, path{keyFsPaths, at})
	if err != nil {
		return body
	}

	entries := make([]any, 0, len(on))

	for _, fs := range on {
		// A device terminates its strings, and reads one that is not
		// terminated as running on into whatever follows it.
		name := fs.Gear + "\x00"

		label := fs.Label
		if label == "" {
			label = fs.Gear
		}

		// `Auto`, `Enabled` and `Momentary` are carried rather than
		// flattened, which is what makes a round trip through an export and
		// back keep what the device had: task 262. Each was written as its
		// zero before, and a preset this tool compiled still carries the zero
		// because it has nothing else to carry.
		//
		// `Primary` is not among them. It stays true for the reason it always
		// was: a switch this tool writes carries one assignment, so it is the
		// one the pedal prints the label of, and reading it off a struct whose
		// zero value is false would silently label nothing.
		//
		// Nor is the custom label, which is written whether or not anybody
		// chose one. That is what makes the pedal print a name at all, and
		// task 263 is the question of what it should say.
		entries = append(entries, map[int]any{
			int(keyFsOrdinal):   0,
			int(keyFsMomentary): fs.Momentary,
			int(keyFsPrimary):   true,
			int(keyFsCustom):    label + "\x00",
			int(keyFsAlways):    true,
			int(keyFsLED):       fs.LED,
			int(keyFsBody): map[int]any{
				int(keyFsBodyOne): 1,
				int(keyFsBodyTwo): 0,
				int(keyFsModel):   name,
				int(keyFsAuto):    fs.Auto,
				int(keyFsEnabled): fs.Enabled,
				int(keyFsBlock):   fs.Block,
			},
		})
	}

	// Keys are int8 and values are strings, integers and booleans, which this
	// encoder takes, so the only way to fail is a type not used above.
	raw, err := msgpack.Marshal(entries)
	if err != nil {
		return body
	}

	return replaceSpan(body, start, end, raw)
}
