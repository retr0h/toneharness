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

package corpus_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/corpus"
)

// OpenPublicTestSuite covers reading statistics from a path or from the binary.
type OpenPublicTestSuite struct {
	suite.Suite
}

// TestNoPathIsTheOnesThatShip is what every caller who has not measured their
// own corpus gets.
func (s *OpenPublicTestSuite) TestNoPathIsTheOnesThatShip() {
	got, err := corpus.Open("")

	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().NotEmpty(got.Models, "the measured corpus, not an empty one")
}

// TestAPathIsRead covers somebody's own statistics.
func (s *OpenPublicTestSuite) TestAPathIsRead() {
	had, err := corpus.Open("")
	s.Require().NoError(err)

	at := filepath.Join(s.T().TempDir(), "stats.json.gz")

	body, err := os.ReadFile(filepath.Join("data", "hx-stomp.stats.json.gz"))
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(at, body, 0o600))

	got, err := corpus.Open(at)

	s.Require().NoError(err)
	s.Require().Len(got.Models, len(had.Models))
}

// TestAPathThatIsNotThereIsReported covers the answer being a failure rather
// than a silent fall back to the built-in ones.
//
// Somebody who named a file meant that file. Falling back would answer a
// question about their corpus with figures from ours.
func (s *OpenPublicTestSuite) TestAPathThatIsNotThereIsReported() {
	_, err := corpus.Open(filepath.Join(s.T().TempDir(), "nowhere.json.gz"))

	s.Require().ErrorContains(err, "nowhere.json.gz")
}

// TestAFileThatWillNotDecodeIsReported covers something that is not statistics.
func (s *OpenPublicTestSuite) TestAFileThatWillNotDecodeIsReported() {
	at := filepath.Join(s.T().TempDir(), "stats.json.gz")
	s.Require().NoError(os.WriteFile(at, []byte("not gzipped json"), 0o600))

	_, err := corpus.Open(at)

	s.Require().Error(err)
}

func TestOpenPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OpenPublicTestSuite))
}
