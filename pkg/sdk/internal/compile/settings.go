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
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// agreementThreshold is how tightly players must agree before the corpus
// overrules the catalog, as a share of a parameter's own range.
//
// Below it, the middle of what people do is a better answer than the factory
// default — Line 6 states 0.77 for the Ampeg SVT bright channel's Treble and
// the corpus median is 0.85. Above it there is no consensus to adopt, only an average
// of disagreement, and the catalog's default is the honest answer until a
// rig or a person says otherwise.
const agreementThreshold = 0.15

// settings decides what every knob on a block is set to.
//
// The catalog's default is the floor: Line 6 states one for every parameter
// and it is never invalid. The corpus can raise on it, but only where players
// agree closely enough that the median means something.
func settings(
	b catalog.Block,
	cat *catalog.Catalog,
	stats *corpus.Stats,
	// instrument is which half of the corpus a pooled figure may be drawn
	// from. Empty pools nothing, which leaves the catalog's defaults.
	instrument string,
) plan.Params {
	out := make(plan.Params, len(b.Params))

	for key, p := range b.Params {
		// A parameter with no stated default has no kind, and writing a value
		// with no kind produces a preset the device rejects.
		if p.Default.Type() == "" {
			continue
		}

		out[key] = p.Default

		if v, ok := agreed(b, key, p, cat, stats, instrument); ok {
			out[key] = v
		}
	}

	return out
}

// thin is the fewest presets a single model's median is worth following on.
//
// Below this the median is one or two people's taste and the spread test
// cannot tell a tight cluster of two from agreement, because two values that
// happen to sit together have a spread of nearly nothing.
//
// Five, and what it costs to not have it was audible. A built preset put its
// cabinet's microphone at Position 0.0, which is dead centre of the cone and
// the brightest, harshest spot on it: the 8x10 Ampeg SVT-810AV appears in
// **two** of the 4,426 presets measured and the 2x15 Brute in none, so there
// was no median to follow and the catalog's default stood. Pooled across the
// 49 mic'd cabinets that do appear, over 225 uses, players sit at **0.39**,
// and the owner of this pedal had dialled **0.75** by hand on a preset he says
// sounds better than anything built here. The default was the outlier.
//
// 113 of the device's models appear in fewer than five presets, so this is not
// a rare path.
const thin = 5

// agreed returns the corpus median for a parameter, when there is one and
// when players agree closely enough to be worth following.
//
// The model's own presets first, and a pool of its category's second. Which
// one answered is not reported: both are "what other people did", and the
// value is the same kind of claim either way.
func agreed(
	b catalog.Block,
	key string,
	p catalog.Param,
	cat *catalog.Catalog,
	stats *corpus.Stats,
	instrument string,
) (catalog.ParamValue, bool) {
	if stats == nil {
		return catalog.ParamValue{}, false
	}

	d, ok := stats.Param(b.ID, key)
	if !ok || d.N < thin {
		// Everything of this kind that carries the same control, which is a
		// weaker claim than the model's own presets and a far stronger one
		// than a factory setting. Nothing where this kind of block has no
		// evidence either, which leaves the default standing.
		pooled, enough := across(b, key, cat, stats, instrument)
		if !enough {
			return catalog.ParamValue{}, false
		}

		d = pooled
	}

	// A switch is never averaged, however unanimous the corpus is about it:
	// the middle of an enumeration is not a value the device accepts. This
	// comes first because a switch has no range to measure agreement against.
	kind := p.Default.Type()
	if kind != catalog.ParamFloat && kind != catalog.ParamInt {
		return catalog.ParamValue{}, false
	}

	span := p.Max - p.Min
	if span <= 0 || d.Spread()/span > agreementThreshold {
		return catalog.ParamValue{}, false
	}

	// Rounding an integer matters: a device given 1.5 for a three-position
	// switch does not round it, it refuses the preset.
	//
	// math.Round rather than adding a half and truncating, because
	// truncation cuts toward zero and this catalog has parameters that go
	// negative: a median of -12.4 became -11.
	if kind == catalog.ParamInt {
		return catalog.Int(int64(math.Round(d.Median))), true
	}

	return catalog.Float(d.Median), true
}

