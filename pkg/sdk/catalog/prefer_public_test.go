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

package catalog_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

type PreferPublicTestSuite struct {
	suite.Suite
}

// TestAnAmplifierBeatsAPreampOfTheSameName is the decision the order exists
// for.
//
// 108 names on this device are both an amplifier and a preamp. Before this
// existed the winner was whichever sorted first by identifier, which happened
// to be the amplifier because HD2_Amp precedes HD2_Preamp. Right by the
// alphabet is not right by decision, and the next device to rename a family
// would have flipped it silently.
func (s *PreferPublicTestSuite) TestAnAmplifierBeatsAPreampOfTheSameName() {
	amp := catalog.Block{ID: "HD2_AmpSVBeastBrt", Family: "amp"}
	pre := catalog.Block{ID: "HD2_PreampSVBeast", Family: "preamp"}

	s.Require().Less(amp.Preferred(), pre.Preferred())
}

// TestAMicdCabinetBeatsTheLegacyOne covers the other real collision.
//
// The same speaker ships three times under one name. The mic'd model costs a
// third of the DSP and carries the twelve-microphone list a comparison solves
// over, so a name that fits both has to mean that one, and the pan variant
// sits behind the plain one because a bass rig has no use for stereo
// placement.
func (s *PreferPublicTestSuite) TestAMicdCabinetBeatsTheLegacyOne() {
	mic := catalog.Block{ID: "HD2_CabMicIr", Family: "cabmicirs"}
	pan := catalog.Block{ID: "HD2_CabMicIrPan", Family: "cabmicirswithpan"}
	old := catalog.Block{ID: "HD2_Cab1x15TucknGo", Family: "cab"}

	s.Require().Less(mic.Preferred(), pan.Preferred())
	s.Require().Less(pan.Preferred(), old.Preferred())
}

// TestAFamilyNobodyRankedSortsLast is every other collision on the device.
//
// Two models of one family carrying one name are two spellings of the same
// thing, so the identifier decides and the family says nothing.
func (s *PreferPublicTestSuite) TestAFamilyNobodyRankedSortsLast() {
	ranked := catalog.Block{ID: "HD2_AmpSVBeastBrt", Family: "amp"}
	other := catalog.Block{ID: "HD2_DistTubeDrive", Family: "distortion"}
	also := catalog.Block{ID: "HD2_DelaySimple", Family: "delay"}

	s.Require().Greater(other.Preferred(), ranked.Preferred())
	s.Require().Equal(other.Preferred(), also.Preferred(),
		"unranked families tie, so the identifier is what breaks it")
}

// TestTheOrderSortsAWholeCollision is how both resolvers use it.
//
// Sorting by Preferred and then by identifier is the shape the gear resolver
// and the compiler each run, which is why the order lives on the block rather
// than beside either of them.
func (s *PreferPublicTestSuite) TestTheOrderSortsAWholeCollision() {
	got := []catalog.Block{
		{ID: "HD2_Cab1x15TucknGo", Family: "cab"},
		{ID: "HD2_PreampSVBeast", Family: "preamp"},
		{ID: "HD2_CabMicIrPan", Family: "cabmicirswithpan"},
		{ID: "HD2_AmpSVBeastBrt", Family: "amp"},
		{ID: "HD2_CabMicIr", Family: "cabmicirs"},
	}

	sort.Slice(got, func(i, j int) bool {
		if a, b := got[i].Preferred(), got[j].Preferred(); a != b {
			return a < b
		}

		return got[i].ID < got[j].ID
	})

	want := []string{
		"HD2_AmpSVBeastBrt",
		"HD2_PreampSVBeast",
		"HD2_CabMicIr",
		"HD2_CabMicIrPan",
		"HD2_Cab1x15TucknGo",
	}

	for at, id := range want {
		s.Require().Equal(id, string(got[at].ID))
	}
}

func TestPreferPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PreferPublicTestSuite))
}
