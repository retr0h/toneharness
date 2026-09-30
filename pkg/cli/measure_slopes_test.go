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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// SlopesPublicTestSuite covers holding a committed sweep to the chain it is
// being used in.
//
// The disagreement it settles is real and measured: `tone reach` answers from
// the committed slopes and says every rig reaches every genre, `tone tune`
// reads its slopes live and stopped 5.3 tolerances out on the same rig and
// target. The sweeps were taken with each block alone.
type SlopesPublicTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockTuner
	dry   string
}

func (s *SlopesPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockTuner(s.ctrl)
	// Every measuring command asks the client which models the attached
	// device has, rather than reading the built-in HX Stomp catalog, so that
	// --catalog and --device reach them.
	s.pedal.EXPECT().Catalog(gomock.Any()).
		DoAndReturn(func(context.Context) (*catalog.Catalog, error) {
			return catalog.BuiltIn()
		}).AnyTimes()
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di-short.wav")

	// The preset read back and rebuilt: every command here takes its chain off
	// the measuring loop before playing it, sending the output to USB rather
	// than to the socket the measuring lead comes from. What that rewrite does
	// is offTheLoop's own test, so it is answered here rather than asserted.
	s.pedal.EXPECT().
		PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, nil).AnyTimes()
}

func (s *SlopesPublicTestSuite) opts() SlopesOptions {
	return SlopesOptions{
		Client: s.pedal, Bench: &sloping{}, Dry: s.dry,
		ID: "matt-freeman", Seconds: 0.1, Takes: 2, Nudge: 0.1,
		Sweeps: filepath.Join("..", "..", "resources", "sweeps", "hx-stomp"),
	}
}

// built makes the pedal answer with a chain whose blocks have committed
// readings.
func (s *SlopesPublicTestSuite) built() {
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true,
		}}}}, nil)
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
}

// TestItPrintsTheLiveSlopeBesideTheCommittedOne is the whole point.
// TestSlopesRefusesAReferenceForTheOtherInstrument covers the guard.
//
// TestSlopes covers Slopes, which reads what each control does now and holds
// the committed sweeps to it.
//
// One method and one table, so a case is a row rather than a file.
func (s *SlopesPublicTestSuite) TestSlopes() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// describes the recording rather than the chain, so what comes back is the
			// wrong instrument rather than the wrong settings. Refused before a reading is
			// taken.
			name: "slopes refuses a reference for the other instrument",
			then: func() {
				{
					s.built()

					// The same recording under a name that says guitar, because a reference's
					// instrument is read off its filename: what is in the file is nobody's to
					// know and a path is what somebody typed.
					body, err := os.ReadFile(s.dry)
					s.Require().NoError(err)

					wrong := filepath.Join(s.T().TempDir(), "guitar-di.wav")
					s.Require().NoError(os.WriteFile(wrong, body, 0o600))

					opts := s.opts()
					opts.Dry = wrong

					s.Require().ErrorIs(Slopes(context.Background(), buffer(), opts), ErrWrongInstrument)
				}
			},
		},
		{
			name: "it prints the live slope beside the committed one",
			then: func() {
				s.built()

				w := buffer()

				s.Require().NoError(Slopes(context.Background(), w, s.opts()))

				said := w.String()
				s.Require().Contains(said, "COMMITTED")
				s.Require().Contains(said, "RATIO")
				s.Require().Contains(said, "HD2_AmpSVBeastBrt Treble")
				s.Require().Contains(said, "centroid")
			},
		},
		{
			name: "one figure on its own",
			then: func() {
				s.built()

				opts := s.opts()
				opts.Figure = "centroid"

				w := buffer()

				s.Require().NoError(Slopes(context.Background(), w, opts))

				s.Require().Contains(w.String(), "centroid")
				s.Require().NotContains(w.String(), "\n  high\n")
			},
		},
		{
			name: "a figure nothing committed carries",
			then: func() {
				s.built()

				opts := s.opts()
				opts.Figure = "nonesuch"

				w := buffer()

				s.Require().NoError(Slopes(context.Background(), w, opts))

				s.Require().Contains(w.String(), "nothing committed carries this figure")
			},
		},
		{
			// with a guitar is not a ratio about the control. Said rather than refused: the
			// comparison is still the only way to see how far a committed slope is from a
			// live one, which is what this command is for, so the run goes ahead and names
			// which files it cannot trust.
			name: "a committed sweep naming another instrument is said",
			then: func() {
				s.built()

				opts := s.opts()
				opts.Sweeps = filepath.Join("testdata", "sweeps-guitar")

				w := buffer()

				s.Require().NoError(Slopes(context.Background(), w, opts))

				said := w.String()
				s.Require().Contains(said, "do not name the instrument")
				s.Require().Contains(said, "(guitar)", "and which one it named instead")
				s.Require().Contains(said, "RATIO", "the comparison still happens")
			},
		},
		{
			name: "a chain with no dial",
			then: func() {
				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
						Model: catalog.ModelID("HD2_NotAModel"), Pos: 0,
					}}}}, nil)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)

				s.Require().Error(Slopes(context.Background(), buffer(), s.opts()))
			},
		},
		{
			name: "the rig will not build",
			then: func() {
				wanted := errors.New("no such rig")

				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sdk.Made{}, wanted)

				s.Require().ErrorIs(
					Slopes(context.Background(), buffer(), s.opts()), wanted)
			},
		},
		{
			name: "a reference that is not there",
			then: func() {
				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
						Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0,
					}}}}, nil)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)

				opts := s.opts()
				opts.Dry = filepath.Join(s.T().TempDir(), "nothing.wav")

				s.Require().Error(Slopes(context.Background(), buffer(), opts))
			},
		},
		{
			name: "a sweep that will not decode",
			then: func() {
				s.built()

				dir := s.T().TempDir()
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "HD2_AmpSVBeastBrt.json"), []byte("{"), 0o600))

				opts := s.opts()
				opts.Sweeps = dir

				s.Require().ErrorContains(
					Slopes(context.Background(), buffer(), opts), "HD2_AmpSVBeastBrt.json")
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

// TestACommittedSweepNamingAnotherInstrumentIsSaid covers the honest warning.
//

// TestRatioSaysWhichKindOfDisagreementItIs covers the column that matters.
//
// A sign change is not a bigger difference, it is a different answer: a control
// whose slope flipped is one the solver pushes the wrong way, confidently, on
// every pass. Four of the SV Beast's eleven do.
func (s *SlopesPublicTestSuite) TestRatioSaysWhichKindOfDisagreementItIs() {
	tests := []struct {
		name    string
		is, was float64
		want    string
	}{
		{"agreeing", 100, 200, "0.50x"},
		{"pointing the other way", 100, -200, "OPPOSITE"},
		{"the other way round", -100, 200, "OPPOSITE"},
		{"nothing committed", 100, 0, "was nil"},
		{"nothing live", 0, 100, "now nil"},
		{"neither", 0, 0, "both nil"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, ratio(tt.is, tt.was))
		})
	}
}

func TestSlopesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SlopesPublicTestSuite))
}
