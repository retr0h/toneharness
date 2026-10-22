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

// Package asking turns what somebody wants into the rig that makes it.
//
// The step between the two documents a person writes and the plan a preset is
// compiled from. It reads a ToneSpec and a Setup, hands them to translate, and
// answers with the rig and what the translation made of the request.
//
// Here rather than in pkg/cli, where it started. A terminal and an agent both
// need this, and an operation living in the renderer is one the MCP server
// cannot reach: the whole ToneSpec path was unavailable over MCP for exactly
// that reason.
package asking

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// Ask is a request to resolve, by where the documents are.
type Ask struct {
	// Spec is the ToneSpec to read.
	Spec string
	// Setup is what somebody owns, and is optional.
	Setup string
}

// Resolve reads a request and answers with the rig it resolves to.
//
// The notes come back whether it succeeded or not, because a request that
// could not be honoured has usually said why in them and the error alone is
// the half that does not help.
func Resolve(
	ctx context.Context,
	in Ask,
	cat *catalog.Catalog,
	lib measured.Library,
	rigNamed func(name string) (rig.Spec, bool),
	// sweeps is the tree of committed curves, which say what each control of
	// a block does. Empty means none were read, and a block added to a chain
	// then arrives at the catalog's defaults.
	sweeps string,
) (result.Resolved, error) {
	if err := ctx.Err(); err != nil {
		return result.Resolved{}, err
	}

	spec, err := read(in.Spec, tone.Load)
	if err != nil {
		return result.Resolved{}, err
	}

	// A setup is optional. Somebody asking what a record sounds like has not
	// necessarily said what they own, and the answer says what it assumed
	// rather than refusing.
	setup := tone.Setup{Schema: tone.SetupSchema}

	if in.Setup != "" {
		if setup, err = read(in.Setup, tone.LoadSetup); err != nil {
			return result.Resolved{}, err
		}
	}

	// The ask half. Resolving reads a request and answers with the gear, and the
	// document the request arrived in is this function's to put back together.
	ask := tone.Ask{}
	if spec.Ask != nil {
		ask = *spec.Ask
	}

	built, notes, err := translate.Translate(ask, setup, translate.Deps{
		Catalog: cat, Measured: lib,
		// Wired here because this is the one place above both: translate runs
		// before compile and must not import it, and compile owns the
		// vocabulary.
		UnknownWords: unknownWords,
		// Handed in for the same reason, from the other direction: a rig store
		// reads a directory and the shipped knowledge, and translate is what
		// produces a rig rather than what reads one.
		RigNamed: rigNamed,
		// And the sweeps, so a block this adds arrives set rather than at
		// whatever the catalog ships as a default. Nil when nobody named a
		// sweeps tree, which a release has not.
		CurvesFor: curvesIn(sweeps),
	})

	return result.Resolved{ID: spec.Id, Ask: spec.Ask, Rig: built, Notes: notes}, err
}

// read loads one of the two documents a person writes.
func read[T any](
	at string,
	load func(io.Reader) (T, error),
) (T, error) {
	var zero T

	f, err := os.Open(at) //nolint:gosec // the path is the requester's own file
	if err != nil {
		return zero, fmt.Errorf("reading %s: %w", at, err)
	}

	defer func() { _ = f.Close() }()

	out, err := load(f)
	if err != nil {
		return zero, fmt.Errorf("reading %s: %w", at, err)
	}

	return out, nil
}

// unknownWords adapts the compiler's vocabulary check to what translate takes.
//
// The types are separate on purpose. translate declares the shape it needs so
// it depends on nothing downhill of itself, and this is the seam where the two
// meet.
func unknownWords(
	words []string,
) []translate.UnknownWord {
	got := compile.CheckWords(words)

	out := make([]translate.UnknownWord, 0, len(got))
	for _, one := range got {
		out = append(out, translate.UnknownWord{Term: one.Term, Near: one.Near})
	}

	return out
}

// curvesIn reads one block's swept curves out of a tree of them.
//
// By model identifier, which is what the files are named after. A block the
// campaign never reached has no file and answers false, which is an ordinary
// answer: 179 of the device's 614 blocks are swept.
//
// Nil for an empty path rather than a function that always fails, so the
// caller can tell "no sweeps were offered" from "that block is not in them".
func curvesIn(
	dir string,
) func(catalog.ModelID) (measured.Curves, bool) {
	if dir == "" {
		return nil
	}

	return func(model catalog.ModelID) (measured.Curves, bool) {
		f, err := os.Open(filepath.Join(dir, string(model)+".json"))
		if err != nil {
			return measured.Curves{}, false
		}

		defer func() { _ = f.Close() }()

		got, err := measured.LoadCurves(f)
		if err != nil {
			return measured.Curves{}, false
		}

		return got, true
	}
}

// CurvesIn is [curvesIn], for the test that covers it.
//
// Exported here rather than through an export_test.go because this package
// already answers across the file boundary and one more name is cheaper than a
// second shim.
var CurvesIn = curvesIn

// ModelID is the identifier a sweep is filed under, re-exported so a test in
// this package's own test package can name one without importing the catalog.
type ModelID = catalog.ModelID
