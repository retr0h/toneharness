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

package solve

import (
	"math"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// This file is the other half of the loop, and it is not a solve.
//
// A dial has a slope: move it a little, see what changed, and the ratio is a
// number the matrix can hold. A list has nothing of the kind. A cabinet's Mic
// is twelve microphones and the fourth does not sit between the third and the
// fifth in any sense a line describes, so "so much centroid per microphone" is
// a quantity with no referent, and a matrix row built from one is arithmetic on
// a number that means nothing.
//
// So a list is compared rather than solved. Twelve settings is twelve readings,
// which is cheaper than the sweep of a single dial in a chain of a dozen, and
// the comparison is exact rather than modelled: each setting's figures are what
// that setting actually produced.
//
// The two halves do not go in one system. They interleave: choose, then solve
// the dials from where the choice left the chain, because the choice changes
// every slope. A 1x15 and a 4x12 give the same Treble two different slopes, and
// two microphones in front of one speaker are the same kind of difference.

// Choice is one control that is a list rather than a dial.
//
// Deliberately close to Knob without embedding it. The overlap is the address
// and the label; what differs is the whole reason this type exists, since a
// Knob carries a range and a slope and neither is a thing a list has.
type Choice struct {
	// Block is the position in the chain, and Param the control's index in
	// that model's own list. Together they are what a live edit addresses.
	Block int
	Param int
	// Control is what the catalog calls it, carrying the model, for saying
	// what was chosen. A label for a person, not a key.
	Control string
	// Setting is the parameter's name on its own, as the catalog spells it,
	// because a path into a plan is built from it.
	Setting string
	// At is which setting it sits on now.
	At int
	// Options is every setting the control has, in the device's own order.
	// That order is not a scale: it is the order Line 6 listed them in.
	Options []int
}

// Where is this control's identity, the same one a Knob has.
func (c Choice) Where() Where { return Where{Block: c.Block, Param: c.Param} }

// Option is one setting of a list, and how far off the target it read.
type Option struct {
	// Value is the setting.
	Value int
	// Residual is what each axis was out by at this setting, in units of its
	// own tolerance, measured rather than predicted.
	Residual map[audio.Figure]float64
	// Worst is the axis this setting leaves furthest out.
	Worst float64
	// Arrived says every axis the target constrained was already inside its
	// tolerance at this setting, with no dial moved.
	Arrived bool
}

// Nearest ranks a list's settings by how near the target each one read.
//
// read is what the chain measured at each setting, in the units the aims are
// stated in. A setting missing from it is one that was never readable: it muted
// the chain, clipped the converters, or the device refused it, and a figure
// computed from any of those describes the noise or the clipping rather than
// the microphone. Left out rather than scored, because one clipped microphone
// of twelve read a centroid of 4,471Hz where the other eleven sat between 126
// and 147, and ranked on that it would have won every target asking for
// brightness.
//
// Ranked on the axis each setting leaves furthest out, which is the measure the
// rest of the loop reports and stops on. Summing the axes instead would rank a
// setting that is slightly wrong everywhere above one that is right on
// everything but the axis the target cares about most, and "2.4 tolerances out"
// would then mean two different things in one run.
//
// Ties break on the setting, so two runs of one request answer the same way
// rather than however a map was walked.
func Nearest(
	aims map[audio.Figure]Aim,
	read map[int]map[audio.Figure]float64,
) []Option {
	out := make([]Option, 0, len(read))

	for value, got := range read {
		_, at := wanted(aims, got)

		out = append(out, Option{
			Value:    value,
			Residual: at.Residual,
			Worst:    worst(at.Residual),
			Arrived:  at.Arrived,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Worst != out[j].Worst {
			return out[i].Worst < out[j].Worst
		}

		return out[i].Value < out[j].Value
	})

	return out
}

// Runner is the next setting worth trying, across every list in a chain.
//
// What makes the two halves interleave rather than run once. A list is chosen
// before the dials are solved, so the choice is made on what the chain reads
// with its dials wherever the compiler left them, and the nearest setting then
// is not always the one the dials can be solved furthest from. When a solve
// stops short, the answer is to back up to another setting and solve again.
//
// Ordered across all the lists at once rather than per list, so a chain with a
// cabinet's microphone and an amplifier's mid frequency tries whichever
// runner-up read nearest rather than exhausting one control before touching the
// other. Every list keeps its own nearest setting except the one named here.
//
// Each entry is a control and the setting to put it on. The first is the
// second-best reading anywhere, the next the third-best, and so on, which is
// the order that spends a reading where it is most likely to pay.
type Runner struct {
	Where   Where
	Control string
	Option  Option
}

// Runners is every setting other than each list's nearest, in the order worth
// trying them.
func Runners(
	ranked map[Where][]Option,
	named map[Where]string,
) []Runner {
	var out []Runner

	for where, options := range ranked {
		// The first is the one already applied, so the alternatives start at
		// the second.
		for _, o := range options[min(1, len(options)):] {
			out = append(out, Runner{Where: where, Control: named[where], Option: o})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Option.Worst != out[j].Option.Worst {
			return out[i].Option.Worst < out[j].Option.Worst
		}

		if out[i].Where.Block != out[j].Where.Block {
			return out[i].Where.Block < out[j].Where.Block
		}

		if out[i].Where.Param != out[j].Where.Param {
			return out[i].Where.Param < out[j].Where.Param
		}

		return out[i].Option.Value < out[j].Option.Value
	})

	return out
}

// worst is how far the furthest axis is, in its own tolerances.
func worst(
	residual map[audio.Figure]float64,
) float64 {
	var out float64

	for _, off := range residual {
		out = math.Max(out, off)
	}

	return out
}
