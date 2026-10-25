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

package contract

import (
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/suite"
)

// FallbackTestSuite covers the two paths that only exist for the cases a
// schema library does not promise.
//
// Reached from inside the package because both take an error this code did not
// make and cannot provoke through a document: one is a failure that is not a
// schema failure, and the other is a schema failure that declines to say why.
// Neither has ever been seen from a real contract, which is exactly why they
// are written and exactly why they need reaching: a branch nothing can enter
// is a branch nobody has read, and these two are what stands between a reader
// and a refusal with no message in it.
type FallbackTestSuite struct {
	suite.Suite
}

// TestAFailureThatIsNotASchemaFailure covers an error arriving from the
// library that is not the kind this knows how to take apart.
//
// A MultiError is one, and so is anything a later version decides to return.
// The document is still named, because somebody holding two files needs to
// know which one was refused even when nothing can say which line.
func (s *FallbackTestSuite) TestAFailureThatIsNotASchemaFailure() {
	err := invalid(errors.New("something else went wrong"), "Thing")

	s.Require().ErrorIs(err, ErrInvalid)
	s.Require().ErrorContains(err, "Thing")
	s.Require().ErrorContains(err, "something else went wrong")

	var fault *InvalidError
	s.Require().NotErrorAs(err, &fault,
		"nothing can name a field, so it does not pretend to")
}

// TestAFailureWithNoReasonOfItsOwn covers the message when the library gives
// a field and no explanation.
//
// Passing the empty reason through refuses a document with a field name and a
// blank after it. The error's own text is worse prose than a reason and is not
// nothing.
func (s *FallbackTestSuite) TestAFailureWithNoReasonOfItsOwn() {
	bare := &openapi3.SchemaError{
		Schema: &openapi3.Schema{},
		Value:  "whatever",
	}

	s.Require().Empty(bare.Reason, "the case this exists for")
	s.Require().NotEmpty(reasonOf(bare))

	spoken := &openapi3.SchemaError{
		Schema: &openapi3.Schema{},
		Value:  "whatever",
		Reason: "it is wrong",
	}

	s.Require().Equal("it is wrong", reasonOf(spoken),
		"a reason it has is the one it keeps")
}

// TestAFailurePointingAtNothing covers a schema error carrying no pointer,
// which would otherwise name an empty field.
func (s *FallbackTestSuite) TestAFailurePointingAtNothing() {
	err := invalid(&openapi3.SchemaError{
		Schema: &openapi3.Schema{},
		Value:  "whatever",
		Reason: "it is wrong",
	}, "Thing")

	var fault *InvalidError
	s.Require().ErrorAs(err, &fault)
	s.Require().Equal("the document", fault.Field)
}

func TestFallbackTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(FallbackTestSuite))
}
