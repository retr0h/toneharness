package cli

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// MissingPublicTestSuite covers turning missed axes into a block shortlist.
type MissingPublicTestSuite struct {
	suite.Suite
}

// TestAnAxisAlreadyInsideItsToleranceIsNotWanted covers asking for nothing.
func (s *MissingPublicTestSuite) TestAnAxisAlreadyInsideItsToleranceIsNotWanted() {
	aims := map[audio.Figure]solve.Aim{
		audio.KeyCentroid: {Want: 144, Tol: 20},
	}

	s.Require().Empty(unreached(aims,
		map[audio.Figure]float64{audio.KeyCentroid: 150}),
		"six hertz out of a twenty hertz tolerance needs no block")
}

// TestTheWantCarriesTheDirection is why the reading is carried rather than the
// residual.
func (s *MissingPublicTestSuite) TestTheWantCarriesTheDirection() {
	aims := map[audio.Figure]solve.Aim{
		audio.KeyCentroid: {Want: 144, Tol: 10},
	}

	up := unreached(aims, map[audio.Figure]float64{audio.KeyCentroid: 100})
	s.Require().Len(up, 1)
	s.Require().Positive(up[0].Need, "it reads low, so it needs raising")

	down := unreached(aims, map[audio.Figure]float64{audio.KeyCentroid: 900})
	s.Require().Len(down, 1)
	s.Require().Negative(down[0].Need, "it reads high, so it needs lowering")
}

// TestABandIsConvertedToTheLibrarysScale is the trap this file exists to avoid.
//
// A target and a reading are in the corpus's units, where a band is a fraction.
// The library reports a band as a percentage. A want handed over unconverted is a
// hundred times wrong on four of the ten axes and entirely plausible on the rest.
func (s *MissingPublicTestSuite) TestABandIsConvertedToTheLibrarysScale() {
	aims := map[audio.Figure]solve.Aim{
		audio.KeyHigh:     {Want: 0.40, Tol: 0.01},
		audio.KeyCentroid: {Want: 500, Tol: 10},
	}

	got := unreached(aims, map[audio.Figure]float64{
		audio.KeyHigh: 0.10, audio.KeyCentroid: 100,
	})

	by := map[audio.Figure]measured.Want{}
	for _, w := range got {
		by[w.Figure] = w
	}

	s.Require().InDelta(30, by[audio.KeyHigh].Need, 0.001,
		"0.30 of a fraction is 30 percentage points")
	s.Require().InDelta(1, by[audio.KeyHigh].Tol, 0.001)

	s.Require().InDelta(400, by[audio.KeyCentroid].Need, 0.001,
		"hertz are hertz in both scales")
	s.Require().InDelta(10, by[audio.KeyCentroid].Tol, 0.001)
}

// TestAnAxisWithNoToleranceIsNotWanted covers an aim nothing constrains.
func (s *MissingPublicTestSuite) TestAnAxisWithNoToleranceIsNotWanted() {
	s.Require().Empty(unreached(
		map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 144}},
		map[audio.Figure]float64{audio.KeyCentroid: 9000}))
}

// TestAnAxisTheChainDidNotReadIsNotWanted covers a figure with no answer.
func (s *MissingPublicTestSuite) TestAnAxisTheChainDidNotReadIsNotWanted() {
	s.Require().Empty(unreached(
		map[audio.Figure]solve.Aim{audio.KeyDecay: {Want: 0.6, Tol: 0.1}},
		map[audio.Figure]float64{}))
}

func TestMissingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MissingPublicTestSuite))
}
