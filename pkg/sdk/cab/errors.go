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

package cab

import (
	"errors"
	"fmt"
)

// ErrTooShort reports signals with too little in them to divide.
var ErrTooShort = errors.New("not enough signal to work from")

// ErrNotPowerOfTwo reports a length the transform cannot take.
var ErrNotPowerOfTwo = errors.New("the length must be a power of two")

// TooShortError says how much signal there was against how much was needed.
//
// Both sides, because which one is short decides what to do about it: a short
// recording is re-recorded, a short target is a different target.
type TooShortError struct {
	// Sent is what went in, Back what came out. For a match they are the
	// target and what is in hand.
	Sent, Back int
	// Taps is the length that was asked for.
	Taps int
	// What names the two sides the way this operation calls them.
	What [2]string
}

// Error implements the error interface.
func (e *TooShortError) Error() string {
	return fmt.Sprintf("not enough signal to work from: %d samples of %s and "+
		"%d of %s, against %d taps",
		e.Sent, e.What[0], e.Back, e.What[1], e.Taps)
}

// Unwrap returns ErrTooShort so callers can match with errors.Is.
func (*TooShortError) Unwrap() error { return ErrTooShort }

// SilenceError names the side that carried nothing.
//
// Its own type rather than a short signal, because the arithmetic fails for a
// different reason: there is signal, and every bin of it is zero, so what
// would be divided by is nothing rather than too little.
type SilenceError struct {
	// Side is what was silent, in this operation's own words.
	Side string
}

// Error implements the error interface.
func (e *SilenceError) Error() string {
	return fmt.Sprintf("not enough signal to work from: %s was silence", e.Side)
}

// Unwrap returns ErrTooShort so callers can match with errors.Is.
func (*SilenceError) Unwrap() error { return ErrTooShort }

// BadLengthError names a length no device will load.
type BadLengthError struct {
	// Taps is what was asked for.
	Taps int
}

// Error implements the error interface.
func (e *BadLengthError) Error() string {
	return fmt.Sprintf("the length must be a power of two: a device loads %d "+
		"or %d samples, not %d", Short, Long, e.Taps)
}

// Unwrap returns ErrNotPowerOfTwo so callers can match with errors.Is.
func (*BadLengthError) Unwrap() error { return ErrNotPowerOfTwo }
