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

	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/toneharness/pkg/cli/internal/paint"
	"github.com/retr0h/toneharness/pkg/sdk/config"
)

// Setup says what the setup holds, in the order somebody asks about it.
//
// The slot map and the favourites first, because those are the two things
// nothing else records: which presets on the pedal belong to this tool rather
// than to whoever was holding it, and which of them are worth one keystroke.
// Everything else on a setup shows up in a build's own answer.
func Setup(
	w io.Writer,
	held config.Config,
) error {
	if err := setupGear(w, held); err != nil {
		return err
	}

	if err := setupSlots(w, held); err != nil {
		return err
	}

	return setupFavourites(w, held)
}

// setupGear prints the pedal, the instruments and the rig builds match.
func setupGear(
	w io.Writer,
	held config.Config,
) error {
	rows := [][]string{}

	if held.Device != nil {
		rows = append(rows, []string{paint.Mute(w, "device"), held.Device.Model})
	}

	if held.Instruments != nil {
		for _, one := range *held.Instruments {
			rows = append(rows, []string{paint.Mute(w, "instrument"), one.Gear})
		}
	}

	if held.MatchLoudness != nil {
		rows = append(rows,
			[]string{paint.Mute(w, "as loud as"), *held.MatchLoudness})
	}

	if len(rows) == 0 {
		_, err := fmt.Fprintf(w, "\n%s%s\n", paint.Indent, paint.Mute(w,
			"nothing about the gear; a build will use the defaults"))

		return err
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return paint.Table(w, rows,
		[]lipgloss.Position{lipgloss.Left, lipgloss.Left})
}

// setupSlots prints which rig each slot holds.
func setupSlots(
	w io.Writer,
	held config.Config,
) error {
	if _, err := fmt.Fprintf(w, "\n%s\n",
		paint.Heading(w, "slots")); err != nil {
		return err
	}

	if held.Slots == nil || len(*held.Slots) == 0 {
		_, err := fmt.Fprintf(w, "\n%s%s\n", paint.Indent, paint.Mute(w,
			"none, so slots fill has nothing to write and writes nothing"))

		return err
	}

	rows := make([][]string, 0, len(*held.Slots))

	for _, placed := range *held.Slots {
		// What this one is built for, where it differs from the file's own
		// answer. Two slots holding one rig are the same name twice
		// otherwise, and the difference between them is the whole reason
		// both are there.
		into := ""
		if placed.PlaysInto != nil {
			into = paint.Mute(w, "into "+string(*placed.PlaysInto))
		}

		rows = append(rows,
			[]string{paint.Accent(w, placed.At), placed.Rig, into})
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return paint.Table(w, rows,
		[]lipgloss.Position{lipgloss.Left, lipgloss.Left, lipgloss.Left})
}

// setupFavourites prints the cycle, and says which one is the start.
func setupFavourites(
	w io.Writer,
	held config.Config,
) error {
	if _, err := fmt.Fprintf(w, "\n%s\n",
		paint.Heading(w, "favourites")); err != nil {
		return err
	}

	if held.Favourites == nil || len(*held.Favourites) == 0 {
		_, err := fmt.Fprintf(w, "\n%s%s\n", paint.Indent, paint.Mute(w,
			"none, so device fav has nowhere to go"))

		return err
	}

	rows := make([][]string, 0, len(*held.Favourites))
	rigs := map[string]string{}

	if held.Slots != nil {
		for _, placed := range *held.Slots {
			rigs[placed.At] = placed.Rig
		}
	}

	for i, at := range *held.Favourites {
		// Saying which is the start, because the rule is that the first one
		// is and a list read off a screen does not say so by itself.
		note := ""
		if i == 0 {
			note = paint.Mute(w, "where device fav lands")
		}

		// A favourite the slot map says nothing about is the one case where
		// cycling cannot tell it is the current one, so it is called out
		// where somebody is looking at the list.
		rig := rigs[at]
		if rig == "" {
			rig = paint.Info(w, "not in slots, so a step cannot recognise it")
		}

		rows = append(rows, []string{paint.Accent(w, at), rig, note})
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return paint.Table(w, rows,
		[]lipgloss.Position{lipgloss.Left, lipgloss.Left, lipgloss.Left})
}
