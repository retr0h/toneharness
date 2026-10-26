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

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ArrivedTestSuite covers whether anything came back from the chain at all.
//
// The third of three guards, and the one that was missing. `seated` reads the
// chain back over USB and says the device loaded the blocks, which it had.
// `working` holds each control to the first one and catches a chain that dies
// partway through, which this had not: it was dead before the first control,
// so every control agreed with the anchor. Neither asks whether any audio
// arrived.
type ArrivedTestSuite struct {
	suite.Suite
}

// TestArrived covers which settled levels are a reading of something.
//
// One method and one table, so a case is a row rather than a file.
func (s *ArrivedTestSuite) TestArrived() {
	for _, tt := range []struct {
		name             string
		reference, quiet float64
		says             bool
	}{
		{
			// The live rig, measured on 2026-10-25: the reference settles at
			// -59.24dB and the same path carrying nothing reads -128.01dB,
			// which is 68.8dB apart. There is no argument at this end.
			name:      "the reference is well clear of the idle path",
			reference: -59.24,
			quiet:     -128.01,
		},
		{
			// The reading this exists for. A sweep ran with nothing plugged
			// into the pedal's input and its reference settled at -136.12dB,
			// which is quieter than the same path carries at rest: the two
			// are the same reading and one of them is being called signal.
			name:      "nothing arrived, so the reference is under the idle path",
			reference: -136.12,
			quiet:     -128.01,
			says:      true,
		},
		{
			// The two agreeing exactly, which is what a dead path looks like
			// when its noise happens to match: still nothing arrived.
			name:      "the reference and the idle path read alike",
			reference: -128.01,
			quiet:     -128.01,
			says:      true,
		},
		{
			// Exactly the margin, which is a reading. The refusal is for what
			// sits under it, the same way `working` treats its own.
			name:      "clear by exactly the silence margin",
			reference: -98.0,
			quiet:     -128.0,
		},
		{
			// A whisker under, which is not. A block may genuinely be quiet
			// and this is where quiet stops being a reading.
			name:      "clear by a shade less than the margin",
			reference: -98.5,
			quiet:     -128.0,
			says:      true,
		},
		{
			// A noisy path with a loud reference, which is fine: the test is
			// the distance between the two rather than either figure. A
			// figure measured on one machine is a figure about that machine.
			name:      "a noisy path the reference still clears",
			reference: -20.0,
			quiet:     -60.0,
		},
	} {
		s.Run(tt.name, func() {
			err := arrived(tt.reference, tt.quiet)

			if !tt.says {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, errNothingArrived)
			s.Require().ErrorContains(err, "Nothing is arriving from the pedal")
			s.Require().ErrorContains(err, "check the lead into its input")
		})
	}
}

func TestArrivedTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ArrivedTestSuite))
}
