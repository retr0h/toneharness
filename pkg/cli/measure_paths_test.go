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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
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
	// Every measuring command asks the client which models the attached
	// device has, rather than reading the built-in HX Stomp catalog, so that
	// --catalog and --device reach them.
	s.pedal.EXPECT().Catalog(gomock.Any()).
		DoAndReturn(func(context.Context) (*catalog.Catalog, error) {
			return catalog.BuiltIn()
		}).AnyTimes()
}

// TestCompile covers compile, which writes one block's rig and turns it into
// a preset.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasurePathsTestSuite) TestCompile() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The scratch directory being unusable.
			name: "compiling says which file it could not write",
			then: func() {
				_, err := compile(context.Background(), s.pedal,
					measured.Block{ID: "HD2_AmpUSDripmanNorm"},
					filepath.Join(s.T().TempDir(), "no", "such", "place"), true, 0)

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "HD2_AmpUSDripmanNorm.yaml")
			},
		},
		{
			// The compiler saying no.
			//
			// Unwrapped, because the compiler's own sentence already names
			// the rig and the reason, and wrapping it again would say the
			// same thing twice.
			name: "compiling passes the pedals refusal back",
			then: func() {
				refused := errors.New("over the DSP budget")

				s.pedal.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					Return(sdk.Built{}, refused)

				_, err := compile(context.Background(), s.pedal,
					measured.Block{
						ID: "HD2_AmpUSDripmanNorm", Name: "US Dripman", Category: "amp",
					}, s.T().TempDir(), true, 0)

				s.Require().ErrorIs(err, refused)
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

// TestHash covers hash, which identifies the reference signal.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasurePathsTestSuite) TestHash() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A reference that is not there.
			//
			// Worth its own sentence because the reference is what every
			// number in the file is measured against: without it there is
			// nothing to compare later readings to, and the run should stop
			// rather than record readings against nothing.
			name: "hashing says which signal it could not read",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "absent.wav")

				_, err := hash(at)

				s.Require().Error(err)
				s.Require().Contains(err.Error(), at)
			},
		},
		{
			// The identification itself.
			name: "hashing is the same for the same bytes",
			then: func() {
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

// TestKeep covers keep, which writes the library out.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasurePathsTestSuite) TestKeep() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The readings having nowhere to land.
			name: "keeping says where it could not write",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "no", "such", "place.json")

				err := keep(at, measured.Library{Device: "HX Stomp"})

				s.Require().Error(err)
				s.Require().Contains(err.Error(), at)
			},
		},
		{
			// The round trip.
			//
			// Through the package's own reader, because a file only this can
			// read is a file the binary cannot embed.
			name: "keeping writes something that reads back",
			then: func() {
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

// TestWriteCurves covers write, which puts the curves where they were asked
// for.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasurePathsTestSuite) TestWriteCurves() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The curves having nowhere to land.
			//
			// A sweep is two minutes a control, so losing a finished campaign
			// to a path typo is the expensive failure here. The parent being
			// a file rather than a directory is the shape that actually
			// happens: an --out pointing at a directory somebody meant to
			// create.
			name: "writing curves says where it could not write",
			then: func() {
				blocked := filepath.Join(s.T().TempDir(), "not-a-directory")
				s.Require().NoError(os.WriteFile(blocked, []byte("a file"), 0o600))

				var buf bytes.Buffer

				err := write(&buf, measured.Curves{Device: "HX Stomp"},
					filepath.Join(blocked, "curves.json"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), blocked)
			},
		},
		{
			// The happy path's report.
			//
			// Sorted, because a map's order is not one, and a campaign whose
			// summary reshuffles between runs is one nobody can diff.
			name: "writing curves names what it measured",
			then: func() {
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

// TestReport covers report, which says what one control did, against what the
// rig can tell apart.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasurePathsTestSuite) TestReport() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The regression this refactor exists for.
			//
			// The report iterated a list of four figures typed into it, while
			// the noise floors beside it and measured.Named() both carried
			// ten. So the mid band was swept, compared against its floor,
			// written to the file, and never printed: somebody reading a
			// campaign's output would have concluded a control did not touch
			// the mids because the line saying so was not there.
			//
			// It iterates the alphabet now, so the next figure added is
			// printed without anybody remembering to.
			name: "the report names every figure it measured",
			then: func() {
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
			},
		},
		{
			// The comparison the answer turns on.
			//
			// The line a person reads to decide whether a control is worth
			// giving to the solver. Inverting it — `moved < floor*3` — calls
			// every control that moves a figure unaimable and every control
			// that does not aimable, and nothing noticed: the suites that
			// reach report only check that each figure is named.
			//
			// Two figures in one reading, one well over three times its floor
			// and one well under, so the two lines have to disagree.
			name: "the report says which way round",
			then: func() {
				var buf bytes.Buffer

				report(&buf, measured.Curve{
					Control: "Bass",
					Noise: map[audio.Figure]float64{
						audio.KeyMid:  0.02,
						audio.KeyHigh: 0.02,
					},
					Points: []measured.Point{
						// Mid moves seven, against a floor of 0.06. High moves 0.01,
						// which is under it.
						{Value: 0, Figures: measured.Figures{Mid: 8, High: 1}},
						{Value: 1, Figures: measured.Figures{Mid: 15, High: 1.01}},
					},
				})

				for _, want := range []string{"mid", "high"} {
					line := ""

					for _, at := range strings.Split(buf.String(), "\n") {
						if strings.Contains(at, want+" ") {
							line = at
						}
					}

					s.Require().NotEmpty(line, "the report names %s", want)

					if want == "mid" {
						s.Require().Contains(line, "real? yes",
							"a figure well over three times its floor is worth solving for")

						continue
					}

					s.Require().Contains(line, "real? no",
						"and one under it is the noise")
				}
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

func TestMeasurePathsTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasurePathsTestSuite))
}
