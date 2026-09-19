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

package audio

// Forward runs a fast Fourier transform in place.
//
// Exported because building an impulse response is the same arithmetic
// pointed the other way, and a second transform written beside this one is
// the shape of mistake this package has already paid for once: two
// implementations of a measurement that agreed to five points on a band share
// and thirty percent on a centroid.
//
// The length must be a power of two.
func Forward(
	re, im []float64,
) {
	transform(re, im)
}

// Inverse undoes Forward, in place and scaled.
//
// By conjugation rather than by a second transform: conjugate, run it
// forwards, conjugate again, divide by the length. Same code, one direction,
// nothing to keep in step.
func Inverse(
	re, im []float64,
) {
	for i := range im {
		im[i] = -im[i]
	}

	transform(re, im)

	n := float64(len(re))

	for i := range re {
		re[i] /= n
		im[i] = -im[i] / n
	}
}
