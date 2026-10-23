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

package measured

// Unpack reads a gzipped library, exported so a test can hand it bytes the
// shipped file will never be: something that is not gzip at all, and a
// library measured in a chain rather than one block at a time.
var Unpack = unpack

// Exposed to this package's external tests. Which field a figure key means,
// and a figure a reading may not carry read as a number, are both tables with
// more rows than any build reaches: nine keys exist and a bass is weighed on
// three of them.
var (
	// Of is one figure by the key the corpus names it with.
	Of = Figures.of
	// OrZero is a figure a reading may not carry, as a number.
	OrZero = orZero
	// Pair is one figure from each side of a comparison, and whether both
	// carry it.
	Pair = pair
	// Shared is a target split among the dials that can each answer it.
	Shared = Curves.shared
)
