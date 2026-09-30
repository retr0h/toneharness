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
	"fmt"
	"io"

	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// This file decides whether the headroom trim is needed, from the rig.
//
// The trim exists for one reason: on a rig where the pedal plays the reference
// out of its own output and a lead carries that back to its own input, the chain
// hears itself, and enough gain around the loop oscillates. Thirty decibels is
// what stops it.
//
// On a rig where the computer plays and the pedal's output goes nowhere, there is
// no path back and the trim buys nothing at all. Measured: the same chain reads
// the same spectrum with and without it, 30dB apart in level. So leaving the
// default at -30 there spends thirty decibels of signal over the noise floor for
// a problem that is not present.
//
// The rig says which it is. Two device names mean the computer plays and the
// pedal records, so the loop is open. One name means one device doing both, which
// is the loop. Nothing else has to be asked for, and an explicit --headroom still
// wins, because somebody naming a number has a reason.

// trimFor is the headroom a rig actually needs, and says when it differs from
// what was asked.
//
// asked is what the flag holds and told whether somebody set it themselves. An
// explicit value is obeyed: a person naming -12 on an open rig may be bounding
// something this cannot see, and second-guessing them would make the flag a
// suggestion.
func trimFor(
	w io.Writer,
	hardware string,
	asked float64,
	told bool,
) float64 {
	if told || !openLoop(hardware) {
		return asked
	}

	if asked == 0 {
		return 0
	}

	_, _ = fmt.Fprintf(w,
		"  two devices named, so the chain cannot hear itself and the %.0fdB\n"+
			"  trim is not applied: it would cost that much signal for nothing\n",
		asked)

	return 0
}

// openLoop reports a rig where the pedal's output reaches nothing.
//
// reamp owns what --hardware may say, so it answers this rather than this
// looking for a comma itself. Two names there means play through the first and
// record from the second, and that arrangement is the whole reason the loop can
// be open: the cable into the pedal carries only what the computer plays.
func openLoop(
	hardware string,
) bool {
	return reamp.TwoSided(hardware)
}
