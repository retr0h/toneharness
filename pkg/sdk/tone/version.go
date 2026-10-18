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
	"fmt"
	"strings"
	"sync"

	"sigs.k8s.io/yaml"
)

// Version is the contract this package reads documents against, as the
// contract's own `info.version` states it.
//
// Read out of the embedded contract rather than written here, so there is one
// place to change and no way for the two to disagree. Parsed on the first
// document checked and not again.
var Version = sync.OnceValues(func() (string, error) {
	return versionOf(Schema)
})

// versionOf reads what a contract says its version is.
//
// Takes the bytes rather than reaching for the embedded contract, because what
// it does when there is no version to find is part of the contract and cannot
// be reached through the one this binary carries. Both answers describe a
// binary built without it.
func versionOf(
	raw []byte,
) (string, error) {
	var doc struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}

	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("reading the %s schema's version: %w", SchemaName, err)
	}

	if doc.Info.Version == "" {
		return "", fmt.Errorf("the %s schema states no version", SchemaName)
	}

	return doc.Info.Version, nil
}

// major is the first part of a semantic version.
//
// Whatever is in front of the first dot, because the pattern on the field has
// already held the whole string to major.minor.patch by the time anything
// calls this.
func major(
	version string,
) string {
	at := strings.Index(version, ".")
	if at < 0 {
		return version
	}

	return version[:at]
}

// sameContract refuses a document written against a different major version
// of the contract.
//
// The major is the whole warning. It moves when something that used to
// validate stops, so a document carrying another one was written against
// rules this reader does not have, and reading it anyway means guessing which
// of its fields still mean what they say. A later minor or patch is fine:
// everything in those either adds a field this document does not use or
// rewords a sentence.
//
// Checked after the schema, so the field is known to be present and to look
// like a version.
func sameContract(
	document any,
	name string,
) error {
	mine, err := Version()
	if err != nil {
		return err
	}

	fields, ok := document.(map[string]any)
	if !ok {
		return nil
	}

	theirs, ok := fields["version"].(string)
	if !ok {
		return nil
	}

	if major(theirs) == major(mine) {
		return nil
	}

	return &InvalidError{
		Document: name,
		Field:    "version",
		Reason: fmt.Sprintf(
			"is %q and this reads %q. A different major means something that "+
				"used to be valid no longer is, so what the fields mean cannot "+
				"be assumed: rewrite the document against %s rather than "+
				"changing the number",
			theirs, mine, mine),
	}
}
