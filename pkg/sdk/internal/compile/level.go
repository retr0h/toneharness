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
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// This file answers the question a build never asked: how loud does this come
// out?
//
// Nothing did, and two rigs built from the same corpus arrive at different
// loudness as a result. The cabinet decides most of it. Across the 133 cabinets
// measured here the readings run -67.88dB to -56.56dB, a spread of 11.3dB, and
// a `2x15 Brute` at -59.62dB against an `8x10 Ampeg SVT-E` at -56.76dB puts two
// chains 2.9dB apart with every other level control identical. That was found
// by playing them: one read as quiet, and the quieter of two presets loses an
// A/B almost regardless of its tone.
//
// So a Setup may state the level it wants and this lands on it, by moving the
// amplifier's own level control. Stated rather than worked out, because there
// is no right figure: normalising to the median of the cabinets would make the
// 8x10 chain quieter rather than the 2x15 chain louder, and the 8x10 is the
// unusual one. The target belongs to the person.
//
// **Two readings, each compared only against its own run.** The chain's level
// comes from the block library and the control's worth comes from that
// amplifier's own sweep, and the two are never subtracted from each other: a
// fingerprint and a sweep are separate runs at separate output levels, 18dB
// apart for the one block measured both ways. What is taken from the library is
// the difference between two cabinets, and what is taken from the sweep is the
// difference between two positions of one control. Both are differences inside
// one run.

// audible is the smallest level difference worth moving a control for.
//
// One decibel. Below that the move is smaller than anybody reliably hears on
// a broadband signal, and the whole point of landing on a level is an A/B
// somebody can trust: a build that reports moving a control by a third of a
// decibel has spent somebody's attention on nothing.
const audible = 1.0

// levelControls are the controls an amplifier offers for its own output, in the
// order they are preferred.
//
// A channel volume first, because it is the one a player reaches for and the
// one that sits after the amplifier's own gain staging, so moving it changes
// the level without changing how hard the valves are driven. Master second.
// `Level` last, and only because a few models call it that.
//
// Drive is not here and must not be. It is a level control in the sense that
// it changes loudness, and the whole of #239 was a solve spending it: the sweep
// says Drive 0.46 reads -15.2dB and 0.60 reads -16.1dB on one amplifier, so
// turning it down is marginally louder, and anybody reasoning about it from the
// name gets the sign wrong.
var levelControls = []string{"ChVol", "Master", "Level"}

// Level moves the amplifier's level control so the chain lands where a Setup
// asked.
//
// Reports what it did either way, including every way it did nothing, because
// a build that silently failed to hit a level is a build somebody compares
// against another one and draws a conclusion from.
func Level(
	built plan.Plan,
	cat *catalog.Catalog,
	want float64,
	lib measured.Library,
	curves map[catalog.ModelID]measured.Curves,
) (plan.Plan, result.Levelled) {
	out := result.Levelled{Wanted: want}

	was, ok := expected(built, cat, lib)
	if !ok {
		out.Because = "nothing in the chain has a measured level"

		return built, out
	}

	out.Was, out.Reached = was, was

	gap := want - was
	if gap < audible && gap > -audible {
		out.Because = fmt.Sprintf(
			"it already comes out within %.0fdB of that", audible)

		return built, out
	}

	at, name, ok := levelAt(built, cat)
	if !ok {
		out.Because = "no amplifier in the chain has a level control"

		return built, out
	}

	curve, swept := curves[built.Blocks[at].Model]
	if !swept {
		out.Because = fmt.Sprintf(
			"%s has no committed sweep, so what a turn of its %s is worth is "+
				"unmeasured", built.Blocks[at].Model, name)

		return built, out
	}

	from, _ := built.Blocks[at].Params[name].Float()

	to, moved, ok := landsOn(curve.Controls[name], from, gap)
	if !ok {
		out.Because = fmt.Sprintf(
			"%s's own sweep does not reach %+.1fdB from where it sits", name, gap)

		return built, out
	}

	built.Blocks[at].Params[name] = catalog.Float(to)
	out.Param, out.From, out.To = name, from, to
	out.Reached = was + moved

	return built, out
}

