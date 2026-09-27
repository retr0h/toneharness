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

// Package toolsdoc writes the MCP reference from the tools the server
// registers.
//
// Nothing documented them. Which tools exist, what each takes and which of
// them write lived in one Go file, and the set changed twice in a month while
// the only sentence about it anywhere was a row in the README. A page written
// by hand would have fallen behind the same way, so this one is generated the
// way docs/commands.md is generated from the cobra tree.
//
// The tools are read back the way an agent reads them, over a session rather
// than out of the registration: the server has no exported list, and what an
// agent is handed has been through JSON both ways. So a schema this page
// reports wrongly is a schema an agent is handed wrongly.
package toolsdoc

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/google/jsonschema-go/jsonschema"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
)

//go:embed page.md.tmpl
var pages embed.FS

// tmpl is the template, parsed once. A template embedded in the binary that
// will not parse is a programming error, found the moment this package is
// loaded rather than handed back as an error nobody can do anything about.
var tmpl = template.Must(template.New("page.md.tmpl").Funcs(template.FuncMap{
	"fill":  fill,
	"yesno": yesno,
}).ParseFS(pages, "page.md.tmpl"))

// The labels the page reports under "writes", each derived from a tool's own
// annotations and from whether the server offers it without --allow-writes.
const (
	// writesNothing is a tool marked read-only, which changes nothing
	// anywhere.
	writesNothing = "no"
	// writesFile is a tool that writes a file and no slot: it is offered
	// without the flag, and the flag decides only whether it may write over a
	// file already there.
	writesFile = "a file"
	// writesPedal is a tool the server withholds unless it was started with
	// --allow-writes, which is every tool that overwrites what a slot holds.
	writesPedal = "the pedal"
	// writesLoaded is what is left: a tool that changes which preset the pedal
	// has loaded and nothing stored.
	writesLoaded = "nothing stored"
)

// Field is one property of a tool's input or output, as the page reports it.
type Field struct {
	// Name is the JSON key, which is what an agent sends or reads.
	Name string
	// Type is the JSON type in a word.
	Type string
	// Required is whether an input has to carry it, or an answer always does.
	Required bool
	// What is the schema's own description. Inputs carry one, from the
	// jsonschema tag on the field; an inferred output schema does not.
	What string
}

// Tool is one registered tool, as the page reports it.
type Tool struct {
	// Name is what an agent calls.
	Name string
	// Description is the tool's own description, which is what the model
	// reads before deciding to call it.
	Description string
	// Sentence is the first sentence of that, for the summary table.
	Sentence string
	// Writes is one of the four labels above.
	Writes string
	// Gated is whether the tool is offered only with --allow-writes.
	Gated bool
	// Pedal is whether the tool reaches the hardware, which is its
	// openWorldHint: a tool whose world is closed answers from what ships.
	Pedal bool
	// Takes and Answers are the top level of its two schemas.
	Takes, Answers []Field
}

// Page is what the reference says about itself.
type Page struct {
	// Tools is every tool the server registers with --allow-writes, which is
	// all of them, in the order an agent is listed them.
	Tools []Tool
	// Always is how many are offered without the flag.
	Always int
	// Gated are the ones the flag adds, and Files the ones it does not add
	// but still decides for: they write a file, and without it they refuse a
	// path that already holds one.
	Gated, Files []Tool
	// Offline are the tools that answer with no pedal attached, and Pedal
	// the ones that reach it over USB.
	Offline, Pedal []Tool
}

// Render writes the page from the tools the server registers.
//
// The tools are listed twice, because nothing on a tool says the flag added it.
// The set offered without --allow-writes subtracted from the set offered with
// it is what the flag is for, and that is the only place the answer lives.
func Render(
	ctx context.Context,
) ([]byte, error) {
	return rendered(ctx, listed)
}

// lister is what answers "which tools does a server offer", so a test can hand
// back a failure or an empty set. The real one stands a server up in memory.
type lister func(ctx context.Context, allowWrites bool) ([]*gomcp.Tool, error)

// rendered builds the page from whatever lists the tools.
func rendered(
	ctx context.Context,
	list lister,
) ([]byte, error) {
	held, err := list(ctx, false)
	if err != nil {
		return nil, err
	}

	all, err := list(ctx, true)
	if err != nil {
		return nil, err
	}

	page, err := read(held, all)
	if err != nil {
		return nil, err
	}

	return render(page)
}

