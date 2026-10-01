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

package rig_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/shipped"
)

// ShippedPublicTestSuite checks the rigs this repository ships.
//
// Drift between the Go types and the contract is impossible — the types are
// generated from data/rigspec.openapi.yaml. What generation does not
// guarantee is that the files on disk satisfy it, or that a rig's filename
// matches the identifier inside it.
type ShippedPublicTestSuite struct {
	suite.Suite
}

// TestEveryRigThatShipsLoads covers every rig and every example loading
// against the contract.
//
// One method and one table, so a case is a row rather than a file.
func (s *ShippedPublicTestSuite) TestEveryRigThatShipsLoads() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "every shipped rig loads",
			then: func() {
				paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.yaml"))
				s.Require().NoError(err)
				s.Require().NotEmpty(paths, "no rigs found to check")

				for _, path := range paths {
					// The ask beside each rig lives in the same directory and is a
					// different document. pkg/sdk/tone checks those; this one is the rigs.
					if strings.HasSuffix(path, ".tone.yaml") {
						continue
					}

					s.Run(filepath.Base(path), func() {
						f, err := shipped.FS.Open(path)
						s.Require().NoError(err)

						defer func() { s.Require().NoError(f.Close()) }()

						spec, err := rig.Load(f)
						s.Require().NoError(err)

						s.Require().Equal(spec.ID+".yaml", filepath.Base(path),
							"a rig must be findable by name without opening it")
					})
				}
			},
		},
		{
			// Checks the fully-filled rigs the docs point at.
			//
			// An example that no longer parses is worse than no example: it
			// is the first thing somebody copies.
			name: "every example loads",
			then: func() {
				paths, err := filepath.Glob(filepath.Join("..", "..", "..", "resources", "reference", "rigspec", "*.yaml"))
				s.Require().NoError(err)
				s.Require().NotEmpty(paths, "no examples found to check")

				for _, path := range paths {
					s.Run(filepath.Base(path), func() {
						f, err := os.Open(path)
						s.Require().NoError(err)

						defer func() { s.Require().NoError(f.Close()) }()

						_, err = rig.Load(f)
						s.Require().NoError(err)
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
