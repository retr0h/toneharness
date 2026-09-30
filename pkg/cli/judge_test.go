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

// JudgePublicTestSuite covers whether a live reading can be believed.
//
// Two commands ask it: a sweep, which buckets a control's positions into what
// muted the chain and what clipped it, and the list comparison, which says so
// and drops the setting. They were two switches printing byte-identical
// strings.
type JudgePublicTestSuite struct {
	suite.Suite
}

// TestSilentBelowIsSettledLessSilent pins the one arithmetic both sides use.
//
// A sweep stores the answer on its Curve, where it is part of the serialised
// contract, and the list comparison works it out again. One function, so a
// stored floor and a recomputed one cannot be different floors.
func (s *JudgePublicTestSuite) TestSilentBelowIsSettledLessSilent() {
	s.Require().InDelta(-62.4, silentBelow(-32.4), 1e-9)
	s.Require().InDelta(silent, silentBelow(0)*-1, 1e-9)
}

func (s *JudgePublicTestSuite) TestJudge() {
	floor := silentBelow(-32.4)

	tests := []struct {
		name  string
		level float64
		want  judgement
		says  string
	}{
		{"a reading of the block", -32.4, believable, ""},
		{
			"one under the floor is the noise", floor - 0.1, tooQuiet,
			"silent, so its figures are of the noise",
		},
		{
			"one over the ceiling is the converters", clipped + 0.1, tooLoud,
			"clipped, so its figures are the converters'",
		},
		{
			"exactly on the floor is still a reading", floor, believable, "",
		},
		{
			"exactly on the ceiling is too", clipped, believable, "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, why := judge(tt.level, floor)

			s.Require().Equal(tt.want, got)
			s.Require().Equal(tt.says, why)
		})
	}
}

// TestSelfNoiseIsItsOwnNumber is the coupling this split undid.
//
// The squeal detector in measure_blocks and the sweep's muted test held one
// constant between them, at the same value for unrelated reasons, so retuning
// either retuned the other in a different file. Equal today and separately
// named, which is the point: this test fails on a merge back into one.
func (s *JudgePublicTestSuite) TestSelfNoiseIsItsOwnNumber() {
	s.Require().InDelta(30.0, silent, 1e-9, "how far under settled is silence")
	s.Require().InDelta(30.0, selfNoise, 1e-9, "how far under a reading is quiet")
}

func TestJudgePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(JudgePublicTestSuite))
}
