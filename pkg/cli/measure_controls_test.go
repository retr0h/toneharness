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
	"context"
	"errors"
	"maps"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// ControlsRunTestSuite covers sweeping one block's controls, without the
// block or the hardware.
type ControlsRunTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockPedal
	cat   *catalog.Catalog
	out   string
	dry   string
}

func (s *ControlsRunTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockPedal(s.ctrl)
	// Every measuring command asks the client which models the attached
	// device has, rather than reading the built-in HX Stomp catalog, so that
	// --catalog and --device reach them.
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	s.cat = cat

	// The suite's own catalog rather than a second copy, so a case that needs
	// a model the symbol table does not name can withhold one.
	s.pedal.EXPECT().Catalog(gomock.Any()).
		DoAndReturn(func(context.Context) (*catalog.Catalog, error) {
			return s.cat, nil
		}).AnyTimes()
	s.out = filepath.Join(s.T().TempDir(), "curves.json")
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di-short.wav")

	// The preset read back and rebuilt: every command here takes its chain off
	// the measuring loop before playing it, sending the output to USB rather
	// than to the socket the measuring lead comes from. What that rewrite does
	// is offTheLoop's own test, so it is answered here rather than asserted.
	s.pedal.EXPECT().
		PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
}

// forget swaps in a catalog whose symbol table does not name a model.
//
// Every block this device carries is named there, so a catalog that claims no
// order for one cannot be had by choosing a model.
//
// A copy rather than an edit. `catalog.BuiltIn` decodes once and hands every
// caller the same pointer, which is only safe while nothing writes to it, so
// striking a symbol out of it here would take that model's order away from
// every other test in the process.
func (s *ControlsRunTestSuite) forget(
	model string,
) {
	kept := make([]catalog.Symbol, 0, len(s.cat.Symbols))

	for _, sym := range s.cat.Symbols {
		switch string(sym.ID) {
		case model, model + "Mono", model + "Stereo":
			continue
		}

		kept = append(kept, sym)
	}

	without := *s.cat
	without.Symbols = kept

	s.cat = &without
}

// ready makes the pedal answer everything a sweep asks of it.
func (s *ControlsRunTestSuite) ready() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			writeBlank(s.T(), in.Out)

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		Return(cabReading(nil), nil).AnyTimes()
}

// readyWithout is ready with the calls that come after the sweeps left out,
// so a test can make one of them fail.
func (s *ControlsRunTestSuite) readyWithout() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			writeBlank(s.T(), in.Out)

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		Return(cabReading(nil), nil).AnyTimes()
}

// cabReading is a device read of a chain holding the cabinet alone.
//
// Both layers, because a read populates both: the rig is what travels with the
// curves, and the probe reads its parameters out of the plan.
func cabReading(
	held plan.Params,
) sdk.Reading {
	return sdk.Reading{
		ID: "measured",
		Rig: rig.Spec{
			Instrument: rig.InstrumentBass,
			Chain:      []rig.ChainEntry{{Role: rig.RoleCab, Gear: "2x15 Brute"}},
		},
		Plan: plan.Plan{
			Name: "measured",
			Blocks: []plan.Block{{
				Model: "HD2_CabMicIr_2x15Brute", Params: held, Enabled: true,
			}},
		},
	}
}

// eqReading is a device read of a chain holding the equaliser alone.
func eqReading(
	held plan.Params,
) sdk.Reading {
	return sdk.Reading{
		ID: "probed",
		Rig: rig.Spec{
			Instrument: rig.InstrumentBass,
			Chain:      []rig.ChainEntry{{Role: rig.RoleEQ, Gear: "Simple EQ"}},
		},
		Plan: plan.Plan{
			Name: "probed",
			Blocks: []plan.Block{{
				Model: "HD2_EQSimple3Band", Params: held, Enabled: true,
			}},
		},
	}
}

