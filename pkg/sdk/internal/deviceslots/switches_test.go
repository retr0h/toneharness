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

package deviceslots

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// SwitchesTestSuite covers turning a document's assignments into a device's.
type SwitchesTestSuite struct {
	suite.Suite
}

// docWith builds a document carrying one footswitch section.
func (s *SwitchesTestSuite) docWith(
	section string,
) *preset.Document {
	doc := &preset.Document{}
	doc.Data.Tone = map[string]preset.Tone{
		footswitchKey: {"dsp0": json.RawMessage(section)},
	}

	return doc
}

// TestSwitchesOf reads the assignments a document carries.
func (s *SwitchesTestSuite) TestSwitchesOf() {
	// A chain of four, laid out the way a device numbers a grid: the first
	// block a document calls `block0` sits at position 0 here and at 1 on the
	// device, which is [wire.GridOffset].
	chain := []wire.Placement{
		{Position: 0}, {Position: 1}, {Position: 2}, {Position: 3},
	}

	tests := []struct {
		name string
		// section is what the document's footswitch member holds.
		section string
		placed  []wire.Placement
		want    []wire.DeviceFootswitch
	}{
		{
			// The bug this guards. A document counts blocks along the chain
			// and the wire counts them across a grid that also holds the
			// input, so a switch written without the offset named the block to
			// its left. On a chain of drive, amp and cabinet the drive asked
			// for position 0, which no block occupies, and the switch lit and
			// did nothing; put a compressor in front and it named the
			// compressor instead.
			name: "a switch names the block by its grid position",
			section: `{"block1":{"@fs_index":1,"@fs_label":"Ampeg Scrambler",
				"@fs_customcolor":2}}`,
			placed: chain,
			want: []wire.DeviceFootswitch{{
				Switch: 1,
				Block:  1 + wire.GridOffset,
				Gear:   "Ampeg Scrambler",
				Label:  "Ampeg Scrambler",
				LED:    2,
			}},
		},
		{
			name: "the first block in the chain is not position zero",
			section: `{"block0":{"@fs_index":1,"@fs_label":"Deluxe Comp",
				"@fs_customcolor":9}}`,
			placed: chain,
			want: []wire.DeviceFootswitch{{
				Switch: 1,
				Block:  wire.GridOffset,
				Gear:   "Deluxe Comp",
				Label:  "Deluxe Comp",
				LED:    9,
			}},
		},
		{
			name:    "a block the chain never placed is skipped",
			section: `{"block9":{"@fs_index":1,"@fs_label":"Teemah!"}}`,
			placed:  chain,
			want:    nil,
		},
		{
			name:    "a member that is not a block is skipped",
			section: `{"split":{"@fs_index":1,"@fs_label":"Teemah!"}}`,
			placed:  chain,
			want:    nil,
		},
		{
			name:    "a section that will not decode is skipped",
			section: `"text"`,
			placed:  chain,
			want:    nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(
				tt.want, switchesOf(s.docWith(tt.section), tt.placed))
		})
	}
}

// TestSwitchesOfWithoutASection reads a document that assigns nothing.
func (s *SwitchesTestSuite) TestSwitchesOfWithoutASection() {
	s.Require().Nil(switchesOf(nil, nil))
	s.Require().Nil(switchesOf(&preset.Document{}, nil))
}

func TestSwitchesTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SwitchesTestSuite))
}
