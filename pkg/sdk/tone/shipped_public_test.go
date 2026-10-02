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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/shipped"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
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

// TestEveryDocumentThatShipsLoads covers every shipped document and every
// worked example loading against the contract.
//
// One method and one table, so a case is a row rather than a file.
func (s *ShippedPublicTestSuite) TestEveryDocumentThatShipsLoads() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Reads every worked example with the loader its own name says.
			//
			// Two kinds live in one directory, because a request and the setup
			// it is answered against are written together and read together.
			// A `.setup.yaml` is what somebody owns; everything else is a
			// ToneSpec. A `.plan.yaml` is the device half of a preset and
			// pkg/sdk/plan is what reads one.
			name: "every example loads",
			then: func() {
				paths, err := filepath.Glob(filepath.Join(
					"..", "..", "..", "marketplace", "core", "examples", "*.yaml"))
				s.Require().NoError(err)
				s.Require().NotEmpty(paths, "no examples found to check")

				for _, path := range paths {
					if strings.HasSuffix(path, ".plan.yaml") {
						continue
					}

					s.Run(filepath.Base(path), func() {
						body, err := os.ReadFile(path) //nolint:gosec // a path this glob found
						s.Require().NoError(err)

						if strings.HasSuffix(path, ".setup.yaml") {
							_, err = tone.LoadSetup(bytes.NewReader(body))
							s.Require().NoError(err)

							return
						}

						_, err = tone.Load(bytes.NewReader(body))
						s.Require().NoError(err)
					})
				}
			},
		},
		{
			// Reads every document this binary ships.
			//
			// The gap this closes was invisible from either side. pkg/sdk/rig
			// checked every shipped rig and skipped the ask beside it, saying
			// "pkg/sdk/tone checks those", and pkg/sdk/tone checked
			// marketplace/core/examples/ and never the shipped documents. Each
			// test was right about itself and the sentence joining them was not,
			// so the documents this project ships as its own knowledge were the
			// only ones nothing validated.
			name: "every shipped document loads",
			then: func() {
				paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.yaml"))
				s.Require().NoError(err)
				s.Require().NotEmpty(paths, "no shipped documents found to check")

				for _, path := range paths {
					s.Run(filepath.Base(path), func() {
						body, err := fs.ReadFile(shipped.FS, path)
						s.Require().NoError(err)

						doc, err := tone.Load(bytes.NewReader(body))
						s.Require().NoError(err)

						// The identifier is also the filename stem, which is what makes
						// a rig findable: the loader globs the directory and the name it
						// answers to is the one inside the file.
						s.Require().Equal(
							strings.TrimSuffix(filepath.Base(path), ".yaml"), doc.Id,
							"%s says it is %q", path, doc.Id)
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

func TestShippedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShippedPublicTestSuite))
}
