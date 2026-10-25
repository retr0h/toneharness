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
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// snapshotPrefix is preset.SnapshotPrefix, which owns the key a snapshot is
// stored under because both this and the editor write the same map with it.
const snapshotPrefix = preset.SnapshotPrefix

// Keys a preset stores a snapshot under. A device owns these names.
const (
	snapName        = "@name"
	snapTempo       = "@tempo"
	snapLED         = "@ledcolor"
	snapPedal       = "@pedalstate"
	snapNamed       = "@custom_name"
	snapValid       = "@valid"
	snapBlocks      = "blocks"
	snapControllers = "controllers"
)

// snapshotsOf reads a preset's snapshots, in the order the device numbers
// them.
//
// Modelled rather than carried as device state, because a snapshot is a
// musical decision: which blocks are on, at what tempo, under what name. What
// somebody put on a footswitch is part of the rig.
func snapshotsOf(
	doc *preset.Document,
) *[]rig.Snapshot {
	keys := make([]string, 0, len(doc.Data.Tone))

	for key := range doc.Data.Tone {
		if strings.HasPrefix(key, snapshotPrefix) && preset.SnapshotIndex(key) >= 0 {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return nil
	}

	sort.Slice(keys, func(i, j int) bool {
		return preset.SnapshotIndex(keys[i]) < preset.SnapshotIndex(keys[j])
	})

	out := make([]rig.Snapshot, 0, len(keys))

	for _, key := range keys {
		out = append(out, snapshotOf(doc.Data.Tone[key]))
	}

	return &out
}

// snapshotOf reads one snapshot entry.
func snapshotOf(
	entry preset.Tone,
) rig.Snapshot {
	var out rig.Snapshot

	decode(entry[snapName], &out.Name)
	decode(entry[snapTempo], &out.Tempo)
	decode(entry[snapLED], &out.Led)
	decode(entry[snapPedal], &out.Pedal)
	decode(entry[snapNamed], &out.Named)
	decode(entry[snapValid], &out.Valid)

	if raw, ok := entry[snapBlocks]; ok {
		out.Blocks = &raw
	}

	if raw, ok := entry[snapControllers]; ok {
		out.Controllers = &raw
	}

	// Whatever this does not model. A handful of presets record commands
	// here, and a format that dropped them would not be lossless the first
	// time Line 6 added a field.
	rest := map[string]json.RawMessage{}

	for key, raw := range entry {
		if !modelled[key] {
			rest[key] = raw
		}
	}

	if len(rest) > 0 {
		out.Rest = &rest
	}

	return out
}

// modelled names the snapshot fields this package reads by name.
var modelled = map[string]bool{
	snapName: true, snapTempo: true, snapLED: true, snapPedal: true,
	snapNamed: true, snapValid: true, snapBlocks: true, snapControllers: true,
}

// restoreSnapshots writes a rig's snapshots back as the device stores them.
//
// Numbered from zero in the order the rig lists them, which is the order they
// were read in. A device names them snapshot0 upward and nothing else refers
// to them by name.
func restoreSnapshots(
	doc *preset.Document,
	snapshots []rig.Snapshot,
) {
	for i, snap := range snapshots {
		entry := preset.Tone{}

		put(entry, snapName, snap.Name)
		put(entry, snapTempo, snap.Tempo)
		put(entry, snapLED, snap.Led)
		put(entry, snapPedal, snap.Pedal)
		put(entry, snapNamed, snap.Named)
		put(entry, snapValid, snap.Valid)

		if snap.Blocks != nil {
			entry[snapBlocks] = *snap.Blocks
		}

		if snap.Controllers != nil {
			entry[snapControllers] = *snap.Controllers
		}

		if snap.Rest != nil {
			for key, raw := range *snap.Rest {
				entry[key] = raw
			}
		}

		doc.Data.Tone[snapshotPrefix+strconv.Itoa(i)] = entry
	}
}

// Trust makes a preset's snapshots ones the device believes, where nothing has
// ever captured one.
//
// A step of its own on the Compiler, beside Sections and Footswitches, because
// the two ways a preset is built do not share an assembly: `presets compile`
// goes through [Lower] and `presets make` builds the document itself and calls
// the steps one at a time. Task 278 was fixed in the first and shipped broken
// in the second, which is exactly what a step reachable from one path and not
// the other buys.
func Trust(
	doc *preset.Document,
	blocks []plan.Block,
) {
	if captured(doc) {
		return
	}

	trustSnapshots(doc, blocks)
}

// trustSnapshots makes the snapshots a preset came with ones the device
// believes, recalling the chain the preset now holds.
//
// **Task 278, and the whole of it.** A preset this tool writes is written into
// an untouched one the device wrote, and an untouched preset's three snapshots
// are blank: `@valid: false` and no `blocks` key at all, because nothing has
// ever been captured into them. Nothing here replaced them, so every preset
// this tool produced carried three snapshots the device does not trust.
//
// What that did on the hardware. Block bypass is snapshot state, so a pedal
// loading a preset whose snapshots are invalid has nothing to recall and
// initialises them from whatever the chain happens to be. The first press of a
// footswitch went into that initialisation and moved nothing audible, the
// second left the pedal's idea of the state and the block's disagreeing, and
// the third settled them. After that the switch worked for good, because by
// then the snapshots were real. It looked like a decode fault for weeks, and
// the preset coming up edited partway through was the snapshot being written.
//
// The pedal's own presets have `@valid: true` on all three, which is what
// finally said where to look: `05C` on this pedal differs from a preset
// written here in exactly this field and in nothing else about its switches.
//
// All of them get the chain as it stands, which is the honest answer rather
// than a musical one: this tool does not know what somebody would want
// snapshot two to recall, and three snapshots that agree are three a player
// can then change. Inventing a difference would be inventing a decision.
//
// Only where the rig brought none of its own. A plan carrying snapshots has
// said what each should recall and [restoreSnapshots] writes those, including
// their own validity.
func trustSnapshots(
	doc *preset.Document,
	blocks []plan.Block,
) {
	paths := map[string]map[string]bool{}

	for _, block := range blocks {
		path := processorKey(block.DSP)
		if paths[path] == nil {
			paths[path] = map[string]bool{}
		}

		// By grid position rather than by the key the block is filed under,
		// which is what a device records and what the two differ by: a preset
		// is free to store block5 at position 6 and a real one does.
		paths[path][blockPrefix+strconv.Itoa(block.Pos)] = block.Enabled
	}

	valid := true

	// The entries the document came with, rather than a count from the
	// limits. How many snapshots a device holds is the device's to say, and
	// the preset underneath was written by one.
	for key, entry := range doc.Data.Tone {
		if preset.SnapshotIndex(key) < 0 {
			continue
		}

		put(entry, snapValid, &valid)
		put(entry, snapBlocks, &paths)

		doc.Data.Tone[key] = entry
	}
}

// captured says whether anything has ever been captured into this preset's
// snapshots.
//
// One valid snapshot is enough. A preset where somebody has set up even a
// single one has made a musical decision, and seeding all three from the
// chain would throw it away: that is the case
// [work-a-device's chain reference] warns about, where a preset that arrived
// with three different sounds comes back holding one.
//
// None valid is the blank an untouched preset ships with, whether it reached
// here as a template nothing replaced or as a plan that lifted those blanks
// and carried them as if they were the rig's own. The second is how
// `presets make` produced an unbelievable preset while `presets compile`
// produced a good one, which is the half of task 278 that a build through one
// command and not the other hid.
func captured(
	doc *preset.Document,
) bool {
	for key, entry := range doc.Data.Tone {
		if preset.SnapshotIndex(key) < 0 {
			continue
		}

		var valid *bool

		decode(entry[snapValid], &valid)

		if valid != nil && *valid {
			return true
		}
	}

	return false
}

// decode reads one of a snapshot's fields, leaving it unset when absent.
//
// A preset omits fields rather than writing nulls, and a field this cannot
// read is left unset for the same reason: writing a zero would claim the
// device said something it did not.
func decode[T any](
	raw json.RawMessage,
	target **T,
) {
	if len(raw) == 0 {
		return
	}

	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return
	}

	*target = &v
}

// put writes a field back, or writes nothing when the rig has none.
func put[T any](
	entry preset.Tone,
	key string,
	v *T,
) {
	if v == nil {
		return
	}

	// A value that came out of JSON goes back into it, so this cannot fail.
	body, _ := json.Marshal(*v)
	entry[key] = body
}
