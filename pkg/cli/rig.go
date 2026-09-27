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
	"fmt"
	"io"
	"strings"

	"github.com/retr0h/toneharness/pkg/cli/internal/paint"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// Rigs prints every rig a directory holds, one to a row.
func Rigs(
	w io.Writer,
	r sdk.Rigs,
) error {
	rows := make([][]string, 0, len(r.Rigs))

	for _, known := range r.Rigs {
		rows = append(rows, []string{
			paint.Accent(w, known.Rig.ID),
			named(known),
			paint.Mute(w, string(known.Rig.Instrument)),
			rig.GearName(known.Rig, rig.RoleAmp),
			source(w, known.Rig),
		})
	}

	return wrapReport(paint.Section{
		Title:   "Rigs",
		Detail:  r.Dir,
		Headers: []string{"id", "name", "instrument", "amp", "source"},
		Rows:    rows,
		Empty:   "no rigs here",
	}.Render(w))
}

// Rig prints one rig in full.
func Rig(
	w io.Writer,
	r sdk.Rig,
) error {
	spec := r.Rig
	d := paint.Detail{Title: named(r.Known), Subtitle: spec.ID}

	d.Fields = append(d.Fields, about(r.Ask)...)
	d.Fields = append(d.Fields,
		paint.Field{Label: "instrument", Value: string(spec.Instrument)})
	d.Fields = append(d.Fields, signalPath(spec)...)
	d.Fields = append(d.Fields, asked(r.Ask)...)
	d.Fields = append(d.Fields, variants(r.Variants)...)
	d.Fields = append(d.Fields, paint.Field{
		Label: "source",
		Value: fmt.Sprintf("%s, %s confidence",
			rig.Sourced(spec), confidence(r.Ask)),
	})

	switch {
	case !rig.Trusted(spec):
		d.Note = "unverified, nobody has confirmed this gear"
	case r.Ask == nil:
		// Worth saying rather than rendering as a rig with no words. The gear is
		// here and what it was for is not, so nothing can tell whether the rig
		// answers the question it was built for.
		d.Note = "no ask beside it, so nothing records what this was built for"
	}

	return wrapReport(d.Render(w))
}

// named is what to call a rig, which only its ask knows.
//
// The rig carries an identifier and the ask carries the subject, so a rig with
// no ask has a name nobody wrote down. Its identifier stands in, which is what
// a listing needs: a blank cell reads as a bug and the identifier is true.
func named(
	known sdk.Known,
) string {
	if known.Ask == nil || known.Ask.Subject == nil {
		return known.Rig.ID
	}

	return known.Ask.Subject.Name
}

// about is who the ask is for, where it says.
func about(
	ask *tone.Spec,
) []paint.Field {
	if ask == nil || ask.Subject == nil {
		return nil
	}

	out := []paint.Field(nil)

	if ask.Subject.Band != nil && *ask.Subject.Band != "" {
		out = append(out, paint.Field{Label: "band", Value: *ask.Subject.Band})
	}

	if ask.Subject.Era != nil && *ask.Subject.Era != "" {
		out = append(out, paint.Field{Label: "era", Value: *ask.Subject.Era})
	}

	return out
}

// asked is how the ask says it should sound and how it is played.
//
// Both come off the ask and neither is on the rig, which is the whole point of
// the split: a word is what somebody meant, and the gear below is what answered
// it. Printed after the signal path so the page reads answer first, then what it
// was answering.
func asked(
	ask *tone.Spec,
) []paint.Field {
	if ask == nil {
		return nil
	}

	out := []paint.Field(nil)

	if ask.Technique != nil {
		out = append(out,
			paint.Field{Label: "technique", Value: technique(*ask.Technique)})
	}

	return append(out, words(ask)...)
}

// variants lists the rigs that are a small change on this one.
//
// Only the first carries the label, so several read as one block rather than
// as the same word repeated down the page.
func variants(
	all []sdk.Variant,
) []paint.Field {
	out := make([]paint.Field, 0, len(all))

	for _, v := range all {
		label := ""
		if len(out) == 0 {
			label = "variants"
		}

		out = append(out, paint.Field{
			Label: label,
			Value: fmt.Sprintf("%s (%s)", v.Name, v.ID),
		})
	}

	return out
}

