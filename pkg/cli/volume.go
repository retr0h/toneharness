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

	"github.com/retr0h/toneharness/pkg/sdk"
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

// holds puts the computer's output level where a campaign wants it, answering
// where it ended up and whether it had to move.
//
// A function rather than a package call, because reamp.Held reaches the
// platform and the four answers below are four branches nothing could reach
// otherwise. reamp.Held is what every caller passes.
type holds func(want int) (int, bool, error)

// levelled puts the computer's output where a campaign wants it, and answers
// what it is actually at.
//
// A platform that cannot be asked is reported and carries on. It is only in the
// signal path on one of the two rigs, so refusing here would stop a campaign on
// the other for a reason that does not apply to it.
func levelled(
	w io.Writer,
	want int,
	hold holds,
) int {
	at, moved, err := hold(want)

	switch {
	case errors.Is(err, reamp.ErrNoVolume):
		_, _ = fmt.Fprintf(w,
			"  [warn] this platform will not say what its output level is, so\n"+
				"         nothing pinned it. Readings taken by playing through the\n"+
				"         computer are not reproducible without it.\n")

		return unknownVolume
	case errors.Is(err, reamp.ErrVolume):
		_, _ = fmt.Fprintf(w,
			"  [warn] the computer's output level could not be set to %d: %v.\n"+
				"         Carrying on at whatever it is, which is recorded.\n", want, err)

		return unknownVolume
	case err != nil:
		// Anything else is not something this knows the shape of, so it says
		// so rather than describing it as a level that would not move.
		_, _ = fmt.Fprintf(w,
			"  [warn] the computer's output level: %v.\n"+
				"         Carrying on at whatever it is, which is recorded.\n", err)

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

// plays is how the computer's output device is set, so a test stands its own
// function here. reamp.PlaysThrough is what every caller passes.
type plays func(want string) (string, bool, error)

// playsThrough makes the computer play through the device the reference is meant
// to go out of, before the level is pinned on it.
//
// The level belongs to a device. Pinning it while the platform's default is some
// other device sets a number about a device that is not in the signal path, and
// the figures then move with a level nothing recorded: a Mac with its default on
// a pair of Bluetooth headphones pinned those to 38 while the headphone jack
// feeding the pedal sat wherever it was left, and the loop read 66dB of loss.
//
// Set rather than asked about, for the same reason the level is set rather than
// asked about: a person holding two settings steady across campaigns weeks apart
// is not a thing anybody does. It reaches outside the tool, so it always says
// what it did and what it was before.
//
// Only where the computer plays the reference. On the one-device rig the pedal
// plays and the computer's output is not in the path, so there is nothing to set.
//
// A platform that cannot be told is warned about and carried on from, because
// that is a rig somebody can still set by hand and refusing would stop a
// campaign over something already correct.
func playsThrough(
	w io.Writer,
	opts TuneOptions,
	through plays,
) {
	if !openLoop(opts.Hardware) {
		return
	}

	want, _ := reamp.Sides(opts.Hardware)

	at, moved, err := through(want)
	if err != nil {
		_, _ = fmt.Fprintf(w,
			"  [warn] the computer's output device could not be set to %q: %v.\n"+
				"         Set it by hand, or the level pinned below is a number about\n"+
				"         a device that is not in the signal path.\n", want, err)

		return
	}

	if moved {
		_, _ = fmt.Fprintf(w,
			"  the computer now plays through %s, so the level below is pinned on\n"+
				"  the device feeding the pedal\n", at)

		return
	}

	_, _ = fmt.Fprintf(w,
		"  the computer plays through %s, which the level below belongs to\n", at)
}

// reaches is a bench that knows whether the pinned output level is its own.
//
// An interface rather than a concrete type, because the benches a test hands in
// are doubles and only the real one talks to CoreAudio. A double that does not
// answer this is a double that does not claim to.
type reaches interface {
	PinReaches() bool
}

// pinLands warns when the level that was pinned is not the level in the signal
// path.
//
// The output level is a tone control, because an amplifier's distortion depends
// on how hard it is driven, and the whole reason it is pinned is that every
// figure moves with it. It is pinned on the platform's default output, and
// `--hardware` names whichever device the reference is played through. When
// those are two different devices the pin reaches one and the bass goes out of
// the other, so the recorded level describes nothing.
//
// Measured, which is how this came to be written: a Mac with its output on a
// pair of Bluetooth headphones pinned those to 38 while the headphone jack
// feeding the pedal sat wherever it was left, and the return read 66dB down. The
// run called it a dead loop, which it was, and nothing said why.
//
// Only where the computer plays the reference. On the one-device rig the pedal
// plays and the computer's output is not in the path at all, so there is nothing
// for a pin to reach and nothing worth saying.
func pinLands(
	w io.Writer,
	opts TuneOptions,
	bench sdk.Bench,
) {
	if !openLoop(opts.Hardware) {
		return
	}

	knows, ok := bench.(reaches)
	if !ok || knows.PinReaches() {
		return
	}

	_, _ = fmt.Fprintf(w,
		"  [warn] the level was pinned on this computer's default output, and the\n"+
			"         reference plays through a different one: %s.\n"+
			"         So the pin reached a device that is not in the signal path, and\n"+
			"         every figure here moves with a level nothing recorded. Make the\n"+
			"         device feeding the pedal the default output, or name the default\n"+
			"         in --hardware.\n", bench.Name())
}
