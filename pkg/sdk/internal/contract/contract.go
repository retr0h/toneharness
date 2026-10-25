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

// Package contract checks a hand-written document against the OpenAPI schema
// its Go types were generated from.
//
// **Shared machinery, not a shared contract.** A ToneSpec and a Config are
// separate documents with separate versions: one is published to a
// marketplace and read by strangers, the other never leaves the machine it is
// on, and neither refers to the other. What they have in common is that both
// are typed by hand and both are generated from a schema, so both want the
// same checking and the same shape of error.
//
// Extracted when the two were split, because the alternative was a thousand
// lines of validator copied into each. Two contracts sharing a checker is the
// same arrangement as two contracts sharing a YAML library.
package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"sigs.k8s.io/yaml"
)

// ErrInvalid reports a document that does not meet its own contract.
var ErrInvalid = errors.New("not valid")

// InvalidError says which part of a document is wrong, and which document.
type InvalidError struct {
	// Document is what the schema is called, so a message says which file to
	// look in when somebody is holding more than one.
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

// Checker holds one OpenAPI document and the schemas in it worth checking
// against.
//
// Parsing an OpenAPI document is not cheap and an embedded one never changes,
// so it happens on the first document checked and not again.
type Checker struct {
	raw    []byte
	names  []string
	parsed func() (map[string]*openapi3.Schema, error)
}

// New returns a Checker over one contract, for the schemas named.
//
// The names are given rather than discovered because a contract holds more
// schemas than it has documents: everything a document is built out of is in
// there too, and only the ones a file may be is worth checking against.
func New(
	raw []byte,
	names ...string,
) *Checker {
	c := &Checker{raw: raw, names: names}
	c.parsed = sync.OnceValues(func() (map[string]*openapi3.Schema, error) {
		return c.load()
	})

	return c
}

// Read decodes a document and checks it, in that order's opposite.
//
// YAML, because these documents are written and corrected by hand and JSON is
// a poor format to argue with. The schema is JSON Schema either way;
// sigs.k8s.io/yaml converts, so the generated types need no second set of
// tags.
func (c *Checker) Read(
	r io.Reader,
	name string,
	into any,
) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading the %s: %w", name, err)
	}

	// The file as it was written, checked before anything decodes it.
	//
	// Decoding drops whatever the generated types have no field for, so a
	// document checked after decoding is checked with its own mistakes
	// already removed: `gnere:` for `genre:` would be dropped on the way in,
	// the document would pass, and the line somebody wrote would be gone with
	// nothing said.
	body, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return fmt.Errorf("decoding the %s: %w", name, err)
	}

	var document any
	if err := json.Unmarshal(body, &document); err != nil {
		return fmt.Errorf("decoding the %s: %w", name, err)
	}

	if err := c.Against(document, name); err != nil {
		return err
	}

	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("reading the %s: %w", name, err)
	}

	return nil
}

// Check marshals a document and holds it to the schema of that name.
//
// For a document built in memory rather than read off disk, which has never
// been through [Checker.Read] and so has never been checked.
func (c *Checker) Check(
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

	return c.Against(document, name)
}

// Against checks a decoded document against the schema of that name.
func (c *Checker) Against(
	document any,
	name string,
) error {
	schemas, err := c.parsed()
	if err != nil {
		return err
	}

	held, ok := schemas[name]
	if !ok {
		return fmt.Errorf("the schema describes no %s", name)
	}

	if err := held.VisitJSON(document); err != nil {
		return invalid(err, name)
	}

	return nil
}

// Version is what the contract says its own version is.
//
// Read out of the document rather than written beside it, so there is one
// place to change and no way for the two to disagree.
func (c *Checker) Version() (string, error) {
	var doc struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}

	if err := yaml.Unmarshal(c.raw, &doc); err != nil {
		return "", fmt.Errorf("reading the schema's version: %w", err)
	}

	if doc.Info.Version == "" {
		return "", errors.New("the schema states no version")
	}

	return doc.Info.Version, nil
}

// load reads the named schemas out of the OpenAPI document.
func (c *Checker) load() (map[string]*openapi3.Schema, error) {
	doc, err := openapi3.NewLoader().LoadFromData(c.raw)
	if err != nil {
		return nil, fmt.Errorf("reading the schema: %w", err)
	}

	out := map[string]*openapi3.Schema{}

	for _, name := range c.names {
		ref, ok := doc.Components.Schemas[name]
		if !ok {
			return nil, fmt.Errorf("the schema describes no %s", name)
		}

		out[name] = ref.Value
	}

	return out, nil
}

// invalid turns a schema failure into one that names the field.
//
// These documents are written by hand, so the field is the useful half of the
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
// The fallback is the point. `SchemaError.Reason` is empty for some failures,
// and passing it through refuses a document with no explanation at all: a
// field name and a blank. The error's own text is worse prose than a reason
// and is not nothing.
func reasonOf(
	err *openapi3.SchemaError,
) string {
	if err.Reason != "" {
		return err.Reason
	}

	return err.Error()
}
