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
	"fmt"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

const (
	// attrPrefix is what the device spells its own attributes with, as against
	// the controls beside them. `@enabled` is an attribute and `Drive` is a
	// control.
	attrPrefix = "@"
	// attrModel is the attribute naming what a member runs. Lifted out of the
	// attributes into a field of its own, because every member that is a thing
	// rather than a container has one and a reader looking for what this is
	// should not have to know the device's spelling for it.
	attrModel = "@model"
)

// membersOf reads everything a preset holds that is not a block in its chain.
//
// The other two thirds of a preset. A chain says the signal path, and a preset
// holds eighteen more kinds of member beside it: the snapshots, the routing in
// and out of each processor, the split and join, the cabinets a dual block points
// at, the footswitch assignments, the controllers, the global settings, the
// Variax, the impulse response table, the MIDI commands and the amplifier remote.
// Before this they were read off the device and dropped, so importing somebody's
// preset kept their gear and lost what they had built with it.
//
// A processor's own blocks are left out, because the chain already states them
// and two parts of one document setting one control is the thing this avoids.
// Blocks under a footswitch or a snapshot are kept: those are what that switch or
// snapshot does about a block rather than the block itself.
// Infallible, which is a claim about the input rather than optimism. Every value
// here came out of a document preset.Read already parsed, so each one is valid
// JSON: an object, a list, a null, or one of the four scalars, and ParamValue
// reads all four. There is no literal left for a decode to fail on, and threading
// an error for one would be five branches nothing can reach.
func membersOf(
	doc *preset.Document,
) *map[string]rig.PresetMember {
	if len(doc.Data.Tone) == 0 {
		return nil
	}

	out := make(map[string]rig.PresetMember, len(doc.Data.Tone))

	for key, body := range doc.Data.Tone {
		// A value that came out of JSON goes back into it, which is the same
		// argument mustRaw makes a few files over.
		raw, _ := json.Marshal(body)

		out[key] = memberFrom(raw, preset.IsProcessorKey(key))
	}

	return &out
}

// memberFrom reads one member, and whatever is under it.
//
// chainBlocks says this member is a processor, so its own `blockN` entries are
// chain entries and belong to nothing here.
func memberFrom(
	raw json.RawMessage,
	chainBlocks bool,
) rig.PresetMember {
	// Already known to be an object, because shapeOf said so before this was
	// called and the top level is one by the document's own schema.
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)

	var (
		out      rig.PresetMember
		attrs    = map[string]*catalog.Held{}
		controls = map[string]*catalog.Held{}
		members  = map[string]rig.PresetMember{}
	)

	for name, v := range fields {
		if chainBlocks && preset.IsBlockKey(name) {
			continue
		}

		switch shapeOf(v) {
		case shapeObject:
			// Not a processor any more. Only a top-level `dspN` owns chain
			// blocks; a `blocks` or a `dsp0` nested under a snapshot or a
			// footswitch is that member's own record of them.
			members[name] = memberFrom(v, false)
		case shapeList:
			entries := entriesFrom(v)
			members[name] = rig.PresetMember{Entries: &entries}
		case shapeNull:
			// The device's own null, which is the absence of a value rather
			// than a value. A nil carries it, and which of the two maps it
			// goes in is still the name's to say.
			if strings.HasPrefix(name, attrPrefix) {
				attrs[name] = nil
			} else {
				controls[name] = nil
			}
		default:
			// Cannot fail: a scalar out of parsed JSON is a string, a number or
			// a boolean, and ParamValue reads all three. A null went to the case
			// above.
			held, _ := catalog.HoldRaw(v)

			if name == attrModel {
				model := spelled(held)
				out.Model = &model

				continue
			}

			if strings.HasPrefix(name, attrPrefix) {
				attrs[name] = &held
			} else {
				controls[name] = &held
			}
		}
	}

	if len(attrs) > 0 {
		out.Attrs = &attrs
	}

	if len(controls) > 0 {
		out.Controls = &controls
	}

	// Present and empty is kept, because the device holds an empty member in five
	// presets and an absent one is a different preset.
	if len(members) > 0 || emptyObject(raw) {
		out.Members = &members
	}

	return out
}

// entriesFrom reads an ordered list of members.
//
// A null entry is a member with nothing in it. `automation` holds twenty slots
// and names a handful, so the gaps are the lane's shape: writing only the named
// ones would move every assignment after the first gap.
func entriesFrom(
	raw json.RawMessage,
) []rig.PresetMember {
	// Already known to be a list, because shapeOf said so before this was called.
	var held []json.RawMessage
	_ = json.Unmarshal(raw, &held)

	out := make([]rig.PresetMember, 0, len(held))

	for _, v := range held {
		if shapeOf(v) == shapeNull {
			out = append(out, rig.PresetMember{})

			continue
		}

		out = append(out, memberFrom(v, false))
	}

	return out
}

// shape is which of the five things a value in a preset is.
type shape uint8

const (
	// shapeScalar is a number, a boolean or a string.
	shapeScalar shape = iota
	// shapeObject is a member under this one.
	shapeObject
	// shapeList is members the device keeps in order.
	shapeList
	// shapeNull is the device having no value at all.
	shapeNull
)

