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

package asking_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/asking"
)

// CurvesInPublicTestSuite covers reading one block's swept curves out of a
// tree of them.
type CurvesInPublicTestSuite struct {
	suite.Suite
}

// TestCurvesIn reads a sweep, or says there is none.
func (s *CurvesInPublicTestSuite) TestCurvesIn() {
	dir := s.T().TempDir()

	// A real sweep, copied so the test does not depend on the committed tree
	// having any particular block in it.
	swept := filepath.Join(dir, "HD2_DistTeemah.json")
	s.Require().NoError(os.WriteFile(swept, []byte(`{
		"device": "HX Stomp",
		"gear": "Teemah!",
		"block": "HD2_DistTeemah",
		"controls": {"Drive": {"index": 1, "control": "Drive", "points": []}}
	}`), 0o600))

	// And a file that is not one, because a tree somebody points at is not a
	// tree somebody curated.
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "HD2_DistBroken.json"), []byte("not json"), 0o600))

	tests := []struct {
		name string
		// dir is the tree to read, and model the block to look for.
		dir   string
		model string
		// found is whether a sweep came back.
		found bool
		// nothing is whether no accessor is handed out at all, which is how a
		// caller tells "no sweeps were offered" from "that block is not in
		// them".
		nothing bool
	}{
		{
			name: "a block the campaign reached",
			dir:  dir, model: "HD2_DistTeemah", found: true,
		},
		{
			// 179 of the device's 614 blocks are swept, so this is the
			// ordinary answer rather than an error.
			name: "a block it did not",
			dir:  dir, model: "HD2_DistNotSwept", found: false,
		},
		{
			name: "a file that will not decode",
			dir:  dir, model: "HD2_DistBroken", found: false,
		},
		{
			name: "no tree at all", dir: "", model: "HD2_DistTeemah",
			nothing: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			read := asking.CurvesIn(tt.dir)

			if tt.nothing {
				s.Require().Nil(read)

				return
			}

			s.Require().NotNil(read)

			got, ok := read(asking.ModelID(tt.model), "")
			s.Require().Equal(tt.found, ok)

			if tt.found {
				s.Require().Equal("Teemah!", got.Gear)
			}
		})
	}
}

func TestCurvesInPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(CurvesInPublicTestSuite))
}
