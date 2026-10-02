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

package compile

import (
	"errors"
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// Fit realises a rig on the device a catalog describes.
//
// Deterministic rather than chosen. Every entry names gear and the catalog says
// which model emulates it, so the same rig fits the same way every time. The
// path for a rig somebody wrote by hand: it carries no knob positions, because
// a position is only meaningful against the model whose knob it is and that
// model is decided here. A build that wants the corpus or a word to set one
// goes through Resolve instead.
func Realise(
	id string,
	spec rig.Spec,
	cat *catalog.Catalog,
) (plan.Plan, error) {
	// Checked on the way in as well as on the way out. A rig can arrive from
	// anywhere — a file somebody wrote, a model that generated one — and
	// building a plan out of one that does not meet its own contract turns a
	// legible error into a device refusing a file.
	if err := rig.Validate(spec); err != nil {
		return plan.Plan{}, err
	}

	blocks := make([]plan.Block, 0, len(spec.Chain))

	for i, entry := range spec.Chain {
		model, err := modelFor(entry, cat, string(spec.Instrument))
		if err != nil {
			return plan.Plan{}, fmt.Errorf("chain entry %d: %w", i, err)
		}

		blk, _ := cat.Block(model)

		blocks = append(blocks, plan.Block{
			Model: model,
			// The catalog's defaults with no corpus behind them: Line 6 state
			// one for every parameter and it is never invalid, while a median
			// is an opinion about what other people did and belongs to a build
			// that asked for one.
			Params:  settings(blk, nil),
			DSP:     0,
			Pos:     i,
			Enabled: true,
		})
	}

	// The identifier rather than a subject's name, because a subject is what
	// somebody asked for and lives on the ask. It arrives as an argument since
	// version 2: the document owns the name and a rig is a section of it.
	return plan.Plan{Name: id, Rig: id, Blocks: blocks}, nil
}

// Lower writes a plan into a preset.
//
// The document is written into rather than built, because a preset holds
// things a plan does not model — the inputs, outputs, split and join a device
// expects, the metadata nobody documented. Building one from nothing would
// produce a file unlike any the device has ever written.
func Lower(
	doc *preset.Document,
	p plan.Plan,
	cat *catalog.Catalog,
) error {
	// What the plan claims beside its chain, against the catalog that has to
	// supply it. A plan arrives from anywhere — read off a device, exported and
	// edited by hand — and a colour this device cannot light or a parameter the
	// block does not carry is worth an error rather than a switch that lights
	// wrongly and a pedal that moves nothing.
	if err := check(p, p.Blocks, cat); err != nil {
		return err
	}

	// Before the chain, so a plan that carries routing writes its own rather
	// than keeping whatever the preset underneath came with.
	restore(doc, p.Device)

	// After the device's own state, because a plan's snapshots are its own
	// even when it carries a verbatim record of everything else.
	if len(p.Snapshots) > 0 {
		pruneSnapshots(doc)
		restoreSnapshots(doc, p.Snapshots)
	}

	// After the device's own state, so a plan's own assignments win over
	// whatever the preset underneath carried. Checked already, by check above,
	// along with everything else the plan claims.
	Controllers(doc, p, p.Blocks, cat)

	Footswitches(doc, p, cat)

	// The plan names the preset, not the document underneath: compiling into
	// an untouched preset would otherwise write out the template's own name. A
	// lifted plan carries the label the device stored, padding and all, which
	// is what restore has already put back.
	if p.Device != nil && p.Device.Name != nil {
		p.Name = *p.Device.Name
	}

	return doc.SetSpec(p)
}

// at reads an optional integer, falling back when a plan does not state one.
func at(
	v *int,
	fallback int,
) int {
	if v == nil {
		return fallback
	}

	return *v
}

// modelFor decides which model an entry means.
//
// The gear name against the catalog, which is the only thing a rig says: an
// exact model identifier is one device's internal name and lives in the plan
// this is building rather than in the document being read.
func modelFor(
	entry rig.ChainEntry,
	cat *catalog.Catalog,
	instrument string,
) (catalog.ModelID, error) {
	b, err := gear(cat, entry.Gear, entry.Role, instrument)

	// The rig named gear this device cannot do and said what to put there
	// instead. Resolving does the same, so a rig built either way lands on
	// the same model.
	if err != nil && entry.Substitute != nil && errors.Is(err, ErrNoSuchGear) {
		b, err = gear(cat, entry.Substitute.Gear, entry.Role, instrument)
		if err != nil {
			return "", fmt.Errorf(
				"%q stands in for %q, and nothing emulates it either: %w",
				entry.Substitute.Gear, entry.Gear, err)
		}
	}

	if err != nil {
		return "", err
	}

	return b.ID, nil
}
