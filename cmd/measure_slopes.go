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

var (
	measureSlopesID       string
	measureSlopesSweeps   string
	measureSlopesDry      string
	measureSlopesHardware string
	measureSlopesFigure   string
	measureSlopesSeconds  float64
	measureSlopesNudge    float64
	measureSlopesHeadroom float64
	measureSlopesVolume   int
	measureSlopesClient   clientFlags
)

// measureSlopesCmd represents the measure slopes command.
var measureSlopesCmd = &cobra.Command{
	Use:   "slopes",
	Short: "Hold a committed sweep to the chain it is being used in",
	Long: `Read what each control does now, and print it beside what the committed sweep says.

There is a disagreement to settle. ` + "`tone reach`" + ` answers from the slopes under
resources/sweeps/ and says every shipped rig reaches every measured genre.
` + "`tone tune`" + ` reads its slopes live and does not agree: the same rig against the
same target stopped 5.3 tolerances out. One of them is wrong.

A slope is not a property of a control. The same Treble into a 4x12 and into a
1x15 are two different numbers, and a committed sweep was measured in whichever
chain it ran in. So the question is not whether the arithmetic is right, it is
whether a number measured there means anything here.

RATIO is the live slope over the committed one. Near 1 the sweep still
describes this chain. OPPOSITE is the one that matters: a control whose slope
changed sign is one the solver will push the wrong way, confidently, on every
pass.

One reading per control, which is what a tuning pass spends. A chain of a dozen
dials is about a minute and a half, and it holds the pedal throughout.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		return cli.Slopes(cmd.Context(), cmd.OutOrStdout(), cli.SlopesOptions{
			Client:       measureSlopesClient.client(),
			ID:           measureSlopesID,
			Sweeps:       measureSlopesSweeps,
			Dry:          measureSlopesDry,
			Hardware:     measureSlopesHardware,
			Figure:       measureSlopesFigure,
			Seconds:      measureSlopesSeconds,
			Headroom:     measureSlopesHeadroom,
			HeadroomTold: cmd.Flags().Changed("headroom"),
			Volume:       measureSlopesVolume,
			Nudge:        measureSlopesNudge,
		})
	},
}

func init() {
	measureCmd.AddCommand(measureSlopesCmd)

	f := measureSlopesCmd.Flags()
	f.StringVar(&measureSlopesID, "id", "", "the curated rig to read the slopes in")
	f.StringVar(&measureSlopesSweeps, "sweeps", "resources/sweeps/hx-stomp",
		"the committed readings to hold to the chain")
	f.StringVar(&measureSlopesDry, "dry", "resources/dry/bass-di.wav",
		"the reference recording to push through the chain")
	f.StringVar(&measureSlopesFigure, "figure", "",
		"one axis to report; four of them without it")
	f.Float64Var(&measureSlopesSeconds, "seconds", 4,
		"how much of the reference to push through per reading")
	f.Float64Var(&measureSlopesNudge, "nudge", 0.1,
		"how far a control is moved to read its slope, as a fraction of its range")
	f.StringVar(
		&measureSlopesHardware,
		"hardware",
		"",
		"which attached audio device to push the signal through; two names separated by a comma play through the first and record from the second, which is how the measuring loop is opened rather than quieted",
	)
	f.Float64Var(
		&measureSlopesHeadroom,
		"headroom",
		measuringHeadroom,
		"decibels to turn the chain's own output down by before measuring; the measuring lead makes the chain feed itself and no routing stops it, so this is the only thing between a figure and a squeal",
	)
	f.IntVar(&measureSlopesVolume, "volume", measuringVolume,
		"where to put the computer's own output level before measuring, 0 to 100; "+
			"it is a tone control rather than a level control, because an "+
			"amplifier's distortion depends on how hard it is driven")
	f.StringVar(&measureSlopesClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&measureSlopesClient.device, "device", "", deviceUsage)

	_ = measureSlopesCmd.MarkFlagRequired("id")
}
