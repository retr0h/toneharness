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

package catalog

import (
	"encoding/json"
)

// Held is one value a preset holds, written down the way a document can keep it.
//
// [Setting] with one more state, and it is a state thousands of real presets are
// in. An unassigned impulse response slot is the empty string, and `irUuidTable`
// holds 128 slots in every preset that carries one: most are empty and a few name
// a file. Setting refuses an empty string on purpose, because a control with no
// value is not a reading and writing one would put it in a preset. That argument
// is about a control somebody is setting. This is a transcription of what a
// device already holds, where empty is what it holds.
//
// The device's own `null` is a third thing, and it is the absence of a Held
// rather than a state of one: a map of these carries pointers, and a nil one is
// the null. `@cursor_path` and `@cursor_position` are null in three presets in
// the corpus, and one spelling per state is what stops a writer having to choose
// between two that mean the same.
//
// The zero value is the empty string, which is the safe direction: a value that
// arrived from nowhere writes back as the thing a device already treats as
// nothing assigned.
type Held struct {
	// value is the reading, where there is one.
	value ParamValue
	// blank is the device's own empty string, which no ParamValue can be.
	blank bool
}

// Hold is a reading ready to write down.
func Hold(
	of ParamValue,
) Held {
	return Held{value: of}
}

// Blank is the empty string a device writes for a slot nothing is assigned to.
func Blank() Held { return Held{blank: true} }

// Value is the reading and whether there is one.
//
// A blank answers false, because the empty string is not a number, a switch or a
// named position, and a caller treating it as a reading would be reading the zero
// value of whichever kind it guessed.
func (h Held) Value() (ParamValue, bool) {
	return h.value, !h.blank
}

// IsBlank reports the device's empty string.
func (h Held) IsBlank() bool { return h.blank }

// MarshalJSON writes the state the value is in.
//
// A reading goes out as the string its kind spells, which is [Setting]'s own
// rule and reached through it rather than restated.
func (h Held) MarshalJSON() ([]byte, error) {
	if h.blank {
		return []byte(`""`), nil
	}

	return Set(h.value).MarshalJSON()
}

// UnmarshalJSON reads whichever state the document spells.
func (h *Held) UnmarshalJSON(
	b []byte,
) error {
	var text string
	if err := json.Unmarshal(b, &text); err == nil && text == "" {
		*h = Blank()

		return nil
	}

	var got Setting
	if err := got.UnmarshalJSON(b); err != nil {
		return err
	}

	*h = Hold(got.ParamValue)

	return nil
}

// Device is the JSON a device writes for this value.
//
// Not the document's spelling: a document keeps a reading quoted so its kind
// survives being read back, and a preset keeps it bare. `"6.0"` here is `6.0`
// there, and writing the quoted form into a preset would hand the device a
// string where it wants a number.
func (h Held) Device() ([]byte, error) {
	if h.blank {
		return []byte(`""`), nil
	}

	return h.value.MarshalJSON()
}

// HoldRaw reads one value as a device wrote it.
//
// The other direction from Device, and the one a lift takes: a preset's own JSON
// in, a Held out. Unlike UnmarshalJSON this reads the bare literal rather than a
// document's quoted spelling, so `6.0` stays a float rather than becoming the
// named position "6.0".
//
// A device's `null` is not read here, because a nil pointer is what carries it
// and a caller holding a value has already decided there is one.
func HoldRaw(
	raw []byte,
) (Held, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil && text == "" {
		return Blank(), nil
	}

	var got ParamValue
	if err := got.UnmarshalJSON(raw); err != nil {
		return Held{}, err
	}

	return Hold(got), nil
}
