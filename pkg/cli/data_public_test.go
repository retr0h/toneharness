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

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
)

// DataPublicTestSuite covers answering as data rather than as a table.
type DataPublicTestSuite struct {
	suite.Suite
}

// TestItWritesSomethingAReaderCanParse is the whole point of the form.
func (s *DataPublicTestSuite) TestItWritesSomethingAReaderCanParse() {
	var buf bytes.Buffer

	s.Require().NoError(cli.Data(&buf, map[string]any{
		"gear": "Ampeg SVT", "role": "amp",
	}))

	var back map[string]any
	s.Require().NoError(json.Unmarshal(buf.Bytes(), &back))

	s.Require().Equal("Ampeg SVT", back["gear"])
	s.Require().Equal("amp", back["role"])
}

// TestItIsIndentedAndEndsInANewline covers the shape somebody actually reads.
//
// Indented because the usual reader is a person checking what an agent will
// see, and a newline because the usual next thing is a pipe.
func (s *DataPublicTestSuite) TestItIsIndentedAndEndsInANewline() {
	var buf bytes.Buffer

	s.Require().NoError(cli.Data(&buf, map[string]int{"blocks": 2}))

	s.Require().Contains(buf.String(), "\n  \"blocks\"")
	s.Require().True(strings.HasSuffix(buf.String(), "\n"))
}

// TestSomethingThatCannotBeWrittenIsReported covers a value with no data
// form.
//
// A channel has none. The answer is an error naming the problem rather than a
// half-written document, because a reader cannot tell a truncated one from a
// short one.
func (s *DataPublicTestSuite) TestSomethingThatCannotBeWrittenIsReported() {
	var buf bytes.Buffer

	err := cli.Data(&buf, make(chan int))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing the answer")
	s.Require().Empty(buf.String())
}

// TestAWriterThatRefusesIsReported covers the other half.
func (s *DataPublicTestSuite) TestAWriterThatRefusesIsReported() {
	err := cli.Data(refuses{}, map[string]string{"gear": "Ampeg SVT"})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing the answer")
}

// errRefused is what a writer that takes nothing says.
var errRefused = errors.New("nothing doing")

// refuses is a writer that will not take anything.
type refuses struct{}

func (refuses) Write(
	_ []byte,
) (int, error) {
	return 0, errRefused
}

func TestDataPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DataPublicTestSuite))
}
