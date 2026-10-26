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

// WorkingTestSuite covers whether the chain is still passing signal partway
// through a sweep.
//
// The silence floor is derived from what the chain settles at, and that is read
// again before every control, so a chain that dies takes the floor with it and
// every reading afterwards clears a floor that is itself silence. This is what
// holds the later controls to the first one instead.
type WorkingTestSuite struct {
	suite.Suite
}

// TestWorking covers which settled levels are the chain still working.
//
// One method and one table, so a case is a row rather than a file.
func (s *WorkingTestSuite) TestWorking() {
	for _, tt := range []struct {
		name            string
		settled, anchor float64
		says            string
	}{
		{
			// The reading this exists for. `6x10 Cali Power` settled at
			// -54.08dB for its Mic and around -130 for the six controls after
			// it, each held to a floor of -160, so six sweeps of hiss were
			// recorded as centroids between 908 and 3083Hz on a bass cabinet.
			name:    "the chain died after the first control",
			settled: -129.20,
			anchor:  -54.08,
			says:    "gone quiet partway through",
		},
		{
			name:    "the chain is where it was",
			settled: -54.09,
			anchor:  -54.08,
		},
		{
			// Quieter, and allowed to be: a control at one end of its travel
			// may genuinely turn the chain down, and `silent` is the margin
			// that says how far before it stops being a reading.
			name:    "quieter by less than the silence margin",
			settled: -80.0,
			anchor:  -54.08,
		},
		{
			// Exactly the margin, which is still a reading. The refusal is for
			// what sits under it.
			name:    "quieter by exactly the silence margin",
			settled: -84.08,
			anchor:  -54.08,
		},
		{
			// Louder than the first control, which is nothing to refuse: a
			// boost measured after a cut reads louder and is still a reading.
			name:    "louder than the first control",
			settled: -20.0,
			anchor:  -54.08,
		},
		{
			// The first control itself, which has nothing to be held to. A
			// chain that is silent from the start is `arrived`'s business:
			// `seated` only reads the chain back over USB and says the device
			// loaded the blocks, which is true of a pedal with nothing
			// plugged into it.
			name:    "no anchor yet",
			settled: -129.20,
			anchor:  0,
		},
	} {
		s.Run(tt.name, func() {
			err := working("Position", tt.settled, tt.anchor)

			if tt.says == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, errChainDied)
			s.Require().ErrorContains(err, tt.says)
			s.Require().ErrorContains(err, "Position")
		})
	}
}

func TestWorkingTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WorkingTestSuite))
}
