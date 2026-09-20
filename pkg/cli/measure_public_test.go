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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// MeasureTestSuite covers the bookkeeping a measuring campaign is mostly made
// of, without the hardware it is nominally about.
type MeasureTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *MeasureTestSuite) SetupSuite() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	s.cat = cat
}

// TestWantedSkipsWhatIsNotABlock covers the four catalog entries that are not.
//
// `@dt`, `@global_params`, `@powercab` and `@variax` are where a preset keeps
// settings about the device rather than anything in the signal path, and
// carry no name because nobody puts one in a chain. Attempted, they land in
// the output as failures and make the refusal list a mix of blocks that would
// not load and things that were never blocks.
func (s *MeasureTestSuite) TestWantedSkipsWhatIsNotABlock() {
	got := Wanted(s.cat, "", nil)

	for _, block := range got {
		s.Require().NotEmpty(block.Name, "%s has no name", block.ID)
		s.Require().NotContains(block.ID, "@")
	}

	s.Require().Len(got, len(s.cat.Blocks)-4)
}

// TestWantedNarrowsToOneKind covers --category.
func (s *MeasureTestSuite) TestWantedNarrowsToOneKind() {
	got := Wanted(s.cat, "amp", nil)

	s.Require().NotEmpty(got)

	for _, block := range got {
		s.Require().Equal(catalog.CategoryAmp, block.Category)
	}
}

// TestWantedIsStable covers two runs agreeing on the order.
//
// A campaign that stops halfway and resumes has to pick up where it left off,
// which it cannot do if the list reshuffles between runs.
func (s *MeasureTestSuite) TestWantedIsStable() {
	first := Wanted(s.cat, "", nil)

	for range 5 {
		s.Require().Equal(first, Wanted(s.cat, "", nil))
	}
}

// TestWantedLeavesOutWhatIsDone covers resuming.
func (s *MeasureTestSuite) TestWantedLeavesOutWhatIsDone() {
	all := Wanted(s.cat, "amp", nil)
	done := map[string]measured.Block{all[0].ID: {ID: all[0].ID}}

	got := Wanted(s.cat, "amp", done)

	s.Require().Len(got, len(all)-1)
	s.Require().NotEqual(all[0].ID, got[0].ID)
}

// TestRigForQuotesWhatWouldBreakTheDocument is the bug this cost.
//
// A reverb called "'63 Spring" opens a YAML quote that nothing closes, and
// the document fails to parse at a line nowhere near the name. One block of
// six hundred and sixty one was lost to it, so the rig is built as the
// contract's own type and written by its own writer rather than as text.
func (s *MeasureTestSuite) TestRigForQuotesWhatWouldBreakTheDocument() {
	spec := RigFor(measured.Block{
		ID: "HD2_Reverb63Spring", Name: "'63 Spring", Category: "reverb",
	}, true)

	var buf bytes.Buffer
	s.Require().NoError(rig.Write(&buf, spec))

	back, err := rig.Load(&buf)
	s.Require().NoError(err)
	s.Require().Equal("'63 Spring", back.Chain[0].Gear)
	s.Require().Equal("HD2_Reverb63Spring", (*back.Chain[0].Models)["HX Stomp"])
}

// TestRigForHoldsOneBlockAndNothingElse is what isolation means.
func (s *MeasureTestSuite) TestRigForHoldsOneBlockAndNothingElse() {
	spec := RigFor(measured.Block{
		ID: "HD2_AmpSVBeastNrm", Name: "Ampeg SVT", Category: "amp",
	}, true)

	s.Require().Len(spec.Chain, 1)
	s.Require().Nil(spec.Chain[0].Enabled, "switched on unless it is the baseline")
	s.Require().NoError(rig.Validate(spec))
}

// TestRigForCanBeBypassed covers the baseline.
//
// One bypassed block rather than none, because a chain has a minimum of one
// item and a rig with nothing in it is not a rig. Bypassed is the same signal
// path either way.
func (s *MeasureTestSuite) TestRigForCanBeBypassed() {
	spec := RigFor(measured.Block{
		ID: "HD2_EQSimple3Band", Name: "Simple EQ", Category: "eq",
	}, false)

	s.Require().NotNil(spec.Chain[0].Enabled)
	s.Require().False(*spec.Chain[0].Enabled)
	s.Require().NoError(rig.Validate(spec))
}

