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

//go:generate go tool go.uber.org/mock/mockgen -source=tone_tune.go -destination=internal/mocks/tuner.gen.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// Builds turns a curated rig into a preset a device will load.
type Builds interface {
	Make(ctx context.Context, rigID string, out string, existing sdk.Existing) (sdk.Made, error)
}

// Tuner is what solving for knob positions needs: build a chain, put it in
// front of the device, move one control and read back what the chain is.
//
// Both kinds of control, because a chain has both and the loop handles them
// differently. Turns is the dials the solver moves together; Chooses is the
// lists it compares one setting at a time.
type Tuner interface {
	Builds
	Compiles
	ReadsFiles
	Plays
	Turns
	Chooses
	Switches
	Reads
}

// ErrNoTarget is a request that names nothing to aim at.
var ErrNoTarget = errors.New("name a genre to aim at")

// TuneOptions is what solving for one rig needs to know.
type TuneOptions struct {
	Client   Tuner
	Genres   Genres
	Bench    sdk.Bench
	Hardware string
	Dry      string
	Seconds  float64
	Takes    int
	// ID is the curated rig to tune, and Genre what to aim it at.
	ID    string
	Genre string
	// Corpus is the tree the genre's figures are measured from.
	Corpus string
	// Passes is how many times to solve before giving up on converging.
	Passes int
	// Tries is how many settings of a chain's lists to solve the dials from.
	//
	// One spends every reading on whichever setting read nearest before any
	// dial moved, which is not the same question as which setting a solve can
	// finish from. More than one costs a whole convergence each, so this is
	// where somebody in a hurry trades the better answer for the afternoon.
	Tries int
	// Nudge is how far a control is moved to read its slope, as a fraction of
	// its own range.
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
	// HeadroomTold says somebody named --headroom themselves.
	//
	// Carried because the default is rig-dependent and an explicit value is
	// not. A person naming -12 on an open rig may be bounding something this
	// cannot see, and overriding them would make the flag a suggestion.
	HeadroomTold bool
	// Headroom is how far the chain's own output is turned down before
	// anything is measured, in decibels, and wants to be negative.
	//
	// A property of the measuring rig rather than of the rig being measured.
	// The lead from the pedal's output to its own input makes the chain feed
	// itself, and enough gain around that loop oscillates.
	Headroom float64
	// Out is where the tuned chain goes, as a plan. Empty keeps nothing.
	Out string
	// Ask is a ToneSpec to append this round to, as a correction. Empty
	// records nothing, which loses the only account of what was asked for.
	Ask string
}

// Genres is what a genre measures as across its records.
type Genres interface {
	MeasuredGenres(ctx context.Context, corpus string) ([]audio.Genre, error)
}

