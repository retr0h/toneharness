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
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// handlers holds what every tool shares.
type handlers struct {
	client Client
	// pedal holds a Session across the tools that reach the pedal, and
	// keeps two calls from claiming the editor interface at once.
	pedal *pedal
	// allowWrites is whether the server was started with --allow-writes. It
	// decides which tools are offered, and whether a file already on disk
	// may be written over.
	allowWrites bool
}

// Register adds toneharness's tools to a server.
//
// The tools that write to a pedal are added only when allowWrites is true, and
// without it no tool writes over a file already on disk. A device has no undo,
// nor does a file, and whoever starts the server decides.
//
// It returns what holds the pedal between device calls. Closing it lets the
// pedal go, and whoever runs the server closes it when the server stops.
func Register(
	s *gomcp.Server,
	c Client,
	allowWrites bool,
) io.Closer {
	return register(s, c, allowWrites, idleClose)
}

// register is Register, with how long the pedal stays held after a device
// call.
func register(
	s *gomcp.Server,
	c Client,
	allowWrites bool,
	idle time.Duration,
) io.Closer {
	h := &handlers{client: c, pedal: newPedal(c, idle), allowWrites: allowWrites}

	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "catalog_list",
		Description:  "Find blocks the device models, by name, real-world gear, category or instrument. Use this before naming any model: a model it does not find does not exist.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Blocks](),
	}, h.catalogList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "catalog_show",
		Description:  "One block's parameters, their ranges and defaults, and its DSP cost.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[catalog.Block](),
	}, h.catalogShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_presets_show",
		Description:  "How players set one model across measured presets: median and quartiles per parameter. A narrow spread is consensus; a wide one is taste.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[Model](),
	}, h.corpusPresetsShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "rigs_list",
		Description:  "The rigs that ship with toneharness.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Rigs](),
	}, h.rigsList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "rigs_show",
		Description:  "One shipped rig, and the rigs that extend it.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Rig](),
	}, h.rigsShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "tone_build",
		Description:  "Turn a ToneSpec and a Setup into the rig they describe. Gear named by hand resolves against the catalog; gear left unnamed is chosen by measuring a recording against every block. Read the notes: they say what it could not honour and what it assumed. Answers with the rig rather than writing a file.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Resolved](),
	}, h.toneBuild)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "tone_reach",
		Description:  "Say whether a rig could reach a genre's sound before spending a tuning run on it. Reads measurements already committed rather than touching a device, so it costs a second where tone tune costs five minutes with a pedal held throughout. Per axis: how far out the chain sits, whether a reading ever landed inside the target, and the most every control added together could move it. An axis marked out of reach is out of reach; one inside is only worth attempting, because that sum flatters the controls on purpose.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Reaching](),
	}, h.toneReach)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "presets_make",
		Description:  "Build a .hlx from a shipped rig or a rig file. Read what it added and what each character word moved before putting it on a pedal. Refuses a file already at out unless the server was started with --allow-writes.",
		Annotations:  &gomcp.ToolAnnotations{OpenWorldHint: new(false), DestructiveHint: new(true)},
		OutputSchema: mustOutputSchema[Outcome](),
	}, h.presetsMake)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_presets_chains",
		Description:  "What a chain of one instrument almost always holds, across the measured presets: how often each kind of block appears and which side of the amplifier it sits. What a build uses to place a block a rig did not name.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Measured](),
	}, h.corpusPresetsChains)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_music_players",
		Description:  "Who the music corpus holds records for, and how many each. A file read: it costs nothing, where measuring the same records costs minutes apiece.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[[]sdk.MusicPlayer](),
	}, h.corpusMusicPlayers)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_music_bands",
		Description:  "Which bands made the records in the corpus, grouped on one slug so two spellings of a name count once.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[[]sdk.MusicGroup](),
	}, h.corpusMusicBands)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_music_genres",
		Description:  "Which genres are tagged in the corpus, how many records carry each and how many different players those come from. The player count is the half usually short, and a fourth album by one band cannot fix it.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[[]sdk.MusicGroup](),
	}, h.corpusMusicGenres)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_music_records",
		Description:  "Every recording the corpus names, with the year, the band and the genres its manifest gives it.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[[]sdk.MusicRecord](),
	}, h.corpusMusicRecords)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "rigs_records",
		Description:  "Every rig held to the era its ask claims. A record made outside that period measures other gear, so this reports which records fall outside it. It reports rather than refuses: which half is wrong is a judgement only somebody who knows the player can make.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[[]sdk.Backing](),
	}, h.rigsRecords)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "measure_genres",
		Description:  "What each genre measures as against the players who play none of it, and which words that earns. Reads the recordings, so it costs minutes per record; corpus_music_genres answers what is tagged without measuring anything. Point it at one instrument: a bass centroid sits an octave below a guitar's.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[[]audio.Genre](),
	}, h.measureGenres)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "measure_players",
		Description:  "What each player's records measure as, and the words that earns them against the others. A word is earned by sitting clear of the rest, so one player alone earns nothing. Reads the recordings, so it costs minutes per record; corpus_music_players answers what the corpus holds for a file read. Point it at one instrument.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[[]audio.Player](),
	}, h.measurePlayers)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "measure_recordings",
		Description:  "Measure a directory of recordings, one entry per file and the figures they make together. Separate the instrument out first: a mix measures the band, so a figure taken from one describes the arrangement rather than the player.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[Recorded](),
	}, h.measureRecordings)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "device_hardware",
		Description:  "The Line 6 Helix hardware attached over USB. HX Edit must be quit for any tool that reaches the pedal.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: true},
		OutputSchema: mustOutputSchema[sdk.Attached](),
	}, h.deviceHardware)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "slots_list",
		Description:  "Every slot on the attached pedal and what it holds.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: true},
		OutputSchema: mustOutputSchema[sdk.Listing](),
	}, h.slotsList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "presets_show",
		Description:  "One slot on the pedal, read back as a rig.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: true},
		OutputSchema: mustOutputSchema[Shown](),
	}, h.presetsShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "slots_export",
		Description:  "Write one slot to a file: a rig by default, or the device's own .hlx with as=hlx. Refuses a file already at out unless the server was started with --allow-writes.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: new(true)},
		OutputSchema: mustOutputSchema[sdk.Written](),
	}, h.slotsExport)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "device_select",
		Description:  "Load a slot on the pedal, as pressing its footswitch does. Changes nothing stored.",
		Annotations:  &gomcp.ToolAnnotations{DestructiveHint: new(false), IdempotentHint: true},
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.deviceSelect)

	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "device_current",
		Description:  "What the pedal is playing right now, read back as a rig. The only way to see a live edit: a turn is not stored, so nothing else shows what it did.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: true},
		OutputSchema: mustOutputSchema[Shown](),
	}, h.deviceCurrent)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "device_play",
		Description:  "Put a .hlx in front of somebody without storing it. Writes no flash and lasts until the next preset is selected, which is the right way to try something: prefer it to slots_import every time, because a slot is flash.",
		Annotations:  &gomcp.ToolAnnotations{DestructiveHint: new(false)},
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.devicePlay)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "device_turn",
		Description:  "Move one control on what the pedal is playing, as a hand does. Writes no flash, and the next preset selection undoes it. Read the chain back with presets_show first: a parameter has no name on the wire, only a position in the model's own list, and counting down a printed table mislabels every control while the numbers stay plausible.",
		Annotations:  &gomcp.ToolAnnotations{DestructiveHint: new(false)},
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.deviceTurn)

	if !allowWrites {
		return h.pedal
	}

	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "rigs_new",
		Description:  "Write a rig and the ask beside it from the gear it names, after checking the catalog carries that gear. Refuses a file already there unless the server was started with --allow-writes.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Scaffolded](),
	}, h.rigsNew)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "presets_compile",
		Description:  "Turn a rig file or a plan file into a .hlx. A rig names gear and is realised against the catalog on the way through; a plan already names the models and every knob, which is what an exported slot tuned by hand is. One or the other, never both.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Built](),
	}, h.presetsCompile)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "slots_import",
		Description:  "Put a .hlx into a slot on the pedal. Whatever the slot held is saved to a file first and then gone from the pedal.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.slotsImport)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "slots_copy",
		Description:  "Copy one slot onto another. The destination's old preset is saved to a file first.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.slotsCopy)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "slots_swap",
		Description:  "Exchange two slots. Both are saved to files first. One slot holding no preset makes it a move: the preset lands there and the slot it came from is emptied. Two slots holding no preset are refused.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.slotsSwap)

	return h.pedal
}

