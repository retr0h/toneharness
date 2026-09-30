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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/reamp"
)

// MeasureOptions is what measuring every block needs to know.
type MeasureOptions struct {
	Client Loader
	// Volume is where the computer's own output level is put before anything
	// is measured, 0 to 100.
	//
	// Only in the signal path on the rig that plays the reference out of the
	// computer's own output. Recorded either way, because a library that does
	// not say what level it was taken at cannot be reproduced.
	Volume int
	// HeadroomTold says somebody named --headroom themselves.
	//
	// Carried because the default is rig-dependent and an explicit value is
	// not. A person naming -12 on an open rig may be bounding something this
	// cannot see, and overriding them would make the flag a suggestion.
	HeadroomTold bool
	// Headroom is how far the chain's own output is turned down before
	// anything is measured, in decibels, and wants to be negative. The
	// measuring lead makes the chain feed itself, and enough gain around
	// that loop oscillates.
	Headroom float64
	Dry      string
	Out      string
	Category catalog.Category
	Seconds  float64
	Resume   bool
	Retry    bool
	Hardware string
	// Bench is somewhere to push the signal through. Left unset, the named
	// hardware is opened. A test sets it, because everything this does
	// besides listening is bookkeeping and ought not to need an audio
	// interface, a cable and somebody in the room to plug them in.
	Bench sdk.Bench
}

// clipped is where a reading stops describing the block and starts describing
// the converters running out of headroom.
//
// A clipped recording's spectrum is the clipping's: flat tops make harmonics
// that were never in the signal, so it reads as a bright block and is not one.
const clipped = -0.5

// MeasureBlocks measures every block the device has, once, at its own
// defaults.
func MeasureBlocks(
	ctx context.Context,
	w io.Writer,
	opts MeasureOptions,
) error {
	cat, err := catalog.BuiltIn()
	if err != nil {
		return err
	}

	signal, err := reference(opts.Dry, opts.Seconds)
	if err != nil {
		return err
	}

	sum, err := hash(opts.Dry)
	if err != nil {
		return err
	}

	lib := measured.Library{
		Device:   "HX Stomp",
		Isolated: true,
		Blocks:   map[string]measured.Block{},
		// Said in the file rather than left to the reference's filename,
		// because a figure is a figure about one instrument through one loop
		// and neither is recoverable from a path somebody may rename.
		Instrument: instrumentOf(opts.Dry),
		Headroom:   opts.Headroom,
		Reference: measured.Reference{
			File:    opts.Dry,
			SHA256:  sum,
			Seconds: opts.Seconds,
		},
	}

	if opts.Resume {
		resume(&lib, opts)
	}

	want := wanted(cat, opts.Category, lib.Blocks)

	bench, release, err := benchFor(opts.Bench, opts.Hardware)
	if err != nil {
		return err
	}

	defer release()

	_, _ = fmt.Fprintf(w, "\n  %d blocks through %s, %.0fs each\n",
		len(want), bench.Name(), opts.Seconds)

	work, err := os.MkdirTemp("", "toneharness-blocks")
	if err != nil {
		return fmt.Errorf("making somewhere to build presets: %w", err)
	}

	defer func() { _ = os.RemoveAll(work) }()

	built := build(ctx, w, opts.Client, want, work, opts.Headroom)

	// Before the baseline, because the baseline is a reading and this decides
	// what it reads. Setting it after would calibrate against a level the
	// campaign then changed.
	lib.Volume = levelled(w, opts.Volume)
	opts.Headroom = trimFor(w, opts.Hardware, opts.Headroom, opts.HeadroomTold)

	if err := baseline(ctx, w, bench, signal, &lib, work, opts); err != nil {
		return err
	}

	started := time.Now()

	for i, block := range want {
		at := built[block.ID]

		if !strings.HasSuffix(at, ".hlx") {
			lib.Blocks[block.ID] = measured.Block{
				ID: block.ID, Name: block.Name,
				Category: block.Category, Refused: at,
			}
		} else if err := readBackingOff(
			ctx, w, opts, bench, signal, &lib, block, at, work); err != nil {
			return err
		}

		reportBlock(w, i+1, len(want), lib.Blocks[block.ID])

		// Written every time. A run that holds its results until the end
		// loses them all when the pedal drops off the bus, which it has done.
		if err := keep(opts.Out, lib); err != nil {
			return err
		}
	}

	return done(w, lib, time.Since(started), opts.Out)
}