// moves is a device that reveals a wire order by answering the probe.
//
// Each index moves the parameter at that position, which is what discovering an
// order reads back. Self-contained rather than layered on ready(), because
// gomock takes the first matching expectation and ready()'s Turn does nothing,
// so a probe behind it would read every index as reaching no parameter.
func (s *ControlsRunTestSuite) moves(
	order []string,
) {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			writeBlank(s.T(), in.Out)

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	held := plan.Params{}
	for _, name := range order {
		held[name] = catalog.Float(0.5)
	}

	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, at sdk.Address, v float32) error {
			if at.Param < len(order) {
				held[order[at.Param]] = catalog.Float(float64(v))
			}

			return nil
		}).AnyTimes()

	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, sdk.Format) (sdk.Reading, error) {
			now := plan.Params{}
			maps.Copy(now, held)

			return eqReading(now), nil
		}).AnyTimes()
}

// run sweeps a model and returns what landed on disk.
//
// A run where nothing carried signal writes a document naming no controls,
// which LoadCurves refuses on purpose, and that comes back as an empty Curves
// rather than a failure. So an assertion of the form NotContains(got.Controls,
// "Mic") holds whether the control was dropped for a reason or the whole
// document was unreadable. A test that means "nothing was usable" says Empty,
// and one that means "this one was dropped and others were not" says NotEmpty
// first.
func (s *ControlsRunTestSuite) run(
	model string,
	b sdk.Bench,
	// after is a model to put behind the one being swept, or nothing for the
	// chain of one every other case here measures. Task 271.
	after ...string,
) (measured.Curves, string, error) {
	var buf bytes.Buffer

	err := MeasureControls(context.Background(), &buf, ControlsOptions{
		Client: s.pedal, Model: model, Dry: s.dry, Out: s.out,
		Seconds: 1, Points: 3, Takes: 2, Bench: b,
		// Its own level, so nothing here asks the machine for one. Left nil
		// this falls back to `reamp.Held`, and the options carry `Volume: 0`:
		// every run of this suite set the output level of whatever computer
		// it was on to zero, which the owner of this one noticed by being
		// muted while playing. `reamp` refuses it in a test binary now and
		// this says so at the call site too.
		Level: func(want int) (int, bool, error) { return want, false, nil },
		After: strings.Join(after, ""),
	})
	if err != nil {
		return measured.Curves{}, buf.String(), err
	}

	f, err := os.Open(s.out)
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	// A run where nothing carried signal writes a document naming no
	// controls, which LoadCurves refuses on purpose. That is the answer here
	// rather than a failure, so it comes back empty.
	got, err := measured.LoadCurves(f)
	if err != nil {
		return measured.Curves{}, buf.String(), nil
	}

	return got, buf.String(), nil
}

// TestControlsAsksTheDeviceWhenTheCatalogClaimsNoOrder covers a block the
// catalog cannot name the indexes of.
//
// A sweep used to refuse outright, which left the blocks whose fingerprints say
// nothing as the blocks nothing described at all: an equaliser measures as the
// baseline at its defaults, because flat is what it ships at, so its curves are
// the only thing that says what it does.
//
// Discovered rather than handed over on a flag. An order somebody types is an
// order somebody can get wrong, and a curve filed under the wrong control is a
// plausible number about a different knob.
//
// The symbol withheld deliberately rather than relying on a category Line 6
// were believed to ship no symbols for. They ship one for every block; an
// equaliser looked unclaimed because the table names it `…Mono` and the lookup
// wanted the name as it stands.
func (s *ControlsRunTestSuite) TestControlsAsksTheDeviceWhenTheCatalogClaimsNoOrder() {
	s.forget("HD2_EQSimple3Band")
	s.moves([]string{"LowGain", "MidFreq", "MidGain", "HighGain", "Level"})

	got, said, err := s.run("HD2_EQSimple3Band", bench{})

	s.Require().NoError(err)
	s.Require().Contains(said, "claims no wire order")
	s.Require().True(got.Probed,
		"the file says the order came from the device, not the catalog")
	s.Require().Contains(got.Controls, "MidGain",
		"a control the catalog never named was found and swept")
}

