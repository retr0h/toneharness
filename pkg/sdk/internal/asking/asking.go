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

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/result"
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

	built, notes, err := translate.Translate(spec, setup, translate.Deps{
		Catalog: cat, Measured: lib,
	})

	return result.Resolved{Rig: built, Notes: notes}, err
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
