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

package plan

// ValidateHolds reports the first block in s the device will not load.
//
// The catalog says what a device models and the hardware says what it will
// hold, and on one family they disagree. Line 6's files offer the 46 pan
// cabinets and the Transient Shaper on every device here; an HX Stomp answers
// a request to load one with a chain they are not in.
//
// The failure it prevents is silent, which is why it is a refusal rather than
// a note. A rig naming one of these resolves, writes a preset the device
// accepts, reads back byte for byte, and plays a chain with the block missing.
// 47 readings were committed off exactly that: a sweep measured the empty slot
// for five minutes and filed the figures under the cabinet's name.
//
// `catalog.Block.Preferred` is not enough on its own. It sorts the pan
// variants behind the plain ones so nothing chooses one, and says so, but a
// document may name gear by hand and `namedGear` rightly lets a named block
// through a filter that would never have picked it.
//
// Keyed off the device because the answer is. 65 presets in the corpus hold a
// pan cabinet on a two-processor Helix and not one holds a pan cabinet on a
// Stomp, so these load elsewhere and this must not refuse them there. A device
// nobody has measured refuses nothing here, which is the honest default: a
// refusal nobody measured would stop a rig that works.
func ValidateHolds(
	l BlockLookup,
	s Plan,
	lim Limits,
) error {
	if len(lim.Refuses) == 0 && len(lim.RefusesModel) == 0 {
		return nil
	}

	for _, b := range s.Blocks {
		if lim.RefusesModel[b.Model] {
			return &RefusedBlockError{Model: string(b.Model)}
		}

		blk, ok := l.Block(b.Model)
		if !ok {
			continue
		}

		if lim.Refuses[blk.Family] {
			return &RefusedBlockError{Model: string(b.Model)}
		}
	}

	return nil
}
