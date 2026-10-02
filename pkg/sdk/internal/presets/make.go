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
// Package presets turns curated knowledge into a preset file.
package presets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/internal/fileslots"
	"github.com/retr0h/toneharness/pkg/sdk/internal/rigs"
	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// MakeOptions says what to build and where to put it.
type MakeOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// RigID names the curated knowledge to build from.
	RigID string
	// Source is where rigs are read from. The zero value is the rigs that
	// ship.
	Source rigs.Source
	// RigPath is a document to build from instead of a RigID.
	//
	// Exactly one of RigID and RigPath. A rig in a directory is found by
	// identifier and its ask is found with it; a rig somebody has in hand,
	// from `tone build` or from somebody who sent it, has a path and no
	// identifier to look up.
	//
	// One file holds both halves, so naming the rig names the ask with it.
	RigPath string
	// StatsPath is measured corpus statistics. Empty means the ones built
	// into this binary.
	StatsPath string
	// SetupPath is what the person has. Empty builds for the record rather
	// than for them.
	SetupPath string
	// OutputPath is where the preset is written.
	OutputPath string
	// Existing is what happens to a file already at OutputPath. The zero
	// value replaces it.
	Existing result.Existing
}

// Make builds a preset from a rig and writes it, reporting what it chose.
//
// Reporting the chain matters as much as writing the file. A generated preset
// is a set of decisions, and a wrong amp should be visible before anyone plugs
// in rather than after.
func Make(
	ctx context.Context,
	opts MakeOptions,
) (result.Made, error) {
	if err := ctx.Err(); err != nil {
		return result.Made{}, err
	}

	known, err := opts.known()
	if err != nil {
		return result.Made{}, err
	}

	// What the person has, which is what makes the answer fit them rather
	// than the record. A path nobody gave is not an error: building for the
	// record is what this did before a Setup could be named, and still the
	// right answer for somebody who did not write one.
	held, err := setupAt(opts.SetupPath)
	if err != nil {
		return result.Made{}, err
	}

	rec, intent := known.Rig, intentOf(known.Ask, held)

	cat, err := opts.catalog(ctx)
	if err != nil {
		return result.Made{}, err
	}

	// Statistics are an improvement on the catalog's defaults, not a
	// requirement, but somebody who named a file asked for those ones.
	// Building without them would hand back a more generic preset than the
	// one asked for, and say nothing about it.
	stats, err := corpus.Open(opts.StatsPath)
	if err != nil {
		return result.Made{}, err
	}

	spec, added, moved, playing, err := opts.compiler().Resolve(
		known.ID, rec, intent, cat, stats)
	if err != nil {
		return result.Made{}, err
	}

	// The device the catalog describes, not whichever one this was written
	// against: an HX Stomp holds eight blocks on one path and a Helix Floor
	// holds 29 across two.
	limits := plan.LimitsFor(cat.Device)

	// Kept, because the fit renumbers: a rig names the block its pedal moves
	// by where that block sits in the chain the rig wrote, and after the fit
	// that number means something else.
	before := append([]plan.Block(nil), spec.Blocks...)

	spec = opts.compiler().Fit(spec, cat, limits)

	// The plan, not the rig. A footswitch names the block it acts on by where
	// that block sits, and the fit is what moves it; the rig names gear by role
	// and has no number the fit could invalidate.
	spec = compile.Refit(spec, before, spec.Blocks)

	if err := plan.Validate(cat, spec, limits); err != nil {
		return result.Made{}, fmt.Errorf(
			"the chain this rig describes will not load: %w", err)
	}

	doc := build(cat.DeviceID, spec)

	// Against the chain as built rather than as the rig wrote it: filling
	// and fitting add and drop blocks, and a section can only turn on what
	// made it into the preset.
	if err := opts.compiler().Sections(doc, rec, spec, spec.Blocks, cat); err != nil {
		return result.Made{}, err
	}

	// After the fit, because the fit decides which position a block ends up
	// at: a move names a role and the role's block only has a position once
	// the chain is laid out.
	if err := opts.compiler().Moves(&spec, rec, spec.Blocks, cat); err != nil {
		return result.Made{}, err
	}

	opts.compiler().Controllers(doc, spec, spec.Blocks, cat)

	opts.compiler().Footswitches(doc, spec, cat)

	if err := write(opts.OutputPath, doc, opts.Existing); err != nil {
		return result.Made{}, err
	}

	return result.Made{
		Plan:       spec,
		Added:      addedFrom(added),
		Moved:      movedFrom(moved),
		Unfamiliar: unfamiliar(intent),
		Playing:    result.Playing{Terms: playing.Terms, Said: playing.Said},
		Path:       opts.OutputPath,
	}, nil
}