// one measures a single block, with its preset already built.
//
// Backed off per block when the reading is of the loop rather than of the
// block, because the gain that makes the loop run away is the block's own.
// Thirty decibels of headroom is enough for almost everything and not for the
// loudest amplifiers: sweeping every one of them at -30, three still read
// between 92% and 95% of their energy above 2kHz where the median was 0.41%.
//
// The same shape as land backing off when a pass mutes the chain. Halving is
// not available here because headroom is already in decibels, so it steps.
func one(
	ctx context.Context,
	client Plays,
	bench sdk.Bench,
	signal []float32,
	lib *measured.Library,
	block measured.Block,
	at string,
) error {
	// A session of its own, which is not an optimisation to remove. One held
	// open across many blocks takes the first live replace and refuses every
	// one after it.
	if err := client.Play(ctx, at); err != nil {
		lib.Blocks[block.ID] = measured.Block{
			ID: block.ID, Name: block.Name,
			Category: block.Category, Refused: short(err),
		}

		return nil
	}

	figures, err := sdk.Fingerprint(ctx, bench, signal)
	if err != nil {
		return err
	}

	block.Figures = figures
	block.Clipped = figures.Level > clipped
	lib.Blocks[block.ID] = block

	return nil
}

// quieterSteps is how many times a block's own reading may be taken again with
// more headroom, and by how much each time.
//
// Four steps of twelve decibels, which reaches -78 from a default of -30. The
// loudest amplifier measured needed more than -30 and the empty loop reads
// -62.68dB, so there is not much room past that: a reading quieter than the
// loop it travelled through says nothing.
const (
	quieterSteps = 4
	quieterBy    = -12.0
)

// readBackingOff measures a block, and takes the reading again with more
// headroom while it is of the loop rather than of the block.
//
// Reported rather than silent, because a block that needed 78 decibels of
// headroom is a block whose reading is near the floor and worth doubting.
func readBackingOff(
	ctx context.Context,
	w io.Writer,
	opts MeasureOptions,
	bench sdk.Bench,
	signal []float32,
	lib *measured.Library,
	block measured.Block,
	at, work string,
) error {
	if err := one(ctx, opts.Client, bench, signal, lib, block, at); err != nil {
		return err
	}

	dry := figuresOfDry(signal)
	headroom := opts.Headroom

	for range quieterSteps {
		got, held := lib.Blocks[block.ID]
		if !held || got.Refused != "" {
			return nil
		}

		bad, err := suspect(ctx, opts.Client, bench, signal, got, block, dry, at)
		if err != nil {
			return err
		}

		if !bad {
			return nil
		}

		headroom += quieterBy

		_, _ = fmt.Fprintf(w,
			"        %s read the loop rather than itself, again at %.0fdB\n",
			block.ID, headroom)

		next, err := compile(ctx, opts.Client, block, work, true, headroom)
		if err != nil {
			return err
		}

		if err := one(
			ctx, opts.Client, bench, signal, lib, block, next); err != nil {
			return err
		}
	}

	// Out of steps and still reading the loop. Marked rather than filed,
	// because the figures describe the loop and a reading nobody can use is
	// worse than a gap: a gap gets looked into and a number gets believed.
	got, held := lib.Blocks[block.ID]
	if !held || got.Refused != "" {
		return nil
	}

	bad, err := suspect(ctx, opts.Client, bench, signal, got, block, dry, at)
	if err != nil {
		return err
	}

	if bad {
		lib.Blocks[block.ID] = measured.Block{
			ID: block.ID, Name: block.Name, Category: block.Category,
			Refused: fmt.Sprintf(
				"read the loop rather than itself at %.0fdB of headroom", headroom),
		}
	}

	return nil
}

