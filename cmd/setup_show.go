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

var setupShowClient clientFlags

// setupShowCmd represents the setup show command.
var setupShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Say what the setup holds",
	Long: `Read the setup back, as the commands that use it read it.

The slot map and the favourites first, because those two are the only record
of which presets on the pedal belong to this tool rather than to whoever was
holding it. A rebuild writes the slots it names and leaves every other slot
alone, so this is the answer to "which ones will change".

A favourite the slot map says nothing about is called out: it can be jumped to
but a cycle cannot tell it is the current one, because recognising where the
pedal is means matching what it is playing against the rig a slot holds.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		setupShowClient.setup = ownSetup(setupShowClient.setup)

		held, err := setupShowClient.client().Setup()
		if err != nil {
			return err
		}

		return answer(cmd, held, cli.Setup)
	},
}

func init() {
	setupCmd.AddCommand(setupShowCmd)

	setupShowCmd.Flags().StringVar(&setupShowClient.setup, "setup", "", setupUsage)
}
