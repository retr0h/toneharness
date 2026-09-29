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

//go:generate go tool go.uber.org/mock/mockgen -source=tone_reach.go -destination=internal/mocks/reaches.gen.go -package=mocks

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// Reaches asks whether a chain could meet a target, without running.
type Reaches interface {
	Reach(ctx context.Context, in sdk.ReachAsk) (sdk.Reaching, error)
}

// ReachOptions is what asking needs.
type ReachOptions struct {
	Client Reaches
	// ID is the curated rig to ask about, and Genre what to aim it at.
	ID    string
	Genre string
	// Corpus is the tree the target is measured from, and Sweeps the tree
	// holding this chain's readings.
	Corpus string
	Sweeps string
}

// Reach prints which axes of a target this chain could reach.
//
// The arithmetic is the SDK's, because the same question is asked over MCP and
// neither surface may answer it differently. This is the painting.
func Reach(
	ctx context.Context,
	w io.Writer,
	opts ReachOptions,
) error {
	got, err := opts.Client.Reach(ctx, sdk.ReachAsk{
		RigID:  opts.ID,
		Genre:  opts.Genre,
		Corpus: opts.Corpus,
		Sweeps: opts.Sweeps,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "\n  %s against %s, from %d readings already taken\n",
		got.Rig, got.Genre, got.Readings)

	// Before the table rather than under it. Every number below is built from
	// slopes measured with each block on its own, and on a chain of more than
	// one block that is not a caveat: it is the answer being about something
	// else. An SV Beast swept with no cabinet reads its Treble at 12,763Hz of
	// centroid per turn where the same control in this chain reads 3,250, and
	// four of its eleven controls change sign.
	if got.Alone && got.Blocks > 1 {
		_, _ = fmt.Fprintf(w,
			"    [!] these readings were taken with each block on its own, and "+
				"this chain has %d.\n"+
				"        A slope is not a property of a control: the same Treble "+
				"into a 4x12 and\n        into a 1x15 are two different numbers. "+
				"Read the table as what the blocks\n        do apart, not as what "+
				"this chain does. `measure slopes --id %s` is\n        the same "+
				"controls read in the chain itself.\n", got.Blocks, got.Rig)
	}

	for _, m := range got.Unswept {
		_, _ = fmt.Fprintf(w,
			"    [!] %s has never been swept, so nothing it could move is counted\n", m)
	}

	axes(w, got)
	verdict(w, got)

	return nil
}

// axes prints one line per axis the target names, worst first.
func axes(
	w io.Writer,
	got sdk.Reaching,
) {
	_, _ = fmt.Fprintf(w, "\n    %-12s %9s %9s %9s\n",
		"AXIS", "OUT BY", "ALONE", "TOGETHER")

	for _, v := range got.Axes {
		mark, say := "  ", "nothing rules it out"

		switch {
		case v.Met:
			mark, say = "ok", "already there"
		case v.Shown:
			mark, say = "ok", "a reading landed there"
		case !v.Within:
			mark, say = "->", "OUT OF REACH"
		}

		if v.Together > 1 {
			mark = "->"
		}

		_, _ = fmt.Fprintf(w, "    %s %-9s %9.1f %9s %9.1f  %s\n",
			mark, v.Figure, v.Gap, onItsOwn(v), v.Together, say)
	}
}

// onItsOwn is what the per-axis check made of this figure, in a word.
//
// A word rather than the swing itself, because the swing runs into the
// thousands on a chain of sixteen dials and a number that large reads as
// precision when it is a bound that flatters every control in the chain.
func onItsOwn(
	v solve.Verdict,
) string {
	switch {
	case v.Met:
		return "met"
	case v.Shown:
		return "reached"
	case v.Within:
		return "maybe"
	default:
		return "no"
	}
}

// verdict is the answer the table was working towards.
func verdict(
	w io.Writer,
	got sdk.Reaching,
) {
	if len(got.Axes) == 0 {
		_, _ = fmt.Fprintf(w,
			"\n  Nothing to aim at: the target names no axis this chain reads.\n")

		return
	}

	if !got.Worth {
		_, _ = fmt.Fprintf(w,
			"\n  Not worth running. %s is %.1f tolerances out and every control "+
				"in this chain, added up and pulling together, moves it %.1f.\n"+
				"  The gear is wrong for this sound. Change the chain, not the knobs.\n",
			got.Decides.Figure, got.Decides.Gap, got.Decides.Swing)

		return
	}

	if got.Together {
		_, _ = fmt.Fprintf(w,
			"\n  Worth running. In the model one set of positions satisfies all "+
				"%d axes at once.\n"+
				"  In the model, and the model flatters: its slopes were read one "+
				"control\n  at a time and hold only near where they were read. "+
				"Every shipped rig\n  reads this way against every measured genre, "+
				"and hardware does not agree.\n", len(got.Axes))

		return
	}

	// The answer that per-axis could never give. Every axis reachable on its
	// own and no single set of positions reaching them together is the ordinary
	// case, and it is what somebody means by asking whether a rig can sound
	// like something.
	var missed []string

	for _, v := range got.Axes {
		if v.Together > 1 {
			missed = append(missed,
				fmt.Sprintf("%s by %.1f", v.Figure, v.Together))
		}
	}

	// Nothing over a tolerance and still not arrived means no axis could be
	// solved for at all: every slope against them is zero, so the joint answer
	// is absent rather than negative. Naming no axis would print a sentence
	// with a hole in it.
	if len(missed) == 0 {
		_, _ = fmt.Fprintf(w,
			"\n  Worth running, though nothing here can say how it will go: no "+
				"control in\n  this chain has a measured slope on any axis the "+
				"target names.\n")

		return
	}

	_, _ = fmt.Fprintf(w,
		"\n  Every axis is reachable on its own and no one set of positions "+
			"reaches them together.\n  Solved as a whole the model still misses "+
			"%s.\n  Worth running, and expect the loop to trade one axis off "+
			"against another.\n", strings.Join(missed, ", "))
}
