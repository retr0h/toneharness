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

// ChainPublicTestSuite covers reading what is loaded, and the guard that says
// the reading is of the loop rather than of the chain.
type ChainPublicTestSuite struct {
	suite.Suite

	dry string
}

func (s *ChainPublicTestSuite) SetupTest() {
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di.wav")
}

// TestItReadsWhatIsLoadedAndChangesNothing is the primitive nothing provided.
func (s *ChainPublicTestSuite) TestItReadsWhatIsLoadedAndChangesNothing() {
	w := buffer()

	s.Require().NoError(Chain(context.Background(), w, ChainOptions{
		Bench: bench{}, Dry: s.dry, Seconds: 0.1, Takes: 2,
	}))

	said := w.String()
	s.Require().Contains(said, "READS")
	s.Require().Contains(said, "WANDERS")
	s.Require().Contains(said, "below 250Hz")
	s.Require().Contains(said, "centroid")
}

// TestAReferenceThatIsNotThere covers a missing signal.
func (s *ChainPublicTestSuite) TestAReferenceThatIsNotThere() {
	s.Require().Error(Chain(context.Background(), buffer(), ChainOptions{
		Bench: bench{}, Dry: filepath.Join(s.T().TempDir(), "nothing.wav"),
		Seconds: 0.1, Takes: 2,
	}))
}

// TestSquealingCatchesAChainThatBrightenedWhatItWasGiven is the invariant.
//
// A cabinet is a loudspeaker and a loudspeaker is a low pass, so a chain
// ending in one can remove energy above 2kHz and cannot add it. Measured on an
// HX Stomp: a compressor into an Ampeg SVT into an 8x10 gave back 84.3% of its
// energy above 2kHz against 0.01% in the reference, and the same chain with
// the amplifier's output at 0.5 gave back 0.0% at a level three decibels
// lower.
func (s *ChainPublicTestSuite) TestSquealingCatchesAChainThatBrightenedWhatItWasGiven() {
	dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

	say, bad := Squealing(
		map[audio.Figure]float64{audio.KeyHigh: 0.843}, dry, nil, true)

	s.Require().True(bad)
	s.Require().Contains(say, "84.3%")
	s.Require().Contains(say, "A loudspeaker cannot add that")
}

// TestSquealingLeavesACleanReadingAlone covers the chain that is fine.
func (s *ChainPublicTestSuite) TestSquealingLeavesACleanReadingAlone() {
	dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

	_, bad := Squealing(
		map[audio.Figure]float64{audio.KeyHigh: 0.0}, dry, nil, true)

	s.Require().False(bad, "a cabinet removing high end is a cabinet working")
}

// TestSquealingSaysNothingAboutAChainWithNoCabinet covers the limit of it.
//
// Only a chain ending in a loudspeaker makes the high band a one-way street.
// Anything else may legitimately be brighter than what it was given, and a
// drive certainly is.
func (s *ChainPublicTestSuite) TestSquealingSaysNothingAboutAChainWithNoCabinet() {
	_, bad := Squealing(
		map[audio.Figure]float64{audio.KeyHigh: 0.9},
		map[audio.Figure]float64{audio.KeyHigh: 0.0001}, nil, false)

	s.Require().False(bad)
}

// TestSquealingNeedsBothReadings covers a figure nothing measured.
func (s *ChainPublicTestSuite) TestSquealingNeedsBothReadings() {
	_, bad := Squealing(
		map[audio.Figure]float64{}, map[audio.Figure]float64{audio.KeyHigh: 0}, nil, true)
	s.Require().False(bad)

	_, bad = Squealing(
		map[audio.Figure]float64{audio.KeyHigh: 0.9}, map[audio.Figure]float64{}, nil, true)
	s.Require().False(bad)
}

// TestTheReferenceIsMeasuredRatherThanAssumed covers what the gap is against.
//
// The comparison is to this recording rather than to a number somebody chose,
// so a different reference moves the line with it.
func (s *ChainPublicTestSuite) TestTheReferenceIsMeasuredRatherThanAssumed() {
	signal, err := reference(s.dry, 1)
	s.Require().NoError(err)

	got := figuresOfDry(signal)

	s.Require().Less(got[audio.KeyHigh], 0.01,
		"a bass DI has almost nothing above 2kHz")
	s.Require().Greater(got[audio.KeyLow], 0.5)
}

// TestAQuietCleanReadingIsNotASqueal covers the margin.
//
// The invariant is exact and the measurement is not. A chain with 25dB of
// headroom reads 0.1% above 2kHz against the reference's 0.01% at a level of
// -55dB, which is the converter's own broadband floor and not a loudspeaker
// adding anything. Clean readings sit between 0.0% and 0.1% and squealing ones
// between 39% and 84%, with nothing in between.
func (s *ChainPublicTestSuite) TestAQuietCleanReadingIsNotASqueal() {
	dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

	_, bad := Squealing(
		map[audio.Figure]float64{audio.KeyHigh: 0.001}, dry, nil, true)
	s.Require().False(bad, "a tenth of a percent is the floor")

	_, bad = Squealing(
		map[audio.Figure]float64{audio.KeyHigh: 0.391}, dry, nil, true)
	s.Require().True(bad, "the quietest squeal measured was 39%")
}

func TestChainPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ChainPublicTestSuite))
}

// TestInstrumentOfReadsItOffTheReference covers what a reading says it is about.
//
// Every figure here is a figure about one instrument. A guitar rig ranked
// against readings taken with a bass is ranked against the wrong distribution,
// and until this was recorded the only trace of it was a filename.
func (s *ChainPublicTestSuite) TestInstrumentOfReadsItOffTheReference() {
	tests := []struct {
		dry  string
		want string
	}{
		{filepath.Join("resources", "dry", "bass-di.wav"), "bass"},
		{filepath.Join("resources", "dry", "guitar-di.wav"), "guitar"},
		{"BASS-DI.WAV", "bass"},
		{filepath.Join("somewhere", "else", "take-3.wav"), ""},
	}

	for _, tt := range tests {
		s.Run(tt.dry, func() {
			s.Require().Equal(tt.want, instrumentOf(tt.dry))
		})
	}
}

// TestAReadingThatCannotNameItsInstrumentClaimsNone covers the honest empty.
//
// A reading that does not know what was played through it should not say. The
// point of recording it is that somebody can tell, and a guess defeats that.
func (s *ChainPublicTestSuite) TestAReadingThatCannotNameItsInstrumentClaimsNone() {
	s.Require().Empty(instrumentOf("reference.wav"))
}
