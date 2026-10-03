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
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// Resolved is what a request and a setup turned into.
//
// Both halves, because a run answering only with the rig would hide what it
// could not honour, and the notes are how somebody finds out that the
// amplifier they asked for is not one the device models. An agent reading this
// over MCP needs that as much as a person reading a terminal, which is why it
// travels in the answer rather than being printed above it.
type Resolved struct {
	// ID is what the document the request came in is called, carried through so
	// the answer can be written back into one.
	ID string `json:"id"`
	// Ask is the request this resolved, carried through so the answer can be
	// written back into the document it came from.
	Ask *tone.Ask `json:"ask,omitempty"`
	// Rig is the gear the request resolved to.
	Rig rig.Spec `json:"rig"`
	// Notes are what the translation made of the request, honoured or not.
	Notes translate.Notes `json:"notes"`
}
