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

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// This file turns the shortlist into something somebody can act on.
//
// The shortlist has been printed since the loop could say a chain was wrong
// rather than its settings, and it stops there on purpose: a block measured on
// its own is not that block in this chain, because the same control reads
// differently with a cabinet in front of it. So the loop proposes and a person
// decides, and the sentence under the table says so.
//
// What was missing is the half between proposing and deciding. A table in a
// terminal is gone at the next command, so acting on it meant somebody reading
// five rows, picking one, finding the gear's real name, working out which side
// of the amplifier it belongs, and hand-editing a document. Nobody did that, and
// the loop went on reporting the same five rows every run.
//
// So it writes the document instead. The rig the run was aimed at, plus the
// block the measurement ranked first, placed where the corpus puts that kind,
// carrying evidence that says it arrived by measurement rather than by anybody
// naming it. Reviewable, rejectable, and a file rather than scrollback.
//
// Still not a choice. Nothing is written to the device and nothing overwrites
// the rig it came from: the amended document goes where `--extend` says, and
// building it is a separate thing somebody does on purpose.

// grammarOf is the corpus statistics, as a seam.
//
// A package-level var the way `audio.Write` takes `creates` and `closes`: the
// embedded copy is a compile-time constant and cannot be damaged at run time,
// so a build that shipped a truncated one is the only thing that reaches the
// error path and a test is the only way to be that build.
var grammarOf = corpus.BuiltIn

