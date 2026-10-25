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
	"path/filepath"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/internal/fileslots"
	"github.com/retr0h/toneharness/pkg/sdk/internal/rigs"
	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// MakeOptions says what to build and where to put it.
type MakeOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// SweepsDir is the tree holding the committed curves, which say what a
	// turn of an amplifier's level control is worth. Empty reads none, and a
	// stated level is reported rather than reached.
	SweepsDir string
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

	// What the records of whoever this rig is for measure as, and the sweeps
	// to reach it with.
	//
	// Resolved here rather than in the compiler, for the reason
	// `Intent.Loudness` is: that package builds a chain and has no business
	// deciding who a rig is for or reading a corpus to find out. The rig's own
	// identifier is who it is for, which is what makes this reach an artist
	// rig at all: a document carries `subject` and no `like`, and reading the
	// subject as a target in `translate` was tried and reverted because a
	// subject-only ask is meant to refuse and name `like` instead.
	// The readings, read once. Both the aim and the effects a foot reaches for
	// are measured against them, and reading them twice would mean two error
	// paths for one file.
	lib, err := opts.library()
	if err != nil {
		return result.Made{}, err
	}

	intent.Aim = opts.aimOf(known.ID, lib)
	intent.Curves = sweptIn(opts.SweepsDir)

	cat, err := opts.catalog(ctx)
	if err != nil {
		return result.Made{}, err
	}

	// And what a foot reaches for, chosen from the same records the aim came
	// from.
	//
	// Here rather than in the compiler for the reason `Intent.Aim` is: that
	// package builds a chain and has no business reading a corpus to find out
	// who a rig is for. And here rather than in `translate` because an artist
	// rig's chain is authored and never passes through `chainFor`, so the
	// genre-only list that used to live there could not reach one: all eleven
	// genres carried the same drive and equaliser and none of the 32 artists
	// carried anything.
	// Statistics are an improvement on the catalog's defaults, not a
	// requirement, but somebody who named a file asked for those ones. Building
	// without them would hand back a more generic preset than the one asked
	// for, and say nothing about it.
	stats, err := corpus.Open(opts.StatsPath)
	if err != nil {
		return result.Made{}, err
	}

	rec, chosen, stomped := stompable(
		known.ID, rec, setupOf(held), cat, lib, stats, opts)

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

	// How loud it comes out, after the fit so the plan being matched is the one
	// that will be written. Only when a Setup named a rig: a build that was
	// not asked to match anything comes out wherever its gear leaves it, which
	// is what every build did before this.
	levelled, err := opts.matched(held, known.ID, spec, cat)
	if err != nil {
		return result.Made{}, err
	}

	if levelled.Param != "" {
		spec = levelled.Plan
	}

	if err := plan.Validate(cat, spec, limits); err != nil {
		return result.Made{}, fmt.Errorf(
			"the chain this rig describes will not load: %w", err)
	}

	doc := build(cat.DeviceID, spec)

	// What the rig said the preset holds beside its chain, put back before
	// anything musical is written over it. A rig's `preset:` compiles into the
	// plan's record of the device, and this is where that record reaches the
	// file: without it the field would be written by a resolve and read by
	// nothing, so an output's destination changed by hand would not reach the
	// pedal. Sections, controllers and footswitches come after, because those are
	// what somebody asked for rather than what a device happened to hold.
	compile.ApplyMembers(doc, rec.Preset)

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

	// A switch for each block worth stomping, where the rig named none itself.
	// `plan.Footswitches` was only ever filled by a lift, so everything built
	// here arrived with its switches unassigned: the blocks in the right order
	// at the right values, and nothing on the pedal turning any of them off.
	//
	// Only where the rig said nothing. A rig that names its own switches has
	// said something musical and this must not overwrite it, the same way a
	// pinned control outranks a computed one.
	if len(spec.Footswitches) == 0 {
		spec.Footswitches = compile.Assign(
			spec, cat, plan.LimitsFor(cat.Device).Switches)
	}

	opts.compiler().Footswitches(doc, spec, cat)

	// Last, because it records the chain's bypass states and everything above
	// may still have changed which blocks there are. Block bypass is snapshot
	// state, so a preset whose snapshots the device does not believe has
	// footswitches that do nothing until they have been pressed three times:
	// task 278, and `build` hands over three blank ones.
	opts.compiler().Trust(doc, spec.Blocks)

	if err := write(opts.OutputPath, doc, opts.Existing); err != nil {
		return result.Made{}, err
	}

	return result.Made{
		Plan:       spec,
		Added:      append(stomped, addedFrom(added)...),
		Chosen:     chosen,
		Moved:      movedFrom(moved),
		Unfamiliar: unfamiliar(intent),
		Playing:    result.Playing{Terms: playing.Terms, Said: playing.Said},
		Levelled:   levelled.Levelled,
		Path:       opts.OutputPath,
		Preset:     doc,
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
//
// No path writes nothing. `rigs resolve` builds a chain to read its controls
// back off and has no use for the preset, and the alternative is a caller
// naming a scratch file it then has to remove.
func write(
	path string,
	doc *preset.Document,
	existing result.Existing,
) error {
	if path == "" {
		return nil
	}

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
				// The scale the distance is read against, so a word that barely
				// cleared the others moves a control less far than one that
				// cleared them outright.
				Spread: &map[string]float64{string(t.Key): t.Spread},
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
			Name:     a.Block.Name,
			Reason:   a.Reason,
			Share:    a.Share,
			Bypassed: a.Bypassed,
			Running:  a.Running,
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
			Term:       m.Term,
			At:         m.At,
			Param:      m.Param,
			From:       m.From,
			To:         m.To,
			Against:    m.Against,
			YieldedTo:  m.YieldedTo,
			Because:    m.Because,
			Already:    m.Already,
			Weight:     m.Weight,
			Held:       m.Held,
			Overridden: m.Overridden,
			Step:       m.Step,
			Default:    m.Default,
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

// ErrNoLoudness is a rig named as a loudness to match whose chain has no
// cabinet anybody has measured.
//
// Refused rather than passed over, because the Setup asked for something this
// cannot do and a build that quietly came out at its own loudness is a build
// somebody compares against another one.
var ErrNoLoudness = errors.New("no measured cabinet to match")

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

// curvesIn reads the committed sweeps for whatever the chain holds.
//
// Off disk rather than out of the binary, because the curves are not packed
// into it: `just pack-measured` embeds the fingerprints and the sweeps stay
// under resources/sweeps/. So a build from an installed release has none, and
// a stated level is reported rather than reached.
//
// A sweep that will not open or will not parse is a sweep this does not have,
// which is the same answer as not having one. The level step says so by name,
// and refusing a whole build over one unreadable curve would be refusing
// somebody a preset over a file they never asked it to read.
func curvesIn(
	dir string,
	spec plan.Plan,
	cat *catalog.Catalog,
) map[catalog.ModelID]measured.Curves {
	if dir == "" {
		return nil
	}

	read := sweptIn(dir)
	out := map[catalog.ModelID]measured.Curves{}

	// The cabinet first, because an amplifier's readings are preferred from a
	// sweep taken through it: see [sweepNames].
	cab := catalog.ModelID("")

	for _, b := range spec.Blocks {
		if one, known := cat.Block(b.Model); known &&
			one.Category == catalog.CategoryCab {
			cab = b.Model

			break
		}
	}

	for _, b := range spec.Blocks {
		if one, known := cat.Block(b.Model); !known ||
			one.Category != catalog.CategoryAmp {
			continue
		}

		got, swept := read(b.Model, cab)
		if !swept {
			continue
		}

		out[b.Model] = got
	}

	return out
}

// matchedTo is a level match and the plan it produced.
//
// The plan travels with the answer because a caller that moved a control has a
// different plan from the one it passed in, and returning only the report would
// leave the two to be reassembled by whoever remembered to.
type matchedTo struct {
	result.Levelled

	Plan plan.Plan
}

// matched makes this build as loud as the rig a Setup named.
//
// Does nothing without one, and nothing when the rig named is the one being
// built: a chain is already as loud as itself, and saying so would be a report
// about nothing.
func (o MakeOptions) matched(
	held *tone.Setup,
	id string,
	spec plan.Plan,
	cat *catalog.Catalog,
) (matchedTo, error) {
	if held == nil || held.MatchLoudness == nil || *held.MatchLoudness == "" ||
		*held.MatchLoudness == id {
		return matchedTo{}, nil
	}

	lib, err := o.library()
	if err != nil {
		return matchedTo{}, err
	}

	want, err := o.loudnessOf(*held.MatchLoudness, cat, lib)
	if err != nil {
		return matchedTo{}, err
	}

	built, at := o.compiler().Level(
		spec, cat, want, lib, curvesIn(o.SweepsDir, spec, cat))
	at.Rig = *held.MatchLoudness

	return matchedTo{Levelled: at, Plan: built}, nil
}

// loudnessOf is the cabinet reading of the rig a Setup named.
//
// Realised rather than built: what is wanted is which cabinet that rig ends in,
// and realising answers it from the rig and the catalog alone. A full build
// would resolve its words and fill from the corpus to arrive at the same
// cabinet, and would read the Setup again on the way, which is the Setup that
// sent it here.
func (o MakeOptions) loudnessOf(
	id string,
	cat *catalog.Catalog,
	lib measured.Library,
) (float64, error) {
	found, err := o.rigs().Find(o.Source, id)
	if err != nil {
		return 0, fmt.Errorf(
			"the Setup says to come out as loud as %q: %w", id, err)
	}

	spec, _, err := o.compiler().Realise(id, found.Rig, cat)
	if err != nil {
		return 0, fmt.Errorf(
			"the Setup says to come out as loud as %q, which will not build: %w",
			id, err)
	}

	for _, b := range spec.Blocks {
		one, known := cat.Block(b.Model)
		if !known || one.Category != catalog.CategoryCab {
			continue
		}

		if read, measured := lib.Blocks[string(b.Model)]; measured &&
			read.Measured() {
			return read.Level, nil
		}
	}

	return 0, fmt.Errorf(
		"%w: the Setup says to come out as loud as %q, and nothing in that "+
			"rig's chain has a measured cabinet", ErrNoLoudness, id)
}

// aimOf is what the records of whoever a rig is for measure as.
//
// The rig's own identifier is who it is for, and it is also the directory the
// corpus files their records under, which is what makes this answerable from
// here at all. An artist rig's gear is authored, so its resolve never passes
// through the gear-choosing path where an ask's `like` is read; and `subject`
// is deliberately not read there, because a rig resolving itself from the rig
// researched for the same person is circular.
//
// Nil for a rig nobody has records for, which is most of them and not an
// error: those keep the controls their sources justify.
//
// Resolved here rather than in the compiler, for the reason `Intent.Loudness`
// is: that package builds a chain and has no business reading a corpus.
func (o MakeOptions) aimOf(
	id string,
	lib measured.Library,
) *measured.Figures {
	// A player first, then a genre. A rig's identifier is who it is for, and
	// the two kinds of rig this ships are a person and a sound: 32 artists and
	// 11 genres, and nothing is both.
	//
	// The genre half is task 242's. Three of the eleven borrow their
	// amplifier's own pairing for a cabinet, so that cabinet is added by the
	// compiler and never reaches the path where a genre's gear is chosen: they
	// arrived on the corpus median microphone because nothing else had the
	// genre's own figures to hand.
	got, ok := translate.AimAtPlayer(id, lib.Baseline)
	if !ok {
		got, ok = translate.AimAtGenre(id, lib.Baseline)
	}

	if !ok {
		return nil
	}

	// As a displacement, because that is what reads it. `AimAtPlayer` answers
	// with the target over the library's own baseline, which is the shape a
	// block is ranked against; `Curves.Toward` and `Curves.TowardFrom` both
	// read their target as a lean and take only its sign, so an absolute
	// figure reads as "more of everything" on every axis. A centroid in hertz
	// and a band share are both positive whatever a player's records say.
	lean := translate.DisplacedFrom(got, lib.Baseline)

	return &lean
}

// sweptIn reads one block's committed curves out of a tree of them.
//
// Nil for an empty path rather than a reader that always fails, so the
// compiler can tell "no sweeps were offered" from "that block is not in them".
func sweptIn(
	dir string,
) func(model, with catalog.ModelID) (measured.Curves, bool) {
	if dir == "" {
		return nil
	}

	// `with` is the cabinet the chain ends in, so a reading taken through
	// that cabinet is preferred over one taken with the block alone. Empty
	// asks only for the isolated reading, which is every caller with no
	// chain in hand.
	return func(model, with catalog.ModelID) (measured.Curves, bool) {
		for _, name := range sweepNames(model, with) {
			got, ok := loadCurves(filepath.Join(dir, name))
			if ok {
				return got, true
			}
		}

		return measured.Curves{}, false
	}
}

// sweepNames is the files that hold a block's readings, best first.
//
// **A chain sweep before the isolated one**, which is the whole of task 271
// reaching the solver. An amplifier swept alone has no cabinet lowpassing it,
// so `measured.StartAt` and `Curves.Toward` both refuse one, and every artist
// rig's amplifier then arrived at corpus medians and catalog defaults: Bass
// 0.41, Drive 0.6, ChVol 0.8, which is the factory setting to the digit and
// the regression task 268 existed to stop.
//
// A reading through this chain's own cabinet is not refused, because it
// describes a chain rather than a block. So naming the cabinet is what lets an
// amplifier be tuned from somebody's records at all.
//
// Lowercased, which is how `measure controls --after` writes them, and the
// isolated name is left exactly as the catalog spells it because that is how
// the committed library is filed.
func sweepNames(
	model catalog.ModelID,
	with catalog.ModelID,
) []string {
	out := make([]string, 0, 2)

	if with != "" && with != model {
		out = append(out, strings.ToLower(
			string(model))+"-into-"+strings.ToLower(string(with))+".json")
	}

	return append(out, string(model)+".json")
}

// loadCurves reads one file of readings, or says it could not.
func loadCurves(
	path string,
) (measured.Curves, bool) {
	f, err := os.Open(path) //nolint:gosec // a path built from the catalog
	if err != nil {
		return measured.Curves{}, false
	}

	defer func() { _ = f.Close() }()

	got, err := measured.LoadCurves(f)
	if err != nil {
		return measured.Curves{}, false
	}

	return got, true
}

// stompable adds the effects this rig's own records ask for, each switched
// off, and says what each one is for.
//
// Spliced into the chain at the position its role implies rather than
// appended, because the compiler walks a rig's chain in the order it is
// written and a reverb at the front of one is a reverb in front of the
// amplifier. The rig's own order is otherwise untouched: whoever wrote it made
// that decision, and a sort here would silently move somebody's pedals.
//
// Silent where the rig is for nobody the corpus measures, which is every rig
// outside the marketplace, and where the records ask for nothing.
func stompable(
	id string,
	rec rig.Spec,
	setup tone.Setup,
	cat *catalog.Catalog,
	lib measured.Library,
	stats *corpus.Stats,
	opts MakeOptions,
) (rig.Spec, []rig.ChainEntry, []result.Added) {
	notes := translate.Notes{}

	deps := translate.Deps{
		Catalog:   cat,
		Measured:  lib,
		Corpus:    stats,
		CurvesFor: sweptIn(opts.SweepsDir),
	}

	got := translate.Stompable(id, rec, setup, deps, &notes)

	out := make([]result.Added, 0, len(got))
	chosen := make([]rig.ChainEntry, 0, len(got))

	// Spliced before the corpus is asked for more, because what the records
	// asked for is already spending the chip and the switches. Asking first
	// and splicing afterwards is how Matt Freeman's chain reached 99.03% of a
	// 95% ceiling and refused to build at all: the offers were costed against
	// a chain that did not yet hold the two blocks his own records had just
	// put in it.
	for _, at := range got {
		rec.Chain = translate.Spliced(rec.Chain, at.Entry)
		chosen = append(chosen, at.Entry)

		out = append(out, result.Added{
			Name: at.Entry.Gear, Reason: at.Said, Bypassed: true,
		})
	}

	// Then whatever measurement could not ask for. A word is earned on four
	// axes which answer with three kinds of block, so three switches is the
	// ceiling and 21 of the 43 rigs here earn nothing at all: two of seven
	// presets put on a pedal arrived with every switch empty. Everything this
	// adds is bypassed, so it is an offer rather than a claim.
	for _, at := range translate.Pizzazz(id, rec, deps, &notes) {
		rec.Chain = translate.Spliced(rec.Chain, at.Entry)
		chosen = append(chosen, at.Entry)

		out = append(out, result.Added{
			Name: at.Entry.Gear, Reason: at.Said, Bypassed: true,
		})
	}

	return rec, chosen, out
}

// setupOf is the Setup a build was given, or an empty one where none was.
//
// The pointer says whether a person wrote one and nothing downstream of here
// needs that distinction: a Setup naming nothing and no Setup at all reach the
// same answers.
func setupOf(
	held *tone.Setup,
) tone.Setup {
	if held == nil {
		return tone.Setup{}
	}

	return *held
}
