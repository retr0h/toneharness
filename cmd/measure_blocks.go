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
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

var (
	measureBlocksDry      string
	measureBlocksOut      string
	measureBlocksCategory string
	measureBlocksSeconds  float64
	measureBlocksResume   bool
	measureBlocksRetry    bool
	measureBlocksHardware string
	measureBlocksHeadroom float64
	measureBlocksClient   clientFlags
)

// measureBlocksCmd represents the measure blocks command.
var measureBlocksCmd = &cobra.Command{
	Use:   "blocks",
	Short: "Measure every block the device has, once, at its own defaults",
	Long: `Play a reference recording through every block the device has, one at a
time, and keep what comes back.

One reading per block rather than a sweep of each. There are 661 blocks on an
HX Stomp carrying about five thousand controls between them, so sweeping all of
them is a hundred and sixty hours. It is also the wrong thing to want: the
matrix a request is solved with is measured fresh for the chain being tuned,
because a slope belongs to its chain, so a stored one per block would be
rebuilt before anything used it.

What cannot be worked out at solve time is which blocks belong in the chain to
begin with, and that needs one number per block.

Nothing is written to a slot. Each chain goes in front of the device with the
same live replace ` + "`device play`" + ` uses, because a slot is flash and
this loads a different chain six hundred times. See the rules that keep a
device alive in pkg/sdk/internal/wire/README.md.

The empty loop is measured first and kept as the baseline. Without it a figure
says nothing: 95 Hz is not what an equaliser does to a bass, it is what the
bass already was.

Everything is written as it goes, so a run interrupted partway keeps what it
had. ` + "`--resume`" + ` picks it up again.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		return cli.MeasureBlocks(cmd.Context(), cmd.OutOrStdout(), cli.MeasureOptions{
			Client:   measureBlocksClient.client(),
			Dry:      measureBlocksDry,
			Out:      measureBlocksOut,
			Category: catalog.Category(measureBlocksCategory),
			Seconds:  measureBlocksSeconds,
			Resume:   measureBlocksResume,
			Retry:    measureBlocksRetry,
			Headroom: measureBlocksHeadroom,
			Hardware: measureBlocksHardware,
		})
	},
}

func init() {
	measureCmd.AddCommand(measureBlocksCmd)

	f := measureBlocksCmd.Flags()
	f.StringVar(&measureBlocksDry, "dry", "resources/dry/bass-di.wav",
		"the reference recording to push through every block")
	f.StringVar(&measureBlocksOut, "out",
		"resources/sweeps/hx-stomp/fingerprints.json",
		"where the readings go")
	f.StringVar(&measureBlocksCategory, "category", "",
		"only blocks of this kind: amp, cab, drive, comp, eq and so on")
	f.Float64Var(&measureBlocksSeconds, "seconds", 4,
		"how much of the reference to push through per block")
	f.BoolVar(&measureBlocksResume, "resume", false,
		"skip blocks already in the output")
	f.BoolVar(&measureBlocksRetry, "retry", false,
		"with --resume, try the ones that refused again")
	f.StringVar(&measureBlocksHardware, "hardware", "hx stomp",
		"which attached audio device to push the signal through")
	f.Float64Var(
		&measureBlocksHeadroom,
		"headroom",
		measuringHeadroom,
		"decibels to turn the chain's own output down by before measuring; what opens the measuring loop is the destination, which is set whatever this says, and this bounds what is left",
	)
	f.StringVar(&measureBlocksClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&measureBlocksClient.device, "device", "", deviceUsage)
}
