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
package sdk_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/toneharness/pkg/sdk/slot"
)

// playing is the rig the fixture preset reads back as when it is the edit
// buffer, which is what a step matches a favourite against.
//
// Taken from the fixture rather than chosen: a preset lifted off a device has
// no document, so the identifier comes from what the device calls the slot,
// and the edit buffer is read with no slot known.
//
// **It is not an identifier any real rig has, and that is the point of this
// comment.** The device's wire format carries no preset name in a chain
// response and there is no call that reports which slot is selected, so a read
// of the edit buffer can only synthesise one. Every row below that recognises
// what is playing does so by naming its favourite's rig this, which no setup
// anybody writes would. So the recognition these rows cover has never worked
// on hardware, and task 293 is that rather than this constant.
//
// It read `slot-01a` until task 292, which was worse: zero is a real slot, so
// the fiction was indistinguishable from a genuine read of 01A.
const playing = "what-is-loaded"

// FavouritePublicTestSuite covers getting to a favourite slot in one step.
//
// What it is for: a person moves between three or four sounds and leaves the
// other hundred and twenty alone. Selecting any of them by number is the wrong
// shape for the three, because it asks somebody to remember a number while
// they are holding a bass.
//
// The part worth testing hard is where the current position comes from. It is
// read off the pedal rather than remembered between runs, since the footswitch
// is right there and a remembered position would be wrong most of the time.
type FavouritePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *FavouritePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// setupSaying writes a Setup with the body given, as somebody would.
func (s *FavouritePublicTestSuite) setupSaying(
	body string,
) string {
	at := filepath.Join(s.T().TempDir(), "setup.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(
		"schema: Config\nversion: \"1.0.0\"\n"+body), 0o600))

	return at
}

// listens is a device that says what it is playing and takes a selection.
//
// reads is false where the edit buffer cannot be read, which is the case a
// step has to survive: the request was to get to a favourite, and failing to
// recognise where the pedal is now is not a reason to refuse one.
func (s *FavouritePublicTestSuite) listens(
	reads bool,
) *attached {
	dev := &attached{
		MockEditor:   mocks.NewMockEditor(s.ctrl),
		MockWriter:   mocks.NewMockWriter(s.ctrl),
		MockSelector: mocks.NewMockSelector(s.ctrl),
		MockTurner:   mocks.NewMockTurner(s.ctrl),
		MockLoaded:   mocks.NewMockLoaded(s.ctrl),
	}
	dev.MockEditor.EXPECT().Model().
		Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.MockEditor.EXPECT().Close().Return(nil).AnyTimes()

	// Selecting reads the setlist first, so the name in the answer is the one
	// being switched to rather than whatever the device says mid-switch.
	dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
		Return(listing(), nil).AnyTimes()

	if reads {
		body, err := os.ReadFile(
			filepath.Join("internal", "wire", "testdata", "preset.bin"))
		s.Require().NoError(err)

		dev.MockLoaded.EXPECT().ReadCurrent(gomock.Any()).
			Return(body, nil).AnyTimes()
	} else {
		dev.MockLoaded.EXPECT().ReadCurrent(gomock.Any()).
			Return(nil, errors.New("the device said nothing")).AnyTimes()
	}

	return dev
}

// clientOver is a Client over the device given, reading the setup given.
func (s *FavouritePublicTestSuite) clientOver(
	dev *attached,
	config string,
) *sdk.Client {
	bus := mocks.NewMockOpener(s.ctrl)
	bus.EXPECT().Open(gomock.Any()).Return(dev, nil).AnyTimes()

	return sdk.New(sdk.WithDevices(bus), sdk.WithConfig(config))
}

// threeFavourites is a setup listing three, with the one the fixture reads
// back as in the middle so a step either way lands somewhere different.
const threeFavourites = `favourites: ["11A", "19B", "24C"]
slots:
  - { at: 11A, rig: punk }
  - { at: 19B, rig: ` + playing + ` }
  - { at: 24C, rig: mike-dirnt }
`

