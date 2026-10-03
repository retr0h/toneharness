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
)

// toneCmd represents the tone command.
var toneCmd = &cobra.Command{
	Use:   "tone",
	Short: "Say what you want to sound like, and what you have",
	Args:  cobra.NoArgs,
	Long: `A ToneSpec says what somebody wants to sound like. A Setup says what
they have to work with. Both are written by hand, in the words a person would
use, and neither carries a knob position.

They are the top of three layers, and the split is what makes the middle one
worth sharing:

    ask + Setup        what we mean, and what we have
          ↓            toneharness tone build
    rig                exact, resolved, deterministic, shareable
          ↓            toneharness presets compile
    .hlx               what the pedal eats

One document holds both halves. The rig names the exact gear, so two people
compiling one get the same preset. The ask does not: "punk, a bit darker"
resolves against a corpus and a library of measurements that both move, so the
same ask next month is a different rig, and that is why both are written down.

What a request cannot say is where a knob goes. That is what the tool works
out, and a number typed into an authored file is how a word came to move
Treble by a quarter of its range because somebody decided a quarter.`,
}

func init() {
	rootCmd.AddCommand(toneCmd)
}