// Tune solves a chain's controls for a target and says how close it got.
//
// The loop the whole project is pointed at: measure what the chain does now,
// read what each control does from here, solve for the moves that close the
// gap, apply them, and measure again. Three to five passes, each one
// measurement per control plus arithmetic.
//
// Nothing is written to a slot and nothing is stored. Every move is a live
// edit, so the answer lasts until the next preset is selected and the cost of
// trying is the time it takes to hear it.
func Tune(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
) error {
	// Pinned first, and it matters more here than anywhere: the loop reads a
	// slope, moves a dial and reads again, so a level that drifts mid-run is a
	// slope the solver will spend dials chasing.
	levelled(w, opts.Volume)
	opts.Headroom = trimFor(w, opts.Hardware, opts.Headroom, opts.HeadroomTold)

	if opts.Genre == "" {
		return ErrNoTarget
	}

	target, err := targetFor(ctx, opts)
	if err != nil {
		return err
	}

	aims := solve.Aims(target, nil)
	if len(aims) == 0 {
		return fmt.Errorf("%w: %q measures as nothing", ErrNoTarget, opts.Genre)
	}

	made, preset, asBuilt, err := built(ctx, opts)
	if err != nil {
		return err
	}

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

	lists := listsOf(made.Plan, cat)

	signal, err := reference(opts.Dry, opts.Seconds)
	if err != nil {
		return err
	}

	bench, release, err := benchFor(opts.Bench, opts.Hardware)
	if err != nil {
		return err
	}

	defer release()

	_, _ = fmt.Fprintf(w, "\n  %s aimed at %s, %d dials and %d lists through %s\n",
		opts.ID, opts.Genre, len(knobs), len(lists), bench.Name())

	// The loop's own wander, which is the floor under every tolerance. Without
	// it an axis the records happen to agree closely about gets a tolerance of
	// nearly nothing and swamps the solve: punk's records agree on the high
	// band to four decimal places, and the first run read that axis as 116
	// tolerances out and spent every control on it.
	floor, settled, err := steady(ctx, bench, signal, opts.Takes)
	if err != nil {
		return err
	}

	aims = solve.Aims(target, inCorpusScale(floor))
	aims[audio.KeyLevel] = solve.Aim{Want: settled, Tol: drift}

	// After the floor, because a nudge is measured in tolerances and a
	// tolerance is not known until the loop's own wander is.
	aims, err = nudged(w, opts, aims)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "  the loop wanders %.4f of a band and %.1fHz\n",
		inCorpusScale(floor)[audio.KeyLow], floor[audio.KeyCentroid])

	first, err := sdk.Fingerprint(ctx, bench, signal)
	if err != nil {
		return err
	}

	// Before a single control is moved. This loop applies what it solves for,
	// so a reading of the loop rather than of the chain does not produce a
	// wrong answer, it produces a chain turned to match one.
	if say, bad := Squealing(figuresOf(first), figuresOfDry(signal),
		inCorpusScale(floor), endsInACab(made.Plan, cat)); bad {
		return fmt.Errorf("%w: %s", ErrSquealing, say)
	}

	did, err := attempt(ctx, w, opts, bench, signal, preset, knobs, lists, aims, settled)
	if err != nil {
		return err
	}

	// Which block would close it, once the dials have stopped being the answer.
	// After the run rather than inside a pass, because "the chain is wrong" is a
	// conclusion about the run and one pass cannot reach it.
	if !did.arrived {
		missing(w, made.Plan, aims, did.read)
	}

	if err := keepTuned(ctx, w, opts, asBuilt); err != nil {
		return err
	}

	return written(w, opts, did)
}

// written appends this round to the ask, so what was asked for outlives the
// settings it produced.
func written(
	w io.Writer,
	opts TuneOptions,
	did round,
) error {
	if opts.Ask == "" {
		_, _ = fmt.Fprintf(w,
			"  Nothing recorded. --ask appends this round to a ToneSpec, "+
				"which is the only account of what was asked for.\n")

		return nil
	}

	if err := record(opts.Ask, opts.Genre, did.steps, did.residual, did.arrived); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w,
		"  [ok] appended a correction to %s, %d setting(s) moved and no "+
			"verdict yet\n", opts.Ask, len(did.steps))

	return nil
}