// TestControlsSweepsDialsAndLists covers a cabinet, which has both.
//
// Its Mic is twelve microphones rather than a dial, and every one of them is
// measured: nothing sits between two microphones, so asking for three evenly
// spaced positions across twelve would measure some twice and miss others.
func (s *ControlsRunTestSuite) TestControlsSweepsDialsAndLists() {
	s.ready()

	got, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})
	s.Require().NoError(err)

	s.Require().True(got.Isolated)
	s.Require().Equal(alone, got.Slot)
	s.Require().NotEmpty(got.Chain, "the rig it ran through travels with it")
	s.Require().NotEmpty(got.Reference.SHA256)

	mic := got.Controls["Mic"]
	s.Require().Equal("int", mic.Kind)
	s.Require().Len(mic.Points, 12, "every microphone, not three of them")
	s.Require().NotEmpty(mic.Spread)
	s.Require().Empty(mic.Fits, "a list of settings has no slope")

	dial := got.Controls["Distance"]
	s.Require().Equal("float", dial.Kind)
	s.Require().Len(dial.Points, 3)
	s.Require().NotEmpty(dial.Fits)
	s.Require().Empty(dial.Spread)
	s.Require().InDelta(1, dial.Span.Low, 0.001, "inches, from the catalog")
	s.Require().InDelta(12, dial.Span.High, 0.001)
	s.Require().True(dial.Span.FromCatalog)

	// The positions themselves, not how many there are. Three readings taken
	// at the same place are still three points, and a line through three
	// identical points is still a fit, so the count and the fit both survive
	// a dial that never moved. This is the curve everything downstream is
	// built on.
	s.Require().InDelta(1, dial.Points[0].Value, 0.001, "the bottom of the range")
	s.Require().InDelta(6.5, dial.Points[1].Value, 0.001, "the middle")
	s.Require().InDelta(12, dial.Points[2].Value, 0.001, "the top")

	// And a list is every setting, in order, rather than twelve of one.
	for at := range mic.Points {
		s.Require().InDelta(float64(at), mic.Points[at].Value, 0.001)
	}
}

// TestControlsRecordsWhatWentSilent covers a control that mutes the chain.
//
// Kept rather than dropped, because where a control mutes a chain is worth
// knowing, and left out of the arithmetic, because two takes of silence agree
// to the last digit and the centroid of hiss reads high.
func (s *ControlsRunTestSuite) TestControlsRecordsWhatWentSilent() {
	s.ready()

	var calls int

	// Only the first control, because each later one measures its own floor
	// from the chain as it then sits, and a bench that is quiet throughout
	// makes quiet the settled level.
	//
	// Three readings of the reference, not two: the pre-flight takes one
	// before any control, to hold against the idle path. See [arrived].
	got, said, err := s.run("HD2_CabMicIr_2x15Brute",
		turning{after: 3, to: bench{quiet: true}, calls: &calls})

	s.Require().NoError(err)
	s.Require().Contains(said, "silent")
	s.Require().Contains(said, "no curve here",
		"a control with nothing to measure is said rather than invented")
	s.Require().NotContains(got.Controls, "Mic",
		"a control with one position that carried signal has no curve")
}

// TestControlsRecordsWhatClipped is the other end of the same problem.
//
// One microphone of a cabinet's twelve clips and read 4,471 Hz where the
// other eleven sat between 126 and 147. Left in, it is filed as the
// microphone choice moving the centre of gravity by 4,345 Hz, and the figure
// is the converters' ceiling.
func (s *ControlsRunTestSuite) TestControlsRecordsWhatClipped() {
	s.ready()

	var calls int

	got, said, err := s.run("HD2_CabMicIr_2x15Brute",
		turning{after: 2, to: bench{clipped: true}, calls: &calls})

	s.Require().NoError(err)
	s.Require().Empty(got.Controls,
		"every microphone hit the ceiling, so none of them is a reading and "+
			"the document names nothing")

	// Each of the twelve, by count rather than by the word appearing once.
	// Dropping the clipped branch leaves a document naming Mic, which the
	// assertion above catches, but it also stops saying which settings were
	// thrown away and why, which is what somebody reads to decide whether to
	// back the headroom off.
	s.Require().GreaterOrEqual(strings.Count(said, "clipped"), 12,
		"each of the twelve microphones says so, and every position of every "+
			"dial beside them, rather than the run saying it once")
}

