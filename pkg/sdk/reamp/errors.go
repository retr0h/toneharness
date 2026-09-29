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

// ErrRate is returned when the audio backend is not running the loop at the
// rate every committed figure was taken at.
var ErrRate = errors.New("the audio device is not running at the measuring rate")

// RateError says what the backend negotiated, and against what.
//
// Refused rather than resampled. Every figure in resources/sweeps/ and the
// reference recording are 48kHz, and a reading taken at another rate is not
// comparable to them however good the resampler is: the converter's own
// artefacts land in a number whose entire purpose is being compared to the
// committed ones.
//
// Loud rather than quiet, because miniaudio will resample instead of refusing
// and the reading that comes back is a plausible figure rather than an error.
// That is the shape of every measuring bug this loop has had: the first take
// after a bench opens, the audio device that was a pair of headphones, the
// noise floor one odd reading set. None of them failed; they all answered.
type RateError struct {
	// Name is the device the loop opened.
	Name string
	// Want is the rate the reference and the committed figures are at.
	Want uint32
	// Capture and Playback are what the backend negotiated for each
	// direction, which is where a silent resample shows.
	Capture  uint32
	Playback uint32
}

// Error implements the error interface.
func (e *RateError) Error() string {
	return fmt.Sprintf(
		"%s: %s negotiated %dHz in and %dHz out, and every committed figure is "+
			"at %dHz. Set the device to %dHz, or name one that runs at it with "+
			"--hardware",
		ErrRate, e.Name, e.Capture, e.Playback, e.Want, e.Want)
}

// Unwrap returns ErrRate so callers can match with errors.Is.
func (*RateError) Unwrap() error { return ErrRate }

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
//
// Two sentences rather than one when nothing was named, because "no device
// matching \"\"" reads as a bug in the tool rather than as a question for the
// person running it.
func (e *NoDeviceError) Error() string {
	if e.Want == "" {
		return fmt.Sprintf("no such audio device: no Helix %s attached, and "+
			"nothing named with --hardware. Attached: %s",
			e.Direction, strings.Join(e.Had, ", "))
	}

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
