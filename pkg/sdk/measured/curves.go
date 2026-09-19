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
	"encoding/json"
	"fmt"
	"io"
	"math"
)

// Curves is what every control of one block does, measured.
//
// The other half of a fingerprint. A fingerprint says where a block sits with
// nobody touching it, which is what picking one out of six hundred needs.
// This says what its controls do, which is what setting the chain needs once
// it has been picked.
type Curves struct {
	// Device is the hardware it was measured on.
	Device string `json:"device"`
	// Gear is what a person calls the block, and Block the model identifier
	// the device addresses it by.
	Gear  string `json:"gear"`
	Block string `json:"block"`
	// Slot is where the block sat in the chain while it was swept. A block's
	// slot is its position plus one.
	Slot int `json:"slot"`
	// Isolated says the chain held this block and nothing else.
	//
	// The most important field here. A slope is not a property of a control:
	// the same Treble into a 4x12 and into a 1x15 are two different numbers.
	// Measured in a chain these describe the chain, and nothing downstream
	// may treat that as describing the block.
	Isolated bool `json:"isolated"`
	// Chain is the whole rig the readings came through, as the device
	// reported it, so a later reader can rebuild it rather than trust a
	// sentence about it.
	Chain string `json:"chain"`
	// Reference identifies the signal every reading was taken against.
	Reference Reference `json:"reference"`
	// Controls is every control that was swept, by name.
	Controls map[string]Curve `json:"controls"`
}

// Curve is one control, moved through its range.
type Curve struct {
	// Index is the parameter's position in the model's own list, which is
	// the only thing that identifies it on the wire.
	Index int `json:"index"`
	// Control is what the catalog calls that position.
	Kind    string `json:"kind"`
	Control string `json:"control"`
	// Span is the range it was swept over.
	Span Span `json:"span"`
	// Noise is how far each figure wandered across repeat takes with nothing
	// touched. A move smaller than this moved nothing anybody can
	// demonstrate.
	Noise map[string]float64 `json:"noise"`
	// Settled is the level a reading sits at when the chain is working, and
	// SilentBelow the level under which it is not.
	Settled     float64 `json:"settled"`
	SilentBelow float64 `json:"silentBelow"`
	// MutedAt is every position where nothing came through.
	//
	// Kept rather than dropped, because where a control mutes a chain is
	// worth knowing, and left out of the arithmetic, because the figures at
	// those positions describe hiss.
	MutedAt []float64 `json:"mutedAt"`
	// ClippedAt is every position that hit the converters' ceiling.
	//
	// The other end of the same problem, and the more dangerous one because
	// it reads as a finding. A clipped recording's spectrum is the
	// clipping's: flat tops make harmonics that were never in the signal, so
	// the centroid leaps. One microphone of a cabinet's twelve clipped and
	// read 4,471 Hz where the other eleven sat between 126 and 147, which
	// would have been filed as this cabinet's microphone choice moving its
	// centre of gravity by 4,345 Hz.
	ClippedAt []float64 `json:"clippedAt"`
	// Points is every position that carried signal.
	Points []Point `json:"points"`
	// Fits is the slope of each figure against this control, for a control
	// that is a dial.
	Fits map[string]Fit `json:"fits,omitempty"`
	// Spread is how far apart the settings sit, for a control that is a
	// list.
	//
	// A list has no slope. A cabinet's Mic is twelve microphones and the
	// fourth does not sit between the third and the fifth in any sense a line
	// describes, so "so much centroid per microphone" is a number with no
	// referent.
	Spread map[string]float64 `json:"spread,omitempty"`
}

// Span is the range a control was swept over.
type Span struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
	// FromCatalog says the range came from the catalog rather than from
	// somebody typing one. It matters: 1,452 of the device's 4,835 float
	// controls do not run zero to one, and a Simple EQ's Mid Freq swept 0..1
	// never leaves its bottom stop and reports as a control that does
	// nothing.
	FromCatalog bool `json:"fromCatalog"`
}

// Point is one position of a control and what the chain read there.
type Point struct {
	Value float64 `json:"value"`
	Figures
}

// Fit is how much a figure moves per full turn, and how straight that is.
type Fit struct {
	// PerTurn is the least-squares slope across the whole range.
	PerTurn float64 `json:"perTurn"`
	// Straight is the fraction of the figure's movement the line accounts
	// for. One is a control that really is a line.
	//
	// It keeps the slope honest. This amplifier's Master leaps from 130Hz to
	// 10,795Hz over one step and then falls steadily to 3,769Hz, so no single
	// slope is true anywhere along it, and the average of a rise and a fall
	// describes neither. Nothing may use PerTurn as a slope without reading
	// this first.
	Straight float64 `json:"straight"`
}

