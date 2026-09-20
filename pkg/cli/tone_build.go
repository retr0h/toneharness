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
	"fmt"
	"io"
	"os"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/tone"
	"github.com/retr0h/tonestack/pkg/sdk/translate"
)

// ToneBuildOptions is what turning a request into a rig needs to know.
type ToneBuildOptions struct {
	Ask   string
	Setup string
	Out   string
	// AsData asks for the rig and the notes as one document rather than as
	// a painted section and a YAML block.
	//
	// Its own field rather than going through the shared answer helper,
	// because this is the one command that does work as well as rendering:
	// it resolves a request and may write a file, so there is no single value
	// a renderer could be handed.
	AsData bool
}

// Resolved is what building a rig from a request produced.
//
// Both halves, because a run that only answered with the rig would hide what
// it could not honour, and the notes are how somebody finds out that the amp
// they asked for is not one the device has.
type Resolved struct {
	Rig   rig.Spec        `json:"rig"`
	Notes translate.Notes `json:"notes"`
}

// ToneBuild reads a request and a setup and writes the rig they resolve to.
func ToneBuild(
	w io.Writer,
	opts ToneBuildOptions,
) error {
	ask, err := read(opts.Ask, tone.Load)
	if err != nil {
		return err
	}

	// A setup is optional. Somebody asking what a record sounds like has not
	// necessarily told anybody what they own, and the rig says what it
	// assumed instead of refusing.
	setup := tone.Setup{Schema: tone.SetupSchema}

	if opts.Setup != "" {
		if setup, err = read(opts.Setup, tone.LoadSetup); err != nil {
			return err
		}
	}

	cat, err := catalog.BuiltIn()
	if err != nil {
		return err
	}

	lib, err := measured.BuiltIn()
	if err != nil {
		return err
	}

	spec, notes, err := translate.Translate(ask, setup, translate.Deps{
		Catalog: cat, Measured: lib,
	})

	if opts.AsData {
		// Still an error when it failed, and the notes still travel, because
		// what could not be honoured is the useful half either way.
		if writing := Data(w, Resolved{Rig: spec, Notes: notes}); writing != nil {
			return writing
		}

		return err
	}

	// Reported before the error is returned. A request that could not be
	// honoured has usually said why in the notes, and the error alone is the
	// half that does not help.
	sayNotes(w, notes)

	if err != nil {
		return err
	}

	return put(w, spec, opts.Out)
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
			"    tonestack presets compile --rig %s --out a.hlx\n\n", at, at)

	return err
}