// extend writes the rig the run aimed at, with the block the measurement asked
// for added to its chain.
//
// Does nothing without `--extend`, because a run that writes files nobody asked
// for is a run somebody stops using.
func extend(
	ctx context.Context,
	w io.Writer,
	opts TuneOptions,
	made plan.Plan,
	aims map[audio.Figure]solve.Aim,
	now map[audio.Figure]float64,
	cat *catalog.Catalog,
) error {
	if opts.Extend == "" {
		return nil
	}

	// Nothing missing, nothing to propose, and no reason to read a document.
	// Checked before the rig is fetched because a run that met its target has
	// no use for either.
	if len(unreached(aims, now)) == 0 {
		_, _ = fmt.Fprintf(w,
			"  nothing measured moves what is missing, so there is no block to add\n")

		return nil
	}

	// Once, here, rather than per candidate inside the ranking. Every other
	// consumer of the statistics in this tree is handed them, and the call
	// gunzips and parses 33KB, which the shortlist loop was doing per block it
	// considered.
	//
	// Before the document, so a run that cannot read the statistics says so
	// without fetching a rig it has no use for.
	stats, err := grammarOf()
	if err != nil {
		return fmt.Errorf("reading the corpus statistics: %w", err)
	}

	// The document's instrument, not the amplifier's. This asked the amplifier
	// until 2026-10-17, on the reasoning that the corpus keys its grammar that
	// way, which is right about the label and wrong about which population a
	// player belongs to.
	//
	// Dusty Hill plays bass through a Marshall Super Lead 100, a guitar head,
	// because ZZ Top ran the same Marshall rig for his bass as for the guitars.
	// Asking the amplifier called his chain a guitar chain, so the corpus filled
	// it from guitar habits and a loudness match read guitar cabinets, for a
	// bass player. The corpus measurer has to ask the amplifier because a preset
	// carries no document. A build has one, and it says bass.
	doc, err := opts.Client.Rig(ctx, opts.ID)
	if err != nil {
		return err
	}

	g := stats.Grammar[string(doc.Rig.Instrument)]

	pick, ok := ranked(made, aims, now, cat, g)
	if !ok {
		_, _ = fmt.Fprintf(w,
			"  nothing measured moves what is missing, so there is no block to add\n")

		return nil
	}

	// What the block's own sweep says to set, where there is one. Reported
	// whichever way it comes back, because "it arrives at its defaults" is a
	// fact about the document somebody is about to read.
	at, set := startAt(w, opts, pick, aims, now)

	amended, err := amend(doc, pick, cat, opts, g, at, set)
	if err != nil {
		return err
	}

	f, err := os.Create(opts.Extend) //nolint:gosec // a path the caller chose
	if err != nil {
		return fmt.Errorf("creating %s: %w", opts.Extend, err)
	}

	if err := tone.Write(f, amended); err != nil {
		_ = f.Close()

		return fmt.Errorf("writing %s: %w", opts.Extend, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", opts.Extend, err)
	}

	_, _ = fmt.Fprintf(w,
		"\n  [ok] wrote %s, which is %s with %s added %s the amplifier\n"+
			"       Build it and tune it again to find out whether it helps: a block\n"+
			"       measured alone is not that block in this chain.\n",
		opts.Extend, opts.ID, pick.Block.Name,
		side(g, pick.Block.Category))

	return nil
}

// ranked is the block the measurement puts first, for the axes still out.
//
// The same call `missing` prints from, asked for one rather than five, so the
// table somebody reads and the block that gets written can never disagree.
// How many of a kind the chain may hold comes from the corpus, because the
// first rule here was that it may hold one.
//
// An amplifier's reading moves the centroid further than almost anything
// measured, so amplifiers win the ranking outright, and acting on that put a
// second amp after the cabinet: `EVH 5150III` appended to a chain that already
// had an Ampeg SVT. Refusing every kind the chain already held stopped that,
// and it also stopped the shape `05C Ultimate Amp` is built on, a drive on each
// side of the amplifier. One rule cannot be right for both, because a second
// drive is a chain people build and a second amplifier in series is not.
//
// So the corpus answers it. `Repeats` is the share of chains holding one of a
// kind that hold two of it in one series run, and the kinds separate cleanly:
//
//	bass     comp 26.7%  drive 19.6%  mod 14.8%  delay 12.0%  eq 7.7%
//	         reverb 4.3%  pitch 4.3%  cab 0.0%   amp 1.2%
//	guitar   eq 32.1%  mod 20.1%  drive 19.9%  delay 17.9%  reverb 9.1%
//	         comp 7.4%  amp 7.3%  pitch 3.6%  cab 0.3%
//
// A series run, counted that way on purpose: two cabinets either side of a Y
// split is a blend people build and reads as a repeat if the split is ignored,
// which is how a first measurement of this put bass cab at 16.3% instead of
// zero. A chain here is flat, so the evidence has to be about flat chains.
//
// The amplifier needs no threshold to be refused. `tonal` keeps it out of the
// grammar altogether, because the amp is the thing every other block is
// positioned around rather than a neighbour of itself, so `Repeats` reads zero
// for it and zero means do not repeat. The threshold below only has to separate
// the cabinet from the kinds above it, and the gap it sits in is wide: nothing
// measured falls between 0.3% and 3.6%.
func ranked(
	made plan.Plan,
	aims map[audio.Figure]solve.Aim,
	now map[audio.Figure]float64,
	cat *catalog.Catalog,
	g corpus.Grammar,
) (measured.Suggestion, bool) {
	lib, err := measured.BuiltIn()
	if err != nil {
		return measured.Suggestion{}, false
	}

	want := unreached(aims, now)
	if len(want) == 0 {
		return measured.Suggestion{}, false
	}

	// More than one, because the first few are filtered below and asking for
	// one would leave nothing to fall back to.
	got := measured.Shortlist(lib, want, inTheChain(made), shortlisted)

	runs := inSeries(made, cat)

	for _, one := range got {
		if room(g, runs, one.Block.Category) {
			return one, true
		}
	}

	return measured.Suggestion{}, false
}

// repeatable is how often a kind has to be doubled in the corpus before this
// will double it.
//
// One chain in fifty, which is a low bar on purpose. The ranking already
// decides whether a block helps; this only has to refuse the shapes nobody
// builds, and those sit at 0.3% and below while everything attested sits at
// 3.6% and above. A number anywhere in that gap behaves identically, so the
// midpoint is the one that will still be right if either side moves.
const repeatable = 0.02

// holdsTwo is the most of one kind a chain may be given.
//
// Two, because `Repeats` counts chains holding two or more without separating
// two from three. Reading evidence for "at least two" as licence for a third
// would be claiming something nobody counted.
const holdsTwo = 2

// inSeries counts how many of each kind the chain already holds, per run.
//
// Keyed the same way the corpus counts, so what is measured and what is checked
// against it are the same question.
func inSeries(
	made plan.Plan,
	cat *catalog.Catalog,
) map[catalog.Category]int {
	most := map[catalog.Category]int{}

	for _, counts := range byRun(made, cat) {
		for c, n := range counts {
			if n > most[c] {
				most[c] = n
			}
		}
	}

	return most
}

// byRun groups a chain's blocks by processor and side of a split.
func byRun(
	made plan.Plan,
	cat *catalog.Catalog,
) map[[2]int]map[catalog.Category]int {
	out := map[[2]int]map[catalog.Category]int{}

	for _, b := range made.Blocks {
		one, known := cat.Block(b.Model)
		if !known {
			continue
		}

		at := [2]int{b.DSP, b.Path()}
		if out[at] == nil {
			out[at] = map[catalog.Category]int{}
		}

		out[at][one.Category]++
	}

	return out
}

// room reports whether the chain may take another of a kind.
//
// An instrument the corpus never measured has an empty grammar and refuses
// every repeat, which is the same direction `side` takes for a kind it never
// counted: a chain left as it is beats one given a block on no evidence.
func room(
	g corpus.Grammar,
	runs map[catalog.Category]int,
	c catalog.Category,
) bool {
	has := runs[c]
	if has == 0 {
		return true
	}

	if has >= holdsTwo {
		return false
	}

	return g.Categories[c].Repeats() >= repeatable
}

// amend puts the block into the document's chain, on the side the corpus puts
// that kind.
//
// Gear by the name Line 6 say the block is based on, falling back to the block's
// own name. A document names gear a person recognises and the catalog resolves
// it; writing the model identifier would be a document that only this tool can
// read.
func amend(
	doc sdk.Rig,
	pick measured.Suggestion,
	cat *catalog.Catalog,
	opts TuneOptions,
	g corpus.Grammar,
	at measured.Starting,
	set bool,
) (tone.Spec, error) {
	block, known := cat.Block(catalog.ModelID(pick.Block.ID))
	if !known {
		return tone.Spec{}, fmt.Errorf(
			"%w: the catalog has no %s", ErrNoTarget, pick.Block.ID)
	}

	gear := block.BasedOn
	if gear == "" {
		gear = block.Name
	}

	kind := tone.EvidenceCorpus
	note := fmt.Sprintf(
		"added because the dials ran out: aimed at %s, the axes still out were "+
			"%s, and of the 661 measured blocks this one's own reading moves them "+
			"furthest in the right direction. Measured alone at its defaults, "+
			"which is not this chain.",
		solvingFor(opts), stillOut(pick))

	entry := rig.ChainEntry{
		Gear:     gear,
		Role:     rig.Role(pick.Block.Category),
		Evidence: &[]tone.Evidence{{Kind: kind, Note: &note}},
	}

	// Partial on purpose, which the contract's own field says is allowed: one
	// control the sweep had something to say about, and the rest answered from
	// corpus medians the way every other build answers them.
	if set {
		value, known := param(cat, catalog.ModelID(pick.Block.ID), at)
		if known {
			entry.Controls = &map[string]catalog.Setting{at.Control: value}
		}
	}

	out := doc.Rig
	out.Chain = placed(
		out.Chain, entry, side(g, pick.Block.Category) == "before")

	return tone.Spec{Schema: tone.SchemaToneSpec, Id: doc.ID, Rig: out, Ask: doc.Ask}, nil
}

// placed puts an entry on the correct side of the amplifier.
//
// The same rule `fill` follows for a block the corpus expected and `beforeAmp`
// for one a word asked for, because a block arriving because a measurement
// ranked it belongs wherever the other two would have put it. Position is not
// decoration: drive ahead of an amplifier overdrives its input and drive after
// it does something else.
func placed(
	chain []rig.ChainEntry,
	add rig.ChainEntry,
	before bool,
) []rig.ChainEntry {
	if !before {
		return append(chain, add)
	}

	at := 0

	for i, one := range chain {
		if one.Role == rig.RoleAmp {
			at = i

			break
		}
	}

	out := make([]rig.ChainEntry, 0, len(chain)+1)
	out = append(out, chain[:at]...)
	out = append(out, add)

	return append(out, chain[at:]...)
}

// side says which end of the amplifier a kind of block belongs, from the corpus.
//
// A kind the corpus never counted goes after, which is the safer end: a block
// ahead of an amplifier changes what the amplifier is fed, and guessing that
// from no evidence would be deciding something nobody measured. An instrument
// nothing measured has an empty grammar and answers the same way.
func side(
	g corpus.Grammar,
	c catalog.Category,
) string {
	s, counted := g.Categories[c]
	if !counted || s.BeforeAmp() < 0.5 {
		return "after"
	}

	return "before"
}

// stillOut names the axes the chain could not reach, for the note.
func stillOut(
	pick measured.Suggestion,
) string {
	out := ""

	for _, key := range measured.Named() {
		if _, moves := pick.Moves[key]; !moves {
			continue
		}

		if out != "" {
			out += ", "
		}

		out += string(key)
	}

	return out
}

// startAt is where the proposed block's own sweep says to start one control.
//
// Read off disk rather than out of the binary, because the curves are not
// packed into it: `just pack-measured` embeds the fingerprints and the sweeps
// stay under resources/sweeps/. So a run from an installed release has no
// curves and says the block arrives at its defaults, which is true.
//
// Says what it found either way. A document nobody can tell the provenance of
// is the thing this whole command exists to avoid, and "its controls are at
// the catalog's defaults" is as much a fact about it as a value would be.
func startAt(
	w io.Writer,
	opts TuneOptions,
	pick measured.Suggestion,
	aims map[audio.Figure]solve.Aim,
	now map[audio.Figure]float64,
) (measured.Starting, bool) {
	if opts.Sweeps == "" {
		return measured.Starting{}, false
	}

	at := filepath.Join(opts.Sweeps, pick.Block.ID+".json")

	f, err := os.Open(at) //nolint:gosec // a path built from the catalog
	if err != nil {
		_, _ = fmt.Fprintf(w,
			"  %s has no committed sweep, so its controls arrive at the catalog's\n"+
				"  defaults. `measure controls --model %s` is what would change that.\n",
			pick.Block.Name, pick.Block.ID)

		return measured.Starting{}, false
	}

	curves, err := measured.LoadCurves(f)

	_ = f.Close()

	if err != nil {
		_, _ = fmt.Fprintf(w,
			"  %s: %s, so its controls arrive at the catalog's defaults\n",
			at, err)

		return measured.Starting{}, false
	}

	got, found := measured.StartAt(
		curves, pick.Block.Category,
		unreached(aims, now), holding(aims, now), valveGuts)
	if !found {
		_, _ = fmt.Fprintf(w,
			"  %s is swept, and no position of any control moves what is missing\n"+
				"  convincingly enough to write down, so its controls arrive at the\n"+
				"  catalog's defaults.\n", pick.Block.Name)

		return measured.Starting{}, false
	}

	_, _ = fmt.Fprintf(w,
		"  %s arrives with %s at %g, which its own sweep moves %s by %.0f times\n"+
			"  the noise beside it. Swept alone, so it is a starting position\n"+
			"  rather than a setting.\n",
		pick.Block.Name, got.Control, got.Value, got.Figure, got.Noises)

	return got, true
}

// param turns a swept position into a value of the control's own kind.
//
// The kind has to survive, because a document is read through a route that
// turns a float of 6.0 into the integer 6 and the device reads those as
// different settings. The catalog is what knows which a control is.
//
// False for a control the catalog does not carry under that name, which is a
// sweep and a catalog that disagree. Dropped rather than written, because a
// value the catalog would refuse is a document that will not build.
func param(
	cat *catalog.Catalog,
	of catalog.ModelID,
	at measured.Starting,
) (catalog.Setting, bool) {
	block, known := cat.Block(of)
	if !known {
		return catalog.Setting{}, false
	}

	spec, carried := block.Params[at.Control]
	if !carried {
		return catalog.Setting{}, false
	}

	switch spec.Type {
	case catalog.ParamFloat:
		return catalog.Set(catalog.Float(at.Value)), true
	case catalog.ParamInt:
		return catalog.Set(catalog.Int(int64(at.Value))), true
	}

	// A bool and an enum are both lists, and a swept position is a number
	// along one. Which number means which setting is the enum's own business
	// and not something to guess at from a sweep's axis.
	return catalog.Setting{}, false
}
