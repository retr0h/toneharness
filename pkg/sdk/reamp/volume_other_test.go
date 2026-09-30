//go:build !darwin

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
	"testing"
)

// TestNoPlatformButMacOSAnswers covers the stub every other platform gets.
//
// Nothing portable reads a system output level, so everywhere but macOS says so
// rather than guessing, and a campaign there records -1 instead of a number
// nobody checked.
//
// Behind the same build tag as the file it covers, because that file does not
// exist on a Mac and the darwin test cannot reach it. It is the inverse of the
// gap in usb_darwin.go: this one is invisible from a Mac and CI is the only
// place it runs.
func TestNoPlatformButMacOSAnswers(
	t *testing.T,
) {
	if _, err := volume(); !errors.Is(err, ErrNoVolume) {
		t.Fatalf("reading the level: want ErrNoVolume, got %v", err)
	}

	if err := setVolume(38); !errors.Is(err, ErrNoVolume) {
		t.Fatalf("setting the level: want ErrNoVolume, got %v", err)
	}
}
