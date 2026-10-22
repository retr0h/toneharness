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

package deviceslots

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/sdk/internal/backup"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device/mocks"
	slotpkg "github.com/retr0h/toneharness/pkg/sdk/slot"
)

// KeepTestSuite covers reading a slot so what it held can be kept before it
// is replaced.
//
// What is kept, and how, is the backup package's policy and its suite. These
// cover the read in front of it and the hand-off.
type KeepTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *KeepTestSuite) SetupTest() { s.ctrl = gomock.NewController(s.T()) }

func (s *KeepTestSuite) TearDownTest() { s.ctrl.Finish() }

// answer returns one slot as an HX Stomp sent it.
func (s *KeepTestSuite) answer() []byte {
	raw, err := os.ReadFile(filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

// TestHolds covers reading a slot that may hold nothing.
func (s *KeepTestSuite) TestHolds() {
	tests := []struct {
		name string
		at   slotpkg.Address
		// held is what the slot is called, which decides whether it is read
		// at all: a slot still carrying the name a device ships with holds
		// nothing worth keeping, and reading one costs a call that has
		// stalled the endpoint.
		held    string
		body    []byte
		err     error
		errText string
	}{
		{name: "a slot with a preset in it", body: []byte{1, 2, 3}},
		{name: "a slot with nothing in it"},
		{
			// Both halves of the address reach the device.
			name: "a slot in another setlist",
			at:   slotpkg.Address{Setlist: 1, Slot: 5},
			body: []byte{4},
		},
		{
			name:    "a device that will not say",
			err:     errors.New("boom"),
			errText: "reading slot 01A before replacing it",
		},
		{
			// Never read, so never kept: there is nothing in an untouched
			// slot to lose.
			name: "a slot nobody has written",
			held: backup.Untouched,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dev := mocks.NewMockEditor(s.ctrl)
			if tt.held != backup.Untouched {
				dev.EXPECT().
					ReadPreset(gomock.Any(), tt.at.Setlist, tt.at.Slot).
					Return(tt.body, tt.err)
			}

			got, err := holds(context.Background(), dev, tt.at, tt.held)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.body, got)
		})
	}
}

// TestReplacing covers reading a slot and keeping what it held.
func (s *KeepTestSuite) TestReplacing() {
	tests := []struct {
		name string
		body []byte
		err  error
		dir  string
		// no Backups on the flows, so the default keeper is the one used.
		defaults bool
		kept     int
		// nothing is whether the slot is reported as having held no preset,
		// which is what tells "there was nothing to keep" from "nothing was
		// kept".
		nothing bool
		errText string
	}{
		{name: "a slot with a preset in it", body: s.answer(), kept: 1},
		{name: "a slot with nothing in it", nothing: true},
		{
			// Nobody configured a keeper, so the flows keep one of their own
			// in the state directory.
			name:     "flows nobody gave a keeper",
			body:     s.answer(),
			defaults: true,
			kept:     1,
		},
		{
			name:    "a device that will not say what is there",
			err:     errors.New("boom"),
			errText: "before replacing it",
		},
		{
			name:    "nowhere to keep it",
			body:    s.answer(),
			dir:     "\x00",
			errText: "making room for a backup",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := tt.dir
			if dir == "" {
				dir = s.T().TempDir()
			}

			dev := mocks.NewMockEditor(s.ctrl)
			dev.EXPECT().ReadPreset(gomock.Any(), 0, 3).Return(tt.body, tt.err)

			f := &Flows{}

			if tt.defaults {
				s.T().Setenv("XDG_STATE_HOME", dir)
			} else {
				f.Backups = backup.New(dir, NewDecoder(f))
			}

			got, nothing, err := f.replacing(
				context.Background(), dev, slotpkg.Address{Slot: 3}, "Black Rusty")

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got, tt.kept)
			s.Require().Equal(tt.nothing, nothing)

			for _, path := range got {
				s.Require().FileExists(path)
				s.Require().Contains(path, dir)
			}
		})
	}
}

// TestUnanswered covers saying what a slot read's silence means.
//
// Task 260. `ReadPreset`'s contract is that no bytes and no error means a slot
// holding nothing, and an HX Stomp asked for a slot it has never held a preset
// in does not take that route: it says nothing, the read times out after six
// seconds, and **the next call on the interface stalls** until the pedal is
// powered off and on. A bare `no reply to opcode 4 within 6s` leaves somebody
// with a dead interface and no idea why.
//
// One method and one table, so a case is a row rather than a file.
func (s *KeepTestSuite) TestUnanswered() {
	tests := []struct {
		name string
		err  error
		// says is what the answer has to carry, and bare whether nothing was
		// added to it.
		says []string
		bare bool
	}{
		{
			// The failure this exists for.
			name: "silence says what it means and what it costs",
			err:  fmt.Errorf("%w to opcode 4 within 6s", device.ErrNoReply),
			says: []string{
				// Added to rather than replaced: the opcode and the budget
				// are what somebody greps for.
				"opcode 4 within 6s",
				"never held a preset",
				"powered off and on",
				"slots list",
			},
		},
		{
			// A refusal is the device answering, so there is nothing to
			// explain and the device's own words are better than any gloss.
			name: "a refusal is left as it is",
			err:  errors.New("the device refused opcode 4"),
			bare: true,
		},
		{
			// A bus that ended says so already, and a session that returned
			// it is finished rather than stalled.
			name: "a bus that ended is left as it is",
			err:  device.ErrBus,
			bare: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := unanswered(tt.err)

			s.Require().ErrorIs(got, tt.err, "the failure has to travel with it")

			if tt.bare {
				s.Require().Equal(tt.err.Error(), got.Error())

				return
			}

			for _, want := range tt.says {
				s.Require().ErrorContains(got, want)
			}
		})
	}
}

func TestKeepTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KeepTestSuite))
}