// suspect says a block's reading is of the loop rather than of the block.
//
// Two tests, because one does not cover both kinds of block.
//
// A cabinet is a low pass, so a reading brighter than what went in is a
// reading of something else. That settles every cabinet and every block that
// cannot add high end.
//
// An amplifier alone is legitimately brighter than a bass DI, so the invariant
// says nothing about it, and the loudest amplifiers are exactly the ones that
// squeal. For those the test is whether the reading depends on its input at
// all: push silence through and an oscillation comes back anyway, because that
// is what self-sustaining means, while a block passing a signal has nothing to
// pass.
//
// The silence reading is only taken for a block whose high band is well above
// the loop's own, because it costs a reading and almost nothing needs it.
func suspect(
	ctx context.Context,
	client Plays,
	bench sdk.Bench,
	signal []float32,
	got, block measured.Block,
	dry map[audio.Figure]float64,
	at string,
) (bool, error) {
	now := figuresOf(got.Figures)

	if _, bad := Squealing(now, dry, nil, endsInACabinet(block)); bad {
		return true, nil
	}

	// Nothing to be suspicious of. A block sitting where the reference sits is
	// a block that passed it.
	if now[audio.KeyHigh] <= dry[audio.KeyHigh]+apart {
		return false, nil
	}

	if err := client.Play(ctx, at); err != nil {
		return false, nil
	}

	quiet, err := sdk.Fingerprint(ctx, bench, make([]float32, len(signal)))
	if err != nil {
		return false, err
	}

	// Louder than the loop's own floor with nothing put in, which is a signal
	// the block is making rather than passing.
	return quiet.Level > got.Level-silent, nil
}

// endsInACabinet says a single-block chain is a low pass.
//
// A sweep measures one block alone, so the chain ends in whatever that block
// is. Only a cabinet makes the high band a one-way street, and an amplifier
// measured alone is legitimately brighter than what it was given.
//
// Which is why this is the harder half of the problem: the loudest amplifiers
// are exactly the blocks the invariant cannot be used on, so their readings
// are checked against the loop's own high band instead.
func endsInACabinet(
	block measured.Block,
) bool {
	return block.Category == catalog.CategoryCab
}

// baseline measures the loop with nothing in it.
//
// Without it a figure says nothing. 95 Hz is not what an equaliser does to a
// bass, it is what the bass already was, and an equaliser flat at its defaults
// passes it straight through.
//
// One bypassed block rather than none, because a chain has a minimum of one
// item and a rig with nothing in it is not a rig. Bypassed is the same signal
// path either way.
func baseline(
	ctx context.Context,
	w io.Writer,
	bench sdk.Bench,
	signal []float32,
	lib *measured.Library,
	work string,
	opts MeasureOptions,
) error {
	empty := measured.Block{
		ID:       "HD2_EQSimple3Band",
		Name:     "Simple EQ",
		Category: "eq",
	}

	at, err := compile(ctx, opts.Client, empty, work, false, opts.Headroom)
	if err != nil {
		return fmt.Errorf("building the empty loop: %w", err)
	}

	if err := opts.Client.Play(ctx, at); err != nil {
		return fmt.Errorf("loading the empty loop: %w", err)
	}

	got, err := sdk.Fingerprint(ctx, bench, signal)
	if err != nil {
		return err
	}

	lib.Baseline = got

	_, _ = fmt.Fprintf(w,
		"  baseline: centroid %.1f Hz, level %.2f dB, %.1f%% low\n",
		got.Centroid, got.Level, got.Low)

	drifted(w, got)

	return nil
}

// moved is how far a baseline may sit from the shipped one before it is worth
// saying so, in decibels of level and hertz of centroid.
//
// Generous, because this is for a rig somebody changed rather than for the
// wander between two takes, which is a hundredth of these. The old rig and the
// one that opened the loop differ by 50 hertz of centroid on a bare amplifier,
// and a nudged volume knob moves the level by more than three.
const (
	movedBy  = 3.0
	movedFar = 25.0
)

