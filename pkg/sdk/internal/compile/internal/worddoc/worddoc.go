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

// Package worddoc writes the vocabulary page from the vocabulary.
//
// The words are the thing a person writing an ask most needs and the one thing
// no page listed. They lived in a JSON file inside an internal package, so the
// only complete answer was to read Go, while four prose pages each described
// some of them and none defined them.
//
// Generated for the reason the other reference pages are: a hand-written list
// goes stale the first time somebody adds a word, and nothing notices.
package worddoc

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
)

//go:embed page.md.tmpl
var pages embed.FS

// page is the template, parsed once.
//
// Parsed at startup rather than per call, because it is embedded and written
// here: a parse failure is a broken build, not a thing a caller can be handed
// and asked to do something about. Returning an error nothing can provoke is a
// claim nobody can check.
var page = template.Must(template.New("page.md.tmpl").
	Funcs(template.FuncMap{"wrap": wrap}).
	ParseFS(pages, "page.md.tmpl"))

// Word is one word as the page reports it.
type Word struct {
	Term  string
	Means string
	// Moves is what it does to a control, as a phrase, or empty.
	Moves string
}

// Axis is one question and the words that answer it.
type Axis struct {
	Name  string
	About string
	Words []Word
	// Acts says whether answering this axis moves anything.
	Acts bool
}

// Page is everything the template draws.
type Page struct {
	Axes []Axis
	// Acting and Describing are how many axes do each, so the prose can state
	// the split without a number anybody has to keep in step by hand.
	Acting     int
	Describing int
	Words      int
}

// Render writes the vocabulary out as markdown.
func Render() ([]byte, error) {
	return render(page)
}

// render draws one template, so a test can hand it one that fails.
func render(
	tmpl *template.Template,
) ([]byte, error) {
	out := Page{}

	for _, axis := range compile.Vocabulary() {
		one := Axis{Name: axis.Name, About: axis.About}

		for _, w := range axis.Words {
			one.Words = append(one.Words, Word{
				Term: w.Term, Means: w.Means, Moves: moves(w),
			})

			if w.Param != "" {
				one.Acts = true
			}

			out.Words++
		}

		if one.Acts {
			out.Acting++
		} else {
			out.Describing++
		}

		out.Axes = append(out.Axes, one)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, out); err != nil {
		return nil, fmt.Errorf("writing the vocabulary page: %w", err)
	}

	return buf.Bytes(), nil
}

// moves says what a word does to a control, or that it does nothing.
//
// The direction in words rather than the number, because the number is a share
// of a step the corpus decides and means nothing without that context. Which
// way it goes is the part a reader can act on.
func moves(
	w compile.Defined,
) string {
	if w.Param == "" {
		return ""
	}

	way := "up"
	if w.Steps < 0 {
		way = "down"
	}

	if w.Steps == 0.5 || w.Steps == -0.5 {
		way = "half a step " + way
	}

	return fmt.Sprintf("%s %s on the %s", w.Param, way, w.Block)
}

// wrap breaks prose at 80 columns.
//
// The page's own job, because the formatter is told to leave generated pages
// alone and would otherwise reflow this one away from what the generator emits.
func wrap(
	at int,
	text string,
) string {
	var (
		out  strings.Builder
		line int
	)

	for i, word := range strings.Fields(text) {
		switch {
		case i == 0:
			line = len(word)
		case line+1+len(word) > at:
			out.WriteString("\n")

			line = len(word)
		default:
			out.WriteString(" ")

			line += 1 + len(word)
		}

		out.WriteString(word)
	}

	return out.String()
}
