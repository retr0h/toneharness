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
	"strings"

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
	// Idle is how loud this signal path reads carrying nothing, measured once
	// per sweep by pushing silence of the reference's own length through it.
	//
	// The field that says the readings are of something. Every other figure
	// here describes what was asked for, so all of them agree whether or not
	// anything arrived: a sweep taken with nothing plugged into the pedal's
	// input wrote 24,854 bytes with `volume: 38`, `open: true`, a matching
	// reference sha256 and a byte-identical preset, and the five controls
	// settled within 0.03dB of each other so the mid-sweep guard passed every
	// one. Only the absolute level was wrong, by 77dB, and nothing in the file
	// had anything to compare it against.
	//
	// This is that comparison, and it is the one reading that does not move
	// with the rig. The converters' floor on the machine these were taken on
	// is -108.5dB and somebody else's is their own, so what matters is not how
	// loud the answer is but how far it sits above the path at rest. Live that
	// is 50 to 60dB; dead it is nothing.
	//
	// Zero in a library written before this was recorded, which is a sweep
	// nobody checked rather than a path with no noise.
	Idle float64 `json:"idle,omitempty"`

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

// Carries is how far a slope read off this control may be taken, as a fraction
// of the control's own range.
//
// Straightness, which is the question exactly. `Straight` is the fraction of a
// figure's movement one line accounts for, so a control reading 0.98 is a line
// and a slope read anywhere on it is true everywhere, while one reading 0.00
// has a slope that is true only where it was read.
//
// **The controls a least-squares solve models best are the ones that reach a
// figure by removing signal, and the controls that matter musically are the
// ones it models worst.** Across this library the filters are nearly perfect
// lines: a drive's MinFreq reads 0.99 straight over 2,399Hz and a cabinet's
// LowCut 0.97 over 235Hz. The voicing controls collapse: an equaliser's Mode
// reads 0.00 over 1,463Hz, a filter's Start 0.00 over 1,436Hz, an amplifier's
// Ch1 Mid Freq 0.02 over 429Hz.
//
// So a slope is worth a distance rather than a yes or a no. Leaving each
// reading out of the library and predicting it, one least-squares line over
// the whole range misses by a median 13.3% of the figure's own movement across
// 28,472 readings, and a slope read locally is worse the further it is taken:
// the two-point derivative at the middle of a curve is a median 64% wrong one
// step past where it was read, 77% two steps on and 86% three.
//
// **`figures` is the axes being solved, and naming them is not optional in
// practice.** The worst of them decides, which is the choice [Reach] makes for
// the same reason: a solve puts every axis it is given into one matrix, so a
// control that is a line for the centroid and a hairpin for the mid band is
// not a control whose slope carries. Which axes those are changes the answer
// more than anything else here. Over the 1,001 committed float curves, the
// worst of the centroid and the three band shares runs to a median of 0.377
// and 31% of curves read under 0.1; add the dynamics, the harmonics and the
// lean a corpus also states and it is a median of 0.058 with 59% under 0.1;
// take every figure in the file, which includes a decay and a transient that
// mostly wandered inside their own noise, and it is 0.019 with 77% under 0.1.
// A target naming harmonics really is asking for a figure almost nothing moves
// in a straight line, and the tight cap is the finding rather than a fault.
//
// No figures is every fit, which is the right answer for a caller asking about
// the control rather than about a target.
//
// False for a control with no fit for any figure named, which is every list. A
// list has no order, so a line through it is meaningless by construction and
// there is no distance to carry one over. Those are compared rather than
// solved, which is what `measure controls` already does with a cabinet's
// twelve microphones.
func (curve Curve) Carries(
	figures ...audio.Figure,
) (float64, bool) {
	out, found := 1.0, false

	for key, fit := range curve.Fits {
		if len(figures) > 0 && !slices.Contains(figures, key) {
			continue
		}

		out, found = math.Min(out, fit.Straight), true
	}

	if !found {
		return 0, false
	}

	// Clamped, because `Straight` is 1 minus a ratio of sums of squares and
	// goes negative wherever the line misses by more than the readings' own
	// spread: a fraction of a range is the unit here, and a negative one is
	// not a shorter distance, it is no distance.
	return math.Min(math.Max(out, 0), 1), true
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

// Toward is the value of each control that leans a block the way a target asks.
//
// What a sweep is for. A fingerprint picks the block and this sets it: every
// position of every control was measured, so the value that moves a figure
// where somebody is aiming is a reading rather than a guess, and a block added
// to a chain no longer has to arrive at the factory default nobody chose.
//
// **Displacements, not positions, and that is task 254's correction.** This
// used to score each swept position's own figures against `want` directly,
// which is only sound while both sides came out of the same run. They do not.
// `want` is built over [Library.Baseline], which is the empty loop measured
// during the fingerprint run, and the points come from this block's own sweep.
// Across the 178 blocks that appear in both, the same block read in the two
// runs differs by a median 20.6Hz of centroid, 5.5dB of level and 4.59 points
// of the low band, where punk's own centroid displacement is 9.6Hz and the
// eleven aimable genres' band shares displace 0.2 to 18.9 points. The offset
// between the runs was larger than the signal being aimed at. [StartAt]'s
// header states the same constraint from the other end: a Deluxe Comp reads
// -49.56dB in the fingerprints and -31.34dB across its own sweep.
//
// So each control is answered by [Curves.TowardFrom], which reads a position
// as its distance from the median across that control's own positions and the
// target as the lean it was given. Both sides become "how far from its own
// normal", the comparison a genre's words are already earned by, and nothing
// is subtracted across two runs.
//
// One control at a time, each against the target independently, and that is a
// real limitation rather than an oversight. A sweep moves one control and
// leaves the rest where they sat, so these answers do not compose: two
// controls that each lean the right way alone may overshoot together.
//
// **An isolated amplifier is refused, which [StartAt] has always done.** A
// sweep of an amplifier alone has no cabinet in front of it and so no speaker
// rolloff, and what comes back is the buzz a cabinet exists to absorb: 43 of
// the 121 usable curves hold a position reading more than half its energy
// above 2kHz and all 43 are amplifiers. It is not a difference of degree.
// [Reach.Alone] records the size of it: an amplifier swept alone moved the
// centroid 12,763Hz per turn of Treble where the same control in a chain moves
// it 3,250, its Mid moved the centroid the other way entirely, and **four of
// eleven controls changed sign.** A displacement keeps the sign and drops the
// units, so a reading whose sign is wrong a third of the time is not rescued
// by reading it as a lean. An amplifier's controls stay where the corpus
// medians put them until a sweep exists that was taken through a cabinet,
// which is task 271.
//
// Nothing else is refused, and that is the evidence rather than a line drawn
// for tidiness: the inverted curves are all amplifiers, and no drive,
// compressor or equaliser swept alone reads upside down.
//
// A list control is included and answered the same way, which is the reading
// [Curves.TowardFrom] was written for: its positions are numbers on the wire
// with no order to run along, so a lean between twelve measured sounds is the
// only thing that can be said about one.
func (c Curves) Toward(
	want Figures,
	by Weights,
) map[string]Setting {
	if c.Isolated && c.amplifier() {
		return nil
	}

	out := map[string]Setting{}
	asked := c.shared(want, by)

	for name, curve := range c.Controls {
		at, found := c.TowardFrom(name, asked, by)
		if !found || !curve.moves(by) {
			continue
		}

		out[name] = Setting{Value: at, Kind: curve.Kind}
	}

	return out
}

// Asks reports whether a displacement asks for anything on an axis these
// weights carry.
//
// Separate from the controls, because "nothing on this block reaches what was
// asked" and "nothing was asked" are two different answers and one of them is
// not about the block at all. Three of the eleven genres this corpus can aim at
// displace their midrange by zero, and hard-rock displaces the centroid by zero
// too, so for a bass, where [SpectralFor] drops the low and high bands, it asks
// for nothing whatsoever. A report saying no control reached it describes a
// block that was never asked.
func (p Figures) Asks(
	by Weights,
) bool {
	for _, each := range [][2]float64{
		{by.Centroid, p.Centroid},
		{by.Low, p.Low},
		{by.Mid, p.Mid},
		{by.High, p.High},
	} {
		if each[0] != 0 && each[1] != 0 {
			return true
		}
	}

	// And harmonics, where the displacement carries a reading of it at all.
	if by.Harmonics != 0 && p.Harmonics != nil && *p.Harmonics != 0 {
		return true
	}

	return false
}

// shared is the target split among the dials that can each answer it.
//
// The answer to the limitation this file already named: a sweep moves one
// control and leaves the rest where they sat, so asking every dial for the
// whole displacement asks for one thing as many times as there are dials. On
// a ten-band equaliser that is ten times, and nine of its bands move a bass's
// midrange past their own noise, so each one was separately told to deliver
// the genre's whole displacement and each separately went as far as it could.
//
// Divided by how many can answer, which is the apportionment a sweep supports
// and no more. The sweeps say what each control does alone and nothing about
// what two of them do together, so a share each is the one split that does not
// claim to know how they interact. A real one is least squares against
// readings taken in the chain, which is [solve] and needs the device.
//
// Dials only. A list or a switch is chosen by direction rather than by size,
// so dividing the ask would change nothing about which option it reaches and
// would quietly take a microphone out of the count that decides a band's share.
//
// A figure one dial alone can move is not divided, which is the common case
// and the one every drive on this device is in.
func (c Curves) shared(
	want Figures,
	by Weights,
) Figures {
	for _, each := range []struct {
		key    audio.Figure
		weight float64
		onto   *float64
	}{
		{audio.KeyCentroid, by.Centroid, &want.Centroid},
		{audio.KeyLow, by.Low, &want.Low},
		{audio.KeyMid, by.Mid, &want.Mid},
		{audio.KeyHigh, by.High, &want.High},
	} {
		if each.weight == 0 {
			continue
		}

		if among := c.dials(each.key); among > 1 {
			*each.onto /= float64(among)
		}
	}

	// Harmonics, which is a pointer on a reading and so cannot join the table
	// above. Replaced rather than divided in place, because the caller's own
	// figures point at the same number.
	if by.Harmonics != 0 && want.Harmonics != nil {
		if among := c.dials(audio.KeyHarmonics); among > 1 {
			was := *want.Harmonics / float64(among)
			want.Harmonics = &was
		}
	}

	return want
}

// dials is how many of a block's dials move one figure further than that
// figure's own noise.
//
// Noise rather than any movement, for the reason [Curve.moves] states: a move
// smaller than the figure wandered across repeat takes with nothing touched
// moved nothing anybody can demonstrate, and counting one would give a real
// dial a smaller share on the strength of a reading that says nothing.
func (c Curves) dials(
	key audio.Figure,
) int {
	among := 0

	for _, curve := range c.Controls {
		if curve.Kind == "int" || curve.Kind == "bool" || len(curve.Points) == 0 {
			continue
		}

		low := curve.Points[0].of(key)
		high := low

		for _, p := range curve.Points {
			v := p.of(key)
			low, high = min(low, v), max(high, v)
		}

		if high-low > curve.Noise[key] {
			among++
		}
	}

	return among
}

// amplifier says whether these readings are of an amplifier.
//
// From the model identifier, because a sweep records which block it swept and
// not which category the catalog files it under. Every amplifier on this
// device is `HD2_Amp...` and every preamp `HD2_Preamp...`; a preamp is the same
// hazard for the same reason, being the front half of an amplifier with no
// cabinet after it.
//
// By prefix rather than by asking the catalog, so this package needs no
// catalog: it reads measurements and a dependency on the model list to answer
// a question about its own files would be the wrong way round.
func (c Curves) amplifier() bool {
	return strings.HasPrefix(c.Block, "HD2_Amp") ||
		strings.HasPrefix(c.Block, "HD2_Preamp")
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
// reached is one axis as a control's own travel sees it: where the middle of
// its positions sits, how far it reaches either side of that, and what
// fraction of the reach in the direction asked the target is asking for.
//
// Both sides, because a control is rarely symmetric about its middle. One
// equaliser band on this device adds 17.5 points of a bass's midrange and
// takes away 2.3, so a single spread is the boost side and every cut in the
// sweep compresses into a tenth of the scale. Asked to cut, the whole cut side
// then reads as too little and the deepest one wins, which is how two bands
// stayed at 15dB after the size of the ask was honoured.
type reached struct {
	weight   float64
	key      audio.Figure
	middle   float64
	up, down float64
	asked    float64
}

// lean is how far one position sits from the middle, as a fraction of the
// reach on that side. Between -1 and 1 by construction.
func (r reached) lean(
	at Point,
) float64 {
	by := at.of(r.key) - r.middle

	switch {
	case by > 0 && r.up > 0:
		return by / r.up
	case by < 0 && r.down > 0:
		return by / r.down
	default:
		return 0
	}
}

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
		key    audio.Figure
		asked  float64
	}{
		{by.Centroid, audio.KeyCentroid, want.Centroid},
		{by.Low, audio.KeyLow, want.Low},
		{by.Mid, audio.KeyMid, want.Mid},
		{by.High, audio.KeyHigh, want.High},
		// Harmonics, where the displacement carries a reading of it. Task
		// 276: a share of the energy above the fundamental is the fourth
		// figure both sides of a comparison measure the same way.
		{by.Harmonics, audio.KeyHarmonics, orZero(want.Harmonics)},
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

	reach := make([]reached, 0, len(axes))

	// Whether the size of the ask means anything to this control, which is the
	// distinction [solve.Choice] and [solve.Knob] are built on and the reason
	// one answer here would be wrong for half the callers.
	//
	// A list or a switch is a choice among options. "Brighter than the other
	// genres" is answered by the brightest microphone this cabinet offers,
	// however much brighter that is, because there is no twelfth-of-a-mic to
	// ask for: the sizes are not comparable either, 126Hz across a population
	// of records against 22Hz across one box. Comparing by direction is what
	// stopped all eleven genres arriving on one microphone.
	//
	// A dial is a quantity, and there the size is the whole of it. Asked for
	// two points of midrange, a band that moves it by twenty should move a
	// tenth of the way and not go to its stop.
	listed := curve.Kind == "int" || curve.Kind == "bool"

	for _, a := range axes {
		if a.weight == 0 || a.asked == 0 {
			continue
		}

		was := middle(at, a.key)

		up, down := 0.0, 0.0

		for _, p := range at {
			off := p.of(a.key) - was
			up, down = max(up, off), max(down, -off)
		}

		// How far this control reaches the way it is being asked to go.
		reaches := up
		if a.asked < 0 {
			reaches = down
		}

		if reaches == 0 {
			continue
		}

		asked := sign(a.asked)

		if !listed {
			// An ask a dial cannot reach is not an ask for that dial.
			//
			// What a stop used to mean, and the reason a ten-band equaliser
			// came out with six of its bands against one: every axis read as
			// "as far that way as you go". Its 8kHz band moves a bass's
			// midrange by 0.049 of a point and its 16kHz band by 0.001,
			// against genres asking for two to twenty-two, and both sat at
			// 15dB delivering nothing.
			//
			// Declining is the honest answer rather than the extreme. A dial
			// with no axis it can reach is left where the catalog put it,
			// which for an equaliser band is flat, and a band that cannot
			// deliver what is asked has nothing to say about it either way.
			if math.Abs(a.asked) > reaches {
				continue
			}

			asked = a.asked / reaches
		}

		reach = append(reach, reached{
			weight: a.weight,
			key:    a.key,
			middle: was,
			up:     up,
			down:   down,
			asked:  asked,
		})
	}

	if len(reach) == 0 {
		return 0, false
	}

	best, found := 0.0, false
	score := math.Inf(1)

	// Weighed across every axis at once rather than the best single pairing of
	// an axis and a position.
	//
	// The weights are weights here and were a mask before: the axis with the
	// smallest difference won outright whatever it counted for, so on a bass,
	// where [SpectralFor] counts the midrange at 1 and the centroid at 0.001,
	// a position chosen for half a hertz of centroid outvoted one chosen for
	// the midrange. Summing is what [Weights.between] already does for
	// ranking blocks, and for the same reason.
	for i, p := range at {
		d := 0.0

		for _, a := range reach {
			d += a.weight * math.Abs(a.lean(p)-a.asked)
		}

		if !found || d < score {
			best, score, found = at[i].Value, d, true
		}
	}

	return best, found
}

// middle is the median of one figure across the positions given.
func middle(
	at []Point,
	key audio.Figure,
) float64 {
	all := make([]float64, 0, len(at))
	for _, p := range at {
		all = append(all, p.of(key))
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
