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

package compile

import (
	"math"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// fine is how many parts of a control's own travel a written value keeps.
//
// A thousandth. Finer than anything that decides a value here and far coarser
// than float arithmetic: a word's nudge is a product of three numbers and a
// swept position is interpolated between two readings eight apart, so neither
// has a thousandth of a control to stand on, let alone the digits past it.
const fine = 1000

// rounded is what to write for a value this package computed.
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
// A control with no travel is returned unchanged. The catalog gives a few a
// range of one value, and dividing by that spread is the one way this could
// fail.
func rounded(
	value float64,
	p catalog.Param,
) float64 {
	travel := p.Max - p.Min
	if travel <= 0 {
		return value
	}

	by := fine / travel

	return math.Round(value*by) / by
}
