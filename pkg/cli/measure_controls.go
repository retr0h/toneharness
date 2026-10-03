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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/reamp"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// ControlsOptions is what sweeping one block's controls needs to know.
type ControlsOptions struct {
	// Volume is where the computer's own output level is put before anything is
	// measured, 0 to 100.
	//
	// Only in the signal path on the rig that plays the reference out of the
	// computer's own output, which is the rig that opens the measuring loop. Set
	// rather than trusted either way, because it is a tone control: an
	// amplifier's distortion depends on how hard it is driven, so two runs at
	// different levels measure two different amplifiers.
	Volume int
	// HeadroomTold says somebody named --headroom themselves.
	//
	// Carried because the default is rig-dependent and an explicit value is
	// not. A person naming -12 on an open rig may be bounding something this
	// cannot see, and overriding them would make the flag a suggestion.
	HeadroomTold bool
	// Headroom is how far the chain's own output is turned down before
	// anything is measured, in decibels, and wants to be negative. The
	// measuring lead makes the chain feed itself.
	Headroom float64
	Client   Pedal
	Model    string
	Dry      string
	Out      string
	Seconds  float64
	Points   int
	Takes    int
	Hardware string
	// Bench is somewhere to push the signal through, as it is for measuring
	// blocks. Left unset, the named hardware is opened.
	Bench sdk.Bench
}

// alone is where a block sits on the grid when it is the only thing in the
// chain: 1, because the grid keeps the input at 0.
//
// A block's slot is its position plus one, and a chain holding one block puts
// it at position zero.
const alone = 1

// silent is how far under the settled level a reading may sit and still be a
// reading.
//
// Below this nothing came through, and the figures stop describing the chain
// and start describing the converters. The noise floor cannot catch it on its
// own, because two takes of silence agree to the last digit and so clear the
// floor more convincingly than music does. What comes out is not a null
// result but a confident one: the centroid of hiss is broadband and reads
// high, so a control that mutes the chain at one end of its travel reports an
// enormous, repeatable move.
const silent = 30.0

// floors are the smallest spread a noise floor is allowed to claim.
//
// Back-to-back takes through a settled loop can agree exactly, and a floor of
// zero would make every difference significant, including the last digit of a
// float.
var floors = map[audio.Figure]float64{
	audio.KeyCentroid: 0.2, audio.KeyLevel: 0.05,
	audio.KeyLow: 0.02, audio.KeyMid: 0.02, audio.KeyHigh: 0.005,
	audio.KeyTransient: 0.005, audio.KeyDecay: 0.01,
	audio.KeyDynamics: 0.05, audio.KeyHarmonics: 0.05, audio.KeyLean: 0.005,
}

// MeasureControls sweeps every control of one block, alone, and writes what
// each of them does.
func MeasureControls(
	ctx context.Context,
	w io.Writer,
	opts ControlsOptions,
) error {
	// Pinned first, because every reading below depends on it and a level
	// somebody left somewhere else is a different measurement.
	level := levelled(w, opts.Volume, reamp.Held)
	opts.Headroom = trimFor(w, opts.Hardware, opts.Headroom, opts.HeadroomTold)
	cat, err := opts.Client.Catalog(ctx)
	if err != nil {
		return err
	}

	block, ok := cat.Blocks[catalog.ModelID(opts.Model)]
	if !ok {
		return fmt.Errorf("the catalog has no %s", opts.Model)
	}

	// The catalog's order where there is one, and the device's own where there
	// is not. Line 6 ship no symbol list for an equaliser, so for those the
	// hardware is the only thing that knows which index is which, and a sweep
	// that refused would have left the blocks whose fingerprints say nothing
	// as the blocks nothing describes at all.
	//
	// Discovered rather than typed in. An order handed over on a flag is an
	// order somebody can get wrong, and a curve filed under the wrong control
	// is a plausible number about a different knob.
	order := wireOrder(cat, opts.Model)
	probed := len(order) == 0

	if probed {
		_, _ = fmt.Fprintf(w,
			"  %s claims no wire order, so asking the device for it\n", opts.Model)

		if order, err = discover(ctx, opts.Client, block, opts.Model, opts.Headroom); err != nil {
			return err
		}
	}

	signal, err := reference(opts.Dry, opts.Seconds)
	if err != nil {
		return err
	}

	sum, err := hash(opts.Dry)
	if err != nil {
		return err
	}

	work, err := os.MkdirTemp("", "toneharness-controls")
	if err != nil {
		return fmt.Errorf("making somewhere to build a preset: %w", err)
	}

	defer func() { _ = os.RemoveAll(work) }()

	entry := measured.Block{
		ID:       string(block.ID),
		Name:     block.Name,
		Category: block.Category,
	}

	preset, err := compile(ctx, opts.Client, entry, work, true, opts.Headroom)
	if err != nil {
		return fmt.Errorf("building a chain holding only %s: %w", opts.Model, err)
	}

	bench, release, err := benchFor(opts.Bench, opts.Hardware)
	if err != nil {
		return err
	}

	defer release()

	out := measured.Curves{
		Device: cat.Device, Gear: block.Name, Block: string(block.ID),
		Slot: alone, Isolated: true, Probed: probed,
		Instrument: referenceIsFor(opts.Dry), Headroom: opts.Headroom,
		Volume: level,
		Reference: measured.Reference{
			File: opts.Dry, SHA256: sum, Seconds: opts.Seconds,
		},
		Controls: map[string]measured.Curve{},
	}

	want := sweepable(block, order)

	_, _ = fmt.Fprintf(w, "\n  %s (%s), %d controls through %s\n\n",
		block.Name, block.ID, len(want), bench.Name())

	for _, c := range want {
		curve, err := sweep(ctx, w, opts, bench, signal, preset, c)
		if err != nil {
			return err
		}

		if curve != nil {
			out.Controls[c.name] = *curve
		}
	}

	// The chain as the device reported it, read after the sweeps and with the
	// preset put back, so what travels in the file is what was measured.
	if err := opts.Client.Play(ctx, preset); err != nil {
		return err
	}

	if out.Chain, err = current(ctx, opts.Client); err != nil {
		return err
	}

	return write(w, out, opts.Out)
}

