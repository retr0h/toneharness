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
package preset

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// Write encodes a preset file.
func Write(
	w io.Writer,
	d *Document,
) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(d); err != nil {
		return fmt.Errorf("encoding preset: %w", err)
	}

	return nil
}

// SetSpec replaces the document's signal chain with spec.
//
// Only processor blocks are rewritten. Routing, snapshots and controller
// assignments are left as they were, because a chain says nothing about them
// and discarding what the device wrote would produce a preset that loads
// differently for reasons nobody asked for.
//
// Every processor loses its blocks, including one spec puts nothing on. A
// template's blocks on a processor the rig does not use would otherwise be
// compiled in beside the rig's own chain.
func (d *Document) SetSpec(
	spec plan.Plan,
) error {
	d.Data.Meta.Name = spec.Name

	if d.Data.Tone == nil {
		d.Data.Tone = map[string]Tone{}
	}

	for key, tone := range d.Data.Tone {
		if !strings.HasPrefix(key, processorPrefix) {
			continue
		}

		for k := range tone {
			if IsBlockKey(k) {
				delete(tone, k)
			}
		}
	}

	byProcessor := map[int][]plan.Block{}
	for _, b := range spec.Blocks {
		byProcessor[b.DSP] = append(byProcessor[b.DSP], b)
	}

	for dsp, blocks := range byProcessor {
		key := processorPrefix + strconv.Itoa(dsp)

		tone := d.Data.Tone[key]
		if tone == nil {
			tone = Tone{}
		}

		for i, b := range blocks {
			raw, err := encodeBlock(b)
			if err != nil {
				return fmt.Errorf("%s block %d: %w", key, i, err)
			}

			tone["block"+strconv.Itoa(b.Pos)] = raw
		}

		d.Data.Tone[key] = tone
	}

	return nil
}

// processorPrefix is what the tone entry of every processor is named with,
// dsp0 and dsp1 on a device that has two.
const processorPrefix = "dsp"

// IsProcessorKey reports whether a tone entry is a processor.
//
// Which decides whether its blocks belong to the chain. A processor's `block3` is
// a chain entry; a footswitch's `block3` and a snapshot's `block3` are what that
// switch or snapshot does about it, and those are not gear.
func IsProcessorKey(
	k string,
) bool {
	return strings.HasPrefix(k, processorPrefix)
}

// IsBlockKey reports whether a tone entry is a chain block rather than
// routing.
//
// Exported because a lift needs the same question answered. A chain already says
// every block, so a rig's record of everything else a preset holds has to leave
// these out or two parts of one document would set one control.
func IsBlockKey(
	k string,
) bool {
	if len(k) <= len("block") || k[:len("block")] != "block" {
		return false
	}

	_, err := strconv.Atoi(k[len("block"):])

	return err == nil
}

// encodeBlock renders one block as the device writes it: @-prefixed
// attributes alongside parameters, each parameter in its own kind.
func encodeBlock(
	b plan.Block,
) (json.RawMessage, error) {
	fields := map[string]any{
		attrModel:   string(b.Model),
		attrEnabled: b.Enabled,
	}

	// Position is an attribute rather than the block's key, so it is only
	// derived from the key when a chain carried none — which is the case for
	// a chain this tool built rather than read.
	if _, ok := b.Attrs[attrPosition]; !ok {
		fields[attrPosition] = b.Pos
	}

	for k, v := range b.Params {
		if _, taken := fields[k]; taken {
			return nil, fmt.Errorf("parameter %q collides with an attribute", k)
		}

		fields[k] = v
	}

	// Attributes the chain carried but has no opinion about, put back as
	// they arrived.
	for k, v := range b.Attrs {
		if _, taken := fields[k]; taken {
			continue
		}

		fields[k] = v
	}

	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encoding block: %w", err)
	}

	return raw, nil
}

// New returns a document for a device, carrying spec.
func New(
	deviceID int,
	spec plan.Plan,
) (*Document, error) {
	d := &Document{
		Schema:  Schema,
		Version: Version,
		Data:    Data{Device: deviceID, Tone: map[string]Tone{}},
	}

	if err := d.SetSpec(spec); err != nil {
		return nil, err
	}

	return d, nil
}
