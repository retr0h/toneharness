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
package cli

import (
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// opened is a bench this package opened and therefore has to give back.
//
// sdk.Bench is what a measuring run needs and says nothing about a lifetime,
// because most of them are handed one. This is the narrower thing benchFor
// returns a closer for.
type opened interface {
	sdk.Bench
	Close() error
}

// opens is reamp's own, named so a test can stand in for it.
//
// Opening for real needs the pedal on USB and the reference going through it,
// so the half of benchFor that succeeds is not reachable from a test otherwise,
// and the part that matters there is the closer: a bench this package opened
// and did not close holds the audio device until the process exits.
var opens = func(hardware string) (opened, error) { return reamp.Open(hardware) }

// benchFor is the audio loop a measuring run reads through, and how to let it
// go.
//
// One copy rather than one per command. Three commands opened their own and the
// tuner made a fourth, each with the same four lines and the same deferred
// close, and none of the four can be reached from a test: opening the loop needs
// the pedal on USB and the reference signal going through it. Four copies of a
// block nothing can execute is four places for the close to be forgotten.
//
// A bench the caller already holds is handed straight back with a closer that
// does nothing, because a caller who supplied one owns its lifetime. That is
// what every test does.
func benchFor(
	given sdk.Bench,
	hardware string,
) (sdk.Bench, func(), error) {
	if given != nil {
		return given, func() {}, nil
	}

	open, err := opens(hardware)
	if err != nil {
		return nil, nil, err
	}

	return open, func() { _ = open.Close() }, nil
}
