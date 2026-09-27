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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/tone"
)

// ShippedPublicTestSuite checks the worked examples the docs point at.
//
// The rig side of this has been checked since it existed and the ask side was
// not, so an example carrying a field the contract had stopped allowing would
// have been found by whoever copied it. An example that no longer parses is
// worse than no example: it is the first thing somebody copies.
type ShippedPublicTestSuite struct {
	suite.Suite
}

// TestEveryExampleLoads reads every worked example with the loader its own
// `schema` line names.
//
// Two kinds live in one directory, because a request and the setup it is
// answered against are written together and read together, so the file says
// which it is and this believes it.
func (s *ShippedPublicTestSuite) TestEveryExampleLoads() {
	paths, err := filepath.Glob(
		filepath.Join("..", "..", "..", "examples", "tonespec", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no examples found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			body, err := os.ReadFile(path) //nolint:gosec // a path this glob found
			s.Require().NoError(err)

			if bytes.Contains(body, []byte("schema: "+tone.SetupSchema)) {
				_, err = tone.LoadSetup(bytes.NewReader(body))
				s.Require().NoError(err)

				return
			}

			_, err = tone.Load(bytes.NewReader(body))
			s.Require().NoError(err)
		})
	}
}

func TestShippedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShippedPublicTestSuite))
}