// across pools one control's readings over every model of a block's own kind.
//
// Weighted by how many presets each model's median came from, so a model
// hundreds of people used counts for more than one two people did. The
// weighted median rather than the mean, for the reason every other figure here
// is a median: one preset with a control at a stop drags a mean and moves a
// median by nothing.
//
// Keyed by the parameter's own name, inside one category. Across categories it
// would be meaningless — a cabinet's `Level` and a delay's `Level` are not the
// same control — and that is why the catalog has to be in hand.
//
// Only models carrying the same controls, which is what makes two blocks the
// same kind of thing rather than merely the same category. A mic'd cabinet
// impulse response carries `Mic`, `Position`, `Angle`, `Distance`, `HighCut`,
// `LowCut` and `Level`; the legacy cabinet of the same speaker carries
// `Distance`, `EarlyReflections`, `High Cut`, `LowCut` and `Level`, with the
// space in the name. They are two different blocks and their figures do not
// pool: without this test a mic'd 8x10's `HighCut` came back at 12,000Hz off a
// pool of thirteen models, eleven of them legacy guitar cabinets sitting at
// 20,000. With it, 4,000Hz off the one mic'd sibling bass players actually
// use, over 65 of their presets.
//
// Only models whose range matches, for a narrower version of the same reason:
// a median is a position on a rail and two rails of different lengths do not
// share one.
//
// **And only this instrument's presets**, which is the whole difference
// between a fix and a worse sound. Guitar chains outnumber bass twenty to one
// here, so pooling over everything pooled over guitar: a mic'd cabinet's
// `HighCut` comes back at 12,000Hz across all 225 uses and at **4,000Hz**
// across the 65 bass ones. The first was brighter than the catalog's own
// 8,000 default and would have made the harsh top end it was meant to fix
// worse. A bass player rolls the top off a cabinet and a guitar player does
// not, and one median cannot say both.
func across(
	b catalog.Block,
	key string,
	cat *catalog.Catalog,
	stats *corpus.Stats,
	instrument string,
) (corpus.ParamStats, bool) {
	if cat == nil || instrument == "" {
		return corpus.ParamStats{}, false
	}

	// The models chains for this instrument are built from. Absent for an
	// instrument nobody has measured, which pools nothing rather than
	// pooling somebody else's.
	grammar, measured := stats.Grammar[instrument]
	if !measured || len(grammar.Models) == 0 {
		return corpus.ParamStats{}, false
	}

	want, carried := b.Params[key]
	if !carried {
		return corpus.ParamStats{}, false
	}

	same := make(map[string]bool, len(b.Params))
	for name := range b.Params {
		same[name] = true
	}

	type weighed struct {
		at float64
		n  int
	}

	read := []weighed(nil)
	total := 0

	for id, one := range cat.Blocks {
		if id == b.ID || one.Category != b.Category {
			continue
		}

		if grammar.Models[id] <= 0 {
			continue
		}

		if !sameControls(one, same) {
			continue
		}

		held, alike := one.Params[key]
		if !alike || held.Min != want.Min || held.Max != want.Max {
			continue
		}

		d, measured := stats.Param(id, key)
		if !measured || d.N <= 0 {
			continue
		}

		read = append(read, weighed{at: d.Median, n: d.N})
		total += d.N
	}

	if total < thin {
		return corpus.ParamStats{}, false
	}

	// Sorted so the weights can be walked to the middle, and by value then
	// weight so two models agreeing on a figure resolve the same way every
	// run rather than however the catalog's map was walked.
	sort.Slice(read, func(i, j int) bool {
		if read[i].at != read[j].at {
			return read[i].at < read[j].at
		}

		return read[i].n < read[j].n
	})

	// The dearest row stands in until the walk reaches the middle, which it
	// always does on a positive total: writing a refusal after the loop read
	// as an outcome this cannot have.
	median := read[len(read)-1].at
	seen := 0

	for _, at := range read {
		seen += at.n

		if seen*2 >= total {
			median = at.at

			break
		}
	}

	// Spread left at zero, which reads as total agreement and is the right
	// answer for a pooled figure: the spread this would report is the spread
	// between models rather than between players, and `agreed` uses spread to
	// decide whether players agree.
	return corpus.ParamStats{N: total, Median: median}, true
}

// sameControls says whether a block's controls are exactly the named set.
//
// Exactly, in both directions. A block with every one of these and one more is
// a different block, and so is one missing a single control: what this answers
// is "are these the same kind of thing", and a block with an extra knob has a
// different answer to what that knob does to the rest.
func sameControls(
	one catalog.Block,
	want map[string]bool,
) bool {
	if len(one.Params) != len(want) {
		return false
	}

	for name := range one.Params {
		if !want[name] {
			return false
		}
	}

	return true
}
