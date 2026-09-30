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
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di.wav")

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
func (s *SlopesPublicTestSuite) TestItPrintsTheLiveSlopeBesideTheCommittedOne() {
	s.built()

	w := buffer()

	s.Require().NoError(Slopes(context.Background(), w, s.opts()))

	said := w.String()
	s.Require().Contains(said, "COMMITTED")
	s.Require().Contains(said, "RATIO")
	s.Require().Contains(said, "HD2_AmpSVBeastBrt Treble")
	s.Require().Contains(said, "centroid")
}

// TestOneFigureOnItsOwn covers narrowing the report.
func (s *SlopesPublicTestSuite) TestOneFigureOnItsOwn() {
	s.built()

	opts := s.opts()
	opts.Figure = "centroid"

	w := buffer()

	s.Require().NoError(Slopes(context.Background(), w, opts))

	s.Require().Contains(w.String(), "centroid")
	s.Require().NotContains(w.String(), "\n  high\n")
}

// TestAFigureNothingCommittedCarries covers an axis with no committed slope.
func (s *SlopesPublicTestSuite) TestAFigureNothingCommittedCarries() {
	s.built()

	opts := s.opts()
	opts.Figure = "nonesuch"

	w := buffer()

	s.Require().NoError(Slopes(context.Background(), w, opts))

	s.Require().Contains(w.String(), "nothing committed carries this figure")
}

// TestAChainWithNoDial covers gear the solver cannot touch.
func (s *SlopesPublicTestSuite) TestAChainWithNoDial() {
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_NotAModel"), Pos: 0,
		}}}}, nil)
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)

	s.Require().Error(Slopes(context.Background(), buffer(), s.opts()))
}

// TestTheRigWillNotBuild covers a rig nobody curated.
func (s *SlopesPublicTestSuite) TestTheRigWillNotBuild() {
	wanted := errors.New("no such rig")

	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{}, wanted)

	s.Require().ErrorIs(
		Slopes(context.Background(), buffer(), s.opts()), wanted)
}

// TestAReferenceThatIsNotThere covers a missing signal.
func (s *SlopesPublicTestSuite) TestAReferenceThatIsNotThere() {
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
			Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0,
		}}}}, nil)
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)

	opts := s.opts()
	opts.Dry = filepath.Join(s.T().TempDir(), "nothing.wav")

	s.Require().Error(Slopes(context.Background(), buffer(), opts))
}

// TestASweepThatWillNotDecode covers a reading file somebody broke.
func (s *SlopesPublicTestSuite) TestASweepThatWillNotDecode() {
	s.built()

	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "HD2_AmpSVBeastBrt.json"), []byte("{"), 0o600))

	opts := s.opts()
	opts.Sweeps = dir

	s.Require().ErrorContains(
		Slopes(context.Background(), buffer(), opts), "HD2_AmpSVBeastBrt.json")
}

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
