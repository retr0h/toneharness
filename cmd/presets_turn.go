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

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/sdk"
)

var (
	presetsTurnBlock  int
	presetsTurnParam  int
	presetsTurnValue  float64
	presetsTurnModel  int
	presetsTurnDirect bool
)

// presetsTurnCmd represents the presets turn command.
var presetsTurnCmd = &cobra.Command{
	Use:   "turn",
	Short: "Move one control on what the device is playing",
	Long: `Move one control on the preset the device is playing.

What a hand does to a knob. The change lands in the preset in front of you and
is heard at once: nothing is written to a slot and no preset is selected.

This is the only way to change one control. Writing a preset into a slot does
not do it — a slot given a new document reads back as the new one and goes on
sounding like what it held before — and it would wear the device's storage out
to sweep a control that way.

A block is named by its position in the chain, counting from zero, and a
parameter by its position in that model's own list. Position is the only thing
that identifies either on the wire, so neither takes a name. ` + "`presets show`" +
		` lists a chain in order, and ` + "`catalog show`" + ` lists a model's parameters in
theirs.

The value is in the parameter's own units, not anything normalised. Most run
zero to one because that is genuinely their range; a cabinet's microphone
distance runs one to twelve inches.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		at := sdk.Address{
			Block:  presetsTurnBlock,
			Param:  presetsTurnParam,
			Model:  presetsTurnModel,
			Direct: presetsTurnDirect,
		}

		if err := newClient().Turn(
			cmd.Context(), at, float32(presetsTurnValue)); err != nil {
			return err
		}

		_, err := fmt.Fprintf(cmd.OutOrStdout(),
			"\n  block %d parameter %d is now %g\n\n",
			presetsTurnBlock, presetsTurnParam, presetsTurnValue)

		return err
	},
}

func init() {
	presetsCmd.AddCommand(presetsTurnCmd)

	f := presetsTurnCmd.Flags()
	f.IntVar(&presetsTurnBlock, "block", 0,
		"which block, by its position in the chain, counting from zero")
	f.IntVar(&presetsTurnParam, "param", 0,
		"which parameter, by its position in the model's own list")
	f.Float64Var(&presetsTurnValue, "value", 0,
		"what to set it to, in the parameter's own units")
	f.IntVar(&presetsTurnModel, "model", 0,
		"the block's own model, or 1 for a cabinet fused into an amplifier")
	f.BoolVar(&presetsTurnDirect, "direct", true,
		"address the parameter the ordinary way; false reaches the value some blocks carry past their list")

	// Fails only for a flag that does not exist, and these are defined above.
	_ = presetsTurnCmd.MarkFlagRequired("param")
	_ = presetsTurnCmd.MarkFlagRequired("value")
}
