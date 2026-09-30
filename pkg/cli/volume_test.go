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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// VolumeTestSuite covers pinning the computer's own output level.
//
// The level reaches the signal on the rig that plays the reference out of the
// computer, so it is a tone control: an amplifier's distortion depends on how
// hard it is driven. What this answers is written into every library as the
// thing that makes a campaign reproducible, so all four branches are worth
// pinning, and each one's message with it.
type VolumeTestSuite struct {
	suite.Suite
}

func (s *VolumeTestSuite) TestLevelled() {
	declined := errors.New("the platform declined")

	tests := []struct {
		name string
		hold holds
		want int
		says string
	}{
		{
			name: "already where a campaign wants it",
			hold: func(int) (int, bool, error) { return 38, false, nil },
			want: 38,
			says: "is 38, where a campaign wants it",
		},
		{
			name: "moved there, which is worth saying because it reaches " +
				"outside this tool",
			hold: func(int) (int, bool, error) { return 38, true, nil },
			want: 38,
			says: "moved to 38",
		},
		{
			// Not a failure. It is only in the path on one of the two rigs,
			// so refusing would stop a campaign on the other for a reason
			// that does not apply to it.
			name: "a platform that will not say",
			hold: func(int) (int, bool, error) { return 0, false, reamp.ErrNoVolume },
			want: unknownVolume,
			says: "will not say what its output level is",
		},
		{
			name: "a platform that would and did not",
			hold: func(int) (int, bool, error) {
				return 0, false, &reamp.VolumeError{Doing: "setting", Said: declined}
			},
			want: unknownVolume,
			says: "could not be set to 38",
		},
		{
			name: "anything else is said rather than described",
			hold: func(int) (int, bool, error) { return 0, false, declined },
			want: unknownVolume,
			says: "the platform declined",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			got := levelled(&buf, 38, tt.hold)

			s.Require().Equal(tt.want, got)
			s.Require().Contains(buf.String(), tt.says)
		})
	}
}

// TestALevelNobodyCheckedIsNotZero is the reason unknownVolume is -1.
//
// Zero is a real output level. A library recording 0 where nothing could be
// read would claim the reference was played in silence.
func (s *VolumeTestSuite) TestALevelNobodyCheckedIsNotZero() {
	s.Require().Equal(-1, unknownVolume)
	s.Require().NotEqual(0, unknownVolume)
}

func TestVolumeTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VolumeTestSuite))
}
