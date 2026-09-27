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
	"time"
)

// ErrNoDevice is returned when the named hardware is not attached.
var ErrNoDevice = errors.New("no such audio device")

// NoDeviceError says what was asked for and what was actually attached.
//
// The list travels with it because the answer to "that device is not here" is
// almost always in the names that are, spelled slightly differently.
type NoDeviceError struct {
	// Want is the name that was asked for.
	Want string
	// Direction is whether this was an input or an output.
	Direction string
	// Had is every device of that direction that was attached.
	Had []string
}

// Error implements the error interface.
func (e *NoDeviceError) Error() string {
	return fmt.Sprintf("no such audio device: no %s device matching %q. "+
		"Attached: %s", e.Direction, e.Want, strings.Join(e.Had, ", "))
}

// Unwrap returns ErrNoDevice so callers can match with errors.Is.
func (*NoDeviceError) Unwrap() error { return ErrNoDevice }

// ErrUnclaimed is returned when a device neither opened nor refused.
var ErrUnclaimed = errors.New("the audio device never handed itself over")

// UnclaimedError says which device went quiet while being opened.
//
// Distinct from a device that refuses, which answers immediately and says why.
// This is the one that answers nothing: on macOS the first capture by a program
// with no Microphone permission blocks inside CoreAudio while the system waits
// for somebody to answer a dialog, and a terminal running unattended has nobody
// to answer it. Before this had a deadline a campaign sat there all night,
// having measured nothing, looking exactly like one still working.
type UnclaimedError struct {
	// Name is the device that went quiet.
	Name string
	// After is how long it was given.
	After time.Duration
}

// Error implements the error interface.
func (e *UnclaimedError) Error() string {
	return fmt.Sprintf(
		"the audio device never handed itself over: %s neither opened nor "+
			"refused within %s. Grant Microphone access to the terminal running "+
			"this in System Settings, Privacy & Security, then answer any dialog "+
			"macOS is holding: the first capture blocks until somebody does",
		e.Name, e.After)
}

// Unwrap returns ErrUnclaimed so callers can match with errors.Is.
func (*UnclaimedError) Unwrap() error { return ErrUnclaimed }