// TestControlsRefusesWhenTheDeviceNamesNothingEither covers the last resort.
//
// With no symbol entry the catalog cannot say which index is which and the
// device is asked instead. When the device answers nothing at any index there
// is no third place to look, and a sweep that went ahead would file every
// curve under a guess. That is the one case still refused.
//
// The refusal used to cover every unclaimed block, which left those blocks
// undescribed: an equaliser measures as the baseline at its defaults, because
// flat is what it ships at, so its curves are the only thing that says what it
// does.
func (s *ControlsRunTestSuite) TestControlsRefusesWhenTheDeviceNamesNothingEither() {
	s.forget("HD2_EQSimple3Band")
	s.ready()

	_, _, err := s.run("HD2_EQSimple3Band", bench{})

	s.Require().ErrorContains(err, "named no parameter")
}

// TestControlsRefusesABlockTheDeviceDidNotLoad covers the pre-flight.
//
// One method and one table, so a case is a row rather than a file.
//
// The readings are the expensive part and they are worthless when the block is
// not in the chain, so nothing may be measured before this passes. Each row
// forbids Turn and Choose outright: a sweep that refused only afterwards is
// the bug this exists for, and it cost four minutes forty-five seconds a
// block while looking like a device fault.
// TestControlsRefusesALevelItCouldNotPin covers the sweep stopping when the
// computer's output level is not the one it asked for.
//
// `FM4 Throbber` was swept at 0 after asking for 38. The platform said so and
// nothing stopped, so eight controls were measured through a muted output:
// -122dB with centroids between 1630 and 2390Hz, hiss recorded as a filter.
func (s *ControlsRunTestSuite) TestControlsRefusesALevelItCouldNotPin() {
	var buf bytes.Buffer

	err := MeasureControls(context.Background(), &buf, ControlsOptions{
		Client: s.pedal, Model: "HD2_CabMicIr_2x15Brute", Dry: s.dry,
		Out: s.out, Seconds: 1, Points: 3, Takes: 2, Bench: bench{},
		Volume: 38,
		// What macOS did: it took the instruction, and the read-back came
		// back somewhere else entirely.
		Level: func(int) (int, bool, error) { return 0, true, nil },
	})

	s.Require().Error(err)
	s.Require().ErrorIs(err, errNotPinned)
	s.Require().ErrorContains(err, "asked for 38 and came back at 0")

	// Said before it stopped, because the level it reached is the thing worth
	// knowing and a refusal that hides it is one nobody can act on.
	s.Require().Contains(buf.String(), "moved to 0")
}

// TestControlsRefusesAChainThatDiesMidSweep covers the sweep stopping when the
// chain stops, rather than carrying on and measuring the converters.
//
// The silence floor is thirty decibels under what the chain settles at, and
// that was read again before every control, so a chain that dies drags the
// floor after it. `6x10 Cali Power` shipped with six controls of hiss recorded
// that way, each clearing its own floor by thirty decibels, reading centroids
// between 908 and 3083Hz on a bass cabinet.
func (s *ControlsRunTestSuite) TestControlsRefusesAChainThatDiesMidSweep() {
	s.ready()

	// Enough readings for the first control and then nothing, which is the
	// shape of the fault: the block that was swept first is fine and every
	// one after it is the converters.
	//
	_, said, err := s.run("HD2_CabMicIr_2x15Brute", &dies{after: 24})

	s.Require().Error(err)
	s.Require().ErrorIs(err, errChainDied)
	s.Require().Contains(err.Error(), "gone quiet partway through")
	s.Require().NotEmpty(said)
}

