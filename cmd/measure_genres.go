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
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/cli"
)

var measureGenresCorpus string

// measureGenresCmd represents the measure genres command.
var measureGenresCmd = &cobra.Command{
	Use:   "genres",
	Short: "Measure every genre against the players who do not play it",
	Long: `Measure what each genre sounds like, and what sets it apart.

A genre is where its records sit. The middle of a cluster is the least
distinctive thing in it, so what makes a genre recognisable is what separates it
from everything else rather than what its members share with recorded bass in
general. Each one is measured against the players who have none of it.

Not the same question as "corpus music genres", which counts what somebody wrote
down in the manifests. That is a file read. This needs the recordings, and takes
minutes per record.

A genre needs eight records from at least three players before anything may aim
at it. Under that, three records by one band is that band's sound wearing a
genre's name. One over it can still earn nothing, which is worth knowing: it
means the figures do not set that genre apart from the rest.

What this measures is packed into the binary by "just generate", so asking for a
genre costs no audio.

    toneharness measure genres --corpus resources/music/bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		got, err := newClient().MeasuredGenres(cmd.Context(), measureGenresCorpus)
		if err != nil {
			return err
		}

		return answer(cmd, got, cli.MeasuredGenres)
	},
}

func init() {
	measureCmd.AddCommand(measureGenresCmd)

	// One instrument, for the reason `measure players` takes one: a genre is
	// measured against the players who do not play it, and a guitar's centre of
	// gravity sits an octave above a bass guitar's. Pointed at the whole tree it
	// would compare bass genres against guitarists and earn every one of them
	// `dark`.
	measureGenresCmd.Flags().StringVar(&measureGenresCorpus, "corpus",
		filepath.Join("resources", "music", "bass"),
		"the tree of recordings, one directory of records per player")
}
