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
)

var (
	slotsFillRigs   string
	slotsFillClient clientFlags
)

// slotsFillCmd represents the slots fill command.
var slotsFillCmd = &cobra.Command{
	Use:   "fill",
	Short: "Build every rig a setup places and write it to its slot",
	Long: `Put the pedal back the way a setup says it should be.

A setup may say which slot each rig lives in. This builds every one of them and
writes it there, so "update my pedal" is a thing that can be asked for twice
and land in the same places both times.

    slots:
      - { at: 11A, rig: punk }
      - { at: 11B, rig: pop-punk }
      - { at: 18A, rig: mike-dirnt }

The layout is somebody's own, which is why it lives on the setup rather than
here: two people building one rig want the same preset and different slots,
since one of them already keeps their own work in bank one. Nothing is written
where no slot says to.

Yours is read from ` + "`$XDG_CONFIG_HOME/toneharness/setup.yaml`" + ` unless
` + "`--setup`" + ` names another, so ` + "`setup init`" + ` once is the whole of
getting started.

Every slot named is a flash write and a device has no undo. Each one keeps
whatever it held first and the answer says where that went, so a change is
recoverable; the paths are worth keeping.

One rig failing does not stop the rest. A rig that will not build is a document
to fix, and leaving the other forty written beats leaving a pedal half updated
with no record of which half.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		slotsFillClient.setup = ownSetup(slotsFillClient.setup)

		placed, err := slotsFillClient.client(
			ownRigs(slotsFillRigs)).Fill(cmd.Context())

		// Said before the error is returned, because a run that wrote thirty
		// slots and failed on the thirty-first has thirty paths somebody
		// needs and an error that carries none of them.
		if said := cli.Filled(cmd.OutOrStdout(), placed); said != nil {
			return said
		}

		return err
	},
}

func init() {
	slotsCmd.AddCommand(slotsFillCmd)

	f := slotsFillCmd.Flags()
	f.StringVar(&slotsFillClient.setup, "setup", "",
		"a Setup saying which slot each rig lives in, instead of yours")
	f.StringVar(&slotsFillRigs, "rigs", "",
		"a directory of rigs to use instead of yours, beside the built-in ones")
	f.StringVar(&slotsFillClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&slotsFillClient.device, "device", "", deviceUsage)
	f.StringVar(&slotsFillClient.sweeps, "sweeps", "resources/sweeps/hx-stomp",
		"the committed curves an amplifier's controls are set from")
}
