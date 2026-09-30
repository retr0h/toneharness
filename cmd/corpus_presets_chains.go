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
	corpusChainsInstrument string
	corpusChainsClient     clientFlags
)

// corpusPresetsChainsCmd represents the corpus presets chains command.
var corpusPresetsChainsCmd = &cobra.Command{
	Use:   "chains",
	Short: "Show what a chain tends to hold",
	Long: `The chain grammar: which kinds of block a chain holds, and where.

Measured across every preset in the corpus, per instrument. Across 159 bass
chains 88% hold a compressor and 62% hold drive, which sits before the amp 88%
of the time. A build uses that for the blocks it adds; blocks a rig names stay
where the rig put them.

It does not say which of two effects comes first. That would need pairs measured,
and nothing needs it until a build generates a chain of several effects.

    toneharness corpus presets chains --instrument bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		got, err := corpusChainsClient.client().
			ChainMeasurements(cmd.Context(), corpusChainsInstrument)
		if err != nil {
			return err
		}

		return answer(cmd, got, cli.Measured)
	},
}

func init() {
	corpusPresetsCmd.AddCommand(corpusPresetsChainsCmd)

	f := corpusPresetsChainsCmd.Flags()
	f.StringVar(&corpusChainsInstrument, "instrument", "",
		"limit the grammar to guitar or bass")
	f.StringVar(&corpusChainsClient.stats, "stats", "",
		"measured statistics to use instead of the built-in ones")
	f.StringVar(&corpusChainsClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&corpusChainsClient.device, "device", "", deviceUsage)
}
