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
package audio_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// TransformPublicTestSuite covers the transform both ways.
//
// Exported because building an impulse response is the same arithmetic
// pointed the other way, and a second implementation beside this one is the
// shape of mistake this package has already paid for once.
type TransformPublicTestSuite struct {
	suite.Suite
}

// TestATransformUndoesItself covers the round trip.
func (s *TransformPublicTestSuite) TestATransformUndoesItself() {
	const n = 256

	want := make([]float64, n)
	for i := range want {
		want[i] = math.Sin(2*math.Pi*float64(i)/32) +
			0.5*math.Cos(2*math.Pi*float64(i)/8)
	}

	re := make([]float64, n)
	im := make([]float64, n)
	copy(re, want)

	audio.Forward(re, im)
	audio.Inverse(re, im)

	for i := range want {
		s.Require().InDeltaf(want[i], re[i], 1e-9,
			"sample %d came back as something else", i)
		s.Require().InDeltaf(0, im[i], 1e-9,
			"sample %d gained an imaginary part", i)
	}
}

// TestATransformFindsAToneWhereItIs covers the bins meaning what they say.
func (s *TransformPublicTestSuite) TestATransformFindsAToneWhereItIs() {
	const (
		n    = 512
		bin  = 17
		peak = 1.0
	)

	re := make([]float64, n)
	im := make([]float64, n)

	for i := range re {
		re[i] = peak * math.Cos(2*math.Pi*bin*float64(i)/n)
	}

	audio.Forward(re, im)

	var loudest int

	var most float64

	for i := range n / 2 {
		if power := re[i]*re[i] + im[i]*im[i]; power > most {
			most, loudest = power, i
		}
	}

	s.Require().Equal(bin, loudest)
}

// TestNothingToTransform covers the lengths that cannot be halved.
func (s *TransformPublicTestSuite) TestNothingToTransform() {
	for _, n := range []int{0, 1} {
		re := make([]float64, n)
		im := make([]float64, n)

		s.Require().NotPanics(func() { audio.Forward(re, im) })
	}
}

func TestTransformPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TransformPublicTestSuite))
}
