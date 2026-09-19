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
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/cab"
)

// CabPublicTestSuite covers building a cabinet the device does not have.
type CabPublicTestSuite struct {
	suite.Suite
}

// speaker is a crude cabinet: everything above a corner frequency rolls off.
//
// Not a real speaker, and it does not have to be. What is being checked is
// the arithmetic that makes one response into another, and a first order
// roll-off has a magnitude curve that is easy to state and easy to check.
func speaker(
	n int,
	corner float64,
) []float64 {
	out := make([]float64, n)

	rc := 1 / (2 * math.Pi * corner)
	dt := 1 / float64(cab.Rate)
	a := dt / (rc + dt)

	held := 0.0

	for i := range out {
		x := 0.0
		if i == 0 {
			x = 1
		}

		held += a * (x - held)
		out[i] = held
	}

	return out
}

// through runs a signal down an impulse response.
func through(
	signal, impulse []float64,
) []float64 {
	out := make([]float64, len(signal))

	for i := range out {
		var sum float64

		for j, tap := range impulse {
			if i-j < 0 {
				break
			}

			sum += signal[i-j] * tap
		}

		out[i] = sum
	}

	return out
}

// centre is where a signal's energy sits, in hertz.
func centre(
	of []float64,
) float64 {
	n := 1
	for n < len(of) {
		n *= 2
	}

	re := make([]float64, n)
	im := make([]float64, n)
	copy(re, of)

	cab.Forward(re, im)

	var weighted, total float64

	for i := range n / 2 {
		power := re[i]*re[i] + im[i]*im[i]
		hz := float64(i) * cab.Rate / float64(n)

		weighted += hz * power
		total += power
	}

	if total == 0 {
		return 0
	}

	return weighted / total
}

// TestMatchMakesOneCabinetMeasureLikeAnother is the whole point.
//
// A bright cabinet and a dark one, and the filter between them. Run the
// bright one through that filter and what comes out should sit where the dark
// one sits.
func (s *CabPublicTestSuite) TestMatchMakesOneCabinetMeasureLikeAnother() {
	bright := speaker(cab.Long, 3000)
	dark := speaker(cab.Long, 600)

	before := centre(bright)
	want := centre(dark)

	s.Require().Greater(before, want*2,
		"the two cabinets are far enough apart for the test to mean anything")

	fix, err := cab.Match(dark, bright, cab.Long)
	s.Require().NoError(err)
	s.Require().Len(fix, cab.Long)

	after := centre(through(bright, fix))

	s.Require().Less(math.Abs(after-want), math.Abs(before-want),
		"the correction moved it toward the target rather than away")
	s.Require().InEpsilon(want, after, 0.25,
		"and landed within a quarter of it: %0.f Hz wanted, %.0f Hz before, "+
			"%.0f Hz after", want, before, after)
}

// TestCaptureRecoversWhatACabinetDid covers deconvolution.
//
// A known signal through a known cabinet, and dividing one by the other in
// the frequency domain gives the cabinet back.
func (s *CabPublicTestSuite) TestCaptureRecoversWhatACabinetDid() {
	want := speaker(cab.Long, 1500)

	// A sweep has energy everywhere, which is what makes the division safe.
	sent := make([]float64, cab.Long*4)
	for i := range sent {
		at := float64(i) / float64(len(sent))
		sent[i] = math.Sin(2 * math.Pi * (20 + 12000*at*at) * float64(i) /
			float64(cab.Rate))
	}

	got, err := cab.Capture(sent, through(sent, want), cab.Long)
	s.Require().NoError(err)
	s.Require().Len(got, cab.Long)

	s.Require().InEpsilon(centre(want), centre(got), 0.15,
		"what came back measures like the cabinet that was in the path")
}

// TestAnImpulseResponseNeverClips covers what a device will load.
func (s *CabPublicTestSuite) TestAnImpulseResponseNeverClips() {
	loud := make([]float64, cab.Long)
	for i := range loud {
		loud[i] = 50
	}

	got, err := cab.Match(loud, speaker(cab.Long, 1000), cab.Long)
	s.Require().NoError(err)

	for i, v := range got {
		s.Require().LessOrEqualf(math.Abs(v), 1.0,
			"tap %d is past full scale at %f", i, v)
	}
}

// TestItEndsQuietly covers the window on the tail.
//
// A response that stops abruptly has a step in it, and a step is broadband:
// it reads as a click on every note, which is not what the cabinet did.
func (s *CabPublicTestSuite) TestItEndsQuietly() {
	got, err := cab.Match(
		speaker(cab.Long, 600), speaker(cab.Long, 3000), cab.Long)
	s.Require().NoError(err)

	var peak float64
	for _, v := range got {
		peak = math.Max(peak, math.Abs(v))
	}

	s.Require().Less(math.Abs(got[len(got)-1]), peak*0.01,
		"the last tap is not where the energy is")
}

// TestOnlyWhatADeviceLoads covers the two lengths.
func (s *CabPublicTestSuite) TestOnlyWhatADeviceLoads() {
	for _, taps := range []int{0, 512, 1000, 4096} {
		_, err := cab.Match(
			speaker(4096, 600), speaker(4096, 3000), taps)

		s.Require().ErrorIs(err, cab.ErrNotPowerOfTwo, "%d taps", taps)
	}

	for _, taps := range []int{cab.Short, cab.Long} {
		got, err := cab.Match(speaker(4096, 600), speaker(4096, 3000), taps)

		s.Require().NoError(err)
		s.Require().Len(got, taps)
	}
}

// TestNotEnoughToWorkFrom covers signals with too little in them.
func (s *CabPublicTestSuite) TestNotEnoughToWorkFrom() {
	short := make([]float64, 10)
	fine := speaker(cab.Long, 1000)

	_, err := cab.Match(short, fine, cab.Long)
	s.Require().ErrorIs(err, cab.ErrTooShort)

	_, err = cab.Match(fine, short, cab.Long)
	s.Require().ErrorIs(err, cab.ErrTooShort)

	_, err = cab.Capture(short, fine, cab.Long)
	s.Require().ErrorIs(err, cab.ErrTooShort)

	_, err = cab.Capture(fine, short, cab.Long)
	s.Require().ErrorIs(err, cab.ErrTooShort)
}

// TestSilenceIsNotSomethingToDivideBy covers a signal holding nothing.
func (s *CabPublicTestSuite) TestSilenceIsNotSomethingToDivideBy() {
	quiet := make([]float64, cab.Long)
	fine := speaker(cab.Long, 1000)

	_, err := cab.Capture(quiet, fine, cab.Long)
	s.Require().ErrorIs(err, cab.ErrTooShort)

	_, err = cab.Match(fine, quiet, cab.Long)
	s.Require().ErrorIs(err, cab.ErrTooShort)
}

func TestCabPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CabPublicTestSuite))
}
