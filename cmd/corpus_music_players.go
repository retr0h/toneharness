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

// corpusMusicPlayersCmd represents the corpus music players command.
var corpusMusicPlayersCmd = &cobra.Command{
	Use:   "players",
	Short: "List who the corpus holds records for",
	Long: `Every player the corpus names, and what their records say they are.

The directory is the rig's identifier, and that is the only thing joining a rig
to its records. A directory spelled any other way leaves the rig reading as one
nobody has measured, which looks identical to the ordinary case.

A player with untagged records is worth seeing: a record naming no genre is
invisible to every genre, so it counts towards none of them.

    toneharness corpus music players --corpus resources/music/bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		of, err := newClient().MusicPlayers(cmd.Context(), musicCorpus)
		if err != nil {
			return err
		}

		return answer(cmd, of, cli.MusicPlayers)
	},
}

func init() {
	corpusMusicCmd.AddCommand(corpusMusicPlayersCmd)
}
