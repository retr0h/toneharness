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

package wire

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// FootswitchesTestSuite covers writing what each switch on the pedal works on.
type FootswitchesTestSuite struct {
	suite.Suite
}

// switchesIn reads back what a document's footswitch section holds, by group.
//
// Through [DecodePreset] rather than by decoding the section here, because the
// reader is what a device's own answer goes through and a writer that only
// satisfies a bespoke parser is a writer nothing has checked.
func (s *FootswitchesTestSuite) switchesIn(
	doc *Document,
) []DeviceFootswitch {
	got, err := DecodePreset(doc.Encode())
	s.Require().NoError(err)

	return got.Footswitches
}

// TestPlaceFootswitches writes a chain's switches over whatever a slot held.
func (s *FootswitchesTestSuite) TestPlaceFootswitches() {
	tests := []struct {
		name string
		// held is what the document already carries, written first to stand
		// for the template a preset is compiled from.
		held []DeviceFootswitch
		// write is what the chain names.
		write []DeviceFootswitch
		// want is what the section must hold afterwards.
		want []DeviceFootswitch
	}{
		{
			name: "nothing to write leaves the section alone",
			held: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Teemah!", Label: "Drive", LED: 4},
			},
			write: nil,
			want: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Teemah!", Label: "Drive", LED: 4},
			},
		},
		{
			name: "one switch on an untouched slot",
			write: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
			},
			want: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
			},
		},
		{
			// The leak this guards. A preset compiled from a template used to
			// inherit that template's assignments on every switch the new
			// chain did not name, so a chain with one drive reached the pedal
			// with a second switch carrying an empty name and pointing at a
			// block it never placed. Stepping on one moved the other.
			name: "a switch the chain does not name is emptied",
			held: []DeviceFootswitch{
				{Switch: 1, Block: 9, Gear: "Clawthorn Drive", Label: "FUZZ", LED: 3},
				{Switch: 2, Block: 4, Gear: "Bass Octaver", Label: "Oct -1", LED: 9},
			},
			write: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
			},
			want: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
			},
		},
		{
			name: "two blocks on one switch both arrive",
			write: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
				{Switch: 1, Block: 3, Gear: "Deluxe Comp", Label: "Comp", LED: 2},
			},
			want: []DeviceFootswitch{
				{Switch: 1, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
				{Switch: 1, Block: 3, Gear: "Deluxe Comp", Label: "Comp", LED: 2},
			},
		},
		{
			name: "a switch past what the device sent has nowhere to go",
			write: []DeviceFootswitch{
				{Switch: 99, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := Blank()
			s.Require().NoError(err)

			if len(tt.held) > 0 {
				PlaceFootswitches(doc, tt.held)
				s.Require().Equal(tt.held, s.switchesIn(doc))
			}

			PlaceFootswitches(doc, tt.write)

			s.Require().Equal(tt.want, s.switchesIn(doc))
		})
	}
}

// TestPlaceFootswitchesWithoutASection writes into a document that has none.
//
// Nothing here fails: a preset with no footswitch section has nothing to write
// into, and this assigns no block and moves no control, so there is no claim
// about the sound to get wrong.
func (s *FootswitchesTestSuite) TestPlaceFootswitchesWithoutASection() {
	doc, err := Blank()
	s.Require().NoError(err)

	doc.SetSection(int8(keyFootswitch), nil)

	PlaceFootswitches(doc, []DeviceFootswitch{
		{Switch: 1, Block: 2, Gear: "Ampeg Scrambler", Label: "Scrambler", LED: 2},
	})
}

func TestFootswitchesTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FootswitchesTestSuite))
}
