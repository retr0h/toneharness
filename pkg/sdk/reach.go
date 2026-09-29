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

package sdk

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// ErrNoSweeps is a chain no block of which has ever been measured.
var ErrNoSweeps = errors.New("no block in this chain has been swept")

// ErrNothingToAimAt is a target that names no axis.
var ErrNothingToAimAt = errors.New("nothing to aim at")

// ReachAsk is which chain to ask about and what to aim it at.
type ReachAsk struct {
	// RigID is the curated rig, and Genre the target by the name the corpus
	// tags records with.
	RigID string `json:"rig"`
	Genre string `json:"genre"`
	// Corpus is the tree the target is measured from and Sweeps the tree
	// holding this chain's own readings. Both have defaults.
	Corpus string `json:"corpus,omitempty"`
	Sweeps string `json:"sweeps,omitempty"`
}

// Reaching is what a chain could do against a target, from readings already
// taken.
type Reaching struct {
	Rig   string `json:"rig"`
	Genre string `json:"genre"`
	// Readings is how many measurements the answer rests on, and Unswept the
	// blocks that have none.
	//
	// Both, because the answer is only as complete as what is behind it. A
	// chain half of which was never measured gives a swing missing whatever
	// that half could have contributed, and saying so is the difference
	// between a narrow answer and a wrong one.
	Readings int      `json:"readings"`
	Unswept  []string `json:"unswept,omitempty"`
	// Axes is every axis the target names, worst first.
	Axes []solve.Verdict `json:"axes"`
	// Worth is whether every axis is at least not ruled out, and Decides the
	// axis that settled it.
	Worth   bool          `json:"worth"`
	Decides solve.Verdict `json:"decides"`
	// Together is whether one set of control positions satisfies every axis at
	// once, which is the question a person means when they ask whether a rig
	// can sound like something.
	//
	// The axes are checked separately as well, and that rules nothing out: a
	// dial that can put any one figure where a target wants it is ordinary,
	// and it says nothing about putting nine there at the same time.
	Together bool `json:"together"`
}

// Reach says which axes of a target a chain could meet, without running.
//
// Tune answers the same question by running: three to five passes, one reading
// per control per pass, five minutes of real-time audio with the pedal held
// throughout, and a sentence at the end saying the chain will not get there.
// Every number that sentence rests on is already committed under
// resources/sweeps/, so this reads those instead and costs about a second.
//
// No device is opened and none is wanted. That is the point: it is cheap
// enough to ask before deciding whether the expensive thing is worth doing.
func (c *Client) Reach(
	ctx context.Context,
	in ReachAsk,
) (Reaching, error) {
	target, err := c.targetFor(ctx, in)
	if err != nil {
		return Reaching{}, err
	}

	// Compiled rather than read, because the chain decides the answer and only
	// the compiler knows which models a rig's gear names resolve to.
	made, err := c.Make(ctx, in.RigID,
		filepath.Join(os.TempDir(), "toneharness.reach.hlx"), ReplaceExisting)
	if err != nil {
		return Reaching{}, err
	}

	curves, unswept, err := swept(made.Plan, in.sweeps())
	if err != nil {
		return Reaching{}, err
	}

	if len(curves) == 0 {
		return Reaching{}, fmt.Errorf("%w: %s", ErrNoSweeps, in.RigID)
	}

	aims := solve.Aims(target, nil)
	if len(aims) == 0 {
		return Reaching{}, fmt.Errorf(
			"%w: %q measures as nothing", ErrNothingToAimAt, in.Genre)
	}

	reach := measured.Reaches(curves...)
	spans := inTargetScale(reach)
	axes := solve.Reachable(aims, spans)

	// And again with every axis solved together, which is the question
	// somebody is actually asking. Nine axes each reachable alone says almost
	// nothing about nine reachable at once: a cabinet's Distance can put the
	// centroid where a target wants it and can put the low band where the
	// target wants it, and those are two different positions of one dial.
	// Only the axes the sweeps read. An aim on a figure no reading carried
	// would be solved against a reading of nought, which is not where the
	// chain sits: it is the absence of a measurement, and spending every
	// control defending it answers a question nobody asked.
	measurable := solve.Only(aims, slices.Collect(maps.Keys(spans)))

	joint, err := solve.Best(
		knobsOf(curves), measurable, from(spans), jointPasses)
	if err != nil {
		// A chain no control of which moves any axis the target names. Said
		// through the per-axis answer rather than refused, because "no control
		// moves this" is exactly what somebody asked to be told.
		joint = solve.Result{}
	}

	for i := range axes {
		axes[i].Together = joint.Residual[axes[i].Figure]
	}

	decides, worth := solve.Worth(axes)

	return Reaching{
		Rig:      in.RigID,
		Genre:    in.Genre,
		Readings: readings(reach),
		Unswept:  unswept,
		Axes:     axes,
		Worth:    worth,
		Decides:  decides,
		Together: joint.Arrived,
	}, nil
}

