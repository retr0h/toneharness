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

package compile_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// SnapshotsPublicTestSuite covers what a footswitch recalls.
type SnapshotsPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *SnapshotsPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// presetWith returns a document holding the given tone entries.
func (s *SnapshotsPublicTestSuite) presetWith(
	entries map[string]string,
) *preset.Document {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	for key := range doc.Data.Tone {
		if key != "dsp0" && key != "dsp1" {
			delete(doc.Data.Tone, key)
		}
	}

	for key, body := range entries {
		var entry preset.Tone
		s.Require().NoError(json.Unmarshal([]byte(body), &entry))

		doc.Data.Tone[key] = entry
	}

	doc.Data.Tone["dsp0"]["block0"] = json.RawMessage(
		`{"@model": "HD2_AmpSVBeastNrm", "@position": 0, "@enabled": true}`)

	return doc
}

// TestLiftSnapshots reads what a footswitch recalls.
func (s *SnapshotsPublicTestSuite) TestLiftSnapshots() {
	tests := []struct {
		name    string
		entries map[string]string
		// the names the snapshots must carry, in order.
		want []string
		// a snapshot naming nothing, with a tempo of its own.
		unnamed bool
		tempo   float64
		// no snapshots at all.
		none bool
	}{
		{
			// Not the order a map happens to iterate: snapshot10 comes after
			// snapshot2, and reading them by name would put it second.
			name: "snapshots in the order they are numbered",
			entries: map[string]string{
				"snapshot2":  `{"@name": "Third"}`,
				"snapshot10": `{"@name": "Eleventh"}`,
				"snapshot0":  `{"@name": "First"}`,
			},
			want: []string{"First", "Third", "Eleventh"},
		},
		{
			name:    "an entry nothing can number",
			entries: map[string]string{"snapshotX": `{"@name": "Nope"}`},
			none:    true,
		},
		{
			// A rig somebody edited can put anything here. A field that
			// cannot be read is left unset rather than written as a zero,
			// which would claim the device said something it did not.
			name:    "a field that will not read",
			entries: map[string]string{"snapshot0": `{"@name": 7, "@tempo": 120}`},
			unnamed: true,
			tempo:   120,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, _, made, err := compile.Lift(s.presetWith(tt.entries), s.cat)
			s.Require().NoError(err)

			if tt.none {
				s.Require().Empty(made.Snapshots)

				return
			}

			if tt.unnamed {
				snap := made.Snapshots[0]

				s.Require().Nil(snap.Name)
				s.Require().InDelta(tt.tempo, *snap.Tempo, 0.001)

				return
			}

			s.Require().Len(made.Snapshots, len(tt.want))

			for i, want := range tt.want {
				s.Require().Equal(want, *made.Snapshots[i].Name)
			}
		})
	}
}

// TestASnapshotKeepsWhatThisFormatDoesNotModel is what makes a round trip
// lossless.
//
// A handful of presets record commands in a snapshot, and Line 6 can add a
// field to the format at any release. Anything unrecognised is carried in
// Rest rather than dropped, so a preset read and written back is the preset
// that went in. Without it the first new field Line 6 ships would be silently
// deleted by every plan that passed through here.
func (s *SnapshotsPublicTestSuite) TestASnapshotKeepsWhatThisFormatDoesNotModel() {
	doc := s.presetWith(map[string]string{
		"snapshot0": `{"@name": "Verse", "@ledcolor": 3, ` +
			`"commands": [{"cc": 41}], "@somethingNew": "from a later release"}`,
	})

	_, _, made, err := compile.Lift(doc, s.cat)
	s.Require().NoError(err)
	s.Require().Len(made.Snapshots, 1)

	rest := made.Snapshots[0].Rest
	s.Require().NotNil(rest, "an unmodelled key is carried, not dropped")
	s.Require().Contains(*rest, "commands")
	s.Require().Contains(*rest, "@somethingNew")

	// And back again, because carrying it and never writing it out is the
	// same loss one step later.
	into, err := preset.Blank()
	s.Require().NoError(err)

	s.Require().NoError(compile.Lower(into, made, s.cat))

	back := into.Data.Tone["snapshot0"]
	s.Require().Contains(back, "commands")
	s.Require().Contains(back, "@somethingNew")
	s.Require().JSONEq(`"from a later release"`, string(back["@somethingNew"]))
}

// TestLowerSnapshots writes a plan's snapshots over the preset's own.
//
// An untouched preset carries three of its own, and keeping those beside a
// plan's would rebuild a preset holding snapshots nobody made.
func (s *SnapshotsPublicTestSuite) TestLowerSnapshots() {
	name := "Verse"

	tests := []struct {
		name string
		// a plan lifted off a preset holding this one snapshot, or one
		// somebody typed.
		lifted string
		typed  *plan.Plan
		want   string
	}{
		{
			name:   "a plan lifted off a preset",
			lifted: "Only",
			want:   `"@name": "Only"`,
		},
		{
			// A plan somebody typed carries no record of a device, so nothing
			// has already cleared the preset it is built into. Its snapshots
			// still have to replace the three an untouched preset ships with.
			name: "a plan somebody typed",
			typed: &plan.Plan{
				Name: "typed",
				Blocks: []plan.Block{
					{Model: "HD2_AmpSVBeastNrm", Pos: 0, Enabled: true},
				},
				Snapshots: []rig.Snapshot{{Name: &name}},
			},
			want: `"@name": "Verse"`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var made plan.Plan

			if tt.typed != nil {
				made = *tt.typed
			} else {
				_, _, lifted, err := compile.Lift(s.presetWith(map[string]string{
					"snapshot0": `{"@name": "` + tt.lifted + `"}`,
				}), s.cat)
				s.Require().NoError(err)

				made = lifted
			}

			doc, err := preset.Blank()
			s.Require().NoError(err)
			s.Require().NoError(compile.Lower(doc, made, s.cat))

			var out bytes.Buffer
			s.Require().NoError(preset.Write(&out, doc))

			s.Require().Contains(out.String(), tt.want)
			s.Require().NotContains(out.String(), "SNAPSHOT 2",
				"the preset's own snapshots are not kept beside the plan's")
		})
	}
}

