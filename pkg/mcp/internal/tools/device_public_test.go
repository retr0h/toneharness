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

package tools_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/mcp/internal/tools"
	"github.com/retr0h/toneharness/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/slot"
)

type DevicePublicTestSuite struct {
	suite.Suite
	ctrl   *gomock.Controller
	client *mocks.MockClient
}

func (s *DevicePublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
	s.client = mocks.NewMockClient(s.ctrl)
}

// deviceRow is one call and what it should come back with. check reads the
// structured answer of a call that succeeded.
type deviceRow struct {
	name string
	args any
	// setup says what the call expects of the client, and of the Session the
	// tools hold on the pedal.
	setup func(c *mocks.MockClient, pedal *mocks.MockSession)
	want  string
	err   bool
	check func(s *DevicePublicTestSuite, res *gomcp.CallToolResult)
	// allowWrites starts the server the way --allow-writes does.
	allowWrites bool
	// moved says this row's Change replaced nothing, which is what a move
	// answers with. Every other row carries the name the mock returned.
	moved bool
}

// held lets the tools open the pedal once, onto pedal, and close it when they
// let it go.
//
// Set after a row's own expectations, which gomock matches first, so a row
// can make opening fail instead.
func held(
	c *mocks.MockClient,
	pedal *mocks.MockSession,
) {
	c.EXPECT().Open(gomock.Any()).Return(pedal, nil).MaxTimes(1)
	pedal.EXPECT().Close().Return(nil).MaxTimes(1)
}

func (s *DevicePublicTestSuite) run(
	tool string,
	tests []deviceRow,
) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			pedal := mocks.NewMockSession(s.ctrl)

			if tt.setup != nil {
				tt.setup(s.client, pedal)
			}

			held(s.client, pedal)

			res := call(s.T(), connect(s.T(), s.client, tt.allowWrites), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if tt.check != nil {
				tt.check(s, res)
			}
		})
	}
}

var errHXEdit = errors.New("the editor interface is in use, quit HX Edit")

// hxEdit is HX Edit holding the pedal, so the tools cannot open it.
func hxEdit(
	c *mocks.MockClient,
	_ *mocks.MockSession,
) {
	c.EXPECT().Open(gomock.Any()).Return(nil, errHXEdit)
}

// TestDevicesList covers what is attached, and one call at a time.
func (s *DevicePublicTestSuite) TestDevicesList() {
	s.run("device_hardware", []deviceRow{
		{
			name: "a pedal attached",
			args: tools.None{},
			setup: func(c *mocks.MockClient, _ *mocks.MockSession) {
				c.EXPECT().Devices(gomock.Any()).
					Return(sdk.Attached{Devices: []sdk.Attachment{{Model: "HX Stomp"}}}, nil)
			},
			want: "1 attached",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Attached
				structured(s.T(), res, &got)
				s.Equal("HX Stomp", got.Devices[0].Model)
			},
		},
		{
			name: "a bus that will not answer",
			args: tools.None{},
			setup: func(c *mocks.MockClient, _ *mocks.MockSession) {
				c.EXPECT().
					Devices(gomock.Any()).
					Return(sdk.Attached{}, errors.New("bus unavailable"))
			},
			want: "bus unavailable",
			err:  true,
		},
	})

	s.Run("two calls at once", func() {
		// A barrier and a counter rather than a sleep: the first call holds
		// the device until the test lets it go, and how many calls were ever
		// inside at once is read after both have finished.
		var inside, most atomic.Int32

		entered := make(chan struct{}, 2)
		release := make(chan struct{})

		s.client.EXPECT().Devices(gomock.Any()).Times(2).DoAndReturn(
			func(context.Context) (sdk.Attached, error) {
				now := inside.Add(1)
				defer inside.Add(-1)

				for {
					seen := most.Load()
					if now <= seen || most.CompareAndSwap(seen, now) {
						break
					}
				}

				entered <- struct{}{}
				<-release

				return sdk.Attached{}, nil
			})

		session := connect(s.T(), s.client, false)
		devices := func() {
			_, _ = session.CallTool(context.Background(), &gomcp.CallToolParams{
				Name: "device_hardware", Arguments: tools.None{},
			})
		}

		var wg sync.WaitGroup

		wg.Go(devices)

		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			close(release)
			wg.Wait()
			s.FailNow("the first call never reached the device")
		}

		wg.Go(devices)
		close(release)
		wg.Wait()

		s.Len(entered, 1, "the second call reached the device too")
		s.Equal(int32(1), most.Load(), "never two calls inside at once")
	})
}