// shapeOf says which, from the first character of the JSON.
//
// The literal rather than a decode into `any`, because a decode is what loses the
// kind of every number on the way past.
func shapeOf(
	raw json.RawMessage,
) shape {
	trimmed := strings.TrimSpace(string(raw))

	switch {
	case strings.HasPrefix(trimmed, "{"):
		return shapeObject
	case strings.HasPrefix(trimmed, "["):
		return shapeList
	// Nothing at all counts as null, which no parsed document produces and a
	// prefix test has to answer anyway.
	case trimmed == "null", trimmed == "":
		return shapeNull
	default:
		return shapeScalar
	}
}

// emptyObject reports a member the device wrote as `{}`.
func emptyObject(
	raw json.RawMessage,
) bool {
	var fields map[string]json.RawMessage

	return json.Unmarshal(raw, &fields) == nil && len(fields) == 0
}

// spelled is a Held as a string, for the one field that is a name rather than a
// reading.
//
// A model identifier is an enum to [catalog.ParamValue], so it comes back quoted
// and the quotes come off. `@global_params` is the identifier, not `"@global_params"`.
func spelled(
	held catalog.Held,
) string {
	// Cannot fail, for the same reason the read above cannot: this value came
	// from a parsed scalar, so it carries a kind.
	raw, _ := held.Device()

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	// A model stated as something other than a name, which no preset in the
	// corpus does and a document somebody edited could.
	return string(raw)
}

// ApplyMembers writes a rig's record of the preset back into a document.
//
// Exported for the build, which needs it because a rig's `preset:` would
// otherwise be written by a resolve and read by nothing: the document would say
// one thing and the pedal do another, which is worse than the field not existing.
//
// Merged rather than replacing, which is the difference between this and the
// `restore` a lifted plan goes through. A lifted plan names every member the
// preset had, so replacing is right there. A rig's `preset:` names only what
// somebody wrote down, and replacing on that would delete the rest: a rig stating
// one output left the device with no snapshots at all and two sections that no
// longer fit.
func ApplyMembers(
	doc *preset.Document,
	members *map[string]rig.PresetMember,
) error {
	return applyMembers(doc, members)
}

// applyMembers writes a rig's record of the preset back into a document.
//
// Merged into what is there rather than replacing it. A build has already put the
// chain's blocks into each processor, and these are everything beside them, so a
// member the rig states is written over the one the template carried and a member
// it does not mention is left alone.
func applyMembers(
	doc *preset.Document,
	members *map[string]rig.PresetMember,
) error {
	if members == nil {
		return nil
	}

	if doc.Data.Tone == nil {
		doc.Data.Tone = map[string]preset.Tone{}
	}

	for _, key := range sorted(*members) {
		fields, err := deviceFields((*members)[key])
		if err != nil {
			return fmt.Errorf("writing %s: %w", key, err)
		}

		tone := doc.Data.Tone[key]
		if tone == nil {
			tone = preset.Tone{}
		}

		for name, raw := range fields {
			tone[name] = raw
		}

		doc.Data.Tone[key] = tone
	}

	return nil
}

// deviceFields renders one member as the device's own JSON, field by field.
//
// Field by field rather than whole, because a processor's entry holds the chain's
// blocks too and replacing it would take them out.
func deviceFields(
	of rig.PresetMember,
) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}

	if of.Model != nil {
		// A string always marshals, so this is the one place here that cannot
		// refuse what it was given.
		raw, _ := json.Marshal(*of.Model)
		out[attrModel] = raw
	}

	for _, held := range []*map[string]*catalog.Held{of.Attrs, of.Controls} {
		if held == nil {
			continue
		}

		for name, v := range *held {
			raw, err := deviceValue(v)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}

			out[name] = raw
		}
	}

	if of.Members == nil {
		return out, nil
	}

	for name, under := range *of.Members {
		raw, err := deviceMember(under)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}

		out[name] = raw
	}

	return out, nil
}

// deviceValue is one value as the device writes it, null included.
func deviceValue(
	of *catalog.Held,
) (json.RawMessage, error) {
	if of == nil {
		return json.RawMessage("null"), nil
	}

	return of.Device()
}

// deviceMember renders a member nested under another, whole.
//
// A list where the member carries entries and an object otherwise, which is the
// shape the device wrote and the shape it expects back.
func deviceMember(
	of rig.PresetMember,
) (json.RawMessage, error) {
	if of.Entries != nil {
		out := make([]json.RawMessage, 0, len(*of.Entries))

		for i, entry := range *of.Entries {
			raw, err := deviceEntry(entry)
			if err != nil {
				return nil, fmt.Errorf("entry %d: %w", i, err)
			}

			out = append(out, raw)
		}

		return json.Marshal(out)
	}

	fields, err := deviceFields(of)
	if err != nil {
		return nil, err
	}

	return json.Marshal(fields)
}

// deviceEntry renders one entry of an ordered list.
//
// A member with nothing in it is the device's null, which is what a lane's
// unassigned slot is. Of 3,234 list entries in the preset corpus 2,136 are null
// and not one is an empty object, so there is no third state to get wrong.
func deviceEntry(
	of rig.PresetMember,
) (json.RawMessage, error) {
	if empty(of) {
		return json.RawMessage("null"), nil
	}

	return deviceMember(of)
}

// empty reports a member saying nothing at all.
func empty(
	of rig.PresetMember,
) bool {
	return of.Model == nil && of.Attrs == nil &&
		of.Controls == nil && of.Members == nil && of.Entries == nil
}
