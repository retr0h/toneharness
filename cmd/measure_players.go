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
	"os"

	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

var (
	measureCorpus          string
	measurePlayersEvidence bool
)

// measurePlayersCmd represents the measure players command.
var measurePlayersCmd = &cobra.Command{
	Use:   "players",
	Short: "Measure every player, and say what each one's records earn",
	Long: `Measure a tree of players and report what their figures earn them.

A word is earned by sitting clear of the other players, so this is the only mode
that produces any: one player has nobody to be clear of. That is why it is its
own command rather than a flag on ` + "`measure recordings`" + ` — the answer is a
comparison, not a profile.

Point it at one instrument. A bass centroid sits an octave below a guitar's, so a
corpus holding both would earn every bassist "dark" and every guitarist "bright"
and mean nothing by either.

    toneharness measure players --corpus resources/music/bass
    toneharness measure players --corpus resources/music/bass --evidence`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		players, err := audio.Corpus(os.DirFS(measureCorpus), ".")
		if err != nil {
			return err
		}

		if len(players) == 0 {
			return fmt.Errorf(
				"no players under %s: one directory of .wav recordings each, "+
					"separated with `just stems <in> <out>`",
				measureCorpus,
			)
		}

		// The same comparison, written two ways: a table to read, or the terms
		// it earned as evidence to paste into a rig. Both carry the figures,
		// because a word without them is an assertion.
		if measurePlayersEvidence {
			return answer(cmd, players, cli.PlayerTerms)
		}

		return answer(cmd, players, cli.Players)
	},
}

func init() {
	measureCmd.AddCommand(measurePlayersCmd)

	f := measurePlayersCmd.Flags()
	f.StringVar(&measureCorpus, "corpus", "",
		"a tree of players, one directory each, compared against each other")
	f.BoolVar(&measurePlayersEvidence, "evidence", false,
		"write what each player earns as rig evidence, to paste into a chain")

	// Fails only for a flag that does not exist, and this one is defined above.
	_ = measurePlayersCmd.MarkFlagRequired("corpus")
}
