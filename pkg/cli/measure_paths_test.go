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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/cli/internal/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
)

// MeasurePathsTestSuite covers what a campaign does when the filesystem or
// the pedal refuses.
//
// A campaign is hours of hardware time, so the answer to a refusal has to be
// a sentence naming what refused rather than a panic nine blocks in. These
// are the paths a happy run never reaches.
type MeasurePathsTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockPedal
}

func (s *MeasurePathsTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockPedal(s.ctrl)
}

// TestCompilingSaysWhichFileItCouldNotWrite covers the scratch directory
// being unusable.
func (s *MeasurePathsTestSuite) TestCompilingSaysWhichFileItCouldNotWrite() {
	_, err := compile(context.Background(), s.pedal,
		measured.Block{ID: "HD2_AmpUSDripmanNorm"},
		filepath.Join(s.T().TempDir(), "no", "such", "place"), true)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "HD2_AmpUSDripmanNorm.yaml")
}

// TestCompilingPassesThePedalsRefusalBack covers the compiler saying no.
//
// Unwrapped, because the compiler's own sentence already names the rig and
// the reason, and wrapping it again would say the same thing twice.
func (s *MeasurePathsTestSuite) TestCompilingPassesThePedalsRefusalBack() {
	refused := errors.New("over the DSP budget")

	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, refused)

	_, err := compile(context.Background(), s.pedal,
		measured.Block{
			ID: "HD2_AmpUSDripmanNorm", Name: "US Dripman", Category: "amp",
		}, s.T().TempDir(), true)

	s.Require().ErrorIs(err, refused)
}

// TestHashingSaysWhichSignalItCouldNotRead covers a reference that is not
// there.
//
// Worth its own sentence because the reference is what every number in the
// file is measured against: without it there is nothing to compare later
// readings to, and the run should stop rather than record readings against
// nothing.
func (s *MeasurePathsTestSuite) TestHashingSaysWhichSignalItCouldNotRead() {
	at := filepath.Join(s.T().TempDir(), "absent.wav")

	_, err := hash(at)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), at)
}

// TestHashingIsTheSameForTheSameBytes covers the identification itself.
func (s *MeasurePathsTestSuite) TestHashingIsTheSameForTheSameBytes() {
	dir := s.T().TempDir()
	one := filepath.Join(dir, "one.wav")
	two := filepath.Join(dir, "two.wav")

	s.Require().NoError(os.WriteFile(one, []byte("a signal"), 0o600))
	s.Require().NoError(os.WriteFile(two, []byte("a signal"), 0o600))

	first, err := hash(one)
	s.Require().NoError(err)

	second, err := hash(two)
	s.Require().NoError(err)

	s.Require().Equal(first, second)
	s.Require().Len(first, 64)
}

// TestKeepingSaysWhereItCouldNotWrite covers the readings having nowhere to
// land.
func (s *MeasurePathsTestSuite) TestKeepingSaysWhereItCouldNotWrite() {
	at := filepath.Join(s.T().TempDir(), "no", "such", "place.json")

	err := keep(at, measured.Library{Device: "HX Stomp"})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), at)
}

// TestKeepingWritesSomethingThatReadsBack covers the round trip.
//
// Through the package's own reader, because a file only this can read is a
// file the binary cannot embed.
func (s *MeasurePathsTestSuite) TestKeepingWritesSomethingThatReadsBack() {
	at := filepath.Join(s.T().TempDir(), "measured.json")

	s.Require().NoError(keep(at, measured.Library{
		Device:   "HX Stomp",
		Isolated: true,
		Blocks: map[string]measured.Block{
			"HD2_AmpUSDripmanNorm": {ID: "HD2_AmpUSDripmanNorm", Name: "US Dripman"},
		},
	}))

	f, err := os.Open(at) //nolint:gosec // a path this test made
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	lib, err := measured.Load(f)
	s.Require().NoError(err)

	s.Require().Equal("HX Stomp", lib.Device)
	s.Require().Len(lib.Blocks, 1)
}

