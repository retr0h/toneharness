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

// Package plan is a rig realised for one device, and the checks that say
// whether the device will load it.
//
// The third of three layers. A [ToneSpec]'s ask is what somebody meant, its rig
// is the gear that answers it named the way a musician names it, and a Plan is
// that rig fitted to hardware: which model each piece of gear resolved to, where
// it sits in the DSP, what the footswitches do.
//
// Deliberately device-bound, holding one manufacturer's model identifiers and
// one manufacturer's parameter keys. Written out and read back, and strictly, so
// a field nobody defined is refused rather than dropped. Not a contract, though:
// the two contracts here are the formats a person authors, and a plan is written
// by a driver.
//
// [ToneSpec]: https://github.com/retr0h/toneharness/blob/main/pkg/sdk/tone/data/tonespec.openapi.yaml
package plan

import (
	"encoding/json"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// Params are parameter values keyed by the device's own parameter key.
type Params map[string]catalog.ParamValue

// Every type here carries JSON tags, because a plan is written out and read
// back, and Load reads YAML through JSON. JSON tags and no others, the way the
// rig types are: a YAML tag beside them would never be consulted, and a dead
// tag that looks load-bearing is worse than none.
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
	Model catalog.ModelID `json:"model"`
	// Params holds what every knob is set to.
	Params Params `json:"params,omitempty"`
	// DSP is which signal path this block occupies, from zero. Devices with
	// one path only ever use zero.
	DSP int `json:"dsp"`
	// Pos is the position within its path. Positions on one path must form
	// the contiguous run 0..n-1.
	Pos int `json:"pos"`
	// Enabled says whether the block is doing anything. A bypassed block
	// still occupies its position and still costs DSP.
	Enabled bool `json:"enabled"`
	// Attrs holds the device attributes a chain has no opinion about, kept
	// so a preset written back out is the one that was read.
	Attrs map[string]json.RawMessage `json:"attrs,omitempty"`
}

// Path is which side of a parallel split the block sits on, zero where it says
// nothing.
//
// Read out of the attributes rather than modelled as a field, because that is
// where a preset keeps it and where a lift puts it back. Two blocks may hold one
// position on one processor as long as they are on different sides, so this is
// half of what says where a block is.
func (b Block) Path() int {
	raw, ok := b.Attrs[attrPath]
	if !ok {
		return 0
	}

	var got int
	if err := json.Unmarshal(raw, &got); err != nil {
		return 0
	}

	return got
}

// attrPath is the device's name for which side of a split a block is on.
const attrPath = "@path"

// Plan is a rig realised on one device.
//
// The third layer. A document's ask is what somebody meant, its rig is the gear
// that answers it named the way a musician names it, and this is that rig fitted
// to hardware: which model each piece of gear resolved to, where it sits in the
// DSP, and everything else that only means anything on a pedal.
type Plan struct {
	// Name is what the preset will be called.
	Name string `json:"name"`
	// Rig is the identifier of the rig this realises, where one was read.
	//
	// A name rather than a copy. The gear and the evidence stay in the rig, so
	// a plan that embedded them would be a second place for a citation to live
	// and drift.
	Rig string `json:"rig,omitempty"`
	// Blocks are in the order the device runs them.
	Blocks []Block `json:"blocks"`
	// Snapshots are what the device switches between without changing preset.
	//
	// Kept as the device wrote them, which is why they are the contract's type
	// rather than one of this package's: a snapshot addresses blocks by the
	// device's own numbering and carries fields Line 6 may add to, so modelling
	// it further here would only find new ways to lose something.
	Snapshots []rig.Snapshot `json:"snapshots,omitempty"`
	// Footswitches is what the pedal prints under each switch.
	//
	// Somebody's decision about how they play, not device state: nothing else
	// records that the switch under your foot says Drive rather than naming the
	// block.
	Footswitches []rig.Footswitch `json:"footswitches,omitempty"`
	// Controllers is what an expression pedal or a footswitch moves.
	Controllers []rig.Controller `json:"controllers,omitempty"`
	// Device is the state a lifted preset arrived with, kept so one written
	// back out is the one that was read.
	Device *rig.DeviceState `json:"device,omitempty"`
	// Target is the device this was tuned for, and how.
	//
	// Advisory rather than a restriction. Somebody with other hardware can
	// still load it, and should know the tuning was not done for them.
	Target *rig.Target `json:"target,omitempty"`
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
