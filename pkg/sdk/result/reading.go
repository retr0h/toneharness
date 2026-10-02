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

package result

import (
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// Reading is one preset, read out of a slot or a file.
//
// Both halves are here because a caller asks for different things from the
// same read. Somebody looking at a preset wants the rig, which names gear the
// way a person would. Somebody copying one wants the document the device
// wrote, byte for byte. Producing both and letting the caller pick beats two
// operations that read the same slot twice.
type Reading struct {
	// ID is what the rig this preset describes is called. A document owns the
	// name since version 2, and a preset lifted off a device has no document
	// until something writes one, so it comes back beside the rig.
	ID string `json:"id"`
	// Name is what the preset is called. A slot has one even when it holds
	// nothing, because a device names every slot whether or not anybody has
	// put anything in it.
	Name string `json:"name"`
	// Doc is the preset itself. Nil when the slot holds nothing.
	Doc *preset.Document `json:"doc"`
	// Rig is what the preset describes, as gear a person recognises. Zero when
	// the slot holds nothing, or when only the device's own document was asked
	// for.
	Rig rig.Spec `json:"rig"`
	// Plan is the same preset as the device holds it: which model each piece of
	// gear resolved to, where it sits, and the snapshots and footswitches that
	// only mean anything on a pedal.
	//
	// Beside the rig rather than instead of it, because the two answer different
	// questions about one read. What gear is this is the rig's; what is this
	// pedal actually doing is the plan's.
	Plan plan.Plan `json:"plan"`
	// Answer is what a device replied with when the reply was not a preset.
	// Nil otherwise.
	Answer *Answer `json:"answer"`
}

// Empty says whether the slot holds a preset.
//
// A name is no guide: a device names every slot, so an untouched one still
// answers with whatever it shipped with. Only the document says.
func (r Reading) Empty() bool { return r.Doc == nil }

// DumpEnv names a file to write a device's raw answer to.
//
// The library does not read it. The CLI does, and hands the file to the
// Client as a capture, which is where a renderer's hint to set it comes from.
//
// Reading a preset off the hardware is the one call whose reply nobody has
// seen. Capturing it is what turns a guess about the wire format into a test,
// and it costs one plugged-in session rather than one per attempt.
const DumpEnv = "TONEHARNESS_USB_DUMP"

// Answer is a device's reply that did not decode as a preset.
//
// Kept and reported rather than discarded as a failure. A preset arrives as
// three concatenated MessagePack values and the models inside it are numbered
// by a scheme that does not index the catalog, so a reply nobody can read is
// how a protocol change becomes visible. Saying plainly what arrived beats
// printing a chain that would be wrong.
type Answer struct {
	// Model is what the device calls itself.
	Model string `json:"model"`
	// Slot is the position that was read, from zero.
	Slot int `json:"slot"`
	// Shape is what arrived, described in whatever detail can be had.
	Shape string `json:"shape"`
}

// Written is a file this wrote, and what went into it.
type Written struct {
	// Slot is the position it came from, from zero.
	Slot int `json:"slot"`
	// Name is what the preset is called.
	Name string `json:"name"`
	// Path is the file that was written.
	Path string `json:"path"`
}
