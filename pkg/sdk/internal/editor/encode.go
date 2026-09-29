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

package editor

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// Turning a chain into what a device lays on its grid.
//
// The inverse of decode.go. A chain names gear the way a person does and a
// device names it by a number, so the catalog's model table is what makes
// one into the other.

// Placements turns a preset into the blocks a device grid holds.
//
// A preset rather than a chain, because an amp's cabinet is not in the
// chain. A preset stores the amp with a `@cab` naming a sibling entry, and
// that entry holds the cabinet's own settings.
func Placements(
	doc *preset.Document,
	cat *catalog.Catalog,
) ([]wire.Placement, error) {
	if len(cat.Symbols) == 0 {
		return nil, fmt.Errorf(
			"this catalog has no model table, so a chain cannot be written to " +
				"a device: regenerate it with go generate in the toneharness repository")
	}

	c, err := doc.Spec()
	if err != nil {
		return nil, err
	}

	out := make([]wire.Placement, 0, len(c.Blocks))

	for i, b := range c.Blocks {
		model, ok := cat.SymbolNumber(b.Model)
		if !ok {
			return nil, fmt.Errorf(
				"chain entry %d names %q, which this catalog's model table of "+
					"%d does not carry: it was generated from a different release",
				i, b.Model, len(cat.Symbols))
		}

		sym, _ := cat.Symbol(model)
		kinds := typesOf(b.Model, cat)

		p := wire.Placement{
			Position: b.Pos,
			Model:    model,
			Values:   valuesOf(sym, b.Params, kinds, micOf(b.Attrs)),
			Named:    len(carried(sym, kinds)),
			Enabled:  b.Enabled,
			CabModel: -1,
		}

		if err := held(&p, doc, b, cat); err != nil {
			return nil, fmt.Errorf("chain entry %d: %w", i, err)
		}

		p.Class = classOf(b.Model, cat, len(p.Cab) > 0)

		out = append(out, p)
	}

	return out, nil
}

// held reads the cabinet an amp carries with it, when it has one.
func held(
	p *wire.Placement,
	doc *preset.Document,
	b plan.Block,
	cat *catalog.Catalog,
) error {
	name, ok := cabNameOf(b)
	if !ok {
		return nil
	}

	entry, ok := doc.Data.Tone["dsp"+strconv.Itoa(b.DSP)][name]
	if !ok {
		return fmt.Errorf("names cabinet %q, which the preset does not hold", name)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(entry, &fields); err != nil {
		return fmt.Errorf("reading cabinet %q: %w", name, err)
	}

	var id catalog.ModelID
	if err := json.Unmarshal(fields[routeModel], &id); err != nil {
		return fmt.Errorf("cabinet %q names no model: %w", name, err)
	}

	model, ok := cat.SymbolNumber(id)
	if !ok {
		return fmt.Errorf(
			"cabinet %q names %q, which this catalog's model table does not carry",
			name, id)
	}

	sym, _ := cat.Symbol(model)

	p.Cab = valuesOf(sym, paramsOfJSON(sym, fields), typesOf(id, cat), micOf(fields))
	p.CabNamed = len(sym.Params)
	p.CabModel = model

	return nil
}

// cabNameOf reads which sibling entry holds this block's cabinet.
//
// A chain built here carries it as a parameter and one read back out of a
// preset carries it as an attribute, because `@cab` is @-prefixed and that
// is where a preset keeps those. Both are the same claim.
func cabNameOf(
	b plan.Block,
) (string, bool) {
	if name, ok := b.Params[attrCab].Enum(); ok {
		return name, true
	}

	raw, ok := b.Attrs[attrCab]
	if !ok {
		return "", false
	}

	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return "", false
	}

	return name, true
}

