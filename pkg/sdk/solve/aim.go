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

		band := spread[string(key)]

		// A figure nothing measured reads as a middle of zero with no spread,
		// because Across.Measured answers for dynamics, harmonics and lean
		// whether or not any recording had one. Aiming at that would pin the
		// axis to zero and then spend the chain defending it, which is worse
		// than leaving it free: zero is not a measurement, it is the absence of
		// one.
		if want == 0 && band.Low == 0 && band.Mid == 0 && band.High == 0 {
			continue
		}

		// Half the ten-to-ninety band, so the tolerance is a distance from the
		// middle rather than the width of the whole spread.
		tol := (band.High - band.Low) / 2

		if got := floor[key]; tol < got {
			tol = got
		}

		// A figure with neither spread nor a floor is one nothing can say it
		// arrived at, so it is left out rather than pinned to a number the rig
		// cannot resolve.
		if tol <= 0 {
			continue
		}

		out[key] = Aim{Want: want, Tol: tol, Hold: alwaysHeld(key)}
	}

	return out
}

// alwaysHeld is whether an axis stays in the matrix while it is already
// satisfied.
//
// The three band shares, and the reason is the biggest lever a mic'd cabinet
// has. `LowCut` moves a median 190Hz of centroid against the microphone's 29,
// six and a half times as much, and it is far straighter: 0.92 to 0.97 where
// the microphone reads 0.00. **It reaches the centroid by removing the
// bottom.** Taken from 19.9Hz to 500Hz across the fourteen swept cabinets that
// carry one it averages +208Hz of centroid for -43.9 points of the energy
// below 250Hz; a 1x15 Ampeg B-15 goes from 131Hz at 97.09% low to 420Hz at
// 32.18%, and a 4x10 Tweed P10R from 124Hz at 95.41% to 484Hz at 18.49%. On a
// bass that band is the fundamental.
//
// `wanted` builds the matrix from `off > 1 || Hold`, and as its own comment
// says the matrix is the only thing that tells the solve what a move costs. So
// while the low band sat inside its tolerance the solve did not know that
// reaching the centroid spent it, and it spent all of it: asked for 72Hz
// brighter on that 1x15 and told to keep the band where it was, it answered
// with LowCut at 131Hz, which by the cabinet's own slope leaves the band at
// 79.85% against the 97.09% it was given. Held, the same request answers with
// LowCut at 32Hz and the band stays inside its tolerance.
//
// Nor was a later pass going to pull it back. `after` reported every axis
// outside the matrix at its pre-move reading, so that pass said the low band
// was 0.00 tolerances out while its own moves had put it 6.9 out, and
// `Arrived` ends the loop. Both halves are fixed, and they do different jobs:
// `after` makes the damage visible, this stops it being done. A pass applies
// its steps before anything reads `Arrived`, so the hold is the half the
// hardware cares about.
//
// Held rather than weighted, which is the same decision [Aim.Hold] records for
// level and for the same reason: a figure inside tolerance has nothing to ask
// for and everything to lose. These are axes the target does name, so holding
// one is keeping a constraint that is already satisfied rather than defending
// a figure nobody asked about.
//
// All three rather than the low band alone. Holding only `low` would be right
// about this cabinet and wrong about the next instrument: which band a cut
// spends is a fact about where the signal sits, and a guitar's fundamentals
// are not under 250Hz. Three rows against the dozen-plus controls a chain
// carries is cheap, and a satisfied row asks for a move of nearly zero, which
// `stepsOf` drops.
//
// Not the centroid, and not the dynamics, harmonics or lean. The centroid is
// what a target is usually reaching for, so holding it would hold the goal
// still. The other three are figures almost nothing moves in a straight line
// (see `measured.Curve.Carries`), and pinning them would spend the chain
// defending readings a slope cannot describe.
func alwaysHeld(
	key audio.Figure,
) bool {
	switch key {
	case audio.KeyLow, audio.KeyMid, audio.KeyHigh:
		return true
	default:
		return false
	}
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
