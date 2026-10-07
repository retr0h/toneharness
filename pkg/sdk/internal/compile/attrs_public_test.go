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
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// AttrsPublicTestSuite covers what a chain entry says about a block beside its
// gear and its controls.
type AttrsPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *AttrsPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// lift reads a chain entry out of a block written by hand.
func (s *AttrsPublicTestSuite) lift(
	attrs string,
) rig.ChainEntry {
	raw := `{"schema":"L6Preset","version":6,"data":{"device":2162694,` +
		`"meta":{"name":"Attrs"},"tone":{"dsp0":{"block0":{` +
		`"@model":"HD2_AmpSVBeastBrt","@enabled":true,"@position":0` + attrs +
		`}}}}}`

	doc, err := preset.Read(bytes.NewReader([]byte(raw)))
	s.Require().NoError(err)

	_, spec, _, err := compile.Lift(doc, s.cat)
	s.Require().NoError(err)
	s.Require().Len(spec.Chain, 1)

	return spec.Chain[0]
}

// TestAttrsOnto covers reading a block's device attributes onto the entry that
// describes it.
//
// Nine of these fields were in the contract, written by the worked example, and
// read by nothing. A preset exported and built again came back with its parallel
// path gone, its stereo blocks mono, its reverb tails cut and three bypassed
// blocks switched on.
//
// One method and one table, so a case is a row rather than a file.
func (s *AttrsPublicTestSuite) TestAttrsOnto() {
	for _, tt := range []struct {
		name string
		// attrs is what the block carries beside its model, as JSON to append.
		attrs string
		// then is what the entry has to say afterwards.
		then func(got rig.ChainEntry)
	}{
		{
			// The parallel path, which decides where in the signal a block sits
			// and was dropped on every block of every preset read off a device.
			name:  "the parallel path",
			attrs: `,"@path":1`,
			then: func(got rig.ChainEntry) {
				s.Require().NotNil(got.Path)
				s.Require().Equal(1, *got.Path)
			},
		},
		{
			name:  "stereo, trails and the bypass level",
			attrs: `,"@stereo":true,"@trails":false,"@bypassvolume":0.5`,
			then: func(got rig.ChainEntry) {
				s.Require().NotNil(got.Stereo)
				s.Require().True(*got.Stereo)
				s.Require().NotNil(got.Trails)
				s.Require().False(*got.Trails)
				s.Require().NotNil(got.BypassVolume)
				s.Require().InDelta(0.5, *got.BypassVolume, 1e-9)
			},
		},
		{
			name:  "what a snapshot leaves alone",
			attrs: `,"@no_snapshot_bypass":true`,
			then: func(got rig.ChainEntry) {
				s.Require().NotNil(got.KeepOnSnapshot)
				s.Require().True(*got.KeepOnSnapshot)
			},
		},
		{
			// A microphone index, the sibling cabinet a dual block points at, and
			// the impulse response it plays.
			name:  "the microphone, the paired cabinet and the impulse response",
			attrs: `,"@mic":5,"@cab":"cab0","@uuid":"02cd584ccf3aecf2d04f9939d04783d0"`,
			then: func(got rig.ChainEntry) {
				s.Require().NotNil(got.Mic)
				s.Require().Equal(5, *got.Mic)
				s.Require().NotNil(got.Cab)
				s.Require().Equal("cab0", *got.Cab)
				s.Require().NotNil(got.Ir)
				s.Require().Equal("02cd584ccf3aecf2d04f9939d04783d0", *got.Ir)
			},
		},
		{
			// An attribute with no field of its own. `@type` is on 99.8% of the
			// 38,473 chain blocks in the corpus and means nothing anybody here has
			// established, so it travels as the device wrote it.
			name:  "an attribute no field claims",
			attrs: `,"@type":3,"@favorite":1`,
			then: func(got rig.ChainEntry) {
				s.Require().NotNil(got.Attrs)
				s.Require().Contains(*got.Attrs, "@type")
				s.Require().Contains(*got.Attrs, "@favorite")
			},
		},
		{
			// The position and the enabled flag are not attributes here: the chain
			// carries its own order and the preset writer puts `@enabled` back from
			// the block itself.
			name:  "the position and the switch are not carried as attributes",
			attrs: ``,
			then: func(got rig.ChainEntry) {
				s.Require().Nil(got.Attrs)
			},
		},
		{
			// A document this tool did not write. Each field stays unsaid rather
			// than taking a zero, because a `@path` that is not a number read as
			// zero would move the block onto the first path.
			// `@path` and `@stereo` are not among them: the preset reader models
			// those two as typed fields and refuses the document outright, which
			// is a better error and means these branches only guard the rest.
			name: "an attribute of the wrong kind says nothing",
			attrs: `,"@bypassvolume":"loud","@cab":7,"@mic":"five",` +
				`"@trails":3,"@no_snapshot_bypass":"yes","@uuid":9`,
			then: func(got rig.ChainEntry) {
				s.Require().Nil(got.BypassVolume)
				s.Require().Nil(got.Cab)
				s.Require().Nil(got.Mic)
				s.Require().Nil(got.Trails)
				s.Require().Nil(got.KeepOnSnapshot)
				s.Require().Nil(got.Ir)
			},
		},
		{
			// A null attribute, which the device writes and takes back.
			name:  "a null attribute no field claims",
			attrs: `,"@favorite":null`,
			then: func(got rig.ChainEntry) {
				s.Require().NotNil(got.Attrs)
				s.Require().Contains(*got.Attrs, "@favorite")
				s.Require().Nil((*got.Attrs)["@favorite"])
			},
		},
		{
			// An attribute that is neither a value nor a null, which no preset
			// holds and a document somebody edited could.
			name:  "an attribute that is an object is left out",
			attrs: `,"@favorite":{"a":1}`,
			then: func(got rig.ChainEntry) {
				s.Require().Nil(got.Attrs)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then(s.lift(tt.attrs))
		})
	}
}

