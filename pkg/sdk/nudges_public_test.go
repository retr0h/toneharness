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
package sdk_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	tonespec "github.com/retr0h/toneharness/pkg/sdk/tone"
)

// NudgesPublicTestSuite covers the seam between the vocabulary and the solver.
type NudgesPublicTestSuite struct {
	suite.Suite
}

// steps is a pointer to a step count, which is how the ask holds one.
func (s *NudgesPublicTestSuite) steps(
	of float32,
) *float32 {
	return &of
}

// TestWhatSomebodySaidBecomesFiguresToMove is the base case.
func (s *NudgesPublicTestSuite) TestWhatSomebodySaidBecomesFiguresToMove() {
	got, err := sdk.Nudges([]tonespec.Nudge{{Word: "darker"}})

	s.Require().NoError(err)
	s.Require().Len(got, 1)

	s.Require().Equal("darker", got[0].Term, "the word as it was said")
	s.Require().Equal(audio.KeyCentroid, got[0].Key)
	s.Require().False(got[0].Up)
	s.Require().InDelta(1.0, got[0].Steps, 0.001, "one is what a person means")
}

// TestStepsCarryThrough covers "much darker".
func (s *NudgesPublicTestSuite) TestStepsCarryThrough() {
	got, err := sdk.Nudges([]tonespec.Nudge{
		{Word: "darker", Steps: s.steps(2.5)},
	})

	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Require().InDelta(2.5, got[0].Steps, 0.001)
}

// TestOneWordMayMoveTwoFigures covers a word on two axes, both scaled.
func (s *NudgesPublicTestSuite) TestOneWordMayMoveTwoFigures() {
	got, err := sdk.Nudges([]tonespec.Nudge{
		{Word: "punchy", Steps: s.steps(2)},
	})

	s.Require().NoError(err)
	s.Require().Len(got, 2)

	for _, n := range got {
		s.Require().Equal("punchy", n.Term)
		s.Require().InDelta(2.0, n.Steps, 0.001)
	}
}

// TestNothingSaidMovesNothing covers a first answer.
//
// A nudge moves from wherever the last answer landed, and a run with nothing
// said has nothing to move from.
func (s *NudgesPublicTestSuite) TestNothingSaidMovesNothing() {
	got, err := sdk.Nudges(nil)

	s.Require().NoError(err)
	s.Require().Empty(got)
}

// TestEveryWordItCannotUseIsReported covers the failure that matters most.
//
// A nudge dropped in silence is the worst outcome available: the run then
// reports a tone nobody asked for and nothing says the instruction was
// ignored. So every bad word is named, not only the first.
func (s *NudgesPublicTestSuite) TestEveryWordItCannotUseIsReported() {
	_, err := sdk.Nudges([]tonespec.Nudge{
		{Word: "darker"},
		{Word: "chunky"},
		{Word: "quiet-strings"},
	})

	s.Require().Error(err)
	s.Require().ErrorContains(err, "chunky")
	s.Require().ErrorContains(err, "quiet-strings")
}

func TestNudgesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NudgesPublicTestSuite))
}
