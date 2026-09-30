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

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
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

	// Every block, because the catalog no longer holds anything that is not
	// one. The four device attributes are dropped where they are read rather
	// than here, so this counts what the catalog has instead of subtracting
	// them again.
	s.Require().Len(got, len(s.cat.Blocks))
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

// TestPlanForWritesADocumentThatParses is the bug this cost.
//
// A reverb called "'63 Spring" opens a YAML quote that nothing closes, and
// the document fails to parse at a line nowhere near the name. One block of
// six hundred and sixty one was lost to it, so the plan is built as the type
// and written by its own writer rather than as text.
func (s *MeasureTestSuite) TestPlanForWritesADocumentThatParses() {
	spec := PlanFor(measured.Block{
		ID: "HD2_Reverb63Spring", Name: "'63 Spring", Category: "reverb",
	}, true)

	var buf bytes.Buffer
	s.Require().NoError(plan.Write(&buf, spec))

	back, err := plan.Load(&buf)
	s.Require().NoError(err)
	s.Require().Equal("measure-hd2-reverb63spring", back.Name)
	s.Require().Equal(catalog.ModelID("HD2_Reverb63Spring"), back.Blocks[0].Model)
}

// TestPlanForPinsOneModel is why a sweep writes a plan and not a rig.
//
// 661 models share 468 names, so "Ampeg SVT" matches both of its channels and
// a rig naming the gear would measure whichever the compiler picked.
func (s *MeasureTestSuite) TestPlanForPinsOneModel() {
	spec := PlanFor(measured.Block{
		ID: "HD2_AmpSVBeastNrm", Name: "Ampeg SVT", Category: "amp",
	}, true)

	s.Require().Len(spec.Blocks, 1)
	s.Require().Equal(catalog.ModelID("HD2_AmpSVBeastNrm"), spec.Blocks[0].Model)
	s.Require().True(spec.Blocks[0].Enabled,
		"switched on unless it is the baseline")
	s.Require().NoError(plan.Validate(s.cat, spec, plan.HXStompLimits()))
}

// TestPlanForCanBeBypassed covers the baseline.
//
// One bypassed block rather than none, because a chain has a minimum of one
// item and a plan with nothing in it is not a plan. Bypassed is the same
// signal path either way.
func (s *MeasureTestSuite) TestPlanForCanBeBypassed() {
	spec := PlanFor(measured.Block{
		ID: "HD2_EQSimple3Band", Name: "Simple EQ", Category: "eq",
	}, false)

	s.Require().False(spec.Blocks[0].Enabled)
	s.Require().NoError(plan.Validate(s.cat, spec, plan.HXStompLimits()))
}

