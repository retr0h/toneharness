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

	"github.com/retr0h/toneharness/pkg/sdk"
)

// configCmd represents the setup command.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Work with the file describing what you have",
	Args:  cobra.NoArgs,
	Long: `Read and start the document that says what you own.

A config is the application's own file and the only one that is about the
person rather than the sound. A rig says what gear makes a noise; a setup says
which pedal you have it on, which bass you are holding, where each rig lives on
that pedal and which slots you reach for. It changes when you buy something
rather than when you want to sound different, which is why it is a file of its
own and not part of every request.

It lives at ` + "`$XDG_CONFIG_HOME/toneharness/config.yaml`" + `, and every command
that reads one falls back to there when ` + "`--setup`" + ` names nothing. So the
usual way to use it is to run ` + "`setup init`" + ` once and never name it again.`,
}

// ownConfig is the setup to read: the one named, or the person's own.
//
// The fallback is here rather than in the library on purpose. The SDK reads
// none of this tool's environment, so a Client built with no setup has none,
// which is what stops a test reading whatever the machine running it happens
// to have. Defaulting is a decision of the thing a person is talking to.
//
// Nowhere to look is not an error. A command that would have read a setup it
// cannot find reads none, which is what it did before any of this existed.
func ownConfig(
	named string,
) string {
	if named != "" {
		return named
	}

	at, err := sdk.ConfigPath()
	if err != nil {
		return ""
	}

	return at
}

func init() {
	rootCmd.AddCommand(configCmd)
}
