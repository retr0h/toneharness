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

// corpusMusicRecordsCmd represents the corpus music records command.
var corpusMusicRecordsCmd = &cobra.Command{
	Use:   "records",
	Short: "List every record the corpus names",
	Long: `Every recording, with the year, the band, the genres and who decided them.

Also whether the bass has been separated out of it. A record named in a manifest
with no stems beside it is measured by nothing, and nothing else says so: the
manifest looks complete and the measurement quietly leaves it out.

    toneharness corpus music records --corpus resources/music/bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		of, err := newClient().MusicRecords(cmd.Context(), musicCorpus)
		if err != nil {
			return err
		}

		return answer(cmd, of, cli.MusicRecords)
	},
}

func init() {
	corpusMusicCmd.AddCommand(corpusMusicRecordsCmd)
}
