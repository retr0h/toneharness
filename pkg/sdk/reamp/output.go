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

import (
	"errors"
	"fmt"
	"strings"
)

// This file makes which device the computer plays through part of the measuring
// rig, rather than something somebody set last week.
//
// The same argument as the output level beside it, one step further out. The
// level is pinned so that every amplifier in a campaign is driven equally hard,
// and the level belongs to a device: pin it while the platform's default is some
// other device and the number is about a device that is not in the signal path.
//
// Measured, which is why this exists. A Mac with its default output on a pair of
// Bluetooth headphones pinned those to 38 while the headphone jack feeding the
// pedal sat wherever it was left. The loop read 66dB of loss, the run called it a
// dead return, and the only thing wrong was which device the number belonged to.

// ErrOutput is a platform that would not say or would not change which device it
// plays through.
var ErrOutput = errors.New("the computer's output device")

// errNoOutput is a platform with no default output to name, and errRefused one
// that would not take the change.
var (
	errNoOutput = errors.New("nothing is set as the default")
	errRefused  = errors.New("the platform refused the change")
)

// OutputError is the platform refusing to read or set the output device.
type OutputError struct {
	// Doing is which half failed, so the message says whether anything moved.
	Doing string
	Said  error
}

func (e *OutputError) Error() string {
	return fmt.Sprintf("%s: %s it: %v", ErrOutput, e.Doing, e.Said)
}

func (e *OutputError) Unwrap() error { return ErrOutput }

// NoOutputError is a device nothing attached answers to.
type NoOutputError struct {
	Want string
	Had  []string
}

func (e *NoOutputError) Error() string {
	if len(e.Had) == 0 {
		return fmt.Sprintf("%s: no device matching %q, and nothing can play",
			ErrOutput, e.Want)
	}

	return fmt.Sprintf("%s: no device matching %q. Attached: %s",
		ErrOutput, e.Want, strings.Join(e.Had, ", "))
}

func (e *NoOutputError) Unwrap() error { return ErrOutput }

// saysOutput and setsOutput are how the platform is asked, so a test stands its
// own functions here.
//
// Variables because they are the only things in this file that leave the
// process, and the branches either side of them — a platform that will not say,
// one that will not change, one that takes the change and does something else —
// are branches nothing could reach otherwise. The same shape `asks` has for the
// level.
var (
	saysOutput = output
	setsOutput = setOutput
)

// Output answers which device the computer is currently set to play through.
func Output() (string, error) {
	return saysOutput()
}

// PlaysThrough makes the named device the one the computer plays through, and
// says what it was before.
//
// Answers the name in force afterwards whether it moved or not, the way Held
// does for the level, so a caller records what the readings were actually taken
// through rather than what it asked for.
//
// Reaches outside this tool, so every caller says that it did. The alternative
// is what happened without it: a rig that looked wired correctly, a level pinned
// on the wrong device, and an hour spent on the cabling.
func PlaysThrough(
	want string,
) (was string, moved bool, err error) {
	was, err = saysOutput()
	if err != nil {
		return "", false, err
	}

	if strings.Contains(strings.ToLower(was), strings.ToLower(want)) {
		return was, false, nil
	}

	if err := setsOutput(want); err != nil {
		return was, false, err
	}

	// Read back rather than assumed, for the same reason the level is: a
	// platform that takes the instruction and does something else leaves the rig
	// somewhere nobody asked for, and this is the only place that shows.
	now, err := saysOutput()
	if err != nil {
		return was, true, err
	}

	return now, true, nil
}
