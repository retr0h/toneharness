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

package reamp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

// AsideTestSuite covers the guard that keeps a test from changing the audio
// settings of the machine it runs on.
//
// Both directions, which is why the predicate is a variable: a guard whose
// pass-through nothing exercises has only been checked in the direction that
// refuses, and the direction that matters to somebody actually running this is
// the other one.
type AsideTestSuite struct {
	suite.Suite

	was func() bool
}

func (s *AsideTestSuite) SetupTest() { s.was = underTest }

func (s *AsideTestSuite) TearDownTest() { underTest = s.was }

// TestAside covers what a wrapped call does in each state.
//
// One method and one table, so a case is a row rather than a file.
func (s *AsideTestSuite) TestAside() {
	boom := errors.New("the platform said no")

	for _, tt := range []struct {
		name string
		// test is what the predicate answers.
		test bool
		// said is what the wrapped call would return, were it reached.
		said error
		// reached is whether it should be.
		reached bool
		// want is the error the caller sees.
		want error
	}{
		{
			// The whole point. The owner of this pedal was muted mid-song by
			// `Held(0)` reaching the platform from a test binary.
			name:    "a test binary never reaches the platform",
			test:    true,
			reached: false,
			want:    ErrUnderTest,
		},
		{
			// And somebody's own run still sets the level, which is the half
			// a guard like this can break without anybody noticing.
			name:    "a real run reaches it",
			test:    false,
			reached: true,
		},
		{
			name:    "and carries back what it said",
			test:    false,
			said:    boom,
			reached: true,
			want:    boom,
		},
	} {
		s.Run(tt.name, func() {
			underTest = func() bool { return tt.test }

			got := 0
			reached := false

			// A fake rather than the platform, so this proves the wiring
			// without touching anything: `writes` and `setsOutput` are the
			// only places the real calls are bound.
			call := aside(func(to int) error {
				got, reached = to, true

				return tt.said
			})

			err := call(38)

			s.Require().Equal(tt.reached, reached)

			if tt.reached {
				s.Require().Equal(38, got)
			}

			if tt.want == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.want)
		})
	}
}

// TestUnderTestIsTrueHere is the one assertion that cannot be faked.
//
// The default has to answer true in a test binary or every other test in this
// file is checking a guard that is not armed.
func (s *AsideTestSuite) TestUnderTestIsTrueHere() {
	s.Require().True(s.was(), "the default must be armed in a test binary")
}

func TestAsideTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AsideTestSuite))
}
