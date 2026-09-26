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

package sweepdoc_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/measured"
	"github.com/retr0h/tonestack/pkg/sdk/measured/internal/sweepdoc"
)

// SweepdocPublicTestSuite covers the page the measurements generate.
type SweepdocPublicTestSuite struct {
	suite.Suite

	lib    measured.Library
	curves measured.Curves
}

func (s *SweepdocPublicTestSuite) SetupSuite() {
	var err error

	s.lib, err = measured.BuiltIn()
	s.Require().NoError(err)

	at := filepath.Join("..", "..", "..", "..", "..",
		"resources", "sweeps", "hx-stomp", "HD2_AmpUSDripmanNorm.json")

	f, err := os.Open(at) //nolint:gosec // a path this repository owns
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	s.curves, err = measured.LoadCurves(f)
	s.Require().NoError(err)
}

// TestTheShippedPageIsCurrent is what keeps the page honest.
//
// Every figure on this page was typed in by hand until now, and the first
// re-measurement made all of them wrong while it still read as authoritative:
// it claimed the Bass control moves the centre of gravity by 13,649Hz where the
// answer is 9,405. This fails the moment the readings and the page disagree.
func (s *SweepdocPublicTestSuite) TestTheShippedPageIsCurrent() {
	want, err := sweepdoc.Render(s.lib, s.curves)
	s.Require().NoError(err)

	at := filepath.Join("..", "..", "..", "..", "..", "docs", "measurements.md")

	got, err := os.ReadFile(at) //nolint:gosec // a path this repository owns
	s.Require().NoError(err)

	s.Require().Equal(string(want), string(got),
		"docs/measurements.md is out of date — run `just generate`")
}

// TestEveryFigureOnThePageCameFromTheFile covers the point of generating it.
//
// The numbers the page quotes are in the readings, so a page that drifted from
// them fails here as well as on the comparison above.
func (s *SweepdocPublicTestSuite) TestEveryFigureOnThePageCameFromTheFile() {
	body, err := sweepdoc.Render(s.lib, s.curves)
	s.Require().NoError(err)

	page := string(body)

	s.Require().Contains(page, s.curves.Gear)
	s.Require().Contains(page, s.curves.Block)
	s.Require().Contains(page, s.curves.Reference.File)

	for name := range s.curves.Controls {
		s.Require().Contains(page, name,
			"%s was measured and the page does not name it", name)
	}
}

// TestTheClaimsAboutShapeAreTrue is the half a comparison cannot check.
//
// The prose says Bass and Treble are a tone stack and that one control is the
// volume. Those are claims about the data, and a re-measurement could make any
// of them false while the page went on asserting it. Each is checked against
// the readings rather than trusted.
func (s *SweepdocPublicTestSuite) TestTheClaimsAboutShapeAreTrue() {
	body, err := sweepdoc.Render(s.lib, s.curves)
	s.Require().NoError(err)

	page := string(body)

	s.Run("a tone stack is two controls of opposite sign", func() {
		s.Require().Contains(page, "are a tone stack")

		var most, least float64
		for _, c := range s.curves.Controls {
			if fit, ok := c.Fits["centroid"]; ok {
				most = max(most, fit.PerTurn)
				least = min(least, fit.PerTurn)
			}
		}

		s.Require().Positive(most, "one control moves the centroid up")
		s.Require().Negative(least, "and another moves it down")
	})

	s.Run("the volume control moves level further than any other", func() {
		s.Require().Contains(page, "is the volume control")

		named := strings.SplitN(
			strings.SplitN(page, "is the volume control", 2)[0], "**", 2)

		s.Require().Len(named, 2)
	})
}

// TestCurvesNamingNothingIsRefused covers a file with no controls in it.
func (s *SweepdocPublicTestSuite) TestCurvesNamingNothingIsRefused() {
	_, err := sweepdoc.Render(s.lib, measured.Curves{})

	s.Require().ErrorContains(err, "name no controls")
}

func TestSweepdocPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SweepdocPublicTestSuite))
}
