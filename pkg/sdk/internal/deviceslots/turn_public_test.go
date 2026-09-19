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
package deviceslots_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// TurnPublicTestSuite covers moving a control on what a device is playing.
type TurnPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

// turnable is a session that can move a control and read what is loaded.
type turnable struct {
	*mocks.MockEditor
	*mocks.MockTurner
	*mocks.MockLoaded
}

func (*turnable) Close() error { return nil }

func (s *TurnPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *TurnPublicTestSuite) dev() *turnable {
	return &turnable{
		MockEditor: mocks.NewMockEditor(s.ctrl),
		MockTurner: mocks.NewMockTurner(s.ctrl),
		MockLoaded: mocks.NewMockLoaded(s.ctrl),
	}
}

// TestTurnMovesAControl covers the ordinary case and the device declining.
func (s *TurnPublicTestSuite) TestTurnMovesAControl() {
	ctx := context.Background()
	at := device.Address{Block: 2, Param: 5, Model: 0, Direct: true}

	tests := []struct {
		name string
		give error
		want string
	}{
		{name: "the device takes it"},
		{
			// A device declines rather than complains, so a nil here would
			// leave a sweep recording the same sound at every step and
			// calling it a measurement.
			name: "the device declines",
			give: errors.New("opcode 30, error -3"),
			want: "setting parameter 5 on block 2",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := s.dev()
			d.MockTurner.EXPECT().
				SetParam(ctx, at, float32(0.25)).
				Return(tt.give)

			err := (&deviceslots.Flows{}).Turn(ctx, d, at, 0.25)

			if tt.want == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorContains(err, tt.want)
		})
	}
}

// TestTurnNeedsASessionThatCanMoveOne covers a session without the capability.
func (s *TurnPublicTestSuite) TestTurnNeedsASessionThatCanMoveOne() {
	err := (&deviceslots.Flows{}).Turn(
		context.Background(),
		mocks.NewMockEditor(s.ctrl),
		device.Address{},
		0.5,
	)

	s.Require().ErrorContains(err, "cannot move a control")
}

// TestChooseAndSwitch covers the two kinds of value that are not a dial.
//
// They exist because a device does not coerce: a cabinet's microphone is an
// index and an amplifier's Bright is a switch, and each refuses a float with
// the same error a block that is not there gives.
func (s *TurnPublicTestSuite) TestChooseAndSwitch() {
	ctx := context.Background()
	at := device.Address{Block: 2, Param: 5, Direct: true}

	s.Run("a microphone", func() {
		d := s.dev()
		d.MockTurner.EXPECT().SetChoice(ctx, at, 3).Return(nil)

		s.Require().NoError((&deviceslots.Flows{}).Choose(ctx, d, at, 3))
	})

	s.Run("a microphone the device refuses", func() {
		d := s.dev()
		d.MockTurner.EXPECT().SetChoice(ctx, at, 99).
			Return(errors.New("error -3"))

		s.Require().ErrorContains(
			(&deviceslots.Flows{}).Choose(ctx, d, at, 99), "to choice 99")
	})

	s.Run("a switch", func() {
		d := s.dev()
		d.MockTurner.EXPECT().SetSwitch(ctx, at, true).Return(nil)

		s.Require().NoError((&deviceslots.Flows{}).Switch(ctx, d, at, true))
	})

	s.Run("a switch the device refuses", func() {
		d := s.dev()
		d.MockTurner.EXPECT().SetSwitch(ctx, at, false).
			Return(errors.New("error -3"))

		s.Require().ErrorContains(
			(&deviceslots.Flows{}).Switch(ctx, d, at, false), "to false")
	})

	s.Run("a session that cannot move a control", func() {
		plain := mocks.NewMockEditor(s.ctrl)

		s.Require().ErrorContains(
			(&deviceslots.Flows{}).Choose(ctx, plain, at, 1),
			"cannot move a control")
		s.Require().ErrorContains(
			(&deviceslots.Flows{}).Switch(ctx, plain, at, true),
			"cannot move a control")
	})
}

// TestLoadedReadsTheEditBuffer covers reading what a device is playing.
func (s *TurnPublicTestSuite) TestLoadedReadsTheEditBuffer() {
	ctx := context.Background()

	tests := []struct {
		name string
		give []byte
		err  error
		want string
	}{
		{
			// A device answering with no document is playing nothing, which
			// is not the same as a read that failed.
			name: "nothing is loaded",
			want: "playing no preset",
		},
		{
			name: "the read fails",
			err:  errors.New("usb: gone"),
			want: "reading what is loaded",
		},
		{
			name: "the answer is not a preset",
			give: []byte("not a document"),
			want: "reading slot",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := s.dev()
			d.MockLoaded.EXPECT().ReadCurrent(ctx).Return(tt.give, tt.err)

			_, err := (&deviceslots.Flows{}).Loaded(ctx, d, result.FormatPreset)

			s.Require().ErrorContains(err, tt.want)
		})
	}
}

// TestLoadedNeedsASessionThatCanRead covers a session without the capability.
func (s *TurnPublicTestSuite) TestLoadedNeedsASessionThatCanRead() {
	_, err := (&deviceslots.Flows{}).Loaded(
		context.Background(),
		mocks.NewMockEditor(s.ctrl),
		result.FormatPreset,
	)

	s.Require().ErrorContains(err, "cannot read what is loaded")
}

func TestTurnPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TurnPublicTestSuite))
}
