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
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
	"github.com/retr0h/tonestack/pkg/sdk/reamp"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// MeasureOptions is what measuring every block needs to know.
type MeasureOptions struct {
	Client   *sdk.Client
	Dry      string
	Out      string
	Category string
	Seconds  float64
	Resume   bool
	Retry    bool
	Hardware string
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
	w io.Writer,
	ctx context.Context,
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

	bench, err := reamp.Open(opts.Hardware)
	if err != nil {
		return err
	}

	defer func() { _ = bench.Close() }()

	_, _ = fmt.Fprintf(w, "\n  %d blocks through %s, %.0fs each\n",
		len(want), bench.Name(), opts.Seconds)

	work, err := os.MkdirTemp("", "tonestack-blocks")
	if err != nil {
		return fmt.Errorf("making somewhere to build presets: %w", err)
	}

	defer func() { _ = os.RemoveAll(work) }()

	built := build(w, opts.Client, ctx, want, work)

	if err := baseline(w, ctx, bench, signal, &lib, work, opts); err != nil {
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
		} else if err := one(ctx, opts.Client, bench, signal, &lib, block, at); err != nil {
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
func one(
	ctx context.Context,
	client *sdk.Client,
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
	w io.Writer,
	ctx context.Context,
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

	at, err := compile(opts.Client, ctx, empty, work, false)
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
		"  baseline: centroid %.1f Hz, level %.2f dB, %.1f%% low\n\n",
		got.Centroid, got.Level, got.Low)

	return nil
}

// build compiles every preset up front, across every core.
//
// Compiling reads the catalog and writes a file. It needs no device, it is the
// slow half of a block's turn, and done inside the measuring loop it made each
// block cost three times what the measurement did.
func build(
	w io.Writer,
	client *sdk.Client,
	ctx context.Context,
	want []measured.Block,
	work string,
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

			at, err := compile(client, ctx, block, work, true)

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
	client *sdk.Client,
	ctx context.Context,
	block measured.Block,
	work string,
	enabled bool,
) (string, error) {
	name := strings.NewReplacer("/", "-", " ", "-").Replace(block.ID)
	spec := filepath.Join(work, name+".yaml")
	out := filepath.Join(work, name+".hlx")

	f, err := os.Create(spec) //nolint:gosec // a path this made up itself
	if err != nil {
		return "", fmt.Errorf("writing %s: %w", spec, err)
	}

	if err := rig.Write(f, rigFor(block, enabled)); err != nil {
		_ = f.Close()

		return "", fmt.Errorf("writing %s: %w", spec, err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("writing %s: %w", spec, err)
	}

	if _, err := client.Compile(ctx, sdk.Compile{
		Rig: spec, Out: out, Existing: sdk.ReplaceExisting,
	}); err != nil {
		return "", err
	}

	return out, nil
}

// rigFor is a rig holding one block, addressed by model rather than by name.
//
// By model because 665 models share 469 names: "Ampeg SVT" matches both of its
// channels, so a name would measure whichever the compiler picked and file it
// under both.
//
// Built as the contract's own type and written by its own writer, rather than
// assembled as text. A cabinet called "'63 Spring" opens a YAML quote that
// nothing closes, and the whole document fails to parse at a line nowhere
// near the name. One block of six hundred and sixty one was lost to that.
func rigFor(
	block measured.Block,
	enabled bool,
) rig.Spec {
	models := map[string]string{"HX Stomp": block.ID}
	entry := rig.ChainEntry{
		Role:   rig.Role(block.Category),
		Gear:   block.Name,
		Models: &models,
	}

	if !enabled {
		off := false
		entry.Enabled = &off
	}

	return rig.Spec{
		Schema: rig.SchemaName,
		ID: "measure-" + strings.ToLower(
			strings.NewReplacer("_", "-", " ", "-", "'", "").Replace(block.ID)),
		Subject: rig.Subject{
			Kind: rig.KindSound,
			Name: block.Name + " alone",
		},
		Instrument: rig.InstrumentBass,
		Chain:      []rig.ChainEntry{entry},
	}
}

// wanted is every block worth trying, in a stable order.
func wanted(
	cat *catalog.Catalog,
	category string,
	done map[string]measured.Block,
) []measured.Block {
	out := make([]measured.Block, 0, len(cat.Blocks))

	for _, block := range cat.Blocks {
		if category != "" && string(block.Category) != category {
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
			Category: string(block.Category),
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