// LoadCurves reads what every control of one block does.
func LoadCurves(
	r io.Reader,
) (Curves, error) {
	var out Curves

	body, err := io.ReadAll(r)
	if err != nil {
		return Curves{}, fmt.Errorf("reading the curves: %w", err)
	}

	if err := json.Unmarshal(body, &out); err != nil {
		return Curves{}, fmt.Errorf("decoding the curves: %w", err)
	}

	if len(out.Controls) == 0 {
		return Curves{}, fmt.Errorf("the curves name no controls")
	}

	return out, nil
}

// Fitted is the slope of one figure across a control's measured positions.
//
// Least squares rather than the difference between the ends, because the ends
// are two readings and this uses every one.
func Fitted(
	points []Point,
	figure string,
) Fit {
	xs := make([]float64, 0, len(points))
	ys := make([]float64, 0, len(points))

	for _, p := range points {
		v, ok := p.figure(figure)
		if !ok {
			continue
		}

		xs = append(xs, p.Value)
		ys = append(ys, v)
	}

	if len(xs) < 2 || spread(xs) == 0 {
		return Fit{Straight: 1}
	}

	slope, intercept := line(xs, ys)

	var miss, about float64

	mean := average(ys)

	for i, y := range ys {
		miss += math.Pow(y-(slope*xs[i]+intercept), 2)
		about += math.Pow(y-mean, 2)
	}

	// A figure that did not move has no shape to miss, so the line accounts
	// for all of nothing rather than none of it.
	straight := 1.0
	if about != 0 {
		straight = 1 - miss/about
	}

	return Fit{PerTurn: slope, Straight: straight}
}

// Apart is how far a figure's readings sit from each other.
//
// What a list of settings has instead of a slope.
func Apart(
	points []Point,
	figure string,
) (float64, bool) {
	var (
		low, high float64
		seen      bool
	)

	for _, p := range points {
		v, ok := p.figure(figure)
		if !ok {
			continue
		}

		if !seen {
			low, high, seen = v, v, true

			continue
		}

		low = math.Min(low, v)
		high = math.Max(high, v)
	}

	return high - low, seen
}

// figure reads one of a point's figures by name.
//
// Two of them can be absent: a transient needs a note starting and a decay
// needs one ending, and a reading holding neither has no answer rather than
// an answer of zero.
func (p Point) figure(
	name string,
) (float64, bool) {
	switch name {
	case "centroid":
		return p.Centroid, true
	case "level":
		return p.Level, true
	case "low":
		return p.Low, true
	case "mid":
		return p.Mid, true
	case "high":
		return p.High, true
	case "dynamics":
		return p.Dynamics, true
	case "harmonics":
		return p.Harmonics, true
	case "lean":
		return p.Lean, true
	case "transient":
		if p.Transient == nil {
			return 0, false
		}

		return *p.Transient, true
	case "decay":
		if p.Decay == nil {
			return 0, false
		}

		return *p.Decay, true
	}

	return 0, false
}

// Named is every figure a curve is reported in.
//
// The same ones a record is described in, so a block's curve and a record's
// reading are in one vocabulary. Level is not among a record's and is among
// these, because a record's loudness is the mastering engineer's and a
// block's is the block's.
func Named() []string {
	return []string{
		"centroid", "level", "low", "mid", "high",
		"transient", "decay", "dynamics", "harmonics", "lean",
	}
}

// line is the least-squares fit through a set of points.
//
// Only ever called with at least two positions that are not all the same,
// because Fitted refuses anything else first. So the denominator cannot be
// zero and there is no guard here for it: a guard nothing can reach is a
// branch nothing can test and a claim nobody can check.
func line(
	xs, ys []float64,
) (float64, float64) {
	mx, my := average(xs), average(ys)

	var top, bottom float64

	for i := range xs {
		top += (xs[i] - mx) * (ys[i] - my)
		bottom += (xs[i] - mx) * (xs[i] - mx)
	}

	slope := top / bottom

	return slope, my - slope*mx
}

// average is the mean of a set of values, which is never empty here.
func average(
	of []float64,
) float64 {
	var sum float64
	for _, v := range of {
		sum += v
	}

	return sum / float64(len(of))
}

// spread is the distance between the largest and smallest of a set.
func spread(
	of []float64,
) float64 {
	low, high := math.Inf(1), math.Inf(-1)

	for _, v := range of {
		low = math.Min(low, v)
		high = math.Max(high, v)
	}

	return high - low
}