// source names where a rig's knowledge came from, and marks it when nobody
// has confirmed it.
//
// This is the column that decides whether to trust the row, so it is the one
// that carries colour. A rig is only shown as confirmed when every claim in
// it rests on something checkable — the weakest link is what the reader needs
// to know about.
func source(
	w io.Writer,
	spec rig.Spec,
) string {
	if rig.Trusted(spec) {
		return paint.OK(w, string(rig.Sourced(spec)))
	}

	return paint.Info(w, string(rig.Sourced(spec)))
}

// chain renders the signal path, in order, one row per piece of gear.
//
// Labelled by role rather than by position, because "amp" is what a person
// reading this wants to find and "3" is not.
func signalPath(
	spec rig.Spec,
) []paint.Field {
	out := make([]paint.Field, 0, len(spec.Chain))

	for _, e := range spec.Chain {
		out = append(out, paint.Field{Label: string(e.Role), Value: e.Gear})
	}

	return out
}

// confidence reports how far an ask says its answer should be trusted.
//
// Low when nothing says otherwise, including when there is no ask at all. A
// claim asserting high confidence with no evidence behind it is worth showing as
// unverified whatever it says about itself, and a rig nobody wrote an ask for
// says nothing about itself at all.
func confidence(
	ask *tone.Spec,
) tone.Confidence {
	if ask == nil || ask.Confidence == nil {
		return tone.ConfidenceLow
	}

	return *ask.Confidence
}

// words renders how it should sound, one per row, labelled only once.
//
// The label repeats as blank so the values line up in the same column as
// every other field rather than starting a block of their own.
func words(
	ask *tone.Spec,
) []paint.Field {
	if ask == nil || ask.Words == nil || len(*ask.Words) == 0 {
		return nil
	}

	out := make([]paint.Field, 0, len(*ask.Words))

	for i, word := range *ask.Words {
		label := ""
		if i == 0 {
			label = "words"
		}

		out = append(out, paint.Field{Label: label, Value: word.Term})
	}

	return out
}

// where reads a position back as the phrase a player would use.
var where = map[tone.Position]string{
	tone.PositionBridge: "near the bridge",
	tone.PositionMiddle: "over the middle",
	tone.PositionNeck:   "over the neck",
}

// technique writes the three things a rig stores as the one sentence a person
// would say.
//
// A rig stores them apart so that two rigs can be compared, and nobody says
// "attack: pick, position: bridge" out loud. Muting is named only when there
// is some, because "not muted" is what every unmuted note already sounds like.
func technique(
	t tone.Technique,
) string {
	parts := []string{string(t.Attack)}

	if t.Position != nil {
		parts = append(parts, where[*t.Position])
	}

	if t.Muting != nil && *t.Muting == tone.MutingPalm {
		parts = append(parts, "palm muted")
	}

	return strings.Join(parts, ", ")
}

// wrapReport gives a reporting failure the same shape everywhere.
func wrapReport(
	err error,
) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("reporting: %w", err)
}

// Scaffolded says what rig was written and what to do with it.
func Scaffolded(
	w io.Writer,
	sc sdk.Scaffolded,
) error {
	rows := [][]string{
		{paint.Mute(w, "id"), paint.Accent(w, sc.ID)},
		{paint.Mute(w, "instrument"), sc.Instrument},
		{paint.Mute(w, "amp"), sc.Amp},
	}

	if sc.Cab != "" {
		rows = append(rows, []string{paint.Mute(w, "cab"), sc.Cab})
	}

	if len(sc.Pedals) > 0 {
		rows = append(rows,
			[]string{paint.Mute(w, "pedals"), strings.Join(sc.Pedals, ", ")})
	}

	if err := (paint.Section{
		Title: sc.Name, Detail: sc.Path, Rows: rows,
		Summary: closing(sc),
	}).Render(w); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}

// closing says what was done and what to do next.
//
// Only a rig scaffolded from gear names has had them resolved, so only that
// one says so. A copy took the parent's chain as it stands and checked
// nothing, and the honest thing to report is what it did: whose chain this is,
// and that nothing was changed on the way.
func closing(
	sc sdk.Scaffolded,
) string {
	next := fmt.Sprintf("next: toneharness presets make --id %s --out %s.hlx", sc.ID, sc.ID)

	if sc.Copied() {
		return fmt.Sprintf("chain copied from %s unchanged — %s", sc.From, next)
	}

	return fmt.Sprintf("every gear name resolves — %s", next)
}