// control is one parameter worth sweeping.
type control struct {
	index     int
	name      string
	kind      string
	low, high float64
}

// sweepable is the controls of a block that a sweep can move.
//
// Dials, lists and switches. A parameter the catalog cannot place is left out
// because it would be swept over a guessed range and filed under a guessed
// name.
//
// A switch used to be left out, on the grounds that two positions is not a
// curve. Two positions is not a curve and is a comparison, which is what a list
// gets: the machinery that ranks a cabinet's twelve microphones does not care
// that there are twelve. Line 6 record a switch's bounds as false and true,
// which carries no range, so the range is the two settings themselves.
func sweepable(
	block catalog.Block,
	order []string,
) []control {
	out := make([]control, 0, len(order))

	for i, name := range order {
		spec, ok := block.Params[name]
		if !ok {
			continue
		}

		low, high := spec.Min, spec.Max

		switch spec.Type {
		case "float", "int":
		case "bool":
			// Off and on, numbered, because a sweep works in positions and a
			// switch's own bounds say nothing: the catalog carries false and
			// true, which is not a range to step through.
			low, high = 0, 1
		default:
			continue
		}

		out = append(out, control{
			index: i, name: name, kind: string(spec.Type),
			low: low, high: high,
		})
	}

	return out
}

// sweep moves one control through its range and measures at every position.
func sweep(
	ctx context.Context,
	w io.Writer,
	opts ControlsOptions,
	bench sdk.Bench,
	signal []float32,
	preset string,
	c control,
) (*measured.Curve, error) {
	// The chain put back before every control, because a live edit writes
	// nothing back: a control stays wherever the last sweep left it, at the
	// top of its range, and the next one is then measured on a chain the
	// first skewed.
	if err := opts.Client.Play(ctx, preset); err != nil {
		return nil, err
	}

	noise, settled, err := steady(ctx, bench, signal, opts.Takes)
	if err != nil {
		return nil, err
	}

	// Every setting, for a control that is a list. Nothing sits between two
	// microphones, so asking for nine evenly spaced positions across twelve
	// would measure some twice and miss others.
	points := opts.Points
	if c.kind == "int" || c.kind == "bool" {
		points = int(c.high-c.low) + 1
	}

	curve := measured.Curve{
		Index: c.index, Kind: c.kind, Control: c.name,
		Span:        measured.Span{Low: c.low, High: c.high, FromCatalog: true},
		Noise:       noise,
		Settled:     settled,
		SilentBelow: silentBelow(settled),
	}

	_, _ = fmt.Fprintf(w, "  %s (index %d, %s), %d positions from %g to %g\n",
		c.name, c.index, c.kind, points, c.low, c.high)

	for i := range points {
		at := c.low
		if points > 1 {
			at = c.low + (c.high-c.low)*float64(i)/float64(points-1)
		}

		if err := turn(ctx, opts.Client, c, at); err != nil {
			_, _ = fmt.Fprintf(w, "    %8g  refused\n", at)

			continue
		}

		got, err := sdk.Fingerprint(ctx, bench, signal)
		if err != nil {
			return nil, err
		}

		switch verdict, why := judge(got.Level, curve.SilentBelow); verdict {
		case tooQuiet:
			curve.MutedAt = append(curve.MutedAt, at)
			_, _ = fmt.Fprintf(w, "    %8g  %s\n", at, why)

			continue
		case tooLoud:
			curve.ClippedAt = append(curve.ClippedAt, at)
			_, _ = fmt.Fprintf(w, "    %8g  %s\n", at, why)

			continue
		case believable:
		}

		curve.Points = append(curve.Points, measured.Point{Value: at, Figures: got})

		_, _ = fmt.Fprintf(w, "    %8g  centroid %9.1f  level %7.2f\n",
			at, got.Centroid, got.Level)
	}

	if len(curve.Points) < 2 {
		_, _ = fmt.Fprintf(w,
			"    only %d position carried signal, so there is no curve here\n\n",
			len(curve.Points))

		return nil, nil
	}

	fill(&curve)
	report(w, curve)

	return &curve, nil
}

