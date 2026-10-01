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

package tools

import (
	"context"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/slot"
)

// Client is what the tools call. FromSDK makes one of an *sdk.Client.
//
// Declared here, where it is used, so a test can put a generated double in
// front of the handlers without a pedal on the bus.
type Client interface {
	Blocks(ctx context.Context, f sdk.Filter) (sdk.Blocks, error)
	Block(ctx context.Context, id string) (catalog.Block, error)
	ModelMeasurements(ctx context.Context, model string) (sdk.Measured, error)
	ChainMeasurements(ctx context.Context, instrument string) (sdk.Measured, error)
	Rigs(ctx context.Context) (sdk.Rigs, error)
	Rig(ctx context.Context, id string) (sdk.Rig, error)
	Backing(ctx context.Context, corpus string) ([]sdk.Backing, error)
	Scaffold(ctx context.Context, in sdk.NewRig) (sdk.Scaffolded, error)
	Tone(ctx context.Context, in sdk.Ask) (sdk.Resolved, error)
	Make(ctx context.Context, in sdk.Build) (sdk.Made, error)
	Compile(ctx context.Context, in sdk.Compile) (sdk.Built, error)
	// The music corpus, read as files rather than measured: a listing costs
	// nothing where measuring costs minutes per record.
	MusicPlayers(ctx context.Context, corpus string) ([]sdk.MusicPlayer, error)
	MusicBands(ctx context.Context, corpus string) ([]sdk.MusicGroup, error)
	MusicGenres(ctx context.Context, corpus string) ([]sdk.MusicGroup, error)
	MusicRecords(ctx context.Context, corpus string) ([]sdk.MusicRecord, error)
	// These read the recordings, which costs minutes per record.
	MeasuredGenres(ctx context.Context, corpus string) ([]audio.Genre, error)
	MeasuredPlayers(ctx context.Context, corpus string) ([]audio.Player, error)
	MeasuredRecordings(ctx context.Context, dir string) ([]audio.Named, audio.Across, error)
	Devices(ctx context.Context) (sdk.Attached, error)
	// Open claims the pedal. The tools hold what it returns across calls.
	Open(ctx context.Context) (Session, error)
}

// Session is what the device tools call while the pedal is held.
// *sdk.Session satisfies it.
type Session interface {
	Presets(ctx context.Context, setlist int) (sdk.Listing, error)
	Preset(ctx context.Context, at slot.Address) (sdk.Reading, error)
	Export(
		ctx context.Context,
		at slot.Address,
		out string,
		as sdk.Format,
		existing sdk.Existing,
	) (sdk.Written, error)
	Import(ctx context.Context, file string, at slot.Address) (sdk.Change, error)
	Copy(ctx context.Context, from, to slot.Address) (sdk.Change, error)
	Swap(ctx context.Context, a, b slot.Address) (sdk.Change, error)
	Select(ctx context.Context, at slot.Address) (sdk.Change, error)
	// Current is what the pedal is playing, including a live edit that was
	// never stored: the only way to see what a turn actually did.
	Current(ctx context.Context, as sdk.Format) (sdk.Reading, error)
	// Play replaces what is in front of somebody and writes no flash, which is
	// how a preset is tried rather than kept.
	Play(ctx context.Context, file string) error
	// Turn moves one control on what is playing, as a hand does. Writes
	// nothing, so the next preset selection undoes it.
	Turn(ctx context.Context, at sdk.Address, value float32) error
	Choose(ctx context.Context, at sdk.Address, value int) error
	Switch(ctx context.Context, at sdk.Address, on bool) error
	Close() error
}

// Search narrows catalog_list.
type Search struct {
	Category    string `json:"category,omitempty"    jsonschema:"only blocks of one kind, such as amp, cab, drive, dynamics, eq, delay, reverb or modulation"`
	Subcategory string `json:"subcategory,omitempty" jsonschema:"Line 6's own grouping, such as Guitar or Bass"`
	Search      string `json:"search,omitempty"      jsonschema:"text in the block's name or in the real-world gear it models"`
}

// ID names one thing: a model for catalog_show and corpus_presets_show, a rig
// for rigs_show.
type ID struct {
	ID string `json:"id" jsonschema:"the identifier, such as HD2_AmpSVBeastBrt for a model or mike-dirnt for a rig"`
}

// None is the input of a tool that takes nothing.
type None struct{}

// Corpus names a tree of recordings, one directory per player under the
// instrument they play.
//
// One instrument, never the whole tree: a bass centroid sits an octave below a
// guitar's, and a corpus holding both earns every bassist "dark" and means
// nothing by it.
type Corpus struct {
	Corpus string `json:"corpus" jsonschema:"a directory holding one instrument's players, such as resources/music/bass"`
}

// Instrument is which instrument's measured presets to answer about.
type Instrument struct {
	Instrument string `json:"instrument" jsonschema:"bass or guitar"`
}

// Make says what presets_make builds from and where the file goes.
type Make struct {
	RigID   string `json:"rig_id,omitempty"   jsonschema:"a rig that ships, by identifier; see rigs_list"`
	RigPath string `json:"rig_path,omitempty" jsonschema:"a rig file on disk"`
	Out     string `json:"out"                jsonschema:"where to write the .hlx"`
}

// Asked says which two documents tone_build resolves.
//
// No `out`, unlike presets_make. This answers with the rig rather than writing
// it, because an agent that has the rig can decide what to do with it, and a
// tool that wrote a file would need the write permission for something that
// produces no file of its own.
type Asked struct {
	Spec  string `json:"spec"            jsonschema:"a ToneSpec file: what somebody wants to sound like"`
	Setup string `json:"setup,omitempty" jsonschema:"a Setup file: what they own. Optional; the answer says what it assumed without one"`
}