// TestEveryBlockCompilesIntoAPlanThatValidates is the check that would have
// caught the lost reverb before a campaign spent ninety five minutes.
//
// The whole of Validate, budget included. It was narrowed to
// ValidateStructure while 29 blocks carried an assumed DSP cost the budget
// layer refuses outright, which got the tree green and hid the thing worth
// knowing: those costs were in Line 6's own file and the generator was
// throwing them away.
func (s *MeasureTestSuite) TestEveryBlockCompilesIntoAPlanThatValidates() {
	for _, block := range Wanted(s.cat, "", nil) {
		spec := PlanFor(block, true)

		s.Require().NoErrorf(
			plan.Validate(s.cat, spec, plan.LimitsFor(s.cat.Device)),
			"%s (%q) does not make a valid plan", block.ID, block.Name)

		var buf bytes.Buffer
		s.Require().NoErrorf(plan.Write(&buf, spec), "%s will not write", block.ID)

		_, err := plan.Load(&buf)
		s.Require().NoErrorf(err, "%s writes a plan it cannot read back", block.ID)
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

// TestSweepableTakesDialsAndLists covers control selection.
//
// A list is taken even though it has no slope, because which of twelve
// microphones sits in front of a speaker changes a cabinet more than any of its
// knobs.
func (s *MeasureTestSuite) TestSweepableTakesDialsAndLists() {
	block := s.cat.Blocks["HD2_CabMicIr_2x15Brute"]
	got := Sweepable(block, WireOrder(s.cat, "HD2_CabMicIr_2x15Brute"))

	kinds := make([]string, 0, len(got))
	for _, c := range got {
		kinds = append(kinds, c.kind)
	}

	s.Require().Contains(kinds, "int", "the microphone is a list")
	s.Require().Contains(kinds, "float")
}

// TestSweepableNumbersASwitchBecauseTheCatalogDoesNot is why a switch needs a
// range invented for it.
//
// Line 6 record a switch's bounds as 0 and 0, not as false and true, so there
// is nothing to step through: swept on the catalog's own numbers a switch would
// be read once, at off, and reported as a control that does nothing. Off and on
// are numbered 0 and 1 here instead.
//
// A switch used to be left out entirely, on the grounds that two positions is
// not a curve. Two positions is a comparison, which is what a list already gets.
func (s *MeasureTestSuite) TestSweepableNumbersASwitchBecauseTheCatalogDoesNot() {
	const model = "HD2_PreampSVT4Pro"

	block := s.cat.Blocks[model]

	s.Require().InDelta(0, block.Params["Bright"].Min, 0.001)
	s.Require().InDelta(0, block.Params["Bright"].Max, 0.001,
		"the catalog carries no range for a switch")

	var bright control

	for _, c := range Sweepable(block, WireOrder(s.cat, model)) {
		if c.name == "Bright" {
			bright = c
		}
	}

	s.Require().Equal("Bright", bright.name, "a switch is swept")
	s.Require().Equal("bool", bright.kind)
	s.Require().InDelta(0, bright.low, 0.001)
	s.Require().InDelta(1, bright.high, 0.001, "off and on")
}

// TestSweepableTakesTheCatalogsRange is why a range is not assumed.
//
// 1,452 of the device's 4,835 float controls do not run zero to one. A Simple
// EQ's Mid Freq runs 125 to 4000 Hz, and swept 0..1 it never leaves its
// bottom stop and reports as a control that does nothing.
func (s *MeasureTestSuite) TestSweepableTakesTheCatalogsRange() {
	block := s.cat.Blocks["HD2_CabMicIr_2x15Brute"]

	// This cabinet carries no switch, which is what makes it the block to ask:
	// a switch is the one kind whose range this does not take, because the
	// catalog does not carry one.
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
func (s *MeasureTestSuite) TestFillGivesADialASlopeAndAListOrSwitchASpread() {
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

	// A switch goes with the list, and getting this wrong is invisible
	// downstream. A line through two points fits perfectly by construction, so
	// a switch given a slope would carry Straight at 1.0 and be believed over
	// every dial on the block, while its own spread stayed empty and the report
	// said it moved nothing.
	sw := measured.Curve{Kind: "bool", Points: points}
	Fill(&sw)

	s.Require().NotEmpty(sw.Spread, "off against on is a spread")
	s.Require().Nil(sw.Fits, "two positions is not a slope")
	s.Require().InDelta(100, sw.Spread["centroid"], 0.001)
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
//
// The count, not a bound. `Less(len(got), 48000*10)` admitted anything under
// 480,000 against a true answer of 108, so breaking the stop outright — reading
// only an eighth of what is there — passed it. A short read means every figure
// is taken against a truncated signal.
func (s *MeasureTestSuite) TestResampleStopsAtTheEndOfWhatItHas() {
	got := Resample(make([]float64, 100), 44100, 10)

	s.Require().Len(got, 108, "100 samples at 44.1kHz is 108 at 48")
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
	before := plan.Params{
		"Bass": catalog.Float(0.5), "Treble": catalog.Float(0.5),
	}
	after := plan.Params{
		"Bass": catalog.Float(0.877), "Treble": catalog.Float(0.5),
	}

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

// TestDriftedSaysWhenTheRigChangedUnderYou covers the only calibration there is.
//
// The baseline carries the whole gain structure in one measured number: which
// devices play and record, how loud the computer's output is, which cable goes
// where. None of that is recorded anywhere and every figure moves with it, so a
// campaign starting from a different baseline produces readings that cannot be
// compared with the committed ones. Eighty minutes is worth a warning first.
func (s *MeasureTestSuite) TestDriftedSaysWhenTheRigChangedUnderYou() {
	lib, err := measured.BuiltIn()
	s.Require().NoError(err)

	was := lib.Baseline

	for _, tt := range []struct {
		name  string
		give  measured.Figures
		warns bool
	}{
		{
			name: "the same rig says nothing",
			give: was,
		},
		{
			// Two takes of the same rig differ by hundredths, so the window has
			// to tolerate that or it cries wolf on every campaign.
			name: "the wander between takes says nothing",
			give: measured.Figures{
				Level: was.Level + 0.05, Centroid: was.Centroid + 0.3,
			},
		},
		{
			name:  "a nudged volume knob is caught by the level",
			give:  measured.Figures{Level: was.Level - 12, Centroid: was.Centroid},
			warns: true,
		},
		{
			// What the rig that opened the loop actually did: the old one added
			// mids and treble to everything, so the empty loop sat higher.
			name:  "a rewired rig is caught by the centroid",
			give:  measured.Figures{Level: was.Level, Centroid: was.Centroid - 50},
			warns: true,
		},
	} {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			Drifted(&buf, tt.give)

			if !tt.warns {
				s.Require().Empty(buf.String())

				return
			}

			said := buf.String()
			s.Require().Contains(said, "[warn]")
			s.Require().Contains(said, "different measuring rig")
			s.Require().Contains(said, "volume",
				"it names what to check, or the warning is just alarm")
		})
	}
}
