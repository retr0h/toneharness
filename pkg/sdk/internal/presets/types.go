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
	"context"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/internal/rigs"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// Catalogs hands over the catalog a rig is built against. The sdk Client
// satisfies it, and keeps the catalog it opened.
type Catalogs interface {
	// Catalog returns the catalog, opening it on first use.
	Catalog(ctx context.Context) (*catalog.Catalog, error)
}

// Rigs finds the curated rig a build starts from, and the ask beside it.
type Rigs interface {
	// Find returns the rig with the given identifier, and the ask beside it,
	// from where src says.
	Find(src rigs.Source, id string) (result.Known, error)
}

// Compiler turns a rig into a preset a device has room for.
//
// Six of the seven methods pkg/compile carries. Resolve and Fit build a chain
// from a rig, Sections writes its song sections, Controllers writes what
// moves while you play, Footswitches writes what the pedal prints, and Lower
// writes a rig into a preset. Lift reads a slot rather than building one, so
// it is not named here.
type Compiler interface {
	// Resolve turns a rig, the ask beside it and a catalog into a chain.
	Resolve(
		spec rig.Spec,
		intent compile.Intent,
		cat *catalog.Catalog,
		stats *corpus.Stats,
	) (plan.Plan, []compile.Added, []compile.Moved, compile.Compensated, error)
	// Fit drops what a device has no room for.
	Fit(spec plan.Plan, cat *catalog.Catalog, lim plan.Limits) plan.Plan
	// Realise turns a rig into the plan that answers it on this device.
	Realise(spec rig.Spec, cat *catalog.Catalog) (plan.Plan, error)
	// Lower writes a plan into a preset.
	Lower(doc *preset.Document, made plan.Plan, cat *catalog.Catalog) error
	// Controllers writes what an expression pedal or footswitch moves.
	Controllers(
		doc *preset.Document, made plan.Plan, blocks []plan.Block, cat *catalog.Catalog,
	)
	// Footswitches writes what the pedal prints under each switch.
	Footswitches(doc *preset.Document, made plan.Plan, cat *catalog.Catalog)
	// Sections writes a rig's song sections into a preset's snapshots.
	Sections(
		doc *preset.Document, spec rig.Spec, made plan.Plan, blocks []plan.Block,
		cat *catalog.Catalog,
	) error
	// Moves turns what a rig says a foot reaches into a plan's assignments.
	Moves(
		made *plan.Plan, spec rig.Spec, blocks []plan.Block, cat *catalog.Catalog,
	) error
}

// Deps are the collaborators building a preset works through.
//
// Every field optional: a zero value reaches the real thing, so a caller
// names only what it wants to stand something else in for.
type Deps struct {
	// Catalogs hands over the catalog. Nil reads the one built into this
	// binary.
	Catalogs Catalogs
	// Rigs finds curated rigs. Nil reads the ones in the binary.
	Rigs Rigs
	// Compiler turns a rig into a chain. Nil uses pkg/compile.
	Compiler Compiler
}

func (d Deps) catalog(
	ctx context.Context,
) (*catalog.Catalog, error) {
	if d.Catalogs != nil {
		return d.Catalogs.Catalog(ctx)
	}

	return catalog.BuiltIn()
}

func (d Deps) rigs() Rigs {
	if d.Rigs != nil {
		return d.Rigs
	}

	return rigs.Store{}
}

func (d Deps) compiler() Compiler {
	if d.Compiler != nil {
		return d.Compiler
	}

	return compile.New()
}
