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

// BlockAt finds the block sitting at a position on one processor.
//
// Both, because both paths of a device that has two count their blocks from
// zero. A chain laid across them holds two blocks numbered 0, and a position
// on its own would find whichever came first.
//
// It lives beside the Plan it searches rather than in either caller, because
// the compiler and the editor had each written it, identically, under two
// names: `blockAt` and `blockAtPath`. What addresses a block is a fact about
// the plan, and the two names read as two questions.
func BlockAt(
	blocks []Block,
	path int,
	position int,
) (Block, bool) {
	for _, b := range blocks {
		if b.DSP == path && b.Pos == position {
			return b, true
		}
	}

	return Block{}, false
}
