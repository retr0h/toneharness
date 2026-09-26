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

// Package sweepdoc writes the measurements page from the measurements.
//
// Every figure on that page used to be typed in by hand, copied out of a run.
// The first re-measurement made all of them wrong while the page still read as
// authoritative: it claimed the Bass control moves the centre of gravity by
// 13,649Hz when the answer is 9,405, and there was nothing to notice by.
//
// So the numbers come from the files and the prose stays written. What a figure
// means is a judgement somebody made; what it is belongs to the data. That is
// the same split docs/rigspec.md and docs/tonespec.md are generated on.
//
// The claims the prose makes about shape are checked rather than asserted. A
// re-measurement that turned the tone stack the other way round would leave a
// sentence here saying otherwise, so the test beside this holds each one.
package sweepdoc

import (
	"bytes"
	"embed"
	"fmt"
	"math"
	"sort"
	"strings"
	"text/template"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
)

//go:embed page.md.tmpl
var pages embed.FS

// Control is one swept control, as the page reports it.
type Control struct {
	// Name is what the catalog calls it.
	Name string
	// Index is its position in the model's own list, which is the only thing
	// that identifies it on the wire.
	Index int
	// Centroid, Level, Low and High are the per-turn slopes, and Straight how
	// much of the movement a line accounts for.
	Centroid, Level, Low, High float64
	Straight                   float64
	// Muted is where the control silenced the chain, if anywhere.
	Muted []float64
	// Aimable is whether any figure moved further than its own noise floor.
	Aimable bool
}

// Page is what the measurements page says about itself.
type Page struct {
	// Blocks is how many the fingerprint library holds, and Amps how many of
	// them are amplifiers.
	Blocks, Amps int
	// Reference identifies the signal every reading was taken against.
	Reference measured.Reference
	// Gear is the block whose controls were swept, and Model its identifier.
	Gear, Model string
	// Controls are its controls, in wire order.
	Controls []Control
	// Widest is the control that moves the centre of gravity most, and
	// Opposite the one that moves it most the other way. Together they are
	// the tone stack, and the page says so.
	Widest, Opposite Control
	// Loudest is the control that moves level most, which is the volume.
	Loudest Control
	// Inert are the controls nothing can aim, because every figure moved
	// less than the rig's own wander.
	Inert []Control
	// Crooked are the controls no single slope describes.
	Crooked []Control
	// Figures is every figure a reading is reported in.
	Figures []audio.Figure
}

// Render writes the page from what was measured.
func Render(
	lib measured.Library,
	curves measured.Curves,
) ([]byte, error) {
	page, err := read(lib, curves)
	if err != nil {
		return nil, err
	}

	t, err := template.New("page.md.tmpl").Funcs(template.FuncMap{
		"hz":     hz,
		"db":     func(v float64) string { return fmt.Sprintf("%+.1f", v) },
		"pct":    func(v float64) string { return fmt.Sprintf("%+.1f", v) },
		"fixed2": func(v float64) string { return fmt.Sprintf("%.2f", v) },
		"list":   list,
	}).ParseFS(pages, "page.md.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading the page template: %w", err)
	}

	var out bytes.Buffer
	if err := t.Execute(&out, page); err != nil {
		return nil, fmt.Errorf("writing the measurements page: %w", err)
	}

	return []byte(wrap(out.String())), nil
}

// list names a set of controls in prose, joined by sep.
func list(
	of []Control,
	sep string,
) string {
	names := make([]string, 0, len(of))
	for _, c := range of {
		names = append(names, c.Name)
	}

	return strings.Join(names, sep)
}

// hz is a centroid slope, with a sign and a thousands separator.
//
// Go's own verbs have no separator, and these run to five digits: nine
// thousand hertz per turn reads as a number where 9405 reads as an
// identifier.
func hz(
	v float64,
) string {
	sign := "+"
	if v < 0 {
		sign = "-"
	}

	digits := fmt.Sprintf("%.0f", math.Abs(v))

	var out []byte
	for i, d := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}

		out = append(out, d)
	}

	return sign + string(out)
}

// read turns the measurements into what the page needs.
func read(
	lib measured.Library,
	curves measured.Curves,
) (Page, error) {
	if len(curves.Controls) == 0 {
		return Page{}, fmt.Errorf("the curves name no controls")
	}

	page := Page{
		Blocks:    len(lib.Blocks),
		Reference: curves.Reference,
		Gear:      curves.Gear,
		Model:     curves.Block,
		Figures:   measured.Named(),
	}

	for _, b := range lib.Blocks {
		if b.Category == "amp" {
			page.Amps++
		}
	}

	for name, c := range curves.Controls {
		page.Controls = append(page.Controls, controlOf(name, c))
	}

	// Wire order, which is the only order that means anything: a parameter has
	// no name on the wire, only a position in the model's own list.
	sort.Slice(page.Controls, func(i, j int) bool {
		return page.Controls[i].Index < page.Controls[j].Index
	})

	page.shape()

	return page, nil
}

// controlOf is one control as the page reports it.
func controlOf(
	name string,
	c measured.Curve,
) Control {
	out := Control{Name: name, Index: c.Index, Muted: c.MutedAt}

	for _, f := range []struct {
		key audio.Figure
		to  *float64
	}{
		{audio.KeyCentroid, &out.Centroid},
		{audio.KeyLevel, &out.Level},
		{audio.KeyLow, &out.Low},
		{audio.KeyHigh, &out.High},
	} {
		if fit, ok := c.Fits[f.key]; ok {
			*f.to = fit.PerTurn
		}
	}

	if fit, ok := c.Fits[audio.KeyCentroid]; ok {
		out.Straight = fit.Straight
	}

	// Aimable when any figure moved further than its own floor. A move
	// smaller than the wander between two untouched takes is not a move
	// anybody can demonstrate.
	for _, f := range measured.Named() {
		moved, ok := measured.Apart(c.Points, f)
		if ok && math.Abs(moved) > c.Noise[f]*3 {
			out.Aimable = true

			break
		}
	}

	return out
}

// shape picks out the controls the page makes a claim about.
func (p *Page) shape() {
	for _, c := range p.Controls {
		switch {
		case c.Centroid > p.Widest.Centroid:
			p.Widest = c
		case c.Centroid < p.Opposite.Centroid:
			p.Opposite = c
		}

		if math.Abs(c.Level) > math.Abs(p.Loudest.Level) {
			p.Loudest = c
		}

		if !c.Aimable {
			p.Inert = append(p.Inert, c)
		}

		// Half is generous. A line accounting for less than that is not a
		// slope anybody may use as one, and the straightness figure exists so
		// nothing does.
		if c.Straight < 0.5 {
			p.Crooked = append(p.Crooked, c)
		}
	}
}
