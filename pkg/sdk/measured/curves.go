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
	"slices"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
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
	// Instrument is what was played through it, as the corpus names one, and
	// Headroom how far the chain's own output was turned down while these
	// were taken.
	//
	// Both because a figure is only a figure about the instrument that made it
	// and the loop that carried it. A guitar rig ranked against readings taken
	// with a bass is ranked against the wrong distribution, and a reading
	// taken without headroom is of the measuring lead feeding the chain back
	// into itself.
	Instrument string `json:"instrument,omitempty"`
	// Volume is the computer's own output level while these were swept, on the
	// platform's own 0 to 100 scale, and -1 where the platform would not say.
	//
	// Recorded for the same reason Headroom is: it decides what the readings are
	// of. It is a tone control rather than a level control, because an
	// amplifier's distortion depends on how hard it is driven, so a sweep at
	// another level is a sweep of a different amplifier.
	Volume   int     `json:"volume"`
	Headroom float64 `json:"headroom,omitempty"`
	// Rig is the audio devices the reference went out of and came back in on,
	// as `--hardware` received them, and Open says the two were different so
	// the chain could not hear itself.
	//
	// **The field that decides whether a reading may be compared with
	// another**, and the one this file went without. AGENTS.md states the rule
	// it was missing: a figure measured on one device is a figure about that
	// device, say which. The two rigs disagree even about a block neither
	// refused, an Ampeg SVT reading 12.7% of its energy in the mids on one and
	// 4.0% on the other, and the committed baseline of 155.27Hz at -58.55dB
	// sits thirty decibels from what the open loop settles at. That is almost
	// certainly because these came off the closed rig, and the file could not
	// say so.
	//
	// Empty in a library written before this was recorded, which is not the
	// same as a closed loop: it is a rig nobody wrote down.
	Rig  []string `json:"rig,omitempty"`
	Open bool     `json:"open,omitempty"`
	// PinnedOn is the output device the level was pinned on, which is the one
	// `Volume` reached.
	//
	// New with the setting that moves it, and the one that silently invalidates
	// everything else. A Mac with its default output on Bluetooth headphones
	// pins those while the jack feeding the pedal sits wherever it was left,
	// which is how a run came to read 66dB of loss with the cabling correct.
	// A library written in that state records its volume honestly and describes
	// a signal path nobody measured.
	PinnedOn string `json:"pinned_on,omitempty"`

	// Probed says the wire order was discovered on the device rather than read
	// out of the catalog.
	//
	// Worth recording, because the two are not equally trustworthy. A catalog
	// order is Line 6's own; a probed one was worked out by moving each index
	// and seeing which named parameter changed. That is the only thing that
	// knows for a block Line 6 ship no symbol list for, and every equaliser is
	// one of those.
	Probed bool `json:"probed,omitempty"`
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
	Noise map[audio.Figure]float64 `json:"noise"`
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
	Fits map[audio.Figure]Fit `json:"fits,omitempty"`
	// Spread is how far apart the settings sit, for a control that is a
	// list.
	//
	// A list has no slope. A cabinet's Mic is twelve microphones and the
	// fourth does not sit between the third and the fifth in any sense a line
	// describes, so "so much centroid per microphone" is a number with no
	// referent.
	Spread map[audio.Figure]float64 `json:"spread,omitempty"`
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
	figure audio.Figure,
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
		off := y - (slope*xs[i] + intercept)
		spread := y - mean
		miss += off * off
		about += spread * spread
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
	figure audio.Figure,
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

