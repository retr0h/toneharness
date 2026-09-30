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
)

// musicCorpus is the tree of recordings to read manifests from.
var musicCorpus string

// corpusMusicCmd represents the corpus music command.
var corpusMusicCmd = &cobra.Command{
	Use:   "music",
	Short: "What the corpus of recordings holds",
	Args:  cobra.NoArgs,
	Long: `Read what somebody wrote down about the records, without measuring them.

The corpus is recordings measured to describe how people play. The audio is
somebody else's and is not in this repository; what is committed is a manifest
per player, naming which songs were measured and linking each one.

These read the manifests. That is a file read, where measuring the same corpus
needs the audio and minutes of work per record, and it answers the questions
somebody growing a corpus actually asks: who is in it, what they played, and
whether a genre has enough behind it to be aimed at yet.

    toneharness corpus music genres --corpus resources/music/bass`,
}

func init() {
	corpusCmd.AddCommand(corpusMusicCmd)

	// Defaulted rather than required, because from a checkout there is one
	// answer. The instrument is still part of the path: a word is earned by
	// sitting clear of the other players, and a guitar's centre of gravity sits
	// an octave above a bass guitar's, so one tree per instrument is what makes
	// the comparison mean anything.
	corpusMusicCmd.PersistentFlags().StringVar(&musicCorpus, "corpus",
		filepath.Join("resources", "music", "bass"),
		"the tree of recordings, one directory of records per player")
}
