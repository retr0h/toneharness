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

// Package measured is what the device actually did, rather than what anybody
// said it would do.
//
// Every number here was taken off an HX Stomp: a known bass recording pushed
// through one block at a time, and the result measured. Nothing in it is
// authored, and nothing in it belongs in Go source. A slope typed into a
// constant is exactly the guessing this replaced.
//
// It answers one question that cannot be answered at solve time. The matrix a
// request is solved with is measured fresh for the chain being tuned, because
// a slope belongs to its chain, so storing one per block would be rebuilding
// it before anything used it. What no amount of solving can work out is which
// of 661 blocks belongs in the chain to begin with, and that is what a
// fingerprint is for.
package measured

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// Figures are what a recording reads as.
//
// The same ones a record is described in, taken by the same code: the
// measuring scripts hand what came back to `toneharness measure --json` rather
// than computing it themselves. That is not tidiness. A second implementation
// in Python read the reference bass as 98.6% low at 95 Hz where this one says
// 93% at 138 Hz, which made every block incomparable with every record while
// both looked reasonable.
//
// Level is the exception and is measured beside them. A record's loudness is
// the mastering engineer's decision and says nothing about the playing; a
// block's is a property of the block, and a volume control has nothing else
// to move.
type Figures struct {
	// Centroid is the centre of gravity of the spectrum, in Hz.
	Centroid float64 `json:"centroid"`
	// Level is loudness, in dBFS.
	Level float64 `json:"level"`
	// Low, Mid and High are the share of energy in each band, as
	// percentages that sum to a hundred.
	Low  float64 `json:"low"`
	Mid  float64 `json:"mid"`
	High float64 `json:"high"`
	// Transient is how sharply notes start, and Decay how long they take to
	// die away. Either can be absent: a transient needs a note starting and
	// a decay needs one ending, and a reading holding neither has no answer
	// rather than an answer of zero.
	Transient *float64 `json:"transient"`
	Decay     *float64 `json:"decay"`
	// Dynamics is the loudest against the typical, in dB. Harmonics is how
	// much energy sits above the fundamental, and Lean whether the even
	// multiples of it or the odd ones carry more.
	//
	// Absent for the same reason Transient and Decay can be, and pointers for
	// the same reason. These three were added after the first sweeps ran, so
	// a file taken before that holds readings without them. As plain numbers
	// they decoded to zero and read back as measured, which is worse than a
	// gap: a control whose harmonics are zero at every position fits a
	// perfectly straight line through nothing, and reports it with total
	// confidence.
	Dynamics  *float64 `json:"dynamics"`
	Harmonics *float64 `json:"harmonics"`
	Lean      *float64 `json:"lean"`
}

// Block is one model, measured alone at its own defaults.
type Block struct {
	Figures

	// ID is the model identifier the device addresses it by.
	ID string `json:"id"`
	// Name is what a person calls it.
	Name string `json:"name"`
	// Category is what it does: amp, cab, drive and so on.
	//
	// The catalog's own type rather than a plain string, because it is the
	// catalog's vocabulary: a reading is filed under the category the block
	// was listed in, and the two drifting apart would put a cabinet in the
	// amplifier shortlist.
	Category catalog.Category `json:"category"`
	// Clipped marks a reading that hit the converters' ceiling. Its
	// spectrum is the clipping's rather than the block's, because flat tops
	// make harmonics that were never in the signal, so it reads as a bright
	// block and is not one.
	Clipped bool `json:"clipped"`
	// Refused says why a block has no reading, for the ones that will not
	// load alone. Some of what the catalog lists means nothing on its own.
	Refused string `json:"refused"`
}

// Measured returns whether this block has a reading worth using.
//
// A refusal has no figures at all, and a clipped reading has figures that
// describe the clipping. Neither should be ranked against a target, and both
// are kept so that a later run can see what happened rather than find a gap.
//
// A reading near the ceiling counts as clipped whatever the flag says, and
// that is not pedantry. Clipping is not a cliff at the number a guard picks:
// a converter shapes a signal well before it hard-clips, and the flat tops
// make harmonics that were never in the block. Fifteen readings sit within
// 3dB of the guard, and two of them are bass amplifiers. HD2_AmpSVT4Pro, an
// Ampeg SVT-4 Pro that a shipped rig names, reads 3.24% of its energy in the
// low band and 96.74% in the high. An Ampeg does not do that. The converters
// do.
//
// Left in the file rather than removed from it, the same way a clipped
// reading is: what happened is worth keeping and only what is ranked matters.
func (b Block) Measured() bool {
	return b.Refused == "" && !b.Clipped && b.Level <= nearCeiling
}

// nearCeiling is the level above which a reading is the converters' rather
// than the block's.
//
// Two decibels below full scale, against the half a decibel `measure blocks`
// writes Clipped with. That guard catches a signal already flat-topped and
// misses the one about to be.
//
// Two rather than more, because this is a floor under what is demonstrably
// broken and not a claim that everything quieter is sound.
//
// **It excludes nothing in the library shipped now**, and that is the point of
// it rather than a reason to remove it. The readings it was written for were
// taken at full output through a measuring lead the chain fed itself down: ten
// of them sat above this line, two were bass amplifiers, and one read 99.97% of
// its energy in the high band. Measured at -30dB nothing comes near the ceiling.
//
// It stays because headroom is a setting and settings get changed. A campaign
// run at full output would put those readings back, and this is what keeps them
// out of a ranking.
const nearCeiling = -2.0

