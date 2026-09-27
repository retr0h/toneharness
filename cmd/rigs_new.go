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
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk"
)

// errKindWithoutFrom refuses --kind on a rig that is not a copy.
var errKindWithoutFrom = errors.New(
	"--kind says what a copy is attributed to, so it needs --from")

var (
	rigsNewOptions sdk.NewRig
	rigsNewFrom    string
	rigsNewKind    string
	rigsNewCatalog string
)

// rigsNewCmd represents the rigs new command.
var rigsNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Scaffold a rig",
	Long: `Write a new rig, after checking the gear it names exists.

Every gear name is resolved against the device catalog before anything is
written. A rig naming an amplifier no device models is otherwise only
discovered when somebody tries to build from it, and by then the name has
usually been copied somewhere else too.

--from copies an existing rig instead, comments and citations included, and
records where it came from in extends. Nothing merges the two: the copy is a
whole rig and editing it does not touch the original. Use it for a rig that
departs from another, such as one song played differently from the rest.

The rig is written to your own rigs directory, where rigs list and
presets make find it, unless --dir names another. The output says which file
it wrote.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// --kind is what a copy is attributed to. A rig scaffolded from gear
		// is always an artist, so without --from there is nothing for it to
		// change, and taking it without a word would say it had.
		if rigsNewKind != "" && rigsNewFrom == "" {
			return errKindWithoutFrom
		}

		dir := rigsDir
		if dir == "" {
			own, err := userRigsDir()
			if err != nil {
				return err
			}

			dir = own
		}

		client := newClient(sdk.WithUserRigs(dir), sdk.WithCatalog(rigsNewCatalog))

		made, err := scaffolded(cmd.Context(), client)
		if err != nil {
			return err
		}

		return answer(cmd, made, cli.Scaffolded)
	},
}

func init() {
	rigsCmd.AddCommand(rigsNewCmd)

	f := rigsNewCmd.Flags()
	f.StringVar(&rigsNewOptions.ID, "id", "",
		"identifier, and the filename stem — lower case, hyphenated")
	f.StringVar(&rigsNewOptions.Name, "name", "", "the player or style")
	f.StringVar(&rigsNewOptions.Band, "band", "", "the group, where there is one")
	f.StringVar(&rigsNewOptions.Instrument, "instrument", "guitar",
		"guitar or bass — it decides which half of the catalog is eligible")
	f.StringVar(&rigsNewOptions.Amp, "amp", "",
		"real-world amplifier, such as \"Ampeg SVT\"")
	f.StringVar(&rigsNewOptions.Cab, "cab", "",
		"real-world cabinet; omit to take the amp's own pairing")
	f.StringSliceVar(&rigsNewOptions.Pedals, "pedal", nil,
		"real-world pedal, in signal order; repeat for more")
	f.StringVar(&rigsNewCatalog, "catalog", "",
		"a generated catalog to check against instead of the built-in one")
	f.StringVar(&rigsNewFrom, "from", "",
		"copy an existing rig by identifier, rather than naming gear")
	f.StringVar(&rigsNewKind, "kind", "",
		"what the copy is attributed to: artist, band, song, genre or sound")
	// Fails only for a flag that does not exist, and these are defined above.
	_ = rigsNewCmd.MarkFlagRequired("id")
	// A copy takes its gear from the rig it copies, so naming any is either a
	// mistake or a misunderstanding of what a copy is.
	rigsNewCmd.MarkFlagsOneRequired("from", "amp")
	rigsNewCmd.MarkFlagsMutuallyExclusive("from", "amp")
	rigsNewCmd.MarkFlagsMutuallyExclusive("from", "cab")
	rigsNewCmd.MarkFlagsMutuallyExclusive("from", "pedal")
	rigsNewCmd.MarkFlagsMutuallyExclusive("from", "band")
}

// scaffolded writes the rig the flags describe: a copy with --from, or one
// naming gear without it.
func scaffolded(
	ctx context.Context,
	client *sdk.Client,
) (sdk.Scaffolded, error) {
	if rigsNewFrom == "" {
		return client.Scaffold(ctx, rigsNewOptions)
	}

	// The report names what the copy holds: the copied rig's instrument and
	// gear rather than the --instrument and --amp flags, which a copy does not
	// read, and the copied rig's name unless --name gave another.
	return client.Extend(ctx, sdk.ExtendRig{
		From: rigsNewFrom,
		ID:   rigsNewOptions.ID,
		Name: rigsNewOptions.Name,
		Kind: rigsNewKind,
	})
}