// keep writes the tuned chain out, so the answer survives the next preset
// selection.
//
// Read back off the device rather than written from what the solver believes it
// set. Those are two different claims and only one of them is checkable: a move
// the pedal refused, clamped or rounded is a move the solver still has in its
// own record, and what leaves here has to be what the hardware holds.
//
// A plan rather than a rig, and that is the whole point. A rig is the portable
// half and has nowhere to put a Helix answer: gear in signal order, named the
// way a musician names it. Every knob this loop just solved for lives on the
// plan, so writing the rig would export the chain and throw away the tuning,
// which is what the first version of this did.
//
// Nothing is written to a slot. A slot is flash and a burst of writes has
// corrupted a setlist, so tuning happens in the edit buffer and the answer
// leaves as a file. `presets compile --plan` is what puts it back.
func keepTuned(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	asBuilt string,
) error {
	if opts.Out == "" {
		_, _ = fmt.Fprintf(w,
			"\n  Nothing kept. The pedal holds this until the next preset is "+
				"selected; --out writes it as a plan.\n")

		return nil
	}

	read, err := opts.Client.Current(ctx, sdk.FormatRig)
	if err != nil {
		return err
	}

	// The measuring rig's own two changes undone. What the device is playing was
	// sent to USB alone and turned down 30dB so it would stop feeding itself
	// down the measuring lead, and neither of those is anything the solver
	// decided: keeping them writes a rig that is silent at the quarter-inch
	// socket and 30dB quiet everywhere else.
	//
	// Every dial the solve moved is kept, because the solve moved dials. Only
	// the one entry that is the measuring rig rather than the tone goes back.
	kept, err := asCompiled(ctx, w, opts.Client, read.Plan, asBuilt)
	if err != nil {
		return err
	}

	f, err := os.Create(opts.Out) //nolint:gosec // a path the caller named
	if err != nil {
		return fmt.Errorf("writing %s: %w", opts.Out, err)
	}

	if err := plan.Write(f, kept); err != nil {
		_ = f.Close()

		return fmt.Errorf("writing %s: %w", opts.Out, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", opts.Out, err)
	}

	_, _ = fmt.Fprintf(w,
		"\n  [ok] wrote %s, %d blocks as the device reports them\n",
		opts.Out, len(read.Plan.Blocks))

	return nil
}

// nudged moves the target by what somebody said after hearing it.
//
// Read off the ask rather than a flag of its own. A nudge is a decision about
// how something should sound, so it belongs beside the words and the genre in
// the ToneSpec, which is also what makes the next session start where this one
// finished rather than from the corpus again.
//
// Nothing to say leaves the target alone, which is the ordinary case: a first
// answer has not been heard yet, so there is nothing to move from.
func nudged(
	w io.Writer,
	opts TuneOptions,
	aims map[audio.Figure]solve.Aim,
) (map[audio.Figure]solve.Aim, error) {
	if opts.Ask == "" {
		return aims, nil
	}

	said, err := nudgesOn(opts.Ask)
	if err != nil {
		return nil, err
	}

	if len(said) == 0 {
		return aims, nil
	}

	of, err := sdk.Nudges(said)
	if err != nil {
		return nil, err
	}

	out := solve.Nudge(aims, of)

	// What moved and by how much, because a target that shifted silently is one
	// nobody can tell from a chain that drifted.
	for _, n := range of {
		was, ok := aims[n.Key]
		if !ok {
			_, _ = fmt.Fprintf(w,
				"  %q asks for %s, which this genre leaves free\n", n.Term, n.Key)

			continue
		}

		_, _ = fmt.Fprintf(w,
			"  %q moves %s from %.4g to %.4g, %g of a tolerance\n",
			n.Term, n.Key, was.Want, out[n.Key].Want, n.Steps)
	}

	return out, nil
}

// nudgesOn reads the nudges off a ToneSpec.
func nudgesOn(
	at string,
) ([]tone.Nudge, error) {
	f, err := os.Open(at) //nolint:gosec // a path the caller named
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", at, err)
	}

	spec, err := tone.Load(f)

	_ = f.Close()

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", at, err)
	}

	if spec.Nudges == nil {
		return nil, nil
	}

	return *spec.Nudges, nil
}

// targetFor is what the genre's records measure as, middle and spread.
func targetFor(
	ctx context.Context,
	opts TuneOptions,
) (audio.Across, error) {
	// The figures shipped with this binary when nobody named a tree. `just
	// generate` writes them from the same corpus, and measuring it again reads
	// fifteen bass stems and takes most of a minute against milliseconds for
	// the arithmetic around it.
	//
	// A caller who named a tree means that tree, and answering from what was
	// committed would be figures of other records than the ones asked about.
	if opts.Corpus == "" {
		got, ok := audio.ShippedGenre(opts.Genre)
		if !ok {
			return audio.Across{}, fmt.Errorf(
				"%w: nothing shipped with this binary measures %q",
				ErrNoTarget, opts.Genre)
		}

		return got.Across, nil
	}

	found, err := opts.Genres.MeasuredGenres(ctx, opts.Corpus)
	if err != nil {
		return audio.Across{}, fmt.Errorf("reading %s: %w", opts.Corpus, err)
	}

	for _, g := range found {
		if g.Slug == opts.Genre || g.Name == opts.Genre {
			return g.Across, nil
		}
	}

	return audio.Across{}, fmt.Errorf("%w: no records are tagged %q under %s",
		ErrNoTarget, opts.Genre, opts.Corpus)
}

// inCorpusScale puts a floor measured off the device into the units a target is
// stated in, which is the same translation figuresOf does for a reading.
func inCorpusScale(
	floor map[audio.Figure]float64,
) map[audio.Figure]float64 {
	out := make(map[audio.Figure]float64, len(floor))

	for key, got := range floor {
		if key.Share() {
			out[key] = got / perCent

			continue
		}

		out[key] = got
	}

	return out
}

