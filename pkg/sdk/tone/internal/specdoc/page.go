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

// Package specdoc writes the page describing what a request may say.
package specdoc

import "github.com/retr0h/tonestack/pkg/sdk/internal/specdoc"

// Page is what the ToneSpec page says about itself.
//
// Only the sentences that are about these contracts. The shape of the page is
// shared with the other one, so the two read as a pair.
var Page = specdoc.Page{
	Root:     "ToneSpec",
	Title:    "What a request may say",
	Holds:    "a ToneSpec and a Setup may carry.",
	Contract: "../pkg/sdk/tone/data/tonespec.openapi.yaml",
	By:       "pkg/sdk/tone/internal/specdoc",
	Sits: "These are the two documents a person writes, and neither carries a\n" +
		"knob position. A ToneSpec is the ask and changes every request; a\n" +
		"Setup is what somebody owns and changes when they buy something.\n" +
		"Every field is optional and any combination is legal: \"punk, but on\n" +
		"my Jazz\" is a genre and a Setup, \"like Dirnt but chunkier\" is a\n" +
		"player and a nudge, and a request nobody filled in completely is the\n" +
		"ordinary case.",
	Other: specdoc.Link{Name: "what a rig resolves to", At: "rigspec.md"},
	Checked: "Nothing here is checked against a catalog. A request names gear the\n" +
		"way a person does, and which device it will be built for is a\n" +
		"separate question; that check happens when the rig is compiled.",
}
