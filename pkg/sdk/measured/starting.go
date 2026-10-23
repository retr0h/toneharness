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
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// This file answers a different question from the shortlist's. The shortlist
// says which block to add; this says where to start one of its controls.
//
// A block proposed with nothing said about its controls arrives at whatever the
// catalog's defaults are, and the defaults are not a position chosen for the
// job the block was picked for. The sweeps are the only evidence that could
// choose one: a fingerprint is the block at its defaults and says nothing about
// its controls, while `<model>.json` is every control moved across its range
// with the reading at each position.
//
// **Everything here is scored inside one sweep, and that is the constraint that
// shaped it.** The obvious design is to score a swept position against the
// target the way the shortlist scores a block, so that the two are comparable.
// They are not comparable. A fingerprint and a sweep are separate runs at
// separate output levels: a Deluxe Comp reads -49.56dB in `fingerprints.json`,
// taken at volume 38, and -31.34dB across its own sweep, which records no
// volume at all. Eighteen decibels apart, for the same block in the same
// isolated chain. Subtracting one from the other produces a number about the
// two runs rather than about the block.
//
// So a position is scored against the other positions of its own curve, which
// is one run, and the unit is that curve's own noise. What comes out is a
// direction and a confidence rather than a distance to the target: "LowGain at
// its bottom stop takes 359 noise-widths of energy out of the low band". That
// is weaker than a figure in tolerances and it is what the readings support.
//
// One control, not every control. Each curve was swept with that control alone
// moving, so two positions chosen from two curves are two independent estimates
// and the chain does not add them up. The tune is what settles the rest, from
// where this starts it.
//
// **And no amplifier.** Every sweep is of one block alone, which the file says
// in `isolated`, and for most kinds of block that is a caveat. For an
// amplifier it decides the readings: nothing is lowpassing it, so what comes
// back is the buzz a cabinet exists to absorb. A MESA Bass 400+ swept this way
// goes from 96.76% of its energy in the low band to 94.85% in the high across
// one step of its Master, at levels nowhere near the converters' ceiling, and
// every one of the six swept amplifiers does the same on nearly every control:
// 43 of the 121 usable curves hold a position reading more than half its energy
// high, and all 43 are amplifiers.
//
// Those are not wrong readings. They are readings of a chain that ends in an
// amplifier, and the chain a proposed block joins ends in a cabinet, so a
// position chosen because it moved energy upward was chosen for a chain that
// will not exist. `Straight` does not separate them, either: the inverted
// curves run from 0.012 to 0.98.
//
// So an amplifier's controls are left at their defaults and the caller says so.
// Re-measuring them behind a cabinet is what would change that.

// Starting is a control and the position a sweep says to start it at.
type Starting struct {
	// Control is the catalog's name for it, which is what a document writes.
	Control string
	// Value is the position, in the control's own units.
	Value float64
	// Figure is the axis this position was chosen for, and Noises how far it
	// moves that axis from where the curve sits, in multiples of the curve's
	// own noise for it.
	//
	// In noise-widths because that is the only unit the sweep can answer in,
	// and because it is the unit that says whether the move is real: a figure
	// that moved less than the noise moved nothing anybody can demonstrate.
	Figure audio.Figure
	Noises float64
}

// convincing is how many noise-widths a position has to move a wanted axis
// before it is worth writing down.
//
// Three, not one. One noise-width is a move the size of the measurement's own
// wobble and the sweeps are full of them: every control of a Deluxe Comp moves
// something by more than its noise, including a Knee that moves three figures
// by barely twice it. Writing a control on that evidence puts a number in a
// document that reads as a decision and is a coin toss.
//
// Three is also comfortably reached by the moves that matter. A Simple EQ's
// LowGain moves the low band 359 noise-widths, a Cali Q's 80Hz band moves the
// centroid 375.
const convincing = 3

// StartAt is the one control a block's own sweep says to move, and where to.
//
// `want` is the axes the chain could not reach, which decide the direction a
// position is scored for. `keep` is the axes it already holds, which a position
// may not disturb.
//
// The valve's guts stay out, from [catalog.SetByHand] rather than from a set
// the caller hands over. A document written with `Sag` moved to close a band
// share is the same decision a solve may not take, one step earlier, and an
// Ampeg's sweep answers a request for harmonics with `Sag` at its top stop, so
// without this it would be written down. Read here rather than passed in
// because a caller that forgets to pass it gets no error and the wrong
// document.
//
// The second is not decoration. A Deluxe Comp asked for less low energy answers
// with Level at +12, which does take low energy out and carries the chain's
// level 13.7dB with it; level is not in `want` precisely because it was already
// where it should be, so nothing else here would notice. That is a solve
// spending the level to buy the tone, in a new place.
//
// False when no sweep, no control or no position moves a wanted axis
// convincingly, which is the common answer twice over: 19 of the device's 661
// blocks have been swept at all, and a swept one may still have nothing to say
// about the axis in question. A caller that gets false should say the block
// arrives at its defaults rather than imply a choice was made.
func StartAt(
	c Curves,
	of catalog.Category,
	want []Want,
	keep []Want,
) (Starting, bool) {
	if c.Isolated && of == catalog.CategoryAmp {
		return Starting{}, false
	}

	var out Starting

	found := false

	// Sorted, so two controls that score alike resolve the same way every run
	// rather than in whatever order the map was walked.
	names := make([]string, 0, len(c.Controls))
	for name := range c.Controls {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		if catalog.SetByHand(name) {
			continue
		}

		curve := c.Controls[name]

		if !swept(curve) {
			continue
		}

		sits, ok := sitsAt(curve, want)
		if !ok {
			continue
		}

		for _, p := range curve.Points {
			if !usable(curve, p) || !holds(curve, p, keep) {
				continue
			}

			got, axis, ok := moves(curve, p, sits, want)
			if !ok || got <= out.Noises {
				continue
			}

			out = Starting{
				Control: name, Value: p.Value, Figure: axis, Noises: got,
			}
			found = got >= convincing
		}
	}

	if !found {
		return Starting{}, false
	}

	return out, true
}

