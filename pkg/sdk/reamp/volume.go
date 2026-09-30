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

package reamp

import "errors"

// This file makes the computer's own output level part of the measuring rig
// rather than something somebody remembers.
//
// It matters because of what it is not. A louder reference is not a louder
// reading of the same tone: an amplifier's distortion depends on how hard it is
// driven, so moving this knob changes what every amplifier in a campaign
// measures as. The level is a tone control and it happens to live on a laptop.
//
// Nothing recorded it before, and the rig that plays the reference out of the
// computer's own output puts it in the signal path. So it is read, set to a
// stated figure, and written into the library beside the headroom.
//
// Setting it reaches outside this tool, which is why it says so when it does.
// The alternative was asking a person to hold a number steady across campaigns
// weeks apart, and that is not a thing anybody does.

// Volume is the computer's own output level, 0 to 100.
//
// The platform's own scale rather than decibels, because that is the number a
// person sees on the slider and can put back. It is not linear in decibels and
// is not comparable between machines, which is what the baseline reading is for.
func Volume() (int, error) { return volume() }

// SetVolume puts the computer's own output level at a stated figure.
func SetVolume(
	to int,
) error {
	if to < 0 || to > 100 {
		return &VolumeError{Doing: "setting", Said: errors.New("0 to 100")}
	}

	return setVolume(to)
}

// Held puts the output level where a measurement wants it and says what it did.
//
// Answers the level in force afterwards, whether it moved or not, so a caller
// can record what the readings were actually taken at rather than what it asked
// for. Those differ when the platform cannot be told, and a library recording
// the request instead of the fact is a library that lies quietly.
func Held(
	want int,
) (int, bool, error) {
	was, err := Volume()
	if err != nil {
		return 0, false, err
	}

	if was == want {
		return was, false, nil
	}

	if err := SetVolume(want); err != nil {
		return was, false, err
	}

	// Read back rather than assumed. A platform that accepts the instruction
	// and rounds it, which macOS does on some hardware, leaves the rig at a
	// level nobody asked for and this is the only place that shows.
	now, err := Volume()
	if err != nil {
		return 0, true, err
	}

	return now, true, nil
}
