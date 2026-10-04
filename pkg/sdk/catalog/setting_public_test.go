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

// SettingPublicTestSuite covers a control value as a document writes it down.
type SettingPublicTestSuite struct {
	suite.Suite
}

// TestSetting covers Setting, which is a parameter value written as the string
// its own kind spells.
//
// One method and one table, so a case is a row rather than a file.
func (s *SettingPublicTestSuite) TestSetting() {
	for _, tt := range []struct {
		name string
		// of is the value to write down, and want the string a document holds.
		of   catalog.ParamValue
		want string
		// kind is what reading it back has to produce, so a float that happens
		// to be whole does not come back an integer.
		kind catalog.ParamType
	}{
		{
			// The case the type exists for. A document is read through a route
			// that turns 6.0 into float64(6) and writes it back as `6`, and a
			// device reads 6 and 6.0 as different settings.
			name: "a whole float keeps its point",
			of:   catalog.Float(6),
			want: `"6.0"`,
			kind: catalog.ParamFloat,
		},
		{
			name: "a fractional float",
			of:   catalog.Float(0.049),
			want: `"0.049"`,
			kind: catalog.ParamFloat,
		},
		{
			name: "a negative float",
			of:   catalog.Float(-37.1),
			want: `"-37.1"`,
			kind: catalog.ParamFloat,
		},
		{
			name: "an integer stays an integer",
			of:   catalog.Int(3),
			want: `"3"`,
			kind: catalog.ParamInt,
		},
		{
			name: "a switch",
			of:   catalog.Bool(true),
			want: `"true"`,
			kind: catalog.ParamBool,
		},
		{
			// A named position has to survive without being read as a number,
			// which is what the quotes going back on is for.
			name: "a named position",
			of:   catalog.Enum("Normal"),
			want: `"Normal"`,
			kind: catalog.ParamEnum,
		},
	} {
		s.Run(tt.name, func() {
			raw, err := json.Marshal(catalog.Set(tt.of))
			s.Require().NoError(err)
			s.Require().JSONEq(tt.want, string(raw))

			var back catalog.Setting

			s.Require().NoError(json.Unmarshal(raw, &back))
			s.Require().Equal(tt.kind, back.Type(), "the kind has to survive")
			s.Require().Equal(tt.of, back.ParamValue)

			// Through YAML the way a document is read, which is the route that
			// loses a kind when the value is not a string.
			out, err := yaml.Marshal(map[string]catalog.Setting{"k": catalog.Set(tt.of)})
			s.Require().NoError(err)

			var held map[string]catalog.Setting

			s.Require().NoError(yaml.Unmarshal(out, &held))
			s.Require().Equal(tt.of, held["k"].ParamValue,
				"a document has to read back what it wrote")
		})
	}
}

// TestSettingRefuses covers what cannot be read as a control value.
func (s *SettingPublicTestSuite) TestSettingRefuses() {
	for _, tt := range []struct {
		name string
		raw  string
	}{
		{
			// A value with no kind is one nothing set, and writing it would put
			// a parameter into a preset the device has no reading for.
			name: "a zero value has no kind to write",
			raw:  "",
		},
		{name: "an object is not a value", raw: `{"a":1}`},
		{name: "a list is not a value", raw: `[1]`},
	} {
		s.Run(tt.name, func() {
			if tt.raw == "" {
				_, err := json.Marshal(catalog.Setting{})
				s.Require().Error(err)

				return
			}

			var held catalog.Setting

			s.Require().Error(json.Unmarshal([]byte(tt.raw), &held))
		})
	}
}

func TestSettingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SettingPublicTestSuite))
}
