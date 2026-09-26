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
	"os"
	"sort"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
)

// NamesOptions is what checking a block's parameter order needs to know.
type NamesOptions struct {
	Client Prober
	Model  string
}

// probed is what moving one index settled.
//
// A type rather than a string, because a probe has four outcomes and only one
// of them is comparable with a name in the catalog. Flattened into one string
// they were all compared, and a switch — which correctly refuses a number and
// so tells you nothing about naming — was counted as the catalog being wrong.
// That reported every curve for this amplifier as misfiled while all twelve
// indexes it could actually test had matched.
type probed struct {
	// Named is the one parameter that moved, and is empty when the probe
	// settled nothing.
	Named string
	// Says is what to print when Named is empty.
	Says string
	// Switch is true when the device refused a number at this index. Not a
	// disagreement: an index nothing can sweep is an index this cannot hold
	// the catalog to either way.
	Switch bool
}

// probes are the values each index is moved to.
//
// Two, so an index already resting on the first is still caught: moving a
// control to where it already sits changes nothing and would read as an index
// that reaches no parameter.
var probes = []float64{0.123, 0.877}

// MeasureNames holds the catalog's parameter order to the device.
//
// A parameter has no name on the wire, only a position in the model's own
// list. The catalog records that order in its symbol table and nothing has
// ever checked the claim against hardware.
//
// It matters more than it sounds. `catalog show` prints the same parameters
// sorted for a reader, so the two orders disagree on nearly every model: one
// amplifier's listing begins Bass, Bias, BiasX and its wire order begins Norm
// Drive, Bass, Mid, Treble. Counting down the printed one mislabels every
// curve, and the numbers stay entirely plausible while it does.
func MeasureNames(
	ctx context.Context,
	w io.Writer,
	opts NamesOptions,
) error {
	cat, err := catalog.BuiltIn()
	if err != nil {
		return err
	}

	block, ok := cat.Blocks[catalog.ModelID(opts.Model)]
	if !ok {
		return fmt.Errorf("the catalog has no %s", opts.Model)
	}

	order := wireOrder(cat, opts.Model)

	work, err := os.MkdirTemp("", "tonestack-names")
	if err != nil {
		return fmt.Errorf("making somewhere to build a preset: %w", err)
	}

	defer func() { _ = os.RemoveAll(work) }()

	preset, err := compile(ctx, opts.Client, measured.Block{
		ID: string(block.ID), Name: block.Name,
		Category: block.Category,
	}, work, true)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "\n  %s (%s)\n\n  %-6s %-22s %s\n",
		block.Name, block.ID, "index", "the catalog says", "the device moved")

	agreed := true
	untested, unclaimed := 0, 0

	for index := range len(block.Params) {
		moved, err := probe(ctx, opts.Client, preset, index)
		if err != nil {
			return err
		}

		said := ""
		if index < len(order) {
			said = order[index]
		}

		// Three outcomes rather than two. An index the catalog says nothing
		// about cannot disagree with anything, and counting it as a
		// disagreement reported every equaliser's order as wrong when the
		// catalog had never claimed one: Line 6 ship no symbol list for them,
		// so the device is the only thing that knows.
		//
		// An index that settled nothing is untested, which is not evidence
		// either way.
		switch {
		case said == "":
			unclaimed++
		case moved.Named == "":
			untested++
		case moved.Named != said:
			agreed = false
		}

		claim := said
		if claim == "" {
			claim = "—"
		}

		says := moved.Named
		if says == "" {
			says = moved.Says
		}

		_, _ = fmt.Fprintf(w, "  %-6d %-22s %s\n", index, claim, says)
	}

	if !agreed {
		_, err = fmt.Fprintf(w,
			"\n  they disagree, so every curve filed by index for this block is\n"+
				"  filed under the wrong control\n\n")

		return err
	}

	// The catalog claims no order at all, so there is nothing to hold it to
	// and the table above is the only record of which index is which. That is
	// the ordinary case for an equaliser, and establishing it is what has to
	// happen before one can be swept.
	if unclaimed == len(block.Params) {
		_, err = fmt.Fprintf(w,
			"\n  the catalog claims no order for this block, so the device's own\n"+
				"  is above and is the only record of it\n\n")

		return err
	}

	if unclaimed > 0 || untested > 0 {
		_, err = fmt.Fprintf(w,
			"\n  the catalog and the device agree on every index the catalog\n"+
				"  claims and this could test, with %d unclaimed and %d untestable\n\n",
			unclaimed, untested)

		return err
	}

	_, err = fmt.Fprintf(w,
		"\n  the catalog and the device agree on every index\n\n")

	return err
}

// probe moves one index and reports which named parameter changed.
func probe(
	ctx context.Context,
	client Prober,
	preset string,
	index int,
) (probed, error) {
	for _, at := range probes {
		// The chain put back first, so an index is measured against the
		// preset as it was authored rather than against whatever the last
		// probe moved.
		if err := client.Play(ctx, preset); err != nil {
			return probed{}, err
		}

		before, err := held(ctx, client)
		if err != nil {
			return probed{}, err
		}

		address := sdk.Address{Block: alone, Param: index, Direct: true}
		if err := client.Turn(ctx, address, float32(at)); err != nil {
			return probed{Says: "refused a number on a dial", Switch: true}, nil
		}

		after, err := held(ctx, client)
		if err != nil {
			return probed{}, err
		}

		moved := changed(before, after)

		switch len(moved) {
		case 0:
			continue
		case 1:
			return probed{Named: moved[0]}, nil
		default:
			return probed{
				Says: fmt.Sprintf("moved %d at once: %v", len(moved), moved),
			}, nil
		}
	}

	return probed{Says: "nothing changed"}, nil
}

// held is the parameters of the block being probed, as the device holds them.
func held(
	ctx context.Context,
	client Prober,
) (map[string]any, error) {
	read, err := client.Current(ctx, sdk.FormatRig)
	if err != nil {
		return nil, err
	}

	if len(read.Rig.Chain) < alone {
		return nil, fmt.Errorf(
			"the chain has %d blocks, so slot %d is not one of them",
			len(read.Rig.Chain), alone)
	}

	entry := read.Rig.Chain[alone-1]
	if entry.Params == nil {
		return map[string]any{}, nil
	}

	return *entry.Params, nil
}

// changed is the parameters that differ between two readings.
func changed(
	before, after map[string]any,
) []string {
	out := make([]string, 0, 1)

	for name, now := range after {
		if fmt.Sprint(before[name]) != fmt.Sprint(now) {
			out = append(out, name)
		}
	}

	sort.Strings(out)

	return out
}
