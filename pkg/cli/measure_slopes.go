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
	"path/filepath"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// SlopesOptions is what checking a committed sweep against the chain needs.
type SlopesOptions struct {
	Client   Tuner
	Bench    sdk.Bench
	Hardware string
	Dry      string
	Seconds  float64
	Takes    int
	// ID is the curated rig whose chain the slopes are read in.
	ID string
	// Sweeps is the tree holding the committed readings to check against.
	Sweeps string
	// Nudge is how far a control moves to read its slope, as a fraction of
	// its range.
	Nudge float64
	// Volume is where the computer's own output level is put before anything is
	// measured, 0 to 100.
	//
	// Only in the signal path on the rig that plays the reference out of the
	// computer's own output, which is the rig that opens the measuring loop. Set
	// rather than trusted either way, because it is a tone control: an
	// amplifier's distortion depends on how hard it is driven, so two runs at
	// different levels measure two different amplifiers.
	Volume int
	// Headroom is how far the chain's own output is turned down before
	// anything is measured, in decibels, and wants to be negative.
	Headroom float64
	// Figure is the axis to report. Empty reports the ones a target uses
	// most.
	Figure string
}

// Slopes reads what each control does now and holds the committed sweeps to it.
//
// `tone reach` answers from slopes committed under resources/sweeps/, and says
// every shipped rig reaches every measured genre. `tone tune` reads its slopes
// live and does not agree: the same rig and target stopped 5.3 tolerances out.
// One of those is wrong and this is what tells them apart.
//
// A slope is not a property of a control. The same Treble into a 4x12 and into
// a 1x15 are two different numbers, and a committed sweep was measured in
// whatever chain it ran in. So the question is not whether the arithmetic is
// right, it is whether a number measured there means anything here.
//
// One reading per control rather than a sweep, which is what the tuning loop
// itself spends. A chain of a dozen dials is about a minute and a half.
func Slopes(
	ctx context.Context,
	w io.Writer,
	opts SlopesOptions,
) error {
	// Pinned first, because the committed slopes this compares against were
	// taken at a stated level and a ratio between two levels is not a ratio
	// about the control.
	levelled(w, opts.Volume)

	made, preset, _, err := built(ctx, TuneOptions{
		Client: opts.Client, ID: opts.ID, Headroom: opts.Headroom,
	})
	if err != nil {
		return err
	}

	_ = preset

	cat, err := catalog.BuiltIn()
	if err != nil {
		return err
	}

	if err := sameInstrument(made.Plan, cat, opts.Dry); err != nil {
		return err
	}

	knobs := knobsOf(made.Plan, cat)
	if len(knobs) == 0 {
		return fmt.Errorf("%w: the chain has no dial to turn", solve.ErrNoKnobs)
	}

	signal, err := reference(opts.Dry, opts.Seconds)
	if err != nil {
		return err
	}

	bench, release, err := benchFor(opts.Bench, opts.Hardware)
	if err != nil {
		return err
	}

	defer release()

	_, _ = fmt.Fprintf(w, "\n  %s, %d dials through %s\n",
		opts.ID, len(knobs), bench.Name())

	// Thrown away, because the first reading after the audio device opens is
	// the stream settling rather than the chain. Six takes of one untouched
	// chain read the low band at 19.35 once and then 31.56 to 31.73 five times
	// over, and a baseline taken from the first makes every slope afterwards
	// the difference between settled and unsettled. Six unrelated controls
	// then read the same slope to two decimal places, which is what this
	// printed before the discard went in.
	if _, err := sdk.Fingerprint(ctx, bench, signal); err != nil {
		return err
	}

	now, err := sdk.Fingerprint(ctx, bench, signal)
	if err != nil {
		return err
	}

	// The same reading the loop takes, through the same code, so what comes
	// out is the matrix the loop would have solved rather than a second
	// opinion about it.
	if err := slopes(ctx, bench, signal, TuneOptions{
		Client: opts.Client, Nudge: opts.Nudge,
	}, knobs, figuresOf(now)); err != nil {
		return err
	}

	return against(w, opts, made.Plan, knobs)
}

