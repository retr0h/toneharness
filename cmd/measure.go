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

// measureCmd represents the measure command.
var measureCmd = &cobra.Command{
	Use:   "measure",
	Short: "Measure what something actually sounds like",
	Args:  cobra.NoArgs,
	Long: `Turn sound into numbers, so a word about it can be checked.

Two subjects, and keeping them apart is the point. A recording is somebody
else's record, measured to describe how they play. A block is this device,
measured by pushing a known signal through it and listening to what comes back.

Nothing here decides what the numbers mean. "Warm" and "percussive" are
judgements two people disagree about; a centroid of 410Hz is not. Turning one
into the other is somebody else's job, and keeping them apart is what leaves
anywhere to stand when they disagree.`,
}

func init() {
	rootCmd.AddCommand(measureCmd)
}