// TestEveryBlockCompilesIntoARigThatValidates is the check that would have
// caught the lost reverb before a campaign spent ninety five minutes.
func (s *MeasureTestSuite) TestEveryBlockCompilesIntoARigThatValidates() {
	for _, block := range Wanted(s.cat, "", nil) {
		spec := RigFor(block, true)

		s.Require().NoErrorf(rig.Validate(spec),
			"%s (%q) does not make a valid rig", block.ID, block.Name)

		var buf bytes.Buffer
		s.Require().NoErrorf(rig.Write(&buf, spec), "%s will not write", block.ID)

		_, err := rig.Load(&buf)
		s.Require().NoErrorf(err, "%s writes a rig it cannot read back", block.ID)
	}
}

// TestResumeKeepsWhatWasMeasured covers picking a run back up.
func (s *MeasureTestSuite) TestResumeKeepsWhatWasMeasured() {
	at := s.libraryFile(`{"device":"HX Stomp","isolated":true,
	  "baseline":{"centroid":94.8},
	  "blocks":{
	    "A":{"id":"A","category":"amp","centroid":300},
	    "B":{"id":"B","category":"amp","refused":"the device said no"}}}`)

	lib := measured.Library{Blocks: map[string]measured.Block{}}
	Resume(&lib, MeasureOptions{Out: at})

	s.Require().Len(lib.Blocks, 2)
	s.Require().InDelta(94.8, lib.Baseline.Centroid, 0.01,
		"the baseline is resumed too, or every later block is a difference "+
			"from nothing")
}

// TestResumeCanTryTheRefusalsAgain covers --retry.
//
// About one block in eighty comes back saying the device stopped taking the
// message, which is the chunk pacing giving up rather than anything about
// that block: the same one loads on the next pass.
func (s *MeasureTestSuite) TestResumeCanTryTheRefusalsAgain() {
	at := s.libraryFile(`{"device":"HX Stomp","isolated":true,
	  "blocks":{
	    "A":{"id":"A","category":"amp","centroid":300},
	    "B":{"id":"B","category":"amp","refused":"the device said no"}}}`)

	lib := measured.Library{Blocks: map[string]measured.Block{}}
	Resume(&lib, MeasureOptions{Out: at, Retry: true})

	s.Require().Len(lib.Blocks, 1)
	s.Require().Contains(lib.Blocks, "A")
	s.Require().NotContains(lib.Blocks, "B", "the refusal is tried again")
}

// TestResumeOnNothingToResumeFrom covers the first run.
func (s *MeasureTestSuite) TestResumeOnNothingToResumeFrom() {
	for _, tt := range []struct{ name, body string }{
		{name: "no file at all"},
		{name: "a file that is not a library", body: "{"},
	} {
		s.Run(tt.name, func() {
			at := filepath.Join(s.T().TempDir(), "none.json")
			if tt.body != "" {
				at = s.libraryFile(tt.body)
			}

			lib := measured.Library{Blocks: map[string]measured.Block{}}
			Resume(&lib, MeasureOptions{Out: at})

			s.Require().Empty(lib.Blocks)
		})
	}
}

// TestSweepableTakesDialsAndListsAndNotSwitches covers control selection.
//
// Two positions is not a curve, so a switch is left out. A list is not, even
// though it has no slope, because which of twelve microphones sits in front
// of a speaker changes a cabinet more than any of its knobs.
func (s *MeasureTestSuite) TestSweepableTakesDialsAndListsAndNotSwitches() {
	block := s.cat.Blocks["HD2_CabMicIr_2x15Brute"]
	got := Sweepable(block, WireOrder(s.cat, "HD2_CabMicIr_2x15Brute"))

	kinds := make([]string, 0, len(got))
	for _, c := range got {
		kinds = append(kinds, c.kind)
	}

	s.Require().Contains(kinds, "int", "the microphone is a list")
	s.Require().Contains(kinds, "float")
	s.Require().NotContains(kinds, "bool")
}

// TestSweepableTakesTheCatalogsRange is why a range is not assumed.
//
// 1,452 of the device's 4,835 float controls do not run zero to one. A Simple
// EQ's Mid Freq runs 125 to 4000 Hz, and swept 0..1 it never leaves its
// bottom stop and reports as a control that does nothing.
func (s *MeasureTestSuite) TestSweepableTakesTheCatalogsRange() {
	block := s.cat.Blocks["HD2_CabMicIr_2x15Brute"]

	for _, c := range Sweepable(block, WireOrder(s.cat, "HD2_CabMicIr_2x15Brute")) {
		spec := block.Params[c.name]

		s.Require().InDelta(spec.Min, c.low, 0.001, c.name)
		s.Require().InDelta(spec.Max, c.high, 0.001, c.name)
	}
}

