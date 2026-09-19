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

// Opcodes that read and replace what a device is playing.
const (
	// opReadCurrent reads the preset a device is playing.
	opReadCurrent = 22
	// opWriteCurrent replaces it, without writing anything to a slot.
	opWriteCurrent = 21
)

// ReadCurrent fetches the preset the device has loaded.
//
// The edit buffer rather than a slot: what somebody is hearing, including
// whatever they have changed since it was loaded. A stored slot is what
// ReadPreset answers with, and the two differ — a loaded document carries the
// firmware build string a stored one does not.
func (s *session) ReadCurrent(
	ctx context.Context,
) ([]byte, error) {
	resp, err := s.Call(ctx, channelData, opReadCurrent, nil)
	if err != nil {
		return nil, err
	}

	return document(resp.Result)
}

// WriteCurrent replaces the preset the device is playing.
//
// Nothing is stored. The document goes into the edit buffer and every slot is
// left alone, which is what makes this the operation for auditioning: a
// caller can put six hundred different chains in front of a device in an
// afternoon without a single write to flash.
//
// That distinction is not a nicety. A burst of slot writes corrupted a
// setlist past what a power cycle could clear, and a device tolerates about a
// dozen racing commits before it stops accepting writes at all — see
// [Rules that keep a device alive](../../../../docs/protocol.md#rules-that-keep-a-device-alive).
// Measuring every block the device has, one at a time, is exactly the shape
// that rule forbids doing through slots.
//
// Sent down the same chunked path a slot write uses, because it carries a
// whole preset and one frame will not hold it. The pause afterwards is not
// for the flash, which this never touches, but for the device: it answers
// while it is still settling the new chain, and a measurement taken inside
// that window measures the changeover.
func (s *session) WriteCurrent(
	ctx context.Context,
	document []byte,
) error {
	return s.write(ctx, opWriteCurrent, []wire.Arg{
		wire.Blob(argDocument, document),
	})
}