// swept reports whether a curve is a reading of the block rather than of the
// converters.
//
// Where it settles says it. A curve whose working level sits at or above the
// ceiling was swept through an output turned up too far, so every position on
// it is shaped by the converters and the figures move for that reason. An
// `HD2_AmpWoodyBlue` Master settles at +0.46dB and reports its centroid leaping
// from 85Hz to 2,453Hz with 43.77% of the energy arriving in the high band,
// which a 1950s bass amplifier does not do. All seven of that block's curves
// settle above the line and so does a Mutant Filter's Mix: 8 of the 129
// committed curves, and the other 121 are unaffected.
//
// Kept in the file rather than deleted from it, and refused here, which is the
// same treatment a clipped fingerprint gets.
func swept(
	c Curve,
) bool {
	return c.Settled <= nearCeiling
}

// sitsAt is where a curve sits on each wanted axis, which positions are
// measured from.
//
// The median of the usable positions rather than the control's default, because
// a sweep does not record which of its positions is the default. The median is
// the one that does not move with an outlier at either stop, which matters for
// a control whose range is lopsided: this amplifier's Master leaps an order of
// magnitude over its first step and then falls steadily.
//
// False where no position is usable at all, which is a curve that says nothing
// rather than one that says zero.
func sitsAt(
	c Curve,
	want []Want,
) (map[audio.Figure]float64, bool) {
	out := map[audio.Figure]float64{}

	for _, w := range want {
		var at []float64

		for _, p := range c.Points {
			if !usable(c, p) {
				continue
			}

			if got, has := p.figure(w.Figure); has {
				at = append(at, got)
			}
		}

		if len(at) == 0 {
			continue
		}

		out[w.Figure] = median(at)
	}

	return out, len(out) > 0
}

// moves is how far a position carries a wanted axis the right way, in the
// curve's own noise, and which axis that was.
//
// The best single axis rather than the sum, the same choice `suggest` makes:
// summing would rank a position that is slightly right everywhere above one
// that is right on the axis the target actually cares about. A position that
// moves an axis the wrong way scores nothing for it rather than scoring
// negative, because the ranking is of what a position offers and the axes it
// does not help are the solver's problem.
func moves(
	c Curve,
	p Point,
	sits map[audio.Figure]float64,
	want []Want,
) (float64, audio.Figure, bool) {
	best := 0.0

	var axis audio.Figure

	ok := false

	for _, w := range want {
		was, had := sits[w.Figure]
		if !had {
			continue
		}

		now, has := p.figure(w.Figure)
		if !has {
			continue
		}

		floor := c.Noise[w.Figure]
		if floor <= 0 {
			continue
		}

		moved := now - was
		if moved == 0 || (moved > 0) != (w.Need > 0) {
			continue
		}

		if moved < 0 {
			moved = -moved
		}

		ok = true

		if got := moved / floor; got > best {
			best = got
			axis = w.Figure
		}
	}

	return best, axis, ok && axis != ""
}

// holds reports whether a position leaves every satisfied axis satisfied.
//
// Against the level the curve settles at, which is the only reference inside
// the sweep that says where the chain sits when it is working. The alternative
// is the block's fingerprint, and that is a different run at a different output
// level: the comparison would be of the two runs rather than of the control.
//
// Only an axis the curve carries a settled reading for, which is level. The
// other figures have no within-sweep reference to hold them to, so a kept axis
// that is not level passes here and is the solver's to protect. Said rather
// than quietly dropped, because it is a real gap: this guard stops a position
// buying tone with level and nothing stops one buying level with tone.
func holds(
	c Curve,
	p Point,
	keep []Want,
) bool {
	for _, w := range keep {
		if w.Figure != audio.KeyLevel {
			continue
		}

		if moved := p.Level - c.Settled; moved > w.Tol || moved < -w.Tol {
			return false
		}
	}

	return true
}

// usable reports whether a swept position is a reading of the block.
//
// Three ways it is not, and the second is the one that cost something. A
// position that muted the chain is already absent from Points, and one the
// sweep flagged as clipped is a reading of flat tops. The third is a position
// the sweep did not flag and should have: a Deluxe Comp's Level at its top stop
// reads -1.14dB, two decibels from full scale, with its low band collapsing
// from 96.75% to 0.73% and its centroid leaping from 116Hz to 1,438Hz. Its
// `clippedAt` is empty. A compressor does not do that; the converters do, and
// picking the position that moves the centroid furthest would have picked
// exactly that one.
//
// So the ceiling is measured rather than taken from the flag, which is the rule
// `Block.Measured` already applies to a fingerprint for the same reason.
//
// And the floor: a position quiet enough that the figures describe hiss. The
// curve records where that is, so there is no threshold to pick here.
func usable(
	c Curve,
	p Point,
) bool {
	if p.Level > nearCeiling || p.Level < c.SilentBelow {
		return false
	}

	for _, at := range c.ClippedAt {
		if at == p.Value {
			return false
		}
	}

	for _, at := range c.MutedAt {
		if at == p.Value {
			return false
		}
	}

	return true
}