// TestWireOrderIsNotTheSortedOne is the mislabelling this prevents.
//
// `catalog show` prints a model's parameters sorted for a reader. The device
// sends them in its own order, and counting down the printed one files every
// curve under the wrong control while the numbers stay plausible.
func (s *MeasureTestSuite) TestWireOrderIsNotTheSortedOne() {
	got := WireOrder(s.cat, "HD2_AmpUSDripmanNorm")

	s.Require().Equal(
		[]string{"Norm Drive", "Bass", "Mid", "Treble"}, got[:4])
	s.Require().Empty(WireOrder(s.cat, "nothing calls itself this"))
}

// TestFillGivesADialASlopeAndAListASpread covers the split.
func (s *MeasureTestSuite) TestFillGivesADialASlopeAndAListASpread() {
	points := []measured.Point{
		{Value: 0, Figures: measured.Figures{Centroid: 100}},
		{Value: 1, Figures: measured.Figures{Centroid: 200}},
	}

	dial := measured.Curve{Kind: "float", Points: points}
	Fill(&dial)

	s.Require().NotEmpty(dial.Fits)
	s.Require().Nil(dial.Spread)
	s.Require().InDelta(100, dial.Fits["centroid"].PerTurn, 0.001)

	list := measured.Curve{Kind: "int", Points: points}
	Fill(&list)

	s.Require().NotEmpty(list.Spread)
	s.Require().Nil(list.Fits, "a list of settings has no slope")
	s.Require().InDelta(100, list.Spread["centroid"], 0.001)
}

// TestResampleTakesTheAskedForLength covers the rate change.
func (s *MeasureTestSuite) TestResampleTakesTheAskedForLength() {
	samples := make([]float64, 44100*3)
	for i := range samples {
		samples[i] = 0.5
	}

	got := Resample(samples, 44100, 1)

	s.Require().Len(got, 48000)
	s.Require().InDelta(0.5, float64(got[100]), 0.001)
}

// TestResampleStopsAtTheEndOfWhatItHas covers asking for more than exists.
func (s *MeasureTestSuite) TestResampleStopsAtTheEndOfWhatItHas() {
	got := Resample(make([]float64, 100), 44100, 10)

	s.Require().Less(len(got), 48000*10)
}

// TestReferenceAndHashReportAFileThatIsNotThere covers a missing signal.
func (s *MeasureTestSuite) TestReferenceAndHashReportAFileThatIsNotThere() {
	_, err := Reference(filepath.Join(s.T().TempDir(), "none.wav"), 4)
	s.Require().ErrorContains(err, "reading")

	_, err = Hash(filepath.Join(s.T().TempDir(), "none.wav"))
	s.Require().ErrorContains(err, "reading")
}

// TestReferenceRefusesWhatIsNotAudio covers a file that is not a WAV.
func (s *MeasureTestSuite) TestReferenceRefusesWhatIsNotAudio() {
	at := filepath.Join(s.T().TempDir(), "not.wav")
	s.Require().NoError(os.WriteFile(at, []byte("not audio"), 0o600))

	_, err := Reference(at, 4)
	s.Require().Error(err)
}

// TestChangedNamesWhatMoved covers reading a probe's answer.
func (s *MeasureTestSuite) TestChangedNamesWhatMoved() {
	before := map[string]any{"Bass": 0.5, "Treble": 0.5}
	after := map[string]any{"Bass": 0.877, "Treble": 0.5}

	s.Require().Equal([]string{"Bass"}, Changed(before, after))
	s.Require().Empty(Changed(before, before))
}

// TestShortIsTheFirstLine covers a table with one column for an error.
func (s *MeasureTestSuite) TestShortIsTheFirstLine() {
	s.Require().Equal("the device said no",
		Short(errors.New("the device said no\nand then some detail")))
}

// TestKeepReportsSomewhereItCannotWrite covers a bad path.
func (s *MeasureTestSuite) TestKeepReportsSomewhereItCannotWrite() {
	err := Keep(filepath.Join(s.T().TempDir(), "no", "such", "dir.json"),
		measured.Library{})

	s.Require().Error(err)
}

// libraryFile writes a library and returns where it went.
func (s *MeasureTestSuite) libraryFile(
	body string,
) string {
	at := filepath.Join(s.T().TempDir(), "library.json")
	s.Require().NoError(os.WriteFile(at, []byte(body), 0o600))

	return at
}

func TestMeasureTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasureTestSuite))
}
