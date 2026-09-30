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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/sdk"
)

// rigsCmd represents the rigs command.
var rigsCmd = &cobra.Command{
	Use:   "rigs",
	Short: "Work with curated gear knowledge",
	Args:  cobra.NoArgs,
	Long: `A rig says which gear a player or style uses, in signal order, and
why each piece of it is believed to be there. The ToneSpec beside it says how
it should sound. Rigs name real-world gear, "Ampeg SVT", never a device model
identifier, so one rig serves every Helix device.

This is the only knowledge here that is ours. A device catalog is generated
from Line 6's files; rigs are written by people.

Your own rigs live in $XDG_DATA_HOME/toneharness/rigs, or in
~/.local/share/toneharness/rigs when that variable is unset. rigs new
writes there, and rigs list, rigs show, presets make and the MCP server
read them beside the built-in ones. One sharing an identifier or alias with a
built-in rig is used in its place. --dir names another directory, which is
read in place of yours, still beside the built-in ones.`,
}

var rigsDir string

func init() {
	rootCmd.AddCommand(rigsCmd)
	rigsCmd.PersistentFlags().StringVar(&rigsDir, "dir", "",
		"a directory of rigs to use instead of yours, beside the built-in ones")
}

// userRigsDir is where somebody's own rigs live.
//
// Under the data directory, resolved the way backups resolve the state
// directory, rather than wherever the command happened to run: a rig
// written into the working directory is one no later command can find.
func userRigsDir() (string, error) {
	if data := os.Getenv("XDG_DATA_HOME"); data != "" {
		return filepath.Join(data, "toneharness", "rigs"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding somewhere to keep rigs: %w", err)
	}

	return filepath.Join(home, ".local", "share", "toneharness", "rigs"), nil
}

// ownRigs is the option naming somebody's own rigs to read: the
// directory named, or theirs.
//
// With nowhere to keep them there are none, which is nothing to refuse a
// read over. Writing one is different, and rigs new asks for the directory
// itself.
func ownRigs(
	named string,
) sdk.Option {
	if named != "" {
		return sdk.WithUserRigs(named)
	}

	dir, err := userRigsDir()
	if err != nil {
		return sdk.WithUserRigs("")
	}

	return sdk.WithUserRigs(dir)
}
