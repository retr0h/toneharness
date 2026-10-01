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
package shipped_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/shipped"
)

// EmbedPublicTestSuite covers the rigs the binary carries.
type EmbedPublicTestSuite struct {
	suite.Suite
}

// core is the marketplace tier this package is a copy of.
var core = filepath.Join("..", "..", "..", "marketplace", "core", "artists")

// TestTheEmbeddedRigsAreTheMarketplacesCore covers the copy this package holds.
//
// `go:embed` cannot name a path above its own package directory, so the rigs
// people read and submit live at the top of the repository in marketplace/ and
// this package holds a generated copy. Two copies of the same thing drift, and
// the way this one drifts is silent: somebody edits a rig in the marketplace,
// does not run `just generate`, and the binary goes on serving the old gear while
// the file they edited says otherwise.
//
// Byte for byte, both directions. A rig added to the marketplace and not packed
// fails, and so does one deleted there and left in the binary.
func (s *EmbedPublicTestSuite) TestTheEmbeddedRigsAreTheMarketplacesCore() {
	entries, err := os.ReadDir(core)
	s.Require().NoError(err)
	s.Require().NotEmpty(entries, "the marketplace has a core tier to copy")

	want := map[string][]byte{}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		body, err := os.ReadFile(filepath.Join(core, entry.Name()))
		s.Require().NoError(err)

		want[entry.Name()] = body
	}

	got := map[string][]byte{}

	err = fs.WalkDir(shipped.FS, ".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), filepath.Ext(path) != ".yaml":
			return nil
		}

		body, err := shipped.FS.ReadFile(path)
		if err != nil {
			return err
		}

		got[filepath.Base(path)] = body

		return nil
	})
	s.Require().NoError(err)

	for name, body := range want {
		s.Require().Contains(got, name,
			"%s is in the marketplace and not in the binary: run `just generate`",
			name)
		s.Require().Equal(string(body), string(got[name]),
			"%s differs between the marketplace and the binary: run `just generate`",
			name)
	}

	for name := range got {
		s.Require().Contains(want, name,
			"%s is in the binary and no longer in the marketplace: run `just generate`",
			name)
	}
}

func TestEmbedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EmbedPublicTestSuite))
}
