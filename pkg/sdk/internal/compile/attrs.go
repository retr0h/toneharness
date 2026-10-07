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

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// The device's own names for what a chain entry says about a block, beside its
// gear and its controls.
const (
	attrPath         = "@path"
	attrStereo       = "@stereo"
	attrTrails       = "@trails"
	attrBypassVolume = "@bypassvolume"
	attrKeepOnSnap   = "@no_snapshot_bypass"
	attrMic          = "@mic"
	attrCab          = "@cab"
	attrIR           = "@uuid"
	// The two the preset writer puts back from the block itself.
	attrPosition = "@position"
	attrEnabled  = "@enabled"
)

// blockAttr is one device attribute and the chain entry field that states it.
//
// One table read both ways, which is the point of it. A lift fills the field from
// the attribute and a build writes the attribute from the field, so the two cannot
// drift: nine of these fields were in the contract, written into documents by the
// worked example, and read by nothing at all. A preset exported and built again
// came back with its parallel path gone, its stereo blocks mono, its reverb tails
// cut and three bypassed blocks switched on.
type blockAttr struct {
	// name is what the device calls it.
	name string
	// out reads the attribute off a block and puts it on the entry.
	out func(raw json.RawMessage, to *rig.ChainEntry)
	// in reads the field off an entry and returns the attribute, or false where
	// the entry does not state it.
	in func(from rig.ChainEntry) (json.RawMessage, bool)
}

// blockAttrs is every attribute a chain entry can state.
//
// `@position` and `@enabled` are not here. A position is where the block sits and
// the plan carries it as a field of its own, and `@enabled` is written from
// `Block.Enabled` by the preset writer rather than from the attributes beside it.
//
// `@type` and `@favorite` are not here either, nor `@input` and `@output` on the
// one block each that carries them. They have no field and no meaning anybody has
// established, so they travel in `attrs` as the device wrote them.
func blockAttrs() []blockAttr {
	return []blockAttr{
		{
			name: attrPath,
			out:  func(raw json.RawMessage, to *rig.ChainEntry) { to.Path = intFrom(raw) },
			in:   func(from rig.ChainEntry) (json.RawMessage, bool) { return numeric(from.Path) },
		},
		{
			name: attrStereo,
			out:  func(raw json.RawMessage, to *rig.ChainEntry) { to.Stereo = boolFrom(raw) },
			in:   func(from rig.ChainEntry) (json.RawMessage, bool) { return boolean(from.Stereo) },
		},
		{
			name: attrTrails,
			out:  func(raw json.RawMessage, to *rig.ChainEntry) { to.Trails = boolFrom(raw) },
			in:   func(from rig.ChainEntry) (json.RawMessage, bool) { return boolean(from.Trails) },
		},
		{
			name: attrKeepOnSnap,
			out: func(raw json.RawMessage, to *rig.ChainEntry) {
				to.KeepOnSnapshot = boolFrom(raw)
			},
			in: func(from rig.ChainEntry) (json.RawMessage, bool) {
				return boolean(from.KeepOnSnapshot)
			},
		},
		{
			name: attrBypassVolume,
			out: func(raw json.RawMessage, to *rig.ChainEntry) {
				to.BypassVolume = floatFrom(raw)
			},
			in: func(from rig.ChainEntry) (json.RawMessage, bool) {
				return numeric(from.BypassVolume)
			},
		},
		{
			name: attrMic,
			out:  func(raw json.RawMessage, to *rig.ChainEntry) { to.Mic = intFrom(raw) },
			in:   func(from rig.ChainEntry) (json.RawMessage, bool) { return numeric(from.Mic) },
		},
		{
			name: attrCab,
			out:  func(raw json.RawMessage, to *rig.ChainEntry) { to.Cab = stringFrom(raw) },
			in:   func(from rig.ChainEntry) (json.RawMessage, bool) { return text(from.Cab) },
		},
		{
			name: attrIR,
			out:  func(raw json.RawMessage, to *rig.ChainEntry) { to.Ir = stringFrom(raw) },
			in:   func(from rig.ChainEntry) (json.RawMessage, bool) { return text(from.Ir) },
		},
	}
}

