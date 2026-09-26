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

package sweepdoc

import (
	"strings"
	"unicode"
)

// width is where prose wraps, which is what mdformat uses on every other page.
const width = 80

// wrap reflows a rendered page's prose to one width.
//
// The generator has to do this itself. A template's line breaks are fixed and
// the values substituted into it are not, so a sentence carrying a five-digit
// number wraps differently from the same sentence carrying a three-digit one,
// and every figure that changes leaves the page ragged somewhere else.
//
// mdformat would do it, and does for every page a person writes. It cannot do
// it here: a generated page is compared against what the generator produced,
// so anything reformatting it afterwards makes the two disagree. So the
// generator produces what the formatter would have.
//
// Left alone: fenced code, tables, headings, list items and anything indented,
// because each of those means its line breaks.
func wrap(
	page string,
) string {
	var (
		out   []string
		para  []string
		fence bool
	)

	flush := func() {
		if len(para) > 0 {
			out = append(out, fill(strings.Join(para, " "))...)
			para = nil
		}
	}

	for _, line := range strings.Split(page, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			flush()

			fence = !fence
			out = append(out, line)

			continue
		}

		switch {
		case fence:
			out = append(out, line)
		case trimmed == "":
			flush()

			out = append(out, "")
		case strings.HasPrefix(trimmed, "#"),
			strings.HasPrefix(trimmed, "|"),
			strings.HasPrefix(trimmed, "-"),
			strings.HasPrefix(trimmed, "<!--"),
			line != trimmed:
			flush()

			out = append(out, line)
		default:
			para = append(para, trimmed)
		}
	}

	flush()

	return strings.Join(out, "\n")
}

// fill breaks one paragraph into lines no wider than width.
//
// Never inside backticks or a link's brackets, because a break there is a
// different document: `presets play` split across a newline is two words and a
// [name](target) split after the bracket stops being a link.
func fill(
	para string,
) []string {
	var (
		lines []string
		at    strings.Builder
	)

	for _, word := range atoms(para) {
		switch {
		case at.Len() == 0:
			at.WriteString(word)
		case at.Len()+1+len(word) <= width:
			at.WriteString(" " + word)
		default:
			lines = append(lines, at.String())
			at.Reset()
			at.WriteString(word)
		}
	}

	if at.Len() > 0 {
		lines = append(lines, at.String())
	}

	return lines
}

// atoms splits a paragraph into the pieces a line break may fall between.
//
// A run inside backticks, or a whole [text](target), counts as one piece
// however many spaces it holds.
func atoms(
	para string,
) []string {
	var (
		out  []string
		at   strings.Builder
		tick bool
		link int
	)

	done := func() {
		if at.Len() > 0 {
			out = append(out, at.String())
			at.Reset()
		}
	}

	for _, r := range para {
		switch {
		case r == '`':
			tick = !tick
		case r == '[' && !tick:
			link++
		case r == ')' && link > 0 && !tick:
			link--
		}

		if unicode.IsSpace(r) && !tick && link == 0 {
			done()

			continue
		}

		at.WriteRune(r)
	}

	done()

	return out
}
