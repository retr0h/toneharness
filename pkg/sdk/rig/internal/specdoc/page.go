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

// Package specdoc writes the RigSpec grammar page.
package specdoc

import "github.com/retr0h/tonestack/pkg/sdk/internal/specdoc"

// Page is what the RigSpec page says about itself.
//
// Only the sentences that are about this contract. The shape of the page is
// shared with the other one, so the two read as a pair.
var Page = specdoc.Page{
	Root:     "RigSpec",
	Title:    "What a rig resolves to",
	Holds:    "a rig may carry, and what it may say.",
	Contract: "../pkg/sdk/rig/data/rigspec.openapi.yaml",
	By:       "pkg/sdk/rig/internal/specdoc",
	Sits: "A rig is not the document somebody writes. It is what a ToneSpec\n" +
		"resolves to and what a preset is compiled from: exact model\n" +
		"identifiers, in order, deterministic. It is the layer worth sharing,\n" +
		"because two people compiling one get the same preset.",
	Other: specdoc.Link{Name: "what a request may say", At: "tonespec.md"},
	Checked: "Reading a rig checks the closed and shaped fields. Building one\n" +
		"checks the looked-up fields as well, because only then is there a\n" +
		"catalog to check them against.",
}
