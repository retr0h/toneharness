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
package solve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// AimTestSuite covers turning measured records into a target.
type AimTestSuite struct {
	suite.Suite
}

// punk reads the genre this project models first, from the measurements that
// ship, rather than from numbers typed here.
func (s *AimTestSuite) punk() audio.Across {
	body, err := os.ReadFile(
		filepath.Join("..", "audio", "data", "genres.json"))
	s.Require().NoError(err)

	var genres []audio.Genre
	s.Require().NoError(json.Unmarshal(body, &genres))

	for _, g := range genres {
		if g.Slug == "punk" {
			return g.Across
		}
	}

	s.Require().Fail("punk is not in the shipped genres")

	return audio.Across{}
}

// TestPunkIsATargetEvenThoughItEarnsNoWord covers the genre modelled first.
//
// TestAims covers Aims, which turns what a body of records measures as into a
// target.
//
// One method and one table, so a case is a row rather than a file.
func (s *AimTestSuite) TestAims() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// reports punk earning no term, which says it is not distinctive against the
			// players who avoid it. It does not say punk has no position: fifteen records
			// give a middle and a spread on every axis, and that is what a solver needs.
			name: "punk is a target even though it earns no word",
			then: func() {
				got := Aims(s.punk(), nil)

				s.Require().NotEmpty(got)

				centre, ok := got[audio.KeyCentroid]
				s.Require().True(ok, "punk has a centre of gravity")
				s.Require().InDelta(144, centre.Want, 1)
				s.Require().Positive(centre.Tol, "and a spread to allow for")

				// A fraction, not a percentage: the CLI prints "97.4% low" and the figure
				// behind it is 0.974. Which is the reason every row of the solve is divided
				// by its own tolerance, because a centroid in hundreds of hertz beside a
				// band share under one would otherwise be the only axis that mattered.
				band, ok := got[audio.KeyLow]
				s.Require().True(ok)
				s.Require().Greater(band.Want, 0.9, "a bass corpus lives in the low band")
				s.Require().Less(band.Want, 1.0)
			},
		},
		{
			name: "the spread is the tolerance",
			then: func() {
				got := Aims(audio.Across{
					Tracks:   4,
					Centroid: audio.Spread{Low: 100, Mid: 150, High: 200},
					Low:      audio.Spread{Low: 90, Mid: 95, High: 100},
				}, nil)

				s.Require().InDelta(50, got[audio.KeyCentroid].Tol, 0.01,
					"half the ten-to-ninety band")
				s.Require().InDelta(5, got[audio.KeyLow].Tol, 0.01)
			},
		},
		{
			// precision the rig cannot demonstrate and the loop would never report
			// arriving.
			name: "the floor is the lower bound",
			then: func() {
				flat := audio.Across{
					Tracks:   1,
					Centroid: audio.Spread{Low: 150, Mid: 150, High: 150},
				}

				s.Require().NotContains(Aims(flat, nil), audio.KeyCentroid,
					"no spread and no floor is an axis nothing can say it reached")

				got := Aims(flat, map[audio.Figure]float64{audio.KeyCentroid: 2})
				s.Require().InDelta(2, got[audio.KeyCentroid].Tol, 0.001)
			},
		},
		{
			name: "a figure no recording answered is absent",
			then: func() {
				got := Aims(audio.Across{
					Tracks:   3,
					Centroid: audio.Spread{Low: 100, Mid: 150, High: 200},
				}, nil)

				s.Require().NotContains(got, audio.KeyTransient,
					"nothing rose, so nothing says how sharply")
				s.Require().NotContains(got, audio.KeyDecay)
			},
		},
		{
			// recording had one, so an unmeasured figure arrives as zero. Pinning an axis to
			// zero because nobody measured it would spend the chain defending the absence of
			// a measurement.
			name: "a figure nothing measured is not a target of zero",
			then: func() {
				got := Aims(audio.Across{
					Tracks:   3,
					Centroid: audio.Spread{Low: 100, Mid: 150, High: 200},
				}, map[audio.Figure]float64{
					audio.KeyDynamics: 0.5,
					audio.KeyLean:     0.01,
				})

				s.Require().Contains(got, audio.KeyCentroid)
				s.Require().NotContains(got, audio.KeyDynamics,
					"a floor does not turn an absent figure into a target")
				s.Require().NotContains(got, audio.KeyLean)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestTheFloorIsTheLowerBound covers records that agree exactly.
//

// TestOnly covers Only, which keeps the axes named, and drops the rest.
//
// One method and one table, so a case is a row rather than a file.
func (s *AimTestSuite) TestOnly() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "only keeps the axes named",
			then: func() {
				all := Aims(s.punk(), nil)
				s.Require().Greater(len(all), 2)

				got := Only(all, []audio.Figure{audio.KeyCentroid, audio.KeyLow})

				s.Require().Len(got, 2)
				s.Require().Contains(got, audio.KeyCentroid)
				s.Require().NotContains(got, audio.KeyHarmonics)
			},
		},
		{
			// measured.
			name: "only ignores an axis the target never had",
			then: func() {
				got := Only(map[audio.Figure]Aim{audio.KeyLow: {Want: 1, Tol: 1}},
					[]audio.Figure{audio.KeyLow, audio.KeyDecay})

				s.Require().Len(got, 1)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestFloorTakesTheWorstOfTheChain covers a chain's repeatability.
func (s *AimTestSuite) TestFloorTakesTheWorstOfTheChain() {
	got := Floor(
		map[audio.Figure]float64{audio.KeyCentroid: 2, audio.KeyLow: 0.5},
		map[audio.Figure]float64{audio.KeyCentroid: 7},
	)

	s.Require().InDelta(7, got[audio.KeyCentroid], 0.001,
		"a chain is as repeatable as its least repeatable part")
	s.Require().InDelta(0.5, got[audio.KeyLow], 0.001)
}

// TestAFigureNothingMeasuredIsNotATargetOfZero covers the absent middle.
//

func TestAimTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AimTestSuite))
}
