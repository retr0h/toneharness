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

// RigTakenTestSuite covers recording which rig took a reading.
//
// AGENTS.md states the rule this exists for: a figure measured on one device is
// a figure about that device, say which. The two rigs disagree even about a
// block neither refused, an Ampeg SVT reading 12.7% of its energy in the mids
// on one and 4.0% on the other.
type RigTakenTestSuite struct {
	suite.Suite
}

// TestRigTaken covers the devices a reading names, and whether the loop was
// open.
func (s *RigTakenTestSuite) TestRigTaken() {
	for _, tt := range []struct {
		name string
		in   string
		want []string
		open bool
	}{
		{
			// Two names, so one device plays and another records and the
			// chain cannot hear its own output.
			name: "two devices",
			in:   "External Headphones,HX Stomp",
			want: []string{"External Headphones", "HX Stomp"},
			open: true,
		},
		{
			// One name is both sides, which is the closed loop: the pedal
			// plays the reference out of the socket its own chain comes out
			// of, so the chain hears itself.
			name: "one device, which is both sides",
			in:   "HX Stomp",
			want: []string{"HX Stomp", "HX Stomp"},
		},
		{
			// Spelt with a space after the comma, which somebody will.
			name: "two devices spelt loosely",
			in:   "External Headphones, HX Stomp",
			want: []string{"External Headphones", "HX Stomp"},
			open: true,
		},
		{
			// Nothing named is not a rig of one device, it is a rig nobody
			// wrote down. Empty rather than a guess, because a reading that
			// cannot name its devices must not claim them.
			name: "nothing named",
		},
	} {
		s.Run(tt.name, func() {
			got, open := rigTaken(tt.in)

			s.Require().Equal(tt.want, got)
			s.Require().Equal(tt.open, open)
		})
	}
}

// TestPinnedOn covers naming the output device the level reached.
//
// The field that silently invalidates the others: a Mac with its default output
// on Bluetooth headphones pins those while the jack feeding the pedal sits
// where it was left, which is how a run read 66dB of loss with the cabling
// correct.
func (s *RigTakenTestSuite) TestPinnedOn() {
	for _, tt := range []struct {
		name string
		says func() (string, error)
		want string
	}{
		{
			// What the platform answered, as it answered it.
			name: "a platform that names the device",
			says: func() (string, error) { return "External Headphones", nil },
			want: "External Headphones",
		},
		{
			// A platform that will not say. Empty rather than a guess, which
			// is the same answer the volume gives in that case: a reading
			// that cannot name the device its level reached should not claim
			// one.
			name: "a platform that will not say",
			says: func() (string, error) { return "", errNoOutput },
		},
	} {
		s.Run(tt.name, func() {
			was := saysOutputFor(tt.says)
			defer was()

			s.Require().Equal(tt.want, pinnedOn())
		})
	}
}

func TestRigTakenTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RigTakenTestSuite))
}

// errNoOutput stands in for a platform that will not name its output device.
var errNoOutput = errors.New("no output device")

// saysOutputFor stands a reader in for the platform's, and returns the undo.
func saysOutputFor(
	with func() (string, error),
) func() {
	was := saysOutput
	saysOutput = with

	return func() { saysOutput = was }
}
