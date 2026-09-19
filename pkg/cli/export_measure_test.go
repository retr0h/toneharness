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

package cli

// The bookkeeping a campaign is mostly made of, exposed so it can be checked
// without an audio interface and six hundred blocks of hardware time.
var (
	// Wanted is every block worth trying, in a stable order.
	Wanted = wanted
	// Resume reads what a previous run got as far as.
	Resume = resume
	// RigFor is a rig holding one block.
	RigFor = rigFor
	// Reference reads the signal every block is measured against, and
	// Resample puts it at the rate the loop runs at.
	Reference = reference
	Resample  = resample
	// Hash identifies the reference signal.
	Hash = hash
	// Sweepable is the controls of a block a sweep can move.
	Sweepable = sweepable
	// Fill works out what a control's measured positions say.
	Fill = fill
	// WireOrder is a model's parameters in the order the device addresses.
	WireOrder = wireOrder
	// Changed is the parameters that differ between two readings.
	Changed = changed
	// Short is the first line of an error.
	Short = short
	// Keep writes a library out.
	Keep = keep
)

// Control is one parameter worth sweeping, for a test that builds one.
type Control = control

// NewControl builds one.
func NewControl(
	index int,
	name, kind string,
	low, high float64,
) Control {
	return control{index: index, name: name, kind: kind, low: low, high: high}
}