// Wander is how far a figure's readings sit from each other once the one
// furthest from the middle is thrown away.
//
// Apart with the worst outlier removed, and the same units, so the two are
// directly comparable. Apart is right for a list control, where the readings
// are settings and the whole range is the answer. It is wrong for readings
// that are all supposed to be the same, because it is a maximum minus a
// minimum and one disagreeing value sets both ends of it.
//
// That is the noise floor. Six takes of one untouched chain read the low band
// at 19.35 once and then 31.56 to 31.73 five times over, and Apart called the
// loop's wander twelve points of a band when five of the six readings agreed
// to three decimal places. The floor is the lower bound on every tolerance a
// solve runs against, so that one reading stretched the target a hundredfold
// and the solve reported arriving while it walked away.
//
// Fewer than three readings are returned as Apart does: with two there is no
// middle to be furthest from, and throwing one away would leave a spread of
// nothing and call it certainty.
func Wander(
	points []Point,
	figure audio.Figure,
) (float64, bool) {
	got := make([]float64, 0, len(points))

	for _, p := range points {
		if v, ok := p.figure(figure); ok {
			got = append(got, v)
		}
	}

	if len(got) == 0 {
		return 0, false
	}

	if len(got) < 3 {
		return spread(got), true
	}

	return spread(without(got, furthest(got))), true
}

// furthest is which reading sits furthest from the middle of them.
//
// The median rather than the mean, because the value being looked for is the
// one dragging the mean about.
func furthest(
	got []float64,
) int {
	middle := median(got)

	at, worst := 0, -1.0

	for i, v := range got {
		if off := math.Abs(v - middle); off > worst {
			at, worst = i, off
		}
	}

	return at
}

// median is the middle of some readings, sorted without disturbing the caller's
// order.
func median(
	got []float64,
) float64 {
	by := make([]float64, len(got))
	copy(by, got)
	sort.Float64s(by)

	half := len(by) / 2
	if len(by)%2 == 1 {
		return by[half]
	}

	return (by[half-1] + by[half]) / 2
}

// without is some readings with the one at an index left out.
func without(
	got []float64,
	at int,
) []float64 {
	out := make([]float64, 0, len(got)-1)
	out = append(out, got[:at]...)

	return append(out, got[at+1:]...)
}

// figure reads one of a point's figures by name.
//
// Two of them can be absent: a transient needs a note starting and a decay
// needs one ending, and a reading holding neither has no answer rather than
// an answer of zero.
func (p Figures) figure(
	name audio.Figure,
) (float64, bool) {
	switch name {
	case audio.KeyCentroid:
		return p.Centroid, true
	case audio.KeyLevel:
		return p.Level, true
	case audio.KeyLow:
		return p.Low, true
	case audio.KeyMid:
		return p.Mid, true
	case audio.KeyHigh:
		return p.High, true
	case audio.KeyDynamics:
		return held(p.Dynamics)
	case audio.KeyHarmonics:
		return held(p.Harmonics)
	case audio.KeyLean:
		return held(p.Lean)
	case audio.KeyTransient:
		return held(p.Transient)
	case audio.KeyDecay:
		return held(p.Decay)
	}

	return 0, false
}

// held reads a figure that a reading may not carry.
//
// The five figures that can be absent are absent in the same way, so they are
// answered in the same way: no value rather than a value of zero.
func held(
	of *float64,
) (float64, bool) {
	if of == nil {
		return 0, false
	}

	return *of, true
}

