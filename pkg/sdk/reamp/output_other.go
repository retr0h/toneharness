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

//go:build !darwin

package reamp

// Only macOS knows how to do this here, and saying so is better than a stub
// that reports success. A platform that cannot be asked leaves the device where
// a person put it, and the caller warns rather than refusing: the rig where the
// pedal plays the reference does not use the computer's output at all.

// output answers which device the computer plays through.
func output() (string, error) {
	return "", &OutputError{Doing: "reading", Said: errNoOutput}
}

// setOutput makes a named device the one the computer plays through.
func setOutput(
	string,
) error {
	return &OutputError{Doing: "setting", Said: errRefused}
}

// outputs answers every device that can play.
func outputs() []string { return nil }
