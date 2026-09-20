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

// ForName returns the built-in catalog for a device, by what it is called.
//
// Loosely matched, so "Helix LT", "helix lt" and "helix-lt" all reach the same
// catalog. Somebody naming their own pedal should not have to guess which
// spelling this tool wants.
func ForName(
	name string,
) (*Catalog, error) {
	want := fold(name)

	for _, d := range devices {
		if fold(d.Name) == want {
			return For(d.ID)
		}
	}

	return nil, &UnknownDeviceError{Name: name, Known: Devices()}
}

// fold reduces a device name to what matching cares about.
//
// Letters and digits. Everything else is somebody's spacing or punctuation,
// and none of it distinguishes one Line 6 device from another.
func fold(
	name string,
) string {
	var out strings.Builder

	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}

	return out.String()
}
