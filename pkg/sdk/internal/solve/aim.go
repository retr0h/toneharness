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

// Aims turns what a body of records measures as into a target.
//
// The middle is what to reach and the spread is how close is close enough, so
// an axis the records agree about is one the answer has to hit and an axis they
// disagree about is one it need not be precise on. That is the whole reason a
// player or a genre is an easier target than it sounds: the freedom on the
// loose axes goes into satisfying the tight ones.
//
// floor is the loop's own repeatability, per figure, and it is the lower bound
// on every tolerance. Without it a body of records that happen to agree
// exactly, or a target taken from one recording, would ask for a precision the
// rig cannot demonstrate, and the loop would never report arriving.
func Aims(
	across audio.Across,
	floor map[audio.Figure]float64,
) map[audio.Figure]Aim {
	mid := across.Measured()
	spread := across.Spreads()

	out := make(map[audio.Figure]Aim, len(mid))

	for _, key := range audio.MeasuredKeys() {
		want, measured := mid[string(key)]
		if !measured {
			continue
		}

		// Half the ten-to-ninety band, so the tolerance is a distance from the
		// middle rather than the width of the whole spread.
		tol := (spread[string(key)].High - spread[string(key)].Low) / 2

		if got := floor[key]; tol < got {
			tol = got
		}

		// A figure with neither spread nor a floor is one nothing can say it
		// arrived at, so it is left out rather than pinned to a number the rig
		// cannot resolve.
		if tol <= 0 {
			continue
		}

		out[key] = Aim{Want: want, Tol: tol}
	}

	return out
}

// Only keeps the axes named, and drops the rest.
//
// What makes a genre a partial target. Punk is distinctive on some axes and
// ordinary on the others, and pinning the ordinary ones would spend the chain
// defending figures nobody asked about. A full point in nine dimensions is
// often unreachable with the controls a device has; a target that pins three
// and shrugs at six usually is not.
func Only(
	aims map[audio.Figure]Aim,
	keep []audio.Figure,
) map[audio.Figure]Aim {
	out := make(map[audio.Figure]Aim, len(keep))

	for _, key := range keep {
		if aim, ok := aims[key]; ok {
			out[key] = aim
		}
	}

	return out
}

// Floor reads the loop's own wander out of a block's sweep.
//
// A sweep records how far each figure moved across repeat takes with nothing
// touched, which is the smallest difference anybody can demonstrate. Taking the
// largest across the blocks in a chain rather than the mean, because a chain is
// as repeatable as its least repeatable part.
func Floor(
	noise ...map[audio.Figure]float64,
) map[audio.Figure]float64 {
	out := map[audio.Figure]float64{}

	for _, one := range noise {
		for key, got := range one {
			out[key] = math.Max(out[key], got)
		}
	}

	return out
}
