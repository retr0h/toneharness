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
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
	"github.com/retr0h/toneharness/pkg/sdk/internal/fileslots"
)

// Play puts a preset in front of the device without storing it anywhere.
//
// The difference from Import is the whole reason it exists. Import writes a
// slot, which is flash, and flash is the one resource here that wears out and
// corrupts: a burst of writes took a setlist past what a power cycle could
// clear, and a device stops accepting them after about a dozen racing
// commits. See
// [Rules that keep a device alive](../wire/README.md#rules-that-keep-a-device-alive).
//
// That rule is what makes auditioning through slots impossible. Measuring
// each of a device's blocks in turn means putting hundreds of different
// chains in front of it, and doing that through Import is hundreds of flash
// writes for readings nobody wanted to keep. This does the same thing and
// stores nothing, so the cost of trying a chain is the time it takes to hear
// it.
//
// Nothing is kept and nothing can be put back, because nothing was replaced.
// Whatever slot the device was playing from still holds what it held; only
// what is coming out of it changes, until the next preset is selected.
func (f *Flows) Play(
	ctx context.Context,
	s device.Editor,
	file string,
) error {
	player, ok := s.(device.Loaded)
	if !ok {
		return fmt.Errorf("this session cannot replace what is playing")
	}

	doc, err := fileslots.ReadPreset(ctx, file)
	if err != nil {
		return err
	}

	// The same bytes an Import would have written. A preset is seeked through
	// by a table of byte offsets, so one built any other way is accepted and
	// then rendered as an empty chain.
	body, err := f.documentFor(ctx, s.Model().Name, doc)
	if err != nil {
		return err
	}

	if err := player.WriteCurrent(ctx, body); err != nil {
		return fmt.Errorf("replacing what is playing: %w", err)
	}

	return nil
}
