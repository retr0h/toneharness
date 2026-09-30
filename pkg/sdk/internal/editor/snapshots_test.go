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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// SnapshotsTestSuite covers moving snapshots between a device and a file.
//
// A device records which blocks a snapshot switches by the position it lays
// them out at, and a preset records it by the entry each block is stored
// under. Those are different numbers, and a real preset proves it.
type SnapshotsTestSuite struct {
	suite.Suite
}

// awkward is a preset storing block5 at position 6 and block7 at position 5.
//
// Which is the case that catches reading a snapshot's block names as if the
// number in the name were the position. HX Edit wrote this one.
func (s *SnapshotsTestSuite) awkward() *preset.Document {
	raw, err := os.Open(
		filepath.Join("..", "compile", "testdata", "preset2.hlx"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(raw.Close()) }()

	doc, err := preset.Read(raw)
	s.Require().NoError(err)

	return doc
}

// blank is a preset with nothing written into it yet.
func (s *SnapshotsTestSuite) blank() *preset.Document {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	return doc
}

// TestSnapshotStates covers SnapshotStates, which reads what a preset's own
// snapshots recall.
//
// One method and one table, so a case is a row rather than a file.
func (s *SnapshotsTestSuite) TestSnapshotStates() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The mapping that matters.
			//
			// preset2.hlx stores block5 at position 6 and block7 at position
			// 5, so a reader trusting the name puts both on the wrong grid
			// position and the snapshot switches the wrong blocks.
			name: "a block is found by its position not its name",
			then: func() {
				got := SnapshotStates(s.awkward())

				s.Require().Len(got, 3)

				first := got[0]
				s.Require().NotNil(first.Name)
				s.Require().Equal("SNAPSHOT 1", *first.Name)
				s.Require().NotNil(first.Valid)
				s.Require().True(*first.Valid)

				// block5 sits at position 6, so the device's grid position 7 — and that
				// snapshot has it switched off.
				on, named := first.On[6+wire.GridOffset]
				s.Require().True(named, "block5 reached the grid")
				s.Require().False(on, "block5 is off in the first snapshot")

				// block7 sits at position 5, so grid position 6, and is switched on.
				on, named = first.On[5+wire.GridOffset]
				s.Require().True(named, "block7 reached the grid")
				s.Require().True(on, "block7 is on in the first snapshot")
			},
		},
		{
			// None.
			name: "a preset with no snapshots",
			then: func() {
				doc := s.blank()
				for key := range doc.Data.Tone {
					if preset.SnapshotIndex(key) >= 0 {
						delete(doc.Data.Tone, key)
					}
				}

				s.Require().Nil(SnapshotStates(doc))
			},
		},
		{
			// Has no record of the grid.
			name: "a snapshot naming nothing",
			then: func() {
				got := SnapshotStates(s.blank())

				s.Require().Len(got, 3)
				s.Require().Nil(got[0].On, "the template names no blocks")
				s.Require().NotNil(got[0].Name)
				s.Require().Equal("SNAPSHOT 1", *got[0].Name)
			},
		},
		{
			// Not read.
			//
			// A second processor names its entries the same way the first
			// does, so reading both would put dsp1's block0 on dsp0's grid.
			name: "a snapshot naming a second path",
			then: func() {
				doc := s.blank()
				doc.Data.Tone["snapshot0"][snapBlocks] = json.RawMessage(
					`{"dsp1": {"block0": true}}`)

				got := SnapshotStates(doc)
				s.Require().Nil(got[0].On)
			},
		},
		{
			// Left unset rather than read as a zero.
			name: "a field that will not read",
			then: func() {
				doc := s.blank()
				doc.Data.Tone["snapshot0"][snapTempo] = json.RawMessage(`"fast"`)
				doc.Data.Tone["snapshot0"][snapBlocks] = json.RawMessage(`"none"`)

				got := SnapshotStates(doc)

				s.Require().Nil(got[0].Tempo, "a tempo of zero is not what the file said")
				s.Require().Nil(got[0].On)
			},
		},
		{
			// Records nothing.
			//
			// A name that reaches no entry reaches no position either, and a
			// snapshot cannot switch a block that is not there.
			name: "a snapshot naming a block the preset has not",
			then: func() {
				doc := s.blank()
				doc.Data.Tone["snapshot0"][snapBlocks] = json.RawMessage(
					`{"dsp0": {"block9": true}}`)

				s.Require().Nil(SnapshotStates(doc)[0].On)
			},
		},
		{
			// Not one.
			name: "a tone entry named like a snapshot",
			then: func() {
				doc := s.blank()
				doc.Data.Tone["snapshotX"] = preset.Tone{}

				s.Require().Len(SnapshotStates(doc), 3)
			},
		},
		{
			// Left unset.
			name: "a field a preset omits",
			then: func() {
				doc := s.blank()
				delete(doc.Data.Tone["snapshot0"], snapName)

				s.Require().Nil(SnapshotStates(doc)[0].Name)
			},
		},
		{
			// What all of this is for.
			//
			// A device's own bytes, out to a preset the way an export writes
			// one and back to a device the way an import writes one. Before
			// this, the export wrote the template's three snapshots and the
			// import wrote every snapshot's record from the chain itself, so
			// a preset holding three sounds came back holding one, under
			// names nobody chose.
			name: "snapshots survive the round trip",
			then: func() {
				raw, err := os.ReadFile(filepath.Join("..", "wire", "testdata", "preset.bin"))
				s.Require().NoError(err)

				sent, err := wire.DecodePreset(raw)
				s.Require().NoError(err)

				cat, err := catalog.BuiltIn()
				s.Require().NoError(err)

				doc, empty, err := Document(sent, cat, "Round Trip")
				s.Require().NoError(err)
				s.Require().False(empty)

				blocks, err := Placements(doc, cat)
				s.Require().NoError(err)

				out, err := wire.Blank()
				s.Require().NoError(err)
				s.Require().NoError(wire.PlaceAsWritten(out, blocks))
				wire.PlaceSnapshots(out, SnapshotStates(doc))

				got, err := wire.DecodePreset(out.Encode())
				s.Require().NoError(err)
				s.Require().Len(got.Snapshots, len(sent.Snapshots))

				// The three differ from each other in this preset, which is what makes
				// the comparison worth making: four of the six block positions are
				// switched differently across them.
				for i, want := range sent.Snapshots {
					s.Require().Equal(want.Name, got.Snapshots[i].Name, "snapshot %d name", i)
					s.Require().Equal(want.Valid, got.Snapshots[i].Valid, "snapshot %d valid", i)
					s.Require().InDelta(want.Tempo, got.Snapshots[i].Tempo, 0.001,
						"snapshot %d tempo", i)

					// Only where the chain sits. What a device keeps its routing on is the
					// blank's own and was never the file's to carry.
					for _, b := range sent.Blocks {
						s.Require().Equal(
							want.On[b.Index], got.Snapshots[i].On[b.Index],
							"snapshot %d at grid position %d", i, b.Index)
					}
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestSnapshotsInto covers snapshotsInto, which writes what each of a
// device's snapshots recalls into a preset.
//
// One method and one table, so a case is a row rather than a file.
func (s *SnapshotsTestSuite) TestSnapshotsInto() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Writes what a device answered.
			name: "snapshots into a preset",
			then: func() {
				doc := s.blank()

				snapshotsInto(doc, wire.DevicePreset{
					Blocks: []wire.DeviceBlock{{Index: 2}, {Index: 3}},
					Snapshots: []wire.DeviceSnapshot{{
						Name:  "Verse",
						Tempo: 92,
						LED:   3,
						Valid: true,
						On:    map[int]bool{2: true, 3: false},
					}},
				})

				entry := doc.Data.Tone["snapshot0"]

				var name string
				s.Require().NoError(json.Unmarshal(entry[snapName], &name))
				s.Require().Equal("Verse", name)

				var valid bool
				s.Require().NoError(json.Unmarshal(entry[snapValid], &valid))
				s.Require().True(valid, "the template ships this false")

				// By the entry each block is stored under, which is its grid position
				// less the offset.
				var blocks map[string]map[string]bool
				s.Require().NoError(json.Unmarshal(entry[snapBlocks], &blocks))
				s.Require().Equal(
					map[string]map[string]bool{"dsp0": {"block1": true, "block2": false}},
					blocks)

				// What the template carried and this does not read is still there.
				s.Require().Contains(entry, "@pedalstate")
			},
		},
		{
			// Writes no block record.
			name: "snapshots into a preset without states",
			then: func() {
				doc := s.blank()

				snapshotsInto(doc, wire.DevicePreset{
					Blocks:    []wire.DeviceBlock{{Index: 2}},
					Snapshots: []wire.DeviceSnapshot{{Name: "Verse"}},
				})

				s.Require().NotContains(doc.Data.Tone["snapshot0"], snapBlocks)
			},
		},
		{
			// Writes no block record.
			name: "snapshots into a preset naming a block it has not",
			then: func() {
				doc := s.blank()

				snapshotsInto(doc, wire.DevicePreset{
					Blocks:    []wire.DeviceBlock{{Index: 9}},
					Snapshots: []wire.DeviceSnapshot{{On: map[int]bool{2: true}}},
				})

				s.Require().NotContains(doc.Data.Tone["snapshot0"], snapBlocks)
			},
		},
		{
			// Written into a preset that ships three.
			name: "a fourth snapshot",
			then: func() {
				doc := s.blank()

				snapshotsInto(doc, wire.DevicePreset{
					Snapshots: []wire.DeviceSnapshot{{}, {}, {}, {Name: "Fourth"}},
				})

				var name string
				s.Require().NoError(
					json.Unmarshal(doc.Data.Tone["snapshot3"][snapName], &name))
				s.Require().Equal("Fourth", name)
			},
		},
		{
			// The preset as it was.
			name: "no snapshots to write",
			then: func() {
				doc := s.blank()

				snapshotsInto(doc, wire.DevicePreset{})

				var name string
				s.Require().NoError(
					json.Unmarshal(doc.Data.Tone["snapshot0"][snapName], &name))
				s.Require().Equal("SNAPSHOT 1", name)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestAnEntryWithNoPosition is not on the grid.
func (s *SnapshotsTestSuite) TestAnEntryWithNoPosition() {
	doc := s.blank()
	doc.Data.Tone[processorKey]["block0"] = json.RawMessage(`{"@model": "x"}`)
	doc.Data.Tone[processorKey]["block1"] = json.RawMessage(`nonsense`)

	s.Require().NotContains(positionsOf(doc), "block0")
	s.Require().NotContains(positionsOf(doc), "block1")
}

func TestSnapshotsTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SnapshotsTestSuite))
}