// paramsOfJSON reads a cabinet entry's named settings.
func paramsOfJSON(
	sym catalog.Symbol,
	fields map[string]json.RawMessage,
) map[string]catalog.ParamValue {
	out := make(map[string]catalog.ParamValue, len(sym.Params))

	for _, name := range sym.Params {
		raw, ok := fields[name]
		if !ok {
			continue
		}

		var v catalog.ParamValue
		if err := json.Unmarshal(raw, &v); err == nil {
			out[name] = v
		}
	}

	return out
}

// micOf reads the microphone a cabinet carries.
//
// A cabinet sends one more value than the model has names for, and that is
// the microphone. A preset keeps it as `@mic` rather than as a parameter, so
// it comes back as the tail of the value list or as nothing at all.
func micOf(
	fields map[string]json.RawMessage,
) []any {
	raw, ok := fields[cabMic]
	if !ok {
		return nil
	}

	// The microphone is past every named parameter, so the catalog says
	// nothing about it. A device sends 11, not 11.0, and the two are
	// different bytes.
	var v catalog.ParamValue
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}

	return []any{rawValue(v, "")}
}

// classOf says what a block calls itself, in the device's own numbering.
//
// Read off the captures rather than documented: every effect says 1, a
// cabinet on its own says 15, an amp with no cabinet says 17, and an amp
// carrying one says 18.
func classOf(
	model catalog.ModelID,
	cat *catalog.Catalog,
	paired bool,
) int {
	blk, known := cat.Block(model)
	if !known {
		return wire.ClassEffect
	}

	switch blk.Category {
	case catalog.CategoryAmp:
		if paired {
			return wire.ClassAmpCab
		}

		return wire.ClassAmp
	case catalog.CategoryCab:
		return wire.ClassCab
	default:
		return wire.ClassEffect
	}
}

// valuesOf puts a block's parameters back in the order a device reads them.
//
// Position is the only thing naming a value on the wire, so a parameter the
// preset does not carry still takes its place, holding a zero.
//
// Not every entry in a model's symbol list is one of those values. A device
// writes a 2x15 cabinet with seven, where its symbol list holds eight: the
// eighth is `IrData`, which is not a parameter the chain carries. Ninety-two
// cabinets are the same, and a handful of flow blocks list `select`, `gain`
// and the two `guitarSense` entries the same way.
//
// Writing the extra one produces a chain entry a device stores, hands back
// unchanged, and renders as nothing — a preset that reads back perfectly and
// shows an empty chain on the pedal. The catalog's own parameter list for the
// block is what says which entries are real, so that is what this follows.
func valuesOf(
	sym catalog.Symbol,
	params map[string]catalog.ParamValue,
	types map[string]catalog.Param,
	tail []any,
) []any {
	if len(sym.Params) == 0 {
		return nil
	}

	out := make([]any, 0, len(sym.Params)+len(tail))

	for _, name := range carried(sym, types) {
		out = append(out, rawValue(settingOf(params, types, name), types[name].Type))
	}

	return append(out, tail...)
}

// settingOf is what one parameter is set to, falling back to the catalog.
//
// A chain that says nothing about a parameter means "leave it alone", not
// "zero". The two are the same value in Go and nothing like each other on a
// device: an amp whose Master and channel volume are absent rather than
// stated comes out 50dB down, which reads as a chain that is present and
// silent, and a campaign measuring every block that way measures its own
// noise floor 661 times.
//
// The catalog states a default for every parameter Line 6 ships, so there is
// always a better answer than zero. A value the chain does carry wins, zero
// included, because a knob somebody turned down is not a knob nobody
// mentioned.
func settingOf(
	params map[string]catalog.ParamValue,
	types map[string]catalog.Param,
	name string,
) catalog.ParamValue {
	if v, ok := params[name]; ok {
		return v
	}

	return types[name].Default
}

