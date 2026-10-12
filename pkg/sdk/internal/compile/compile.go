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
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// Compiler is this package's work as a value.
//
// It holds nothing, and every method is the package-level function of the
// same name. The type exists so that a caller can say what it depends on and
// stand something else in its place, which a package-level function does not
// allow. The pair is the one net/http draws between Get and a Client.
//
// The interface a caller needs is the caller's to declare, and will be
// smaller than this: reading a device wants Lift and Lower, building a rig
// wants Resolve and Fit, and neither wants all four.
type Compiler struct{}

// New returns a Compiler.
func New() *Compiler { return &Compiler{} }

// Lift reads a preset into its identifier, the rig it describes, and the plan
// that realises it.
func (*Compiler) Lift(
	doc *preset.Document,
	cat *catalog.Catalog,
) (string, rig.Spec, plan.Plan, error) {
	return Lift(doc, cat)
}

// Lower writes a plan into a preset.
func (*Compiler) Lower(
	doc *preset.Document,
	made plan.Plan,
	cat *catalog.Catalog,
) error {
	return Lower(doc, made, cat)
}

// Realise turns a rig into the plan that answers it on one device.
func (*Compiler) Realise(
	id string,
	spec rig.Spec,
	cat *catalog.Catalog,
) (plan.Plan, []result.Dropped, error) {
	return Realise(id, spec, cat)
}

// Sections writes a rig's song sections into a preset's snapshots.
func (*Compiler) Sections(
	doc *preset.Document,
	spec rig.Spec,
	made plan.Plan,
	blocks []plan.Block,
	cat *catalog.Catalog,
) error {
	return Sections(doc, spec, made, blocks, cat)
}

// Moves turns what a rig says a foot reaches into a plan's assignments.
func (*Compiler) Moves(
	made *plan.Plan,
	spec rig.Spec,
	blocks []plan.Block,
	cat *catalog.Catalog,
) error {
	return Moves(made, spec, blocks, cat)
}

// Controllers writes what an expression pedal or footswitch moves.
func (*Compiler) Controllers(
	doc *preset.Document,
	made plan.Plan,
	blocks []plan.Block,
	cat *catalog.Catalog,
) {
	Controllers(doc, made, blocks, cat)
}

// Footswitches writes what the pedal prints under each switch.
func (*Compiler) Footswitches(
	doc *preset.Document,
	made plan.Plan,
	cat *catalog.Catalog,
) {
	Footswitches(doc, made, cat)
}

// Resolve turns a rig, the ask beside it and a catalog into a chain.
func (*Compiler) Resolve(
	id string,
	spec rig.Spec,
	intent Intent,
	cat *catalog.Catalog,
	stats *corpus.Stats,
) (plan.Plan, []Added, []Moved, Compensated, error) {
	return Resolve(id, spec, intent, cat, stats)
}

// Fit drops what a device has no room for.
func (*Compiler) Fit(
	spec plan.Plan,
	cat *catalog.Catalog,
	lim plan.Limits,
) plan.Plan {
	return Fit(spec, cat, lim)
}
