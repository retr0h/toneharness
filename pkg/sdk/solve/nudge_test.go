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

package solve

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// NudgeTestSuite covers moving a target by what somebody heard.
//
// All arithmetic, and the unit is the point: a nudge is measured in tolerances
// because a tolerance is the spread across a genre's own records, which is the
// only definition of "noticeable" here that nobody chose.
type NudgeTestSuite struct {
	suite.Suite
}

// aims is a target with a wide axis and a narrow one.
func (s *NudgeTestSuite) aims() map[audio.Figure]Aim {
	return map[audio.Figure]Aim{
		audio.KeyCentroid: {Want: 140, Tol: 20},
		audio.KeyLow:      {Want: 0.9, Tol: 0.05},
	}
}

// TestOneStepIsOneTolerance is the whole design.
func (s *NudgeTestSuite) TestOneStepIsOneTolerance() {
	got := Nudge(s.aims(), []Nudged{
		{Term: "bright", Key: audio.KeyCentroid, Up: true, Steps: 1},
	})

	s.Require().InDelta(160.0, got[audio.KeyCentroid].Want, 0.001)
	s.Require().InDelta(20.0, got[audio.KeyCentroid].Tol, 0.001,
		"the tolerance is the unit, so moving the target does not widen it")
}

// TestDownTheAxis covers the other direction.
func (s *NudgeTestSuite) TestDownTheAxis() {
	got := Nudge(s.aims(), []Nudged{
		{Term: "dark", Key: audio.KeyCentroid, Up: false, Steps: 1},
	})

	s.Require().InDelta(120.0, got[audio.KeyCentroid].Want, 0.001)
}

// TestStepsScaleIt covers "much darker" and "a touch darker".
func (s *NudgeTestSuite) TestStepsScaleIt() {
	tests := []struct {
		name  string
		steps float64
		want  float64
	}{
		{"a touch", 0.5, 150},
		{"the default when nothing says", 0, 160},
		{"much", 3, 200},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := Nudge(s.aims(), []Nudged{
				{Key: audio.KeyCentroid, Up: true, Steps: tt.steps},
			})

			s.Require().InDelta(tt.want, got[audio.KeyCentroid].Want, 0.001)
		})
	}
}

// TestTwoFiguresFromOneWord covers a word answering more than one axis.
//
// "punchy" is a tight low end and a hard attack, and both are measured, so one
// word moves two figures.
func (s *NudgeTestSuite) TestTwoFiguresFromOneWord() {
	got := Nudge(s.aims(), []Nudged{
		{Term: "punchy", Key: audio.KeyCentroid, Up: true, Steps: 1},
		{Term: "punchy", Key: audio.KeyLow, Up: false, Steps: 1},
	})

	s.Require().InDelta(160.0, got[audio.KeyCentroid].Want, 0.001)
	s.Require().InDelta(0.85, got[audio.KeyLow].Want, 0.001)
}

// TestAFigureNothingAimsAtIsLeftAlone covers a genre that shrugs at the axis.
//
// A genre pins the figures its records agree about and leaves the rest free by
// design, and a word cannot move a target that is not there.
func (s *NudgeTestSuite) TestAFigureNothingAimsAtIsLeftAlone() {
	got := Nudge(s.aims(), []Nudged{
		{Term: "saturated", Key: audio.KeyHarmonics, Up: true, Steps: 1},
	})

	s.Require().NotContains(got, audio.KeyHarmonics)
	s.Require().Len(got, 2, "and nothing else moved either")
}

// TestAToleranceOfZeroIsNotSomethingToStepBy covers an unconstrained axis.
func (s *NudgeTestSuite) TestAToleranceOfZeroIsNotSomethingToStepBy() {
	got := Nudge(map[audio.Figure]Aim{audio.KeyCentroid: {Want: 140}}, []Nudged{
		{Key: audio.KeyCentroid, Up: true, Steps: 1},
	})

	s.Require().InDelta(140.0, got[audio.KeyCentroid].Want, 0.001)
}

// TestAShareCannotGoPastTheWhole covers clamping.
//
// Asking for 1.05 of the low band asks for more energy than there is, and the
// solve would spend every control chasing it and then report a chain that
// cannot reach the target.
func (s *NudgeTestSuite) TestAShareCannotGoPastTheWhole() {
	got := Nudge(map[audio.Figure]Aim{
		audio.KeyLow:       {Want: 0.97, Tol: 0.2},
		audio.KeyHarmonics: {Want: 0.1, Tol: 0.4},
	}, []Nudged{
		{Key: audio.KeyLow, Up: true, Steps: 1},
		{Key: audio.KeyHarmonics, Up: false, Steps: 1},
	})

	s.Require().InDelta(1.0, got[audio.KeyLow].Want, 0.001)
	s.Require().InDelta(0.0, got[audio.KeyHarmonics].Want, 0.001,
		"and not below nothing either")
}

// TestAFigureInItsOwnUnitsHasNoCeiling covers the other half of clamping.
//
// A centroid in hertz has no top this package knows, so only the floor at zero
// applies.
func (s *NudgeTestSuite) TestAFigureInItsOwnUnitsHasNoCeiling() {
	got := Nudge(map[audio.Figure]Aim{audio.KeyCentroid: {Want: 140, Tol: 500}},
		[]Nudged{{Key: audio.KeyCentroid, Up: true, Steps: 1}})

	s.Require().InDelta(640.0, got[audio.KeyCentroid].Want, 0.001)
}

// TestTheAimsHandedInAreNotChanged covers the loop reading them every pass.
//
// A nudge applied to the same map twice would walk the target away a tolerance
// at a time, and the residual reported each pass is read off these.
func (s *NudgeTestSuite) TestTheAimsHandedInAreNotChanged() {
	was := s.aims()

	_ = Nudge(was, []Nudged{{Key: audio.KeyCentroid, Up: true, Steps: 1}})

	s.Require().InDelta(140.0, was[audio.KeyCentroid].Want, 0.001)
}

func TestNudgeTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NudgeTestSuite))
}
