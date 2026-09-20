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
	"encoding/json"
	"fmt"
	"io"
)

// Data writes what an operation answered, as data rather than as a table.
//
// Here rather than in cmd for the same reason the painted half is here: how
// output looks is this package's job, and a caller should not have to know
// which of the two it is getting.
//
// The whole answer, not a summary of it. A table leaves things out on purpose
// — a chain drawn in box characters cannot carry every parameter of every
// block — and that is right for somebody reading it and wrong for anything
// that has to act on it.
//
// Indented, and with a trailing newline, because the usual reader is a person
// checking what an agent will see, and the usual next thing is a pipe.
func Data(
	w io.Writer,
	of any,
) error {
	body, err := json.MarshalIndent(of, "", "  ")
	if err != nil {
		return fmt.Errorf("writing the answer: %w", err)
	}

	if _, err := w.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("writing the answer: %w", err)
	}

	return nil
}
