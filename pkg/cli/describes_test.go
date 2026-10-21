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

// DescribesTestSuite covers saying whether a committed slope describes its own
// curve.
//
// The comparison prints a ratio between a committed slope and a live one, and a
// ratio has two explanations: the rig changed, or the committed figure is a line
// through a curve that is not one. This is what tells them apart.
type DescribesTestSuite struct {
	suite.Suite
}

// TestDescribes covers each verdict a straightness earns.
//
// One method and one table, so a case is a row rather than a file.
func (s *DescribesTestSuite) TestDescribes() {
	for _, tt := range []struct {
		name     string
		straight float64
		want     string
	}{
		{
			// A cabinet's LowCut, which is both the largest lever in the
			// library and almost perfectly linear. A ratio against this one
			// means what it says.
			name: "a line that fits", straight: 0.97, want: "0.97",
		},
		{
			name: "exactly the floor", straight: 0.8, want: "0.80",
		},
		{
			// An amplifier's Mid Freq reads 0.02 and a filter's Envelope 0.05.
			// A slope here is arithmetic on a number with no meaning, so a
			// ratio against it is not a disagreement about the rig.
			name: "a line through a curve", straight: 0.02, want: "0.02 NONE",
		},
		{
			// Every list reads zero and is right to: twelve microphones have no
			// order for a line to run through. Those are compared rather than
			// solved.
			name: "a list, which has no order", straight: 0, want: "0.00 NONE",
		},
		{
			name: "between the two", straight: 0.46, want: "0.46 poor",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, describes(tt.straight))
		})
	}
}

func TestDescribesTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DescribesTestSuite))
}