// built compiles the rig and puts it in front of the device.
//
// Three answers: what was compiled, the preset being played, and the path of
// the compiled one. The last is there because the played one is not the answer
// to keep. It has been taken off the measuring loop, and that has to be undone
// before anything is written out.
func built(
	ctx context.Context,
	opts TuneOptions,
) (sdk.Made, string, string, error) {
	out := filepath.Join(os.TempDir(), opts.ID+".tune.hlx")

	made, err := opts.Client.Make(ctx, opts.ID, out, sdk.ReplaceExisting)
	if err != nil {
		return sdk.Made{}, "", "", err
	}

	// Off the measuring loop before it is ever played: the lead from the
	// pedal's output socket into its own input makes the chain feed itself, so
	// the output is sent to USB alone and turned down. Neither is the tone,
	// which is why neither survives into what is kept.
	playing, err := quieter(
		ctx, opts.Client, out, os.TempDir(), opts.ID, opts.Headroom)
	if err != nil {
		return sdk.Made{}, "", "", err
	}

	if err := opts.Client.Play(ctx, playing); err != nil {
		return sdk.Made{}, "", "", err
	}

	return made, playing, out, nil
}

// knobsOf is every dial in the chain the solver may turn.
//
// A list is left out, and that is a gap rather than a decision: a cabinet's
// microphone has no position between the third and the fourth, so it has no
// slope and cannot be solved for. It needs choosing instead, which is its own
// piece of work.
func knobsOf(
	made plan.Plan,
	cat *catalog.Catalog,
) []solve.Knob {
	var out []solve.Knob

	for _, b := range made.Blocks {
		block, ok := cat.Blocks[b.Model]
		if !ok {
			continue
		}

		order := wireOrder(cat, string(b.Model))

		for index, name := range order {
			spec, known := block.Params[name]
			if !known || spec.Type != catalog.ParamFloat {
				continue
			}

			at, _ := b.Params[name].Float()

			out = append(out, solve.Knob{
				Block:   b.Pos,
				Param:   index,
				Control: fmt.Sprintf("%s %s", b.Model, name),
				Setting: name,
				At:      at,
				Low:     spec.Min,
				High:    spec.Max,
			})
		}
	}

	return out
}

// converge runs the loop until the target is met or it stops improving.
//
// Stopping on "no longer improving" rather than on a pass count. The model is
// local, so a chain that cannot reach a target converges to a residual larger
// than the noise floor and then sits there, and that is a result: the gear is
// wrong for the sound, which is a real answer to somebody who owns that gear.
func converge(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	bench sdk.Bench,
	signal []float32,
	preset string,
	knobs []solve.Knob,
	aims map[audio.Figure]solve.Aim,
	settled float64,
) (round, error) {
	var best float64

	var did round

	for pass := 1; pass <= opts.Passes; pass++ {
		got, err := sdk.Fingerprint(ctx, bench, signal)
		if err != nil {
			return did, err
		}

		now := figuresOf(got)
		did.read = now

		// Level beside the pass, because it is the one figure no target
		// constrains and the one a solve can spend without being told not to.
		_, _ = fmt.Fprintf(w, "  level %.1fdB against %.1f settled\n",
			got.Level, settled)

		// Before reading any slope, because arriving needs no slopes and reading
		// them costs a reading per control. A chain of a dozen dials would spend
		// a minute and a half measuring what to do about a target it is already
		// inside.
		if at, arrived := solve.Reached(aims, now); arrived {
			aimed(w, pass, at, furthest(at.Residual))

			did.residual, did.arrived = at.Residual, true

			return did, nil
		}

		if err := slopes(ctx, bench, signal, opts, knobs, now); err != nil {
			return did, err
		}

		step, err := solve.Toward(knobs, aims, now)
		if err != nil {
			return did, err
		}

		worst := furthest(step.Residual)
		aimed(w, pass, step, worst)

		did.residual, did.arrived = step.Residual, step.Arrived
		did.steps = append(did.steps, step.Steps...)

		if step.Arrived {
			return did, nil
		}

		if pass > 1 && worst >= best {
			_, _ = fmt.Fprintf(w,
				"\n  stopped improving at %.1f tolerances out. The chain will not "+
					"reach this target.\n", worst)

			return did, nil
		}

		best = worst

		if err := land(ctx, w, opts, bench, signal, knobs, step.Steps, settled); err != nil {
			return did, err
		}

		// The chain is not reloaded between passes on purpose: the next pass
		// reads its slopes from where this one landed, which is the whole
		// reason the model being local is survivable.
		_ = preset
	}

	_, _ = fmt.Fprintf(w, "\n  %d passes and still %.1f tolerances out.\n",
		opts.Passes, best)

	return did, nil
}

// round is what one run of the loop did, for the correction it becomes.
type round struct {
	steps    []solve.Step
	residual map[audio.Figure]float64
	arrived  bool
	// read is what the chain measured on the last pass, in the corpus's units.
	//
	// Carried because residual is absolute: it says an axis is 2.7 tolerances
	// out and not which side of the target it sits on. Choosing a block to
	// close a gap needs the direction, and a shortlist built on the wrong sign
	// would suggest a bright block for a chain that is already too bright.
	read map[audio.Figure]float64
}

