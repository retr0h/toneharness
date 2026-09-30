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
	"io"

	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/cli"
	sdk "github.com/retr0h/toneharness/pkg/sdk"
)

// corpusMusicGenresCmd represents the corpus music genres command.
var corpusMusicGenresCmd = &cobra.Command{
	Use:   "genres",
	Short: "List the genres the records cover, and which can be aimed at",
	Long: `Every genre the manifests name, with how far each one is from a distribution.

A genre is where its records sit in the figures everything else is measured in.
Nobody defines it: anybody holding the same recordings gets the same answer. That
needs enough records, from enough different people, and the threshold is eight
records from at least three players.

Three records by one band is that band's sound wearing a genre's name, and a
request for the genre would get the band. The figures cannot tell those apart,
so the count is the only thing that can.

The checked column is the other half. A genre can reach the threshold entirely
on tags a model guessed, which still reaches it, and somebody aiming at it
should know nobody has looked.

    toneharness corpus music genres --corpus resources/music/bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		of, err := newClient().MusicGenres(cmd.Context(), musicCorpus)
		if err != nil {
			return err
		}

		return answer(cmd, of, func(w io.Writer, got []sdk.MusicGroup) error {
			return cli.MusicGroups(w, got, "genre", true)
		})
	},
}

func init() {
	corpusMusicCmd.AddCommand(corpusMusicGenresCmd)
}
