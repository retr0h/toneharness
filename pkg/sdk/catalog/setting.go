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
package catalog

import (
	"encoding/json"
	"strings"
)

// Setting is a parameter value as a document writes it down, which is a string.
//
// A string because the kind has to survive the trip, and as a number it does
// not. A document is read with sigs.k8s.io/yaml, which routes every value
// through `interface{}`: a float of 6.0 becomes float64(6) and is written back
// as `6`, and `6` reads as an integer. A device takes 6 and 6.0 as different
// readings, so a build recorded as one compiled to the other and stopped
// matching the preset it was recorded from.
//
// Quoted, the literal is the document's own and nothing reinterprets it.
// `"6.0"` is a float, `"6"` an integer, `"true"` a switch and `"Normal"` a
// named position, which is the same rule [ParamValue] applies to JSON and for
// the same reason.
//
// The cost is a pair of quotes in a file somebody edits by hand. Worth it: the
// alternative is a file that reads correctly and builds something else.
type Setting struct {
	ParamValue
}

// Set is a parameter value ready to write down.
func Set(
	of ParamValue,
) Setting {
	return Setting{ParamValue: of}
}

// MarshalJSON writes the value as the string its own kind spells.
func (s Setting) MarshalJSON() ([]byte, error) {
	raw, err := s.ParamValue.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// An enum arrives already quoted and a number does not, so the quotes come
	// off before the whole thing goes back on as one string.
	return json.Marshal(strings.Trim(string(raw), `"`))
}

// UnmarshalJSON reads the value back from the literal the document spells.
//
// A bare number or boolean is read too, rather than refused. Somebody writing a
// build by hand should not have their file rejected over quotes; what they lose
// by leaving them off is only the promise that a whole-numbered float stays one.
func (s *Setting) UnmarshalJSON(
	b []byte,
) error {
	var text string
	if err := json.Unmarshal(b, &text); err != nil {
		return s.ParamValue.UnmarshalJSON(b)
	}

	return s.ParamValue.UnmarshalJSON(literal(text))
}

// literal is the JSON a string stands for.
//
// A number or a boolean stands for itself; anything else is a named position and
// needs its quotes back before [ParamValue] will read it as one.
func literal(
	text string,
) []byte {
	switch {
	case text == "true", text == "false":
		return []byte(text)
	case numeric(text):
		return []byte(text)
	default:
		quoted, _ := json.Marshal(text)

		return quoted
	}
}

// numeric says whether a string is a JSON number, which decides whether it is a
// reading or a name.
func numeric(
	text string,
) bool {
	if text == "" {
		return false
	}

	var held json.Number

	return json.Unmarshal([]byte(text), &held) == nil
}