// aimed prints one pass: what moved and how far off the target still is.
func aimed(
	w io.Writer,
	pass int,
	step solve.Result,
	worst float64,
) {
	_, _ = fmt.Fprintf(w, "\n  pass %d, %.1f tolerances out\n", pass, worst)

	keys := make([]audio.Figure, 0, len(step.Residual))
	for key := range step.Residual {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return step.Residual[keys[i]] > step.Residual[keys[j]]
	})

	for _, key := range keys {
		mark := "  "
		if step.Residual[key] > 1 {
			mark = "->"
		}

		_, _ = fmt.Fprintf(w, "    %s %-12s %6.2f\n", mark, key, step.Residual[key])
	}

	for _, s := range step.Steps {
		_, _ = fmt.Fprintf(w, "       %-34s %8.3f to %8.3f\n",
			s.Control, s.At, s.To)
	}
}

// land applies a pass's moves and backs off if they muted the chain.
//
// The failure this exists for looked like the solver working. A first pass
// reached one tolerance out and got there partly by taking the amplifier's
// Master from 1.0 to 0.037, and the pass after it read 643 tolerances out
// because every figure was computed on hiss. Level is not one of the axes a
// corpus states, so nothing in the target stops a solve spending it, and two
// takes of silence agree to the last digit so the noise floor cannot catch it.
//
// Halved rather than refused, because the direction was right and only the
// distance was wrong, which is an ordinary line search and the same reason the
// loop takes several passes at all.
func land(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	bench sdk.Bench,
	signal []float32,
	knobs []solve.Knob,
	steps []solve.Step,
	settled float64,
) error {
	scale := 1.0

	for range backoffs {
		if err := apply(ctx, opts, knobs, scaled(steps, scale)); err != nil {
			return err
		}

		got, err := sdk.Fingerprint(ctx, bench, signal)
		if err != nil {
			return err
		}

		if got.Level >= settled-silent {
			return nil
		}

		scale /= 2

		_, _ = fmt.Fprintf(w,
			"       that muted the chain at %.1fdB against %.1f settled, "+
				"so half of it instead\n", got.Level, settled)
	}

	return fmt.Errorf("%w: every move that closes this target mutes the chain",
		ErrNoTarget)
}

// backoffs is how many times a pass may halve its moves before giving up.
const backoffs = 4

// drift is how far the chain's level may move from where it started, in dB.
//
// Chosen rather than measured, and the only figure in the loop that is. Every
// other tolerance comes from the spread across a genre's records, and no corpus
// states a level because a record's loudness is a mastering decision rather
// than a fact about the sound: the target has nothing to say here.
//
// Unconstrained, level is free, and a solve spends what is free. Asked for
// punk, the loop took an SV Beast's Master from 1.000 to 0.024 and then to
// 0.000 in two passes, turning the amplifier off to move the band shares a
// little, and every step cleared the mute guard on the way down.
//
// Six, because a change of about that much stops being heard as a different
// tone and starts being heard as a different volume, which is the point at
// which the solve has stopped answering the question it was asked. Wide enough
// that a gain or a drive may still be turned for its tone, narrow enough that
// the amplifier cannot be spent.
const drift = 6.0

// scaled is a pass's moves at a fraction of their length, measured from where
// each control started rather than from where the last attempt left it.
func scaled(
	steps []solve.Step,
	by float64,
) []solve.Step {
	if by == 1 {
		return steps
	}

	out := make([]solve.Step, 0, len(steps))

	for _, s := range steps {
		s.By *= by
		s.To = s.At + s.By
		out = append(out, s)
	}

	return out
}

