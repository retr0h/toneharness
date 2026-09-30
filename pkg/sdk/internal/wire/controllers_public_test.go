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
package wire_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
)

// ControllersPublicTestSuite covers writing what a pedal moves.
//
// Every case reads the section back with the decoder rather than comparing
// bytes. The decoder is what a device's own answer goes through, so a section
// it cannot read is one the pedal was never going to show, and that is the
// failure this exists to catch before anything is written to flash.
type ControllersPublicTestSuite struct {
	suite.Suite
}

func (s *ControllersPublicTestSuite) blank() *wire.Document {
	doc, err := wire.Blank()
	s.Require().NoError(err)

	return doc
}

// read decodes a document the way a device's answer is read.
func (s *ControllersPublicTestSuite) read(
	doc *wire.Document,
) wire.DevicePreset {
	got, err := wire.DecodePreset(doc.Encode())
	s.Require().NoError(err)

	return got
}

// TestAnAssignmentSurvivesBeingWrittenAndRead is the whole point.
//
// TestPlaceControllers covers PlaceControllers, which writes what an
// expression pedal or a footswitch moves.
//
// One method and one table, so a case is a row rather than a file.
func (s *ControllersPublicTestSuite) TestPlaceControllers() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// the compiler resolved it and the file kept it, and the section never reached
			// the device: place.go wrote the chain and the snapshots and nothing else, so
			// importing such a preset and reading it back showed no controllers at all.
			name: "an assignment survives being written and read",
			then: func() {
				doc := s.blank()

				s.Require().NoError(wire.PlaceControllers(doc, []wire.PlacedController{{
					Controller: 2, Block: 1, Param: 3,
					Min: 0.3, Max: 0.85, NoSnapshot: true,
				}}))

				got := s.read(doc)

				s.Require().Len(got.Controllers, 1)

				one := got.Controllers[0]
				s.Require().Equal(2, one.Controller)
				s.Require().Equal(3, one.Param)
				s.Require().InDelta(0.3, one.Min, 0.0001)
				s.Require().InDelta(0.85, one.Max, 0.0001)
				s.Require().True(one.NoSnapshot)

				// Written as a chain position and stored as a grid one, which is what
				// PlaceAsWritten does with the block it points at. An assignment that did
				// not shift would move whichever block sits one place earlier.
				s.Require().Equal(1+wire.GridOffset, one.Block)
			},
		},
		{
			// from a release that had no such switch rather than as one with it off.
			name: "the switch off is written rather than left out",
			then: func() {
				doc := s.blank()

				s.Require().NoError(wire.PlaceControllers(doc, []wire.PlacedController{{
					Controller: 2, Block: 0, Param: 1,
				}}))

				got := s.read(doc)
				s.Require().Len(got.Controllers, 1)
				s.Require().False(got.Controllers[0].NoSnapshot)
			},
		},
		{
			// two parameters at once.
			name: "two assignments on one controller",
			then: func() {
				doc := s.blank()

				s.Require().NoError(wire.PlaceControllers(doc, []wire.PlacedController{
					{Controller: 2, Block: 1, Param: 3, Min: 0, Max: 1},
					{Controller: 2, Block: 2, Param: 0, Min: 0.2, Max: 0.4},
				}))

				got := s.read(doc)
				s.Require().Len(got.Controllers, 2)

				for _, one := range got.Controllers {
					s.Require().Equal(2, one.Controller)
				}
			},
		},
		{
			name: "controllers on different numbers",
			then: func() {
				doc := s.blank()

				s.Require().NoError(wire.PlaceControllers(doc, []wire.PlacedController{
					{Controller: 0, Block: 1, Param: 1},
					{Controller: 9, Block: 2, Param: 2},
				}))

				got := s.read(doc)
				s.Require().Len(got.Controllers, 2)

				numbers := []int{got.Controllers[0].Controller, got.Controllers[1].Controller}
				s.Require().Contains(numbers, 0)
				s.Require().Contains(numbers, 9)
			},
		},
		{
			// and a file that names none end up saying the file's thing rather than the
			// slot's.
			name: "none clears the section",
			then: func() {
				doc := s.blank()

				s.Require().NoError(wire.PlaceControllers(doc, []wire.PlacedController{{
					Controller: 2, Block: 1, Param: 3,
				}}))
				s.Require().Len(s.read(doc).Controllers, 1)

				s.Require().NoError(wire.PlaceControllers(doc, nil))
				s.Require().Empty(s.read(doc).Controllers)
			},
		},
		{
			// refuse it, so the refusal has to come before the write rather than after.
			name: "a controller the device does not have is refused",
			then: func() {
				tests := []struct {
					name string
					of   wire.PlacedController
				}{
					{"past the last", wire.PlacedController{Controller: 10}},
					{"before the first", wire.PlacedController{Controller: -1}},
					{"a parameter that is not one", wire.PlacedController{Param: -1}},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						err := wire.PlaceControllers(s.blank(), []wire.PlacedController{tt.of})

						s.Require().ErrorIs(err, wire.ErrNoRoom)
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestTheSwitchOffIsWrittenRatherThanLeftOut covers the flags map.
//

// TestTwoAssignmentsOnOneController covers a list rather than a single entry.
//

// TestNoneClearsTheSection covers a preset that assigns nothing.
//

// TestAControllerTheDeviceDoesNotHaveIsRefused covers the bounds.
//

func TestControllersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ControllersPublicTestSuite))
}
