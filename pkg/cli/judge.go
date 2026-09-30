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

// judgement says whether a live measurement can be believed.
type judgement int

const (
	// believable is a reading of the block.
	believable judgement = iota
	// tooQuiet is a reading of the noise floor: the chain is muted here and
	// what came back is hiss. The centroid of hiss is broadband and reads
	// high, so a control that mutes one end of its travel reports an enormous,
	// repeatable move that is not a move.
	tooQuiet
	// tooLoud is a reading of the converters running out of headroom. Flat
	// tops make harmonics that were never in the signal, so it reads as a
	// bright block and is not one.
	tooLoud
)

// silentBelow is the level under which a reading is the noise floor's.
//
// The one place `settled - silent` is worked out. A sweep stores it on its
// Curve as SilentBelow, which is part of the serialised contract, and the
// list comparison rebuilt it inline — the same rule in two places, one saved
// and one recomputed, with nothing holding them together.
func silentBelow(
	settled float64,
) float64 {
	return settled - silent
}

// judge says whether a live reading can be believed, and what to print if not.
//
// Both callers need the words and one of them needs which way it failed: a
// sweep buckets into a Curve's MutedAt against its ClippedAt, and the list
// comparison only says so and moves on. Before this they were two switches
// printing byte-identical strings, which agree until somebody rewords one.
func judge(
	level float64,
	floor float64,
) (judgement, string) {
	switch {
	case level < floor:
		return tooQuiet, "silent, so its figures are of the noise"
	case level > clipped:
		return tooLoud, "clipped, so its figures are the converters'"
	default:
		return believable, ""
	}
}
