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
	"fmt"

	"github.com/spf13/cobra"
)

var (
	presetsPlayPreset string
	presetsPlayClient clientFlags
)

// devicePlayCmd represents the presets play command.
var devicePlayCmd = &cobra.Command{
	Use:   "play",
	Short: "Put a preset in front of the device without storing it",
	Long: `Put a preset file in front of the device without storing it anywhere.

The device starts making that sound at once. No slot is written and no slot is
read, so nothing it holds changes and there is nothing to put back.

Use this rather than ` + "`presets import`" + ` for anything being tried rather
than kept. A slot is flash, flash wears out, and a burst of writes has taken a
setlist past what a power cycle could clear. Auditioning six hundred chains
through slots is six hundred flash writes for readings nobody wanted to keep;
through here it is none.

What it replaces lasts until the next preset is selected or the device is power
cycled, at which point the slot's own version comes back.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		if err := presetsPlayClient.client().Play(
			cmd.Context(), presetsPlayPreset); err != nil {
			return err
		}

		_, err := fmt.Fprintf(cmd.OutOrStdout(),
			"\n  the device is playing %s, and holds what it held\n\n",
			presetsPlayPreset)

		return err
	},
}

func init() {
	deviceCmd.AddCommand(devicePlayCmd)

	f := devicePlayCmd.Flags()
	f.StringVar(&presetsPlayPreset, "preset", "",
		"the .hlx to put in front of the device")
	f.StringVar(&presetsPlayClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&presetsPlayClient.device, "device", "", deviceUsage)

	// Fails only for a flag that does not exist, and it is defined above.
	_ = devicePlayCmd.MarkFlagRequired("preset")
}