// expected is the level the chain's own gear leaves it at.
//
// The cabinet's reading, because the cabinet is what decides it and the
// evidence is that it decides nearly all of it: the one case this was found on
// had every other level control identical between two chains and the two
// cabinets accounted for the whole 2.9dB.
//
// A chain with no cabinet has no answer here rather than a guess. Every figure
// this project measured came through one, so a chain without one is a chain
// nothing has a reading for.
func expected(
	built plan.Plan,
	cat *catalog.Catalog,
	lib measured.Library,
) (float64, bool) {
	for _, b := range built.Blocks {
		one, known := cat.Block(b.Model)
		if !known || one.Category != catalog.CategoryCab {
			continue
		}

		read, measured := lib.Blocks[string(b.Model)]
		if !measured || !read.Measured() {
			continue
		}

		return read.Level, true
	}

	return 0, false
}

// levelAt finds the amplifier's level control, and where in the chain it is.
func levelAt(
	built plan.Plan,
	cat *catalog.Catalog,
) (int, string, bool) {
	for i, b := range built.Blocks {
		one, known := cat.Block(b.Model)
		if !known || one.Category != catalog.CategoryAmp {
			continue
		}

		for _, name := range levelControls {
			if _, carried := one.Params[name]; carried {
				return i, name, true
			}
		}
	}

	return 0, "", false
}

// nearest is the position of a control that moves the level closest to a gap,
// and how far it actually moves it.
//
// Read off the curve's own positions rather than from a slope. A slope would
// have to be extrapolated past the positions that were measured, and the five
// amplifiers swept here disagree about what a turn of the same control is worth
// by a factor of two: 19.8dB to 42.3dB across ChVol. A position that was
// measured is a position the device was actually set to.
//
// Both levels come off the same curve, so their difference is a difference
// inside one run. Nothing here compares a swept level against a library one.
func landsOn(
	c measured.Curve,
	from float64,
	gap float64,
) (float64, float64, bool) {
	sits, ok := levelOf(c, from)
	if !ok {
		return 0, 0, false
	}

	best, moved := 0.0, 0.0
	found := false

	for _, p := range c.Points {
		if !usableFor(c, p) {
			continue
		}

		got := p.Level - sits
		if !found || nearerThan(got, moved, gap) {
			best, moved, found = p.Value, got, true
		}
	}

	// A position that lands no nearer the target than staying put is not a
	// move, it is a control turned for the report's benefit.
	if !found || off(moved, gap) >= off(0, gap) {
		return 0, 0, false
	}

	return best, moved, true
}

// levelOf is what the curve read at the position a control currently sits at.
//
// The nearest measured position, because a build sets a control from corpus
// medians and the sweep stepped through its own range: the two do not land on
// the same numbers.
func levelOf(
	c measured.Curve,
	at float64,
) (float64, bool) {
	level := 0.0
	best := 0.0
	found := false

	for _, p := range c.Points {
		if !usableFor(c, p) {
			continue
		}

		if d := off(p.Value, at); !found || d < best {
			level, best, found = p.Level, d, true
		}
	}

	return level, found
}

// usableFor reports whether a swept position is a reading of the block.
//
// The same three exclusions `measured.StartAt` applies, and for the same
// reasons: a position at the converters' ceiling reads the converters, one
// below the curve's own floor reads hiss, and a flagged one was already known
// to be either.
func usableFor(
	c measured.Curve,
	p measured.Point,
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

// nearCeiling is the level above which a reading is the converters' rather
// than the block's.
//
// Two decibels below full scale, which is what `measured` uses on a
// fingerprint. Repeated as a constant rather than exported from there because
// it is the same judgement rather than the same number: a change to one is not
// automatically right for the other.
const nearCeiling = -2.0

// nearerThan reports whether one move lands nearer a gap than another.
func nearerThan(
	a float64,
	b float64,
	gap float64,
) bool {
	return off(a, gap) < off(b, gap)
}

// off is the distance between two figures.
func off(
	a float64,
	b float64,
) float64 {
	if d := a - b; d < 0 {
		return -d
	}

	return a - b
}
