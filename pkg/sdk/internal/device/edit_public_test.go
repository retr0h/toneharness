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
package device_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

// EditPublicTestSuite covers moving one control on the running preset.
//
// Opcode 30, which is what HX Edit sends when somebody drags a knob. The
// distinction from writing a preset is the whole reason it exists: a slot
// given a new document goes on sounding like what it held before, so this is
// the only way a sweep is heard.
type EditPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *EditPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// moved is the device's answer to a live edit.
//
// On success it echoes the whole target map back, which makes the reply
// self-verifying. A refusal is status 255 carrying a code, with nothing
// applied and no error frame after it.
func (s *EditPublicTestSuite) moved(
	txn uint64,
	refuse bool,
) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))

	if refuse {
		s.Require().NoError(enc.EncodeInt(255))
		s.Require().NoError(enc.EncodeInt(104))
		s.Require().NoError(enc.EncodeMapLen(1))
		s.Require().NoError(enc.EncodeInt(111))
		s.Require().NoError(enc.EncodeInt(-3))

		return device.Reply(device.DataChannel, buf.Bytes())
	}

	s.Require().NoError(enc.EncodeInt(0))
	s.Require().NoError(enc.EncodeInt(104))
	s.Require().NoError(enc.EncodeMapLen(0))

	return device.Reply(device.DataChannel, buf.Bytes())
}

// TestSetParam covers a control moving, and a device declining to move it.
func (s *EditPublicTestSuite) TestSetParam() {
	tests := []struct {
		name string
		at   device.Address
		// what the device answers with.
		refuse bool
		says   string
	}{
		{
			name: "an ordinary parameter",
			at:   device.Address{Block: 2, Param: 5, Direct: true},
		},
		{
			// The cabinet fused into an amplifier's slot carries its own
			// parameters, under a different model number.
			name: "a paired cabinet's parameter",
			at:   device.Address{Block: 2, Param: 1, Model: 1, Direct: true},
		},
		{
			// False reaches the value some blocks carry past their list,
			// where the index is then zero.
			name: "the value past a block's list",
			at:   device.Address{Block: 2, Param: 0, Direct: false},
		},
		{
			// A device declines rather than complains: nothing is applied,
			// no error frame follows, and the next read is byte-identical.
			// Returning nil here would leave a sweep recording the same
			// sound at every step and calling it a measurement.
			name:   "a parameter the device will not address that way",
			at:     device.Address{Block: 2, Param: 3, Direct: true},
			refuse: true,
			says:   "error -3",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := context.Background()

			d := answers(s.ctrl, s.moved(device.FirstTxn, tt.refuse))

			session := device.NewTestSessionWith(s.T(), d.out, d.in,
				device.ShortBudgets())
			session.OpenChannels()

			err := session.SetParam(ctx, tt.at, 0.25)

			if tt.says == "" {
				s.Require().NoError(err)
				s.Require().Zero(d.pending(), "every answer was read")

				return
			}

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

// TestSetChoiceAndSetSwitch covers the two other kinds of value.
//
// They exist because a device does not coerce. The value's tag is its type on
// the wire, so a cabinet's microphone refuses a float and an amplifier's
// Bright refuses one too, both with the same error a block that is not there
// gives. Sending the wrong kind reads as the block being wrong.
func (s *EditPublicTestSuite) TestSetChoiceAndSetSwitch() {
	at := device.Address{Block: 2, Param: 5, Direct: true}

	tests := []struct {
		name   string
		choice bool
		refuse bool
		says   string
	}{
		{name: "a microphone, which is one of a list", choice: true},
		{name: "a switch, which is on or off"},
		{
			name:   "a choice the device will not take",
			choice: true,
			refuse: true,
			says:   "error -3",
		},
		{
			name:   "a switch the device will not take",
			refuse: true,
			says:   "error -3",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := answers(s.ctrl, s.moved(device.FirstTxn, tt.refuse))

			session := device.NewTestSessionWith(s.T(), d.out, d.in,
				device.ShortBudgets())
			session.OpenChannels()

			ctx := context.Background()

			// One message, because the double answers once. Sending both
			// would spend the answer on the first and time the second out.
			var err error

			if tt.choice {
				err = session.SetChoice(ctx, at, 3)
			} else {
				err = session.SetSwitch(ctx, at, true)
			}

			if tt.says == "" {
				s.Require().NoError(err)
				s.Require().Zero(d.pending(), "every answer was read")

				return
			}

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

func TestEditPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EditPublicTestSuite))
}