// drifted says when the empty loop no longer reads what the shipped library's
// did.
//
// The baseline is the calibration and nothing else is. It carries the whole gain
// structure of the rig in one measured number: which devices are playing and
// recording, how loud the computer's output is set, which cable goes where. None
// of those are recorded anywhere and every figure moves with them, so a campaign
// that starts from a different baseline is a campaign whose readings cannot be
// compared with the ones already committed.
//
// Said rather than refused, and printed before the sweep rather than after. A
// deliberate change of rig is the reason this is expected to fire, and somebody
// who has just rewired on purpose needs it to carry on. Somebody who has not
// needs to know before spending eighty minutes.
func drifted(
	w io.Writer,
	got measured.Figures,
) {
	had, err := measured.BuiltIn()
	if err != nil {
		return
	}

	level := got.Level - had.Baseline.Level
	centroid := got.Centroid - had.Baseline.Centroid

	if math.Abs(level) < movedBy && math.Abs(centroid) < movedFar {
		return
	}

	_, _ = fmt.Fprintf(w,
		"  [warn] the empty loop reads %+.1f dB and %+.0f Hz against the shipped\n"+
			"         library's own baseline, so this is a different measuring rig\n"+
			"         and these readings will not compare with the committed ones.\n"+
			"         Expected if you rewired on purpose. If not, check which\n"+
			"         devices --hardware named and whether the computer's output\n"+
			"         volume moved.\n",
		level, centroid)
}

// build compiles every preset up front, across every core.
//
// Compiling reads the catalog and writes a file. It needs no device, it is the
// slow half of a block's turn, and done inside the measuring loop it made each
// block cost three times what the measurement did.
func build(
	ctx context.Context,
	w io.Writer,
	client Quiets,
	want []measured.Block,
	work string,
	headroom float64,
) map[string]string {
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		out  = make(map[string]string, len(want))
		gate = make(chan struct{}, runtime.NumCPU())
	)

	for _, block := range want {
		wg.Add(1)

		go func() {
			defer wg.Done()

			gate <- struct{}{}
			defer func() { <-gate }()

			at, err := compile(ctx, client, block, work, true, headroom)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				out[block.ID] = short(err)

				return
			}

			out[block.ID] = at
		}()
	}

	wg.Wait()

	made := 0

	for _, at := range out {
		if strings.HasSuffix(at, ".hlx") {
			made++
		}
	}

	_, _ = fmt.Fprintf(w, "  built %d of %d presets\n", made, len(want))

	return out
}