// slopes reads what each control does from where the chain currently sits.
//
// One reading per control, and the control is put back before the next: a live
// edit writes nothing back, so a control left moved would have the next one
// measured on a chain this one skewed.
//
// Read here rather than taken from the shipped curves, because a slope is not a
// property of a control. The same Treble into a 4x12 and into a 1x15 are two
// different numbers, so the matrix belongs to the chain in hand. The shipped
// curves say which controls are worth putting in it.
func slopes(
	ctx context.Context,
	bench sdk.Bench,
	signal []float32,
	opts TuneOptions,
	knobs []solve.Knob,
	from map[audio.Figure]float64,
) error {
	for i := range knobs {
		k := &knobs[i]

		step := (k.High - k.Low) * opts.Nudge

		// Away from whichever stop it is against, so a control already at its
		// top reads a slope going down rather than none at all.
		to := k.At + step
		if to > k.High {
			to = k.At - step
			step = -step
		}

		if err := move(ctx, opts.Client, *k, to); err != nil {
			return err
		}

		got, err := sdk.Fingerprint(ctx, bench, signal)
		if err != nil {
			return err
		}

		k.Slope = map[audio.Figure]float64{}

		for key, was := range from {
			k.Slope[key] = (figuresOf(got)[key] - was) / step
		}

		if err := move(ctx, opts.Client, *k, k.At); err != nil {
			return err
		}
	}

	return nil
}

// apply moves every control the solve asked for, and records where it landed.
func apply(
	ctx context.Context,
	opts TuneOptions,
	knobs []solve.Knob,
	steps []solve.Step,
) error {
	// By block and param together. A cabinet's fourth parameter and an
	// amplifier's fourth are both 4, so keying on the param alone moved one and
	// recorded it against the other.
	at := make(map[solve.Where]float64, len(steps))

	for _, s := range steps {
		if err := move(ctx, opts.Client, s.Knob, s.To); err != nil {
			return err
		}

		at[s.Where()] = s.To
	}

	for i := range knobs {
		if to, moved := at[knobs[i].Where()]; moved {
			knobs[i].At = to
		}
	}

	return nil
}

// move sets one dial on the running chain, addressed the way a preset records
// its block rather than the way the wire numbers it.
func move(
	ctx context.Context,
	client Turns,
	k solve.Knob,
	to float64,
) error {
	return client.Turn(ctx, sdk.Control(k.Block, k.Param), float32(to))
}

// figuresOf keys a device reading the way a target is keyed, and puts the two
// on the same scale.
//
// The scale is the whole point of this function and it is a trap that would not
// have failed. A sweep reports a band share as a percentage summing to a
// hundred and a corpus reports it as a fraction summing to one: this cabinet
// reads `low: 99.23` where punk's records say `0.974`. Solving one against the
// other would have chased a target a hundred times out and reported moves with
// total confidence.
//
// Proven by the sums rather than by comparing two sounds, because two sounds
// legitimately differ: 99.23 + 0.77 + 0.0006 is a hundred, and 0.974 + 0.0275 +
// 0.0013 is one. Harmonics is the same pair of conventions, stated in
// audio.Profile as "the share of energy above the fundamental, from zero to"
// one against a sweep reading 5.39.
//
// Converted into the corpus's units rather than the other way about, so
// everything past this point speaks one convention and a device reading is the
// only thing that has to be translated.
func figuresOf(
	got measured.Figures,
) map[audio.Figure]float64 {
	out := map[audio.Figure]float64{
		audio.KeyLow:      got.Low,
		audio.KeyMid:      got.Mid,
		audio.KeyHigh:     got.High,
		audio.KeyCentroid: got.Centroid,
		audio.KeyLevel:    got.Level,
	}

	// The five that a reading may not have an answer for. Absent rather than
	// zero, because a transient needs a note starting and a figure of zero
	// would be solved for as though it had been measured.
	// `at` rather than `got` for the loop variable, because the map it ranges
	// over is built from the `got` parameter and a second `got` inside the body
	// shadows it. Harmless while nothing in the body needs the reading itself,
	// and a compile error the first time something does.
	for key, at := range map[audio.Figure]*float64{
		audio.KeyTransient: got.Transient,
		audio.KeyDecay:     got.Decay,
		audio.KeyDynamics:  got.Dynamics,
		audio.KeyHarmonics: got.Harmonics,
		audio.KeyLean:      got.Lean,
	} {
		if at == nil {
			continue
		}

		out[key] = *at
	}

	// The divide happens once, over whatever audio.Figure.Share names, rather
	// than beside each field. Spelled out per field it was four lists in four
	// packages, and a fifth share added to the sweep would have been divided in
	// none of them.
	for key, at := range out {
		if key.Share() {
			out[key] = at / perCent
		}
	}

	return out
}

// perCent is what a sweep reports a share as, against the fraction a corpus
// reports the same share as.
const perCent = 100

// furthest is how far the worst axis still is, in its own tolerances.
func furthest(
	residual map[audio.Figure]float64,
) float64 {
	var out float64

	for _, off := range residual {
		out = math.Max(out, off)
	}

	return out
}
