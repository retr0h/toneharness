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
	corpusShowModel  string
	corpusShowClient clientFlags
)

// corpusShowCmd represents the corpus show command.
var corpusShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show how people set one model",
	Long: `The distribution of every knob, across every preset that used this model.

One model, named the way ` + "`catalog show`" + ` names one: the two commands answer the
same question from either side, what the device allows against what people did.

The spread is the useful column. A parameter everybody sets the same way is one
this tool can be confident about; one nobody agrees on belongs to the player.

    toneharness corpus presets show --model HD2_AmpSVBeastBrt`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		got, err := corpusShowClient.client().
			ModelMeasurements(cmd.Context(), corpusShowModel)
		if err != nil {
			return err
		}

		return answer(cmd, got, cli.Measured)
	},
}

func init() {
	corpusPresetsCmd.AddCommand(corpusShowCmd)

	f := corpusShowCmd.Flags()
	f.StringVar(&corpusShowModel, "model", "",
		"model identifier, e.g. HD2_AmpSVBeastBrt")
	// Fails only for a flag that does not exist, and this one is defined above.
	_ = corpusShowCmd.MarkFlagRequired("model")
	f.StringVar(&corpusShowClient.stats, "stats", "",
		"measured statistics to use instead of the built-in ones")
	f.StringVar(&corpusShowClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&corpusShowClient.device, "device", "", deviceUsage)
}
