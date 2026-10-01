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

package plan_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"

	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// CoveragePublicTestSuite holds the plan's types to a plan somebody can read.
//
// The rig has the same test against its contract. The plan has no contract, so
// the Go type is the thing to walk: it is hand-written, and a field nobody has
// ever written is a field whose shape nobody has checked.
//
// Five of these types moved off the RigSpec, where they were declared in the
// schema and referenced by nothing in it. The rig's walk never reached them and
// this is the walk that does.
type CoveragePublicTestSuite struct {
	suite.Suite
}

// exempt names the fields no plan here can honestly carry, and why.
//
// A reason rather than a list, so adding to it is a decision somebody has to
// defend rather than a way past a failing test.
var exempt = map[string]string{}

// TestEveryFieldAppearsInAPlan walks the types and finds each field written
// down somewhere a person can read it.
func (s *CoveragePublicTestSuite) TestEveryFieldAppearsInAPlan() {
	declared := s.declared()
	s.Require().NotEmpty(declared)

	written := s.written()
	s.Require().NotEmpty(written)

	missing := []string(nil)

	for _, field := range declared {
		if written[field] {
			continue
		}

		if _, ok := exempt[field]; ok {
			continue
		}

		missing = append(missing, field)
	}

	s.Require().Empty(missing,
		"no plan under resources/reference/plan writes these. Add one to a "+
			"plan, or add "+
			"it to exempt with a reason: %v", missing)
}

// declared returns every json key the plan's types name, however deep.
func (s *CoveragePublicTestSuite) declared() []string {
	out := map[string]bool{}

	var walk func(t reflect.Type, depth int)

	walk = func(t reflect.Type, depth int) {
		// A plan nests a handful of levels. The bound is against a type that
		// grows a cycle rather than against the one there is.
		if depth > 8 {
			return
		}

		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
			t = t.Elem()
		}

		// A map's keys are data rather than fields: a block's params are the
		// device's own parameter names and its attrs are whatever it stored.
		if t.Kind() != reflect.Struct {
			return
		}

		for i := range t.NumField() {
			f := t.Field(i)

			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				continue
			}

			out[name] = true

			walk(f.Type, depth+1)
		}
	}

	walk(reflect.TypeOf(plan.Plan{}), 0)

	names := make([]string, 0, len(out))
	for name := range out {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// written returns every key any plan in this repository uses.
//
// The committed example and testdata/everything.yaml, which are two different
// claims. The example is a real device read and says what a plan looks like in
// practice; the fixture says every field has been spelled once. The rig makes
// the same split across its two examples.
//
// Off disk rather than assembled either way. The fields that go unexercised are
// the awkward ones, and a struct literal makes them as easy to write as the
// rest, so nothing would have caught them.
func (s *CoveragePublicTestSuite) written() map[string]bool {
	found, err := filepath.Glob(
		filepath.Join("..", "..", "..", "resources", "reference", "plan", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(found, "no plans under resources/reference/plan")

	fixture, err := filepath.Glob(filepath.Join("testdata", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(fixture, "no plans under testdata")

	found = append(found, fixture...)

	out := map[string]bool{}

	for _, at := range found {
		raw, err := os.ReadFile(at) //nolint:gosec // a path from this repository
		s.Require().NoError(err)

		var document any
		s.Require().NoError(yaml.Unmarshal(raw, &document), at)

		keys(document, out)
	}

	return out
}

// keys records every mapping key in a decoded document.
func keys(
	of any,
	out map[string]bool,
) {
	switch v := of.(type) {
	case map[string]any:
		for name, sub := range v {
			out[name] = true

			keys(sub, out)
		}
	case []any:
		for _, sub := range v {
			keys(sub, out)
		}
	}
}

func TestCoveragePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CoveragePublicTestSuite))
}
