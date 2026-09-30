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

// TestUnreached covers unreached, which is the axes a target still wants, in
// the library's own units.
//
// One method and one table, so a case is a row rather than a file.
func (s *MissingPublicTestSuite) TestUnreached() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Asking for nothing.
			name: "an axis already inside its tolerance is not wanted",
			then: func() {
				aims := map[audio.Figure]solve.Aim{
					audio.KeyCentroid: {Want: 144, Tol: 20},
				}

				s.Require().Empty(unreached(aims,
					map[audio.Figure]float64{audio.KeyCentroid: 150}),
					"six hertz out of a twenty hertz tolerance needs no block")
			},
		},
		{
			// Why the reading is carried rather than the residual.
			name: "the want carries the direction",
			then: func() {
				aims := map[audio.Figure]solve.Aim{
					audio.KeyCentroid: {Want: 144, Tol: 10},
				}

				up := unreached(aims, map[audio.Figure]float64{audio.KeyCentroid: 100})
				s.Require().Len(up, 1)
				s.Require().Positive(up[0].Need, "it reads low, so it needs raising")

				down := unreached(aims, map[audio.Figure]float64{audio.KeyCentroid: 900})
				s.Require().Len(down, 1)
				s.Require().Negative(down[0].Need, "it reads high, so it needs lowering")
			},
		},
		{
			// The trap this file exists to avoid.
			//
			// A target and a reading are in the corpus's units, where a band
			// is a fraction. The library reports a band as a percentage. A
			// want handed over unconverted is a hundred times wrong on four
			// of the ten axes and entirely plausible on the rest.
			name: "a band is converted to the librarys scale",
			then: func() {
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
			},
		},
		{
			// An aim nothing constrains.
			name: "an axis with no tolerance is not wanted",
			then: func() {
				s.Require().Empty(unreached(
					map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 144}},
					map[audio.Figure]float64{audio.KeyCentroid: 9000}))
			},
		},
		{
			// A figure with no answer.
			name: "an axis the chain did not read is not wanted",
			then: func() {
				s.Require().Empty(unreached(
					map[audio.Figure]solve.Aim{audio.KeyDecay: {Want: 0.6, Tol: 0.1}},
					map[audio.Figure]float64{}))
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
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

	// Two axes, so the separator between them is there. A block that closes
	// more than one gap is the interesting kind and its line read as one
	// figure run into the next.
	two := measured.Suggestion{
		Helps: []audio.Figure{audio.KeyHarmonics, audio.KeyCentroid},
		Moves: map[audio.Figure]float64{
			audio.KeyHarmonics: 12.5, audio.KeyCentroid: -84,
		},
	}
	said = why(two)

	s.Require().Contains(said, "harmonics +12.5")
	s.Require().Contains(said, "centroid -84")
	s.Require().Contains(said, ", ", "the two are separated rather than run together")
}

// TestMissing covers missing, which prints the blocks that would close what
// the dials could not.
//
// One method and one table, so a case is a row rather than a file.
func (s *MissingPublicTestSuite) TestMissing() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The advisory staying quiet.
			//
			// It prints after a run that did not arrive, so a run that did,
			// or one whose axes are all inside their tolerances, must produce
			// no table at all. A shortlist printed under a converged run
			// reads as a complaint about a chain that worked.
			name: "missing says nothing when there is nothing to say",
			then: func() {
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
			},
		},
		{
			// The live path.
			//
			// Against the shipped library rather than a fixture, because the
			// point of the table is which of 661 real blocks is worth eight
			// seconds of measuring, and a fixture would prove only that the
			// printing works.
			name: "missing prints a shortlist when an axis is out",
			then: func() {
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
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}