// build puts a chain into a preset the device would recognise.
//
// Written into an untouched preset rather than assembled from nothing: a
// device expects inputs, outputs, a split and a join around a chain, and
// 98.6% of real presets carry them. One built without them is unlike anything
// the hardware has ever written.
func build(
	deviceID int,
	spec plan.Plan,
) *preset.Document {
	// The blank is embedded and covered by its own test, so reading it cannot
	// fail here. SetSpec refuses a parameter named like a block attribute,
	// and a resolved chain cannot hold one because the catalog excludes them.
	doc, _ := preset.Blank()
	doc.Data.Device = deviceID
	_ = doc.SetSpec(spec)

	return doc
}

// write puts the preset on disk.
//
// The document is rendered to memory first and the whole file put in place at
// once, so a document that will not encode or a write that stops partway
// leaves no half a preset behind. existing says what happens to a file already
// at path.
func write(
	path string,
	doc *preset.Document,
	existing result.Existing,
) error {
	var buf bytes.Buffer

	if err := preset.Write(&buf, doc); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return fileslots.Save(path, buf.Bytes(), existing)
}

// intentOf is what the ask contributes to the build.
//
// A rig with no ask beside it is legal and ordinary: somebody's own directory
// holds rigs they wrote, and nothing obliges them to write down the ask that
// produced one. The zero Intent is the right answer for that rather than an
// error, because a rig already carries the settings somebody applied.
//
// Everything on a ToneSpec is optional and arrives as a pointer, so each field
// is taken only where the ask actually said it.
func intentOf(
	ask *tone.Ask,
	held *tone.Setup,
) compile.Intent {
	out := compile.Intent{}

	// How this person plays, which is the one thing here that comes off the
	// Setup rather than the ask. It stands on its own: a rig with no ask
	// beside it still gets compensated for a right hand, because the rig says
	// how it was played and the Setup says how this person does.
	if held != nil && held.Technique != nil {
		out.Playing = compile.Playing{Attack: string(held.Technique.Attack)}
	}

	if ask == nil {
		return out
	}

	if ask.Words != nil {
		out.Words = make([]compile.Word, 0, len(*ask.Words))

		for _, w := range *ask.Words {
			// The evidence travels with the word. A figure measured off a
			// record and held against what other players read is what decides
			// how far the word moves its control, and a word arriving without
			// it moves the whole step.
			word := compile.Word{Term: w.Term}
			if w.Evidence != nil {
				word.Evidence = *w.Evidence
			}

			out.Words = append(out.Words, word)
		}
	}

	// What the ask says it is for, which a genre's figures have to agree with.
	// Empty leaves it to the Setup and claims nothing either way.
	wants := ""
	if ask.Instrument != nil {
		wants = string(*ask.Instrument)
	}

	// A genre's words, after the ask's own, so anything somebody wrote by hand
	// outranks what a measurement produced. Both carry their figures, so
	// weightOf sizes each by how far it actually sits from the rest.
	// Every genre the ask names, in the order it names them, so a sound that is
	// both punk and pop-punk gets what each earns. The words carry their own
	// figures, so a term two genres both earn is weighed twice rather than
	// counted once, which is right: two populations agreeing is more evidence
	// than one.
	for _, named := range ask.Genre {
		if named == "" {
			continue
		}

		got, ok := audio.ShippedGenre(slug.Of(named))
		out.Words = append(out.Words, genreWords(got, ok, wants)...)
	}

	if ask.Technique != nil {
		out.Attack = string(ask.Technique.Attack)
	}

	if ask.Subject != nil {
		out.Name = ask.Subject.Name
	}

	return out
}

