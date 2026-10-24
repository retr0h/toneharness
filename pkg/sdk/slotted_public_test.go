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
)

// SlottedPublicTestSuite covers building every rig a setup places and writing
// it to the slot it names.
//
// The thing that makes "put my pedal back" askable twice. The layout lives on
// the setup because it is a preference about somebody's own pedal: two people
// building one rig want the same preset and different slots, since one of them
// already keeps their own work in bank one.
type SlottedPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *SlottedPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// absent is a bus that finds nothing, which every client here is built over
// unless the case is about a device taking a write.
//
// Never omitted. A `sdk.New` with no `WithDevices` opens the real USB bus, and
// one of these tests did: it built punk and wrote it to slot 11B of the pedal
// on the desk. The slot was empty so nothing was lost, which was luck. A test
// must not reach somebody's hardware, and `WithDevices` is the only thing
// standing between it and flash.
func (s *SlottedPublicTestSuite) absent() sdk.Option {
	bus := mocks.NewMockOpener(s.ctrl)
	bus.EXPECT().Open(gomock.Any()).
		Return(nil, errors.New("no device found")).AnyTimes()

	return sdk.WithDevices(bus)
}

// setupSaying writes a Setup naming the slots given, as somebody would.
func (s *SlottedPublicTestSuite) setupSaying(
	body string,
) string {
	at := filepath.Join(s.T().TempDir(), "setup.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(
		"schema: Setup\nversion: \"4.2.0\"\n"+body), 0o600))

	return at
}

// pedal is a Client over a device that takes every write, which is what a
// successful fill meets.
func (s *SlottedPublicTestSuite) pedal(
	setup string,
) *sdk.Client {
	body, err := os.ReadFile(
		filepath.Join("internal", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	dev := &attached{
		MockEditor:   mocks.NewMockEditor(s.ctrl),
		MockWriter:   mocks.NewMockWriter(s.ctrl),
		MockSelector: mocks.NewMockSelector(s.ctrl),
		MockTurner:   mocks.NewMockTurner(s.ctrl),
		MockLoaded:   mocks.NewMockLoaded(s.ctrl),
	}
	dev.MockEditor.EXPECT().Model().
		Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
		Return(listing(), nil).AnyTimes()
	dev.MockEditor.EXPECT().
		ReadPreset(gomock.Any(), 0, gomock.Any()).
		Return(body, nil).AnyTimes()
	dev.MockEditor.EXPECT().Close().Return(nil).AnyTimes()
	dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	bus := mocks.NewMockOpener(s.ctrl)
	bus.EXPECT().Open(gomock.Any()).Return(dev, nil).AnyTimes()

	return sdk.New(
		sdk.WithDevices(bus),
		sdk.WithSetup(setup),
		sdk.WithBackupDir(s.T().TempDir()))
}

// TestFillRefusesWhatCannotBeRead covers the answers that are not a write.
//
// One method and one table, so a case is a row rather than a file.
func (s *SlottedPublicTestSuite) TestFillRefusesWhatCannotBeRead() {
	for _, tt := range []struct {
		name string
		// setup is the body of a Setup, or empty to name no setup at all.
		setup string
		// named is false where no --setup is given.
		named bool
		says  string
		is    error
	}{
		{
			// Nothing is written on anybody's behalf, which is the right
			// default: a device has no undo and a slot holds somebody's work
			// until they say otherwise.
			name:  "no setup at all",
			named: false,
			says:  "--setup",
			is:    sdk.ErrNoSlots,
		},
		{
			name:  "a setup naming no slots",
			setup: "device:\n  model: HX Stomp\n",
			named: true,
			is:    sdk.ErrNoSlots,
		},
		{
			name:  "a setup whose slot list is empty",
			setup: "slots: []\n",
			named: true,
			is:    sdk.ErrNoSlots,
		},
		{
			name:  "a setup whose slots are not a list",
			setup: "slots: \"not a list\"\n",
			named: true,
			says:  "slots",
		},
		{
			// The contract gives `at` a pattern, so a label the pedal could
			// never print is refused before anything is built. Forty presets
			// compiled to discover the first address is unreadable would be
			// forty seconds for nothing.
			name:  "a slot label the pedal would never print",
			setup: "slots:\n  - { at: \"not a slot\", rig: punk }\n",
			named: true,
			says:  "at",
		},
	} {
		s.Run(tt.name, func() {
			at := ""
			if tt.named {
				at = s.setupSaying(tt.setup)
			}

			placed, err := sdk.New(s.absent(), sdk.WithSetup(at)).
				Fill(context.Background())

			s.Require().Error(err)
			s.Require().Empty(placed)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
			}

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)
			}
		})
	}
}