// attrsOnto puts a block's device attributes onto the chain entry describing it.
//
// Whatever has a field of its own goes in the field, and the rest goes in `attrs`
// as it arrived. Nothing is dropped, which is the whole point: a preset carries 15
// distinct block attributes and a chain entry had fields for eight of them.
func attrsOnto(
	b plan.Block,
	to *rig.ChainEntry,
) {
	if len(b.Attrs) == 0 {
		return
	}

	named := map[string]bool{}

	for _, a := range blockAttrs() {
		named[a.name] = true

		if raw, ok := b.Attrs[a.name]; ok {
			a.out(raw, to)
		}
	}

	rest := map[string]*catalog.Held{}

	for name, raw := range b.Attrs {
		// A position is the plan's own field and the chain's order, and the
		// writer puts `@enabled` back from the block rather than from here.
		if named[name] || name == attrPosition || name == attrEnabled {
			continue
		}

		// A null is the device having no value, which a nil carries. HoldRaw
		// refuses one rather than reading it as the empty string, so the two
		// cannot be confused here.
		held, err := catalog.HoldRaw(raw)
		if err != nil {
			if shapeOf(raw) == shapeNull {
				rest[name] = nil
			}

			continue
		}

		rest[name] = &held
	}

	if len(rest) > 0 {
		to.Attrs = &rest
	}
}

// attrsFrom is what a chain entry says a block's device attributes are.
//
// The other direction, off the same table. A build that did not write these back
// would hand the device a preset missing the parallel path, the stereo flags and
// the reverb tails the document states.
func attrsFrom(
	entry rig.ChainEntry,
) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}

	for _, a := range blockAttrs() {
		if raw, ok := a.in(entry); ok {
			out[a.name] = raw
		}
	}

	// Everything the device wrote that no field claims, back as it arrived.
	if entry.Attrs != nil {
		for name, held := range *entry.Attrs {
			// A nil is the device's own null, which it wrote and takes back.
			if held == nil {
				out[name] = json.RawMessage("null")

				continue
			}

			// Ignored rather than wrapped, because nothing can reach it. A Held
			// only arrives through a document, and reading one either produces a
			// value with a kind or refuses the literal outright.
			raw, _ := held.Device()
			out[name] = raw
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// The readers and writers for the four kinds of attribute a block carries.
//
// Each returns nil rather than a zero where the attribute is not a value of that
// kind, because a `@path` that is not a number is a preset this tool did not write
// and guessing zero would move the block to the first path.
func intFrom(
	raw json.RawMessage,
) *int {
	var got int
	if err := json.Unmarshal(raw, &got); err != nil {
		return nil
	}

	return &got
}

func boolFrom(
	raw json.RawMessage,
) *bool {
	var got bool
	if err := json.Unmarshal(raw, &got); err != nil {
		return nil
	}

	return &got
}

func floatFrom(
	raw json.RawMessage,
) *float64 {
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		return nil
	}

	return &got
}

func stringFrom(
	raw json.RawMessage,
) *string {
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		return nil
	}

	return &got
}

// numeric, boolean and text render a field back, and say nothing where the
// document said nothing.
func numeric[T int | float64](
	of *T,
) (json.RawMessage, bool) {
	if of == nil {
		return nil, false
	}

	raw, err := json.Marshal(*of)

	return raw, err == nil
}

func boolean(
	of *bool,
) (json.RawMessage, bool) {
	if of == nil {
		return nil, false
	}

	raw, _ := json.Marshal(*of)

	return raw, true
}

func text(
	of *string,
) (json.RawMessage, bool) {
	if of == nil {
		return nil, false
	}

	raw, _ := json.Marshal(*of)

	return raw, true
}
