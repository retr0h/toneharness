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

import "strings"

// lineSixOriginal is what BasedOn reads for a model that emulates nothing. It
// names no gear, so it is never an answer to what a block is.
const lineSixOriginal = "Line 6 Original"

// Gear describes the block the way a person would: the real-world gear it
// emulates where Line 6 say what that is, and the model's own name otherwise.
//
// This is the only string a rig may name a block by. A model name is one
// manufacturer's internal label for somebody else's amplifier, it changes
// between releases, and it does not survive being read on other hardware,
// which is why the contract's `gear` field refuses one.
func (b Block) Gear() string {
	if b.BasedOn != "" && b.BasedOn != lineSixOriginal {
		return said(withoutCollection(b.BasedOn))
	}

	return said(b.Name)
}

// withoutCollection drops Line 6's own shelf label from the front of a gear
// name.
//
// "Legacy" is where Line 6 keep an older port of a model, not something a
// maker ever put on a cabinet, and it has no business in a document's `gear`.
// It also pinned every rig that carried it: a name beginning with it reaches
// the legacy model and nothing else, so a derived document naming a
// `Legacy 6x10" MESA/Boogie Power House` could never resolve to the mic'd
// 6x10 of the same cabinet, which carries Mic, Position and Angle and costs
// 2.5% of the chip against 7.2%.
//
// Only off the front, and only this word. A document somebody wrote naming
// the collection still means it and still gets that model: this changes what
// a derived document says the gear is, not what an existing one asks for.
func withoutCollection(
	s string,
) string {
	return strings.TrimPrefix(s, collection+" ")
}

// collection is Line 6's grouping for a model kept for compatibility rather
// than because it is the best one of its gear on the device.
const collection = "Legacy"

// said is the spelling without the marks a manufacturer puts in and nobody
// says out loud.
//
// Line 6's own data is not consistent about them: "Ampeg SVT® (normal
// channel)" and "Ampeg SVT (bright channel)" are two spellings of one
// amplifier's two channels. Documents written from either read the same way
// afterwards.
func said(
	s string,
) string {
	return strings.Join(strings.Fields(noise.Replace(s)), " ")
}