// render puts a page through the template.
func render(
	page Page,
) ([]byte, error) {
	return drawn(tmpl, page)
}

// drawn puts a page through one template, so a test can hand it one that fails.
//
// The template this ships is embedded and parsed at startup, so nothing a caller
// does provokes the error below. Reaching it any other way would leave a claim
// nobody can check.
func drawn(
	with *template.Template,
	page Page,
) ([]byte, error) {
	var out bytes.Buffer
	if err := with.Execute(&out, page); err != nil {
		return nil, fmt.Errorf("writing the MCP page: %w", err)
	}

	return out.Bytes(), nil
}

// read turns the listed tools into what the page needs.
//
// held is what an agent is offered by a server started without
// --allow-writes, and all is what one started with it offers.
func read(
	held, all []*gomcp.Tool,
) (Page, error) {
	if len(all) == 0 {
		return Page{}, fmt.Errorf("the server registers no tools")
	}

	without := make(map[string]bool, len(held))
	for _, t := range held {
		without[t.Name] = true
	}

	page := Page{Always: len(held)}

	for _, t := range all {
		one, err := toolOf(t, !without[t.Name])
		if err != nil {
			return Page{}, err
		}

		page.Tools = append(page.Tools, one)

		if one.Gated {
			page.Gated = append(page.Gated, one)
		}

		if one.Writes == writesFile {
			page.Files = append(page.Files, one)
		}

		if one.Pedal {
			page.Pedal = append(page.Pedal, one)
		} else {
			page.Offline = append(page.Offline, one)
		}
	}

	return page, nil
}

// listed is every tool an agent connecting to the server is offered.
//
// Over the library's in-memory transport, because the server exports no list of
// its tools. A session is also what every test here asks through, so the page
// reports what an agent is handed rather than what the registration meant. The
// iterator rather than one call, so a page size the library picks cannot
// silently truncate the reference.
//
// The tools are registered against a nil Client. Listing calls no handler, so
// there is nothing for a client to answer, and a page generated on a machine
// with no pedal is the same page.
func listed(
	ctx context.Context,
	allowWrites bool,
) ([]*gomcp.Tool, error) {
	server := gomcp.NewServer(
		&gomcp.Implementation{Name: "tonestack", Version: "docgen"}, nil)

	pedal := tools.Register(server, nil, allowWrites)
	defer func() { _ = pedal.Close() }()

	serverEnd, clientEnd := gomcp.NewInMemoryTransports()

	if _, err := server.Connect(ctx, serverEnd, nil); err != nil {
		return nil, fmt.Errorf("serving the tools: %w", err)
	}

	session, err := gomcp.NewClient(
		&gomcp.Implementation{Name: "toolsdoc", Version: "docgen"}, nil,
	).Connect(ctx, clientEnd, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to the tools: %w", err)
	}

	defer func() { _ = session.Close() }()

	var out []*gomcp.Tool

	for t, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, fmt.Errorf("listing the tools: %w", err)
		}

		out = append(out, t)
	}

	return out, nil
}

// toolOf is one tool as the page reports it.
func toolOf(
	t *gomcp.Tool,
	gated bool,
) (Tool, error) {
	in, err := schemaOf(t.InputSchema)
	if err != nil {
		return Tool{}, fmt.Errorf("%s: reading its input schema: %w", t.Name, err)
	}

	out, err := schemaOf(t.OutputSchema)
	if err != nil {
		return Tool{}, fmt.Errorf("%s: reading its output schema: %w", t.Name, err)
	}

	return Tool{
		Name:        t.Name,
		Description: t.Description,
		Sentence:    sentence(t.Description),
		Writes:      writes(t.Annotations, gated),
		Gated:       gated,
		Pedal:       openWorld(t.Annotations),
		Takes:       fields(in),
		Answers:     fields(out),
	}, nil
}

// writes says what calling a tool can change.
//
// The gate comes first. A server that withholds a tool has said more about it
// than any annotation on it could.
func writes(
	a *gomcp.ToolAnnotations,
	gated bool,
) string {
	switch {
	case gated:
		return writesPedal
	case a == nil:
		// Nothing was said, so nothing is claimed. The protocol's own default
		// for a tool that says nothing is that it may destroy.
		return writesPedal
	case a.ReadOnlyHint:
		return writesNothing
	case a.DestructiveHint != nil && *a.DestructiveHint:
		return writesFile
	default:
		return writesLoaded
	}
}

