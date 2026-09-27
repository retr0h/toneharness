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

// Package plan is a signal chain realised for one device, and the checks that
// say whether the device will load it.
//
// A Plan is the compiler's intermediate form, not a format. Nobody authors
// one, nothing exchanges one, and it has no schema: it exists between
// resolving a [RigSpec] against a catalog and writing a preset file. What is
// authored and shared is a RigSpec, which names real-world gear by its
// real-world name; what a device loads is a preset. This sits between them and
// is deliberately device-bound, holding one manufacturer's model identifiers
// and one manufacturer's parameter keys.
//
// [RigSpec]: https://github.com/retr0h/tonestack/blob/main/pkg/sdk/rig/data/rigspec.openapi.yaml
package plan

import (
	"encoding/json"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Params are parameter values keyed by the device's own parameter key.
type Params map[string]catalog.ParamValue

// Every type here carries tags, because a plan is written out and read back.
//
// Not a contract, though. The two contracts in this SDK are the formats a person
// authors, and a plan is written by a driver and read by the compiler. The
// nearest thing to it is pkg/sdk/preset, the .hlx document, which is hand-written
// Go with tags for the same reason. Load refuses a field it does not know, which
// is the one thing a contract would have bought.

// Block is one model in a chain, positioned.
type Block struct {
	// Model is a device-internal identifier such as HD2_AmpSVBeastNrm.
	// Validity is decided against a catalog, not here.
	Model catalog.ModelID `json:"model" yaml:"model"`
	// Params holds what every knob is set to.
	Params Params `json:"params,omitempty" yaml:"params,omitempty"`
	// DSP is which signal path this block occupies, from zero. Devices with
	// one path only ever use zero.
	DSP int `json:"dsp" yaml:"dsp"`
	// Pos is the position within its path. Positions on one path must form
	// the contiguous run 0..n-1.
	Pos int `json:"pos" yaml:"pos"`
	// Enabled says whether the block is doing anything. A bypassed block
	// still occupies its position and still costs DSP.
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Attrs holds the device attributes a chain has no opinion about, kept
	// so a preset written back out is the one that was read.
	Attrs map[string]json.RawMessage `json:"attrs,omitempty" yaml:"attrs,omitempty"`
}

// Snapshot is one set of parameter overrides a device can switch between
// without changing preset.
//
// Device-bound, like everything else here: a RigSpec expresses the same idea
// as variants, in gear terms.
type Snapshot struct {
	// Name is what the device shows for it.
	Name string `json:"name" yaml:"name"`
	// Overrides are keyed by block index, as a string, because that is how
	// the preset format stores them.
	Overrides map[string]Params `json:"overrides,omitempty" yaml:"overrides,omitempty"`
}

// Plan is a rig realised on one device.
//
// The third layer. A ToneSpec is what somebody meant, a RigSpec is the gear that
// answers it named the way a musician names it, and this is that rig fitted to
// hardware: which model each piece of gear resolved to, where it sits in the
// DSP, and everything else that only means anything on a pedal.
type Plan struct {
	// Name is what the preset will be called.
	Name string `json:"name" yaml:"name"`
	// Rig is the identifier of the rig this realises, where one was read.
	//
	// A name rather than a copy. The gear and the evidence stay in the rig, so
	// a plan that embedded them would be a second place for a citation to live
	// and drift.
	Rig string `json:"rig,omitempty" yaml:"rig,omitempty"`
	// Blocks are in the order the device runs them.
	Blocks []Block `json:"blocks" yaml:"blocks"`
	// Snapshots are the switchable parameter sets, if any.
	Snapshots []Snapshot `json:"snapshots,omitempty" yaml:"snapshots,omitempty"`
	// Footswitches is what the pedal prints under each switch.
	//
	// Somebody's decision about how they play, not device state: nothing else
	// records that the switch under your foot says Drive rather than naming the
	// block.
	Footswitches []rig.Footswitch `json:"footswitches,omitempty" yaml:"footswitches,omitempty"`
	// Controllers is what an expression pedal or a footswitch moves.
	Controllers []rig.Controller `json:"controllers,omitempty" yaml:"controllers,omitempty"`
	// Device is the state a lifted preset arrived with, kept so one written
	// back out is the one that was read.
	Device *rig.DeviceState `json:"device,omitempty" yaml:"device,omitempty"`
	// Target is the device this was tuned for, and how.
	//
	// Advisory rather than a restriction. Somebody with other hardware can
	// still load it, and should know the tuning was not done for them.
	Target *rig.Target `json:"target,omitempty" yaml:"target,omitempty"`
}

// BlockLookup is whatever can say what a model is.
//
// A catalog satisfies it. Taking the interface rather than the catalog keeps
// validation testable against a handful of blocks instead of six hundred.
type BlockLookup interface {
	Block(id catalog.ModelID) (catalog.Block, bool)
}

// Limits are what one device will accept.
type Limits struct {
	// MaxBlocks is the most blocks the device will hold.
	MaxBlocks int
	// Paths is how many signal paths the device has.
	//
	// An HX Stomp has one: no preset in a corpus of 714 uses a second, while
	// 73% of Helix Floor presets do. A chain that does not fit a device with
	// one path has nowhere to overflow to and must be refused.
	Paths int
	// ChipCeiling is how much of one processor a chain may occupy, in the
	// same units the catalog states a block's cost: percent. Line 6 records
	// an Ampeg SVT at 26.67, meaning a quarter of a processor.
	//
	// It sits below 100 deliberately. A chain that exactly fills a processor
	// in theory is a chain that fails to load in practice.
	ChipCeiling float64
}
