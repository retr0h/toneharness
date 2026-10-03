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

package presets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/fileslots"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// CompileOptions says which rig to build, what to build it into, and where the
// preset goes.
type CompileOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// RigPath is the rig to read, and PlanPath the plan.
	//
	// Exactly one. A rig is the gear in signal order and carries no model, so it
	// is realised against the catalog on the way through; a plan already names
	// which model answered and what every knob is set to, which is what somebody
	// exporting a slot and tuning it by hand has.
	RigPath  string
	PlanPath string
	// TemplatePath is a preset to write the chain into. Empty uses an
	// untouched preset the device itself wrote.
	TemplatePath string
	// OutputPath is where the preset is written.
	OutputPath string
	// Existing is what happens to a file already at OutputPath. The zero
	// value replaces it.
	Existing result.Existing
}

// Compile turns a rig into a preset a device will load.
//
// The chain is written into a preset rather than assembled from nothing. A
// device expects inputs, outputs, a split and a join around a chain, and
// 98.6% of real presets carry them; one built without them is unlike anything
// the hardware has written. A template is what makes a rig lifted off a device
// rebuild exactly.
func Compile(
	ctx context.Context,
	opts CompileOptions,
) (result.Built, error) {
	cat, err := opts.catalog(ctx)
	if err != nil {
		return result.Built{}, err
	}

	made, name, blocks, err := readOne(ctx, opts, cat)
	if err != nil {
		return result.Built{}, err
	}

	// Checked before anything is written, and checked here for the same reason
	// Make checks: a preset that will not load is worse than no preset. This is
	// the path with the weaker input, because what it reads is a file somebody
	// may have typed, where Make resolves a rig the project ships.
	if err := plan.Validate(cat, made, plan.LimitsFor(cat.Device)); err != nil {
		return result.Built{}, fmt.Errorf(
			"the chain this document describes will not load: %w", err)
	}

	doc, err := template(ctx, opts.TemplatePath)
	if err != nil {
		return result.Built{}, err
	}

	doc.Data.Device = cat.DeviceID

	// The identifier names the preset. A document read off disk arrives on its
	// own, with no ask beside it to say whose sound it is, and the identifier is
	// the one name it has. Lower says the same thing again from the plan it is
	// handed, and a device label on that plan beats both.
	doc.Data.Meta.Name = name

	if err := opts.compiler().Lower(doc, made, cat); err != nil {
		return result.Built{}, err
	}

	var buf bytes.Buffer

	// A compiler can leave something in the document that does not encode,
	// and a preset file holding nothing is worse than no file.
	if err := preset.Write(&buf, doc); err != nil {
		return result.Built{}, fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	if err := fileslots.Save(opts.OutputPath, buf.Bytes(), opts.Existing); err != nil {
		return result.Built{}, err
	}

	return result.Built{
		Name:   name,
		Blocks: blocks,
		Path:   opts.OutputPath,
	}, nil
}

// ErrOnePath reports neither path given, or both.
var ErrOnePath = errors.New("name one of --rig or --plan")

// readOne reads whichever document was named, and answers with the plan to
// write, the name to write it under and how many blocks it holds.
//
// A rig is realised on the way through because it names gear rather than models,
// which is the whole reason it reads the same on hardware nobody has written a
// driver for. A plan is already realised and goes straight to being written,
// which is what somebody who exported a slot and turned a knob has in hand.
func readOne(
	ctx context.Context,
	opts CompileOptions,
	cat *catalog.Catalog,
) (plan.Plan, string, int, error) {
	named := opts.RigPath != ""
	alsoNamed := opts.PlanPath != ""

	if named == alsoNamed {
		return plan.Plan{}, "", 0, ErrOnePath
	}

	if alsoNamed {
		made, err := readPlan(ctx, opts.PlanPath)
		if err != nil {
			return plan.Plan{}, "", 0, err
		}

		return made, made.Name, len(made.Blocks), nil
	}

	doc, err := readDoc(ctx, opts.RigPath)
	if err != nil {
		return plan.Plan{}, "", 0, err
	}

	id, spec := doc.Id, doc.Rig

	made, err := opts.compiler().Realise(id, spec, cat)
	if err != nil {
		return plan.Plan{}, "", 0, err
	}

	// After realising, because a move names a role and a role has no position
	// until the chain is laid out. Nothing fits on this path, so the position
	// Realise gave each block is the one the preset gets.
	if err := opts.compiler().Moves(&made, spec, made.Blocks, cat); err != nil {
		return plan.Plan{}, "", 0, err
	}

	return made, id, len(spec.Chain), nil
}

// readPlan loads a plan from disk.
func readPlan(
	ctx context.Context,
	path string,
) (plan.Plan, error) {
	if err := ctx.Err(); err != nil {
		return plan.Plan{}, err
	}

	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return plan.Plan{}, fmt.Errorf("opening %s: %w", path, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	return plan.Load(f)
}

// readDoc loads one document from disk.
//
// The whole thing rather than its gear, because a file holds both halves since
// version 2 and whichever caller reads one wants a different part of it. The
// document owns the identifier, so a rig no longer carries one that could
// disagree with it.
func readDoc(
	ctx context.Context,
	path string,
) (tone.Spec, error) {
	if err := ctx.Err(); err != nil {
		return tone.Spec{}, err
	}

	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return tone.Spec{}, fmt.Errorf("opening %s: %w", path, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	return tone.Load(f)
}

// template returns the preset a chain is written into.
//
// A named one is read the way a standalone preset is read anywhere else.
func template(
	ctx context.Context,
	path string,
) (*preset.Document, error) {
	if path == "" {
		return preset.Blank()
	}

	return fileslots.ReadPreset(ctx, path)
}
