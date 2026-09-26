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

package translate

import (
	"errors"
	"fmt"
)

// ErrInsisted is returned when a request named gear the device does not have
// and said not to substitute.
var ErrInsisted = errors.New("the device has no such gear")

// ErrWrongDevice is returned when the measurements describe one device and
// the setup names another.
//
// Not a warning. Every block was measured on one piece of hardware, and a
// ranking built from those readings is a ranking of that device's blocks. Run
// against another it answers confidently with models the device in hand may
// not even have.
var ErrWrongDevice = errors.New("the measurements were taken on a different device")

// ErrNothingToBuildFrom is returned when a request names no gear and gives
// nothing that can be measured.
//
// A sentinel rather than a sentence, because the useful answer is what to ask
// for next and only the caller knows whether it is talking to a person or to
// an agent. "Make me a punk bass tone" lands here: it is a real request and
// there is nothing in it to resolve, so the next move is to ask which records
// rather than to guess at an amplifier.
var ErrNothingToBuildFrom = errors.New(
	"nothing in the request names gear, and nothing in it can be measured " +
		"against, so there is no chain to build")

// InsistedError names the gear a request would not give up.
type InsistedError struct {
	// Gear is what was asked for, as the request wrote it.
	Gear string
	// Why is what the catalog said about it.
	Why string
}

// Error implements the error interface.
func (e *InsistedError) Error() string {
	return fmt.Sprintf("the device has no such gear: %s, and the request "+
		"insisted on it (%s)", e.Gear, e.Why)
}

// Unwrap returns ErrInsisted so callers can match with errors.Is.
func (*InsistedError) Unwrap() error { return ErrInsisted }

// WrongDeviceError names both devices, because the fix depends on which is
// wrong: re-measure on the device in hand, or point the setup at the one the
// readings came from.
type WrongDeviceError struct {
	// Measured is the device every reading was taken on.
	Measured string
	// Setup is the device the request is being built for.
	Setup string
}

// Error implements the error interface.
func (e *WrongDeviceError) Error() string {
	return fmt.Sprintf("the measurements were taken on a different device: "+
		"%s, not %s", e.Measured, e.Setup)
}

// Unwrap returns ErrWrongDevice so callers can match with errors.Is.
func (*WrongDeviceError) Unwrap() error { return ErrWrongDevice }