// carried is the symbol's parameters that a chain entry actually holds, in
// the device's own order.
//
// Empty types means the catalog has no block for this model, which leaves
// nothing to filter against. The whole list travels then, because writing a
// short entry is the worse guess of the two.
func carried(
	sym catalog.Symbol,
	types map[string]catalog.Param,
) []string {
	if len(types) == 0 {
		return sym.Params
	}

	out := make([]string, 0, len(sym.Params))

	for _, name := range sym.Params {
		if _, carried := types[name]; carried {
			out = append(out, name)
		}
	}

	return out
}

// typesOf is what the catalog says each of a model's parameters holds.
//
// A preset is JSON and JSON has one number. A cabinet's Distance of 3 is
// written `3` whether the device sent a whole number or a fraction, and only
// the catalog knows which it was. Every named parameter on every cabinet in
// the captures is a float there, and every one arrived as a float on the
// wire.
func typesOf(
	model catalog.ModelID,
	cat *catalog.Catalog,
) map[string]catalog.Param {
	blk, ok := cat.Block(model)
	if !ok {
		return nil
	}

	return blk.Params
}

// rawValue is a parameter in the shape the wire layer narrows one to.
//
// A parameter the preset does not carry arrives here as the zero ParamValue,
// which has no kind, and goes out as a zero fraction. That is what a device
// sends for a value it has nothing to say about.
func rawValue(
	v catalog.ParamValue,
	want catalog.ParamType,
) any {
	switch want {
	case catalog.ParamFloat:
		return asFloat(v)
	case catalog.ParamBool:
		b, _ := v.Bool()

		return b
	case catalog.ParamInt, catalog.ParamEnum:
		return asWhole(v)
	}

	// A model the catalog does not carry says nothing about its parameters,
	// so the value's own shape is all there is to go on.
	if b, ok := v.Bool(); ok {
		return b
	}

	if n, ok := v.Int(); ok {
		return n
	}

	f, _ := v.Float()

	return f
}

// asFloat reads a value the catalog calls a fraction.
func asFloat(
	v catalog.ParamValue,
) float64 {
	if f, ok := v.Float(); ok {
		return f
	}

	n, _ := v.Int()

	return float64(n)
}

// asWhole reads a value the catalog calls a whole number.
func asWhole(
	v catalog.ParamValue,
) int64 {
	if n, ok := v.Int(); ok {
		return n
	}

	f, _ := v.Float()

	return int64(f)
}

// Keys a preset stores one controller assignment under, and where it keeps
// them. The device owns these names; compile.Controllers writes them and this
// is its inverse.
const (
	controllerKey = "controller"
	ctlNumber     = "@controller"
	ctlMin        = "@min"
	ctlMax        = "@max"
	ctlNoSnapshot = "@snapshot_disable"
)

// PlacedControllers turns what a preset says moves into what a device stores.
//
// Read from the controller section rather than from Spec, which is the mistake
// worth naming: a plan built off a preset carries its blocks and not its
// assignments, so this resolved nothing at all and the section reached the
// pedal empty. The document keeps them under `tone.controller`, keyed by
// processor, then by the position a block is stored at, then by the
// parameter's own name.
//
// The device knows none of those names. It stores the parameter's place in its
// model's own list, which is why this needs the catalog and the wire package
// does not get one.
//
// Indexed into the symbol's whole parameter list rather than the values a block
// carries, because that is what the same section is read back as. The two
// differ where the catalog names fewer parameters than the model table does,
// and a controller written against one order and read against the other moves
// the wrong knob.
func PlacedControllers(
	doc *preset.Document,
	cat *catalog.Catalog,
) ([]wire.PlacedController, error) {
	held, ok := doc.Data.Tone[controllerKey]
	if !ok {
		return nil, nil
	}

	c, err := doc.Spec()
	if err != nil {
		return nil, err
	}

	out := []wire.PlacedController(nil)

	// Sorted, so a preset with two processors writes the same bytes every run
	// rather than whichever order a map yielded.
	for _, key := range slices.Sorted(maps.Keys(held)) {
		got, err := placedOnPath(key, held[key], c.Blocks, cat)
		if err != nil {
			return nil, err
		}

		out = append(out, got...)
	}

	return out, nil
}