func (s *ControlsRunTestSuite) TestControlsRefusesABlockTheDeviceDidNotLoad() {
	for _, tt := range []struct {
		name    string
		reading sdk.Reading
		err     error
		says    string
	}{
		{
			// What an HX Stomp answers when asked to hold a dual cabinet.
			name:    "an empty chain",
			reading: sdk.Reading{},
			says:    "an empty chain",
		},
		{
			// Worse than empty, because the readings would be of real gear
			// and would still be filed under the wrong name.
			name:    "a chain holding something else",
			reading: eqReading(nil),
			says:    "a chain of HD2_EQSimple3Band",
		},
		{
			name: "a device that will not say what it is playing",
			err:  errors.New("usb: gone"),
			says: "usb: gone",
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			s.pedal.EXPECT().
				Compile(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
					writeBlank(s.T(), in.Out)

					return sdk.Built{}, nil
				}).AnyTimes()
			s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			s.pedal.EXPECT().
				Current(gomock.Any(), gomock.Any()).
				Return(tt.reading, tt.err).AnyTimes()

			// Nothing is measured, which is the assertion.
			s.pedal.EXPECT().
				Turn(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			s.pedal.EXPECT().
				Choose(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

// ampIntoCab is what the device reports while playing an amplifier with a
// cabinet behind it, which is the chain task 271 exists for.
func ampIntoCab() sdk.Reading {
	return sdk.Reading{
		ID: "measured",
		Rig: rig.Spec{
			Instrument: rig.InstrumentBass,
			Chain: []rig.ChainEntry{
				{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
				{Role: rig.RoleCab, Gear: "8x10 Ampeg SVT-E"},
			},
		},
		Plan: plan.Plan{
			Name: "measured",
			Blocks: []plan.Block{
				{Model: "HD2_AmpSVBeastNrm", Enabled: true},
				{Model: "HD2_Cab8x10SVBeast", Pos: 1, Enabled: true},
			},
		},
	}
}

// TestControlsSweepsIntoABlockBehindIt covers --after, which is the whole of
// task 271 that does not need the pedal.
//
// An amplifier swept with nothing in front of it has no speaker rolloff, so
// what comes back is the buzz a cabinet exists to absorb: its Treble moved
// the centroid 12,763Hz per turn where the same control in a chain moves it
// 3,250, and four of its eleven controls changed sign. So the reading has to
// say which kind it is, and say it in the file rather than in a sentence
// somewhere.
func (s *ControlsRunTestSuite) TestControlsSweepsIntoABlockBehindIt() {
	var compiled plan.Plan

	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			// The plan on its way to the device, read back off disk, because
			// what the chain holds is the claim the file then makes about it.
			f, err := os.Open(in.Plan)
			s.Require().NoError(err)

			defer func() { _ = f.Close() }()

			got, err := plan.Load(f)
			s.Require().NoError(err)

			if len(got.Blocks) > len(compiled.Blocks) {
				compiled = got
			}

			writeBlank(s.T(), in.Out)

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Current(gomock.Any(), gomock.Any()).
		Return(ampIntoCab(), nil).AnyTimes()

	got, _, err := s.run("HD2_AmpSVBeastNrm", bench{}, "HD2_Cab8x10SVBeast")
	s.Require().NoError(err)

	s.Require().Len(compiled.Blocks, 2, "the cabinet went behind the amplifier")
	s.Require().Equal(
		catalog.ModelID("HD2_Cab8x10SVBeast"), compiled.Blocks[1].Model)
	s.Require().True(compiled.Blocks[1].Enabled,
		"a bypassed cabinet leaves the amplifier measured alone")

	s.Require().False(got.Isolated,
		"nothing downstream may read a chain sweep as a reading about the block")
	s.Require().Contains(got.Chain, "Ampeg SVT")
	s.Require().Contains(got.Chain, "8x10 Ampeg SVT-E",
		"the chain the device reported, so a later reader can rebuild it")
}

// TestControlsRefusesAChainMissingTheBlockBehind covers a cabinet the device
// quietly dropped.
//
// The failure that matters most here, because it is silent: the device
// answers a chain it could not build by playing what it could, so the
// amplifier would be measured alone while the file said `isolated: false`
// and claimed a cabinet was in the path.
func (s *ControlsRunTestSuite) TestControlsRefusesAChainMissingTheBlockBehind() {
	s.ready()

	// Nothing is measured, which is half the assertion.
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, _, err := s.run(
		"HD2_CabMicIr_2x15Brute", bench{}, "HD2_Cab8x10SVBeast")

	s.Require().ErrorContains(err, "the device did not load HD2_Cab8x10SVBeast")
	s.Require().ErrorContains(err, "a chain of HD2_CabMicIr_2x15Brute")
}

// TestControlsRefusesABlockBehindItThatIsNotOne covers --after naming
// something the catalog has no model of, and naming the block itself.
//
// Refused before any hardware is touched, for the same reason the sweep
// checks what the device loaded: a chain the device could not build is a
// chain it answers by playing what it could.
//
// One method and one table, so a case is a row rather than a file.
func (s *ControlsRunTestSuite) TestControlsRefusesABlockBehindItThatIsNotOne() {
	for _, tt := range []struct {
		name  string
		after string
		says  string
	}{
		{
			name:  "a model nobody has",
			after: "nothing calls itself this",
			says:  "the catalog has no nothing calls itself this",
		},
		{
			// Both blocks answer to the same name, so a curve filed under a
			// control is a curve of whichever of the two the sweep reached.
			name:  "the block being swept",
			after: "HD2_CabMicIr_2x15Brute",
			says:  "cannot follow itself",
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			// Nothing is compiled and nothing is played, which is the point
			// of checking it here.
			s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).Times(0)
			s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Times(0)

			_, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{}, tt.after)

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

// TestControlsRefusesAModelNobodyHas covers a name that is not a block.
func (s *ControlsRunTestSuite) TestControlsRefusesAModelNobodyHas() {
	_, _, err := s.run("nothing calls itself this", bench{})

	s.Require().ErrorContains(err, "the catalog has no")
}

// TestControlsRefusesAnOutItCannotMake covers the last step.
//
// An hour of readings is worth nothing if the file cannot be written, and the
// directory is made rather than required, so a parent that is a file rather
// than a directory is the one way that fails. Said with the path in it,
// because the path is what somebody has to fix.
func (s *ControlsRunTestSuite) TestControlsRefusesAnOutItCannotMake() {
	s.ready()

	// A file where the output's parent directory would go.
	blocked := filepath.Join(s.T().TempDir(), "in-the-way")
	s.Require().NoError(os.WriteFile(blocked, []byte("not a directory"), 0o600))

	s.out = filepath.Join(blocked, "curves.json")

	_, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

	s.Require().ErrorContains(err, "in-the-way")
}

// TestMeasureControls covers MeasureControls, which sweeps every control of
// one block, alone, and writes what each of them does.
//
// One method and one table, so a case is a row rather than a file.
func (s *ControlsRunTestSuite) TestMeasureControls() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A missing signal.
			name: "controls reports a reference it cannot read",
			then: func() {
				var buf bytes.Buffer

				err := MeasureControls(context.Background(), &buf, ControlsOptions{
					Client: s.pedal, Model: "HD2_CabMicIr_2x15Brute",
					Dry: "nowhere.wav", Out: s.out, Seconds: 1, Points: 3, Takes: 2,
					Bench: bench{},
				})

				s.Require().ErrorContains(err, "nowhere.wav")
			},
		},
		{
			// --device naming a pedal this binary ships no catalog for.
			name: "controls reports a catalog it cannot read",
			then: func() {
				ctrl := gomock.NewController(s.T())
				pedal := mocks.NewMockPedal(ctrl)
				pedal.EXPECT().Catalog(gomock.Any()).
					Return(nil, errors.New("no catalog for that pedal")).AnyTimes()

				var buf bytes.Buffer

				err := MeasureControls(context.Background(), &buf, ControlsOptions{
					Client: pedal, Model: "HD2_CabMicIr_2x15Brute", Dry: s.dry,
					Out: s.out, Seconds: 1, Points: 3, Takes: 2, Bench: bench{},
				})

				s.Require().ErrorContains(err, "no catalog for that pedal")
			},
		},
		{
			// The last two steps.
			//
			// The chain travelling in the file is read after the sweeps, with
			// the preset put back, so what it records is what was measured
			// rather than whatever position the last control was left at.
			// Both calls can fail, and a file carrying a chain nobody read
			// would be a file claiming a measurement it does not have.
			name: "controls puts the preset back before reading the chain",
			then: func() {
				for _, tt := range []struct {
					name  string
					setup func()
					says  string
				}{
					{
						name: "the preset would not go back",
						setup: func() {
							s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
								Return(nil).Times(1)
							s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
								Return(errors.New("would not load")).AnyTimes()
						},
						says: "would not load",
					},
					{
						// Only the read that asks for a rig, which is the one at the end.
						// The sweeps read the plan, and failing those would stop the run
						// somewhere else entirely.
						name: "the device would not say what it is playing",
						setup: func() {
							s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
								Return(nil).AnyTimes()
							s.pedal.EXPECT().Current(gomock.Any(), sdk.FormatRig).
								Return(sdk.Reading{}, errors.New("would not say")).AnyTimes()
						},
						says: "would not say",
					},
				} {
					s.Run(tt.name, func() {
						s.SetupTest()
						tt.setup()
						s.readyWithout()

						var buf bytes.Buffer

						err := MeasureControls(context.Background(), &buf, ControlsOptions{
							Client: s.pedal, Model: "HD2_CabMicIr_2x15Brute", Dry: s.dry,
							Out: s.out, Seconds: 1, Points: 3, Takes: 2, Bench: bench{},
						})

						s.Require().ErrorContains(err, tt.says)
					})
				}
			},
		},
		{
			// The five measuring commands all open the loop through benchFor,
			// and the preset is built before the loop is opened, so this is
			// the one refusal that arrives after the compiler has already
			// done its work.
			name: "no bench supplied and no interface of that name",
			then: func() {
				s.ready()

				err := MeasureControls(context.Background(), &bytes.Buffer{},
					ControlsOptions{
						Client: s.pedal, Model: "HD2_CabMicIr_2x15Brute",
						Dry: s.dry, Out: s.out, Seconds: 1, Points: 3, Takes: 2,
						Hardware: "no such interface anybody owns",
					})

				s.Require().ErrorContains(err, "no such interface anybody owns")
			},
		},
		{
			// A sweep builds one preset and plays it several hundred times,
			// so a machine with nowhere to write it cannot start.
			name: "there is nowhere to build the preset",
			then: func() {
				s.T().Setenv("TMPDIR",
					filepath.Join(s.T().TempDir(), "not a directory"))

				_, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

				s.Require().ErrorContains(err,
					"making somewhere to build a preset")
			},
		},
		{
			// The directory is made on the way, so the path being wrong is
			// only found out at the write itself, after the whole sweep has
			// been taken. Losing an hour of readings to a mistyped path is
			// the reason it says which path.
			name: "the curves have nowhere to go",
			then: func() {
				s.ready()

				held := s.T().TempDir()

				err := MeasureControls(context.Background(), &bytes.Buffer{},
					ControlsOptions{
						Client: s.pedal, Model: "HD2_CabMicIr_2x15Brute",
						Dry: s.dry, Out: held, Seconds: 1, Points: 3,
						Takes: 2, Bench: bench{},
					})

				s.Require().ErrorContains(err, held)
			},
		},
		{
			// The preset is loaded again before each control is swept,
			// because a control stays where the last sweep left it and the
			// next one would be measured on a chain the first skewed.
			name: "the device will not load the preset for a sweep",
			then: func() {
				s.pedal.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					DoAndReturn(
						func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
							writeBlank(s.T(), in.Out)

							return sdk.Built{}, nil
						}).AnyTimes()
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(errors.New("the device stopped taking the message")).
					AnyTimes()

				_, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

				s.Require().ErrorContains(err,
					"the device stopped taking the message")
			},
		},
		{
			// The noise floor is read before any control moves, so a bench
			// that stops there stops the sweep rather than costing one
			// reading of one position.
			name: "the bench stops while the noise floor is being read",
			then: func() {
				s.ready()

				calls := 0

				_, _, err := s.run("HD2_CabMicIr_2x15Brute",
					stopping{after: 0, calls: &calls})

				s.Require().ErrorContains(err, "stopped answering")
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

// TestControlsReportsAChainItCannotBuild covers the compiler failing.
func (s *ControlsRunTestSuite) TestControlsReportsAChainItCannotBuild() {
	s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, errors.New("will not build")).AnyTimes()

	_, _, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

	s.Require().ErrorContains(err, "building a chain holding HD2_CabMicIr_2x15Brute")
}

// TestControlsCarriesOnPastAControlTheDeviceRefuses covers a declined move.
func (s *ControlsRunTestSuite) TestControlsCarriesOnPastAControlTheDeviceRefuses() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			writeBlank(s.T(), in.Out)

			return sdk.Built{}, nil
		}).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
		Return(cabReading(nil), nil).AnyTimes()
	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("error -3")).AnyTimes()
	s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("error -3")).AnyTimes()

	_, said, err := s.run("HD2_CabMicIr_2x15Brute", bench{})

	s.Require().NoError(err)
	s.Require().Contains(said, "refused")
}

