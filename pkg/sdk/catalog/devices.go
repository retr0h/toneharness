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

// devices are the catalogs this binary carries, in the order they were
// generated, which puts first the device everything here was written against.
//
// One list rather than a name map and an id map, so a device cannot be added
// to half of them.
var devices = []struct {
	// Name is how Line 6 markets the device.
	Name string
	// ID is what a preset carries in data.device.
	ID int
}{
	{"HX Stomp", HXStomp},
	{"HX Stomp XL", HXStompXL},
	{"Helix Floor", HelixFloor},
	{"Helix LT", HelixLT},
}

// Devices names every device this binary carries a catalog for.
func Devices() []string {
	out := make([]string, 0, len(devices))

	for _, d := range devices {
		out = append(out, d.Name)
	}

	return out
}

// For returns the built-in catalog for one device.
//
// By the id a preset carries in data.device, which is also what filtered the
// model table when the catalog was generated.
//
// Only an HX Stomp has been checked against real hardware. The other three are
// read from Line 6's own files and describe devices nobody here has written
// to.
func For(
	device int,
) (*Catalog, error) {
	body, ok := packed[device]
	if !ok {
		return nil, &NoDeviceError{Device: device}
	}

	// Decoded once per device, and see held for why that matters.
	held.Lock()
	defer held.Unlock()

	if got, already := held.by[device]; already {
		return got, nil
	}

	got, err := decode(body)
	if err != nil {
		return nil, err
	}

	held.by[device] = got

	return got, nil
}
