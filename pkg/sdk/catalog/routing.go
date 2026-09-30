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

// This file answers which number a device files one end of its chain under.
//
// Apart from types.go because `at` is a package-level function rather than a
// method, and CONTRIBUTING: "types.go holds only type declarations... A
// function belongs in a file named for what it does." The exemption there
// covers a method beside its type, which SourceAt and DestinationAt are and
// which `at` is not.

// SourceAt is the number a device files one chain input under, by name.
//
// By name because the number is the device family's and the name is not:
// "USB 5/6" means the same thing on every Helix and is 15 on only some of them.
func (c *Catalog) SourceAt(
	name string,
) (int, bool) {
	return at(c.Sources, name)
}

// DestinationAt is the number a device files one chain output under, by name.
func (c *Catalog) DestinationAt(
	name string,
) (int, bool) {
	return at(c.Destinations, name)
}

// at finds a name's position in a device's own list.
//
// Case-insensitive, because these are written for a menu and read here as
// values, and the same list spells "USB 1/2" and "S/PDIF" to be looked at
// rather than matched.
func at(
	in []string,
	name string,
) (int, bool) {
	for i, got := range in {
		if strings.EqualFold(got, name) {
			return i, true
		}
	}

	return 0, false
}
