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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

// UnaccountableTestSuite covers telling a control that mutes a chain from a
// computer whose playback stopped.
//
// Both arrive as silence in the middle of a sweep, and the tool reported the
// second as the first for a campaign: three of eleven amplifiers were lost
// and the message sent somebody looking at the wrong control for an hour. The
// shape of the positions is what tells them apart.
type UnaccountableTestSuite struct {
	suite.Suite
}

// TestUnaccountable covers which shapes a control can account for.
//
// One method and one table, so a case is a row rather than a file.
func (s *UnaccountableTestSuite) TestUnaccountable() {
	for _, tt := range []struct {
		name string
		// carried is whether each position, in order, came back with signal.
		carried []bool
		says    bool
	}{
		{
			// The reading this exists for. Cali 400 Ch2's Hum: -47.47 and
			// -47.46dB, six silent positions, then -58.67dB at the top. No
			// dial mutes an amplifier across the middle of its range and
			// un-mutes past it.
			name: "signal, then silence, then signal again",
			carried: []bool{
				true, true, false, false, false, false, false, false, true,
			},
			says: true,
		},
		{
			// Tuck n Go's Bass, worse and the same: one position of signal on
			// an Ampeg B-15's bass control. The run touches neither end, so
			// the control would have to mute at both extremes of its own
			// travel.
			name:    "one position of signal with silence either side",
			carried: []bool{false, false, false, true, false, false, false},
			says:    true,
		},
		{
			// A Level swept from -60: silent for its first four positions and
			// signal for the rest. A reading about the control, and the shape
			// every committed sweep of a Level has.
			name:    "silence first and signal after it",
			carried: []bool{false, false, false, false, true, true, true, true, true},
		},
		{
			// The same thing reversed, which is a control that mutes as it is
			// turned up.
			name:    "signal first and silence after it",
			carried: []bool{true, true, true, false, false, false},
		},
		{
			name:    "every position carried signal",
			carried: []bool{true, true, true},
		},
		{
			// The whole control silent, which is a different answer: the
			// caller keeps no curve and says so. Not this.
			name:    "no position carried signal",
			carried: []bool{false, false, false},
		},
		{
			name:    "a single position, which has no shape to read",
			carried: []bool{true},
		},
		{
			// Every position refused, so nothing was played anywhere and
			// there is nothing to conclude.
			name: "nothing was measured at all",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.says, unaccountable(tt.carried))
		})
	}
}

// TestDroppedOut covers what the refusal says.
//
// One method and one table, so a case is a row rather than a file.
func (s *UnaccountableTestSuite) TestDroppedOut() {
	for _, tt := range []struct {
		name  string
		level func() (int, error)
		says  string
	}{
		{
			// The level is the figure that tells the two apart, and the one
			// the third of the three failures reported on its own: asked for
			// 38 and came back at 0.
			name:  "the level reads back",
			level: func() (int, error) { return 0, nil },
			says:  "output level reads 0",
		},
		{
			// Said rather than guessed at. A refusal that cannot read the
			// level still names the shape it saw, because that is the part
			// the person has to act on.
			name:  "the level cannot be read",
			level: func() (int, error) { return 0, errors.New("no output device") },
			says:  "could not be read back",
		},
	} {
		s.Run(tt.name, func() {
			err := droppedOut("Hum", tt.level)

			s.Require().ErrorIs(err, errPlaybackDropped)
			s.Require().ErrorContains(err, "Hum")
			s.Require().ErrorContains(err, tt.says)
			s.Require().ErrorContains(err, "the chain is not what went quiet",
				"the point of the message is which thing to look at")
		})
	}
}

func TestUnaccountableTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UnaccountableTestSuite))
}