// Library is every block a device was measured on.
type Library struct {
	// Device is which one answered.
	Device string `json:"device"`
	// Reference identifies the signal every reading was taken against. A
	// different one invalidates the lot, the way changing a record
	// invalidates the words derived from it.
	Reference Reference `json:"reference"`
	// Instrument is what was played through it, as the corpus names one:
	// "bass", "guitar".
	//
	// Every figure here is a figure about that instrument and nothing else. A
	// bass DI and a guitar DI do not sit in the same place on any axis, so a
	// guitar rig ranked against readings taken with a bass is ranked against
	// the wrong distribution. Until 2026-09-29 the only trace of it was the
	// reference recording's filename.
	Instrument string `json:"instrument,omitempty"`
	// Headroom is how far the chain's own output was turned down while these
	// were taken, in decibels.
	//
	// Kept because it decides whether a reading is of the chain at all. The
	// measuring lead makes the chain feed itself and enough gain around that
	// loop oscillates, so readings taken without it are of the squeal: every
	// one of this device's full amplifiers read a median 46.4% of its energy
	// above 2kHz where the preamps of the same circuits read 0.047%.
	Headroom float64 `json:"headroom,omitempty"`
	// Baseline is the empty loop, measured first.
	//
	// Without it a figure says nothing. 95 Hz is not what an equaliser does
	// to a bass, it is what the bass already was, and an equaliser flat at
	// its defaults passes it straight through.
	Baseline Figures `json:"baseline"`
	// Isolated says every reading was taken with one block in the chain.
	// Nothing measured in a chain may be compared with these.
	Isolated bool `json:"isolated"`
	// Blocks is the readings, by model identifier.
	Blocks map[string]Block `json:"blocks"`
}

// Reference is the signal every reading was taken against.
type Reference struct {
	File    string  `json:"file"`
	SHA256  string  `json:"sha256"`
	Seconds float64 `json:"seconds"`
}

// Load reads a library of measurements.
func Load(
	r io.Reader,
) (Library, error) {
	var lib Library

	body, err := io.ReadAll(r)
	if err != nil {
		return Library{}, fmt.Errorf("reading the measurements: %w", err)
	}

	if err := json.Unmarshal(body, &lib); err != nil {
		return Library{}, fmt.Errorf("decoding the measurements: %w", err)
	}

	if len(lib.Blocks) == 0 {
		return Library{}, fmt.Errorf("the measurements name no blocks")
	}

	return lib, nil
}

// Match is one block held against a target, with how far off it was.
type Match struct {
	Block

	// Distance is how far this block's figures sit from the target, in the
	// units Nearest was given. Zero is exact and larger is worse.
	Distance float64
}

// Nearest ranks the blocks of one category by how close they sit to a target.
//
// This is what turns "sound like this record" into a shortlist. A record and
// a block are measured by the same code into the same figures, so the ones
// that land nearest are the ones worth putting in a chain and tuning.
//
// Ranked rather than chosen, and the whole ranking comes back rather than a
// winner, because the nearest block is not always the right one: a chain has
// a shape, a person has preferences, and the tuning that follows can close a
// gap the ranking cannot see. What this refuses to do is guess.
//
// Blocks that refused to load or clipped are left out. Their figures describe
// a failure or the converters, and ranking them would put a bright-looking
// reading of clipping at the top of a list of bright amplifiers.
func (l Library) Nearest(
	category catalog.Category,
	want Figures,
	weigh Weights,
) []Match {
	out := make([]Match, 0, len(l.Blocks))

	for _, block := range l.Blocks {
		if block.Category != category || !block.Measured() {
			continue
		}

		out = append(out, Match{
			Block:    block,
			Distance: weigh.between(block.Figures, want),
		})
	}

	// By distance, then by identifier, so two blocks that measure the same
	// come back in the same order every run. A ranking that reshuffles is a
	// ranking nobody can act on twice.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}

		return out[i].ID < out[j].ID
	})

	return out
}

// Weights say how much each figure counts toward a distance.
//
// They exist because the figures are not in the same units and are not
// equally meaningful. A centroid is thousands of Hz and a band share is a
// percentage, so an unweighted distance is a distance in Hz with rounding
// attached. Worse, most of the difference between two bass tones is where the
// energy sits rather than how loud it is, and level is the one figure a knob
// can fix for nothing.
type Weights struct {
	Centroid, Level, Low, Mid, High float64
}

// Spectral weighs where the energy sits and ignores everything else.
//
// The default for choosing a block. Three things are left at zero and each
// for its own reason.
//
// Level, because a block that is right and quiet is right: the next block's
// level control fixes it for nothing, and weighing loudness ranks a loud
// wrong answer over a quiet correct one.
//
// Transient and decay, because the measuring loop pads its recording with a
// second of silence to catch a reverb's tail, and that moves both. It moves
// them the same way for every block, so comparing two blocks is fair, and it
// does not move a record at all, so comparing a block with a record is not.
//
// The centroid is scaled to per-thousand-Hz so it counts about as much as a
// band share rather than swamping all three of them.
func Spectral() Weights {
	return Weights{Centroid: 1.0 / 1000.0, Low: 1, Mid: 1, High: 1}
}

// between is the weighted distance from one set of figures to another.
func (w Weights) between(
	got, want Figures,
) float64 {
	sum := 0.0

	for _, d := range []struct{ by, from, to float64 }{
		{w.Centroid, got.Centroid, want.Centroid},
		{w.Level, got.Level, want.Level},
		{w.Low, got.Low, want.Low},
		{w.Mid, got.Mid, want.Mid},
		{w.High, got.High, want.High},
	} {
		apart := d.by * (d.from - d.to)
		sum += apart * apart
	}

	return math.Sqrt(sum)
}