// mustOutputSchema infers T's output schema, correcting the types whose JSON
// reflection cannot see. It panics when T has no schema at all, which is a
// programming error found the moment the server starts.
//
// The SDK validates every structured result against this schema, so a schema
// narrower than what marshalling produces fails a real call rather than a test.
func mustOutputSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](&jsonschema.ForOptions{TypeSchemas: outputTypeSchemas()})
	if err != nil {
		panic(fmt.Sprintf("mustOutputSchema[%s]: %v", reflect.TypeFor[T](), err))
	}

	return s
}

// outputTypeSchemas are the schemas reflection gets wrong on a tool's output.
func outputTypeSchemas() map[reflect.Type]*jsonschema.Schema {
	// anyJSON is every JSON value, spelled out rather than left as {}: the
	// inference adds "null" to a pointer's types, and added to an empty list
	// that would leave null as the only value allowed.
	anyJSON := &jsonschema.Schema{
		Types: []string{"null", "boolean", "number", "string", "array", "object"},
	}

	return map[reflect.Type]*jsonschema.Schema{
		// A ParamValue marshals to a bare number, string or bool depending on
		// a kind it keeps in unexported fields, so reflection sees an empty
		// struct and would demand an object.
		reflect.TypeFor[catalog.ParamValue](): {},
		// A RawMessage is a []byte to reflection, an array of small integers,
		// but it marshals as the JSON it holds. rig.Spec carries these for
		// device state it keeps without modelling.
		reflect.TypeFor[json.RawMessage](): anyJSON,
		// plan.Block.Attrs and rig.Spec's kept device fields. Nil on a
		// built chain, so it marshals to null, which a map's inferred
		// object-only schema refuses.
		reflect.TypeFor[map[string]json.RawMessage](): {
			Types:                []string{"null", "object"},
			AdditionalProperties: anyJSON,
		},
	}
}

// readOnly marks a tool that changes nothing anywhere.
func readOnly() *gomcp.ToolAnnotations {
	return &gomcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)}
}

// destructive marks a tool that overwrites what a pedal holds.
func destructive() *gomcp.ToolAnnotations {
	return &gomcp.ToolAnnotations{DestructiveHint: new(true)}
}

// said is the one line of text beside a tool's structured answer, for an agent
// that reads text rather than structure.
func said(
	format string,
	args ...any,
) *gomcp.CallToolResult {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{&gomcp.TextContent{Text: fmt.Sprintf(format, args...)}},
	}
}
