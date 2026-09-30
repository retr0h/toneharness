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
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/cli"
)

var (
	measureControlsModel    string
	measureControlsDry      string
	measureControlsOut      string
	measureControlsSeconds  float64
	measureControlsPoints   int
	measureControlsTakes    int
	measureControlsHardware string
	measureControlsHeadroom float64
	measureControlsClient   clientFlags
)

// measureControlsCmd represents the measure controls command.
var measureControlsCmd = &cobra.Command{
	Use:   "controls",
	Short: "Measure what every control of one block does",
	Long: `Put one block in front of the device on its own and move each of its
controls through its range, measuring at every position.

The expensive half of measuring a device. A control is about two minutes and a
twelve-control amplifier most of an hour, so this is aimed at the blocks a
chain reaches for rather than run across all 661.
` + "`toneharness measure blocks`" + ` is the cheap half and covers everything.

Three things a reading has to control for, and each was got wrong first.

The chain, because a slope is not a property of a control: the same Treble
into a 4x12 and into a 1x15 are two different numbers. This builds a chain
holding one block and nothing else, and the whole rig travels in the answer.

The starting point, because a live edit writes nothing back. A control stays
where the last sweep left it, at the top of its range, so the chain goes back
in front of the device before every control.

And silence, because the noise floor cannot catch a figure computed on
nothing: two takes of silence agree to the last digit. A position more than
30dB under the settled level is recorded as muted and left out of the
arithmetic.

A control's range comes from the catalog. That is not a nicety: 1,452 of the
device's 4,835 float controls do not run zero to one, and a Simple EQ's Mid
Freq swept 0..1 never leaves its bottom stop and reports as inert.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		out := measureControlsOut
		if out == "" {
			out = filepath.Join("resources", "sweeps", "hx-stomp",
				strings.ToLower(measureControlsModel)+".json")
		}

		return cli.MeasureControls(cmd.Context(), cmd.OutOrStdout(),
			cli.ControlsOptions{
				Client:   measureControlsClient.client(),
				Model:    measureControlsModel,
				Dry:      measureControlsDry,
				Out:      out,
				Seconds:  measureControlsSeconds,
				Points:   measureControlsPoints,
				Takes:    measureControlsTakes,
				Headroom: measureControlsHeadroom,
				Hardware: measureControlsHardware,
			})
	},
}

func init() {
	measureCmd.AddCommand(measureControlsCmd)

	f := measureControlsCmd.Flags()
	f.StringVar(&measureControlsModel, "model", "",
		"the block to measure, by its model identifier")
	f.StringVar(&measureControlsDry, "dry", "resources/dry/bass-di.wav",
		"the reference recording to push through it")
	f.StringVar(&measureControlsOut, "out", "",
		"where the curves go; named for the model by default")
	f.Float64Var(&measureControlsSeconds, "seconds", 4,
		"how much of the reference to push through per position")
	f.IntVar(&measureControlsPoints, "points", 9,
		"how many positions to measure a dial at; a list gets all of its settings")
	f.IntVar(&measureControlsTakes, "takes", 3,
		"how many takes the noise floor is measured from")
	f.StringVar(&measureControlsHardware, "hardware", "hx stomp",
		"which attached audio device to push the signal through")
	f.Float64Var(
		&measureControlsHeadroom,
		"headroom",
		measuringHeadroom,
		"decibels to turn the chain's own output down by before measuring; what opens the measuring loop is the destination, which is set whatever this says, and this bounds what is left",
	)
	f.StringVar(&measureControlsClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&measureControlsClient.device, "device", "", deviceUsage)

	// Fails only for a flag that does not exist, and it is defined above.
	_ = measureControlsCmd.MarkFlagRequired("model")
}
