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
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/tone"
)

// LoadPublicTestSuite covers reading a request and a setup off disk.
type LoadPublicTestSuite struct {
	suite.Suite
}

// TestReadsARequest covers the ordinary case.
func (s *LoadPublicTestSuite) TestReadsARequest() {
	spec, err := tone.Load(strings.NewReader(`
schema: ToneSpec
genre: pop-punk
words:
  - term: bright
  - term: tight-low-end
    evidence: [{ kind: llm }]
like:
  artist: Mike Dirnt
  years: { from: 1994, to: 2004 }
nudges:
  - word: darker
    steps: 2
`))

	s.Require().NoError(err)
	s.Require().Equal("pop-punk", *spec.Genre)
	s.Require().Len(*spec.Words, 2)
	s.Require().Equal("bright", (*spec.Words)[0].Term)
	// A word carries why it is believed, because that is what sizes how far it
	// moves a control. The first here carries none, which is legal and is what
	// a request somebody typed looks like.
	s.Require().Nil((*spec.Words)[0].Evidence)
	s.Require().Equal("tight-low-end", (*spec.Words)[1].Term)
	s.Require().Len(*(*spec.Words)[1].Evidence, 1)
	s.Require().Equal("Mike Dirnt", *spec.Like.Artist)
	s.Require().Equal(1994, spec.Like.Years.From)
	s.Require().Equal("darker", (*spec.Nudges)[0].Word)
}

// TestReadsASetup covers the other document.
func (s *LoadPublicTestSuite) TestReadsASetup() {
	setup, err := tone.LoadSetup(strings.NewReader(`
schema: Setup
device:
  model: HX Stomp
instruments:
  - gear: Fender Jazz Bass
    strings: flat
    default: true
owns:
  - kind: ir
    name: Owned 4x10
    slot: 3
`))

	s.Require().NoError(err)
	s.Require().Equal("HX Stomp", setup.Device.Model)
	s.Require().Equal(tone.StringsFlat, *(*setup.Instruments)[0].Strings)
	s.Require().Equal(tone.OwnedIR, (*setup.Owns)[0].Kind)
}

// TestAMisspeltFieldIsRefused is why the raw document is checked first.
//
// Decoding drops what the types have no field for, so a request checked after
// decoding is checked with its own mistake already removed: the line would be
// gone and nothing said about it.
func (s *LoadPublicTestSuite) TestAMisspeltFieldIsRefused() {
	_, err := tone.Load(strings.NewReader("schema: ToneSpec\ngnere: punk\n"))

	s.Require().ErrorIs(err, tone.ErrInvalid)
	s.Require().Contains(err.Error(), "gnere")
}

// TestTheWrongDocumentSaysSo covers passing a Setup where the ask goes.
//
// The enum would refuse it anyway and say `schema` is not an allowed value,
// which is true and unhelpful to somebody who passed the wrong file.
func (s *LoadPublicTestSuite) TestTheWrongDocumentSaysSo() {
	_, err := tone.Load(strings.NewReader("schema: Setup\n"))

	s.Require().ErrorIs(err, tone.ErrInvalid)
	s.Require().Contains(err.Error(), "says Setup, so this is not a ToneSpec")

	_, err = tone.LoadSetup(strings.NewReader("schema: ToneSpec\n"))
	s.Require().Contains(err.Error(), "says ToneSpec, so this is not a Setup")
}

// TestADocumentThatIsNotFieldsIsRefused covers a file holding a list.
func (s *LoadPublicTestSuite) TestADocumentThatIsNotFieldsIsRefused() {
	_, err := tone.Load(strings.NewReader("- one\n- two\n"))

	s.Require().ErrorIs(err, tone.ErrInvalid)
	s.Require().Contains(err.Error(), "is not a set of fields")
}

// TestUnreadableYAMLIsRefused covers a file that is not YAML at all.
func (s *LoadPublicTestSuite) TestUnreadableYAMLIsRefused() {
	_, err := tone.Load(strings.NewReader("\tschema: [unclosed\n"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "decoding the ToneSpec")
}

// TestAReadFailureIsReported covers the reader itself failing.
func (s *LoadPublicTestSuite) TestAReadFailureIsReported() {
	_, err := tone.LoadSetup(iotest{})

	s.Require().ErrorContains(err, "reading the Setup")
}

// TestANumberTooLargeForTheTypesIsRefused is the case the schema allows.
//
// JSON Schema calls 2000000000000000000000 an integer and Go's int cannot
// hold it, so without the second check this returns a document with the field
// silently zeroed and no error at all. The contract caps a year at 2100, so
// the number has to arrive somewhere uncapped: `slot` on an owned impulse
// response has a minimum and no maximum.
func (s *LoadPublicTestSuite) TestANumberTooLargeForTheTypesIsRefused() {
	_, err := tone.LoadSetup(strings.NewReader(`
schema: Setup
owns:
  - kind: ir
    name: Owned 4x10
    slot: 2000000000000000000000
`))

	s.Require().ErrorContains(err, "decoding the Setup")
}

// TestWritesWhatItRead covers the round trip.
func (s *LoadPublicTestSuite) TestWritesWhatItRead() {
	genre := "grunge"
	spec := tone.Spec{Schema: "ToneSpec", Genre: &genre}

	var buf bytes.Buffer
	s.Require().NoError(tone.Write(&buf, spec))

	back, err := tone.Load(&buf)
	s.Require().NoError(err)
	s.Require().Equal("grunge", *back.Genre)
}

// TestWritesASetup covers the other document's round trip.
func (s *LoadPublicTestSuite) TestWritesASetup() {
	setup := tone.Setup{
		Schema: "Setup",
		Device: &tone.Device{Model: "HX Stomp"},
	}

	var buf bytes.Buffer
	s.Require().NoError(tone.WriteSetup(&buf, setup))

	back, err := tone.LoadSetup(&buf)
	s.Require().NoError(err)
	s.Require().Equal("HX Stomp", back.Device.Model)
}

// TestAnInvalidDocumentIsNotWritten covers the check before the render.
//
// Writing one that does not meet its own contract would put a file into the
// world that nothing else will accept.
func (s *LoadPublicTestSuite) TestAnInvalidDocumentIsNotWritten() {
	var buf bytes.Buffer

	s.Require().ErrorIs(
		tone.Write(&buf, tone.Spec{}), tone.ErrInvalid)
	s.Require().ErrorIs(
		tone.WriteSetup(&buf, tone.Setup{}), tone.ErrInvalid)
	s.Require().Empty(buf.String())
}

// TestAWriteFailureIsReported covers the writer itself failing.
func (s *LoadPublicTestSuite) TestAWriteFailureIsReported() {
	err := tone.Write(broken{}, tone.Spec{Schema: "ToneSpec"})

	s.Require().ErrorContains(err, "writing the ToneSpec")
}

// iotest is a reader that always fails.
type iotest struct{}

func (iotest) Read(
	[]byte,
) (int, error) {
	return 0, errors.New("no")
}

// broken is a writer that always fails.
type broken struct{}

func (broken) Write(
	[]byte,
) (int, error) {
	return 0, errors.New("no")
}

func TestLoadPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LoadPublicTestSuite))
}
