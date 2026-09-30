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
	measureNamesModel    string
	measureNamesHeadroom float64
	measureNamesClient   clientFlags
)

// measureNamesCmd represents the measure names command.
var measureNamesCmd = &cobra.Command{
	Use:   "names",
	Short: "Check the catalog's parameter order against the device",
	Long: `Move each of a block's parameters by index and report which named
parameter changed.

A parameter has no name on the wire, only a position in the model's own list.
Every curve this tool measures is filed under that position, and the catalog's
claim about which position is which has never been held to the hardware.

It matters more than it sounds. ` + "`catalog show`" + ` prints the same
parameters sorted for a reader, so the two orders disagree on nearly every
model: one amplifier's listing begins Bass, Bias, BiasX where its wire order
begins Norm Drive, Bass, Mid, Treble. Counting down the printed one mislabels
every curve, and the numbers stay entirely plausible while it does.

Two values are tried per index, because a control already resting on the first
would show no change and be reported as an index that reaches nothing.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		return cli.MeasureNames(cmd.Context(), cmd.OutOrStdout(), cli.NamesOptions{
			Client: measureNamesClient.client(),
			Model:  measureNamesModel,
		})
	},
}

func init() {
	measureCmd.AddCommand(measureNamesCmd)

	f := measureNamesCmd.Flags()
	f.StringVar(&measureNamesModel, "model", "",
		"the block to check, by its model identifier")
	f.Float64Var(
		&measureNamesHeadroom,
		"headroom",
		measuringHeadroom,
		"decibels to turn the chain's own output down by before measuring; what opens the measuring loop is the destination, which is set whatever this says, and this bounds what is left",
	)
	f.StringVar(&measureNamesClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&measureNamesClient.device, "device", "", deviceUsage)

	// Fails only for a flag that does not exist, and it is defined above.
	_ = measureNamesCmd.MarkFlagRequired("model")
}
