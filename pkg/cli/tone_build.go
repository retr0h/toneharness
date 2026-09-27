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

package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/retr0h/toneharness/pkg/cli/internal/paint"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

//go:generate go tool go.uber.org/mock/mockgen -source=tone_build.go -destination=internal/mocks/resolver.gen.go -package=mocks

// Resolver turns a request and a setup into the rig they describe.
//
// Declared here rather than taking *sdk.Client, because this is all this
// renderer needs and a test for how an answer looks should not need a catalog
// and six hundred measurements to produce one. *sdk.Client satisfies it.
type Resolver interface {
	Tone(ctx context.Context, in sdk.Ask) (sdk.Resolved, error)
}

// ToneBuildOptions is what turning a request into a rig needs to know.
type ToneBuildOptions struct {
	// Client resolves the request.
	//
	// Optional, the way every other collaborator in this repository is: a
	// zero value reaches the real thing, and a caller names this only to
	// stand something else in for it. That is what net/http does with a nil
	// Transport.
	Client Resolver
	Ask    string
	Setup  string
	Out    string
	// AsData asks for the rig and the notes as one document rather than as
	// a painted section and a YAML block.
	//
	// Its own field rather than going through the shared answer helper,
	// because this is the one command that does work as well as rendering:
	// it resolves a request and may write a file, so there is no single value
	// a renderer could be handed.
	AsData bool
}

// resolver is what the request is resolved by, real unless one was handed in.
func (o ToneBuildOptions) resolver() Resolver {
	if o.Client != nil {
		return o.Client
	}

	return sdk.New()
}

// ToneBuild reads a request and a setup and writes the rig they resolve to.
//
// The resolving is sdk.Client.Tone's, not this package's. It used to be here,
// which made the whole ToneSpec path unreachable over MCP: an agent calls a
// tool rather than a command, and an operation living in the renderer is one
// no tool can call. What is left here is how the answer looks.
func ToneBuild(
	w io.Writer,
	opts ToneBuildOptions,
) error {
	got, err := opts.resolver().Tone(context.Background(), sdk.Ask{
		Spec: opts.Ask, Setup: opts.Setup,
	})

	if opts.AsData {
		// Written whether it failed or not, and the error still returned,
		// because what could not be honoured is the useful half either way.
		if writing := Data(w, got); writing != nil {
			return writing
		}

		return err
	}

	// Reported before the error is returned, for the same reason.
	sayNotes(w, got.Notes)

	if err != nil {
		return err
	}

	return put(w, got.Rig, opts.Out)
}

// say reports what the translation did and could not do.
//
// Both, because a run that only listed its failures would hide the choice it
// made for somebody, and a run that only listed its choices would hide the
// half of the ask it dropped.
func sayNotes(
	w io.Writer,
	notes translate.Notes,
) {
	if len(notes) == 0 {
		return
	}

	rows := make([][]string, 0, len(notes))

	for _, note := range notes {
		mark := "could not"
		if note.Honoured {
			mark = "did"
		}

		rows = append(rows, []string{mark, note.About, note.Said})
	}

	_ = paint.Section{
		Title:   "What it made of the request",
		Headers: []string{"", "about", "what happened"},
		Rows:    rows,
	}.Render(w)
}

// put writes the rig, to a file or to whatever is reading.
func put(
	w io.Writer,
	spec rig.Spec,
	at string,
) error {
	var buf bytes.Buffer

	if err := rig.Write(&buf, spec); err != nil {
		return err
	}

	if at == "" {
		_, err := io.WriteString(w, paint.YAML(w, buf.String()))

		return err
	}

	if err := os.WriteFile(at, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", at, err)
	}

	_, err := fmt.Fprintf(w,
		"\n  [ok] wrote %s\n\n  compile it with:\n"+
			"    toneharness presets compile --rig %s --out a.hlx\n\n", at, at)

	return err
}
