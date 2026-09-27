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

// corpusMusicBandsCmd represents the corpus music bands command.
var corpusMusicBandsCmd = &cobra.Command{
	Use:   "bands",
	Short: "List the bands that made the records",
	Long: `Every band the manifests name, and how many records and players each has.

The band is on the record rather than on the player, because a bassist plays in
several over a career and a record belongs to one. Grouped on the slug, so
"Guns N' Roses" and "Guns n Roses" are one band rather than two.

No threshold here. A band is not something to aim at: it is a way of finding the
records, where a genre is a distribution somebody asks for.

    toneharness corpus music bands --corpus resources/music/bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		of, err := newClient().MusicBands(cmd.Context(), musicCorpus)
		if err != nil {
			return err
		}

		return answer(cmd, of, func(w io.Writer, got []sdk.MusicGroup) error {
			return cli.MusicGroups(w, got, "band", false)
		})
	},
}

func init() {
	corpusMusicCmd.AddCommand(corpusMusicBandsCmd)
}
