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

// TestTakenStampsEachReadingWithItsRun covers task 257: filing a reading under
// the session that took it.
//
// One method and one table, so a case is a row rather than a file.
func (s *SameRigTestSuite) TestTakenStampsEachReadingWithItsRun() {
	lib := func() measured.Library {
		return measured.Library{
			Volume:   38,
			Baseline: measured.Figures{Centroid: 155.27},
			Blocks:   map[string]measured.Block{},
		}
	}

	tests := []struct {
		name string
		then func()
	}{
		{
			// The whole point. A reading says which run took it, and the run
			// is in the file beside it.
			name: "a reading is stamped and its run recorded",
			then: func() {
				got := lib()

				taken(&got, measured.Block{ID: "HD2_DistOne"})

				block := got.Blocks["HD2_DistOne"]
				s.Require().Equal(got.Header().Key(), block.Run)

				run, named := got.RunOf(block)
				s.Require().True(named, "the run came from the file")
				s.Require().Equal(38, run.Volume)
			},
		},
		{
			// Every write site calls this and none of them tracks whether the
			// run is registered yet.
			name: "two readings from one run share its entry",
			then: func() {
				got := lib()

				taken(&got, measured.Block{ID: "HD2_DistOne"})
				taken(&got, measured.Block{ID: "HD2_DistTwo"})

				s.Require().Len(got.Runs, 1)
				s.Require().Equal(
					got.Blocks["HD2_DistOne"].Run,
					got.Blocks["HD2_DistTwo"].Run)
			},
		},
		{
			// A header that moves mid-run is a different run, which is what
			// the digest is for: this is how a resumed file comes to hold two.
			name: "a reading taken at another volume is another run",
			then: func() {
				got := lib()

				taken(&got, measured.Block{ID: "HD2_DistOne"})

				got.Volume = 50
				taken(&got, measured.Block{ID: "HD2_DistTwo"})

				s.Require().Len(got.Runs, 2)
				s.Require().NotEqual(
					got.Blocks["HD2_DistOne"].Run,
					got.Blocks["HD2_DistTwo"].Run)

				at, _ := got.RunOf(got.Blocks["HD2_DistOne"])
				s.Require().Equal(38, at.Volume,
					"the earlier reading was relabelled")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestRunsSaysWhenAFileHoldsMoreThanOne covers reporting a mixed library.
//
// Said because the way a library comes to hold more than one run is `--resume`,
// which looks like carrying on rather than like mixing two runs.
func (s *SameRigTestSuite) TestRunsSaysWhenAFileHoldsMoreThanOne() {
	earlier := measured.Run{Volume: 20}
	later := measured.Run{Volume: 38, Open: true}

	tests := []struct {
		name string
		lib  measured.Library
		// said is whether the remark is printed.
		said bool
	}{
		{
			// The ordinary case, which needs no remark.
			name: "one run says nothing",
			lib: measured.Library{
				Runs: map[string]measured.Run{later.Key(): later},
				Blocks: map[string]measured.Block{
					"a": {Run: later.Key()}, "b": {Run: later.Key()},
				},
			},
		},
		{
			// A library written before runs were recorded, which is not the
			// same as a library from one session: it is a file that cannot
			// say.
			name: "readings naming no run say nothing either",
			lib: measured.Library{
				Blocks: map[string]measured.Block{"a": {}, "b": {}},
			},
		},
		{
			name: "two runs are said, with how many blocks each",
			lib: measured.Library{
				Runs: map[string]measured.Run{
					earlier.Key(): earlier, later.Key(): later,
				},
				Blocks: map[string]measured.Block{
					"a": {Run: earlier.Key()},
					"b": {Run: later.Key()},
					"c": {Run: later.Key()},
				},
			},
			said: true,
		},
		{
			// What resuming the committed library produces: 614 readings that
			// named no run, frozen under the header they were taken with, and
			// this run's beside them.
			name: "carried readings and this run's are two runs",
			lib: measured.Library{
				Runs:   map[string]measured.Run{later.Key(): later},
				Blocks: map[string]measured.Block{"a": {}, "b": {Run: later.Key()}},
			},
			said: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := runs(tt.lib)

			if !tt.said {
				s.Require().Empty(out)

				return
			}

			s.Require().Contains(out, "may be ranked against every other")
			s.Require().Contains(out, "blocks")
		})
	}
}

func TestSameRigTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SameRigTestSuite))
}
