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

// EvidencePublicTestSuite covers naming a measurement for a rig.
type EvidencePublicTestSuite struct {
	suite.Suite
}

// keys is every name a measurement is written under.
func (s *EvidencePublicTestSuite) keys() []audio.Figure {
	return audio.MeasuredKeys()
}

// TestEveryKeyIsMeasured covers the order naming exactly what is produced.
//
// A key in the order that nothing measures would write an absent figure into
// a rig; a key measured but missing from the order would be dropped silently.
//
// Measured against a recording that answered everything. A profile that did
// not answer writes fewer keys, which is the point of the two below.
func (s *EvidencePublicTestSuite) TestEveryKeyIsMeasured() {
	got := audio.Profile{
		Transient: audio.Reading{Known: true},
		Decay:     audio.Reading{Known: true},
	}.Measured()

	s.Require().Len(audio.MeasuredKeys(), len(got))

	for _, key := range audio.MeasuredKeys() {
		s.Require().Contains(got, string(key))
	}
}

// TestSpreadsCarriesTheSameGuardMeasuredDoes is the other side of Measured.
//
// Measured writes what one recording answered; this writes what a population
// of them agreed about, and a target's tolerances come from it. Both leave out
// a figure nobody could take rather than writing the number it would have
// been: a spread of zero on an axis nothing measured reads as total agreement,
// which is the tightest tolerance there is and the one a solver will spend
// every control defending.
func (s *EvidencePublicTestSuite) TestSpreadsCarriesTheSameGuardMeasuredDoes() {
	answered := audio.Across{
		Low:       audio.Spread{Low: 0.2, Mid: 0.3, High: 0.4},
		Transient: audio.Ranged{From: 4, Spread: audio.Spread{Mid: 0.5}},
		Decay:     audio.Ranged{From: 2, Spread: audio.Spread{Mid: 2.9}},
	}.Spreads()

	// The seven every recording answers, and the two that need a note
	// starting and a note ending.
	s.Require().Len(answered, len(audio.MeasuredKeys()))

	for _, key := range audio.MeasuredKeys() {
		s.Require().Contains(answered, string(key))
	}

	s.Require().InDelta(0.3, answered[string(audio.KeyLow)].Mid, 0.001)
	s.Require().InDelta(0.5, answered[string(audio.KeyTransient)].Mid, 0.001)
	s.Require().InDelta(2.9, answered[string(audio.KeyDecay)].Mid, 0.001)

	// And a population where no record answered either.
	silent := audio.Across{Low: audio.Spread{Mid: 0.3}}.Spreads()

	s.Require().NotContains(silent, string(audio.KeyTransient),
		"no record's notes started, so there is nothing to agree about")
	s.Require().NotContains(silent, string(audio.KeyDecay))
	s.Require().Contains(silent, string(audio.KeyLow), "the rest still arrives")
}

// TestOnlyTheBandsAndHarmonicsAreShares pins the set every scale conversion
// reads.
//
// A sweep reports a share as a percentage and a corpus as a fraction, so this
// answer decides which figures get divided by a hundred on the way between
// them. A figure wrongly named a share is that figure a hundred times out, and
// one wrongly left off is the same error the other way, in silence.
func (s *EvidencePublicTestSuite) TestOnlyTheBandsAndHarmonicsAreShares() {
	shares := []audio.Figure{
		audio.KeyLow, audio.KeyMid, audio.KeyHigh, audio.KeyHarmonics,
	}

	for _, key := range shares {
		s.Require().True(key.Share(), string(key))
	}

	rest := []audio.Figure{
		audio.KeyCentroid, audio.KeyTransient, audio.KeyDecay,
		audio.KeyDynamics, audio.KeyLean, audio.KeyLevel,
	}

	for _, key := range rest {
		s.Require().False(key.Share(), string(key))
	}

	s.Require().Len(audio.MeasuredKeys(), len(shares)+len(rest)-1,
		"Level is the one figure outside MeasuredKeys; anything else added to "+
			"the alphabet has to be answered for here too")
}

// TestAMeasureNobodyCouldTakeIsLeftOut is the difference between a figure of
// zero and no figure.
//
// An earlier version wrote all nine keys always, so a tone that never decayed
// carried `decay: 3.0` into a rig and was compared against a preset's real
// decay as though somebody had measured it.
func (s *EvidencePublicTestSuite) TestAMeasureNobodyCouldTakeIsLeftOut() {
	got := audio.Profile{Centroid: 175}.Measured()

	s.Require().NotContains(got, string(audio.KeyDecay))
	s.Require().NotContains(got, string(audio.KeyTransient))
	s.Require().Contains(got, string(audio.KeyCentroid), "the rest still arrives")
}