// genreWords is what a genre earned, as words a build can act on.
//
// The measurement ships in the binary, so asking for a genre costs no audio.
// Each word carries the figures behind it, which is what tells a word somebody
// measured from one somebody asserted: a genre sitting just past the others
// moves a control barely at all, and one far past it moves a full step.
//
// Nothing where the genre was never measured or earned nothing. Saying so is
// translate's job, and doing it here too would say it twice.
// Given what was found rather than finding it, so each of the three ways a
// genre contributes nothing has a test. Only two are reachable through the
// shipped data, and which two depends on whichever records somebody tagged.
func genreWords(
	got audio.Genre,
	ok bool,
	wants string,
) []compile.Word {
	// Under the threshold is reported, never computed from: eight records from
	// three players, or a request for the genre gets one band's sound. A genre
	// pooled across two instruments clears Usable for the same reason.
	if !ok || !got.Usable {
		return nil
	}

	// Measured on another instrument, which is not a weaker version of the same
	// claim. Every word here carries the figures that earned it and those
	// figures size the step a control moves, so a bass genre's `dark` handed to
	// a guitar build moves a guitar control by how far a bass sat from other
	// basses. The words are dropped and translate says why.
	if wants != "" && got.Instrument != "" && wants != got.Instrument {
		return nil
	}

	out := make([]compile.Word, 0, len(got.Terms))

	for _, t := range got.Terms {
		out = append(out, compile.Word{
			Term: t.Term,
			// Earned rather than asked for, which is what lets a word somebody
			// wrote outrank it on an axis they both answer.
			Derived: true,
			Evidence: []tone.Evidence{{
				Kind:     tone.EvidenceAudio,
				Measured: &map[string]float64{string(t.Key): t.Mine},
				Against:  &map[string]float64{string(t.Key): t.Others},
			}},
		})
	}

	return out
}

// unfamiliar names the character terms nothing defines.
//
// Said rather than refused. A term moves no knob, so an unfamiliar one costs
// the preset nothing, and a build that stopped over a word would be refusing
// somebody the right to describe a sound in their own words. The asks this
// project ships are held to the list by a test instead.
func unfamiliar(
	intent compile.Intent,
) []result.Unfamiliar {
	terms := make([]string, 0, len(intent.Words))
	for _, w := range intent.Words {
		terms = append(terms, w.Term)
	}

	unknown := compile.CheckWords(terms)

	out := make([]result.Unfamiliar, 0, len(unknown))
	for _, u := range unknown {
		out = append(out, result.Unfamiliar{Term: u.Term, Near: u.Near})
	}

	return out
}

// addedFrom says what went into the chain that the rig did not name.
func addedFrom(
	added []compile.Added,
) []result.Added {
	out := make([]result.Added, 0, len(added))
	for _, a := range added {
		out = append(out, result.Added{
			Name:   a.Block.Name,
			Reason: a.Reason,
			Share:  a.Share,
		})
	}

	return out
}

// movedFrom says which words turned which knobs.
func movedFrom(
	moved []compile.Moved,
) []result.Moved {
	out := make([]result.Moved, 0, len(moved))
	for _, m := range moved {
		out = append(out, result.Moved{
			Term:      m.Term,
			Param:     m.Param,
			From:      m.From,
			To:        m.To,
			Against:   m.Against,
			YieldedTo: m.YieldedTo,
			Because:   m.Because,
			Already:   m.Already,
			Weight:    m.Weight,
		})
	}

	return out
}

// setupAt reads what the person has, where they named a file.
//
// Nothing is not an error. Every build before a Setup could be named built for
// the record, and that is still the answer for somebody who has not written
// one: the compensation is an improvement on building blind rather than a
// requirement for building at all.
// known is the rig to build and the ask beside it, from wherever it was named.
//
// The two sources answer with the same pair because everything downstream wants
// the same pair. Which one was named is this function's whole business, and Make
// does not ask again.
func (o MakeOptions) known() (result.Known, error) {
	byID := o.RigID != ""
	byPath := o.RigPath != ""

	if byID == byPath {
		return result.Known{}, ErrOneRig
	}

	if byID {
		return o.rigs().Find(o.Source, o.RigID)
	}

	// The whole document, so the ask comes with it. There was an --ask flag until
	// version 2, for a rig whose ask did not sit beside it; one file has no
	// beside.
	doc, err := readDoc(context.Background(), o.RigPath)
	if err != nil {
		return result.Known{}, err
	}

	return result.Known{ID: doc.Id, Rig: doc.Rig, Ask: doc.Ask}, nil
}

// ErrOneRig reports neither a rig named nor both.
var ErrOneRig = errors.New("name one of a rig identifier or a rig file")

func setupAt(
	at string,
) (*tone.Setup, error) {
	if at == "" {
		return nil, nil
	}

	f, err := os.Open(at) //nolint:gosec // the path is the caller's own file
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", at, err)
	}

	defer func() { _ = f.Close() }()

	held, err := tone.LoadSetup(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", at, err)
	}

	return &held, nil
}
