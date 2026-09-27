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

import "github.com/spf13/cobra"

// corpusCmd represents the corpus command.
var corpusCmd = &cobra.Command{
	Use:   "corpus",
	Short: "Work with the bodies of evidence this is built on",
	Args:  cobra.NoArgs,
	Long: `Two bodies of evidence, and they answer different questions.

A corpus of presets says what people set their knobs to. A corpus of recordings
says what players actually sound like. Neither is authority and both are
somebody else's work, which is why every figure here comes with a spread or a
count rather than a verdict.

They are named rather than sharing the word, because "corpus" meant only the
presets for long enough that a reader could reasonably assume it still does.`,
}

func init() {
	rootCmd.AddCommand(corpusCmd)
}
