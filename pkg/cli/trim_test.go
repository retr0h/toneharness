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
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

// OpenLoopTestSuite covers deciding the headroom trim from the rig.
type OpenLoopTestSuite struct {
	suite.Suite
}

// TestTheTrimFollowsTheRigUnlessSomebodySaidOtherwise is the whole rule.
//
// The trim exists because a lead from the pedal's output to its own input makes
// the chain hear itself. Where the computer plays and the pedal's output reaches
// nothing there is no path back, and the same chain reads the same spectrum
// with and without the trim, 30dB apart in level. So applying it there spends
// thirty decibels of signal over the noise floor on a problem that is absent.
func (s *OpenLoopTestSuite) TestTheTrimFollowsTheRigUnlessSomebodySaidOtherwise() {
	for _, tt := range []struct {
		name     string
		hardware string
		asked    float64
		told     bool
		want     float64
		says     bool
	}{
		{
			name:     "one device is the loop, so the trim stands",
			hardware: "HX Stomp", asked: -30, want: -30,
		},
		{
			name:     "two devices open the loop, so it is dropped and said",
			hardware: "External Headphones,HX Stomp", asked: -30, want: 0,
			says: true,
		},
		{
			// A person naming a number may be bounding something this cannot
			// see. Overriding them would make the flag a suggestion.
			name:     "an explicit value wins on either rig",
			hardware: "External Headphones,HX Stomp", asked: -12, told: true,
			want: -12,
		},
		{
			name:     "nothing to drop says nothing",
			hardware: "External Headphones,HX Stomp", asked: 0, want: 0,
		},
		{
			// Naming no device means the pedal, which is one device, which is
			// the loop. Reading that as open would take the trim off the rig
			// that needs it most.
			name:     "naming no device is still one device",
			hardware: "", asked: -30, want: -30,
		},
	} {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			got := trimFor(&buf, tt.hardware, tt.asked, tt.told)

			s.Require().InDelta(tt.want, got, 0.001)

			if tt.says {
				s.Require().Contains(buf.String(), "cannot hear itself",
					"dropping a trim silently is a 30dB change nobody asked about")
			} else {
				s.Require().Empty(buf.String())
			}
		})
	}
}

func TestOpenLoopTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OpenLoopTestSuite))
}
