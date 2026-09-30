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

package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// This file is the list half of the tuning loop. knobsOf in tone_tune.go finds
// the dials and the solver moves them together; this finds the controls that
// are lists and compares their settings one at a time.
//
// Why they cannot be one pass is in solve/choose.go. Why the comparison is
// measured here rather than read out of resources/sweeps/ is the same reason
// slopes are: a figure a control produces is a figure about the chain it was
// produced in, and the same microphone in front of a 1x15 and a 4x12 reads
// differently. The committed sweeps say which controls are lists worth
// comparing, and the chain in hand says what each setting does.

// listsOf is every control in the chain that is compared rather than turned.
//
// Lists and switches. A switch was left out while nothing could measure one:
// `measure controls` swept floats and ints only, so there was no committed
// evidence that any switch moved a figure, and a pair of readings spent on
// Bright was a pair not spent on the eleven microphones that move a cabinet
// further than any of its dials.
//
// It is in now because the cost turned out to be two readings and the wire
// call already existed. A switch is a choice of two and `solve.Nearest` does
// not care how many settings there are. Across the 661 blocks of an HX Stomp
// there are 315 switches over 197 blocks, a median of one on a block that has
// any, so a chain of five or six adds a handful of readings rather than a pass.
func listsOf(
	made plan.Plan,
	cat *catalog.Catalog,
) []solve.Choice {
	var out []solve.Choice

	for _, b := range made.Blocks {
		block, ok := cat.Blocks[b.Model]
		if !ok {
			continue
		}

		for index, name := range wireOrder(cat, string(b.Model)) {
			spec, known := block.Params[name]
			if !known {
				continue
			}

			var (
				at   int
				flip bool
			)

			switch spec.Type {
			case catalog.ParamInt:
				got, _ := b.Params[name].Int()
				at = int(got)
			case catalog.ParamBool:
				// Off and on, numbered, because the comparison works in
				// settings. The catalog's bounds for a switch are false and
				// true, which is not a range to enumerate.
				flip = true

				if on, _ := b.Params[name].Bool(); on {
					at = 1
				}
			default:
				continue
			}

			out = append(out, solve.Choice{
				Block:   b.Pos,
				Param:   index,
				Control: fmt.Sprintf("%s %s", b.Model, name),
				Setting: name,
				At:      at,
				Options: options(spec),
				Flip:    flip,
			})
		}
	}

	return out
}

// settings is every value a list holds, in the device's own order.
//
// Every one of them, because nothing sits between two microphones: asking for
// nine evenly spaced positions across twelve would measure some twice and miss
// others, which is the same reason `measure controls` sweeps a list this way.
//
// Named by number rather than by name, because the catalog carries no names for
// them. A cabinet's Mic is an integer from 0 to 11 and Line 6 ship no symbol
// list saying which microphone each one is, so the report can only say which
// setting won and not what it is called.
func options(
	spec catalog.Param,
) []int {
	if spec.Type == catalog.ParamBool {
		return []int{0, 1}
	}

	var out []int

	for at := int(spec.Min); at <= int(spec.Max); at++ {
		out = append(out, at)
	}

	return out
}

// compared tries every setting of every list and leaves each on its nearest.
//
// One list at a time and in chain order, so each is compared with the earlier
// ones already on their answers. Choosing them independently and applying the
// results together would compare each against a chain that no longer exists by
// the time the choices land.
//
// The rankings come back because the nearest setting is not the last word. It
// is the nearest before any dial has moved, and the dials are what the solve
// then spends; a setting that read second may be the one a solve can finish
// from. Runners is how the loop backs up to it.
func compared(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	bench sdk.Bench,
	signal []float32,
	lists []solve.Choice,
	aims map[audio.Figure]solve.Aim,
	settled float64,
) (map[solve.Where][]solve.Option, map[solve.Where]string, error) {
	ranked := make(map[solve.Where][]solve.Option, len(lists))
	named := make(map[solve.Where]string, len(lists))

	for _, c := range lists {
		read, err := readings(ctx, w, opts, bench, signal, c, settled)
		if err != nil {
			return nil, nil, err
		}

		if len(read) == 0 {
			_, _ = fmt.Fprintf(w,
				"    no setting of %s carried a signal worth reading, so it is "+
					"left where the compiler put it\n", c.Setting)

			continue
		}

		order := solve.Nearest(aims, read)

		ranked[c.Where()], named[c.Where()] = order, c.Control

		if err := choose(ctx, opts, c, order[0].Value); err != nil {
			return nil, nil, err
		}

		said(w, c, order)
	}

	return ranked, named, nil
}

// readings is what the chain measures at each setting of one list.
//
// A setting is left out rather than scored when the device refuses it, when it
// mutes the chain, or when it clips the converters, because in each case the
// figures describe something other than the setting. The clipping guard is the
// one that matters: one microphone of a cabinet's twelve clipped and read a
// centroid of 4,471Hz where the other eleven sat between 126 and 147, and
// scored on that it wins every target asking for a bright sound.
func readings(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	bench sdk.Bench,
	signal []float32,
	c solve.Choice,
	settled float64,
) (map[int]map[audio.Figure]float64, error) {
	out := make(map[int]map[audio.Figure]float64, len(c.Options))

	for _, at := range c.Options {
		if err := choose(ctx, opts, c, at); err != nil {
			_, _ = fmt.Fprintf(w, "    %s %-3d refused\n", c.Setting, at)

			continue
		}

		got, err := sdk.Fingerprint(ctx, bench, signal)
		if err != nil {
			return nil, err
		}

		if verdict, why := judge(got.Level, silentBelow(settled)); verdict != believable {
			_, _ = fmt.Fprintf(w, "    %s %-3d %s\n", c.Setting, at, why)

			continue
		}

		out[at] = figuresOf(got)
	}

	return out, nil
}

