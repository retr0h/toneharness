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
	toneTuneID       string
	toneTuneGenre    string
	toneTuneCorpus   string
	toneTuneDry      string
	toneTuneSeconds  float64
	toneTuneTries    int
	toneTunePasses   int
	toneTuneTakes    int
	toneTuneNudge    float64
	toneTuneOut      string
	toneTuneAsk      string
	toneTuneHardware string
	toneTuneHeadroom float64
	toneTuneVolume   int
	toneTuneClient   clientFlags
)

// toneTuneCmd represents the tone tune command.
var toneTuneCmd = &cobra.Command{
	Use:   "tune",
	Short: "Solve a chain's controls for what a genre measures as",
	Long: `Put a curated rig in front of the device and turn its dials until what
comes back measures where the target is.

Nothing here searches. One amplifier with nine controls at five positions each
is 1,953,125 combinations, which is 226 days of measuring, so this measures what
each control does on its own, stacks those slopes into a matrix and solves the
matrix for the moves that close the gap. Measuring is linear in the number of
controls and solving is arithmetic.

Not every control is a dial. A cabinet's microphone is a list of twelve, and the
fourth does not sit between the third and the fifth in any sense a slope
describes, so lists are compared rather than solved: every setting is tried, the
nearest is kept, and the dials are then solved from there. Twelve settings is
twelve readings, which is cheaper than the sweep of one dial in a long chain.

Five things worth knowing before reading the output.

The model is local. A slope is true near where it was read and drifts away from
it, so one solve overshoots and the answer is another pass from wherever the
last one landed rather than a better model. Three to five passes is usual.

The target is a point with tolerances rather than a point. A genre's records
give a middle and a spread on every figure, so an axis its records agree about
has to be hit and one they disagree about need not be precise. That is what
makes a genre an easier target than it sounds.

A choice is revisited. The nearest setting is the nearest before any dial has
moved, which is not the same question as which setting a solve can finish from,
so a run that falls short backs up to the next-nearest and solves again.
` + "`--tries`" + ` is how many of those it is worth, and each costs a whole
convergence.

Nothing is stored. Every move is a live edit on what the pedal is playing, so
the answer lasts until the next preset is selected and no flash is written.
` + "`presets compile`" + ` is how a result is kept.

And it can fail honestly. A chain that cannot reach a target converges to a
residual larger than the noise floor and then stops improving, and that is
reported as which axes were met and by how far the rest missed. The gear being
wrong for the sound is a real answer to somebody who owns that gear.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		client := toneTuneClient.client()

		return cli.Tune(cmd.Context(), cmd.OutOrStdout(), cli.TuneOptions{
			Client:   client,
			Genres:   client,
			ID:       toneTuneID,
			Genre:    toneTuneGenre,
			Corpus:   toneTuneCorpus,
			Dry:      toneTuneDry,
			Seconds:  toneTuneSeconds,
			Passes:   toneTunePasses,
			Tries:    toneTuneTries,
			Takes:    toneTuneTakes,
			Headroom: toneTuneHeadroom,
			Volume:   toneTuneVolume,
			Nudge:    toneTuneNudge,
			Out:      toneTuneOut,
			Ask:      toneTuneAsk,
			Hardware: toneTuneHardware,
		})
	},
}

func init() {
	toneCmd.AddCommand(toneTuneCmd)

	f := toneTuneCmd.Flags()
	f.StringVar(&toneTuneID, "id", "", "the curated rig to tune")
	f.StringVar(&toneTuneGenre, "genre", "",
		"what to aim it at, by the name the corpus tags records with")
	f.StringVar(&toneTuneCorpus, "corpus",
		"resources/music/bass", "the recordings the target is measured from")
	f.StringVar(&toneTuneDry, "dry", "resources/dry/bass-di.wav",
		"the reference recording to push through the chain")
	f.Float64Var(&toneTuneSeconds, "seconds", 4,
		"how much of the reference to push through per reading")
	f.IntVar(&toneTunePasses, "passes", 5,
		"how many times to solve before giving up on converging")
	f.IntVar(&toneTuneTries, "tries", 2,
		"how many settings of a chain's lists to solve the dials from; "+
			"each past the first costs another convergence")
	f.IntVar(&toneTuneTakes, "takes", 3,
		"how many takes the noise floor is measured from")
	f.Float64Var(&toneTuneNudge, "nudge", 0.1,
		"how far a control is moved to read its slope, as a fraction of its range")
	f.StringVar(&toneTuneAsk, "ask", "",
		"a ToneSpec to append this round to as a correction; "+
			"without it nothing records what was asked for")
	f.StringVar(&toneTuneOut, "out", "",
		"where the tuned chain goes, as a plan; without it nothing is kept")
	f.StringVar(
		&toneTuneHardware,
		"hardware",
		"",
		"which attached audio device to push the signal through; two names separated by a comma play through the first and record from the second, which is how the measuring loop is opened rather than quieted",
	)
	f.Float64Var(
		&toneTuneHeadroom,
		"headroom",
		measuringHeadroom,
		"decibels to turn the chain's own output down by before measuring; the measuring lead makes the chain feed itself and no routing stops it, so this is the only thing between a figure and a squeal",
	)
	f.IntVar(&toneTuneVolume, "volume", measuringVolume,
		"where to put the computer's own output level before measuring, 0 to 100; "+
			"it is a tone control rather than a level control, because an "+
			"amplifier's distortion depends on how hard it is driven")
	f.StringVar(&toneTuneClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&toneTuneClient.device, "device", "", deviceUsage)

	_ = toneTuneCmd.MarkFlagRequired("id")
	_ = toneTuneCmd.MarkFlagRequired("genre")
}
