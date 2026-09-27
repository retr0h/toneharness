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
package measured_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// MeasuredPublicTestSuite covers reading measurements and ranking against a
// target.
type MeasuredPublicTestSuite struct {
	suite.Suite

	lib measured.Library
}

func (s *MeasuredPublicTestSuite) SetupTest() {
	f, err := os.Open(filepath.Join("testdata", "library.json"))
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	s.lib, err = measured.Load(f)
	s.Require().NoError(err)
}

// dark is a bass target: most of the energy low, centroid a few hundred Hz.
func (s *MeasuredPublicTestSuite) dark() measured.Figures {
	return measured.Figures{Centroid: 300, Low: 80, Mid: 15, High: 5}
}

// TestLoadReadsALibrary covers the ordinary case, including the baseline.
//
// The baseline matters as much as the readings. Without it a figure says
// nothing: 95 Hz is not what a block does to a bass, it is what the bass
// already was.
func (s *MeasuredPublicTestSuite) TestLoadReadsALibrary() {
	s.Require().Equal("HX Stomp", s.lib.Device)
	s.Require().True(s.lib.Isolated)
	s.Require().InDelta(95.3, s.lib.Baseline.Centroid, 0.01)
	s.Require().Equal("abc", s.lib.Reference.SHA256)
	s.Require().Len(s.lib.Blocks, 7)
}

// TestLoadRefusesWhatIsNotALibrary covers the three ways reading fails.
func (s *MeasuredPublicTestSuite) TestLoadRefusesWhatIsNotALibrary() {
	tests := []struct {
		name string
		give string
		want string
	}{
		{name: "not JSON", give: "{", want: "decoding the measurements"},
		{
			// A document that parses and names nothing is worse than one
			// that fails, because every lookup against it answers "no
			// block is close" rather than "there is no library".
			name: "no blocks at all",
			give: `{"device":"HX Stomp"}`,
			want: "name no blocks",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := measured.Load(strings.NewReader(tt.give))

			s.Require().ErrorContains(err, tt.want)
		})
	}
}

// TestLoadReportsAReadFailure covers the reader itself failing.
func (s *MeasuredPublicTestSuite) TestLoadReportsAReadFailure() {
	_, err := measured.Load(broken{})

	s.Require().ErrorContains(err, "reading the measurements")
}

// TestNearestRanksByDistance covers the ordinary case.
func (s *MeasuredPublicTestSuite) TestNearestRanksByDistance() {
	got := s.lib.Nearest("amp", s.dark(), measured.Spectral())

	s.Require().NotEmpty(got)
	s.Require().InDelta(0, got[0].Distance, 0.001,
		"a block measuring exactly the target is a distance of zero from it")
	s.Require().Equal("AmpBright", got[len(got)-1].ID,
		"the brightest amp is furthest from a dark target")
}

// TestNearestStaysInItsCategory covers a cab never answering for an amp.
func (s *MeasuredPublicTestSuite) TestNearestStaysInItsCategory() {
	for _, m := range s.lib.Nearest("amp", s.dark(), measured.Spectral()) {
		s.Require().Equal(catalog.CategoryAmp, m.Category)
	}

	s.Require().Len(s.lib.Nearest("cab", s.dark(), measured.Spectral()), 1)
}

// TestNearestLeavesOutWhatItCannotTrust is the point of the two flags.
//
// A clipped reading's spectrum is the clipping's, not the block's, so ranking
// it puts a bright-looking reading of the converters at the top of a list of
// bright amplifiers. A refusal has no figures at all, and a zero centroid
// would read as the darkest block on the device.
func (s *MeasuredPublicTestSuite) TestNearestLeavesOutWhatItCannotTrust() {
	got := s.lib.Nearest("amp", s.dark(), measured.Spectral())

	for _, m := range got {
		s.Require().NotEqual("AmpClipped", m.ID, "a clipped reading was ranked")
		s.Require().NotEqual("AmpRefused", m.ID, "a refusal was ranked")
	}

	s.Require().Len(got, 4)
}

// TestSpectralIgnoresLoudness is a decision worth a test.
//
// A block that is right and quiet is right: the next block's level control
// fixes it for nothing. Weighing loudness would rank a loud wrong answer over
// a quiet correct one, which is how a chooser ends up preferring whatever
// clips hardest.
func (s *MeasuredPublicTestSuite) TestSpectralIgnoresLoudness() {
	got := s.lib.Nearest("amp", s.dark(), measured.Spectral())

	var dark, loud float64

	for _, m := range got {
		switch m.ID {
		case "AmpDark":
			dark = m.Distance
		case "AmpLoud":
			loud = m.Distance
		}
	}

	s.Require().Equal(dark, loud,
		"two blocks with the same spectrum and 17 dB between them rank alike")
}

// TestNearestIsStable covers two blocks that measure identically.
//
// Ranked by identifier after distance, because a ranking that reshuffles
// between runs is one nobody can act on twice.
func (s *MeasuredPublicTestSuite) TestNearestIsStable() {
	first := s.lib.Nearest("amp", s.dark(), measured.Spectral())

	for range 20 {
		s.Require().Equal(first, s.lib.Nearest("amp", s.dark(), measured.Spectral()))
	}

	s.Require().Equal("AmpDark", first[0].ID,
		"of three blocks measuring the target exactly, the first by identifier")
}

// TestWeightsCanBeGivenOutright covers weighing a figure the default ignores.
func (s *MeasuredPublicTestSuite) TestWeightsCanBeGivenOutright() {
	want := s.dark()
	want.Level = -20

	got := s.lib.Nearest("amp", want, measured.Weights{Level: 1})

	s.Require().Equal("AmpLoud", got[len(got)-1].ID,
		"weighed on level alone, the one 17 dB away is furthest")
}

// TestMeasuredSaysWhichReadingsAreUsable covers the flags directly.
func (s *MeasuredPublicTestSuite) TestMeasuredSaysWhichReadingsAreUsable() {
	s.Require().True(s.lib.Blocks["AmpDark"].Measured())
	s.Require().False(s.lib.Blocks["AmpClipped"].Measured())
	s.Require().False(s.lib.Blocks["AmpRefused"].Measured())
}

// broken is a reader that always fails.
type broken struct{}

func (broken) Read(
	[]byte,
) (int, error) {
	return 0, errors.New("no")
}

func TestMeasuredPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasuredPublicTestSuite))
}
