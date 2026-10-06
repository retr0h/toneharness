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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

var (
	presetsCurrentClient clientFlags
	deviceCurrentOut     string
	deviceCurrentID      string
)

// deviceCurrentCmd represents the device current command.
var deviceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show what the device is playing right now",
	Long: `Show the preset the device is playing, with every control where it now sits.

The edit buffer rather than a slot, and that is the whole difference. ` +
		"`device turn`" + ` moves a control in the buffer without writing anything
back, so reading the slot it came from answers with the stored document and
makes it look as though nothing happened.

This is how a measurement says what it measured. A control's slope belongs to
the chain it was taken in: Treble on an amplifier into a 4x12 and the same
Treble into a 1x15 are two different numbers. A sweep that records the move
but not the chain records a number nobody can attribute later, so the sweep
runs this and keeps the answer beside the figures.

--out writes it as a document instead of printing it, which is how a chain tuned
by ear is kept. Everything moved with ` + "`device turn`" + ` lives in the buffer and
nowhere else, so this is the only thing that can save it: writing the slot back
would store what the slot already had, and reading the slot back would read as
though nothing happened.

No flash write either way. --out costs the device nothing, which is the point of
keeping the loop in the buffer.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		read, err := presetsCurrentClient.client().Current(
			cmd.Context(), sdk.FormatRig)
		if err != nil {
			return err
		}

		if deviceCurrentOut == "" {
			return answer(cmd, read, cli.Reading)
		}

		return keepCurrent(cmd, read, deviceCurrentOut, deviceCurrentID)
	},
}

func init() {
	deviceCmd.AddCommand(deviceCurrentCmd)

	f := deviceCurrentCmd.Flags()
	f.StringVar(&presetsCurrentClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&presetsCurrentClient.device, "device", "", deviceUsage)
	f.StringVar(&deviceCurrentOut, "out", "",
		"write what is playing as a document, instead of printing it")
	f.StringVar(&deviceCurrentID, "id", "",
		"the identifier the document states; read off the preset's name when absent")
}

// errNoCurrentID refuses a document with no identifier to be named by.
var errNoCurrentID = errors.New(
	"what is playing has no name to take an identifier from, so pass --id")

// named turns a preset's name into the shape an identifier takes.
//
// The same rule the rest of the project slugs by, spelled here because the
// package that owns it is internal to the SDK and a command may not reach in.
func named(
	of string,
) string {
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '-'
		}
	}, of)

	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}

	return strings.Trim(out, "-")
}

// keepCurrent writes what the device is playing as a document.
//
// The identifier is the document's name on disk and in `rigs list`, so it is
// asked for rather than guessed at: a preset called "Punk wip" would otherwise
// become a rig called `punk-wip`, which is a different rig from `punk` and would
// sit beside it forever.
func keepCurrent(
	cmd *cobra.Command,
	read sdk.Reading,
	at string,
	id string,
) error {
	if id == "" {
		id = named(read.Name)
	}

	if id == "" {
		id = named(read.ID)
	}

	if id == "" {
		return errNoCurrentID
	}

	f, err := os.Create(filepath.Clean(at))
	if err != nil {
		return fmt.Errorf("writing %s: %w", at, err)
	}

	err = tone.Write(f, tone.Spec{
		Schema: tone.SchemaToneSpec,
		Id:     id,
		Rig:    read.Rig,
	})
	if err != nil {
		return errors.Join(fmt.Errorf("writing %s: %w", at, err), f.Close())
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", at, err)
	}

	cmd.Printf("  wrote %s as %s\n", at, id)

	return nil
}
