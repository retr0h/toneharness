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
	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// Resolved prints the chain a rig resolved to, and how many controls each block
// carries.
//
// The count is the point. A rig that said one word about an amplifier resolves to
// a block with a dozen controls set, and seeing that is how somebody knows the
// document now describes the preset rather than gesturing at it.
func Resolved(
	w io.Writer,
	of result.Made,
) error {
	rows := make([][]string, 0, len(of.Plan.Blocks))

	for _, b := range of.Plan.Blocks {
		rows = append(rows, []string{
			paint.Accent(w, string(b.Model)),
			fmt.Sprintf("%d", b.DSP),
			fmt.Sprintf("%d", b.Pos),
			fmt.Sprintf("%d", len(b.Params)),
		})
	}

	return paint.Section{
		Title:   "What it resolved to",
		Detail:  fmt.Sprintf("written to %s", of.Path),
		Headers: []string{"model", "dsp", "pos", "controls"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Right, lipgloss.Right, lipgloss.Right,
		},
		Empty: "the rig resolved to no blocks, which is a rig naming no gear",
		Summary: "every control is in the document now, so editing one and " +
			"building again uses what you wrote rather than solving it over",
	}.Render(w)
}
