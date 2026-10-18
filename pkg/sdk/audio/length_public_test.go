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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// LengthPublicTestSuite covers how long a recording is, and how much of it is
// sound.
type LengthPublicTestSuite struct {
	suite.Suite
}

// TestLengthen covers Lengthen, which reads a recording's length and where its
// sound begins and ends.
//
// One method and one table, so a case is a row rather than a file.
func (s *LengthPublicTestSuite) TestLengthen() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The whole length, which is what a download is checked against.
			name: "a tone is as long as it is",
			then: func() {
				got := audio.Lengthen(audio.Sine(110, 2.0, rate, 0.9), rate)

				s.Require().InDelta(2.0, got.Seconds, 0.02)
				s.Require().InDelta(0.0, got.Head, 0.02)
				s.Require().InDelta(0.0, got.Tail, 0.02)
				s.Require().InDelta(2.0, got.Sounding(), 0.02)
			},
		},
		{
			// What a padded download looks like, and the reason the head and
			// the tail are reported apart from the length.
			name: "silence at each end is found and kept out of the sounding part",
			then: func() {
				samples := append([]float64(nil), audio.Silence(0.5, rate)...)
				samples = append(samples, audio.Sine(110, 1.0, rate, 0.9)...)
				samples = append(samples, audio.Silence(0.8, rate)...)

				got := audio.Lengthen(samples, rate)

				s.Require().InDelta(2.3, got.Seconds, 0.02, "the file is all of it")
				s.Require().InDelta(0.5, got.Head, 0.05)
				s.Require().InDelta(0.8, got.Tail, 0.05)
				s.Require().InDelta(1.0, got.Sounding(), 0.05)
			},
		},
		{
			// A recording with nothing in it has no sound to find, and a head
			// and a tail that each ran to the end would overlap and report
			// more quiet than there is recording.
			name: "a silent recording has a length and no sounding part",
			then: func() {
				got := audio.Lengthen(audio.Silence(1.5, rate), rate)

				s.Require().InDelta(1.5, got.Seconds, 0.02)
				s.Require().Zero(got.Head)
				s.Require().Zero(got.Tail)
			},
		},
		{
			name: "nothing to measure",
			then: func() {
				s.Require().Zero(audio.Lengthen(nil, rate).Seconds)
				s.Require().Zero(audio.Lengthen([]float64{0.1, 0.2}, 0).Seconds)
			},
		},
		{
			// A rate so low that a hundredth of a second is less than one
			// sample. The length still answers; there are no frames to find
			// the sound inside.
			name: "a rate too low to cut into frames",
			then: func() {
				got := audio.Lengthen([]float64{0.5, 0.5, 0.5}, 50)

				s.Require().InDelta(0.06, got.Seconds, 0.001)
				s.Require().Zero(got.Head)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestLengthPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LengthPublicTestSuite))
}