// TestFillSaysWhichSlotThePedalHasNoPositionFor covers a label the contract's
// pattern lets through and the pedal's own numbering does not.
//
// Through the pattern because it is digits and a letter, refused here because
// banks start at one. On its own entry rather than failing the run, for the
// reason a rig that will not build is: the other forty still go.
func (s *SlottedPublicTestSuite) TestFillSaysWhichSlotThePedalHasNoPositionFor() {
	at := s.setupSaying(
		"slots:\n" +
			"  - { at: \"0A\", rig: punk }\n" +
			"  - { at: \"11B\", rig: pop-punk }\n")

	placed, err := sdk.New(s.absent(), sdk.WithSetup(at)).
		Fill(context.Background())

	s.Require().Error(err)
	s.Require().Len(placed, 2)

	s.Require().Equal("0A", placed[0].At)
	s.Require().ErrorContains(placed[0].Err, `slot "0A" for "punk"`)
	s.Require().Contains(placed[0].Why, "0A")

	// And the next one was still attempted, which is the whole shape of this:
	// one bad address does not cost the rest of the layout.
	s.Require().Equal("11B", placed[1].At)
}

// TestFillReadsNoSetupFile covers a path that is not there.
func (s *SlottedPublicTestSuite) TestFillReadsNoSetupFile() {
	at := filepath.Join(s.T().TempDir(), "nothing.yaml")

	_, err := sdk.New(s.absent(), sdk.WithSetup(at)).Fill(context.Background())

	s.Require().ErrorContains(err, "nothing.yaml")
}

// TestFillSaysWhichRigWouldNotGo covers one entry failing without taking the
// rest with it.
//
// A rig that will not build is a document to fix, and leaving the other forty
// written beats leaving a pedal half updated with no record of which half. The
// failure travels on its own entry so a caller can say which.
func (s *SlottedPublicTestSuite) TestFillSaysWhichRigWouldNotGo() {
	at := s.setupSaying(
		"slots:\n" +
			"  - { at: \"11A\", rig: nobody-has-this-rig }\n" +
			"  - { at: \"11B\", rig: punk }\n")

	placed, err := sdk.New(s.absent(), sdk.WithSetup(at)).
		Fill(context.Background())

	// The error joins them, so a caller that only checks the error still
	// learns something went wrong.
	s.Require().Error(err)
	s.Require().Len(placed, 2)

	s.Require().Error(placed[0].Err)
	s.Require().Equal("11A", placed[0].At)
	s.Require().ErrorContains(placed[0].Err, "nobody-has-this-rig")

	// The same reason as text, because an error marshals to `{}` and a
	// caller reading the answer over MCP would learn a rig failed and
	// nothing about why. Both are set together.
	s.Require().Contains(placed[0].Why, "nobody-has-this-rig")

	// And the one after it was still attempted, which is the point: a rig
	// nobody has is a document to fix rather than a reason to leave a pedal
	// half updated. It fails too, for want of a device.
	s.Require().Equal("11B", placed[1].At)
	s.Require().Error(placed[1].Err)
}

// TestFillReportsNowhereToBuild covers losing the directory the presets are
// built in on the way to the pedal.
//
// Nothing is written when this fails, which is the right answer: these are a
// means of getting bytes to flash rather than something to keep, and with
// nowhere to put them there is nothing to write.
func (s *SlottedPublicTestSuite) TestFillReportsNowhereToBuild() {
	s.T().Setenv("TMPDIR", filepath.Join(s.T().TempDir(), "not-there"))

	at := s.setupSaying("slots:\n  - { at: \"11A\", rig: punk }\n")

	placed, err := sdk.New(s.absent(), sdk.WithSetup(at)).
		Fill(context.Background())

	s.Require().ErrorContains(err, "making somewhere to build the presets")
	s.Require().Empty(placed)
}

// TestFillWritesEveryRigToItsSlot is the whole point, against a device that
// takes the writes.
func (s *SlottedPublicTestSuite) TestFillWritesEveryRigToItsSlot() {
	at := s.setupSaying(
		"slots:\n" +
			"  - { at: \"11A\", rig: punk }\n" +
			"  - { at: \"11B\", rig: pop-punk }\n")

	placed, err := s.pedal(at).Fill(context.Background())
	s.Require().NoError(err)
	s.Require().Len(placed, 2)

	s.Require().Equal("11A", placed[0].At)
	s.Require().Equal("punk", placed[0].Rig)
	s.Require().NoError(placed[0].Err)

	s.Require().Equal("11B", placed[1].At)
	s.Require().Equal("pop-punk", placed[1].Rig)
	s.Require().NoError(placed[1].Err)

	// In the order the setup wrote them, which is why the contract keeps a
	// list rather than a map: somebody reading it wants the layout they chose.
	s.Require().Equal(
		[]string{"punk", "pop-punk"},
		[]string{placed[0].Rig, placed[1].Rig})
}

// TestFillSaysWhenTheDeviceWillNotTakeIt covers the write failing rather than
// the build.
func (s *SlottedPublicTestSuite) TestFillSaysWhenTheDeviceWillNotTakeIt() {
	at := s.setupSaying("slots:\n  - { at: \"11A\", rig: punk }\n")

	placed, err := sdk.New(s.absent(), sdk.WithSetup(at)).
		Fill(context.Background())

	s.Require().Error(err)
	s.Require().Len(placed, 1)
	s.Require().ErrorContains(placed[0].Err, "punk")
	s.Require().ErrorContains(placed[0].Err, "11A")
}

func TestSlottedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SlottedPublicTestSuite))
}
