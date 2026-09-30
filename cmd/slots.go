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

// slotsCmd represents the slots command.
var slotsCmd = &cobra.Command{
	Use:   "slots",
	Short: "Work with the positions a setlist holds",
	Args:  cobra.NoArgs,
	Long: `Read and rearrange the slots of a setlist, on a device or in a file.

A slot is a position, and a setlist is somewhere those positions live. Both
backings answer the same questions, so every command here takes either: name a
file and it edits that file, name none and it reaches the attached device.

That is the seam rather than device-against-file, because the operation is the
same either way. What only ever means anything on live hardware is ` + "`device`" + `,
and building a preset document is ` + "`presets`" + `.`,
}

func init() {
	rootCmd.AddCommand(slotsCmd)
}
