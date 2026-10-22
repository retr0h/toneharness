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

package plan_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

type ValidateHoldsPublicTestSuite struct {
	suite.Suite
}

// panCab is one of the 46 the family covers, as the catalog describes it.
func panCab() catalog.Block {
	return catalog.Block{
		ID:       "HD2_CabMicIr_1x10USPrincessWithPan",
		Name:     "1x10 US Princess",
		Category: catalog.CategoryCab,
		Family:   "cabmicirswithpan",
		DSP:      catalog.DSPCost{Mono: 2.5, Prov: catalog.ProvOfficial},
	}
}

// shaper is the one refused by identifier rather than by family.
func shaper() catalog.Block {
	return catalog.Block{
		ID:       "HD2_CompressorTransientShaper",
		Name:     "Transient Shaper",
		Category: catalog.CategoryComp,
		Family:   "compressor",
		DSP:      catalog.DSPCost{Mono: 2.5, Prov: catalog.ProvOfficial},
	}
}

// TestValidateHolds covers ValidateHolds, which reports the first block in s
// the device will not load.
//
// Against the real limits rather than invented ones, because which blocks a
// device refuses is the measured fact this exists to carry and a fixture would
// let it drift from what the hardware said.
//
// One method and one table, so a case is a row rather than a file.
func (s *ValidateHoldsPublicTestSuite) TestValidateHolds() {
	stomp := plan.LimitsFor("HX Stomp")
	floor := plan.LimitsFor("Helix Floor")

	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The whole point.
			//
			// A rig naming one of these used to resolve, write a preset the
			// device accepted, read back byte for byte, and play a chain the
			// block was missing from.
			name: "a pan cabinet is refused on an HX Stomp",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{{Model: panCab().ID}}}

				err := plan.ValidateHolds(newCatalog(panCab()), spec, stomp)

				s.Require().ErrorIs(err, plan.ErrRefusedBlock)

				var target *plan.RefusedBlockError
				s.Require().True(errors.As(err, &target))
				s.Require().Equal(string(panCab().ID), target.Model)

				// And it says what happens, not only that something is wrong,
				// because the failure it prevents leaves no other trace.
				s.Require().ErrorContains(err, "plays without it")
			},
		},
		{
			// By identifier rather than by family, since a Transient Shaper
			// is a compressor and the rest of that family loads.
			name: "the Transient Shaper is refused by name",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{{Model: shaper().ID}}}

				s.Require().ErrorIs(
					plan.ValidateHolds(newCatalog(shaper()), spec, stomp),
					plan.ErrRefusedBlock)
			},
		},
		{
			// The limit of it, and the reason this is keyed off the device.
			//
			// 65 presets in the corpus hold a pan cabinet on a two-processor
			// Helix and none holds one on a Stomp, so these load elsewhere and
			// refusing them there would refuse a rig that works.
			name: "the same cabinet is allowed on a Helix Floor",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{{Model: panCab().ID}}}

				s.Require().NoError(
					plan.ValidateHolds(newCatalog(panCab()), spec, floor))
			},
		},
		{
			// A device nobody has asked.
			//
			// Empty limits refuse nothing, which is the honest default: a
			// refusal nobody measured would stop a rig that works.
			name: "a device with nothing measured refuses nothing",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{{Model: panCab().ID}}}

				s.Require().NoError(plan.ValidateHolds(
					newCatalog(panCab()), spec, plan.Limits{MaxBlocks: 8, Paths: 1}))
			},
		},
		{
			name: "a block the device holds passes",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{{Model: testAmp().ID}}}

				s.Require().NoError(
					plan.ValidateHolds(newCatalog(testAmp()), spec, stomp))
			},
		},
		{
			// A model the catalog does not carry is not this layer's to
			// report. ValidateStructure answers that and names it, and two
			// layers reporting one fault is how a failure stops naming one
			// cause.
			name: "an unknown model is left to the layer that names it",
			then: func() {
				spec := plan.Plan{Blocks: []plan.Block{{Model: "HD2_Nope"}}}

				s.Require().NoError(
					plan.ValidateHolds(newCatalog(testAmp()), spec, stomp))
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestValidateHoldsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidateHoldsPublicTestSuite))
}
