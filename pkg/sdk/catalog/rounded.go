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

package catalog

import "math"

// fine is how many parts of a control's own travel a written value keeps.
//
// A thousandth. Finer than anything that decides a value anywhere here and far
// coarser than float arithmetic: a word's nudge is a product of three numbers
// and a swept position is interpolated between two readings eight apart, so
// neither has a thousandth of a control to stand on, let alone the digits past
// it.
const fine = 1000

// Rounded is what to write for a value something computed for a control.
//
// **The digits past this are not a decision anybody made.** `built:` carried
// controls like Treble 0.828986013986014, Mid 0.9760000000000001 and Bass
// 0.6067073170731707, which claim a precision a hundred thousand times finer
// than the thing that produced them and are unreadable beside the catalog's
// own two digits. See task 270.
//
// Of the control's own travel rather than a flat thousandth, because 1,452 of
// this device's 4,835 float controls do not run zero to one: a cabinet's
// LowCut runs 19.9 to 500 and an equaliser's band runs -15 to 15, and a flat
// rule would keep five significant figures on one and three on the other.
//
// **To decimal places rather than onto a grid of thousandths**, which is task
// 275. A thousandth of a travel of 30 is a step of 0.03 and no binary float
// holds a multiple of one: snapping -3.75 onto that grid and dividing back out
// gave -3.7499999999999996, which went into five shipped documents. The places
// are the largest power of ten no coarser than a thousandth of the travel, so
// every value written lands somewhere a decimal can hold exactly, and the
// claim above it still holds as the furthest this will ever round.
//
// An equaliser's band keeps hundredths where a thousandth of its travel is
// 0.03, a cabinet's LowCut keeps tenths where a thousandth is 0.48Hz, and a
// compressor's Attack, running 0.0001 to 0.2, keeps ten-thousandths. A control
// running over a thousand units keeps none, because its own step is wider than
// one.
//
// A control with no travel is returned unchanged. The catalog gives a few a
// range of one value, and taking a logarithm of that spread is the one way
// this could fail.
func Rounded(
	value float64,
	p Param,
) float64 {
	travel := p.Max - p.Min
	if travel <= 0 {
		return value
	}

	// The places a step of a thousandth of the travel needs, which is where
	// `fine` enters: log10 of it against the travel's own magnitude.
	//
	// Rounded, because `math.Log10(1000)` is 2.9999999999999996 and raising
	// ten to 1.9999999999999996 gives 99.99999999999997, which puts every
	// value back off the grid this exists to put it on.
	places := math.Round(math.Log10(fine)) - math.Floor(math.Log10(travel))
	if places <= 0 {
		return math.Round(value)
	}

	scale := math.Pow(10, places)

	return math.Round(value*scale) / scale
}