// against prints the live slopes beside the committed ones.
func against(
	w io.Writer,
	opts SlopesOptions,
	made plan.Plan,
	knobs []solve.Knob,
) error {
	committed, err := committedSlopes(
		made, opts.Sweeps, instrumentOf(opts.Dry), w)
	if err != nil {
		return err
	}

	figures := []audio.Figure{
		audio.KeyCentroid, audio.KeyLow, audio.KeyMid, audio.KeyHigh,
	}
	if opts.Figure != "" {
		figures = []audio.Figure{audio.Figure(opts.Figure)}
	}

	for _, figure := range figures {
		_, _ = fmt.Fprintf(w, "\n  %s\n", figure)
		_, _ = fmt.Fprintf(w, "    %-34s %12s %12s %9s\n",
			"CONTROL", "LIVE", "COMMITTED", "RATIO")

		rows := 0

		for _, k := range knobs {
			was, ok := committed[k.Where()][figure]
			if !ok {
				continue
			}

			is := k.Slope[figure]
			rows++

			_, _ = fmt.Fprintf(w, "    %-34s %12.4f %12.4f %9s\n",
				k.Control, is, was, ratio(is, was))
		}

		if rows == 0 {
			_, _ = fmt.Fprintf(w, "    nothing committed carries this figure\n")
		}
	}

	return nil
}

// ratio is how far the live slope sits from the committed one.
//
// A ratio rather than a difference, because the slopes span four orders of
// magnitude across the figures and a difference in hertz per turn says nothing
// about a band share. Signs matter more than sizes: a slope that changed
// direction is a control the matrix will push the wrong way.
func ratio(
	is, was float64,
) string {
	switch {
	case was == 0 && is == 0:
		return "both nil"
	case was == 0:
		return "was nil"
	case is == 0:
		return "now nil"
	case (is < 0) != (was < 0):
		return "OPPOSITE"
	}

	return fmt.Sprintf("%.2fx", is/was)
}

// committedSlopes reads the sweeps for a chain, keyed the way a knob is.
//
// Addressed by block position and parameter index rather than by name, because
// that is what a live edit moves and what the solver's matrix is built on. A
// name would match a control in the wrong block.
func committedSlopes(
	made plan.Plan,
	dir string,
	reference string,
	w io.Writer,
) (map[solve.Where]map[audio.Figure]float64, error) {
	out := map[solve.Where]map[audio.Figure]float64{}

	cat, err := catalog.BuiltIn()
	if err != nil {
		return nil, err
	}

	// Which committed sweeps say nothing about the instrument they were taken
	// with. Said rather than refused: the comparison is still the only way to
	// see how far a committed slope is from a live one, which is what this
	// command is for. But a ratio between a reading of one instrument and a
	// reading of another is not a ratio about the control, and a file that
	// predates the field cannot say which it is.
	var silent []string

	for _, b := range made.Blocks {
		at := filepath.Join(dir, string(b.Model)+".json")

		f, err := os.Open(at) //nolint:gosec // a path built from the catalog
		if err != nil {
			continue
		}

		curves, err := measured.LoadCurves(f)

		_ = f.Close()

		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", at, err)
		}

		switch {
		case curves.Instrument == "":
			silent = append(silent, string(b.Model))
		case reference != "" && curves.Instrument != reference:
			silent = append(silent, fmt.Sprintf("%s (%s)",
				b.Model, curves.Instrument))
		}

		order := wireOrder(cat, string(b.Model))

		for index, name := range order {
			curve, swept := curves.Controls[name]
			if !swept || len(curve.Fits) == 0 {
				continue
			}

			slope := map[audio.Figure]float64{}
			for key, fit := range curve.Fits {
				slope[key] = inCorpusScale(
					map[audio.Figure]float64{key: fit.PerTurn})[key]
			}

			out[solve.Where{Block: b.Pos, Param: index}] = slope
		}
	}

	if len(silent) > 0 {
		_, _ = fmt.Fprintf(w,
			"\n  %d of these blocks' committed sweeps do not name the instrument\n"+
				"  they were taken with, or name another: %s.\n"+
				"  Every figure committed here was taken with a bass, and the\n"+
				"  files predate the field that says so, so a RATIO below may be\n"+
				"  the two instruments rather than the control.\n",
			len(silent), strings.Join(silent, ", "))
	}

	return out, nil
}
