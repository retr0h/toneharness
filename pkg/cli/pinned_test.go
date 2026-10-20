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
)

// PinnedTestSuite covers refusing a campaign whose output level is not the one
// it asked for.
//
// The level is a tone control rather than a level control, because an
// amplifier's distortion depends on how hard it is driven. So two runs at
// different levels measure two different amplifiers, and a run at a level
// nobody asked for measures neither.
type PinnedTestSuite struct {
	suite.Suite
}

// TestPinned covers which reached levels a campaign may carry on from.
//
// One method and one table, so a case is a row rather than a file.
func (s *PinnedTestSuite) TestPinned() {
	for _, tt := range []struct {
		name          string
		want, reached int
		says          string
	}{
		{
			// The reading this exists for. `FM4 Throbber` asked for 38, the
			// platform set it, read it back as 0 and said so, and eight
			// controls were then swept through a muted output: -122dB with
			// centroids between 1630 and 2390Hz.
			name:    "muted when it was asked for a level",
			want:    38,
			reached: 0,
			says:    "asked for 38 and came back at 0",
		},
		{
			name:    "the level it asked for",
			want:    38,
			reached: 38,
		},
		{
			// A platform that rounds. macOS does it on some hardware, which is
			// why `reamp.Held` reads back at all, and a level near the one
			// asked for is still not the one asked for: the figures belong to
			// where it landed and nothing else records that.
			name:    "rounded to its nearest step",
			want:    38,
			reached: 37,
			says:    "asked for 38 and came back at 37",
		},
		{
			// A platform that will not say. Nothing can be checked, and
			// `unknownVolume` is what the file records in place of a level.
			name:    "a platform that will not say",
			want:    38,
			reached: unknownVolume,
		},
		{
			// Asked for silence and got it, which is somebody's choice rather
			// than a fault. Nothing here second-guesses the level, only
			// whether it is the one that was asked for.
			name:    "asked for zero and reached zero",
			want:    0,
			reached: 0,
		},
	} {
		s.Run(tt.name, func() {
			err := pinned(tt.want, tt.reached)

			if tt.says == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, errNotPinned)
			s.Require().ErrorContains(err, tt.says)
		})
	}
}

func TestPinnedTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PinnedTestSuite))
}