// openWorld is whether a tool reaches something outside this machine's own
// files, which for every tool here is the pedal on the USB bus.
//
// The protocol's default is an open world, so a tool saying nothing is
// reported as reaching the pedal rather than as answering offline. That is the
// safe way round: a new tool nobody annotated shows up where somebody will
// notice it.
func openWorld(
	a *gomcp.ToolAnnotations,
) bool {
	return a == nil || a.OpenWorldHint == nil || *a.OpenWorldHint
}

// sentence is the first sentence of a description, for the summary table.
//
// A description is written for a model deciding whether to call the tool, so
// it carries the warnings too, and the whole of it in a table cell is a table
// nobody can read. The rest is under the tool's own heading.
func sentence(
	of string,
) string {
	at := strings.Index(of, ". ")
	if at < 0 {
		return strings.TrimSuffix(strings.TrimSpace(of), ".")
	}

	return of[:at]
}

// schemaOf reads a schema back off a tool.
//
// A listed tool carries its schemas as whatever JSON unmarshalling produced,
// which is a map. Marshalling that and reading it as a schema is how the
// library's own validation gets at it, and it is the same document the agent
// holds.
func schemaOf(
	of any,
) (*jsonschema.Schema, error) {
	if of == nil {
		return nil, nil
	}

	body, err := json.Marshal(of)
	if err != nil {
		return nil, err
	}

	var s jsonschema.Schema
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// fields is the top level of a schema, in name order.
//
// The top level only. An answer's full schema runs to hundreds of lines
// because a rig travels inside it, and a page holding that is a page nobody
// reads; the shape of a rig has a page of its own. Every agent is handed the
// whole schema anyway, generated from the same Go type.
//
// Name order because a JSON object has none. The order a Go struct declares
// its fields in does not survive being marshalled, so alphabetical is the only
// order that is the same on the next run.
func fields(
	of *jsonschema.Schema,
) []Field {
	if of == nil {
		return nil
	}

	required := make(map[string]bool, len(of.Required))
	for _, name := range of.Required {
		required[name] = true
	}

	out := make([]Field, 0, len(of.Properties))
	for name, p := range of.Properties {
		out = append(out, Field{
			Name:     name,
			Type:     typeOf(p),
			Required: required[name],
			What:     cell(p.Description),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out
}

// typeOf names a property's JSON type in a word.
//
// Null is dropped. Every optional field in these answers is a pointer, so
// inference adds null to each of them, and "null or string" says the same
// thing as the column beside it already says. A field that may be any JSON
// value is reported as any rather than as five words.
func typeOf(
	of *jsonschema.Schema,
) string {
	if of == nil {
		return "any"
	}

	if of.Items != nil {
		return "array of " + typeOf(of.Items)
	}

	named := of.Types
	if of.Type != "" {
		named = []string{of.Type}
	}

	var kinds []string

	for _, k := range named {
		if k != "null" {
			kinds = append(kinds, k)
		}
	}

	if len(kinds) != 1 {
		return "any"
	}

	return kinds[0]
}

// cell makes a string safe to put in a table cell, where a pipe would end the
// column.
func cell(
	of string,
) string {
	return strings.ReplaceAll(strings.TrimSpace(of), "|", `\|`)
}

// yesno is a boolean as a table reads it.
func yesno(
	of bool,
) string {
	if of {
		return "yes"
	}

	return "no"
}

// fill breaks a description into lines no wider than the width mdformat uses.
//
// The generator has to do it. A generated page is compared against what the
// generator produced, so the formatter is not allowed to touch it, and a
// description substituted into a template arrives as one long line. Only a
// description: every other value on this page sits in a table cell or a fenced
// block, where a line break would mean something.
func fill(
	para string,
) string {
	// 80 columns, which is what mdformat wraps every page a person writes to.
	const width = 80

	var (
		lines []string
		at    strings.Builder
	)

	for _, word := range strings.Fields(para) {
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

	return strings.Join(lines, "\n")
}
