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

package specdoc

import "errors"

// errNoSchema reports a document that describes no schemas.
var errNoSchema = errors.New("not a schema this can render")

// Buckets is the part of a preamble that every contract shares: how a field's
// values are constrained, which is this package's own scheme rather than
// anything a contract says about itself.
// Buckets is the part of a preamble every contract shares.
//
// How a field's values are constrained is this package's own scheme rather
// than anything a contract says about itself, so it is written once here and
// the rest of a preamble is written beside the contract it describes.
const Buckets = "A field marked `*` is required. A field that holds another " +
	"object has\nno grammar of its own and shows `\u2014`; the question moves " +
	"to that\nobject's table. Every other field is in one of four buckets:\n" +
	`
| grammar | means |
| --- | --- |
| closed | one of a fixed set, listed here and refused if it is not one of them |
| looked up | checked against the catalog for the device in hand, so the valid values depend on which device |
| shaped | checked against a pattern |
| open | prose. Nothing parses it, and nothing will refuse it for what it says |
`
