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

//go:build !darwin

package reamp

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// OutputOtherTestSuite covers the platforms that cannot be told which device to
// play through.
//
// Only built where that is true, which is also the only place these functions
// are the real ones. On macOS the same names reach CoreAudio and calling them
// from a test would change the audio output of the machine running it.
type OutputOtherTestSuite struct {
	suite.Suite
}

// TestTheseCannotBeAsked covers all three answering that they cannot.
//
// Said rather than reported as success, because a stub that claims the device
// was set leaves a caller believing the level it pins afterwards is in the
// signal path.
//
// One method and one table, so a case is a row rather than a file.
func (s *OutputOtherTestSuite) TestTheseCannotBeAsked() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "reading which device plays",
			then: func() {
				at, err := output()

				s.Require().ErrorIs(err, ErrOutput)
				s.Require().Empty(at)
			},
		},
		{
			name: "setting which device plays",
			then: func() {
				s.Require().ErrorIs(setOutput("anything"), ErrOutput)
			},
		},
		{
			name: "listing what could play",
			then: func() {
				s.Require().Empty(outputs())
			},
		},
	} {
		s.Run(tt.name, func() { tt.then() })
	}
}

func TestOutputOtherTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OutputOtherTestSuite))
}
