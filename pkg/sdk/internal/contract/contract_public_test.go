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

package contract_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/contract"
)

// A contract small enough to read, with the three shapes that matter: a
// required field, an enumeration, and a list whose entries are objects.
const schema = `
openapi: 3.0.3
info: { title: Thing, version: "2.1.0" }
paths: {}
components:
  schemas:
    Thing:
      type: object
      additionalProperties: false
      required: [schema, name]
      properties:
        schema: { type: string, enum: [Thing] }
        name: { type: string, minLength: 1 }
        colour: { type: string, enum: [red, blue] }
        parts:
          type: array
          items:
            type: object
            additionalProperties: false
            required: [at]
            properties:
              at: { type: integer }
`

// thing is what the contract above describes.
type thing struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Colour string `json:"colour,omitempty"`
	Parts  []part `json:"parts,omitempty"`
}

type part struct {
	At int `json:"at"`
}

// unreadable is a reader that fails, for the path where a file cannot be read
// at all rather than cannot be understood.
type unreadable struct{}

func (unreadable) Read(
	[]byte,
) (int, error) {
	return 0, errors.New("boom")
}

// ContractPublicTestSuite covers checking a hand-written document against the
// schema its types were generated from.
//
// Shared machinery rather than a shared contract: a ToneSpec and a Config are
// separate documents with separate versions, and what they have in common is
// that both are typed by hand and want the same checking and the same shape of
// error.
type ContractPublicTestSuite struct {
	suite.Suite

	checker *contract.Checker
}

func (s *ContractPublicTestSuite) SetupTest() {
	s.checker = contract.New([]byte(schema), "Thing")
}

// TestRead covers decoding a document and checking it.
//
// One method and one table, so a case is a row rather than a file.
func (s *ContractPublicTestSuite) TestRead() {
	for _, tt := range []struct {
		name string
		body string
		// says is what the refusal must contain. Empty expects it to pass.
		says string
		// field is the path the error names, where it names one.
		field string
		// then checks what was decoded, on the rows that pass.
		then func(got thing)
	}{
		{
			name: "a document that meets its contract",
			body: "schema: Thing\nname: a thing\ncolour: red\n",
			then: func(got thing) {
				s.Require().Equal("a thing", got.Name)
				s.Require().Equal("red", got.Colour)
			},
		},
		{
			name:  "a required field nobody wrote",
			body:  "schema: Thing\n",
			says:  "name",
			field: "name",
		},
		{
			name:  "a word that is not in the enumeration",
			body:  "schema: Thing\nname: a thing\ncolour: green\n",
			says:  "allowed values",
			field: "colour",
		},
		{
			// Checked before anything decodes it, because decoding drops what
			// the generated types have no field for: the typo would be gone
			// on the way in, the document would pass, and the line somebody
			// wrote would be lost with nothing said.
			name: "a field nobody spelt right",
			body: "schema: Thing\nname: a thing\ncolourr: red\n",
			says: "colourr",
		},
		{
			// A schema points at a value with a JSON pointer, where every
			// step is a name. Somebody looking at their own YAML sees a list,
			// so a step that is a position is written as one.
			name:  "a failure inside a list names the position the file does",
			body:  "schema: Thing\nname: a thing\nparts:\n  - at: 1\n  - at: not a number\n",
			says:  "",
			field: "parts[1].at",
		},
		{
			name: "a file that is not YAML at all",
			body: "\tnot: [yaml",
			says: "decoding",
		},
	} {
		s.Run(tt.name, func() {
			var got thing

			err := s.checker.Read(strings.NewReader(tt.body), "Thing", &got)

			if tt.says == "" && tt.field == "" {
				s.Require().NoError(err)

				if tt.then != nil {
					tt.then(got)
				}

				return
			}

			s.Require().Error(err)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)
			}

			if tt.field != "" {
				var fault *contract.InvalidError
				s.Require().ErrorAs(err, &fault)
				s.Require().Equal(tt.field, fault.Field)
				s.Require().Equal("Thing", fault.Document)
				s.Require().ErrorIs(err, contract.ErrInvalid)
			}
		})
	}
}

// TestReadAFileThatWillNotRead covers the reader failing rather than the
// document being wrong.
func (s *ContractPublicTestSuite) TestReadAFileThatWillNotRead() {
	var got thing

	err := s.checker.Read(unreadable{}, "Thing", &got)

	s.Require().ErrorContains(err, "boom")
}

// TestCheck covers a document built in memory rather than read off disk.
//
// Nothing in a generated struct stops a required field being empty or an
// enumeration carrying a word that is not in it, so one assembled in Go has
// never been checked.
func (s *ContractPublicTestSuite) TestCheck() {
	s.Require().NoError(s.checker.Check(
		thing{Schema: "Thing", Name: "a thing"}, "Thing"))

	s.Require().ErrorIs(
		s.checker.Check(thing{}, "Thing"), contract.ErrInvalid)
}

// TestASchemaNobodyNamed covers asking for a schema the contract does not
// describe, which is a caller's mistake rather than a document's.
func (s *ContractPublicTestSuite) TestASchemaNobodyNamed() {
	err := s.checker.Check(thing{Schema: "Thing", Name: "a thing"}, "Other")

	s.Require().ErrorContains(err, "describes no Other")
}

// TestAContractThatWillNotLoad covers the document holding the rules being
// unreadable, which is a build nobody could have shipped working.
func (s *ContractPublicTestSuite) TestAContractThatWillNotLoad() {
	for _, tt := range []struct {
		name string
		raw  string
		says string
	}{
		{
			name: "not a schema at all",
			raw:  "not a schema",
			says: "reading the schema",
		},
		{
			name: "a schema describing nothing by that name",
			raw: `
openapi: 3.0.3
info: { title: Empty, version: "1.0.0" }
paths: {}
components:
  schemas:
    Something: { type: object }
`,
			says: "describes no Thing",
		},
	} {
		s.Run(tt.name, func() {
			c := contract.New([]byte(tt.raw), "Thing")

			err := c.Check(thing{Schema: "Thing", Name: "a thing"}, "Thing")

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

// TestVersion covers the contract stating its own version, which is how a
// document says which rules it was written under.
func (s *ContractPublicTestSuite) TestVersion() {
	got, err := s.checker.Version()

	s.Require().NoError(err)
	s.Require().Equal("2.1.0", got)
}

// TestVersionRefusesAContractThatStatesNone covers the two ways it can fail.
//
// Neither is reachable in a correct build, which is why it is reachable from
// here: a binary carrying a contract with no version has one fact missing from
// every document it writes.
func (s *ContractPublicTestSuite) TestVersionRefusesAContractThatStatesNone() {
	for _, tt := range []struct {
		name string
		raw  string
		says string
	}{
		{
			name: "a contract that is not YAML",
			raw:  "\tnope: [",
			says: "reading the schema's version",
		},
		{
			name: "a contract that states no version",
			raw:  "openapi: 3.0.3\ninfo: { title: Thing }\n",
			says: "states no version",
		},
	} {
		s.Run(tt.name, func() {
			_, err := contract.New([]byte(tt.raw), "Thing").Version()

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

func TestContractPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(ContractPublicTestSuite))
}