// settling is a bench whose first reading is the stream still coming up.
//
// What an HX Stomp does. Six takes of a punk chain read low 19.35 once and
// then 31.56 to 31.73 five times over, so the first take is the one reading
// in the set that describes the audio device rather than the chain.
type settling struct {
	seen int
}

func (b *settling) Name() string { return "settling" }

func (b *settling) Through(
	_ context.Context,
	signal []float32,
) ([]float32, error) {
	b.seen++

	// A tone an octave up stands in for an unsettled stream: it is a reading
	// that disagrees with every other one, which is the only property of the
	// first take that matters here.
	hz := 110.0
	if b.seen == 1 {
		hz = 220.0
	}

	out := make([]float32, len(signal))
	for i := range out {
		out[i] = float32(0.2 * math.Sin(2*math.Pi*hz*float64(i)/sdk.Rate))
	}

	return out, nil
}

// TestTheFirstReadingAfterABenchOpensIsThrownAway covers the noise floor.
//
// The floor is the lower bound on every tolerance the loop solves against, so
// one unsettled take does not merely add noise: it stretches the target until
// anything hits it. On hardware it took the floor from 0.0010 of a band to
// 0.1238, and a run then reported arriving while its residual grew from 1.1
// tolerances out to 2.4.
func (s *ControlsRunTestSuite) TestTheFirstReadingAfterABenchOpensIsThrownAway() {
	b := &settling{}

	floor, _, err := steady(s.T().Context(), b, make([]float32, sdk.Rate), 3)

	s.Require().NoError(err)
	s.Require().Equal(4, b.seen, "three takes, and one thrown away before them")

	// The three kept takes are identical, so every figure sits on its own
	// floor rather than on a spread. Keeping the first would put the centre
	// an octave away from the other three.
	s.Require().InDelta(floors[audio.KeyCentroid], floor[audio.KeyCentroid], 0.001,
		"the kept takes agree, so the centre wanders by its floor and no more")
}

