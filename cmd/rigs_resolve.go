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
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk"
)

var rigsResolveOptions sdk.Resolve

// rigsResolveCmd represents the rigs resolve command.
var rigsResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Write a rig out with every control it resolves to",
	Long: `Build a rig and write the whole answer back as a document.

A rig names gear and says a sound in words, and the compiler turns that into a
chain with every control set. This writes that chain down: every block, in the
order the device holds it, with every control at the value it was given.

What it is for is editing. Open the result, change one control, and build: the
value is used as it stands, with nothing re-derived and nothing to teach the
compiler first. It is also what makes a change survive the trip to the pedal and
back, because the controls come back into the same fields they went out of.

Blocks the compiler added appear in the chain too. A rig that named an amplifier
and quietly became five blocks could not be read as a description of the preset
it produced.

The words stay in the ask, where they belong: they are what somebody wanted, and
the controls are what answered. Nothing re-reads them afterwards.

Writes over the file the rig came from unless --out names somewhere else. A rig
that ships in the binary has no file, so that one needs --out.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		made, err := newClient().Resolve(cmd.Context(), rigsResolveOptions)
		if err != nil {
			return err
		}

		return answer(cmd, made, cli.Resolved)
	},
}

func init() {
	rigsCmd.AddCommand(rigsResolveCmd)

	rigsResolveCmd.Flags().StringVar(
		&rigsResolveOptions.RigID, "id", "", "which rig to resolve")
	rigsResolveCmd.Flags().StringVar(
		&rigsResolveOptions.Out, "out", "",
		"where to write it, instead of over the file it came from")

	_ = rigsResolveCmd.MarkFlagRequired("id")
}
