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

// TestPreferred covers Preferred, which is how far down the list this block's
// family sits, lowest first.
//
// One method and one table, so a case is a row rather than a file.
func (s *PreferPublicTestSuite) TestPreferred() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The decision the order exists for.
			//
			// 108 names on this device are both an amplifier and a preamp.
			// Before this existed the winner was whichever sorted first by
			// identifier, which happened to be the amplifier because HD2_Amp
			// precedes HD2_Preamp. Right by the alphabet is not right by
			// decision, and the next device to rename a family would have
			// flipped it silently.
			name: "an amplifier beats a preamp of the same name",
			then: func() {
				amp := catalog.Block{ID: "HD2_AmpSVBeastBrt", Family: "amp"}
				pre := catalog.Block{ID: "HD2_PreampSVBeast", Family: "preamp"}

				s.Require().Less(amp.Preferred(), pre.Preferred())
			},
		},
		{
			// The other real collision.
			//
			// The same speaker ships three times under one name. The mic'd
			// model costs a third of the DSP and carries the
			// twelve-microphone list a comparison solves over, so a name that
			// fits both has to mean that one, and the pan variant sits behind
			// the plain one because a bass rig has no use for stereo
			// placement.
			name: "a micd cabinet beats the legacy one",
			then: func() {
				mic := catalog.Block{ID: "HD2_CabMicIr", Family: "cabmicirs"}
				pan := catalog.Block{ID: "HD2_CabMicIrPan", Family: "cabmicirswithpan"}
				old := catalog.Block{ID: "HD2_Cab1x15TucknGo", Family: "cab"}

				s.Require().Less(mic.Preferred(), pan.Preferred())
				s.Require().Less(pan.Preferred(), old.Preferred())
			},
		},
		{
			// Every other collision on the device.
			//
			// Two models of one family carrying one name are two spellings of
			// the same thing, so the identifier decides and the family says
			// nothing.
			name: "a family nobody ranked sorts last",
			then: func() {
				ranked := catalog.Block{ID: "HD2_AmpSVBeastBrt", Family: "amp"}
				other := catalog.Block{ID: "HD2_DistTubeDrive", Family: "distortion"}
				also := catalog.Block{ID: "HD2_DelaySimple", Family: "delay"}

				s.Require().Greater(other.Preferred(), ranked.Preferred())
				s.Require().Equal(other.Preferred(), also.Preferred(),
					"unranked families tie, so the identifier is what breaks it")
			},
		},
		{
			// How both resolvers use it.
			//
			// Sorting by Preferred and then by identifier is the shape the
			// gear resolver and the compiler each run, which is why the order
			// lives on the block rather than beside either of them.
			name: "the order sorts a whole collision",
			then: func() {
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
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestPreferPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PreferPublicTestSuite))
}
