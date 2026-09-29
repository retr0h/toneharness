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
	toneReachID     string
	toneReachGenre  string
	toneReachCorpus string
	toneReachSweeps string
	toneReachClient clientFlags
)

// toneReachCmd represents the tone reach command.
var toneReachCmd = &cobra.Command{
	Use:   "reach",
	Short: "Say whether a chain could reach a target, without running",
	Long: `Ask whether this rig can sound like this before spending an afternoon on it.

` + "`tone tune`" + ` answers the same question by running. Three to five passes, one
reading per control per pass, five minutes of real-time audio with the pedal
held for all of it, and at the end a sentence saying the chain will not get
there. Every number that sentence rests on is already committed under
resources/sweeps/. This reads them instead, in about a second, with nothing
plugged in.

Two numbers per axis, and they are different kinds of claim.

OUT BY is how far the chain sits from the target, in that axis's own
tolerances, measured from the middle of every reading ever taken of these
blocks.

CAN MOVE is the most this chain's controls could shift that axis, summed across
every one of them. It flatters them on purpose: it assumes each slope holds
across a whole range it was only read locally, and that every control pulls the
same way, and neither is true. That is what makes it useful. A gap wider than
this is a gap nothing optimistic could close, so an axis marked OUT OF REACH is
out of reach, while one inside is only worth attempting.

So the answer is never "this will work". It is "this is worth running" or "no
arrangement of these controls gets there, change the chain rather than the
knobs", and the second is the one that saves the afternoon.

It is only as good as what has been swept. A block with no readings is named,
and whatever it could have moved is missing from CAN MOVE.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := toneReachClient.client()

		return cli.Reach(cmd.Context(), cmd.OutOrStdout(), cli.ReachOptions{
			Client: client,
			ID:     toneReachID,
			Genre:  toneReachGenre,
			Corpus: toneReachCorpus,
			Sweeps: toneReachSweeps,
		})
	},
}

func init() {
	toneCmd.AddCommand(toneReachCmd)

	f := toneReachCmd.Flags()
	f.StringVar(&toneReachID, "id", "", "the curated rig to ask about")
	f.StringVar(&toneReachGenre, "genre", "",
		"what to aim it at, by the name the corpus tags records with")
	f.StringVar(&toneReachCorpus, "corpus", "",
		"recordings to measure the target from, instead of the figures shipped "+
			"with this binary; measuring is most of a minute")
	f.StringVar(&toneReachSweeps, "sweeps", "",
		"the readings of this chain's blocks; resources/sweeps/hx-stomp without it")
	f.StringVar(&toneReachClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&toneReachClient.device, "device", "", deviceUsage)

	_ = toneReachCmd.MarkFlagRequired("id")
	_ = toneReachCmd.MarkFlagRequired("genre")
}
