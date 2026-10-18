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

// corpusMusicLengthsCmd represents the corpus music lengths command.
var corpusMusicLengthsCmd = &cobra.Command{
	Use:   "lengths",
	Short: "Hold every separated record to the length its link states",
	Long: `Check the download is the recording the evidence names.

A wrong file measures without complaining. A lyric video pads the end, these
downloads pad the front, a live take measures another room, and one record here
carried the segue into the next song on the album and had eight seconds of it
measured as the bass. The length the source holds is the only thing that catches
any of that, which is why every entry carries one.

The file's own length decides it. Padding makes a download longer than its
source and nothing makes it shorter, so a file that matches is the right
recording however much quiet it carries, and the quiet beside it only ever
explains one that is long.

Reads the separated stems, so it costs seconds a record rather than a file read,
and it reads those rather than the downloads because the stem is a WAV and the
download is an MP3. ` + "`corpus music records`" + ` answers from the manifests alone and
says only whether a stem exists.

    toneharness corpus music lengths --corpus resources/music/bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		of, err := newClient().MusicLengths(cmd.Context(), musicCorpus)
		if err != nil {
			return err
		}

		return answer(cmd, of, cli.MusicLengths)
	},
}

func init() {
	corpusMusicCmd.AddCommand(corpusMusicLengthsCmd)
}
