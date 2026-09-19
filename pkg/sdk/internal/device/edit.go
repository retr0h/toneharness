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
package device

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// opSetParam changes one parameter in the preset a device is playing.
//
// The difference from writing a preset is the whole reason this exists.
// `presets import` puts a document in a slot, and what the device is playing
// does not change: a slot given a copy of another preset reads back as the
// copy and goes on sounding like what it held before. This reaches the
// running preset instead, and is heard at once.
//
// It is what HX Edit sends when somebody drags a knob, one message per
// intermediate value, dozens in a gesture. So a sweep is a loop rather than a
// sequence of writes, and nothing touches the device's storage at all.
const opSetParam = 30

// Keys one of these calls is made of. A device owns these names.
const (
	// editBlock is the block's position in the chain. The same number the
	// rest of this package calls a block's number, and renumbering a chain
	// moves it.
	editBlock = 98
	// editDirect says the parameter is addressed the ordinary way. False
	// reaches the one value some blocks carry past their parameter list,
	// where the index is then zero.
	editDirect = 29
	// editModel picks between the block's own model and a cabinet fused into
	// an amplifier's slot, which carries its own parameters under 1.
	editModel = 26
	// editParam is the parameter's position in the model's parameter list.
	// Not its name: position is the only thing that identifies it on the
	// wire, which is what catalog.Symbol.Params already records.
	editParam = 28
	// editValue is the value, in that parameter's own units.
	editValue = 119
)

// SetParam moves one control on the preset the device is playing.
//
// The value is in the parameter's own units rather than anything normalised.
// Most run zero to one because that is genuinely their range, and a cabinet's
// microphone distance runs one to twelve inches.
//
// A device refuses rather than complains. The reply carries a status and the
// call fails on it, because a refusal that returned nil would leave a sweep
// recording the same sound at every step and calling it a measurement.
func (s *session) SetParam(
	ctx context.Context,
	block int,
	param int,
	value float32,
) error {
	_, err := s.Call(ctx, channelData, opSetParam, []wire.Arg{
		wire.Number(editBlock, uint64(block)), //nolint:gosec // a chain index
		wire.Flag(editDirect, true),
		wire.Number(editModel, 0),
		wire.Number(editParam, uint64(param)), //nolint:gosec // a list index
		wire.Real(editValue, value),
	})

	return err
}
