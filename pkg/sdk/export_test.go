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

package sdk

import "github.com/retr0h/toneharness/pkg/sdk/internal/device"

// Opener is the bus a Client was built over, so a test can see which one New
// chose and what it handed that bus.
func (c *Client) Opener() device.Opener {
	return c.opts.devices
}

// WithDevices is the bus a Client reaches hardware through, in place of USB.
//
// Here rather than beside the other options because a caller outside the
// library cannot build a device.Editor: it answers in wire types. A test
// builds its own Client over a generated double, so no two tests share a bus
// and every suite can run in parallel.
func WithDevices(
	d device.Opener,
) Option {
	return func(o *options) { o.devices = d }
}

// HomeDirWas and ContractVersionWas replace what a starter setup needs, and
// hand back a function putting the real one back.
//
// Neither can fail in a correct build, which is the reason to be able to make
// them fail: these two branches decide whether somebody gets a file they can
// edit or a message saying why not, and a branch nothing can reach is one
// nobody has read.
func HomeDirWas(
	f func() (string, error),
) func() {
	was := homeDir
	homeDir = f

	return func() { homeDir = was }
}

// ContractVersionWas replaces where the version in a starter setup comes from.
func ContractVersionWas(
	f func() (string, error),
) func() {
	was := contractVersion
	contractVersion = f

	return func() { contractVersion = was }
}