// TestFavouriteSteps covers which favourite each step lands on.
//
// One method and one table, so a case is a row rather than a file.
func (s *FavouritePublicTestSuite) TestFavouriteSteps() {
	for _, tt := range []struct {
		name string
		// setup is the body of a Setup.
		config string
		step   sdk.Step
		// reads is whether the pedal will say what it is playing.
		reads bool
		// at is the slot the step should land on, and position its place in
		// the list from one.
		at       string
		rig      string
		position int
		of       int
		// from is the favourite it was recognised as having come from, empty
		// where nothing was recognised.
		from string
		// was is what the pedal reported playing, empty where it said
		// nothing.
		was string
	}{
		{
			// The one that needs nothing read off the device, which is why it
			// is the default: it cannot be confused by what is loaded.
			name:   "first goes to the first listed",
			config: threeFavourites,
			step:   sdk.AtFirst, reads: true,
			at: "11A", rig: "punk", position: 1, of: 3,
		},
		{
			name:   "next moves one along from what is playing",
			config: threeFavourites,
			step:   sdk.Onward, reads: true,
			at: "24C", rig: "mike-dirnt", position: 3, of: 3,
			from: "19B", was: playing,
		},
		{
			name:   "prev moves one back from what is playing",
			config: threeFavourites,
			step:   sdk.Back, reads: true,
			at: "11A", rig: "punk", position: 1, of: 3,
			from: "19B", was: playing,
		},
		{
			// A person who has reached the end and presses next again means
			// the first. Refusing there would make them read the list to find
			// out why nothing happened.
			name: "next wraps round the end",
			config: `favourites: ["19B", "11A"]
slots:
  - { at: 19B, rig: ` + playing + ` }
  - { at: 11A, rig: punk }
`,
			step: sdk.Onward, reads: true,
			at: "11A", rig: "punk", position: 2, of: 2,
			from: "19B", was: playing,
		},
		{
			name: "prev wraps round the start",
			config: `favourites: ["19B", "11A"]
slots:
  - { at: 19B, rig: ` + playing + ` }
  - { at: 11A, rig: punk }
`,
			step: sdk.Back, reads: true,
			at: "11A", rig: "punk", position: 2, of: 2,
			from: "19B", was: playing,
		},
		{
			// Not an error. Somebody asking to cycle wanted to be among the
			// favourites, and the first is where that starts.
			name: "a pedal playing no favourite goes to the first",
			config: `favourites: ["11A", "24C"]
slots:
  - { at: 11A, rig: punk }
  - { at: 24C, rig: mike-dirnt }
`,
			step: sdk.Onward, reads: true,
			at: "11A", rig: "punk", position: 1, of: 2,
			was: playing,
		},
		{
			// The slot map is what makes the match possible, so favourites
			// with no entry in it can be jumped to and never recognised.
			name:   "favourites the slot map says nothing about",
			config: `favourites: ["11A", "24C"]`,
			step:   sdk.Onward, reads: true,
			at: "11A", position: 1, of: 2, was: playing,
		},
		{
			// A pedal that will not say what it holds still gets somebody to
			// a favourite, which is what was asked for.
			name:   "a pedal that will not say what it is playing",
			config: threeFavourites,
			step:   sdk.Onward, reads: false,
			at: "11A", rig: "punk", position: 1, of: 3,
		},
		{
			name:   "one favourite is a cycle of one",
			config: `favourites: ["19B"]` + "\nslots:\n  - { at: 19B, rig: " + playing + " }\n",
			step:   sdk.Onward, reads: true,
			at: "19B", rig: playing, position: 1, of: 1, from: "19B", was: playing,
		},
	} {
		s.Run(tt.name, func() {
			dev := s.listens(tt.reads)

			// The slot the step lands on, asserted on the wire rather than
			// only in the answer: what a person cares about is which preset
			// the pedal loaded, and the answer is this tool's account of it.
			at, err := slot.Parse(tt.at)
			s.Require().NoError(err)
			dev.MockSelector.EXPECT().SelectPreset(gomock.Any(), 0, at).
				Return(nil).Times(1)

			got, err := s.clientOver(dev, s.setupSaying(tt.config)).
				Favourite(context.Background(), tt.step)

			s.Require().NoError(err)
			s.Equal(tt.at, got.At, "the slot it landed on")
			s.Equal(tt.rig, got.Rig, "the rig the setup puts there")
			s.Equal(tt.position, got.Position)
			s.Equal(tt.of, got.Of)
			s.Equal(tt.from, got.From, "the favourite it came from")
			s.Equal(tt.was, got.Playing, "what the pedal reported playing")
			s.Equal(sdk.Selected, got.Change.Action)
		})
	}
}

