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
	measureChainDry      string
	measureChainHardware string
	measureChainSeconds  float64
	measureChainTakes    int
	measureChainCab      bool
)

// measureChainCmd represents the measure chain command.
var measureChainCmd = &cobra.Command{
	Use:   "chain",
	Short: "Read what the pedal is playing now, and change nothing",
	Long: `Push the reference through whatever is loaded and say what came back.

Nothing is built, nothing is loaded and no control is moved. Play a preset, turn
a knob with ` + "`device turn`" + `, and read what that did.

This is the primitive the rest of the tool assumes and did not have. ` + "`tone tune`" + `,
` + "`tone reach`" + ` and every other measuring command build a preset and load it before
they read, so none of them can answer "what is it doing right now", and every
figure they print is already folded into a distance from a target. A chain
reading 84% of its energy above 2kHz went a whole session unnoticed that way.

The three band shares sum to one hundred and are where a fault shows first. A
bass DI through the empty loop reads about 96% below 250Hz. Anything in the same
rig reading most of its energy above 2kHz is not the chain being bright, it is
something else being measured.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		return cli.Chain(cmd.Context(), cmd.OutOrStdout(), cli.ChainOptions{
			Dry:        measureChainDry,
			Hardware:   measureChainHardware,
			Seconds:    measureChainSeconds,
			Takes:      measureChainTakes,
			EndsInACab: measureChainCab,
		})
	},
}

func init() {
	measureCmd.AddCommand(measureChainCmd)

	f := measureChainCmd.Flags()
	f.StringVar(&measureChainDry, "dry", "resources/dry/bass-di.wav",
		"the reference recording to push through what is loaded")
	f.Float64Var(&measureChainSeconds, "seconds", 4,
		"how much of the reference to push through per reading")
	f.IntVar(&measureChainTakes, "takes", 3,
		"how many takes the wander is measured from")
	f.BoolVar(&measureChainCab, "ends-in-a-cab", true,
		"whether the loaded chain finishes with a cabinet, which makes a reading "+
			"brighter than the reference a fault rather than a tone")
	f.StringVar(&measureChainHardware, "hardware", "",
		"which attached audio device to push the signal through")
}
