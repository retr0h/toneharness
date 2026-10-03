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
	toneBuildAsk    string
	toneBuildSetup  string
	toneBuildOut    string
	toneBuildClient clientFlags
)

// toneBuildCmd represents the tone build command.
var toneBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Turn a request and a setup into a rig",
	Long: `Read a document and a Setup and write the document the ask resolves to.

Two halves, and they answer different kinds of ask.

Gear a request names by hand resolves against the catalog. That is exact and
always was: "LA Studio Comp" is one model and the rig says which.

Gear it does not name is chosen by measuring. A request pointing at a
recording has that recording measured through the same figures every block on
the device was, so what comes back is the nearest of 224 amplifiers rather
than whichever name somebody wrote down. Nothing in that path needs anybody
to have described an amplifier.

Every choice it makes, and every part of the ask it could not honour, is
reported. A genre no records carry, a player whose records this does not read
yet, a gear name that fits four models and so fits none, a nudge with no
previous answer to move from: each is said rather than quietly dropped.

Compile what comes out with ` + "`toneharness presets compile --rig`" + `.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cli.ToneBuild(cmd.OutOrStdout(), cli.ToneBuildOptions{
			Client: toneBuildClient.client(),
			Ask:    toneBuildAsk,
			Setup:  toneBuildSetup,
			Out:    toneBuildOut,
			AsData: asData,
		})
	},
}

func init() {
	toneCmd.AddCommand(toneBuildCmd)

	f := toneBuildCmd.Flags()
	f.StringVar(&toneBuildAsk, "ask", "", "the ToneSpec to read")
	f.StringVar(&toneBuildSetup, "setup", "",
		"the Setup to read; without one the rig is for a bass")
	f.StringVar(&toneBuildOut, "out", "",
		"where the rig goes; standard output without it")

	// Fails only for a flag that does not exist, and it is defined above.
	_ = toneBuildCmd.MarkFlagRequired("ask")
}
