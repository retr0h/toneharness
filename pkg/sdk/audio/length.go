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

// Length is how long a recording is, and how much of it is sound.
type Length struct {
	// Seconds is the whole file, start to finish.
	Seconds float64 `json:"seconds"`
	// Head and Tail are the quiet at each end, in seconds.
	Head float64 `json:"head"`
	Tail float64 `json:"tail"`
}

// Sounding is Seconds with the quiet at both ends taken off.
func (l Length) Sounding() float64 { return l.Seconds - l.Head - l.Tail }

// Lengthen reads how long a recording is and where its sound begins and ends.
//
// Both halves, because which one answers depends on what is being asked.
// Comparing a download with the length its source states wants the file's own
// length first: padding makes a download longer than its source and nothing
// makes it shorter, so a file that already matches is the right recording and
// trimming it can only invent a problem. Three records in this corpus were
// reported wrong that way, each with real quiet of its own, a prog intro and
// two fades.
//
// The quiet is what explains a file that is long. A lyric video pads the end
// and these downloads pad the front: one arrived 169 seconds against a stated
// 162, six of it leading silence and two trailing.
//
// The same floor as playing uses, 40dB under the recording's own loudest
// frame, so "quiet" means the same thing here as it does to every other
// measure.
func Lengthen(
	samples []float64,
	rate int,
) Length {
	if rate < 1 || len(samples) < 1 {
		return Length{}
	}

	out := Length{Seconds: float64(len(samples)) / float64(rate)}

	frame := rate / 100
	if frame < 1 {
		return out
	}

	levels := frames(samples, frame)
	held := playing(levels)

	per := float64(frame) / float64(rate)

	// Where the sound starts, and where it stops. A recording that is quiet
	// throughout has no sound to find, and saying its whole length is quiet
	// would report a head and a tail that overlap.
	first, last := -1, -1

	for i, on := range held {
		if !on {
			continue
		}

		if first < 0 {
			first = i
		}

		last = i
	}

	if first < 0 {
		return out
	}

	out.Head = float64(first) * per
	out.Tail = out.Seconds - float64(last+1)*per

	return out
}
