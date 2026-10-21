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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// SameRigTestSuite covers refusing to resume a library taken on other hardware.
//
// `resume` carries the earlier run's blocks and baseline into a header this run
// has just built, and the rig fields are per-library rather than per-block. So
// without this a resumed file comes out a mixture: one run's baseline beside
// another run's rig, and every carried block relabelled with hardware it was
// never measured through.
type SameRigTestSuite struct {
	suite.Suite
}

// on is a library describing one rig.
func on(
	instrument, pinned, sum string,
	open bool,
) measured.Library {
	return measured.Library{
		Instrument: instrument,
		PinnedOn:   pinned,
		Open:       open,
		Reference:  measured.Reference{SHA256: sum},
	}
}

// TestSameRig covers which earlier libraries this run may add to.
//
// One method and one table, so a case is a row rather than a file.
func (s *SameRigTestSuite) TestSameRig() {
	this := on("bass", "External Headphones", "abc", true)

	for _, tt := range []struct {
		name string
		had  measured.Library
		says string
	}{
		{
			name: "the same rig",
			had:  on("bass", "External Headphones", "abc", true),
		},
		{
			// A bass figure and a guitar figure are not the same figure, and
			// the header would claim they were.
			name: "another instrument",
			had:  on("guitar", "External Headphones", "abc", true),
			says: `instrument "guitar"`,
		},
		{
			// The level belongs to a device. A Mac with its default on
			// Bluetooth headphones pinned those to 38 while the jack feeding
			// the pedal sat wherever it was left.
			name: "another output device",
			had:  on("bass", "MacBook Pro Speakers", "abc", true),
			says: "MacBook Pro Speakers",
		},
		{
			name: "another reference signal",
			had:  on("bass", "External Headphones", "zzz", true),
			says: "the reference signal",
		},
		{
			// The rig that matters most. An Ampeg SVT reads 12.7% of its
			// energy in the mids with the loop closed and 4.0% with it open.
			name: "the loop closed rather than open",
			had:  on("bass", "External Headphones", "abc", false),
			says: "measuring loop closed",
		},
		{
			// 614 readings were taken before these fields existed. Refusing to
			// add to them would refuse the only library there is.
			name: "a library with nothing to say about its rig",
			had:  measured.Library{},
		},
	} {
		s.Run(tt.name, func() {
			lib := this
			err := sameRig(&lib, tt.had)

			if tt.says == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, errAnotherRig)
			s.Require().ErrorContains(err, tt.says)
		})
	}
}

func TestSameRigTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SameRigTestSuite))
}
