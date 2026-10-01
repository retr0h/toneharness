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

// TestEveryAskThatShipsLoads covers every shipped ask and every example
// loading against the contract.
//
// One method and one table, so a case is a row rather than a file.
func (s *ShippedPublicTestSuite) TestEveryAskThatShipsLoads() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Reads every worked example with the loader its own `schema`
			// line names.
			//
			// Two kinds live in one directory, because a request and the
			// setup it is answered against are written together and read
			// together, so the file says which it is and this believes it.
			name: "every example loads",
			then: func() {
				paths, err := filepath.Glob(
					filepath.Join("..", "..", "..", "marketplace", "core", "examples", "*.tone.yaml"))
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
			},
		},
		{
			// Reads the ask beside every rig this binary ships.
			//
			// The gap this closes was invisible from either side. pkg/sdk/rig
			// checks every shipped rig and skips the `.tone.yaml` files
			// beside them, saying "pkg/sdk/tone checks those", and
			// pkg/sdk/tone checked marketplace/core/examples/ and never the shipped asks.
			// Each test was right about itself and the sentence joining them
			// was not, so the documents this project ships as its own
			// knowledge were the only ones nothing validated.
			name: "every shipped ask loads",
			then: func() {
				paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.tone.yaml"))
				s.Require().NoError(err)
				s.Require().NotEmpty(paths, "no shipped asks found to check")

				for _, path := range paths {
					s.Run(filepath.Base(path), func() {
						body, err := fs.ReadFile(shipped.FS, path)
						s.Require().NoError(err)

						_, err = tone.Load(bytes.NewReader(body))
						s.Require().NoError(err)

						// An ask is reached through the rig it sits beside, so one with no
						// rig is unreachable. It loads cleanly and nothing can ever ask it.
						//
						// Either suffix counts. `.rig.yaml` is what anything writes now
						// and a bare `.yaml` is what a rig read off a device still is, and
						// an ask is paired by the stem rather than by the spelling.
						stem := strings.TrimSuffix(path, ".tone.yaml")

						var found bool
						for _, suffix := range []string{".rig.yaml", ".yaml"} {
							if _, err := fs.Stat(shipped.FS, stem+suffix); err == nil {
								found = true

								break
							}
						}

						s.Require().True(found, "%s has no rig beside it", path)
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
