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
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// ChainTestSuite covers reading what is loaded, and the guard that says
// the reading is of the loop rather than of the chain.
type ChainTestSuite struct {
	suite.Suite

	dry string
}

func (s *ChainTestSuite) SetupTest() {
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di-short.wav")
}

// TestChain covers Chain, which reads what the pedal is playing, and changes
// nothing.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChainTestSuite) TestChain() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The primitive nothing provided.
			name: "it reads what is loaded and changes nothing",
			then: func() {
				w := buffer()

				s.Require().NoError(Chain(context.Background(), w, ChainOptions{
					Bench: bench{}, Dry: s.dry, Seconds: 0.1, Takes: 2,
				}))

				said := w.String()
				s.Require().Contains(said, "READS")
				s.Require().Contains(said, "WANDERS")
				s.Require().Contains(said, "below 250Hz")
				s.Require().Contains(said, "centroid")
			},
		},
		{
			// A missing signal.
			name: "a reference that is not there",
			then: func() {
				s.Require().Error(Chain(context.Background(), buffer(), ChainOptions{
					Bench: bench{}, Dry: filepath.Join(s.T().TempDir(), "nothing.wav"),
					Seconds: 0.1, Takes: 2,
				}))
			},
		},
		{
			// Both reads.
			//
			// The floor is measured first and the chain second, and a bench
			// that fails on either has nothing to report: a figure against a
			// floor nobody took is a figure about nothing.
			name: "chain reports a bench that stops answering",
			then: func() {
				for _, tt := range []struct {
					name  string
					after int
				}{
					{"on the floor", 0},
					{"after the floor, on the chain", 3},
				} {
					s.Run(tt.name, func() {
						var calls int

						err := Chain(context.Background(), buffer(), ChainOptions{
							Bench:   stopping{after: tt.after, calls: &calls},
							Dry:     s.dry,
							Seconds: 0.1, Takes: 2,
						})

						s.Require().Error(err)
					})
				}
			},
		},
		{
			// The warning rather than the refusal.
			//
			// This command exists to be pointed at a loop somebody is setting
			// up, so an oscillating one is what they want told about rather
			// than stopped for. Every other command refuses; this one prints
			// and carries on.
			name: "chain says so when the loop is squealing",
			then: func() {
				w := buffer()

				s.Require().NoError(Chain(context.Background(), w, ChainOptions{
					Bench: bench{clipped: true}, Dry: s.dry, Seconds: 0.1, Takes: 2,
					EndsInACab: true,
				}))

				s.Require().Contains(w.String(), "READS", "and it still prints the reading")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// TestSquealing covers every case Squealing answers.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChainTestSuite) TestSquealing() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The invariant.
			//
			// A cabinet is a loudspeaker and a loudspeaker is a low pass, so
			// a chain ending in one can remove energy above 2kHz and cannot
			// add it. Measured on an HX Stomp: a compressor into an Ampeg SVT
			// into an 8x10 gave back 84.3% of its energy above 2kHz against
			// 0.01% in the reference, and the same chain with the amplifier's
			// output at 0.5 gave back 0.0% at a level three decibels lower.
			name: "squealing catches a chain that brightened what it was given",
			then: func() {
				dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

				say, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.843}, dry, nil, true)

				s.Require().True(bad)
				s.Require().Contains(say, "84.3%")
				s.Require().Contains(say, "A loudspeaker cannot add that")
			},
		},
		{
			// The chain that is fine.
			name: "squealing leaves a clean reading alone",
			then: func() {
				dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

				_, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.0}, dry, nil, true)

				s.Require().False(bad, "a cabinet removing high end is a cabinet working")
			},
		},
		{
			// The limit of it.
			//
			// Only a chain ending in a loudspeaker makes the high band a
			// one-way street. Anything else may legitimately be brighter than
			// what it was given, and a drive certainly is.
			name: "squealing says nothing about a chain with no cabinet",
			then: func() {
				_, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.9},
					map[audio.Figure]float64{audio.KeyHigh: 0.0001}, nil, false)

				s.Require().False(bad)
			},
		},
		{
			// A figure nothing measured.
			name: "squealing needs both readings",
			then: func() {
				_, bad := Squealing(
					map[audio.Figure]float64{}, map[audio.Figure]float64{audio.KeyHigh: 0}, nil, true)
				s.Require().False(bad)

				_, bad = Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.9}, map[audio.Figure]float64{}, nil, true)
				s.Require().False(bad)
			},
		},
		{
			// The margin.
			//
			// The invariant is exact and the measurement is not. A chain with
			// 25dB of headroom reads 0.1% above 2kHz against the reference's
			// 0.01% at a level of -55dB, which is the converter's own
			// broadband floor and not a loudspeaker adding anything. Clean
			// readings sit between 0.0% and 0.1% and squealing ones between
			// 39% and 84%, with nothing in between.
			name: "a quiet clean reading is not a squeal",
			then: func() {
				dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

				_, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.001}, dry, nil, true)
				s.Require().False(bad, "a tenth of a percent is the floor")

				_, bad = Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.391}, dry, nil, true)
				s.Require().True(bad, "the quietest squeal measured was 39%")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// TestTheReferenceIsMeasuredRatherThanAssumed covers what the gap is against.
//
// The comparison is to this recording rather than to a number somebody chose,
// so a different reference moves the line with it.
func (s *ChainTestSuite) TestTheReferenceIsMeasuredRatherThanAssumed() {
	signal, err := reference(s.dry, 1)
	s.Require().NoError(err)

	got := figuresOfDry(signal)

	s.Require().Less(got[audio.KeyHigh], 0.01,
		"a bass DI has almost nothing above 2kHz")
	s.Require().Greater(got[audio.KeyLow], 0.5)
}

func TestChainTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ChainTestSuite))
}

// TestReferenceIsFor covers referenceIsFor, which is what a reference
// recording was played on.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChainTestSuite) TestReferenceIsFor() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// What a reading says it is about.
			//
			// Every figure here is a figure about one instrument. A guitar
			// rig ranked against readings taken with a bass is ranked against
			// the wrong distribution, and until this was recorded the only
			// trace of it was a filename.
			name: "instrument of reads it off the reference",
			then: func() {
				tests := []struct {
					dry  string
					want string
				}{
					{filepath.Join("resources", "dry", "bass-di-short.wav"), "bass"},
					{filepath.Join("resources", "dry", "guitar-di.wav"), "guitar"},
					{"BASS-DI.WAV", "bass"},
					{filepath.Join("somewhere", "else", "take-3.wav"), ""},
				}

				for _, tt := range tests {
					s.Run(tt.dry, func() {
						s.Require().Equal(tt.want, referenceIsFor(tt.dry))
					})
				}
			},
		},
		{
			// The honest empty.
			//
			// A reading that does not know what was played through it should
			// not say. The point of recording it is that somebody can tell,
			// and a guess defeats that.
			name: "a reading that cannot name its instrument claims none",
			then: func() {
				s.Require().Empty(referenceIsFor("reference.wav"))
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}