// TestPresetsList covers reading the setlist.
func (s *DevicePublicTestSuite) TestPresetsList() {
	s.run("slots_list", []deviceRow{
		{
			name: "a setlist",
			args: tools.None{},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Presets(gomock.Any(), 0).
					Return(sdk.Listing{Slots: []sdk.Held{{}, {}}}, nil)
			},
			want: "of 2 slots",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Listing
				structured(s.T(), res, &got)
				s.Len(got.Slots, 2)
			},
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.None{},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestPresetShow covers reading one slot.
func (s *DevicePublicTestSuite) TestPresetShow() {
	s.run("presets_show", []deviceRow{
		{
			name: "a slot by its label",
			args: tools.Slot{Slot: "01B"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Preset(gomock.Any(), slot.Address{Slot: 1}).
					Return(sdk.Reading{Name: "Chunky Monkey"}, nil)
			},
			want: "01B holds Chunky Monkey",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Shown
				structured(s.T(), res, &got)
				s.Equal("Chunky Monkey", got.Name)
			},
		},
		{
			// "99Z" fails to parse: no bank has a letter past C, so
			// slot.Value.Set refuses it before the pedal is ever opened.
			name: "a label the pedal does not have",
			args: tools.Slot{Slot: "99Z"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Slot{Slot: "01A"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestPresetExport covers writing a slot out.
func (s *DevicePublicTestSuite) TestPresetExport() {
	dir := s.T().TempDir()
	fresh := filepath.Join(dir, "fresh.yaml")
	taken := filepath.Join(dir, "held.yaml")
	s.Require().NoError(os.WriteFile(taken, []byte("somebody's rig"), 0o600))
	racy := filepath.Join(dir, "racy.yaml")
	// A setlist whose first slot holds a preset, so a real export has
	// something to write.
	setlist := filepath.Join("..", "..", "..", "sdk", "testdata", "setlist.hls")

	s.run("slots_export", []deviceRow{
		{
			name: "a path nothing is at",
			args: tools.Export{Slot: "01A", Out: fresh},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Export(gomock.Any(), slot.Address{}, fresh, sdk.FormatRig, sdk.KeepExisting).
					Return(sdk.Written{Path: fresh}, nil)
			},
			want: "wrote " + fresh,
		},
		{
			// No Session call is expected, so reaching the device fails the
			// row.
			name: "a path a file is at, with writes off",
			args: tools.Export{Slot: "01A", Out: taken},
			want: tools.ErrWouldOverwrite.Error() + ": " + taken,
			err:  true,
		},
		{
			name: "a path a file is at, with writes on",
			args: tools.Export{Slot: "01A", Out: taken},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Export(gomock.Any(), slot.Address{}, taken, sdk.FormatRig, sdk.ReplaceExisting).
					Return(sdk.Written{Path: taken}, nil)
			},
			want:        "wrote " + taken,
			allowWrites: true,
		},
		{
			// Nothing is at the path when the tool decides, and somebody's
			// rig is by the time the file is written. The hook lands it
			// there and then exports for real, through the write a device
			// export shares with a setlist's, so what refuses it can only be
			// the write itself.
			name: "a file that appears between the decision and the write, with writes off",
			args: tools.Export{Slot: "01A", Out: racy},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Export(gomock.Any(), slot.Address{}, racy, sdk.FormatRig, sdk.KeepExisting).
					DoAndReturn(func(
						ctx context.Context, at slot.Address, out string, as sdk.Format,
						existing sdk.Existing,
					) (sdk.Written, error) {
						if err := os.WriteFile(out, []byte("somebody's rig"), 0o600); err != nil {
							return sdk.Written{}, err
						}

						return sdk.New().Setlist(setlist).Export(ctx, at, out, as, existing)
					})
			},
			want: tools.ErrWouldOverwrite.Error() + ": " + racy,
			err:  true,
			check: func(s *DevicePublicTestSuite, _ *gomcp.CallToolResult) {
				got, err := os.ReadFile(racy) //nolint:gosec // a path this test chose
				s.Require().NoError(err)
				s.Require().Equal("somebody's rig", string(got))
			},
		},
		{
			name: "a slot as a rig",
			args: tools.Export{Slot: "01A", Out: "a.yaml"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Export(gomock.Any(), slot.Address{}, "a.yaml", sdk.FormatRig, sdk.KeepExisting).
					Return(sdk.Written{Path: "a.yaml"}, nil)
			},
			want: "wrote a.yaml from 01A",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Written
				structured(s.T(), res, &got)
				s.Equal("a.yaml", got.Path)
			},
		},
		{
			name: "the device's own file",
			args: tools.Export{Slot: "01A", Out: "a.hlx", As: "hlx"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Export(gomock.Any(), slot.Address{}, "a.hlx", sdk.FormatPreset, sdk.KeepExisting).
					Return(sdk.Written{Path: "a.hlx"}, nil)
			},
			want: "wrote a.hlx from 01A",
		},
		{
			// Refused before the pedal is claimed, rather than written as a
			// rig nobody asked for. No Session call is expected, so reaching
			// the device fails the row.
			name: "a format that does not exist",
			args: tools.Export{Slot: "01A", Out: "a.yaml", As: "yaml"},
			want: "unknown format",
			err:  true,
		},
		{
			// "nope" fails to parse: its trailing letter, E, is past the
			// last bank letter C, so slot.Value.Set refuses it before the
			// pedal is ever opened.
			name: "a label the pedal does not have",
			args: tools.Export{Slot: "nope", Out: "a.yaml"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Export{Slot: "01A", Out: "a.hlx", As: "hlx"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestPresetSelect covers loading a slot.
func (s *DevicePublicTestSuite) TestPresetSelect() {
	s.run("device_select", []deviceRow{
		{
			name: "a slot by its label",
			args: tools.Slot{Slot: "07A"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Select(gomock.Any(), slot.Address{Slot: 18}).
					Return(sdk.Change{Action: sdk.Selected, To: sdk.At{Slot: 18, Name: "Chunky Monkey"}}, nil)
			},
			want: "loaded 07A",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Change
				structured(s.T(), res, &got)
				s.Equal(sdk.Selected, got.Action)
			},
		},
		{
			// "0A" fails to parse: there is no bank zero, banks count from
			// one, so slot.Value.Set refuses it before the pedal is ever
			// opened. ("43A" was tried first, but a bank number carries no
			// upper bound in slot.parse, so it decodes to a slot index
			// rather than failing.)
			name: "a label the pedal does not have",
			args: tools.Slot{Slot: "0A"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Slot{Slot: "07A"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestDeviceCurrent covers reading back what the pedal is playing.
//
// The only way to see a live edit. A turn stores nothing, so no slot read shows
// what it did, and this is what a measuring run uses to state what it measured
// rather than what it asked for.
func (s *DevicePublicTestSuite) TestDeviceCurrent() {
	s.run("device_current", []deviceRow{
		{
			name: "a chain in front of somebody",
			args: tools.None{},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Current(gomock.Any(), sdk.FormatRig).
					Return(sdk.Reading{Name: "Chinky Monkey"}, nil)
			},
			want: "the pedal is playing Chinky Monkey",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Shown
				structured(s.T(), res, &got)
				s.Equal("Chinky Monkey", got.Name)
			},
		},
		{
			name: "a pedal that will not answer",
			args: tools.None{},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
					Return(sdk.Reading{}, errors.New("the device stopped answering"))
			},
			want: "the device stopped answering",
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.None{},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestDevicePlay covers putting a preset in front of somebody.
//
// It writes no flash, which is the whole reason it exists: auditioning through
// a slot is a flash write per attempt, and a burst of those took a setlist past
// what a power cycle could clear.
func (s *DevicePublicTestSuite) TestDevicePlay() {
	s.run("device_play", []deviceRow{
		{
			name: "a preset played and nothing stored",
			args: tools.Play{Preset: "mine.hlx"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Play(gomock.Any(), "mine.hlx").Return(nil)
			},
			want:  "the pedal is playing mine.hlx, and holds what it held",
			moved: true,
		},
		{
			name: "a preset the device refuses",
			args: tools.Play{Preset: "broken.hlx"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Play(gomock.Any(), "broken.hlx").
					Return(errors.New("not a preset document"))
			},
			want: "not a preset document",
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Play{Preset: "mine.hlx"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestDeviceTurn covers moving one control on what is playing.
//
// Three kinds of value and exactly one of them per call, because a device does
// not coerce: a value's tag is its type on the wire, and a switch handed 1.0 is
// refused with the same error a block that is not there gives.
//
// Every row states the wire address it expects. A block a preset records at
// position 4 answers to 5, and getting that wrong moves a different block while
// every number stays plausible.
func (s *DevicePublicTestSuite) TestDeviceTurn() {
	dial := float32(0.75)
	mic := 3
	on := true

	s.run("device_turn", []deviceRow{
		{
			name: "a dial, addressed one past the position a preset records",
			args: tools.Turn{Block: 4, Param: 1, Value: &dial},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Turn(gomock.Any(), sdk.Address{Block: 5, Param: 1, Direct: true}, dial).
					Return(nil)
			},
			want:  "block 4 parameter 1 moved, and nothing was written",
			moved: true,
		},
		{
			name: "one of a list, such as a microphone",
			args: tools.Turn{Block: 0, Param: 2, Choice: &mic},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Choose(gomock.Any(), sdk.Address{Block: 1, Param: 2, Direct: true}, mic).
					Return(nil)
			},
			want:  "block 0 parameter 2 moved",
			moved: true,
		},
		{
			name: "a switch on a cabinet fused into an amplifier",
			args: tools.Turn{Block: 2, Param: 7, Switch: &on, Model: 1},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Switch(gomock.Any(),
						sdk.Address{Block: 3, Param: 7, Model: 1, Direct: true}, on).
					Return(nil)
			},
			want:  "block 2 parameter 7 moved",
			moved: true,
		},
		{
			// The value some blocks carry past their own list.
			name: "direct addressing turned off",
			args: tools.Turn{Block: 1, Param: 0, Value: &dial, Direct: new(false)},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().
					Turn(gomock.Any(), sdk.Address{Block: 2, Param: 0, Direct: false}, dial).
					Return(nil)
			},
			want:  "block 1 parameter 0 moved",
			moved: true,
		},
		{
			name: "no value at all",
			args: tools.Turn{Block: 4, Param: 1},
			want: tools.ErrOneValue.Error(),
			err:  true,
		},
		{
			name: "two kinds of value at once",
			args: tools.Turn{Block: 4, Param: 1, Value: &dial, Switch: &on},
			want: tools.ErrOneValue.Error(),
			err:  true,
		},
		{
			name: "a parameter the block does not have",
			args: tools.Turn{Block: 4, Param: 99, Value: &dial},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("device refused the request"))
			},
			want: "device refused the request",
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Turn{Block: 4, Param: 1, Value: &dial},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

func TestDevicePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DevicePublicTestSuite))
}
