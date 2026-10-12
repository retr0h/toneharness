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

	"github.com/retr0h/toneharness/pkg/sdk"
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
// TestPlaysThrough covers setting which device the computer plays through.
//
// The level is pinned on whichever device the platform plays through, so a rig
// that plays the reference out of the computer has to agree about which one that
// is. It did not, and nothing said so: a Mac with its default on a pair of
// Bluetooth headphones pinned those while the jack feeding the pedal sat
// wherever it was left, and the loop read 66dB of loss.
//
// One method and one table, so a case is a row rather than a file.
func (s *VolumeTestSuite) TestPlaysThrough() {
	for _, tt := range []struct {
		name string
		// hardware is what the run was given.
		hardware string
		// at, moved and err are what the platform answers.
		at    string
		moved bool
		err   error
		// asked is the device the setter was told to use, empty for never called.
		asked string
		says  string
	}{
		{
			// The pedal plays, so the computer's output is not in the path.
			name:     "one device plays and records",
			hardware: "HX Stomp",
		},
		{
			name:     "the device was already the one that plays",
			hardware: "External Headphones,HX Stomp",
			at:       "External Headphones",
			asked:    "External Headphones",
			says:     "which the level below belongs to",
		},
		{
			// The case this exists for.
			name:     "the device had to be moved",
			hardware: "External Headphones,HX Stomp",
			at:       "External Headphones",
			moved:    true,
			asked:    "External Headphones",
			says:     "the computer now plays through External Headphones",
		},
		{
			// Warned and carried on from, because a rig somebody sets by hand is
			// still a rig, and refusing would stop a campaign over a setting that
			// may already be right.
			name:     "a platform that will not be told",
			hardware: "External Headphones,HX Stomp",
			err:      errors.New("no such thing here"),
			asked:    "External Headphones",
			says:     "could not be set",
		},
	} {
		s.Run(tt.name, func() {
			asked := ""

			w := buffer()

			playsThrough(w, TuneOptions{Hardware: tt.hardware},
				func(want string) (string, bool, error) {
					asked = want

					return tt.at, tt.moved, tt.err
				})

			s.Require().Equal(tt.asked, asked,
				"the playback half of --hardware, or nothing at all")

			if tt.says == "" {
				s.Require().Empty(w.String())

				return
			}

			s.Require().Contains(w.String(), tt.says)
		})
	}
}

// pins is a bench that says whether the pinned level is its own.
type pins struct {
	bench

	reaches bool
}

func (p pins) PinReaches() bool { return p.reaches }

func (pins) Name() string { return "External Headphones into HX Stomp" }

// TestPinLands covers the check that the pinned level reached the device the
// reference plays through.
//
// Belt and braces over `playsThrough`, and worth it because the two halves match
// a device name against two different lists: `--hardware` against the ones the
// audio backend enumerates, and the output switch against the ones CoreAudio
// does. Agreeing on a name is not the same as agreeing on a device.
//
// One method and one table, so a case is a row rather than a file.
func (s *VolumeTestSuite) TestPinLands() {
	for _, tt := range []struct {
		name     string
		hardware string
		bench    sdk.Bench
		says     string
	}{
		{
			// The pedal plays, so nothing is pinned into the path.
			name:     "one device plays and records",
			hardware: "HX Stomp",
			bench:    pins{},
		},
		{
			name:     "the pin reached the device that plays",
			hardware: "External Headphones,HX Stomp",
			bench:    pins{reaches: true},
		},
		{
			name:     "the pin reached something else",
			hardware: "External Headphones,HX Stomp",
			bench:    pins{},
			says:     "not in the signal path",
		},
		{
			// A bench that does not claim to know, which is every double but
			// one and the plain reader a test hands in.
			name:     "a bench that cannot say",
			hardware: "External Headphones,HX Stomp",
			bench:    bench{},
		},
	} {
		s.Run(tt.name, func() {
			w := buffer()

			pinLands(w, TuneOptions{Hardware: tt.hardware}, tt.bench)

			if tt.says == "" {
				s.Require().Empty(w.String())

				return
			}

			s.Require().Contains(w.String(), tt.says)
		})
	}
}

func (s *VolumeTestSuite) TestALevelNobodyCheckedIsNotZero() {
	s.Require().Equal(-1, unknownVolume)
	s.Require().NotEqual(0, unknownVolume)
}

func TestVolumeTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VolumeTestSuite))
}
