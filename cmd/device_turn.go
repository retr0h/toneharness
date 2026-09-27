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

	"github.com/retr0h/toneharness/pkg/sdk"
)

var (
	presetsTurnBlock  int
	presetsTurnParam  int
	presetsTurnValue  float64
	presetsTurnChoice int
	presetsTurnSwitch bool
	presetsTurnModel  int
	presetsTurnDirect bool
)

// deviceTurnCmd represents the presets turn command.
var deviceTurnCmd = &cobra.Command{
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

A device does not coerce, so the kind of value has to match the parameter.
` + "`--value`" + ` is a number on a dial, ` + "`--choice`" + ` is one of a
list such as a cabinet's microphone, and ` + "`--switch`" + ` is on or off
such as an amplifier's Bright. The wrong one is refused with the same error a
block that is not there gives.

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

		// Which of the three the device is sent is decided here, because a
		// device does not coerce: the value's tag is its type on the wire,
		// and a switch given 1.0 is refused with the same error it gives
		// for a block that is not there.
		client := newClient()
		said := any(presetsTurnValue)

		var err error

		switch {
		case cmd.Flags().Changed("choice"):
			said = presetsTurnChoice
			err = client.Choose(cmd.Context(), at, presetsTurnChoice)
		case cmd.Flags().Changed("switch"):
			said = presetsTurnSwitch
			err = client.Switch(cmd.Context(), at, presetsTurnSwitch)
		default:
			err = client.Turn(cmd.Context(), at, float32(presetsTurnValue))
		}

		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(),
			"\n  block %d parameter %d is now %v\n\n",
			presetsTurnBlock, presetsTurnParam, said)

		return err
	},
}

func init() {
	deviceCmd.AddCommand(deviceTurnCmd)

	f := deviceTurnCmd.Flags()
	f.IntVar(&presetsTurnBlock, "block", 0,
		"which block, by its position in the chain, counting from zero")
	f.IntVar(&presetsTurnParam, "param", 0,
		"which parameter, by its position in the model's own list")
	f.Float64Var(&presetsTurnValue, "value", 0,
		"what to set it to, in the parameter's own units")
	f.IntVar(
		&presetsTurnChoice,
		"choice",
		0,
		"what to set it to, for a parameter that is a list rather than a range, such as a cabinet's microphone",
	)
	f.BoolVar(&presetsTurnSwitch, "switch", false,
		"what to set it to, for a parameter that is a switch, such as an amplifier's Bright")
	f.IntVar(&presetsTurnModel, "model", 0,
		"the block's own model, or 1 for a cabinet fused into an amplifier")
	f.BoolVar(
		&presetsTurnDirect,
		"direct",
		true,
		"address the parameter the ordinary way; false reaches the value some blocks carry past their list",
	)

	// One kind of value, because a parameter has one type and the device
	// refuses the other two.
	deviceTurnCmd.MarkFlagsMutuallyExclusive("value", "choice", "switch")
	deviceTurnCmd.MarkFlagsOneRequired("value", "choice", "switch")

	// Fails only for a flag that does not exist, and it is defined above.
	_ = deviceTurnCmd.MarkFlagRequired("param")
}
