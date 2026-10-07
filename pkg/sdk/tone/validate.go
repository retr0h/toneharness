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

// Package tone is what somebody wants to sound like, and what they have.
//
// Two documents, checked against the one contract they are generated from.
// The types in gen/ have shape but no rules: nothing in a generated struct
// stops a required field being empty or an enumeration carrying a word that
// is not in it.
//
// Those rules are enforced by checking against data/tonespec.openapi.yaml
// rather than against a second copy written in Go, for the reason the rig
// package gives: two copies drift, and a constraint added to the schema
// becomes a type nothing enforces.
package tone

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

// ErrInvalid reports a document that does not meet its own contract.
var ErrInvalid = errors.New("not valid")

// InvalidError says which part of a document is wrong, and which document.
type InvalidError struct {
	// Document is ToneSpec or Setup, so a message says which file to look in
	// when somebody is holding both.
	Document string
	// Field is the path to the offending value, as it appears in the file.
	Field string
	// Reason says what is wrong with it.
	Reason string
}

func (e *InvalidError) Error() string {
	return fmt.Sprintf("not a valid %s: %s %s", e.Document, e.Field, e.Reason)
}

func (*InvalidError) Unwrap() error { return ErrInvalid }

// Validate reports whether a request meets the contract.
func Validate(
	s Spec,
) error {
	if err := check(s, SchemaName); err != nil {
		return err
	}

	return OneSetOfSnapshots(s.Rig)
}

// OneSetOfSnapshots refuses a rig that says its snapshots twice.
//
// `sections:` is what somebody wants, by role: the build turns each one into a
// snapshot. A `snapshotN` under `preset:` is what a device stored. A rig carrying
// both is one question with two answers, and the schema cannot say so because the
// two live in different fields.
//
// An error rather than a precedence rule, because guessing does not produce a
// worse preset but a confusing one. A preset's snapshots are replaced rather than
// merged, so a rig naming `snapshot0` alone leaves the device holding one, and the
// sections then fail to fit with a message about the device's limits that points
// nowhere near the cause.
//
// A resolved rig names every snapshot the device had and no sections, which is
// consistent and the ordinary case.
//
// Exported because two packages ask it of the same rig and one definition is the
// point: `pkg/sdk/rig` validates a rig on its own, and this validates the document
// around one.
func OneSetOfSnapshots(
	r Rig,
) error {
	if r.Sections == nil || len(*r.Sections) == 0 || r.Preset == nil {
		return nil
	}

	for name := range *r.Preset {
		if !strings.HasPrefix(name, snapshotPrefix) {
			continue
		}

		return &InvalidError{
			Document: SchemaName,
			Field:    "rig.preset." + name,
			Reason: "is a snapshot, and this rig also has sections: say what " +
				"plays in each part of the song, or say the snapshots the device " +
				"stored, never both",
		}
	}

	return nil
}

// snapshotPrefix is what a device names every snapshot member with, snapshot0
// through snapshot7 on hardware that holds eight.
const snapshotPrefix = "snapshot"

// ValidateSetup reports whether a setup meets the contract.
func ValidateSetup(
	s Setup,
) error {
	return check(s, SetupSchema)
}

// check marshals a document and holds it to the schema of that name.
func check(
	of any,
	name string,
) error {
	// Through JSON, because that is the shape a schema describes, and the
	// tags doing the mapping came from the same document as the rules.
	body, err := json.Marshal(of)
	if err != nil {
		return fmt.Errorf("reading the %s: %w", name, err)
	}

	// What Marshal produced is JSON, so reading it back cannot fail.
	var document any
	_ = json.Unmarshal(body, &document)

	return against(document, name)
}

// contracts returns the two schemas, parsed once.
//
// Parsing an OpenAPI document is not cheap and this one never changes, so it
// happens on the first document checked and not again.
var contracts = sync.OnceValues(func() (map[string]*openapi3.Schema, error) {
	return load(Schema)
})

// load reads both contracts out of an OpenAPI document.
func load(
	raw []byte,
) (map[string]*openapi3.Schema, error) {
	doc, err := openapi3.NewLoader().LoadFromData(raw)
	if err != nil {
		return nil, fmt.Errorf("reading the %s schema: %w", SchemaName, err)
	}

	out := map[string]*openapi3.Schema{}

	for _, name := range []string{SchemaName, SetupSchema} {
		ref, ok := doc.Components.Schemas[name]
		if !ok {
			return nil, fmt.Errorf("the schema describes no %s", name)
		}

		out[name] = ref.Value
	}

	return out, nil
}

// against checks a document against the schema of that name.
func against(
	document any,
	name string,
) error {
	schemas, err := contracts()
	if err != nil {
		return err
	}

	if err := schemas[name].VisitJSON(document); err != nil {
		return invalid(err, name)
	}

	return nil
}

// invalid turns a schema failure into one that names the field.
//
// Both documents are written by hand, so the field is the useful half of the
// message: somebody holding a file wants the line, not the rule.
func invalid(
	err error,
	name string,
) error {
	var fault *openapi3.SchemaError
	if !errors.As(err, &fault) {
		return fmt.Errorf("%w: %s: %w", ErrInvalid, name, err)
	}

	field := fieldOf(fault)
	if field == "" {
		field = "the document"
	}

	return &InvalidError{Document: name, Field: field, Reason: reasonOf(fault)}
}

// fieldOf names what failed, the way the file writes it.
//
// A schema points at a value with a JSON pointer, where every step is a name:
// `chain.0.role`. Somebody looking at their own YAML sees a list, so the steps
// that are positions are written as ones: `chain[0].role`.
//
// The same rule `rig` applies to the gear half, deliberately. One document is
// checked by two packages, and an error from one that spelled a position
// differently would read as a different kind of fault rather than the same one
// further down the same file.
func fieldOf(
	err *openapi3.SchemaError,
) string {
	var out strings.Builder

	for _, step := range err.JSONPointer() {
		if _, err := strconv.Atoi(step); err == nil {
			fmt.Fprintf(&out, "[%s]", step)

			continue
		}

		if out.Len() > 0 {
			out.WriteString(".")
		}

		out.WriteString(step)
	}

	return out.String()
}

// reasonOf says what was wrong with it, without repeating the field.
//
// The fallback is the point. `SchemaError.Reason` is empty for some failures, and
// this used to pass it through, so a document could be refused with an
// InvalidError carrying no explanation at all: a field name and a blank. The
// error's own text is worse prose than a reason and is not nothing.
func reasonOf(
	err *openapi3.SchemaError,
) string {
	if err.Reason != "" {
		return err.Reason
	}

	return err.Error()
}
