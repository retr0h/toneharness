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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// FootswitchesIntoTestSuite covers writing a device's switch assignments into
// a preset.
//
// Task 262. The reader found them and nothing wrote them, so
// `slots export --as hlx` produced `"footswitch": null` for a slot with
// working switches on it, and a round trip through this tool silently
// discarded somebody's own decisions about their own pedal.
type FootswitchesIntoTestSuite struct {
	suite.Suite
}

// switched is slot 09A as an HX Stomp sent it: `B15 Eras`, two amps carrying
// cabinets, and four assignments across three switches with names and colours
// somebody chose.
//
// A real preset rather than a fixture, because the thing under test is whether
// what this writes is what HX Edit writes, and only the device settles that.
func (s *FootswitchesIntoTestSuite) switched() wire.DevicePreset {
	raw, err := os.ReadFile(
		filepath.Join("..", "wire", "testdata", "switches.bin"))
	s.Require().NoError(err)

	got, err := wire.DecodePreset(raw)
	s.Require().NoError(err)

	return got
}

func (s *FootswitchesIntoTestSuite) catalog() *catalog.Catalog {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	return cat
}

// section reads the footswitch section a document carries, as a map.
func (s *FootswitchesIntoTestSuite) section(
	doc *preset.Document,
) map[string]map[string]any {
	held, ok := doc.Data.Tone[footswitchKey]
	s.Require().True(ok, "the document carries no footswitch section")

	body, ok := held[processorKey]
	s.Require().True(ok, "the section names no processor")

	out := map[string]map[string]any{}
	s.Require().NoError(json.Unmarshal(body, &out))

	return out
}

// TestFootswitchesInto covers what the section says, field for field.
//
// Against HX Edit's own export of a slot, which is the only thing that settles
// whether this tool's `.hlx` is the device's file or merely looks like one.
//
// One method and one table, so a case is a row rather than a file.
func (s *FootswitchesIntoTestSuite) TestFootswitchesInto() {
	doc, empty, err := Document(s.switched(), s.catalog(), "B15 Eras")
	s.Require().NoError(err)
	s.Require().False(empty)

	got := s.section(doc)

	tests := []struct {
		name string
		// at is the entry the block is stored under, which is not the grid
		// position the device records: the two differ by wire.GridOffset, and
		// this preset's blocks sit at 5, 6, 8 and 13.
		at string
		// want is every field, exactly as HX Edit writes them.
		want map[string]any
	}{
		{
			// Two blocks on one switch, which is the amplifier and the
			// cabinet of one Ampeg B-15NF toggled together. Both carry the
			// same switch and the same chosen name and colour, and differ in
			// the block's own colour and whether it is on.
			name: "the cabinet of a pair",
			at:   "block0",
			want: map[string]any{
				fsIndex:       float64(2),
				fsLabel:       "Ampeg B-15NF",
				fsCustomLabel: "60s / 70s",
				fsCustomColor: float64(2),
				fsLEDColor:    float64(786434),
				fsEnabled:     false,
				fsMomentary:   false,
				fsPrimary:     true,
			},
		},
		{
			name: "the amplifier of the same pair",
			at:   "block1",
			want: map[string]any{
				fsIndex:       float64(2),
				fsLabel:       "Ampeg B-15NF",
				fsCustomLabel: "60s / 70s",
				fsCustomColor: float64(2),
				fsLEDColor:    float64(16711683),
				fsEnabled:     true,
				fsMomentary:   false,
				fsPrimary:     true,
			},
		},
		{
			name: "a compressor on its own switch",
			at:   "block2",
			want: map[string]any{
				fsIndex:       float64(3),
				fsLabel:       "Tube Comp",
				fsCustomLabel: "Comp",
				fsCustomColor: float64(5),
				fsLEDColor:    float64(8716032),
				fsEnabled:     true,
				fsMomentary:   false,
				fsPrimary:     true,
			},
		},
		{
			// The one whose block sits furthest along the grid, at 13, and
			// first in the chain's own numbering by position.
			name: "a drive switched off",
			at:   "block3",
			want: map[string]any{
				fsIndex:       float64(1),
				fsLabel:       "Teemah!",
				fsCustomLabel: "Drive",
				fsCustomColor: float64(4),
				fsLEDColor:    float64(525824),
				fsEnabled:     false,
				fsMomentary:   false,
				fsPrimary:     true,
			},
		},
	}

	s.Require().Len(got, len(tests), "every assignment the device sent")

	for _, tt := range tests {
		s.Run(tt.name, func() {
			entry, held := got[tt.at]
			s.Require().True(held, "%s carries no assignment", tt.at)
			s.Require().Equal(tt.want, entry)
		})
	}
}