// fill works out what the measured positions say.
//
// A list and a switch get a spread and no slope; a dial gets a slope. The split
// is the same one the solver makes, and getting it wrong here is not visible
// downstream: a line through a switch's two points fits perfectly by
// construction, so it would carry a slope with Straight at 1.0 and be believed
// over every dial on the block.
func fill(
	curve *measured.Curve,
) {
	if curve.Kind == "int" || curve.Kind == "bool" {
		curve.Spread = map[audio.Figure]float64{}

		for _, f := range measured.Named() {
			if apart, ok := measured.Apart(curve.Points, f); ok {
				curve.Spread[f] = apart
			}
		}

		return
	}

	curve.Fits = map[audio.Figure]measured.Fit{}

	for _, f := range measured.Named() {
		curve.Fits[f] = measured.Fitted(curve.Points, f)
	}
}

// report says what one control did, against what the rig can tell apart.
func report(
	w io.Writer,
	curve measured.Curve,
) {
	for _, f := range measured.Named() {
		floor := curve.Noise[f]

		var moved float64
		if curve.Kind == "int" || curve.Kind == "bool" {
			moved = curve.Spread[f]
		} else {
			moved, _ = measured.Apart(curve.Points, f)
		}

		aimable := "no"
		if moved > floor*3 {
			aimable = "yes"
		}

		_, _ = fmt.Fprintf(w, "    %-10s moved %9.3f  floor %7.3f  real? %s\n",
			f, moved, floor, aimable)
	}

	_, _ = fmt.Fprintln(w)
}

// turn moves one control, with the kind of value the device will take.
//
// A device does not coerce. The value's tag is its type on the wire, so a
// parameter wanting an index refuses a float with the same error it gives for
// a block that is not there.
func turn(
	ctx context.Context,
	client Pedal,
	c control,
	at float64,
) error {
	address := sdk.Address{Block: alone, Param: c.index, Direct: true}

	switch c.kind {
	case "int":
		return client.Choose(ctx, address, int(math.Round(at)))
	case "bool":
		return client.Switch(ctx, address, at >= 0.5)
	}

	return client.Turn(ctx, address, float32(at))
}

