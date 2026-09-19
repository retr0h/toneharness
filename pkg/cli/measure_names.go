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
	Client Pedal
	Model  string
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
		Category: string(block.Category),
	}, work, true)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "\n  %s (%s)\n\n  %-6s %-22s %s\n",
		block.Name, block.ID, "index", "the catalog says", "the device moved")

	agreed := true

	for index := range len(block.Params) {
		moved, err := probe(ctx, opts.Client, preset, index)
		if err != nil {
			return err
		}

		said := "—"
		if index < len(order) {
			said = order[index]
		}

		if said != moved {
			agreed = false
		}

		_, _ = fmt.Fprintf(w, "  %-6d %-22s %s\n", index, said, moved)
	}

	if agreed {
		_, _ = fmt.Fprintf(w,
			"\n  the catalog and the device agree on every index\n\n")

		return nil
	}

	_, err = fmt.Fprintf(w,
		"\n  they disagree, so every curve filed by index for this block is\n"+
			"  filed under the wrong control\n\n")

	return err
}

// probe moves one index and reports which named parameter changed.
func probe(
	ctx context.Context,
	client Pedal,
	preset string,
	index int,
) (string, error) {
	for _, at := range probes {
		// The chain put back first, so an index is measured against the
		// preset as it was authored rather than against whatever the last
		// probe moved.
		if err := client.Play(ctx, preset); err != nil {
			return "", err
		}

		before, err := held(ctx, client)
		if err != nil {
			return "", err
		}

		address := sdk.Address{Block: alone, Param: index, Direct: true}
		if err := client.Turn(ctx, address, float32(at)); err != nil {
			return "refused a number on a dial", nil
		}

		after, err := held(ctx, client)
		if err != nil {
			return "", err
		}

		moved := changed(before, after)

		switch len(moved) {
		case 0:
			continue
		case 1:
			return moved[0], nil
		default:
			return fmt.Sprintf("moved %d at once: %v", len(moved), moved), nil
		}
	}

	return "nothing changed", nil
}

// held is the parameters of the block being probed, as the device holds them.
func held(
	ctx context.Context,
	client Pedal,
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
