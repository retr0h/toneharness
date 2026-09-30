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
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// MeasuringPublicTestSuite covers pushing a signal through hardware and
// reading what came back.
//
// Without hardware. The bench is an interface for exactly this reason: what
// sits above it is arithmetic and ought to be testable without an audio
// interface, a cable and somebody in the room to plug them in.
type MeasuringPublicTestSuite struct {
	suite.Suite
}

// bench answers with whatever it was given.
type bench struct {
	back []float32
	err  error
}

func (b bench) Through(
	context.Context,
	[]float32,
) ([]float32, error) {
	return b.back, b.err
}

func (bench) Name() string { return "a bench" }

// tone is a signal loud enough to measure, at the given amplitude.
func tone(
	seconds float64,
	amplitude float64,
) []float32 {
	n := int(seconds * sdk.Rate)
	out := make([]float32, n)

	for i := range out {
		out[i] = float32(amplitude *
			math.Sin(2*math.Pi*110*float64(i)/sdk.Rate))
	}

	return out
}

// TestHear covers Hear, which plays a signal through the hardware and reports
// what came back.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuringPublicTestSuite) TestHear() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary case.
			name: "hear measures what came back",
			then: func() {
				back := tone(1, 0.5)

				read, got, err := sdk.Hear(context.Background(), bench{back: back}, back)

				s.Require().NoError(err)
				s.Require().Equal(back, got, "the samples travel with the reading")
				s.Require().Positive(read.Centroid)
				s.Require().InDelta(1, read.Low+read.Mid+read.High, 0.001,
					"the bands are shares of one whole")
			},
		},
		{
			// The bench failing.
			name: "hear reports hardware that would not answer",
			then: func() {
				_, _, err := sdk.Hear(context.Background(),
					bench{err: errors.New("stopped answering")}, tone(1, 0.5))

				s.Require().ErrorContains(err, "through a bench")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestFingerprint covers Fingerprint, which measures whatever is in front of
// the device, at its own defaults.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuringPublicTestSuite) TestFingerprint() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A bug worth a test.
			//
			// The level of the signal sent is a property of the file on disk
			// and is the same for every block. The level of what came back is
			// the one thing a volume control moves, and measuring the wrong
			// one reports every block as equally loud.
			name: "fingerprint measures the answer not the question",
			then: func() {
				sent := tone(1, 0.5)
				quieter := tone(1, 0.05)

				got, err := sdk.Fingerprint(context.Background(), bench{back: quieter}, sent)

				s.Require().NoError(err)
				s.Require().InDelta(sdk.Level(quieter), got.Level, 0.001)
				s.Require().Less(got.Level, sdk.Level(sent)-15,
					"a tenth of the amplitude is twenty decibels down")
			},
		},
		{
			// The bench failing.
			name: "fingerprint reports hardware that would not answer",
			then: func() {
				_, err := sdk.Fingerprint(context.Background(),
					bench{err: errors.New("stopped answering")}, tone(1, 0.5))

				s.Require().Error(err)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestLevelIsLoudness covers the figure a record does not carry.
func (s *MeasuringPublicTestSuite) TestLevelIsLoudness() {
	tests := []struct {
		name  string
		give  []float32
		want  float64
		delta float64
	}{
		{
			// Full scale sine: root mean square is one over root two,
			// which is about three decibels down.
			name: "a full scale tone", give: tone(1, 1), want: -3.01, delta: 0.1,
		},
		{name: "half of it", give: tone(1, 0.5), want: -9.03, delta: 0.1},
		{
			// A long way down rather than negative infinity, which is not a
			// number anything downstream can hold.
			name: "silence", give: make([]float32, 100), want: -999, delta: 0.001,
		},
		{name: "nothing at all", want: -999, delta: 0.001},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().InDelta(tt.want, sdk.Level(tt.give), tt.delta)
		})
	}
}

// TestFigures covers Figures, which turns a reading into the shape a measured
// library holds.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuringPublicTestSuite) TestFigures() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Transient and decay.
			//
			// A transient needs a note starting and a decay needs one ending,
			// so a reading can hold neither. Zero would be an answer; absent
			// is the truth.
			name: "figures carries what can be absent",
			then: func() {
				s.Run("present", func() {
					got := sdk.Figures(audio.Profile{
						Transient: audio.Reading{Value: 0.75, Known: true},
						Decay:     audio.Reading{Value: 1.5, Known: true},
					}, -20)

					s.Require().NotNil(got.Transient)
					s.Require().InDelta(0.75, *got.Transient, 0.001)
					s.Require().NotNil(got.Decay)
					s.Require().InDelta(1.5, *got.Decay, 0.001)
				})

				s.Run("absent", func() {
					got := sdk.Figures(audio.Profile{}, -20)

					s.Require().Nil(got.Transient)
					s.Require().Nil(got.Decay)
				})
			},
		},
		{
			// The conversion.
			//
			// A reading carries them from zero to one and every other number
			// here is a percentage, so one of the two has to move and this is
			// where.
			name: "figures reports shares as percentages",
			then: func() {
				got := sdk.Figures(audio.Profile{
					Low: 0.93, Mid: 0.07, High: 0.0001, Centroid: 138,
					Harmonics: audio.Spread{Low: 0.1, Mid: 0.2, High: 0.3},
					EvenOdd:   audio.Spread{Low: -0.6, Mid: 0.3, High: 0.9},
				}, -22.5)

				s.Require().InDelta(93, got.Low, 0.001)
				s.Require().InDelta(7, got.Mid, 0.001)
				s.Require().InDelta(138, got.Centroid, 0.001)
				s.Require().InDelta(-22.5, got.Level, 0.001)
				// Measuring answers all three, so each is set rather than left absent.
				s.Require().NotNil(got.Harmonics)
				s.Require().NotNil(got.Lean)
				s.Require().NotNil(got.Dynamics)

				s.Require().InDelta(20, *got.Harmonics, 0.001,
					"averaged across the three bands, because a block is one sound")
				s.Require().InDelta(0.2, *got.Lean, 0.001)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestMeasuringPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasuringPublicTestSuite))
}
