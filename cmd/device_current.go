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

var presetsCurrentClient clientFlags

// deviceCurrentCmd represents the presets current command.
var deviceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show what the device is playing right now",
	Long: `Show the preset the device is playing, with every control where it now sits.

The edit buffer rather than a slot, and that is the whole difference. ` +
		"`presets turn`" + ` moves a control in the buffer without writing anything
back, so reading the slot it came from answers with the stored document and
makes it look as though nothing happened.

This is how a measurement says what it measured. A control's slope belongs to
the chain it was taken in: Treble on an amplifier into a 4x12 and the same
Treble into a 1x15 are two different numbers. A sweep that records the move
but not the chain records a number nobody can attribute later, so the sweep
runs this and keeps the answer beside the figures.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		read, err := presetsCurrentClient.client().Current(
			cmd.Context(), sdk.FormatRig)
		if err != nil {
			return err
		}

		return answer(cmd, read, cli.Reading)
	},
}

func init() {
	deviceCmd.AddCommand(deviceCurrentCmd)

	f := deviceCurrentCmd.Flags()
	f.StringVar(&presetsCurrentClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&presetsCurrentClient.device, "device", "", deviceUsage)
}