// Named is every figure a curve is reported in.
//
// The same ones a record is described in, so a block's curve and a record's
// reading are in one vocabulary. Level is not among a record's and is among
// these, because a record's loudness is the mastering engineer's and a
// block's is the block's.
func Named() []audio.Figure {
	return append(audio.MeasuredKeys(), audio.KeyLevel)
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

// Setting is a swept position, and what kind of control it belongs to.
//
// The kind travels with the value because a control is not always a number a
// document may write as one: of the swept drives, 163 controls are floats, 20
// are integers and 16 are booleans, and writing a float where the contract
// wants a boolean is refused. The Kinky Boost's `Bright` is one of the
// sixteen, and a preset built with 1.0 on it would not load.
type Setting struct {
	// Value is the position, as the sweep recorded it.
	Value float64
	// Kind is what the catalog calls that control's type: float, int or bool.
	Kind string
}

// atLevel is how far a swept position may sit from where its block settles
// and still be read as the same block, in decibels.
const atLevel = 3.0

// Toward is the value of each control that brings a block nearest a target.
//
// What a sweep is for. A fingerprint picks the block and this sets it: every
// position of every control was measured, so the value that moves a figure
// where somebody is aiming is a reading rather than a guess, and a block added
// to a chain no longer has to arrive at the factory default nobody chose.
//
// One control at a time, each against the target independently, and that is a
// real limitation rather than an oversight. A sweep moves one control and
// leaves the rest where they sat, so these answers do not compose: two
// controls that each reach the target alone may overshoot it together. See
// task 254, where every sweep being of a block alone is the wider version of
// the same problem. It is still the best answer available without putting the
// chain on hardware, and it beats a default by the width of the whole range.
//
// A control whose whole sweep moves a figure less than that figure wandered
// between repeat takes is left out. Setting one of those is spending a control
// on noise, and `Noise` is what the sweep recorded it as.
//
// A list control is included and answered the same way: its positions are
// numbers on the wire and the nearest reading wins, which is the only thing
// that can be said about a list nobody can interpolate across.
func (c Curves) Toward(
	want Figures,
	by Weights,
) map[string]Setting {
	out := map[string]Setting{}

	for name, curve := range c.Controls {
		best, at, found := 0.0, 0.0, false

		for _, p := range curve.Points {
			// Only positions that leave the block at working level.
			//
			// Matching figures cannot tell a quieter block from a differently
			// voiced one. The band shares are normalised and the noise floor
			// is not, so turning a block down moves its shares toward the
			// converters' and that reads as a close match to almost any
			// target. The first run of this pinned an equaliser's `Level` at
			// -60dB, which is 16dB below where that block settles and would
			// have silenced the chain the moment somebody stepped on it.
			//
			// `Settled` is the level the sweep recorded the block working at,
			// so the window is the sweep's own answer about this block rather
			// than a figure chosen here. The width is not: 3dB is the
			// smallest level difference worth calling one, and anything
			// further is a loudness change wearing a tone control's name.
			// Level-matching before comparing is the same rule task 222 put
			// on comparing two rigs by ear.
			if math.Abs(p.Level-curve.Settled) > atLevel {
				continue
			}

			d := by.between(p.Figures, want)
			if !found || d < best {
				best, at, found = d, p.Value, true
			}
		}

		if !found || !curve.moves(by) {
			continue
		}

		out[name] = Setting{Value: at, Kind: curve.Kind}
	}

	return out
}

// moves says whether a control changed any weighed figure by more than that
// figure's own noise.
//
// Read off the sweep rather than assumed. A control that moved nothing is one
// the readings cannot tell from the converters, and 59% of the fits in this
// library account for less than 0.8 of their figure's movement.
func (curve Curve) moves(
	by Weights,
) bool {
	for _, each := range []struct {
		key    audio.Figure
		weight float64
		of     func(Figures) float64
	}{
		{audio.KeyCentroid, by.Centroid, func(f Figures) float64 { return f.Centroid }},
		{audio.KeyLow, by.Low, func(f Figures) float64 { return f.Low }},
		{audio.KeyMid, by.Mid, func(f Figures) float64 { return f.Mid }},
		{audio.KeyHigh, by.High, func(f Figures) float64 { return f.High }},
	} {
		if each.weight == 0 || len(curve.Points) == 0 {
			continue
		}

		low, high := each.of(curve.Points[0].Figures), each.of(curve.Points[0].Figures)

		for _, p := range curve.Points {
			v := each.of(p.Figures)
			low, high = min(low, v), max(high, v)
		}

		if high-low > curve.Noise[each.key] {
			return true
		}
	}

	return false
}

// TowardFrom is the position of one control whose own deviation best matches a
// deviation asked for.
//
// The other way of reading a sweep, and the one a list needs. [Curves.Toward]
// compares each position's figures to a target directly, which works while
// both sides were measured the same way and fails where they were not: a
// genre's figures come off finished records and a block's off a dry signal
// through one block, so the nearest position is whichever end of the control
// is closest rather than a setting anybody would choose. Choosing a cabinet's
// microphone that way sent every one of eleven genres to the same microphone,
// and letting it choose the rest of the cabinet sent Distance to its maximum
// and LowCut to 79.9Hz, which on a bass cuts the fundamentals.
//
// So both sides become "how far from its own normal", which is the comparison
// the genre figures already earn their words from. Each position is read as
// its distance from the median across the control's own positions, and the
// target as the displacement it was given. A microphone 11Hz brighter than the
// others on this cabinet then answers a genre 20Hz brighter than the other
// genres, and a darker one answers a darker genre.
//
// The direction is what carries the information and the size is not
// comparable: a genre's centroid displacement runs to 126Hz where a
// microphone's runs to 12, because one is a population of records and the
// other is one box. So each axis is scaled by the spread of its own side
// before the two are compared, which leaves the sign and the relative
// magnitude and drops the units.
//
// Answers false for a control with fewer than two positions, or one whose
// positions all read alike, because neither offers a choice.
func (c Curves) TowardFrom(
	control string,
	want Figures,
	by Weights,
) (float64, bool) {
	curve, swept := c.Controls[control]
	if !swept || len(curve.Points) < 2 {
		return 0, false
	}

	axes := []struct {
		weight float64
		of     func(Figures) float64
		asked  float64
	}{
		{by.Centroid, func(f Figures) float64 { return f.Centroid }, want.Centroid},
		{by.Low, func(f Figures) float64 { return f.Low }, want.Low},
		{by.Mid, func(f Figures) float64 { return f.Mid }, want.Mid},
		{by.High, func(f Figures) float64 { return f.High }, want.High},
	}

	// Only the positions that leave the block at working level, for the reason
	// Toward states: a quieter block reads as a close match to anything.
	at := make([]Point, 0, len(curve.Points))

	for _, p := range curve.Points {
		if math.Abs(p.Level-curve.Settled) <= atLevel {
			at = append(at, p)
		}
	}

	if len(at) < 2 {
		return 0, false
	}

	best, found := 0.0, false
	score := math.Inf(1)

	// Each axis normalised by its own side's spread, so a sign and a
	// proportion survive and the units do not.
	for _, a := range axes {
		if a.weight == 0 {
			continue
		}

		mid := middle(at, a.of)

		wide := 0.0
		for _, p := range at {
			wide = max(wide, math.Abs(a.of(p.Figures)-mid))
		}

		if wide == 0 || a.asked == 0 {
			continue
		}

		for i, p := range at {
			// Positive where the position and the genre both lean the same
			// way, and the best score is the smallest difference between the
			// two leans.
			d := math.Abs((a.of(p.Figures)-mid)/wide - sign(a.asked))

			if !found || d < score {
				best, score, found = at[i].Value, d, true
			}
		}
	}

	return best, found
}

// middle is the median of one figure across the positions given.
func middle(
	at []Point,
	of func(Figures) float64,
) float64 {
	all := make([]float64, 0, len(at))
	for _, p := range at {
		all = append(all, of(p.Figures))
	}

	slices.Sort(all)

	if n := len(all); n%2 == 1 {
		return all[n/2]
	}

	return (all[len(all)/2-1] + all[len(all)/2]) / 2
}

// sign is which way a displacement leans, as one or minus one.
//
// The lean rather than the size, because the two sides are not comparable: a
// genre's centroid displacement runs to 126Hz across its records where a
// microphone's runs to 12 across one cabinet.
func sign(
	of float64,
) float64 {
	if of < 0 {
		return -1
	}

	return 1
}
