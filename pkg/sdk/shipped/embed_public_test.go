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
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/shipped"
)

// EmbedPublicTestSuite covers the rigs the binary carries.
type EmbedPublicTestSuite struct {
	suite.Suite
}

// core is the marketplace tier this package is a copy of.
//
// The tier rather than one directory inside it, because a tier holds a
// directory per kind of subject: `artists/` for people and `genres/` for a
// sound nobody owns. Naming one of them would let the other drift unwatched,
// which is the whole failure this test exists for.
var core = filepath.Join("..", "..", "..", "marketplace", "core")

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
	want := map[string][]byte{}

	err := filepath.WalkDir(core, func(at string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), filepath.Ext(at) != ".yaml":
			return nil
		}

		body, err := os.ReadFile(at) //nolint:gosec // a path this walk found
		if err != nil {
			return err
		}

		want[filepath.Base(at)] = body

		return nil
	})
	s.Require().NoError(err)
	s.Require().NotEmpty(want, "the marketplace has a core tier to copy")

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

// TestNoShippedControlClaimsMorePrecisionThanItHas is task 275's guard.
//
// A control written down carries at most a thousandth of its own travel, which
// `catalog.Rounded` owns and two paths go through: `compile.set` for a block
// the compiler aimed and `translate.tuned` for one a build added. Neither
// rounding was wrong and the arithmetic behind the first one was: snapping a
// value onto a grid of thousandths and dividing back out of it is not
// representable, so -3.75 on an equaliser band came out
// -3.7499999999999996 and went into five of these documents.
//
// Against the files rather than the arithmetic, because that is where it was
// found and where somebody reads it. Six places is well past what any control
// here claims: the finest is a compressor's Attack at four.
func (s *EmbedPublicTestSuite) TestNoShippedControlClaimsMorePrecisionThanItHas() {
	// A control line and its value, which is what a document writes a number
	// as: two spaces of indent per level and the value quoted.
	long := regexp.MustCompile(`^\s+[^:\s][^:]*: "-?\d+\.\d{7,}"$`)

	err := filepath.WalkDir(core, func(at string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), filepath.Ext(at) != ".yaml":
			return nil
		}

		body, err := os.ReadFile(at) //nolint:gosec // a path this walk found
		if err != nil {
			return err
		}

		for i, line := range strings.Split(string(body), "\n") {
			// The preset's own fields are the device's numbers rather than
			// anything computed here: `@variax_toneknob` ships as
			// -0.10000002384185791 in every blank template Line 6 write, and
			// rewriting it would be this project editing the pedal's data.
			if strings.Contains(line, "'@") {
				continue
			}

			s.Require().NotRegexp(long, line,
				"%s:%d claims more precision than the control has", at, i+1)
		}

		return nil
	})
	s.Require().NoError(err)
}
