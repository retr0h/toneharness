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

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// InstrumentPublicTestSuite covers refusing to measure a chain through the
// wrong reference.
//
// The models are named rather than invented, because the claim rests on Line 6's
// own Guitar and Bass tags in the catalog: a fixture asserting its own
// subcategory would pass while the catalog said something else.
type InstrumentPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *InstrumentPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// chain is a plan holding one amplifier.
func (s *InstrumentPublicTestSuite) chain(
	model string,
) plan.Plan {
	return plan.Plan{Blocks: []plan.Block{{
		Model: catalog.ModelID(model), Pos: 0, Enabled: true,
	}}}
}

// TestTheCatalogsOwnTagsDecideWhatAChainIsFor is the claim underneath the guard.
func (s *InstrumentPublicTestSuite) TestTheCatalogsOwnTagsDecideWhatAChainIsFor() {
	bass := s.cat.Blocks["HD2_AmpSVBeastNrm"]
	s.Require().Equal(catalog.CategoryAmp, bass.Category)
	s.Require().Equal("Bass", bass.Subcategory,
		"Line 6 tag this one Bass, which is what the guard reads")

	s.Require().Equal("bass", chainIsFor(s.chain("HD2_AmpSVBeastNrm"), s.cat))
}

// TestABassAmpAnywhereDecidesIt is the divergence that would have refused a
// rig for the instrument it says it is.
//
// The compiler scans every amplifier and answers bass if any one of them is a
// bass amplifier, so that is the rig's own `instrument` field. Returning on the
// first amplifier found reads a guitar-amp-then-bass-amp chain as guitar, and
// then this guard refuses a bass rig pushed a bass recording. The two rules have
// to be the same rule.
func (s *InstrumentPublicTestSuite) TestABassAmpAnywhereDecidesIt() {
	// A guitar amplifier first, a bass amplifier after it.
	both := plan.Plan{Blocks: []plan.Block{
		{Model: catalog.ModelID("HD2_AmpBrit2203"), Pos: 0, Enabled: true},
		{Model: catalog.ModelID("HD2_AmpSVBeastNrm"), Pos: 1, Enabled: true},
	}}

	guitar := s.cat.Blocks["HD2_AmpBrit2203"]
	s.Require().Equal(catalog.CategoryAmp, guitar.Category)
	s.Require().NotEqual("Bass", guitar.Subcategory,
		"the first amplifier here is not a bass amplifier")

	s.Require().Equal("bass", chainIsFor(both, s.cat))

	s.Require().NoError(sameInstrument(both, s.cat, "resources/dry/bass-di.wav"),
		"and a bass reference through it is not refused")
}

// TestAGuitarOnlyChainStillAnswersGuitar covers not over-correcting.
func (s *InstrumentPublicTestSuite) TestAGuitarOnlyChainStillAnswersGuitar() {
	s.Require().Equal("guitar", chainIsFor(s.chain("HD2_AmpBrit2203"), s.cat))
}

// TestAChainWithNoAmplifierNamesNoInstrument covers claiming nothing.
//
// An effect serves either instrument, so there is no disagreement to find and
// refusing one would stop somebody measuring a delay.
func (s *InstrumentPublicTestSuite) TestAChainWithNoAmplifierNamesNoInstrument() {
	s.Require().Empty(chainIsFor(s.chain("HD2_CabMicIr_2x15Brute"), s.cat))
	s.Require().Empty(chainIsFor(plan.Plan{}, s.cat))

	s.Require().NoError(sameInstrument(
		s.chain("HD2_CabMicIr_2x15Brute"), s.cat, "resources/dry/guitar-di.wav"),
		"nothing known, nothing refused")
}

// TestABassChainThroughABassReferenceIsFine is the ordinary case.
func (s *InstrumentPublicTestSuite) TestABassChainThroughABassReferenceIsFine() {
	s.Require().NoError(sameInstrument(
		s.chain("HD2_AmpSVBeastNrm"), s.cat, "resources/dry/bass-di.wav"))
}

// TestABassChainThroughAGuitarReferenceIsRefused is the whole point.
//
// Refused rather than reported, which is the opposite of how a squeal or a
// missing slope is handled. Those make one figure doubtful and the run still
// says something. This makes every figure an answer about the reference rather
// than the chain, and five minutes of measuring would report tolerances met
// against a target the recording invented.
func (s *InstrumentPublicTestSuite) TestABassChainThroughAGuitarReferenceIsRefused() {
	err := sameInstrument(
		s.chain("HD2_AmpSVBeastNrm"), s.cat, "resources/dry/guitar-di.wav")

	s.Require().ErrorIs(err, ErrWrongInstrument)
	s.Require().ErrorContains(err, "bass amplifier")
	s.Require().ErrorContains(err, "guitar recording")
	s.Require().ErrorContains(err, "--dry",
		"it says what to do, not only that it will not")
}

// TestAReferenceNamingNeitherIsNotRefused covers somebody's own recording.
//
// A file called `di.wav` or `take-3.wav` is not a claim about an instrument, and
// refusing it would be the tool inventing a disagreement out of a filename.
func (s *InstrumentPublicTestSuite) TestAReferenceNamingNeitherIsNotRefused() {
	s.Require().NoError(sameInstrument(
		s.chain("HD2_AmpSVBeastNrm"), s.cat, "take-3.wav"))
}

func TestInstrumentPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InstrumentPublicTestSuite))
}
