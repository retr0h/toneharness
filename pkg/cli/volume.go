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
	"errors"
	"fmt"
	"io"

	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// This file pins the computer's own output level before anything is measured.
//
// Left to a person it drifts, and it drifts silently: an amplifier's distortion
// depends on how hard it is driven, so a campaign taken at a different setting
// measures every amplifier as a different amplifier rather than as the same one
// louder. It is a tone control that happens to live on a laptop.
//
// So the tool sets it rather than asking. That reaches outside the tool, which
// is why it always says what it did, and why the level it actually ends up at is
// what gets recorded rather than the level that was asked for.
//
// It only reaches the signal on a rig that plays the reference out of the
// computer's own output, which is the rig that opens the measuring loop. On the
// one-device rig the pedal plays and this is not in the path at all, so a
// campaign there records it and nothing depends on it.

// unknownVolume is what a library records when the platform would not say.
//
// A number rather than an absent field, because zero is a real volume and
// "nobody checked" is not the same claim as "it was silent".
const unknownVolume = -1

// levelled puts the computer's output where a campaign wants it, and answers
// what it is actually at.
//
// A platform that cannot be asked is reported and carries on. It is only in the
// signal path on one of the two rigs, so refusing here would stop a campaign on
// the other for a reason that does not apply to it.
func levelled(
	w io.Writer,
	want int,
) int {
	at, moved, err := reamp.Held(want)

	switch {
	case errors.Is(err, reamp.ErrNoVolume):
		_, _ = fmt.Fprintf(w,
			"  [warn] this platform will not say what its output level is, so\n"+
				"         nothing pinned it. Readings taken by playing through the\n"+
				"         computer are not reproducible without it.\n")

		return unknownVolume
	case err != nil:
		_, _ = fmt.Fprintf(w,
			"  [warn] the computer's output level could not be set to %d: %v.\n"+
				"         Carrying on at whatever it is, which is recorded.\n", want, err)

		return unknownVolume
	case moved:
		_, _ = fmt.Fprintf(w,
			"  the computer's output level moved to %d, for a consistent rig\n", at)
	default:
		_, _ = fmt.Fprintf(w,
			"  the computer's output level is %d, where a campaign wants it\n", at)
	}

	return at
}
