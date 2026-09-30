package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
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

// TestInTheChainNamesWhatIsAlreadyThere covers not suggesting what a chain holds.
func (s *MissingPublicTestSuite) TestInTheChainNamesWhatIsAlreadyThere() {
	got := inTheChain(plan.Plan{Blocks: []plan.Block{
		{Model: catalog.ModelID("HD2_AmpSVBeastBrt")},
		{Model: catalog.ModelID("HD2_Cab8x10SVBeast")},
	}})

	s.Require().True(got["HD2_AmpSVBeastBrt"])
	s.Require().True(got["HD2_Cab8x10SVBeast"])
	s.Require().False(got["HD2_DistMinotaur"], "a block nobody has is an addition")
	s.Require().Empty(inTheChain(plan.Plan{}), "an empty chain holds nothing")
}

// TestWhyReportsTheFiguresRatherThanAVerdict is what makes two suggestions
// distinguishable.
//
// Two blocks that both "add harmonics" are not the same suggestion when one adds
// 40 and the other adds 4, so the numbers travel. And a block that closes one
// axis while opening another says both, because that trade is a real answer and
// an invisible one is not.
func (s *MissingPublicTestSuite) TestWhyReportsTheFiguresRatherThanAVerdict() {
	helps := measured.Suggestion{
		Helps: []audio.Figure{audio.KeyHarmonics},
		Moves: map[audio.Figure]float64{audio.KeyHarmonics: 38.4},
	}
	s.Require().Contains(why(helps), "harmonics")
	s.Require().Contains(why(helps), "38.4")

	both := measured.Suggestion{
		Helps: []audio.Figure{audio.KeyHarmonics},
		Hurts: []audio.Figure{audio.KeyCentroid},
		Moves: map[audio.Figure]float64{
			audio.KeyHarmonics: 31.2, audio.KeyCentroid: 210,
		},
	}
	said := why(both)
	s.Require().Contains(said, "31.2")
	s.Require().Contains(said, "wrong way", "the trade has to be visible")
	s.Require().Contains(said, "210")

	s.Require().Contains(why(measured.Suggestion{}), "nothing in the right direction",
		"a block that helps with none of it says so rather than printing blank")
}

// TestMissingSaysNothingWhenThereIsNothingToSay covers the advisory staying quiet.
//
// It prints after a run that did not arrive, so a run that did, or one whose
// axes are all inside their tolerances, must produce no table at all. A
// shortlist printed under a converged run reads as a complaint about a chain
// that worked.
func (s *MissingPublicTestSuite) TestMissingSaysNothingWhenThereIsNothingToSay() {
	made := plan.Plan{Blocks: []plan.Block{
		{Model: catalog.ModelID("HD2_AmpSVBeastBrt")},
	}}

	var buf bytes.Buffer

	// Every axis inside its tolerance, so nothing is wanted.
	missing(&buf, made,
		map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 144, Tol: 50}},
		map[audio.Figure]float64{audio.KeyCentroid: 150})

	s.Require().Empty(buf.String())

	// And no aims at all, which is what a run with nothing to hit looks like.
	missing(&buf, made, nil, nil)
	s.Require().Empty(buf.String())
}

// TestMissingPrintsAShortlistWhenAnAxisIsOut is the live path.
//
// Against the shipped library rather than a fixture, because the point of the
// table is which of 661 real blocks is worth eight seconds of measuring, and a
// fixture would prove only that the printing works.
func (s *MissingPublicTestSuite) TestMissingPrintsAShortlistWhenAnAxisIsOut() {
	made := plan.Plan{Blocks: []plan.Block{
		{Model: catalog.ModelID("HD2_AmpSVBeastBrt")},
	}}

	var buf bytes.Buffer

	// Far more harmonics than the chain reads, which is the case the shortlist
	// exists for: no dial closes it, so the answer is a block.
	missing(&buf, made,
		map[audio.Figure]solve.Aim{audio.KeyHarmonics: {Want: 0.60, Tol: 0.01}},
		map[audio.Figure]float64{audio.KeyHarmonics: 0.02})

	said := buf.String()
	s.Require().Contains(said, "the chain rather than the")
	s.Require().Contains(said, "BLOCK")
	s.Require().Contains(said, "harmonics",
		"the axis it was found for is named, or the row says nothing")
	s.Require().Contains(said, "measured on its own",
		"the caution travels with the table, because the table is the most "+
			"tempting thing in the output to act on directly")
}
