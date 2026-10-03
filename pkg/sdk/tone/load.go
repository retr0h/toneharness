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

package tone

import (
	"encoding/json"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

// Load reads a request and checks it against its own contract.
//
// YAML, because both documents are written and corrected by hand and JSON is
// a poor format to argue with. The schema is JSON Schema either way;
// sigs.k8s.io/yaml converts, so the generated types need no second set of
// tags.
func Load(
	r io.Reader,
) (Spec, error) {
	var spec Spec

	return spec, read(r, SchemaName, &spec)
}

// LoadSetup reads what somebody has and checks it against its own contract.
func LoadSetup(
	r io.Reader,
) (Setup, error) {
	var setup Setup

	return setup, read(r, SetupSchema, &setup)
}

// read is Load and LoadSetup, which differ only in what they are reading.
//
// Sharing it rather than writing it twice, because the interesting part is
// the order the steps happen in and two copies of that order is two chances
// to get it wrong in one of them.
func read(
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
	// the request would pass, and the line somebody wrote would be gone with
	// nothing said.
	body, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return fmt.Errorf("decoding the %s: %w", name, err)
	}

	// What YAMLToJSON produced is JSON, so reading it back cannot fail.
	var document any
	_ = json.Unmarshal(body, &document)

	// Which document this is, before it is held to one.
	//
	// Both carry `schema`, so a Setup handed to Load can be told apart from a
	// Setup that is wrong, and the two are worth different messages: one is a
	// mistake about which file to pass and the other is a mistake in the
	// file.
	if err := is(document, name); err != nil {
		return err
	}

	if err := against(document, name); err != nil {
		return err
	}

	// Checked rather than assumed. A document can satisfy the contract and
	// still not fit the types: JSON Schema calls 99999999999999999999 an
	// integer and Go's int cannot hold it, which returns a document with the
	// field silently zeroed and no error at all.
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("decoding the %s: %w", name, err)
	}

	return nil
}

// is reports a document handed to the wrong reader.
//
// The schema would refuse it anyway, on the enum, and say that `schema` is
// not one of its allowed values. That is true and unhelpful: somebody who
// passed their Setup where the ask goes wants to be told that, not to be told
// about an enum.
func is(
	document any,
	name string,
) error {
	fields, ok := document.(map[string]any)
	if !ok {
		return &InvalidError{
			Document: name,
			Field:    "the document",
			Reason:   "is not a set of fields",
		}
	}

	said, ok := fields["schema"].(string)
	if !ok || said == name {
		return nil
	}

	return &InvalidError{
		Document: name,
		Field:    "schema",
		Reason:   fmt.Sprintf("says %s, so this is not a %s", said, name),
	}
}

// Write renders a request.
//
// Validated first: writing one that does not meet its own contract would put
// a file into the world that nothing else will accept.
func Write(
	w io.Writer,
	spec Spec,
) error {
	return write(w, SchemaName, spec, Validate(spec))
}

// WriteSetup renders what somebody has.
func WriteSetup(
	w io.Writer,
	setup Setup,
) error {
	return write(w, SetupSchema, setup, ValidateSetup(setup))
}

// write is Write and WriteSetup, with the check already made by the caller
// because only the caller knows which of the two to make.
func write(
	w io.Writer,
	name string,
	of any,
	err error,
) error {
	if err != nil {
		return err
	}

	// sigs.k8s.io/yaml, which marshals through JSON, so the `json:omitempty` the
	// generated types carry is honoured and an absent field is absent rather than
	// `null`.
	//
	// That sorts keys, and a document comes out ask, id, rig, schema. Field order
	// was tried and is not available: go.yaml.in/yaml/v3 marshals in declaration
	// order but reads `yaml:` tags, which these types do not have, so every
	// optional field was written out as null. Giving them a second set of tags
	// means teaching the generator to, which is a change to make on purpose
	// rather than on the way past.
	//
	// Both documents hold only strings, numbers and booleans, so they always
	// encode.
	raw, _ := yaml.Marshal(of)

	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("writing the %s: %w", name, err)
	}

	return nil
}
