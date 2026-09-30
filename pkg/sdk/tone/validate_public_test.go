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
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// ValidatePublicTestSuite covers reading the contract and the failures that
// only happen if it cannot be read.
//
// Nothing here is reachable in a shipped binary: the document is embedded and
// generation would have failed on one this could not parse. It is here so
// that if the contract ever could not be read, a document would be reported
// as unchecked rather than passed as valid.
type ValidatePublicTestSuite struct {
	suite.Suite
}

// TestError covers every case Error answers.
//
// One method and one table, so a case is a row rather than a file.
func (s *ValidatePublicTestSuite) TestError() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "load schema",
			then: func() {
				tests := []struct {
					name    string
					doc     []byte
					errText string
				}{
					{name: "the contract this binary ships", doc: tone.Schema},
					{
						name:    "a document it cannot read",
						doc:     []byte("not a schema"),
						errText: "ToneSpec schema",
					},
					{
						name: "a document describing only one of the two",
						doc: []byte(`
openapi: 3.0.3
info: { title: Half A Contract, version: "1.0.0" }
paths: {}
components:
  schemas:
    ToneSpec: { type: object }
`),
						errText: "describes no Setup",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, err := tone.LoadSchema(tt.doc)

						if tt.errText != "" {
							s.Require().Error(err)
							s.Require().Contains(err.Error(), tt.errText)

							return
						}

						s.Require().NoError(err)
						s.Require().Len(got, 2)
					})
				}
			},
		},
		{
			// check as something other than a set of fields.
			name: "against refuses what the types could not build",
			then: func() {
				err := tone.Against([]any{"a list, not a document"}, "ToneSpec")

				s.Require().ErrorIs(err, tone.ErrInvalid)
				s.Require().NotEmpty(err.Error())
			},
		},
		{
			name: "invalid",
			then: func() {
				tests := []struct {
					name     string
					in       error
					contains string
				}{
					{
						// The library reports a failed field today. If it ever reports
						// something else, that has to reach somebody rather than be
						// swallowed.
						name:     "a failure of some other kind",
						in:       errors.New("something else went wrong"),
						contains: "something else went wrong",
					},
					{
						name: "a failure that names no field",
						in: &openapi3.SchemaError{
							Schema: openapi3.NewStringSchema(),
							Value:  1,
						},
						contains: "the document",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						err := tone.Invalid(tt.in, "ToneSpec")

						s.Require().ErrorIs(err, tone.ErrInvalid)
						s.Require().Contains(err.Error(), tt.contains)
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestAnUnreadableContractRefusesBothDocuments covers the cached failure.
func (s *ValidatePublicTestSuite) TestAnUnreadableContractRefusesBothDocuments() {
	restore := *tone.Contracts
	defer func() { *tone.Contracts = restore }()

	boom := errors.New("no contract")
	*tone.Contracts = func() (map[string]*openapi3.Schema, error) {
		return nil, boom
	}

	s.Require().ErrorIs(tone.Validate(tone.Spec{}), boom)
	s.Require().ErrorIs(tone.ValidateSetup(tone.Setup{}), boom)
}

// TestAFailureInsideAListIsWrittenTheWayTheFileIs covers the field path.
//
// A schema points at a value with a JSON pointer, where every step is a name:
// `genre.1`. Somebody looking at their own YAML sees a list, so a step that is
// a position is written as one. It is the same rule rig applies to a RigSpec,
// deliberately: the two contracts are read by the same person holding both
// files, and a ToneSpec error spelling a position differently would read as a
// different kind of fault rather than the same one in the other document.
func (s *ValidatePublicTestSuite) TestAFailureInsideAListIsWrittenTheWayTheFileIs() {
	err := tone.Validate(tone.Spec{Genre: []string{"punk", ""}})

	s.Require().ErrorIs(err, tone.ErrInvalid)
	s.Require().ErrorContains(err, "genre[1]")
	s.Require().NotContains(err.Error(), "genre.1")

	var fault *tone.InvalidError
	s.Require().ErrorAs(err, &fault)
	s.Require().Equal("genre[1]", fault.Field)
	s.Require().NotEmpty(fault.Reason, "and it says what was wrong with it")
}

func TestValidatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidatePublicTestSuite))
}

// TestCheckReportsWhatCannotBeMarshalled covers the guard before the schema.
func (s *ValidatePublicTestSuite) TestCheckReportsWhatCannotBeMarshalled() {
	err := tone.Check(make(chan int), "ToneSpec")

	s.Require().ErrorContains(err, "reading the ToneSpec")
}
