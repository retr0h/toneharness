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
	"bytes"
	"encoding/json"
	"slices"

	yamlv3 "go.yaml.in/yaml/v3"
)

// first is the order keys are written in, for the ones worth reading in an
// order rather than alphabetically.
//
// A document is read top to bottom and the top is where somebody decides
// whether this is the file they wanted, so what it is comes before what it
// says, and what somebody asked for comes before what a machine worked out.
// Alphabetical put `ask` above `id` and `built` above `rig`, which reads as
// though the computed half were the point.
//
// Inside a chain entry the same argument: the role and the gear are what a
// person is looking for, the sources are why it is believed, and the overrides
// are the exception rather than the rule.
//
// One list rather than one per level, because these names do not collide
// across levels and a table of tables would be a second thing to keep in step.
// Anything absent from here is written after everything present, in the
// alphabetical order the marshaller already produced, so a field added to the
// contract appears without this needing to know about it.
var first = []string{
	// What the document is.
	"schema", "version", "id",
	// What somebody asked for, and what answers it.
	"ask", "rig", "built",
	// Inside a chain entry.
	"role", "gear", "capture", "confidence", "substitute", "evidence",
	"controls", "settings",
	// Inside the computed half.
	"catalog", "device", "chain", "preset",
	// Inside a subject, which is read as a sentence.
	"kind", "name", "band", "era", "years",
}

// ordered renders a document with [first]'s keys in that order.
//
// Through JSON on the way, which is how everything here reads and writes YAML:
// the generated types carry `json:` tags and no others, so marshalling any
// other way writes every optional field out as null. So the types go to JSON,
// the JSON is read back as a tree, each mapping's keys are put in order, and
// the tree is emitted.
//
// A second library is not a second parser. go.yaml.in/yaml/v3 is used here to
// emit, never to decide what a field means, and nothing in this package reads a
// document through it.
func ordered(
	of any,
) ([]byte, error) {
	// Both documents hold only strings, numbers and booleans, so neither step
	// can fail on content; a failure here is a bug in the types rather than in
	// a document, and it is returned rather than hidden.
	raw, err := json.Marshal(of)
	if err != nil {
		return nil, err
	}

	var tree yamlv3.Node
	if err := yamlv3.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}

	sortNode(&tree)

	// Two spaces, because that is what every committed document already uses
	// and this library indents four by default. A whole-tree reindent would
	// have shown up as a diff on all sixty rigs saying nothing.
	var out bytes.Buffer

	enc := yamlv3.NewEncoder(&out)
	enc.SetIndent(indent)

	if err := enc.Encode(&tree); err != nil {
		return nil, err
	}

	if err := enc.Close(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// indent is how far a nested key sits in, matching what is already committed.
const indent = 2

// sortNode puts every mapping's keys in [first]'s order, depth first, and
// clears the style the tree arrived with.
//
// The style matters as much as the order: a tree read back from JSON carries
// flow style on every node, so emitting it unchanged writes the whole document
// on one line as `{"schema": "ToneSpec", ...}`. It parses and it is unreadable,
// which for a file whose point is being read by a person is the same as broken.
func sortNode(
	n *yamlv3.Node,
) {
	n.Style = 0

	for _, c := range n.Content {
		sortNode(c)
	}

	if n.Kind != yamlv3.MappingNode {
		return
	}

	// A mapping's Content is key, value, key, value, so it is sorted in pairs
	// rather than element by element.
	pairs := make([][2]*yamlv3.Node, 0, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		pairs = append(pairs, [2]*yamlv3.Node{n.Content[i], n.Content[i+1]})
	}

	slices.SortStableFunc(pairs, func(a, b [2]*yamlv3.Node) int {
		return rank(a[0].Value) - rank(b[0].Value)
	})

	n.Content = n.Content[:0]
	for _, p := range pairs {
		n.Content = append(n.Content, p[0], p[1])
	}
}

// rank is where a key sorts: its place in [first], or after everything in it.
//
// Stable beyond that, so keys this does not name keep the alphabetical order
// the marshaller gave them rather than taking an order nobody chose.
func rank(
	key string,
) int {
	if at := slices.Index(first, key); at >= 0 {
		return at
	}

	return len(first)
}
