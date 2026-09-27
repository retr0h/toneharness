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

// corpusPresetsCmd represents the corpus presets command.
var corpusPresetsCmd = &cobra.Command{
	Use:   "presets",
	Short: "What a body of presets other people made says",
	Args:  cobra.NoArgs,
	Long: `Measurements over presets strangers made, including their mistakes.

The catalog says what a device can do. This says what people actually do with
it, which is a different question: Line 6 states a default Treble of 0.77 for the
Ampeg SVT's bright channel, and across the presets using it the median is 0.85.

Nothing here is authority, which is why every median comes with a spread. A
parameter everybody sets the same way is one this tool can be confident about;
one nobody agrees on belongs to the player.`,
}

func init() {
	corpusCmd.AddCommand(corpusPresetsCmd)
}
