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
	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// ErrNothingLoaded is returned when the device answers with no preset.
var ErrNothingLoaded = fmt.Errorf("the device is playing no preset")

// Loaded reads the preset the device is playing, as the rig it describes.
//
// The edit buffer rather than a slot, which is the whole reason it exists: a
// control moved live shows here and not in the slot it came from. Reading the
// slot back after moving one answers with the stored document, unchanged,
// which reads as though nothing happened.
//
// That makes this the only way to see what a device is actually doing, and
// the way a build checks its own work: move a control, read what the device
// now holds, and compare it to what was asked for.
func (f *Flows) Loaded(
	ctx context.Context,
	s device.Editor,
	as result.Format,
) (result.Reading, error) {
	r, ok := s.(device.Loaded)
	if !ok {
		return result.Reading{}, fmt.Errorf("this session cannot read what is loaded")
	}

	body, err := r.ReadCurrent(ctx)
	if err != nil {
		return result.Reading{}, fmt.Errorf("reading what is loaded: %w", err)
	}

	if body == nil {
		return result.Reading{}, ErrNothingLoaded
	}

	// No slot, because the slot is not knowable here: what the device hands
	// back is a document rather than a position, and it may have been edited
	// away from whatever it was loaded from, so naming a slot would name one
	// whose contents this is no longer.
	//
	// Passed as nil rather than as zero, which is what it used to be. Zero is
	// slot 01A, so this read reported `slot 01A` and an id of `slot-01a`
	// whatever was loaded, and the command that exists to tell a slot write
	// from a slot load named the wrong slot every time.
	return f.deviceReading(ctx, body, nil, "", as)
}
