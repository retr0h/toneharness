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

// Package slug turns a name somebody wrote into the shape a path or a flag
// takes.
//
// One implementation, because three near-copies diverge. A preset lifted off a
// device, a directory of records and a band selected on the command line all
// have to agree on what "Red Hot Chili Peppers" is called, and two spellings of
// one band that slug differently are two bands as far as any grouping is
// concerned.
package slug

import "strings"

// Of is name as a slug: lower case, words joined by hyphens, nothing else.
//
// Anything that is not a letter or a digit becomes a hyphen, runs of hyphens
// collapse, and the ends are trimmed. So "Guns N' Roses" and "Guns n Roses"
// both come out "guns-n-roses", which is the point: the corpus groups on the
// slug and two spellings of one band must not read as two.
//
// A name with nothing usable in it answers "untitled" rather than empty,
// because an empty path segment is a directory nobody can name.
func Of(
	name string,
) string {
	var b strings.Builder

	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}

	out := strings.Trim(collapse(b.String()), "-")
	if out == "" {
		return "untitled"
	}

	return out
}

// collapse squeezes runs of hyphens down to one.
func collapse(
	s string,
) string {
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	return s
}