// TestAPresetWithNoSwitchesCarriesNoSection covers the three ways there is
// nothing to write.
//
// Absent rather than empty, because HX Edit writes no section at all for a
// preset nobody assigned anything on, and an empty one would be this tool's
// file differing from the device's for a reason nobody asked for.
//
// One method and one table, so a case is a row rather than a file.
func (s *FootswitchesIntoTestSuite) TestAPresetWithNoSwitchesCarriesNoSection() {
	tests := []struct {
		name string
		got  wire.DevicePreset
	}{
		{
			// Most presets. Three of this project's own eleven genre rigs
			// carried switches before anything assigned them.
			name: "a preset the device sent no assignments for",
			got:  wire.DevicePreset{},
		},
		{
			// An assignment naming a block the chain does not hold, which is
			// a device describing a layout it no longer has. The import side
			// drops one of these too, because that is where it would reach a
			// pedal.
			name: "an assignment pointing at a block nothing placed",
			got: wire.DevicePreset{
				Footswitches: []wire.DeviceFootswitch{
					{Switch: 1, Block: 99, Gear: "Teemah!", Label: "Drive"},
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := preset.Blank()
			s.Require().NoError(err)

			delete(doc.Data.Tone, footswitchKey)

			footswitchesInto(doc, tt.got)

			s.Require().NotContains(doc.Data.Tone, footswitchKey)
		})
	}
}

// TestTheRoundTripKeepsEverySwitch is the test task 262 named: this tool's
// export of a slot, written back, read again.
//
// Which is the failure it was filed for. Export a preset and re-import it and
// every switch was gone, so a round trip through this tool discarded
// assignments the device had.
func (s *FootswitchesIntoTestSuite) TestTheRoundTripKeepsEverySwitch() {
	was := s.switched()

	// Out, through the document the export writes.
	doc, _, err := Document(was, s.catalog(), "B15 Eras")
	s.Require().NoError(err)

	blocks, err := Placements(doc, s.catalog())
	s.Require().NoError(err)

	// And back, the way an import does it.
	out, err := wire.Blank()
	s.Require().NoError(err)
	s.Require().NoError(wire.PlaceAsWritten(out, blocks))

	wire.PlaceFootswitches(out, s.placed(doc, blocks))

	again, err := wire.DecodePreset(out.Encode())
	s.Require().NoError(err)

	s.Require().Len(again.Footswitches, len(was.Footswitches),
		"a round trip dropped an assignment")

	// As a set rather than in order, and without the block number.
	//
	// Two of these four assignments share a gear name, being the amplifier
	// and the cabinet of one Ampeg B-15NF on one switch, so nothing keyed on
	// the name alone can tell them apart. And the grid position is the
	// device's own layout rather than anything somebody chose: what has to
	// survive is every decision, not where the chain happened to land.
	s.Require().ElementsMatch(s.decisions(was), s.decisions(again),
		"a round trip changed what a switch does")
}

// decisions is what each assignment says, without where its block sits.
//
// The switch it is on, the words the pedal prints, the colour somebody chose,
// and the block's own colour and state. Those last three are what used to be
// flattened to zero and false on the way out, which is what task 262 cost.
func (s *FootswitchesIntoTestSuite) decisions(
	got wire.DevicePreset,
) []string {
	out := make([]string, 0, len(got.Footswitches))

	for _, fs := range got.Footswitches {
		out = append(out, fmt.Sprintf(
			"FS%d %q on %q, chose %d, auto %d, enabled %t, momentary %t",
			fs.Switch, fs.Label, fs.Gear, fs.LED, fs.Auto,
			fs.Enabled, fs.Momentary))
	}

	return out
}

// placed turns the document's section into what a device stores, the way an
// import does.
//
// Here rather than reaching into deviceslots, because that package imports
// this one and a test that reversed it would be a cycle. The arithmetic is the
// part worth repeating: a document counts blocks along the chain and the wire
// counts them across the grid.
func (s *FootswitchesIntoTestSuite) placed(
	doc *preset.Document,
	blocks []wire.Placement,
) []wire.DeviceFootswitch {
	held := s.section(doc)
	out := make([]wire.DeviceFootswitch, 0, len(held))

	for at, entry := range held {
		slot := 0
		_, err := fmt.Sscanf(at, blockPrefix+"%d", &slot)
		s.Require().NoError(err)
		s.Require().Less(slot, len(blocks))

		label, _ := entry[fsCustomLabel].(string)
		if label == "" {
			label, _ = entry[fsLabel].(string)
		}

		gear, _ := entry[fsLabel].(string)
		index, _ := entry[fsIndex].(float64)
		custom, _ := entry[fsCustomColor].(float64)
		auto, _ := entry[fsLEDColor].(float64)
		enabled, _ := entry[fsEnabled].(bool)
		momentary, _ := entry[fsMomentary].(bool)

		out = append(out, wire.DeviceFootswitch{
			Switch:    int(index),
			Label:     label,
			Gear:      gear,
			Block:     blocks[slot].Position + wire.GridOffset,
			LED:       int(custom),
			Auto:      int(auto),
			Enabled:   enabled,
			Momentary: momentary,
		})
	}

	return out
}

func TestFootswitchesIntoTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(FootswitchesIntoTestSuite))
}