// Outcome is what presets_make answers: exactly one side is set.
//
// One type on both sides, because both resolve. Which side says where the rig
// came from, not how much of it was honoured: a rig named by identifier and a
// rig file handed over are both built against the ask beside them.
type Outcome struct {
	FromShipped *sdk.Made `json:"from_shipped,omitempty"`
	FromRig     *sdk.Made `json:"from_rig,omitempty"`
}

// Model is one model as corpus_presets_show answers it: the block, and how players
// set it. The whole corpus and catalog stay out, because an agent reading them
// would read nothing else.
type Model struct {
	Block  catalog.Block                `json:"block"`
	Uses   int                          `json:"uses"`
	Params map[string]corpus.ParamStats `json:"params"`
}

// Turn says which control to move on what the pedal is playing, and to what.
//
// A device does not coerce, so the kind of value has to match the parameter:
// exactly one of value, choice or switch. `block` is the position
// presets_show reports, not the slot the wire wants; the arithmetic between
// them is this tool's.
type Turn struct {
	Block  int      `json:"block"            jsonschema:"which block, by the position presets_show reports for it"`
	Param  int      `json:"param"            jsonschema:"the parameter's place in that model's own list, which measure names holds to the hardware"`
	Value  *float32 `json:"value,omitempty"  jsonschema:"a number on a dial, in the parameter's own units"`
	Choice *int     `json:"choice,omitempty" jsonschema:"one of a list, such as a cabinet's microphone"`
	Switch *bool    `json:"switch,omitempty" jsonschema:"on or off, such as an amplifier's Bright"`
	Model  int      `json:"model,omitempty"  jsonschema:"0 for the block's own model, 1 for a cabinet fused into an amplifier"`
	Direct *bool    `json:"direct,omitempty" jsonschema:"omitted addresses the parameter the ordinary way; false reaches the value some blocks carry past their list"`
}

// Recordings is a directory of separated audio to measure.
type Recordings struct {
	Dir string `json:"dir" jsonschema:"a directory of separated recordings, such as the output of the stems recipe"`
}

// Recorded is what measure_recordings answers: each recording, and what they
// measure as together.
type Recorded struct {
	Tracks   []audio.Named `json:"tracks"`
	Together audio.Across  `json:"together"`
}

// Scaffold says what rig to write, from the gear it names.
//
// The gear is checked against the catalog before anything is written, because a
// rig naming gear no device models is otherwise only found out when somebody
// tries to build from it, by which point the name has usually been copied
// somewhere else too.
type Scaffold struct {
	ID         string   `json:"id"               jsonschema:"the identifier, and the filename stem"`
	Name       string   `json:"name"             jsonschema:"the player or style, as a person would write it; read off the id when absent"`
	Band       string   `json:"band,omitempty"   jsonschema:"the group, where there is one"`
	Instrument string   `json:"instrument"       jsonschema:"guitar or bass"`
	Amp        string   `json:"amp"              jsonschema:"the real-world amplifier; the one thing nothing downstream recovers from getting wrong"`
	Cab        string   `json:"cab,omitempty"    jsonschema:"the real-world cabinet; empty takes the amp's own pairing"`
	Pedals     []string `json:"pedals,omitempty" jsonschema:"real-world pedals, in signal order"`
	Genre      []string `json:"genre"            jsonschema:"which genres the sound belongs to, such as punk; required, because the ask written beside the rig carries one"`
}

// Build says which document presets_compile writes into a preset.
//
// A rig names gear and is realised against the catalog on the way through; a
// plan already names which model answered and what every knob is set to, which
// is what somebody who exported a slot and tuned it by hand has. One or the
// other, never both.
type Build struct {
	Rig      string `json:"rig,omitempty"      jsonschema:"a rig file to build"`
	Plan     string `json:"plan,omitempty"     jsonschema:"a plan file to build, for work that has already chosen its models"`
	Template string `json:"template,omitempty" jsonschema:"a preset to write the chain into; empty uses an untouched one the device wrote"`
	Out      string `json:"out"                jsonschema:"where to write the .hlx"`
}

// Play is a preset to put in front of somebody without storing it.
type Play struct {
	Preset string `json:"preset" jsonschema:"a .hlx file to play; nothing is written to a slot"`
}

// Slot addresses one slot on the pedal.
type Slot struct {
	Slot string `json:"slot" jsonschema:"a slot as the pedal labels it, 01A to 42C"`
}

// Export says which slot slots_export writes out, where, and as what.
type Export struct {
	Slot string `json:"slot"         jsonschema:"a slot as the pedal labels it, 01A to 42C"`
	Out  string `json:"out"          jsonschema:"where to write the file"`
	As   string `json:"as,omitempty" jsonschema:"empty for a rig, which reads on other hardware; hlx for the device's own file"`
}

// Shown is a slot as presets_show answers it. The .hlx document stays out: the
// rig is what an agent reasons about, and slots_export writes the rest.
type Shown struct {
	Name   string      `json:"name"`
	Rig    rig.Spec    `json:"rig"`
	Answer *sdk.Answer `json:"answer,omitempty"`
}

// Put says which .hlx slots_import places, and where.
type Put struct {
	Preset string `json:"preset" jsonschema:"the .hlx file to place"`
	Slot   string `json:"slot"   jsonschema:"a slot as the pedal labels it, 01A to 42C"`
}

// Move names the two slots slots_copy and slots_swap work on.
type Move struct {
	From string `json:"from" jsonschema:"the source slot, 01A to 42C"`
	To   string `json:"to"   jsonschema:"the destination slot, 01A to 42C"`
}