// placedOnPath resolves every assignment on one processor.
func placedOnPath(
	key string,
	body json.RawMessage,
	blocks []plan.Block,
	cat *catalog.Catalog,
) ([]wire.PlacedController, error) {
	path, err := strconv.Atoi(strings.TrimPrefix(key, "dsp"))
	if err != nil {
		return nil, fmt.Errorf("unreadable processor key %q: %w", key, err)
	}

	var held map[string]map[string]map[string]any
	if err := json.Unmarshal(body, &held); err != nil {
		return nil, fmt.Errorf("reading what %s says moves: %w", key, err)
	}

	out := []wire.PlacedController(nil)

	for _, at := range slices.Sorted(maps.Keys(held)) {
		pos, err := strconv.Atoi(strings.TrimPrefix(at, "block"))
		if err != nil {
			return nil, fmt.Errorf("unreadable block key %q: %w", at, err)
		}

		for _, name := range slices.Sorted(maps.Keys(held[at])) {
			one, ok := placedOne(path, pos, name, held[at][name], blocks, cat)
			if !ok {
				continue
			}

			out = append(out, one)
		}
	}

	return out, nil
}

// placedOne resolves a single assignment, or skips it.
//
// Skipped rather than refused, the way the writer skips: both paths into a
// preset have already been through check, where a block the chain does not
// have and a control the model does not carry are reported with every other
// complaint about the rig. Anything that does not line up here is a preset
// somebody hand-edited, and writing it would put a controller on whatever
// happens to sit at that number.
func placedOne(
	path, pos int,
	name string,
	attrs map[string]any,
	blocks []plan.Block,
	cat *catalog.Catalog,
) (wire.PlacedController, bool) {
	b, ok := blockAtPath(blocks, path, pos)
	if !ok {
		return wire.PlacedController{}, false
	}

	model, ok := cat.SymbolNumber(b.Model)
	if !ok {
		return wire.PlacedController{}, false
	}

	sym, _ := cat.Symbol(model)

	param := slices.Index(sym.Params, name)
	if param < 0 {
		return wire.PlacedController{}, false
	}

	number, ok := attrs[ctlNumber].(float64)
	if !ok {
		return wire.PlacedController{}, false
	}

	lo, hi := travelOf(attrs, b.Model, name, cat)

	out := wire.PlacedController{
		Controller: int(number),
		Block:      pos,
		Param:      param,
		Min:        lo,
		Max:        hi,
	}

	out.NoSnapshot, _ = attrs[ctlNoSnapshot].(bool)

	return out, true
}

// blockAtPath finds the block at a position on one processor.
//
// By position along the path rather than by place in the list, because a chain
// read off a device numbers its blocks the way the device laid them out and one
// that states its positions leaves gaps in them.
func blockAtPath(
	blocks []plan.Block,
	path, pos int,
) (plan.Block, bool) {
	for _, b := range blocks {
		if b.DSP == path && b.Pos == pos {
			return b, true
		}
	}

	return plan.Block{}, false
}

// travelOf is the ends of a controller's sweep.
//
// A preset may leave either out, and what it means by that is the parameter's
// own range: at rest it reads the bottom of the knob and all the way over it
// reads the top. Zero and one would be right for most parameters here and
// wrong for every one measured in hertz or decibels.
func travelOf(
	attrs map[string]any,
	model catalog.ModelID,
	name string,
	cat *catalog.Catalog,
) (float64, float64) {
	lo, hi := 0.0, 1.0

	if blk, ok := cat.Block(model); ok {
		if p, ok := blk.Params[name]; ok {
			lo, hi = p.Min, p.Max
		}
	}

	if v, ok := attrs[ctlMin].(float64); ok {
		lo = v
	}

	if v, ok := attrs[ctlMax].(float64); ok {
		hi = v
	}

	return lo, hi
}
