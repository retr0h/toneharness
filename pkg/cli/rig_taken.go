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
	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// This file answers one question about a reading: which rig took it.
//
// Nothing recorded it, and AGENTS.md states the rule that misses: a figure
// measured on one device is a figure about that device, say which. The two rigs
// here disagree even about a block neither refused, an Ampeg SVT reading 12.7%
// of its energy in the mids on one and 4.0% on the other, and the shipped
// library's baseline of 155.27Hz at -58.55dB sits thirty decibels from where
// the open loop settles. That is almost certainly because it came off the
// closed rig, and the file could not say so.
//
// So "do these figures need taking again" could not be answered from the file,
// which is the whole job the recorded volume was there to do.

// rigTaken is the audio devices a reading went out of and came back in on, and
// whether those were two.
//
// As `--hardware` received them, split the way reamp splits them, so what the
// file records and what the run used cannot disagree. One name is both sides,
// which is the closed loop where the chain hears its own output.
func rigTaken(
	hardware string,
) ([]string, bool) {
	if hardware == "" {
		return nil, false
	}

	play, rec := reamp.Sides(hardware)

	return []string{play, rec}, reamp.TwoSided(hardware)
}

// saysOutput is how the platform is asked which device plays, as a seam.
//
// A package-level var the way `grammarOf` is: the answer comes from CoreAudio
// through cgo, so a test cannot arrange for it to fail and the branch that
// handles failing is the one worth having. It is the branch that keeps a
// reading from claiming a device nobody named.
var saysOutput = reamp.Output

// pinnedOn is the output device the computer's level was pinned on.
//
// The field that silently invalidates the others. A Mac with its default output
// on Bluetooth headphones pins those while the jack feeding the pedal sits
// wherever it was left, which is how a run came to read 66dB of loss with the
// cabling correct: the recorded volume was honest and described a signal path
// nobody measured.
//
// Empty where the platform would not say, which is the same answer the volume
// gives in that case and is honest: a reading that cannot name the device its
// level reached should not claim one.
func pinnedOn() string {
	at, err := saysOutput()
	if err != nil {
		return ""
	}

	return at
}