// said reports what a list's settings read and which one won.
//
// The runner-up as well as the winner, because a gap of nothing between them
// says the control does not matter for this target and a gap of tolerances says
// it is the most important thing in the chain.
func said(
	w io.Writer,
	c solve.Choice,
	order []solve.Option,
) {
	_, _ = fmt.Fprintf(w, "    %s: %d of %d settings read, nearest is %d at "+
		"%.1f tolerances out\n",
		c.Setting, len(order), len(c.Options), order[0].Value, order[0].Worst)

	if len(order) > 1 {
		_, _ = fmt.Fprintf(w, "      next nearest is %d at %.1f\n",
			order[1].Value, order[1].Worst)
	}
}

// choose puts one list on one of its settings.
//
// Choose rather than Turn. A device does not coerce: a parameter wanting an
// index refuses a float with the same error it gives for a block that is not
// there.
func choose(
	ctx context.Context,
	opts TuneOptions,
	c solve.Choice,
	to int,
) error {
	at := sdk.Control(c.Block, c.Param)

	// A switch takes the other call. The device does not coerce, so sending
	// the index 1 to one is refused with the error it gives for a block that
	// is not there, which reads as the address being wrong.
	if c.Flip {
		return opts.Client.Switch(ctx, at, to == 1)
	}

	return opts.Client.Choose(ctx, at, to)
}

// attempt is the two halves interleaved: choose the lists, solve the dials, and
// back up to another setting if the solve fell short.
//
// The order is not arbitrary. Choosing first and solving second is the only way
// round that works, because a choice changes every slope the solver then reads:
// the same Treble in front of two microphones is two different numbers, so
// slopes read before the choice describe a chain that no longer exists. Solving
// first and choosing second would be worse still, since it would spend the
// dials answering a target and then move the thing that moves furthest.
//
// The backing up is what makes it a loop rather than two steps. The nearest
// setting is the nearest with the dials wherever the compiler left them, and
// that is not the same question as which setting a solve can finish from. A
// microphone that reads a hair nearer and leaves every dial against a stop is a
// worse answer than one that reads further out with the whole range in hand,
// and nothing short of solving from both finds that out.
//
// Bounded by Tries, because each one costs a whole convergence. A chain of a
// dozen dials over three passes is around forty readings, which is five minutes
// of somebody's afternoon, and there is no target worth twelve of those.
func attempt(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	bench sdk.Bench,
	signal []float32,
	preset string,
	knobs []solve.Knob,
	lists []solve.Choice,
	aims map[audio.Figure]solve.Aim,
	settled float64,
) (round, error) {
	if len(lists) == 0 {
		return converge(ctx, w, opts, bench, signal, preset, knobs, aims, settled)
	}

	_, _ = fmt.Fprintf(w, "\n  comparing %d list(s), every setting of each\n",
		len(lists))

	ranked, named, err := compared(
		ctx, w, opts, bench, signal, lists, aims, settled)
	if err != nil {
		return round{}, err
	}

	did, err := converge(ctx, w, opts, bench, signal, preset, knobs, aims, settled)
	if err != nil || did.arrived {
		return did, err
	}

	return backedUp(ctx, w, opts, bench, signal, preset, knobs, lists,
		aims, settled, solve.Runners(ranked, named), did)
}

// backedUp tries the next-nearest settings, solving the dials from each.
//
// Kept rather than discarded when a later attempt is worse, because the answer
// to a target is the best chain anybody demonstrated and not the last one
// tried. A run that finds nothing better reports the first attempt's figures,
// which is what it would have reported without ever backing up.
func backedUp(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	bench sdk.Bench,
	signal []float32,
	preset string,
	knobs []solve.Knob,
	lists []solve.Choice,
	aims map[audio.Figure]solve.Aim,
	settled float64,
	runners []solve.Runner,
	best round,
) (round, error) {
	at := furthest(best.residual)

	for i, r := range runners {
		// One fewer than Tries, because the first attempt was already spent
		// on every list's nearest setting.
		//
		// No test for an empty runners: this ranges over it, so an empty one
		// never enters the loop at all.
		if i+1 >= opts.Tries {
			break
		}

		c, found := listAt(lists, r.Where)
		if !found {
			continue
		}

		_, _ = fmt.Fprintf(w,
			"\n  %.1f tolerances out, so %s goes to %d, which read %.1f, and the "+
				"dials are solved again\n", at, c.Setting, r.Option.Value, r.Option.Worst)

		if err := choose(ctx, opts, c, r.Option.Value); err != nil {
			return best, err
		}

		did, err := converge(ctx, w, opts, bench, signal, preset, knobs, aims, settled)
		if err != nil {
			return best, err
		}

		if got := furthest(did.residual); got < at {
			best, at = did, got
		}

		if did.arrived {
			return best, nil
		}
	}

	return best, nil
}

// listAt finds the control one runner names.
func listAt(
	lists []solve.Choice,
	where solve.Where,
) (solve.Choice, bool) {
	for _, c := range lists {
		if c.Where() == where {
			return c, true
		}
	}

	return solve.Choice{}, false
}
