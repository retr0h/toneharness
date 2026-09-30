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

// RoutingStatesTestSuite covers reading a preset's routing for a device.
//
// A device sends fewer values than a model has names for, so what matters
// here is that each list comes back the length the device writes and in the
// order the catalog gives, rather than the length the catalog implies.
type RoutingStatesTestSuite struct {
	suite.Suite

	cat  *catalog.Catalog
	held []wire.DeviceRouting
}

func (s *RoutingStatesTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)

	s.held = wire.BlankRouting()
	s.Require().NotEmpty(s.held)
}

// preset0 is a preset HX Edit wrote, with routing somebody set.
func (s *RoutingStatesTestSuite) preset0() *preset.Document {
	f, err := os.Open(
		filepath.Join("..", "compile", "testdata", "preset0.hlx"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := preset.Read(f)
	s.Require().NoError(err)

	return doc
}

// slot finds one entry in what was read.
func (s *RoutingStatesTestSuite) slot(
	got []wire.Routing,
	name string,
) wire.Routing {
	for _, r := range got {
		if r.Slot == name {
			return r
		}
	}

	s.Require().FailNow("no routing slot " + name)

	return wire.Routing{}
}

// entry replaces one routing slot with a body of its own, which is what every
// row that is about a malformed preset does.
func (s *RoutingStatesTestSuite) entry(
	slot, body string,
) *preset.Document {
	doc := s.preset0()
	doc.Data.Tone[processorKey][slot] = json.RawMessage(body)

	return doc
}

// TestRoutingStatesReadsWhatTheDeviceWrote covers every entry a preset can
// carry, sound or otherwise.
//
// A device sends fewer values than a model has names for, and the two counts
// come from different places: the count from the device's own entry, the names
// from the model the file states. A file this tool wrote keeps them in step; a
// file somebody edited can name anything the catalog carries, and each row is
// one way those two can disagree.
func (s *RoutingStatesTestSuite) TestRoutingStatesReadsWhatTheDeviceWrote() {
	const output = "HelixStomp_AppDSPFlowOutputMain"

	for _, tt := range []struct {
		name string
		doc  func() *preset.Document
		then func([]wire.Routing)
	}{
		{
			name: "an output carries what the file says",
			doc:  s.preset0,
			then: func(got []wire.Routing) {
				out := s.slot(got, "outputA")

				// Two, not the three the catalog names: a device stores
				// `select` under its own key rather than among the values.
				s.Require().NotNil(out.Values)
				s.Require().Len(*out.Values, 2)
				s.Require().Equal(2, out.Named)

				s.Require().InDelta(0.5, (*out.Values)[0], 0.0001, "pan")
				s.Require().InDelta(-2.9, (*out.Values)[1], 0.0001, "gain")

				s.Require().NotNil(out.Select)
				s.Require().Equal(1, *out.Select)
			},
		},
		{
			// The widest gap: the model names seven parameters and a device
			// sends three.
			name: "an input is capped at what a device sends",
			doc:  s.preset0,
			then: func(got []wire.Routing) {
				in := s.slot(got, "inputA")

				s.Require().NotNil(in.Values)
				s.Require().Len(*in.Values, 3, "noiseGate, threshold and decay")

				s.Require().Equal(false, (*in.Values)[0], "noiseGate")
				s.Require().InDelta(-48, (*in.Values)[1], 0.0001, "threshold")
				s.Require().InDelta(0.5, (*in.Values)[2], 0.0001, "decay")
			},
		},
		{
			name: "a split carries its model and its place",
			doc:  s.preset0,
			then: func(got []wire.Routing) {
				split := s.slot(got, "split")

				s.Require().NotNil(split.Model)
				s.Require().NotNil(split.Enabled)
				s.Require().True(*split.Enabled)
				s.Require().NotNil(split.Position)
				s.Require().Equal(0, *split.Position)

				s.Require().NotNil(split.Values)
				s.Require().Len(*split.Values, 3,
					"BalanceA, BalanceB and bypass")
			},
		},
		{
			name: "an input carries no model or place",
			doc:  s.preset0,
			then: func(got []wire.Routing) {
				in := s.slot(got, "inputA")

				s.Require().Nil(in.Model,
					"a device knows which input is its own")
				s.Require().Nil(in.Enabled)
				s.Require().Nil(in.Position)
			},
		},
		{
			name: "a preset with no chain carries no routing either",
			doc: func() *preset.Document {
				doc, err := preset.Blank()
				s.Require().NoError(err)

				delete(doc.Data.Tone, processorKey)

				return doc
			},
			then: func(got []wire.Routing) { s.Require().Nil(got) },
		},
		{
			name: "a preset naming none of them has nothing to write",
			doc: func() *preset.Document {
				doc := s.preset0()

				for _, r := range s.held {
					delete(doc.Data.Tone[processorKey], r.Slot)
				}

				return doc
			},
			then: func(got []wire.Routing) { s.Require().Nil(got) },
		},
		{
			name: "an entry that will not read is skipped rather than guessed",
			doc:  func() *preset.Document { return s.entry("outputA", `nonsense`) },
			then: func(got []wire.Routing) {
				for _, r := range got {
					s.Require().NotEqual("outputA", r.Slot)
				}
			},
		},
		{
			// The model is the only thing that says which parameters an entry
			// has, and a list built without one would put values in the wrong
			// places.
			name: "an entry naming no model carries no values",
			doc: func() *preset.Document {
				return s.entry("outputA", `{"@output": 1}`)
			},
			then: func(got []wire.Routing) {
				out := s.slot(got, "outputA")

				s.Require().Nil(out.Values)
				s.Require().NotNil(out.Select, "what it could read, it read")
			},
		},
		{
			name: "an entry naming a model nobody carries also carries none",
			doc: func() *preset.Document {
				return s.entry("outputA",
					`{"@output": 1, "@model": "HD2_NoSuchFlow"}`)
			},
			then: func(got []wire.Routing) {
				s.Require().Nil(s.slot(got, "outputA").Values)
			},
		},
		{
			name: "a model name that will not read is the same as naming none",
			doc: func() *preset.Document {
				return s.entry("outputA", `{"@model": 7}`)
			},
			then: func(got []wire.Routing) {
				s.Require().Nil(s.slot(got, "outputA").Values)
			},
		},
		{
			// A return block naming two parameters in a slot the device sent
			// three values for used to slice past the end of the list.
			name: "a model carrying fewer parameters than the device sent",
			doc: func() *preset.Document {
				return s.entry("inputA",
					`{"@model": "HD2_ReturnMono1", "Return": 0.5}`)
			},
			then: func(got []wire.Routing) {
				in := s.slot(got, "inputA")

				s.Require().NotNil(in.Values)
				s.Require().Len(*in.Values, 2,
					"what the model names, not what the device sent")
				s.Require().Equal(2, in.Named,
					"and the count the device is told matches")
			},
		},
		{
			// Position is the only thing naming a value on the wire, so a
			// missing one cannot be left out without moving every value after
			// it.
			name: "a parameter the preset omits still takes its place",
			doc: func() *preset.Document {
				return s.entry("outputA",
					`{"@output": 1, "@model": "`+output+`", "gain": -2.9}`)
			},
			then: func(got []wire.Routing) {
				out := s.slot(got, "outputA")

				s.Require().NotNil(out.Values)
				s.Require().Len(*out.Values, 2)
				s.Require().InDelta(0, (*out.Values)[0], 0.0001,
					"the pan it does not name")
				s.Require().InDelta(-2.9, (*out.Values)[1], 0.0001)
			},
		},
		{
			name: "a parameter that will not read leaves that one at nothing",
			doc: func() *preset.Document {
				return s.entry("outputA",
					`{"@model": "`+output+`", "pan": "loud", "gain": -1}`)
			},
			then: func(got []wire.Routing) {
				out := s.slot(got, "outputA")

				s.Require().NotNil(out.Values)
				s.Require().InDelta(0, (*out.Values)[0], 0.0001)
				s.Require().InDelta(-1, (*out.Values)[1], 0.0001)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then(RoutingStates(tt.doc(), s.cat, s.held))
		})
	}
}

func TestRoutingStatesTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RoutingStatesTestSuite))
}
