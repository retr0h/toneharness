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

// Middle is the median of several readings of the same thing, figure by
// figure.
//
// For the baseline, which is the one reading everything else is calibrated
// against and the noisiest one the rig produces. An empty loop is measuring
// hiss, and hiss has no stable centroid: five consecutive reads of an
// untouched rig spanned 56 Hz and 5.1 dB, against a drift gate of 25 Hz and
// 3 dB. So a single take could not tell a rewired rig from the same rig read
// twice, and two of those five fired the warning on different axes with
// nothing changed between them.
//
// The median rather than the mean, because three takes is the point where one
// wild reading stops mattering at all rather than being averaged in. Three
// takes of four seconds is eight seconds added to a campaign measured in
// hours.
//
// A figure absent from every take stays absent. One present in some is the
// median of those: a transient needs a note starting, so a take that found
// none is silent about it rather than claiming zero, and dropping the takes
// that did find one would throw away the only answers there are.
func Middle(
	takes []Figures,
) Figures {
	if len(takes) == 0 {
		return Figures{}
	}

	if len(takes) == 1 {
		return takes[0]
	}

	plain := func(of func(Figures) float64) float64 {
		got := make([]float64, 0, len(takes))
		for _, t := range takes {
			got = append(got, of(t))
		}

		return median(got)
	}

	maybe := func(of func(Figures) *float64) *float64 {
		got := make([]float64, 0, len(takes))

		for _, t := range takes {
			if v := of(t); v != nil {
				got = append(got, *v)
			}
		}

		if len(got) == 0 {
			return nil
		}

		at := median(got)

		return &at
	}

	return Figures{
		Centroid:  plain(func(f Figures) float64 { return f.Centroid }),
		Level:     plain(func(f Figures) float64 { return f.Level }),
		Low:       plain(func(f Figures) float64 { return f.Low }),
		Mid:       plain(func(f Figures) float64 { return f.Mid }),
		High:      plain(func(f Figures) float64 { return f.High }),
		Transient: maybe(func(f Figures) *float64 { return f.Transient }),
		Decay:     maybe(func(f Figures) *float64 { return f.Decay }),
		Dynamics:  maybe(func(f Figures) *float64 { return f.Dynamics }),
		Harmonics: maybe(func(f Figures) *float64 { return f.Harmonics }),
		Lean:      maybe(func(f Figures) *float64 { return f.Lean }),
	}
}