// TestReadingTheChainPassesTheDevicesRefusalBack covers the pedal refusing to
// say what it is playing.
func (s *MeasurePathsTestSuite) TestReadingTheChainPassesTheDevicesRefusalBack() {
	refused := errors.New("no device on the bus")

	s.pedal.EXPECT().
		Current(gomock.Any(), sdk.FormatRig).
		Return(sdk.Reading{}, refused)

	_, err := current(context.Background(), s.pedal)

	s.Require().ErrorIs(err, refused)
}

// TestWritingCurvesSaysWhereItCouldNotWrite covers the curves having nowhere
// to land.
//
// A sweep is two minutes a control, so losing a finished campaign to a path
// typo is the expensive failure here. The parent being a file rather than a
// directory is the shape that actually happens: an --out pointing at a
// directory somebody meant to create.
func (s *MeasurePathsTestSuite) TestWritingCurvesSaysWhereItCouldNotWrite() {
	blocked := filepath.Join(s.T().TempDir(), "not-a-directory")
	s.Require().NoError(os.WriteFile(blocked, []byte("a file"), 0o600))

	var buf bytes.Buffer

	err := write(&buf, measured.Curves{Device: "HX Stomp"},
		filepath.Join(blocked, "curves.json"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), blocked)
}

// TestWritingCurvesNamesWhatItMeasured covers the happy path's report.
//
// Sorted, because a map's order is not one, and a campaign whose summary
// reshuffles between runs is one nobody can diff.
func (s *MeasurePathsTestSuite) TestWritingCurvesNamesWhatItMeasured() {
	at := filepath.Join(s.T().TempDir(), "deep", "curves.json")

	var buf bytes.Buffer

	s.Require().NoError(write(&buf, measured.Curves{
		Device: "HX Stomp",
		Controls: map[string]measured.Curve{
			"Treble": {Control: "Treble"},
			"Bass":   {Control: "Bass"},
		},
	}, at))

	s.Require().FileExists(at)
	s.Require().Contains(buf.String(), "2 controls measured")
	s.Require().Contains(buf.String(), "Bass, Treble")
}

// TestProbingPassesTheDevicesRefusalBack covers the pedal refusing to play
// the chain back between probes.
//
// It has to be put back before each index, so a failure here means every
// reading after it would have been taken against whatever the last probe
// left behind.
func (s *MeasurePathsTestSuite) TestProbingPassesTheDevicesRefusalBack() {
	refused := errors.New("no device on the bus")

	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(refused)

	_, err := probe(context.Background(), s.pedal, "/tmp/absent.hlx", 1)

	s.Require().ErrorIs(err, refused)
}

// TestTheReportNamesEveryFigureItMeasured is the regression this refactor
// exists for.
//
// The report iterated a list of four figures typed into it, while the noise
// floors beside it and measured.Named() both carried ten. So the mid band was
// swept, compared against its floor, written to the file, and never printed:
// somebody reading a campaign's output would have concluded a control did not
// touch the mids because the line saying so was not there.
//
// It iterates the alphabet now, so the next figure added is printed without
// anybody remembering to.
func (s *MeasurePathsTestSuite) TestTheReportNamesEveryFigureItMeasured() {
	var buf bytes.Buffer

	report(&buf, measured.Curve{
		Control: "Bass",
		Noise:   map[audio.Figure]float64{audio.KeyMid: 0.02},
		Points: []measured.Point{
			{Value: 0, Figures: measured.Figures{Mid: 8}},
			{Value: 1, Figures: measured.Figures{Mid: 15}},
		},
	})

	for _, f := range measured.Named() {
		s.Require().Contains(buf.String(), string(f),
			"%s is measured and the report does not name it", f)
	}
}

func TestMeasurePathsTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasurePathsTestSuite))
}