// TestAttrsFrom covers writing them back, which is what makes an edit reach the
// pedal rather than sit in the file.
func (s *AttrsPublicTestSuite) TestAttrsFrom() {
	for _, tt := range []struct {
		name string
		// attrs is what the block carried, round-tripped through a document and
		// back into a preset.
		attrs string
		// want is what the rebuilt block has to hold.
		want map[string]string
	}{
		{
			name:  "the parallel path comes back",
			attrs: `,"@path":1`,
			want:  map[string]string{"@path": "1"},
		},
		{
			name:  "stereo and trails come back",
			attrs: `,"@stereo":true,"@trails":false`,
			want:  map[string]string{"@stereo": "true", "@trails": "false"},
		},
		{
			name:  "an attribute no field claims comes back",
			attrs: `,"@type":3`,
			want:  map[string]string{"@type": "3"},
		},
		{
			name:  "a null attribute comes back as a null",
			attrs: `,"@favorite":null`,
			want:  map[string]string{"@favorite": "null"},
		},
		{
			// Bypassed, which was hardcoded to on: a preset exported with three
			// blocks bypassed came back with all three playing.
			name:  "a bypassed block stays bypassed",
			attrs: ``,
			want:  map[string]string{"@enabled": "false"},
		},
	} {
		s.Run(tt.name, func() {
			entry := s.lift(tt.attrs)

			if tt.name == "a bypassed block stays bypassed" {
				off := false
				entry.Enabled = &off
			}

			made, err := compile.Realise("x", rig.Spec{
				Instrument: rig.InstrumentBass,
				Chain:      []rig.ChainEntry{entry},
			}, s.cat)
			s.Require().NoError(err)

			doc, _ := preset.Blank()
			doc.Data.Device = s.cat.DeviceID
			s.Require().NoError(doc.SetSpec(made))

			var held map[string]json.RawMessage

			s.Require().NoError(json.Unmarshal(doc.Data.Tone["dsp0"]["block0"], &held))

			for name, want := range tt.want {
				s.Require().Contains(held, name)
				s.Require().JSONEq(want, string(held[name]))
			}
		})
	}
}

func TestAttrsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AttrsPublicTestSuite))
}