// TestFavouriteRefuses covers the answers that are not a selection.
func (s *FavouritePublicTestSuite) TestFavouriteRefuses() {
	for _, tt := range []struct {
		name string
		// setup is the body of a Setup, or empty to name no setup at all.
		config string
		// named is false where no --setup is given.
		named bool
		// selects is true where the device will be asked to load something,
		// which only the failing-device case is.
		selects bool
		says    string
		is      error
	}{
		{
			name: "no setup at all",
			says: "name one with --setup",
			is:   sdk.ErrNoSlots,
		},
		{
			name:   "a setup listing no favourites",
			config: `match_loudness: punk`, named: true,
			says: "list them under favourites",
			is:   sdk.ErrNoFavourites,
		},
		{
			// Refused by the contract, before anything here looks at it.
			name:   "a favourite that is not a slot label at all",
			config: `favourites: ["99Z"]`, named: true,
			says: `favourites[0]`,
		},
		{
			// Through the contract's pattern and refused by the pedal's own
			// numbering, where banks start at one. Refused before the pedal
			// is claimed, because claiming one takes the front panel off
			// whoever is playing and this is a file to fix.
			name:   "a label the pedal has no position for",
			config: `favourites: ["0A"]`, named: true,
			says: "favourite 0A",
		},
		{
			name:   "a device that will not load it",
			config: `favourites: ["11A"]`, named: true,
			selects: true,
			says:    "going to 11A",
		},
	} {
		s.Run(tt.name, func() {
			dev := s.listens(false)

			if tt.selects {
				dev.MockSelector.EXPECT().
					SelectPreset(gomock.Any(), 0, gomock.Any()).
					Return(errors.New("the device refused")).AnyTimes()
			}

			at := ""
			if tt.named {
				at = s.setupSaying(tt.config)
			}

			_, err := s.clientOver(dev, at).
				Favourite(context.Background(), sdk.AtFirst)

			s.Require().ErrorContains(err, tt.says)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
			}
		})
	}
}

// TestFavouriteTakesTheFirstOfTwoHoldingTheSame covers a layout nothing can
// disambiguate.
//
// Two favourites holding one rig means the edit buffer names a rig both slots
// hold, so either answer is as true as the other. The earlier one wins, which
// at least makes cycling repeatable.
func (s *FavouritePublicTestSuite) TestFavouriteTakesTheFirstOfTwoHoldingTheSame() {
	dev := s.listens(true)
	dev.MockSelector.EXPECT().SelectPreset(gomock.Any(), 0, gomock.Any()).
		Return(nil).Times(1)

	at := s.setupSaying(`favourites: ["11A", "19B", "24C"]
slots:
  - { at: 11A, rig: ` + playing + ` }
  - { at: 19B, rig: punk }
  - { at: 24C, rig: ` + playing + ` }
`)

	got, err := s.clientOver(dev, at).
		Favourite(context.Background(), sdk.Onward)

	s.Require().NoError(err)
	s.Equal("11A", got.From, "the earlier of the two")
	s.Equal("19B", got.At)
}

func TestFavouritePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FavouritePublicTestSuite))
}