// targetFor is what the genre's records measure as, middle and spread.
func (c *Client) targetFor(
	ctx context.Context,
	in ReachAsk,
) (audio.Across, error) {
	if in.Genre == "" {
		return audio.Across{}, ErrNothingToAimAt
	}

	// The embedded figures first, which `just generate` writes from the same
	// corpus. Measuring it again reads fifteen bass stems and takes 47
	// seconds, against milliseconds for everything else this does, and a
	// command whose whole point is being cheap enough to ask first cannot
	// cost most of a minute.
	//
	// Only where nobody named a corpus. A caller who did means that tree
	// rather than whatever was committed, and answering from the committed
	// one would ignore what they asked.
	if in.Corpus == "" {
		if got, ok := audio.ShippedGenre(in.Genre); ok {
			return got.Across, nil
		}
	}

	found, err := c.MeasuredGenres(ctx, in.corpus())
	if err != nil {
		// Named, because the tree is a default when nobody passes one and
		// "stat .: no such file or directory" says nothing about which
		// directory was looked in or that anybody chose it.
		return audio.Across{}, fmt.Errorf("reading %s: %w", in.corpus(), err)
	}

	for _, g := range found {
		if g.Slug == in.Genre || g.Name == in.Genre {
			return g.Across, nil
		}
	}

	return audio.Across{}, fmt.Errorf("%w: no records are tagged %q under %s",
		ErrNothingToAimAt, in.Genre, in.corpus())
}

// corpus and sweeps are where to read from, with the committed trees as the
// defaults so an agent naming neither still gets an answer.
func (a ReachAsk) corpus() string {
	if a.Corpus != "" {
		return a.Corpus
	}

	return filepath.Join("resources", "music", "bass")
}

func (a ReachAsk) sweeps() string {
	if a.Sweeps != "" {
		return a.Sweeps
	}

	return filepath.Join("resources", "sweeps", "hx-stomp")
}

// swept reads the committed readings for every block in a chain.
//
// A block with no readings is named rather than skipped in silence, because
// whatever it could have moved is missing from every number that follows. A
// file that is there and will not decode is an error instead: that is a
// different thing from one never taken, and treating it as missing would
// answer from a chain quietly short of a block.
func swept(
	made plan.Plan,
	dir string,
) ([]measured.Curves, []string, error) {
	var (
		out     []measured.Curves
		unswept []string
	)

	for _, b := range made.Blocks {
		at := filepath.Join(dir, string(b.Model)+".json")

		f, err := os.Open(at) //nolint:gosec // a path built from the catalog
		if err != nil {
			unswept = append(unswept, string(b.Model))

			continue
		}

		got, err := measured.LoadCurves(f)

		_ = f.Close()

		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", at, err)
		}

		out = append(out, got)
	}

	return out, unswept, nil
}

// inTargetScale puts what the sweeps measured into the units a target is
// stated in.
//
// Every end of the span, not only the middle: a sweep reports a band share as
// a percentage and a corpus reports the same share as a fraction, so a range
// converted at one scale and compared against a target at the other is out by
// a hundred. The band shares are exactly the figures nobody would notice that
// on.
func inTargetScale(
	reach map[audio.Figure]measured.Reach,
) map[audio.Figure]solve.Span {
	out := make(map[audio.Figure]solve.Span, len(reach))

	for key, got := range reach {
		out[key] = solve.Span{
			From:  asFraction(key, got.Mid),
			Low:   asFraction(key, got.Low),
			High:  asFraction(key, got.High),
			Swing: asFraction(key, got.Swing),
		}
	}

	return out
}

// asFraction is one reading in the units a corpus states it in.
func asFraction(
	key audio.Figure,
	got float64,
) float64 {
	switch key {
	case audio.KeyLow, audio.KeyMid, audio.KeyHigh, audio.KeyHarmonics:
		return got / perCent
	default:
		return got
	}
}

// jointPasses is how many times the joint bound re-solves from where clamping
// left it.
//
// Five, matching what the loop takes on hardware. The slopes are fixed here so
// it converges rather than wandering, and it stops as soon as a pass stops
// improving, so this is a ceiling rather than a count.
const jointPasses = 5

// knobsOf turns the committed sweeps into controls the solver can move.
//
// Every control that has a slope on some figure. A list is left out: it has no
// slope, so the matrix has no row to put it in, and what a list is worth is
// answered by comparing its settings rather than by arithmetic.
//
// Each control starts at the middle of its own range, because the readings
// behind the chain's position are the median across every setting swept. A
// control started where the compiler happens to leave it would be answering
// against a position nothing here measured.
func knobsOf(
	of []measured.Curves,
) []solve.Knob {
	var out []solve.Knob

	for block, curves := range of {
		for name, curve := range curves.Controls {
			if len(curve.Fits) == 0 {
				continue
			}

			slope := make(map[audio.Figure]float64, len(curve.Fits))
			for key, fit := range curve.Fits {
				slope[key] = asFraction(key, fit.PerTurn)
			}

			out = append(out, solve.Knob{
				Block:   block,
				Param:   len(out),
				Control: curves.Block + " " + name,
				Setting: name,
				At:      (curve.Span.Low + curve.Span.High) / 2,
				Low:     curve.Span.Low,
				High:    curve.Span.High,
				Slope:   slope,
			})
		}
	}

	return out
}

// from is where the chain sits, per axis.
func from(
	spans map[audio.Figure]solve.Span,
) map[audio.Figure]float64 {
	out := make(map[audio.Figure]float64, len(spans))
	for key, span := range spans {
		out[key] = span.From
	}

	return out
}

// perCent is what a sweep reports a share as, against the fraction a corpus
// reports the same share as.
const perCent = 100

// readings is the most any one figure was measured, which is how many
// measurements the answer rests on.
func readings(
	reach map[audio.Figure]measured.Reach,
) int {
	var out int
	for _, got := range reach {
		out = max(out, got.Readings)
	}

	return out
}
