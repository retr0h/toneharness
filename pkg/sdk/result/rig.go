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
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// Rigs is every rig under one directory, each with the ask beside it.
type Rigs struct {
	// Dir is where they were read from.
	Dir string `json:"dir"`
	// Rigs are what was found, in the order they were read.
	Rigs []Known `json:"rigs"`
}

// Known is one rig and the ask it answers.
//
// Two documents, because they say two different things. The rig is the gear in
// signal order and why each piece of it is believed to be there; the ask is who
// it is for, how it should sound and how it is played. Held together here
// because every reader of one wants the other in the same breath: a listing
// shows the subject's name from the ask beside the amplifier from the rig.
type Known struct {
	// ID is what the document is called, and the stem of its filename.
	//
	// Here rather than on the rig since version 2. One document holds both
	// halves and owns the name, so neither half carries one that could disagree
	// with the other.
	ID string `json:"id"`
	// Rig is the gear, in order.
	Rig rig.Spec `json:"rig"`
	// Ask is what somebody wanted, where it was written down.
	//
	// Absent is ordinary rather than an error. Somebody's own directory holds
	// rigs they wrote, and nothing obliges them to write down the ask that
	// produced one.
	Ask *tone.Ask `json:"ask"`
}

// Rig is one rig, and what reading it needs that the rig does not carry.
type Rig struct {
	Known
	// Variants are the rigs that say they are a small change on this one.
	//
	// A rig cannot know this about itself. The link points the other way:
	// a variant names what it extends, so only the whole set can answer it,
	// and reading the characteristic rig is where somebody wants it.
	Variants []Variant `json:"variants"`
}

// Variant is a rig that extends another.
type Variant struct {
	// ID is what to ask for to read it.
	ID string `json:"id"`
	// Name is what its subject is called.
	Name string `json:"name"`
}

// Scaffolded is a rig this wrote.
//
// What was written rather than what was asked for. The two are the same today,
// and saying so from the answer rather than from the request is what keeps a
// report honest if they ever stop being.
type Scaffolded struct {
	// ID is what to ask for to read it back.
	ID string `json:"id"`
	// Name is what the subject is called.
	Name string `json:"name"`
	// Instrument is what it is played on.
	Instrument string `json:"instrument"`
	// Amp and Cab are the gear it names. Cab is empty when the amp carries
	// its own, which some models do.
	Amp string `json:"amp"`
	Cab string `json:"cab"`
	// Pedals are what else is in the chain, in order.
	Pedals []string `json:"pedals"`
	// From is the rig this was copied from, by identifier. Empty when the
	// rig was scaffolded from gear names.
	//
	// The two paths differ in what has been checked. Gear names are resolved
	// against the catalog before a scaffold is written; a copy resolves
	// nothing, because the chain is the parent's and no name has changed. A
	// reader of the answer can tell which it holds, rather than a caller
	// having to remember which call it made.
	From string `json:"from"`
	// Path is the file that was written.
	Path string `json:"path"`
}

// Copied says whether the rig came from another rather than from gear names.
func (s Scaffolded) Copied() bool { return s.From != "" }
