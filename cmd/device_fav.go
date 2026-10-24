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

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk"
)

var (
	deviceFavStep   string
	deviceFavClient clientFlags
)

// steps are what --step takes, and what each one means.
var steps = map[string]sdk.Step{
	"first": sdk.AtFirst,
	"next":  sdk.Onward,
	"prev":  sdk.Back,
}

// deviceFavCmd represents the device fav command.
var deviceFavCmd = &cobra.Command{
	Use:   "fav",
	Short: "Load one of your favourite slots",
	Long: `Go to a favourite without naming a slot.

The pedal holds a hundred and twenty-six presets and you play four.
` + "`device select`" + ` reaches any of them and wants the number every
time, which is the wrong shape for the four: it asks you to remember a number
while you are holding a bass. This asks the setup.

    favourites: ["11A", "18A", "11B"]

Without ` + "`--step`" + ` it goes to the first, which is what the list's
order is for. ` + "`--step next`" + ` and ` + "`--step prev`" + ` move one
along and wrap at either end.

Where the pedal is now is read off the pedal rather than remembered between
runs, because the footswitch is right there and a remembered position would be
wrong most of the time. What it is playing is matched against the rig the
setup's ` + "`slots`" + ` puts in each favourite. Playing something that is no
favourite is not an error: the step goes to the first and says that is what it
did.

Nothing is written. The preset goes into the edit buffer and the slot it came
from is untouched, so this costs no flash write however often you run it.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		step, ok := steps[deviceFavStep]
		if !ok {
			return fmt.Errorf(
				"--step is %q; it takes first, next or prev", deviceFavStep)
		}

		pedal.claim()

		deviceFavClient.setup = ownSetup(deviceFavClient.setup)

		chosen, err := deviceFavClient.client().Favourite(cmd.Context(), step)
		if err != nil {
			return err
		}

		return answer(cmd, chosen, cli.Chosen)
	},
}

func init() {
	deviceCmd.AddCommand(deviceFavCmd)

	f := deviceFavCmd.Flags()
	f.StringVar(&deviceFavStep, "step", "first",
		"which favourite: first, next or prev")
	f.StringVar(&deviceFavClient.setup, "setup", "", setupUsage)
	f.StringVar(&deviceFavClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&deviceFavClient.device, "device", "", deviceUsage)
}
