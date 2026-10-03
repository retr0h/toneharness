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

func TestSnapshotsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SnapshotsPublicTestSuite))
}
