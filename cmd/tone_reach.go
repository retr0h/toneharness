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
	toneReachID       string
	toneReachGenre    string
	toneReachCorpus   string
	toneReachDry      string
	toneReachHardware string
	toneReachSeconds  float64
	toneReachTakes    int
	toneReachNudge    float64
	toneReachHeadroom float64
	toneReachClient   clientFlags
)

// toneReachCmd represents the tone reach command.
var toneReachCmd = &cobra.Command{
	Use:   "reach",
	Short: "Say whether a chain could reach a target, without running",
	Long: `Ask whether this rig can sound like this before spending an afternoon on it.

` + "`tone tune`" + ` answers by running: three to five passes, one reading per control
per pass, and it applies every move it solves for. This reads the same chain
once and applies nothing. A minute and a half against five minutes, and the
chain is left where the compiler put it.

It measures rather than reading resources/sweeps/, and that is the whole
design. The first version of this answered from those committed readings and
cost a second. Every one of them was taken with its block alone, which is a
different signal: an SV Beast swept with no cabinet has a median centroid of
8,139Hz where the chain a rig builds reads about 144, and four of its eleven
controls move the centroid the other way. Nothing computed from them was about
the chain it was asked about.

Three numbers per axis, and they are different kinds of claim.

OUT BY is how far the chain sits from the target, in that axis's own
tolerances, read off the chain now.

ALONE is what that axis's own dials could do for it with the others ignored. It
flatters them: it assumes each slope holds across a whole range it was read
locally and that every dial pulls the same way. It refuses well and promises
badly, so an axis marked OUT OF REACH is out of reach and one inside is only
worth attempting.

TOGETHER is what is left once every axis is solved at once, which is the
question somebody actually means. A dial can put the centroid where a target
wants it and can put the low band where the target wants it, and those are two
different positions of one dial.

So the answer is never "this will work". It is "worth running", or "no
arrangement of these dials gets there, change the chain rather than the knobs",
and the second is the one that saves the afternoon.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pedal.claim()

		client := toneReachClient.client()

		return cli.Reach(cmd.Context(), cmd.OutOrStdout(), cli.ReachOptions{
			Client:   client,
			Genres:   client,
			ID:       toneReachID,
			Genre:    toneReachGenre,
			Corpus:   toneReachCorpus,
			Dry:      toneReachDry,
			Hardware: toneReachHardware,
			Seconds:  toneReachSeconds,
			Takes:    toneReachTakes,
			Headroom: toneReachHeadroom,
			Nudge:    toneReachNudge,
		})
	},
}

func init() {
	toneCmd.AddCommand(toneReachCmd)

	f := toneReachCmd.Flags()
	f.StringVar(&toneReachID, "id", "", "the curated rig to ask about")
	f.StringVar(&toneReachGenre, "genre", "",
		"what to aim it at, by the name the corpus tags records with")
	f.StringVar(&toneReachCorpus, "corpus", "",
		"recordings to measure the target from, instead of the figures shipped "+
			"with this binary; measuring is most of a minute")
	f.StringVar(&toneReachDry, "dry", "resources/dry/bass-di.wav",
		"the reference recording to push through the chain")
	f.Float64Var(&toneReachSeconds, "seconds", 4,
		"how much of the reference to push through per reading")
	f.IntVar(&toneReachTakes, "takes", 3,
		"how many takes the noise floor is measured from")
	f.Float64Var(&toneReachNudge, "nudge", 0.1,
		"how far a control is moved to read its slope, as a fraction of its range")
	f.StringVar(&toneReachHardware, "hardware", "",
		"which attached audio device to push the signal through")
	f.StringVar(&toneReachClient.rigs, "rigs", "",
		"a directory of rigs to use instead of yours, beside the built-in ones")
	f.Float64Var(
		&toneReachHeadroom,
		"headroom",
		measuringHeadroom,
		"decibels to turn the chain's own output down by before measuring; the lead from the pedal back to itself oscillates without it",
	)
	f.StringVar(&toneReachClient.catalog, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&toneReachClient.device, "device", "", deviceUsage)

	_ = toneReachCmd.MarkFlagRequired("id")
	_ = toneReachCmd.MarkFlagRequired("genre")
}
