//go:build darwin

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
	"os/exec"
	"strconv"
	"strings"
)

// macOS keeps the output level in the same place the slider does, and AppleScript
// is the only interface to it that does not need a signed helper or a private
// framework. So this shells out, which is the one thing in this package that is
// not Go talking to a library.
//
// The scale is 0 to 100 and is not decibels. It is what the slider shows, which
// is what a person can put back by hand if this ever goes wrong.
const osascript = "/usr/bin/osascript"

// asks runs one line of AppleScript and hands back what it printed.
//
// A variable because it is the only thing in this package that leaves the
// process, and the four answers either side of it — a shell that failed, a
// shell that printed something that is not a number, a level read, a level set
// — are four branches nothing could reach otherwise. A test stands its own
// function here; nothing else assigns it.
var asks = func(
	script string,
) ([]byte, error) {
	return exec.Command(osascript, "-e", script).Output()
}

// volume reads the output level.
func volume() (int, error) {
	out, err := asks("output volume of (get volume settings)")
	if err != nil {
		return 0, &VolumeError{Doing: "reading", Said: err}
	}

	got, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, &VolumeError{Doing: "reading", Said: err}
	}

	return got, nil
}

// setVolume puts the output level where a measurement wants it.
func setVolume(
	to int,
) error {
	if _, err := asks("set volume output volume " + strconv.Itoa(to)); err != nil {
		return &VolumeError{Doing: "setting", Said: err}
	}

	return nil
}