// TestAGatheredMeasureThatWasTakenIsWritten covers the other side of it.
//
// The middle of the records that answered, written under the same key a
// single recording uses. Three so the middle is the middle rather than an
// end.
func (s *EvidencePublicTestSuite) TestAGatheredMeasureThatWasTakenIsWritten() {
	got := audio.Together([]audio.Profile{
		{
			Centroid:  175,
			Transient: audio.Reading{Value: 0.70, Known: true},
			Decay:     audio.Reading{Value: 0.80, Known: true},
		},
		{
			Centroid:  180,
			Transient: audio.Reading{Value: 0.74, Known: true},
			Decay:     audio.Reading{Value: 0.82, Known: true},
		},
		{
			Centroid:  185,
			Transient: audio.Reading{Value: 0.78, Known: true},
			Decay:     audio.Reading{Value: 0.90, Known: true},
		},
	}).Measured()

	s.Require().InDelta(0.74, got[string(audio.KeyTransient)], 1e-9)
	s.Require().InDelta(0.82, got[string(audio.KeyDecay)], 1e-9)

	s.Require().Len(got, len(audio.MeasuredKeys()))
}

// TestAGatheredMeasureNobodyCouldTakeIsLeftOut covers the same across records.
func (s *EvidencePublicTestSuite) TestAGatheredMeasureNobodyCouldTakeIsLeftOut() {
	got := audio.Together([]audio.Profile{{Centroid: 175}, {Centroid: 185}}).
		Measured()

	s.Require().NotContains(got, string(audio.KeyDecay))
	s.Require().NotContains(got, string(audio.KeyTransient))
	s.Require().Contains(got, string(audio.KeyCentroid))
}

// TestTheOrderIsStable covers evidence written twice reading the same way.
func (s *EvidencePublicTestSuite) TestTheOrderIsStable() {
	s.Require().Equal(audio.MeasuredKeys(), audio.MeasuredKeys())
}

// TestOneRecordingIsNamedInFull covers a profile writing every key.
func (s *EvidencePublicTestSuite) TestOneRecordingIsNamedInFull() {
	got := audio.Profile{
		Low: 0.9134, Mid: 0.0812, High: 0.0054,
		Centroid:     175.4,
		Transient:    audio.Reading{Value: 0.7431, Known: true},
		Decay:        audio.Reading{Value: 0.8249, Known: true},
		DynamicRange: 7.84,
		Harmonics:    audio.Spread{Mid: 0.3512},
		EvenOdd:      audio.Spread{Mid: -0.6049},
	}.Measured()

	for _, key := range s.keys() {
		s.Require().Contains(got, string(key))
	}

	s.Require().Len(got, len(s.keys()), "and nothing nobody measures")
}

// TestGatheredRecordingsAreNamedTheSameWay covers the two agreeing.
//
// A rig cannot compare a figure from one recording against a figure from four
// unless both are written under the same name.
func (s *EvidencePublicTestSuite) TestGatheredRecordingsAreNamedTheSameWay() {
	one := audio.Profile{Centroid: 175}.Measured()
	many := audio.Together([]audio.Profile{{Centroid: 175}}).Measured()

	s.Require().Equal(len(one), len(many))

	for key := range one {
		s.Require().Contains(many, key, "%s is named by one and not the other", key)
	}
}

// TestFiguresAreRoundedToWhatTheyCanClaim covers the numbers being readable.
//
// The bins are about 10Hz apart, so a centroid to six places claims a
// precision the transform does not have.
func (s *EvidencePublicTestSuite) TestFiguresAreRoundedToWhatTheyCanClaim() {
	got := audio.Profile{
		Low:          0.913471,
		Centroid:     175.4382,
		Decay:        audio.Reading{Value: 0.824913, Known: true},
		DynamicRange: 7.8449,
	}.Measured()

	s.Require().InDelta(0.91, got[string(audio.KeyLow)], 1e-9)
	s.Require().InDelta(175, got[string(audio.KeyCentroid)], 1e-9)
	s.Require().InDelta(0.82, got[string(audio.KeyDecay)], 1e-9)
	s.Require().InDelta(7.8, got[string(audio.KeyDynamics)], 1e-9)
}

// TestTheMiddleIsWhatIsCarried covers a gathered measurement writing the
// middle rather than an end.
func (s *EvidencePublicTestSuite) TestTheMiddleIsWhatIsCarried() {
	got := audio.Together([]audio.Profile{
		{Centroid: 135}, {Centroid: 175}, {Centroid: 195},
	}).Measured()

	s.Require().InDelta(175, got[string(audio.KeyCentroid)], 1e-9)
}

// TestNothingMeasuredNamesOnlyWhatItCould covers an empty measurement.
//
// This test used to assert the opposite, that all nine keys are always
// written, on the reasoning that a rig could then tell a figure of zero from a
// figure nobody took. That had it backwards. The absent key is what tells them
// apart, and writing a number for a measure nobody could take is how a
// generated tone came to carry a decay of three seconds into a rig.
func (s *EvidencePublicTestSuite) TestNothingMeasuredNamesOnlyWhatItCould() {
	got := audio.Across{}.Measured()

	for _, key := range []audio.Figure{
		audio.KeyLow, audio.KeyMid, audio.KeyHigh, audio.KeyCentroid,
		audio.KeyDynamics, audio.KeyHarmonics, audio.KeyLean,
	} {
		s.Require().Contains(got, string(key), "%s is a real zero", key)
	}

	s.Require().NotContains(got, string(audio.KeyDecay))
	s.Require().NotContains(got, string(audio.KeyTransient))

	s.Require().Len(got, len(audio.MeasuredKeys())-2)
}

func TestEvidencePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EvidencePublicTestSuite))
}