// steady is how much a reading wanders when nothing is touched, and how loud
// it is when the chain is working.
//
// The level comes back with the floor because the floor alone cannot catch
// silence: two takes of silence agree perfectly, so a figure computed from
// them looks more trustworthy than one computed from music.
func steady(
	ctx context.Context,
	bench sdk.Bench,
	signal []float32,
	takes int,
) (map[audio.Figure]float64, float64, error) {
	// One reading thrown away before any is kept, because the first one after
	// a bench opens is the audio stream still settling rather than the chain.
	//
	// Six takes of a punk chain, the first against the other five: low 19.35
	// against 31.56 to 31.73, high 80.35 against 67.96 to 68.14, the centre
	// 2109.7Hz against 1805 to 1809. The five that follow agree to three
	// decimal places; the first is twelve points of a band away from all of
	// them.
	//
	// What that cost is every tolerance in the loop. The floor is the lower
	// bound on all of them, so one unsettled take stretched it from 0.0010 of
	// a band to 0.1238 and made the target that much easier to hit. A run then
	// reported arriving while its residual grew from 1.1 tolerances out to
	// 2.4, which is the loop saying a chain had converged as it walked away
	// from the target.
	if _, err := sdk.Fingerprint(ctx, bench, signal); err != nil {
		return nil, 0, err
	}

	rows := make([]measured.Figures, 0, takes)

	for range takes {
		got, err := sdk.Fingerprint(ctx, bench, signal)
		if err != nil {
			return nil, 0, err
		}

		rows = append(rows, got)
	}

	points := make([]measured.Point, 0, len(rows))
	levels := make([]float64, 0, len(rows))

	for _, r := range rows {
		points = append(points, measured.Point{Figures: r})
		levels = append(levels, r.Level)
	}

	out := map[audio.Figure]float64{}

	// Wander rather than Apart, so one take that disagrees with the rest does
	// not become the floor on its own.
	//
	// Both are needed and they catch different things. The discarded first
	// reading above is the stream settling, which is known and always the
	// first; this is any other reading that comes back odd, whichever one it
	// is. With the discard alone the floor still came back at 0.2615 of a band
	// about one run in nine, against 0.0002 either side of it.
	for _, f := range measured.Named() {
		apart, ok := measured.Wander(points, f)
		if !ok {
			continue
		}

		out[f] = math.Max(apart, floors[f])
	}

	sort.Float64s(levels)

	return out, levels[len(levels)/2], nil
}

// discover asks the device which index is which.
//
// The same probe `measure names` uses, and for the same reason: a parameter has
// no name on the wire, only a position in the model's own list. Moving each
// index and reading back which named parameter changed is the only way to
// learn that for a block the catalog says nothing about.
//
// An index the device refuses a number for takes the parameter's own name from
// the catalog, so the slot is held and the sweep skips it rather than shifting
// every control after it by one.
func discover(
	ctx context.Context,
	client Pedal,
	block catalog.Block,
	model string,
	headroom float64,
) ([]string, error) {
	work, err := os.MkdirTemp("", "toneharness-order")
	if err != nil {
		return nil, fmt.Errorf("making somewhere to build a preset: %w", err)
	}

	defer func() { _ = os.RemoveAll(work) }()

	preset, err := compile(ctx, client, measured.Block{
		ID: model, Name: block.Name, Category: block.Category,
	}, work, true, headroom)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(block.Params))

	for index := range len(block.Params) {
		moved, err := probe(ctx, client, preset, index)
		if err != nil {
			return nil, err
		}

		// Empty holds the position without naming it, so a control this could
		// not settle is skipped rather than mistaken for its neighbour.
		out = append(out, moved.Named)
	}

	if !slices.ContainsFunc(out, func(s string) bool { return s != "" }) {
		return nil, fmt.Errorf(
			"the device named no parameter of %s at any index, so nothing "+
				"here can tell which control a reading would belong to", model)
	}

	return out, nil
}

// wireOrder is a model's parameters in the order the device addresses them.
//
// From the symbol list, which records them "in the order a device sends their
// values". The per-model map beside it is keyed by name and has no order at
// all, and `catalog show` prints that one sorted, so counting down the
// printout files every curve under the wrong control.
func wireOrder(
	cat *catalog.Catalog,
	model string,
) []string {
	for _, sym := range cat.Symbols {
		if string(sym.ID) == model {
			return sym.Params
		}
	}

	return nil
}

// current is the chain the device is playing, as the rig it describes.
func current(
	ctx context.Context,
	client Reads,
) (string, error) {
	read, err := client.Current(ctx, sdk.FormatRig)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tone.Write(&buf, tone.Spec{
		Schema: tone.SchemaName, Id: read.ID, Rig: read.Rig,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// write puts the curves where they were asked for.
func write(
	w io.Writer,
	out measured.Curves,
	where string,
) error {
	body, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		return fmt.Errorf("writing the curves: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(where), 0o750); err != nil {
		return fmt.Errorf("writing %s: %w", where, err)
	}

	if err := os.WriteFile(where, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", where, err)
	}

	named := make([]string, 0, len(out.Controls))
	for name := range out.Controls {
		named = append(named, name)
	}

	sort.Strings(named)

	_, err = fmt.Fprintf(w, "  %d controls measured: %s\n  wrote %s\n\n",
		len(named), strings.Join(named, ", "), where)

	return err
}
