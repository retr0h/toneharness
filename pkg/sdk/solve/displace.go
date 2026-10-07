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
	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// Displaced is a target as a distance from where the chain already is.
//
// **A record's figures are not a chain's, and subtracting one from the other is
// the mistake this exists to stop.** A record is mixed, mastered and limited; a
// chain is one dry signal through one amplifier into one computer. Mike Dirnt's
// records centre at 126Hz, and a chain measured through a cabinet sits far above
// that, so aiming at 126Hz asks for a darkness that is not in the signal and
// spends every control chasing it. On this amplifier the control with the most
// authority over the centroid is `ChVol`, which carries +27.6dB of level with it,
// so the chain goes quiet as a consequence of the tone it was asked for. That is
// the whole of why a tuned preset came back thin.
//
// What both sides can answer is "how far from its own normal". Dirnt sits 17Hz
// darker than other bass players and 0.035 of the energy lower; Matt Freeman sits
// 56Hz brighter and 0.145 more of it in the mids. Those are small, measured and
// reachable, and they are the same comparison a player's words are earned from.
//
// So the middle moves to the chain's own reading plus the shift, and the spread
// stays exactly as it was: how far a player's records disagree with each other is
// a fact about the player, and it is what the tolerance comes from either way.
//
// A figure the comparison cannot be made for keeps its own middle. That is the
// honest fallback rather than dropping the axis: with one player measured there is
// nobody to be displaced from, and with two the median of the rest is one of them.
//
// The same treatment `translate` has given block choosing since it picked a Motown
// flip-top for punk, for the same reason and with the same arithmetic.
// here is the chain's own reading, already in the units a corpus figure is stated
// in. The caller converts, because it is the one holding the device reading and
// `figuresOf` is the single place that conversion lives.
func Displaced(
	target audio.Across,
	elsewhere map[audio.Figure]float64,
	here map[audio.Figure]float64,
) audio.Across {
	if len(elsewhere) == 0 {
		return target
	}

	mine := target.Measured()
	out := target

	for _, want := range []struct {
		key  audio.Figure
		onto *audio.Spread
	}{
		{audio.KeyLow, &out.Low},
		{audio.KeyMid, &out.Mid},
		{audio.KeyHigh, &out.High},
		{audio.KeyCentroid, &out.Centroid},
	} {
		theirs, held := elsewhere[want.key]
		if !held {
			continue
		}

		at, measuredHere := here[want.key]
		if !measuredHere {
			continue
		}

		// No guard on the subject's own figure. Across.Measured answers for all
		// four of these whether or not a recording had one, so there is nothing
		// here a check could catch.
		want.onto.Mid = at + (mine[string(want.key)] - theirs)
	}

	return out
}
