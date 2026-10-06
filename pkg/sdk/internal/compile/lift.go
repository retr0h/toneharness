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
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// Lift reads a preset into a rig.
//
// Every block records both the gear it emulates and the exact model it was,
// keyed by device. The name alone cannot identify a model — 661 of them share
// 468 names — so a rig that only carried the name would rebuild into a
// different preset.
// The identifier comes back beside the rig rather than on it. A rig is a section
// of a document since version 2 and the document owns the name, so lifting one
// off a preset answers with both and the caller assembles them.
func Lift(
	doc *preset.Document,
	cat *catalog.Catalog,
) (string, rig.Spec, plan.Plan, error) {
	c, err := doc.Spec()
	if err != nil {
		return "", rig.Spec{}, plan.Plan{}, fmt.Errorf("reading the chain: %w", err)
	}

	device := cat.Device

	entries := make([]rig.ChainEntry, 0, len(c.Blocks))

	for _, b := range c.Blocks {
		entries = append(entries, entryFor(b, cat))
	}

	snapshots := snapshotsOf(doc)
	switches := footswitchesOf(doc, cat)
	movers := controllersOf(doc, cat)

	id := slug.Of(doc.Data.Meta.Name)

	out := rig.Spec{
		Chain:      entries,
		Instrument: instrumentFieldFor(c, cat),
	}

	// The same preset read twice, into the two documents it is. The rig is the
	// gear in signal order, which reads the same on hardware nobody has written
	// a driver for; the plan is everything that only means anything on this one.
	//
	// Both, rather than one and a conversion, because they come out of one
	// decode and a caller wanting the portable half should not have to throw the
	// other away and read the file again to get it back.
	made := plan.Plan{
		Name:         doc.Data.Meta.Name,
		Rig:          id,
		Blocks:       c.Blocks,
		Snapshots:    deref(snapshots),
		Footswitches: deref(switches),
		Controllers:  deref(movers),
		Target:       &rig.Target{Device: &device},
		Device:       deviceState(doc, modelledKeys(doc, snapshots, switches, movers)),
	}

	// A rig this package produced must be one anybody else can read. Lifting
	// something that does not meet its own contract is a bug here, not input
	// worth passing on.
	if err := rig.Validate(out); err != nil {
		return "", rig.Spec{}, plan.Plan{}, fmt.Errorf(
			"lifting %q: %w", doc.Data.Meta.Name, err)
	}

	return id, out, made, nil
}

// deref reads an optional list as a list, since absent and empty are the same
// thing to a plan: nothing to write.
func deref[T any](
	of *[]T,
) []T {
	if of == nil {
		return nil
	}

	return *of
}

// entryFor describes one block as gear.
//
// The gear and the role, and nothing about the device. Which model answered, the
// parameters it was set to and where it sat are the plan's, and the plan gets
// them from the blocks this same read produced rather than from a copy here.
func entryFor(
	b plan.Block,
	cat *catalog.Catalog,
) rig.ChainEntry {
	blk, known := cat.Block(b.Model)

	out := rig.ChainEntry{
		Gear: gearName(blk, b.Model, known),
		Role: roleFor(blk.Category, known),
	}

	// Every control the block is set to, which is what makes a lifted rig the
	// specification of the preset rather than a sketch of it. Before this a
	// lifted rig named the gear and dropped every value, so a mic moved on the
	// pedal came back as nothing at all.
	if len(b.Params) > 0 {
		held := make(map[string]catalog.Setting, len(b.Params))
		for name, v := range b.Params {
			held[name] = catalog.Set(v)
		}

		out.Controls = &held
	}

	if b.DSP > 0 {
		out.Dsp = &b.DSP
	}

	if !b.Enabled {
		off := b.Enabled
		out.Enabled = &off
	}

	return out
}

// gearName describes a block the way a person would.
//
// The gear it emulates when Line 6 say what that is, and the model's own name
// otherwise — every Line 6 original reads "Line 6 Original", which names
// nothing.
func gearName(
	blk catalog.Block,
	id catalog.ModelID,
	known bool,
) string {
	if !known {
		return string(id)
	}

	if blk.BasedOn != "" && blk.BasedOn != "Line 6 Original" {
		return blk.BasedOn
	}

	if blk.Name != "" {
		return blk.Name
	}

	return string(id)
}

// roleFor maps a catalog category onto the rig vocabulary.
//
// A model the catalog has never heard of is described as other rather than
// left blank: a rig has to say what every block is, and "something this
// device carries and we do not recognise" is a truthful answer.
func roleFor(
	c catalog.Category,
	known bool,
) rig.Role {
	if !known {
		return rig.RoleOther
	}

	if role, ok := roles[c]; ok {
		return role
	}

	return rig.RoleOther
}

// roles is the correspondence between what the catalog calls a block and what
// a rig calls it. They are deliberately the same words.
var roles = map[catalog.Category]rig.Role{
	catalog.CategoryAmp:     rig.RoleAmp,
	catalog.CategoryCab:     rig.RoleCab,
	catalog.CategoryDrive:   rig.RoleDrive,
	catalog.CategoryComp:    rig.RoleComp,
	catalog.CategoryGate:    rig.RoleGate,
	catalog.CategoryEQ:      rig.RoleEQ,
	catalog.CategoryMod:     rig.RoleMod,
	catalog.CategoryDelay:   rig.RoleDelay,
	catalog.CategoryReverb:  rig.RoleReverb,
	catalog.CategoryWah:     rig.RoleWah,
	catalog.CategoryPitch:   rig.RolePitch,
	catalog.CategoryFilter:  rig.RoleFilter,
	catalog.CategoryUtility: rig.RoleUtility,
	catalog.CategoryOther:   rig.RoleOther,
}

// instrumentFieldFor reports which instrument a chain is for, from its amplifier.
//
// Line 6 tag amps Guitar or Bass. A chain with no amp names no instrument, so
// guitar stands as the more common default.
func instrumentFieldFor(
	c plan.Plan,
	cat *catalog.Catalog,
) rig.Instrument {
	// Guitar for a chain with no amplifier, because a rig's `instrument` field
	// has to hold something. That is this caller's decision rather than the
	// rule's, which is why plan.InstrumentFor answers NoAmp and leaves it.
	if plan.InstrumentFor(c, cat) == plan.Bass {
		return rig.InstrumentBass
	}

	return rig.InstrumentGuitar
}

// modelledKeys names the tone entries a rig carries as fields of its own.
//
// Only the ones it actually carried: a preset can hold an empty footswitch
// section, which produces no fields and is still something the file said.
func modelledKeys(
	doc *preset.Document,
	snapshots *[]rig.Snapshot,
	switches *[]rig.Footswitch,
	movers *[]rig.Controller,
) map[string]bool {
	out := map[string]bool{}

	if snapshots != nil {
		for key := range doc.Data.Tone {
			if preset.SnapshotIndex(key) >= 0 {
				out[key] = true
			}
		}
	}

	if switches != nil {
		out[footswitchKey] = true
	}

	if movers != nil {
		out[controllerKey] = true
	}

	return out
}
