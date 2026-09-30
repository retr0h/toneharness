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
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// DevicePublicTestSuite covers what a hand-edited plan can put in the section
// a device wrote.
//
// Everything under `device` is carried verbatim, which means a person can
// type anything into it. Building a preset out of nonsense must leave the
// preset alone rather than fail or write nonsense through.
type DevicePublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *DevicePublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// made returns a buildable plan carrying the given device state.
func (s *DevicePublicTestSuite) made(
	state *rig.DeviceState,
) plan.Plan {
	spec := rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "test",
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
		},
	}

	out := realised(&s.Suite, spec, s.cat)
	out.Device = state

	return out
}

// TestLowerDeviceState builds a preset out of what a plan carries under
// `device`.
func (s *DevicePublicTestSuite) TestLowerDeviceState() {
	tests := []struct {
		name string
		// one entry of the tone section, or one of the routing.
		tone    map[string]json.RawMessage
		routing map[string]json.RawMessage
		version json.RawMessage

		contains string
		absent   string
	}{
		{
			// A person edited the file and put a string where a device wrote
			// a map. Dropping that one entry beats refusing to build the rest
			// of the plan.
			name: "a tone entry that is not an object",
			tone: map[string]json.RawMessage{
				"controller": json.RawMessage(`"nonsense"`),
			},
			absent: "nonsense",
		},
		{
			name: "a tone entry that will not read",
			tone: map[string]json.RawMessage{
				"controller": json.RawMessage(`[1, 2]`),
			},
			absent: `"controller": [`,
		},
		{
			// Routing is keyed by processor and entry — "dsp0.inputA". A key
			// with no processor names nowhere to put it.
			name: "routing naming no processor",
			routing: map[string]json.RawMessage{
				"inputA": json.RawMessage(`{"@model":"X"}`),
			},
			absent: `"@model":"X"`,
		},
		{
			name: "routing reaching a processor the preset lacks",
			routing: map[string]json.RawMessage{
				"dsp7.inputA": json.RawMessage(`{"@model":"HD2_AppDSPFlow1Input"}`),
			},
			contains: "dsp7",
		},
		{
			// A device writes its version as a number or a string. Anything
			// else leaves the blank's own rather than refusing the plan.
			name:    "a device version that is neither a number nor a string",
			version: json.RawMessage(`{"nonsense":1}`),
			absent:  "nonsense",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := preset.Blank()
			s.Require().NoError(err)

			state := &rig.DeviceState{}

			if tt.tone != nil {
				state.Tone = &tt.tone
			}

			if tt.routing != nil {
				state.Routing = &tt.routing
			}

			if tt.version != nil {
				state.Version = &tt.version
			}

			s.Require().NoError(compile.Lower(doc, s.made(state), s.cat))

			var out bytes.Buffer
			s.Require().NoError(preset.Write(&out, doc))

			if tt.contains != "" {
				s.Require().Contains(out.String(), tt.contains)
			}

			if tt.absent != "" {
				s.Require().NotContains(out.String(), tt.absent)
			}
		})
	}
}

// TestAnOutputDestinationReachesTheBuiltPreset is the check that a map edit
// became a file.
//
// Measuring goes through `quieter`, which sets `@output` on the plan's routing
// so the chain is sent to USB rather than to the socket the measuring lead comes
// from. Every layer of that is separately tested and the question this answers
// is whether they join up: an edit that stops anywhere short of the preset
// leaves a chain feeding itself while reporting that it does not.
//
// It is the defect shape this project has been bitten by twice. A chain whose
// blocks were stored and read back byte for byte rendered as nothing, and a
// loop that strode the frame buffer by two measured a figure rather than
// failing. Neither failed; both measured.
func (s *DevicePublicTestSuite) TestAnOutputDestinationReachesTheBuiltPreset() {
	const usbAlone = 10

	to, ok := s.cat.DestinationAt("USB 1/2")
	s.Require().True(ok, "the device lists somewhere off the loop")
	s.Require().Equal(usbAlone, to, "on an HX Stomp, which is what this asserts")

	doc, err := preset.Blank()
	s.Require().NoError(err)

	routing := map[string]json.RawMessage{
		"dsp0.outputA": json.RawMessage(
			`{"@model":"HelixStomp_AppDSPFlowOutputMain",` +
				fmt.Sprintf(`"@output":%d,"pan":0.5,"gain":-30}`, to)),
	}

	s.Require().NoError(compile.Lower(doc,
		s.made(&rig.DeviceState{Routing: &routing}), s.cat))

	// Through the file format and back, because the question is what a preset
	// on disk says, not what a struct in hand says.
	var written bytes.Buffer
	s.Require().NoError(preset.Write(&written, doc))

	back, err := preset.Read(bytes.NewReader(written.Bytes()))
	s.Require().NoError(err)

	entry, ok := back.Data.Tone["dsp0"]["outputA"]
	s.Require().True(ok, "the preset carries an output to send")

	var fields map[string]any
	s.Require().NoError(json.Unmarshal(entry, &fields))

	s.Require().InDelta(float64(to), fields["@output"], 0.001,
		"the built preset sends the chain off the measuring loop")
	s.Require().InDelta(-30, fields["gain"], 0.001,
		"and carries the headroom, which is the half that was already working")
}

func TestDevicePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DevicePublicTestSuite))
}
