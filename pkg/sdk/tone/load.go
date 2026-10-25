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

// read decodes a document and checks it.
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

	// After the schema, which has already said the version is there and looks
	// like one, and before the decode, because a document written against
	// another major is one whose fields cannot be trusted to mean what this
	// reader's types say they mean.
	if err := sameContract(document, name); err != nil {
		return err
	}

	// Checked rather than assumed. A document can satisfy the contract and
	// still not fit the types: JSON Schema calls 99999999999999999999 an
	// integer and Go's int cannot hold it, which returns a document with the
	// field silently zeroed and no error at all.
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("decoding the %s: %w", name, err)
	}

	// After the decode, because this one compares two fields against each other
	// and the schema can only speak about one at a time. A document read and
	// never checked for it would build something nobody asked for.
	if spec, ok := into.(*Spec); ok {
		return OneSetOfSnapshots(spec.Rig)
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
//
// The contract version is stamped rather than asked for. Whoever built this
// document built it against the contract this binary carries, which is the
// only version they could have built it against, so making every caller say
// so would be making them repeat the one answer available.
func Write(
	w io.Writer,
	spec Spec,
) error {
	stamp(&spec.Version)

	return write(w, SchemaName, spec, Validate(spec))
}

// stamp fills in the contract version where a document does not carry one.
//
// A version already there is left alone: a document read and written back is
// the same document, and silently moving its version would lose the one fact
// that says which rules it was written under.
func stamp(
	version *ContractVersion,
) {
	if *version != "" {
		return
	}

	// Nothing to do on an error, and nothing is hidden by it: the document
	// goes out without the field and whatever reads it next refuses it.
	if mine, err := Version(); err == nil {
		*version = mine
	}
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

	// Through JSON either way, so the `json:omitempty` the generated types carry
	// is honoured and an absent field is absent rather than `null`.
	//
	// [ordered] rather than a plain marshal, because a plain one sorts keys and
	// a document came out ask, built, id, rig, schema: the computed half above
	// the authored one, and what the file is below both. Declaration order was
	// tried first and is not available, since go.yaml.in/yaml/v3 marshals in
	// that order but reads `yaml:` tags these types do not have, and the
	// generated order is alphabetical anyway.
	raw, err := ordered(of)
	if err != nil {
		return fmt.Errorf("rendering the %s: %w", name, err)
	}

	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("writing the %s: %w", name, err)
	}

	return nil
}
