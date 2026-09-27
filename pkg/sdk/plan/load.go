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

package plan

import (
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// ErrNotAPlan reports a document that is not a plan this can read.
var ErrNotAPlan = errors.New("not a readable plan")

// NotAPlanError says what was wrong with it.
type NotAPlanError struct {
	// Why names the thing that was wrong.
	Why string
}

// Error implements the error interface.
func (e *NotAPlanError) Error() string {
	return fmt.Sprintf("not a readable plan: %s", e.Why)
}

// Unwrap returns ErrNotAPlan so callers can match with errors.Is.
func (*NotAPlanError) Unwrap() error { return ErrNotAPlan }

// Load reads a plan, refusing any field it does not know.
//
// Strictly, which is the whole reason this exists rather than a plain
// Unmarshal. A plan is machine-written, so the case worth guarding is somebody
// exporting one, editing a knob by hand and misspelling the key: decoding
// loosely would drop the line and report success, which is how `requires` sat
// in the contract for months doing nothing.
//
// No contract behind it, unlike a rig or an ask. Those are formats a person
// authors and their schemas say what a field may hold; this is written by a
// driver, and the only thing a reader of one needs protecting from is a field
// nobody defined.
func Load(
	r io.Reader,
) (Plan, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	var out Plan

	if err := dec.Decode(&out); err != nil {
		return Plan{}, &NotAPlanError{Why: err.Error()}
	}

	if len(out.Blocks) == 0 {
		return Plan{}, &NotAPlanError{Why: "it holds no blocks"}
	}

	return out, nil
}

// Write writes a plan out as YAML.
func Write(
	w io.Writer,
	of Plan,
) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	if err := enc.Encode(of); err != nil {
		return fmt.Errorf("writing the plan: %w", err)
	}

	return enc.Close()
}
