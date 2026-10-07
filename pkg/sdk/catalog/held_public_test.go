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

package catalog_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// HeldPublicTestSuite covers a value a preset holds, which is a control value
// with the device's own empty string beside it.
type HeldPublicTestSuite struct {
	suite.Suite
}

// TestHeld covers Held, which is one value a preset holds as a document writes
// it down.
//
// One method and one table, so a case is a row rather than a file.
func (s *HeldPublicTestSuite) TestHeld() {
	for _, tt := range []struct {
		name string
		// of is the value to write down, blank asking for the device's own
		// empty string instead.
		of    catalog.ParamValue
		blank bool
		// want is the document's spelling and device the preset's, which differ:
		// a document quotes a reading so its kind survives and a preset does
		// not.
		want   string
		device string
		// kind is what reading it back has to produce, so a float that happens
		// to be whole does not come back an integer.
		kind catalog.ParamType
	}{
		{
			// The case the type exists for, beside Setting. An unassigned
			// impulse response slot is the empty string, and 128 of them are in
			// every preset carrying a table.
			name:   "the device's own empty string",
			blank:  true,
			want:   `""`,
			device: `""`,
		},
		{
			// Setting's own rule, reached through it rather than restated: a
			// document is read through a route that turns 6.0 into float64(6),
			// and a device reads 6 and 6.0 as different settings.
			name:   "a whole float keeps its point",
			of:     catalog.Float(6),
			want:   `"6.0"`,
			device: `6.0`,
			kind:   catalog.ParamFloat,
		},
		{
			name:   "an integer stays one",
			of:     catalog.Int(2),
			want:   `"2"`,
			device: `2`,
			kind:   catalog.ParamInt,
		},
		{
			name:   "a switch",
			of:     catalog.Bool(true),
			want:   `"true"`,
			device: `true`,
			kind:   catalog.ParamBool,
		},
		{
			// A model identifier and a routing topology are both this: a name
			// rather than a reading, which keeps its quotes in a preset.
			name:   "a named position",
			of:     catalog.Enum("SABJ"),
			want:   `"SABJ"`,
			device: `"SABJ"`,
			kind:   catalog.ParamEnum,
		},
		{
			name:   "a negative reading",
			of:     catalog.Float(-48.5),
			want:   `"-48.5"`,
			device: `-48.5`,
			kind:   catalog.ParamFloat,
		},
	} {
		s.Run(tt.name, func() {
			held := catalog.Hold(tt.of)
			if tt.blank {
				held = catalog.Blank()
			}

			out, err := json.Marshal(held)
			s.Require().NoError(err)
			s.Require().Equal(tt.want, string(out))

			forDevice, err := held.Device()
			s.Require().NoError(err)
			s.Require().Equal(tt.device, string(forDevice),
				"a preset keeps a reading bare and a document keeps it quoted")

			// Read back through the same route a document takes, which is where
			// a kind is lost if it was not written as a string.
			var back map[string]catalog.Held

			s.Require().NoError(yaml.Unmarshal([]byte(`k: `+tt.want), &back))
			s.Require().Equal(tt.blank, back["k"].IsBlank())

			got, ok := back["k"].Value()
			s.Require().Equal(!tt.blank, ok)

			if tt.blank {
				return
			}

			s.Require().Equal(tt.of, got, "a document has to read back what it wrote")
			s.Require().Equal(tt.kind, got.Type())
		})
	}
}

// TestHoldRaw covers HoldRaw, which reads one value as a device wrote it.
//
// The other direction from Device, and the one a lift takes: a preset's own bare
// literal rather than a document's quoted spelling.
func (s *HeldPublicTestSuite) TestHoldRaw() {
	for _, tt := range []struct {
		name string
		raw  string
		// want is the document's spelling of what came back, blank asking for
		// the empty string, and err that the literal was refused.
		want  string
		blank bool
		err   bool
	}{
		{
			// The difference from UnmarshalJSON, and the reason both exist: a
			// preset writes a float bare, and read as a document's spelling this
			// would be a named position called "6.0".
			name: "a bare float is a reading rather than a name",
			raw:  `6.0`,
			want: `"6.0"`,
		},
		{name: "a bare integer", raw: `2`, want: `"2"`},
		{name: "a bare switch", raw: `false`, want: `"false"`},
		{name: "a quoted name keeps its quotes", raw: `"SABJ"`, want: `"SABJ"`},
		{
			// What the empty string is for. A device writes one for a slot
			// nothing is assigned to, and Setting refuses it.
			name:  "the device's own empty string",
			raw:   `""`,
			want:  `""`,
			blank: true,
		},
		{name: "an object is not a value", raw: `{"a":1}`, err: true},
		{name: "a list is not a value", raw: `[1]`, err: true},
		{
			// A null is the device having no value, which is not the same as the
			// empty string. Refused rather than read as a blank: a caller that
			// cannot tell them apart writes `""` where the device wrote nothing,
			// and an attribute with no value came back set to the empty string.
			name: "a null is not a value either",
			raw:  `null`,
			err:  true,
		},
	} {
		s.Run(tt.name, func() {
			got, err := catalog.HoldRaw([]byte(tt.raw))

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.blank, got.IsBlank())

			out, err := json.Marshal(got)
			s.Require().NoError(err)
			s.Require().Equal(tt.want, string(out))
		})
	}
}

// TestHeldRefuses covers what cannot be read as a value a preset holds.
func (s *HeldPublicTestSuite) TestHeldRefuses() {
	for _, tt := range []struct {
		name string
		raw  string
	}{
		{name: "an object is not a value", raw: `{"a":1}`},
		{name: "a list is not a value", raw: `[1]`},
		{
			// A document's spelling is a string, and a word that is not a number,
			// a switch or a name is not one of the three.
			name: "a literal of no kind",
			raw:  `{}`,
		},
	} {
		s.Run(tt.name, func() {
			var held catalog.Held

			s.Require().Error(json.Unmarshal([]byte(tt.raw), &held))
		})
	}
}

func TestHeldPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HeldPublicTestSuite))
}
