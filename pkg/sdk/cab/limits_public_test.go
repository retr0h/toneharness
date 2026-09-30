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

package cab_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/cab"
)

// LimitsPublicTestSuite holds the disclosure to the things it has to disclose.
//
// A test about prose, which is unusual and earns its place here. `Limits` is what
// somebody reads before deciding whether a file they built may be sold, and it
// did not exist at all until somebody followed the two references pointing at it.
// A disclosure that can be silently emptied is worse than none, because the
// references stay.
type LimitsPublicTestSuite struct {
	suite.Suite
}

// TestLimitsNamesEveryWayAFileCanMeasureRightAndBeWrong is the contract.
//
// Each of these is a way an impulse response passes every figure measured here
// and is still the wrong thing, which is the failure this package exists to
// disclose rather than the one it exists to avoid.
func (s *LimitsPublicTestSuite) TestLimitsNamesEveryWayAFileCanMeasureRightAndBeWrong() {
	// Each phrase has to sit on one line of the const, because this is a
	// substring match and the const is wrapped prose. A phrase chosen across a
	// line break fails on the wrapping rather than on the disclosure, which is
	// how this test first failed.
	for _, want := range []struct {
		about string
		says  string
	}{
		{about: "the phase Match invents", says: "minimum phase"},
		{about: "where that phase is wrong", says: "reflection"},
		{about: "Capture wanting one clock", says: "separate clocks"},
		{about: "what a truncated response loses", says: "low end"},
		{about: "a response recovered from music", says: "where the music had energy"},
		{about: "nothing being verified on hardware", says: "verified on a pedal"},
		{about: "what may be sold", says: "theirs to sell"},
		{about: "what may not", says: "copy"},
	} {
		s.Require().Contains(cab.Limits, want.says,
			"Limits says nothing about %s", want.about)
	}
}

// TestLimitsIsProseSomebodyCanRead guards against it being emptied to a stub.
//
// The references to it stay whatever it holds, so a one-line placeholder would
// read as a disclosure having been made.
func (s *LimitsPublicTestSuite) TestLimitsIsProseSomebodyCanRead() {
	s.Require().Greater(len(strings.Fields(cab.Limits)), 60,
		"a disclosure this short is a placeholder, and the references to it stay")
}

func TestLimitsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LimitsPublicTestSuite))
}
