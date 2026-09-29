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

package solve

import (
	"math"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// Nudged is one figure somebody asked for more or less of.
//
// Resolved rather than a word: which figure a word speaks to and which way it
// points are the vocabulary's business, and this package only knows how far a
// genre's records disagree.
type Nudged struct {
	// Term is the word that asked, carried so a run can say why it moved.
	Term string
	// Axis is the question the word answers, as the vocabulary names it.
	Axis string
	// Key is the figure to aim differently at.
	Key audio.Figure
	// Up is which way. "bright" is up on the centroid and "dark" is down it.
	Up bool
	// Steps is how many tolerances. One is what a person means by a
	// noticeable change; "much darker" is more and "a touch darker" is less.
	Steps float64
}

// Nudge moves a target by what somebody heard.
//
// One tolerance per nudge, which is the only honest unit available. A
// tolerance is the spread across a genre's own records, so a nudge of one asks
// for a sound as far from the target as those records are from each other:
// audibly different rather than a fraction somebody chose. It is also the unit
// the loop already reports residuals in, so "2.4 tolerances out" and "nudged
// by one" are the same measure.
//
// A figure nothing aims at is left alone. A word can only move a target that
// exists, and a genre pins the figures its records agree about and shrugs at
// the rest by design.
//
// Returns a new map. The aims are read again every pass to report what is
// still out, and a nudge applied to the same map twice would walk the target
// away a tolerance at a time.
func Nudge(
	aims map[audio.Figure]Aim,
	of []Nudged,
) map[audio.Figure]Aim {
	out := make(map[audio.Figure]Aim, len(aims))
	for key, aim := range aims {
		out[key] = aim
	}

	for _, n := range of {
		aim, ok := out[n.Key]
		if !ok || aim.Tol <= 0 {
			continue
		}

		steps := n.Steps
		if steps == 0 {
			steps = 1
		}

		step := aim.Tol * steps
		if !n.Up {
			step = -step
		}

		aim.Want = held(n.Key, aim.Want+step)
		out[n.Key] = aim
	}

	return out
}

// held keeps a nudged target inside what the figure can be.
//
// A band share is a share: asking for 1.05 of the low band is asking for more
// energy than there is, and the solve would spend every control chasing it and
// report a chain that cannot reach the target. Clamping says the same thing
// more usefully, because the target it stops at is the one the records
// themselves top out at.
//
// Everything else is clamped at zero only. A centroid in hertz has no ceiling
// this package knows, and a decay long enough to be silly is still a decay.
func held(
	key audio.Figure,
	want float64,
) float64 {
	want = math.Max(want, 0)

	if share(key) {
		return math.Min(want, 1)
	}

	return want
}

// share reports a figure measured as a fraction of the whole rather than in a
// unit of its own.
func share(
	key audio.Figure,
) bool {
	switch key {
	case audio.KeyLow, audio.KeyMid, audio.KeyHigh, audio.KeyHarmonics:
		return true
	default:
		return false
	}
}
