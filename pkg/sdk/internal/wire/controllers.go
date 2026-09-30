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

package wire

import "fmt"

// This file writes the controller section, which is what makes an expression
// pedal or a footswitch on a rig reach the hardware.
//
// Everything else in place.go replaces a span: the blank preset already holds
// the structure and the bytes are overwritten where they sit. This cannot,
// because a preset with nothing assigned holds nil where an assignment goes.
// There is nothing to overwrite, so the section is built and the whole of it
// replaces what was there.

// controllerSlots is how many controllers a preset carries, one per index.
//
// Ten, as the decoder reads: the section is an array of that length and the
// index is the controller, so an expression pedal assigned to one parameter
// leaves nine entries holding nothing.
const controllerSlots = 10

// PlacedController is one assignment, in the numbers a device stores.
//
// Resolved rather than named, the way Placement is. The device stores a
// parameter's place in its model's own order and knows no names, so turning
// "Drive" into a number needs the catalog and belongs above this package.
type PlacedController struct {
	// Controller is which one, as the device numbers them. The expression
	// pedal on an HX Stomp is 2.
	Controller int
	// Block is the block it works on, counted the way a chain counts its
	// entries. PlaceControllers shifts it onto the grid.
	Block int
	// Param is the parameter's place in the model's own order, which is the
	// order the block's values are written in.
	Param int
	// Min and Max are the ends of the controller's travel.
	Min float64
	Max float64
	// NoSnapshot is whether snapshots leave this assignment alone.
	NoSnapshot bool
}

// PlaceControllers writes what an expression pedal or a footswitch moves.
//
// Positions are a chain's and shifted onto the grid here, which is what
// PlaceAsWritten does with the blocks they point at. An assignment naming a
// block by its chain position and landing on the grid position of another is
// the mistake this prevents.
//
// The whole section is written every time, so a preset says the same thing
// about its controllers whatever the slot held before. That matches how the
// chain is placed: every position is written, one the chain names gets a block
// and one it does not gets emptied.
func PlaceControllers(
	doc *Document,
	of []PlacedController,
) error {
	at := make(map[int][]PlacedController, len(of))

	for _, c := range of {
		if c.Controller < 0 || c.Controller >= controllerSlots {
			return fmt.Errorf(
				"%w: controller %d, and a preset holds %d",
				ErrNoRoom, c.Controller, controllerSlots)
		}

		if c.Param < 0 {
			return fmt.Errorf(
				"%w: controller %d has parameter %d", ErrNoRoom, c.Controller, c.Param)
		}

		at[c.Controller] = append(at[c.Controller], c)
	}

	doc.SetSection(int8(keyControllers), controllerSection(at))

	return nil
}

// controllerSection renders the array of ten.
func controllerSection(
	at map[int][]PlacedController,
) []byte {
	out := arrayHeader(controllerSlots)

	for i := range controllerSlots {
		held, ok := at[i]
		if !ok {
			// Nothing assigned to this one, which is nine of the ten on any
			// real preset.
			out = append(out, codeNil)

			continue
		}

		out = append(out, arrayHeader(len(held))...)

		for _, c := range held {
			out = append(out, assignment(c)...)
		}
	}

	return out
}

// assignment renders one, which is a parameter and a body.
func assignment(
	c PlacedController,
) []byte {
	out := mapHeader(2)

	out = append(out, byte(keyCtrlParam))
	out = append(out, encodeNumber(c.Param)...)
	out = append(out, byte(keyCtrlBody))

	return append(out, assignmentBody(c)...)
}

// assignmentBody renders the block, the travel and the flags.
//
// The flags map is written whether or not the switch is set, because the
// device writes it either way and a section that leaves it out reads as a
// preset from an older release rather than as one with the switch off.
func assignmentBody(
	c PlacedController,
) []byte {
	out := mapHeader(4)

	out = append(out, byte(keyCtrlBlock))
	out = append(out, encodeNumber(c.Block+GridOffset)...)

	// A float32, which is what a device writes and what it sends back. Written
	// as a float64 the value survives a round trip through this package and
	// comes back off the hardware a different number.
	out = append(out, byte(keyCtrlMin))
	out = append(out, encodeFloat(c.Min, codeFloat32)...)
	out = append(out, byte(keyCtrlMax))
	out = append(out, encodeFloat(c.Max, codeFloat32)...)

	out = append(out, byte(keyCtrlFlags))
	out = append(out, mapHeader(1)...)
	out = append(out, byte(keyCtrlNoSnapshot), boolean(c.NoSnapshot))

	return out
}
