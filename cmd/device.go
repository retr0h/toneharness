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

// devicesCmd represents the devices command.
// deviceCmd represents the device command.
var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Work with attached Helix hardware",
	Args:  cobra.NoArgs,
	Long: `Read and change what a Line 6 Helix-family device holds, over USB.

Everything here touches hardware. What a device is playing, which slots hold
what, loading and rearranging them, and moving one control while you listen.
Building a preset file is ` + "`presets`" + `, which needs no device at all.

Device access works on macOS. On other operating systems these commands say it
is not supported yet; describing, validating and writing presets works
everywhere.`,
}

func init() {
	rootCmd.AddCommand(deviceCmd)
}
