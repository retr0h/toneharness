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

package editor

import (
	"encoding/json"
	"strconv"

	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// Keys a preset stores a footswitch assignment under. A device owns these
// names, and HX Edit's own export of a slot is where each of them was read.
const (
	footswitchKey = "footswitch"
	fsIndex       = "@fs_index"
	fsLabel       = "@fs_label"
	fsCustomLabel = "@fs_customlabel"
	fsCustomColor = "@fs_customcolor"
	fsLEDColor    = "@fs_ledcolor"
	fsEnabled     = "@fs_enabled"
	fsMomentary   = "@fs_momentary"
	fsPrimary     = "@fs_primary"
)

// footswitchesInto writes what each of a device's switches works on into a
// preset.
//
// Task 262. The reader found them and this did not exist, so
// `slots export --as hlx` wrote `"footswitch": null` for a slot with three
// working switches on it. Export a preset, re-import it, and every assignment
// was gone: a round trip through this tool silently discarded somebody's own
// decisions about their own pedal.
//
// **Keyed by the entry a block is stored under, not by the grid position the
// device records it at.** The two differ by [wire.GridOffset], and this is
// written alongside a chain written the same way. HX Edit's export of one of
// this pedal's presets calls them `block0`, `block1` and `block5` for a chain
// of six where the wire says 1, 3 and 14.
//
// Every field the device stores, because the format this writes is the
// faithful copy. A label somebody chose and the block's own name are recorded
// separately, so a file that carried only the resolved one could not say
// whether anybody had chosen anything, and `@fs_ledcolor` is kept whole rather
// than guessed at: two drives on one preset share 525824 and an octaver reads
// 196619, and nothing here has a table of them.
//
// Into whatever the template holds rather than over it, the way
// [snapshotsInto] does, so a field nothing here reads survives the trip.
//
// A block the chain does not hold is still written. The device put the
// assignment there and a faithful copy says so; the import side is where an
// assignment naming nothing is dropped, because that is where it would reach a
// pedal.
func footswitchesInto(
	doc *preset.Document,
	got wire.DevicePreset,
) {
	if len(got.Footswitches) == 0 {
		return
	}

	at := map[int]int{}
	for i, b := range got.Blocks {
		at[b.Index] = i
	}

	blocks := map[string]any{}

	for _, fs := range got.Footswitches {
		slot, held := at[fs.Block]
		if !held {
			continue
		}

		fields := map[string]any{
			fsIndex:     fs.Switch,
			fsLabel:     fs.Gear,
			fsLEDColor:  fs.Auto,
			fsEnabled:   fs.Enabled,
			fsMomentary: fs.Momentary,
			fsPrimary:   fs.Primary,
		}

		// Only where somebody chose. HX Edit leaves both out on a switch
		// carrying the block's own name and colour, so writing a zero and an
		// empty string would be this file claiming two decisions nobody made.
		if fs.Custom != "" && fs.Custom != fs.Gear {
			fields[fsCustomLabel] = fs.Custom
		}

		if fs.LED != 0 {
			fields[fsCustomColor] = fs.LED
		}

		blocks[blockPrefix+strconv.Itoa(slot)] = fields
	}

	if len(blocks) == 0 {
		return
	}

	entry := doc.Data.Tone[footswitchKey]
	if entry == nil {
		entry = preset.Tone{}
	}

	// Values that came off the wire, and maps of them, always marshal.
	body, _ := json.Marshal(blocks)
	entry[processorKey] = body

	doc.Data.Tone[footswitchKey] = entry
}
