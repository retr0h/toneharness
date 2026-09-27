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
package tone_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// EmbedPublicTestSuite covers the contract a request is checked against.
//
// The Go types are generated from the same document, so one this binary could
// not read would have failed generation first. What is worth holding is that
// the document and the types have not drifted apart in the ways generation
// does not catch: a value added to an enum in the contract and nowhere else
// compiles, ships, and is refused at runtime by a check nobody wrote.
type EmbedPublicTestSuite struct {
	suite.Suite

	doc *openapi3.T
}

func (s *EmbedPublicTestSuite) SetupSuite() {
	doc, err := openapi3.NewLoader().LoadFromData(tone.Schema)
	s.Require().NoError(err)

	s.doc = doc
}

// TestTheContractDescribesBothDocuments covers the two roots existing.
func (s *EmbedPublicTestSuite) TestTheContractDescribesBothDocuments() {
	for _, name := range []string{"ToneSpec", "Setup"} {
		s.Run(name, func() {
			ref, ok := s.doc.Components.Schemas[name]
			s.Require().True(ok, "the contract describes no %s", name)
			s.Require().NotNil(ref.Value)

			// Both say what they are, so a file on disk is identified by its
			// contents rather than by its extension.
			schema, ok := ref.Value.Properties["schema"]
			s.Require().True(ok, "%s does not say what it is", name)
			s.Require().Equal([]any{name}, schema.Value.Enum)
		})
	}
}

// TestAskAndSetupStayApart is the split, held to.
//
// The two documents change on different clocks: what somebody wants changes
// every request and what they own changes when they buy something. Folding an
// instrument into the ask would mean restating it every time, and the twelfth
// request contradicting the first.
func (s *EmbedPublicTestSuite) TestAskAndSetupStayApart() {
	ask := s.doc.Components.Schemas["ToneSpec"].Value.Properties
	setup := s.doc.Components.Schemas["Setup"].Value.Properties

	for _, field := range []string{"instruments", "device", "owns"} {
		s.Require().NotContains(ask, field,
			"%s belongs to the person, not to the request", field)
	}

	for _, field := range []string{"genre", "words", "nudges", "like"} {
		s.Require().NotContains(setup, field,
			"%s belongs to the request, not to the person", field)
	}
}

// TestNoKnobPositions is the reason the split exists at all.
//
// A hand-written document may say what somebody wants and what they own. It
// may not say where a knob goes: that is what the tool works out, and a number
// typed into an authored file is how `dark` came to move Treble by a quarter
// of its range because somebody decided a quarter.
func (s *EmbedPublicTestSuite) TestNoKnobPositions() {
	for name, ref := range s.doc.Components.Schemas {
		for field := range ref.Value.Properties {
			s.Require().NotContains(
				[]string{"settings", "params", "knobs", "values"}, field,
				"%s.%s would let somebody write a knob position by hand", name, field)
		}
	}
}

// TestEnumsMatchTheGoConstants covers the drift generation does not catch.
func (s *EmbedPublicTestSuite) TestEnumsMatchTheGoConstants() {
	tests := []struct {
		schema string
		have   []string
	}{
		{
			schema: "Strings",
			have: []string{
				string(tone.StringsRound), string(tone.StringsFlat),
				string(tone.StringsTape), string(tone.StringsUnknown),
			},
		},
		{
			schema: "Role",
			have: []string{
				"amp", "cab", "drive", "comp", "eq", "mod",
				"delay", "reverb", "filter", "pitch", "wah", "other",
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.schema, func() {
			ref, ok := s.doc.Components.Schemas[tt.schema]
			s.Require().True(ok)

			want := make([]string, 0, len(ref.Value.Enum))
			for _, v := range ref.Value.Enum {
				want = append(want, v.(string))
			}

			s.Require().ElementsMatch(want, tt.have)
		})
	}
}

// TestGenreIsNotAnEnum is a decision worth a test.
//
// Enumerating genres in the contract would mean a release to add one, and the
// schema is not what decides whether a genre works: eight records from three
// artists is, because three records by one band is that band's sound wearing a
// genre's name.
func (s *EmbedPublicTestSuite) TestGenreIsNotAnEnum() {
	genre := s.doc.Components.Schemas["ToneSpec"].Value.Properties["genre"]

	s.Require().NotNil(genre)
	s.Require().Empty(genre.Value.Enum,
		"a genre is a word, and which ones work is a question about records")
}

func TestEmbedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EmbedPublicTestSuite))
}
