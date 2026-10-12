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

	pick, ok := ranked(made, aims, now, cat)
	if !ok {
		_, _ = fmt.Fprintf(w,
			"  nothing measured moves what is missing, so there is no block to add\n")

		return nil
	}

	doc, err := opts.Client.Rig(ctx, opts.ID)
	if err != nil {
		return err
	}

	instrument := string(doc.Rig.Instrument)

	amended, err := amend(doc, pick, cat, opts, instrument)
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
		side(pick.Block.Category, instrument))

	return nil
}

// ranked is the block the measurement puts first, for the axes still out.
//
// The same call `missing` prints from, asked for one rather than five, so the
// table somebody reads and the block that gets written can never disagree.
// A kind the chain already holds is passed over, which is the rule `fill`
// follows for the blocks the corpus expects and the one this got wrong first.
// An amplifier's reading moves the centroid further than almost anything
// measured, so amplifiers win the ranking outright, and acting on that put a
// second amp after the cabinet: `EVH 5150III` appended to a chain that already
// had an Ampeg SVT. Two amplifiers in series is not a chain anybody would build,
// and the shortlist cannot know that because it ranks one block's reading rather
// than a chain.
func ranked(
	made plan.Plan,
	aims map[audio.Figure]solve.Aim,
	now map[audio.Figure]float64,
	cat *catalog.Catalog,
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

	held := map[catalog.Category]bool{}
	for _, b := range made.Blocks {
		if one, known := cat.Block(b.Model); known {
			held[one.Category] = true
		}
	}

	for _, one := range got {
		if held[one.Block.Category] {
			continue
		}

		return one, true
	}

	return measured.Suggestion{}, false
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
	instrument string,
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

	out := doc.Rig
	out.Chain = placed(
		out.Chain, entry, side(pick.Block.Category, instrument) == "before")

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
// from no evidence would be deciding something nobody measured.
func side(
	c catalog.Category,
	instrument string,
) string {
	stats, err := corpus.BuiltIn()
	if err != nil {
		return "after"
	}

	g, seen := stats.Grammar[instrument]
	if !seen {
		return "after"
	}

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