// compile writes one block's rig and turns it into a preset.
func compile(
	ctx context.Context,
	client Quiets,
	block measured.Block,
	work string,
	enabled bool,
	headroom float64,
) (string, error) {
	name := strings.NewReplacer("/", "-", " ", "-").Replace(block.ID)
	spec := filepath.Join(work, name+".yaml")
	out := filepath.Join(work, name+".hlx")

	f, err := os.Create(spec) //nolint:gosec // a path this made up itself
	if err != nil {
		return "", fmt.Errorf("writing %s: %w", spec, err)
	}

	if err := plan.Write(f, planFor(block, enabled)); err != nil {
		_ = f.Close()

		return "", fmt.Errorf("writing %s: %w", spec, err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("writing %s: %w", spec, err)
	}

	if _, err := client.Compile(ctx, sdk.Compile{
		Plan: spec, Out: out, Existing: sdk.ReplaceExisting,
	}); err != nil {
		return "", err
	}

	// The chain's own output turned down before it is ever played. Every
	// measuring command comes through here, so this is the one place the
	// measuring rig's own feedback is dealt with.
	return quieter(ctx, client, out, work, name, headroom)
}

// planFor is a plan holding one block, named by model rather than by gear.
//
// A plan rather than a rig because a sweep is device work: it wants this one
// model and no other. 661 models share 468 names, so "Ampeg SVT" matches both
// of its channels, and a rig naming the gear would measure whichever the
// compiler picked and file it under both. Only a plan can pin the model.
//
// Built as the type and written by its own writer, rather than assembled as
// text. A cabinet called "'63 Spring" opens a YAML quote that nothing closes,
// and the whole document fails to parse at a line nowhere near the name. One
// block of six hundred and sixty one was lost to that.
func planFor(
	block measured.Block,
	enabled bool,
) plan.Plan {
	return plan.Plan{
		Name: "measure-" + strings.ToLower(
			strings.NewReplacer("_", "-", " ", "-", "'", "").Replace(block.ID)),
		Blocks: []plan.Block{{
			Model:   catalog.ModelID(block.ID),
			Enabled: enabled,
		}},
	}
}

// wanted is every block worth trying, in a stable order.
func wanted(
	cat *catalog.Catalog,
	category catalog.Category,
	done map[string]measured.Block,
) []measured.Block {
	out := make([]measured.Block, 0, len(cat.Blocks))

	for _, block := range cat.Blocks {
		if category != "" && block.Category != category {
			continue
		}

		// Four entries in the catalog are not blocks. `@dt`,
		// `@global_params`, `@powercab` and `@variax` are where a preset
		// keeps settings about the device rather than anything in the signal
		// path, and carry no name because nobody puts one in a chain.
		if block.Name == "" {
			continue
		}

		if _, had := done[string(block.ID)]; had {
			continue
		}

		out = append(out, measured.Block{
			ID:       string(block.ID),
			Name:     block.Name,
			Category: block.Category,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}

		return out[i].ID < out[j].ID
	})

	return out
}

// resume reads what a previous run got as far as.
func resume(
	lib *measured.Library,
	opts MeasureOptions,
) {
	f, err := os.Open(opts.Out) //nolint:gosec // the path is the operator's own
	if err != nil {
		return
	}

	defer func() { _ = f.Close() }()

	had, err := measured.Load(f)
	if err != nil {
		return
	}

	lib.Baseline = had.Baseline

	for id, block := range had.Blocks {
		// A refusal is worth another go when asked for. About one block in
		// eighty comes back saying the device stopped taking the message,
		// which is the chunk pacing giving up rather than anything about that
		// block, and the same one loads on the next pass.
		if opts.Retry && block.Refused != "" {
			continue
		}

		lib.Blocks[id] = block
	}
}

// reference reads the signal every block is measured against.
func reference(
	path string,
	seconds float64,
) ([]float32, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the operator's own
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	samples, rate, err := audio.Read(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return resample(samples, rate, seconds), nil
}

// resample puts the reference at the rate the loop runs at.
//
// Linear, which is coarse for audio and does not matter here: the same file
// goes through the same conversion before every reading, so whatever it does
// is in the baseline and cancels the moment two blocks are compared.
func resample(
	samples []float64,
	rate int,
	seconds float64,
) []float32 {
	want := int(seconds * reamp.Rate)
	out := make([]float32, 0, want)

	for i := range want {
		at := float64(i) * float64(rate) / reamp.Rate

		j := int(at)
		if j+1 >= len(samples) {
			break
		}

		f := at - float64(j)
		out = append(out, float32(samples[j]*(1-f)+samples[j+1]*f))
	}

	return out
}

// hash identifies the reference signal.
//
// A different one invalidates every number taken against the old one, the way
// changing a record invalidates the words derived from it.
func hash(
	path string,
) (string, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the operator's own
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	return hex.EncodeToString(sum.Sum(nil)), nil
}

// keep writes the library out.
func keep(
	path string,
	lib measured.Library,
) error {
	body, err := json.MarshalIndent(lib, "", " ")
	if err != nil {
		return fmt.Errorf("writing the measurements: %w", err)
	}

	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// say reports one block as it lands.
func reportBlock(
	w io.Writer,
	at, of int,
	block measured.Block,
) {
	if block.Refused != "" {
		_, _ = fmt.Fprintf(w, "  %4d/%d  %-40s refused\n", at, of, block.ID)

		return
	}

	mark := ""
	if block.Clipped {
		mark = "  CLIPPED"
	}

	_, _ = fmt.Fprintf(w, "  %4d/%d  %-40s centroid %8.1f  level %7.2f%s\n",
		at, of, block.ID, block.Centroid, block.Level, mark)
}

// done reports what a run got.
func done(
	w io.Writer,
	lib measured.Library,
	took time.Duration,
	out string,
) error {
	var read, refused, clip int

	for _, block := range lib.Blocks {
		switch {
		case block.Refused != "":
			refused++
		case block.Clipped:
			clip++
		default:
			read++
		}
	}

	_, err := fmt.Fprintf(w,
		"\n  %d measured, %d refused, %d clipped, in %.0f minutes\n"+
			"  wrote %s\n\n",
		read, refused, clip, took.Minutes(), out)

	return err
}

// short is the first line of an error, for a table that has one column for it.
func short(
	err error,
) string {
	line, _, _ := strings.Cut(err.Error(), "\n")

	return line
}
