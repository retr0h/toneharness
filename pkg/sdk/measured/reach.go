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

package measured

import (
	"math"
	"slices"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// This file answers a question the loop could only answer by running: can this
// chain get anywhere near this target at all.
//
// A run that cannot reach its target still costs everything a run that can
// costs. A pass reads one measurement per control and a chain of a dozen dials
// over three passes is around forty readings, five minutes of real-time audio
// with a pedal held for all of it, and at the end of it the answer is "the
// chain will not reach this target". The sweeps under resources/sweeps/ already
// hold what every control does. They can say it in milliseconds.
//
// What they cannot say is that a target IS reachable. Slopes are local, they
// are read one control at a time, and controls interact, so the arithmetic here
// bounds the answer rather than deciding it. Read the two numbers as what they
// are: Span is ground a chain was measured standing on, and Swing is the most
// its controls could move it under a model that flatters them.

// Reach is how far one chain was measured moving one figure.
type Reach struct {
	// Low and High are the lowest and highest this figure actually read
	// across every position of every control that was swept, and Mid the
	// middle of those readings.
	//
	// Measured rather than modelled: these are numbers the hardware produced.
	// Nothing here is inferred from a slope, so a target inside this range is
	// one some setting of this chain demonstrably hit.
	Low, High, Mid float64
	// Swing is the most the controls could move this figure, summed across
	// them: a dial contributes its slope across its whole range, weighted by
	// how straight that slope was, and a list the distance between its
	// furthest settings.
	//
	// An optimistic bound, and deliberately so. It assumes every control
	// pulls the same way and that each slope holds across a whole range it
	// was only read locally, neither of which is true. What that buys is the
	// direction of the error: a gap wider than the Swing is a gap nothing
	// flattering could close, so this refuses a target rather than promising
	// one.
	Swing float64
	// Alone says every sweep behind this figure was taken with its block on
	// its own, so none of it describes the chain.
	//
	// The field Curves.Isolated exists to carry, and it is load-bearing. An
	// amplifier swept with no cabinet in front of it has no speaker rolloff,
	// so its Treble moved the centroid 12,763Hz per turn where the same
	// control in a real chain moves it 3,250, and its Mid moved the centroid
	// the other way entirely. Four of eleven controls changed sign.
	Alone bool
	// Readings is how many positions carried this figure, and Straight the
	// least straight of the slopes that went into Swing.
	//
	// Straight keeps Swing honest, because a slope through a curve that rises
	// and then falls describes neither half. Near zero, Swing is arithmetic
	// on a number with no meaning.
	Readings int
	Straight float64
}

// Reaches is what a chain's blocks were measured doing, per figure.
//
// Every sweep handed to it, taken together. A chain is its blocks, and the
// question is what the chain can do rather than what any one block can.
//
// The caller is trusted to hand over sweeps of the chain in hand. A slope is
// not a property of a control — the same Treble into a 4x12 and into a 1x15
// are two different numbers — so sweeps of blocks that never met describe a
// chain nobody built.
func Reaches(
	of ...Curves,
) map[audio.Figure]Reach {
	out := map[audio.Figure]Reach{}

	for _, figure := range Named() {
		got, ok := reachOf(figure, of)
		if !ok {
			continue
		}

		out[figure] = got
	}

	return out
}

// reachOf gathers one figure across every control of every sweep.
func reachOf(
	figure audio.Figure,
	of []Curves,
) (Reach, bool) {
	var (
		read     []float64
		swing    float64
		straight = math.Inf(1)
		alone    bool
	)

	for _, curves := range of {
		alone = alone || curves.Isolated

		for _, curve := range curves.Controls {
			for _, p := range curve.Points {
				if v, ok := p.figure(figure); ok {
					read = append(read, v)
				}
			}

			// wasStraight rather than `worst`, which reads as the worst figure
			// and is this one control's own straightness. The accumulator
			// beside it keeps the least straight of them, which is where the
			// word worst would belong.
			moved, wasStraight, ok := movement(curve, figure)
			if !ok {
				continue
			}

			// Weighted by how straight the slope was, because Straight is the
			// fraction of the figure's movement the line accounts for. A
			// control whose readings rise and then fall has a slope that
			// describes neither half, and counting its full range as movement
			// is arithmetic on a number with no referent. This amplifier's
			// Bias reads 0.21 straight and its Hum 0.11.
			swing += moved * wasStraight
			straight = math.Min(straight, wasStraight)
		}
	}

	if len(read) == 0 {
		return Reach{}, false
	}

	if math.IsInf(straight, 1) {
		straight = 0
	}

	return Reach{
		Low:      slices.Min(read),
		High:     slices.Max(read),
		Mid:      median(read),
		Swing:    swing,
		Alone:    alone,
		Readings: len(read),
		Straight: straight,
	}, true
}

// movement is the most one control could move one figure.
//
// A dial's slope runs over its own range, which is read from the span it was
// swept over rather than assumed to be nought to one: 1,452 of this device's
// float controls are neither.
//
// A list has no slope and contributes the distance between its furthest
// settings, which is exactly what Spread holds. Counted as fully straight,
// because straightness is a statement about a line and a list is not one.
func movement(
	curve Curve,
	figure audio.Figure,
) (moved, straight float64, ok bool) {
	if spread, listed := curve.Spread[figure]; listed {
		return spread, 1, true
	}

	fit, fitted := curve.Fits[figure]
	if !fitted {
		return 0, 0, false
	}

	return math.Abs(fit.PerTurn) * (curve.Span.High - curve.Span.Low),
		fit.Straight, true
}