// TestTrust covers making a preset's snapshots ones the device believes.
//
// Task 278, and the whole of what it was. Block bypass is snapshot state, so a
// pedal loading a preset whose snapshots are invalid has nothing to recall and
// initialises them from whatever the chain happens to be. Every footswitch on
// the preset then takes three presses before it works, and after that works
// for good, which is why it read as a decode fault for weeks.
//
// One method and one table, so a case is a row rather than a file.
func (s *SnapshotsPublicTestSuite) TestTrust() {
	on := plan.Block{Model: "HD2_AmpSVBeastNrm", Pos: 0, Enabled: true}
	off := plan.Block{Model: "HD2_DistHedgehogD9", Pos: 2, Enabled: false}

	for _, tt := range []struct {
		name    string
		entries map[string]string
		blocks  []plan.Block
		// valid is what each snapshot's @valid must read, by key.
		valid map[string]bool
		// states is what the snapshot's block record must hold, by key.
		// Absent means the record must be left alone.
		states map[string]map[string]bool
	}{
		{
			// The blank an untouched preset ships with, which is what a
			// built preset inherits: neither valid nor carrying a record.
			name: "three blank snapshots get the chain",
			entries: map[string]string{
				"snapshot0": `{"@valid": false}`,
				"snapshot1": `{"@valid": false}`,
				"snapshot2": `{"@valid": false}`,
			},
			blocks: []plan.Block{on, off},
			valid:  map[string]bool{"snapshot0": true, "snapshot1": true, "snapshot2": true},
			states: map[string]map[string]bool{
				"snapshot0": {"block0": true, "block2": false},
			},
		},
		{
			// A field absent is the same as false here: an untouched preset
			// omits it rather than writing one.
			name: "a snapshot that says nothing about being valid",
			entries: map[string]string{
				"snapshot0": `{"@name": "SNAPSHOT 1"}`,
			},
			blocks: []plan.Block{off},
			valid:  map[string]bool{"snapshot0": true},
			states: map[string]map[string]bool{"snapshot0": {"block2": false}},
		},
		{
			// **One captured snapshot and none of them are touched.**
			// Somebody setting up even a single one has made a musical
			// decision, and seeding all three from the chain would throw it
			// away: a preset that arrived with three different sounds would
			// come back holding one.
			name: "a preset somebody has captured a snapshot into",
			entries: map[string]string{
				"snapshot0": `{"@valid": true, "blocks": {"dsp0": {"block0": false}}}`,
				"snapshot1": `{"@valid": false}`,
			},
			blocks: []plan.Block{on, off},
			valid:  map[string]bool{"snapshot0": true, "snapshot1": false},
			states: map[string]map[string]bool{
				"snapshot0": {"block0": false},
			},
		},
		{
			// Nothing to make believable. A preset keeping no snapshots is
			// not a caller's mistake.
			name:    "a preset with no snapshots at all",
			entries: map[string]string{},
			blocks:  []plan.Block{on},
			valid:   map[string]bool{},
		},
		{
			// A chain of nothing still marks them valid: three snapshots
			// recalling an empty grid is a true record, and leaving them
			// invalid is what costs the footswitches.
			name:    "a chain with no blocks",
			entries: map[string]string{"snapshot0": `{"@valid": false}`},
			blocks:  nil,
			valid:   map[string]bool{"snapshot0": true},
		},
	} {
		s.Run(tt.name, func() {
			doc := s.presetWith(tt.entries)

			// Through the Compiler rather than the package function, because
			// that is how both build paths reach it: a step beside Sections
			// and Footswitches, since `presets compile` goes through Lower
			// and `presets make` calls the steps one at a time.
			(&compile.Compiler{}).Trust(doc, tt.blocks)

			for key, want := range tt.valid {
				entry, held := doc.Data.Tone[key]
				s.Require().True(held, "%s is gone", key)

				var got bool
				s.Require().NoError(json.Unmarshal(entry["@valid"], &got))
				s.Require().Equal(want, got, "%s @valid", key)
			}

			for key, want := range tt.states {
				var record map[string]map[string]bool
				s.Require().NoError(
					json.Unmarshal(doc.Data.Tone[key]["blocks"], &record))
				s.Require().Equal(want, record["dsp0"], "%s blocks", key)
			}

			// A snapshot nobody named is not invented. Only the entries the
			// document came with are written, because how many a device holds
			// is the device's to say.
			if len(tt.entries) == 0 {
				for key := range doc.Data.Tone {
					s.Require().Negative(preset.SnapshotIndex(key),
						"%s was invented", key)
				}
			}
		})
	}
}

func TestSnapshotsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SnapshotsPublicTestSuite))
}