// TestAlongside covers saying what a sweep of an amplifier with no cabinet is
// worth, before the run that costs most of an hour.
//
// Task 254. Every sweep this command takes is of one block and nothing else,
// which for most kinds of block is a caveat and for an amplifier decides the
// readings: 43 of the 121 usable curves in the committed library hold a
// position reading more than half its energy above 2kHz, and all 43 are
// amplifiers.
//
// One method and one table, so a case is a row rather than a file.
func (s *ControlsRunTestSuite) TestAlongside() {
	tests := []struct {
		name string
		of   catalog.Category
		// after is the model put behind the block, empty where it was swept
		// on its own.
		after string
		// said is whether the caveat is printed.
		said bool
	}{
		{
			name: "an amplifier is warned about",
			of:   catalog.CategoryAmp, said: true,
		},
		{
			// The run that found this said "no cabinet after it" and "the
			// file records isolated: true" with a cabinet behind the block
			// and `isolated: false` in the file. A warning that contradicts
			// the file is worse than none.
			name:  "an amplifier with a cabinet behind it is not",
			of:    catalog.CategoryAmp,
			after: "HD2_CabMicIr_8x10SVTAV", said: false,
		},
		{
			// Nothing else. A drive or a compressor swept alone is a caveat
			// rather than a different reading: the inverted curves are all
			// amplifiers.
			name: "a drive is not",
			of:   catalog.CategoryDrive, said: false,
		},
		{
			name: "nor is a cabinet",
			of:   catalog.CategoryCab, said: false,
		},
		{
			name: "nor a block whose category nothing says",
			of:   "", said: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			alongside(&buf, tt.of, tt.after)

			if !tt.said {
				s.Require().Empty(buf.String())

				return
			}

			out := buf.String()

			// The measurement rather than an adjective, and the field a reader
			// downstream keys on.
			s.Require().Contains(out, "no cabinet")
			s.Require().Contains(out, "3,250Hz")
			s.Require().Contains(out, "12,763")
			s.Require().Contains(out, "isolated: true")
		})
	}
}

func TestControlsRunTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ControlsRunTestSuite))
}
