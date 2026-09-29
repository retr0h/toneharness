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

	"github.com/retr0h/toneharness/pkg/sdk"
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
	_, _ = fmt.Fprintf(w, "\n    %-12s %10s %10s\n", "AXIS", "OUT BY", "CAN MOVE")

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

		_, _ = fmt.Fprintf(w, "    %s %-9s %10.1f %10.1f  %s\n",
			mark, v.Figure, v.Gap, v.Swing, say)
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

	if got.Worth {
		var shown int

		for _, v := range got.Axes {
			if v.Met || v.Shown {
				shown++
			}
		}

		_, _ = fmt.Fprintf(w,
			"\n  Worth running. %d of %d axes have a reading that already landed "+
				"inside the target, and nothing rules the rest out.\n",
			shown, len(got.Axes))

		return
	}

	_, _ = fmt.Fprintf(w,
		"\n  Not worth running. %s is %.1f tolerances out and every control in "+
			"this chain, added up and pulling together, moves it %.1f.\n"+
			"  The gear is wrong for this sound. Change the chain, not the knobs.\n",
		got.Decides.Figure, got.Decides.Gap, got.Decides.Swing)
}
