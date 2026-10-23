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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
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
	// Run names the measuring session this reading came out of, as a key into
	// [Library.Runs].
	//
	// **What stops a resumed run relabelling earlier readings.** Everything
	// that decides what a figure is of — the rig, the volume, the device the
	// level was pinned on, the baseline it is read against — used to live on
	// the library and nowhere else, and `measure blocks --resume` carries
	// earlier blocks into a header this run has just built. So a library
	// assembled over several runs described every block in it with the last
	// run's settings, including blocks measured at another volume or through
	// another device.
	//
	// The campaign's own history is the argument. 19 of the committed curve
	// files predate `instrument` and `volume` entirely and read as unknown,
	// which is honest; a fingerprint taken in an earlier run read as whatever
	// the latest run claimed, which is not.
	//
	// Empty for a reading taken before this was recorded, which is all 614 in
	// the committed library. [Library.RunOf] answers for those from the
	// header, because that is what they were always implicitly claiming, and
	// a caller that needs to know the difference reads this field.
	Run string `json:"run,omitempty"`
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
	// Volume is the computer's own output level while these were taken, on the
	// platform's own 0 to 100 scale, and -1 where the platform would not say.
	//
	// Recorded because it is a tone control rather than a level control, which
	// is the thing about it that is easy to get wrong. An amplifier's
	// distortion depends on how hard it is driven, so a campaign taken at a
	// different setting measures every amplifier as a different amplifier, not
	// as the same one louder.
	//
	// It only reaches the signal at all on a rig that plays the reference out
	// of the computer's own output, which is the rig that opens the measuring
	// loop. On the one-device rig the pedal plays and this is not in the path.
	//
	// Not comparable between machines: the scale is the slider's, not decibels,
	// and two laptops at 38 put out different levels. Baseline is what makes
	// two libraries comparable. This is what makes one reproducible.
	Volume int `json:"volume"`
	// Rig is the audio devices the reference went out of and came back in on,
	// as `--hardware` received them, and Open says the two were different so
	// the chain could not hear itself.
	//
	// **The field that decides whether a reading may be compared with
	// another**, and the one this file went without. AGENTS.md states the rule
	// it was missing: a figure measured on one device is a figure about that
	// device, say which. The two rigs disagree even about a block neither
	// refused, an Ampeg SVT reading 12.7% of its energy in the mids on one and
	// 4.0% on the other, and the committed baseline of 155.27Hz at -58.55dB
	// sits thirty decibels from what the open loop settles at. That is almost
	// certainly because these came off the closed rig, and the file could not
	// say so.
	//
	// Empty in a library written before this was recorded, which is not the
	// same as a closed loop: it is a rig nobody wrote down.
	Rig  []string `json:"rig,omitempty"`
	Open bool     `json:"open,omitempty"`
	// PinnedOn is the output device the level was pinned on, which is the one
	// `Volume` reached.
	//
	// New with the setting that moves it, and the one that silently invalidates
	// everything else. A Mac with its default output on Bluetooth headphones
	// pins those while the jack feeding the pedal sits wherever it was left,
	// which is how a run came to read 66dB of loss with the cabling correct.
	// A library written in that state records its volume honestly and describes
	// a signal path nobody measured.
	PinnedOn string `json:"pinned_on,omitempty"`
	// Baseline is the empty loop, measured first.
	//
	// Without it a figure says nothing. 95 Hz is not what an equaliser does
	// to a bass, it is what the bass already was, and an equaliser flat at
	// its defaults passes it straight through.
	Baseline Figures `json:"baseline"`
	// Isolated says every reading was taken with one block in the chain.
	// Nothing measured in a chain may be compared with these.
	Isolated bool `json:"isolated"`
	// Runs is every measuring session this file holds readings from, keyed by
	// [Run.Key].
	//
	// The header above describes the run that wrote the file last, which is
	// what it has always described and what nothing used to say. This is the
	// rest of them. A library assembled in one run holds one entry and every
	// block points at it; one assembled over several holds several, and a
	// reader can tell which readings may be compared with which.
	//
	// Absent in a library written before this was recorded, which is not the
	// same as a library from one run: it is a file that cannot say.
	Runs map[string]Run `json:"runs,omitempty"`
	// Blocks is the readings, by model identifier.
	Blocks map[string]Block `json:"blocks"`
}

// Run is one measuring session: the rig it was taken on, and the settings that
// decide what its readings are of.
//
// Every field here is one a figure is only a figure under. The device the level
// was pinned on is the one that silently invalidates the rest: a Mac with its
// default output on Bluetooth headphones pins those while the jack feeding the
// pedal sits wherever it was left, and a run came to read 66dB of loss with the
// cabling correct.
//
// Not the device or whether the readings were isolated. Both are refused at
// resume rather than recorded per run, because a library mixing two pedals or
// mixing isolated with in-chain readings is one nothing may rank inside at all.
type Run struct {
	// Reference identifies the signal every reading in this run was taken
	// against. A different one invalidates the lot.
	Reference Reference `json:"reference"`
	// Instrument is what was played through it, as the corpus names one.
	Instrument string `json:"instrument,omitempty"`
	// Headroom is how far the chain's own output was turned down, in decibels,
	// and Volume the computer's own output level on the platform's 0 to 100
	// scale.
	Headroom float64 `json:"headroom,omitempty"`
	Volume   int     `json:"volume"`
	// Rig is the audio devices the reference went out of and came back in on,
	// Open says the two were different so the chain could not hear itself, and
	// PinnedOn is the output device the level was pinned on.
	Rig      []string `json:"rig,omitempty"`
	Open     bool     `json:"open,omitempty"`
	PinnedOn string   `json:"pinned_on,omitempty"`
	// Baseline is the empty loop, measured first. Every figure in this run is
	// read against this one and against no other run's.
	Baseline Figures `json:"baseline"`
}

// Key identifies a run by what makes its readings comparable.
//
// A digest of the run's own fields rather than a clock or a counter, so two
// sessions taken on the same rig at the same settings against the same baseline
// collapse into one entry. That is the right answer: as far as anything
// downstream is concerned they are one run, and a timestamp would split them
// into two the reader then has to decide are equivalent.
//
// Twelve hex characters, which is plenty for the handful of runs a library ever
// holds and short enough to read in a file.
func (r Run) Key() string {
	// Through the same encoder the file uses, so two runs that serialise alike
	// key alike and a field added to Run reaches the key without being added
	// here too.
	//
	// The error is dropped because Run holds strings, numbers and a []string
	// and none of them can fail to encode. A branch for it would be one
	// nothing can reach and nobody has checked.
	raw, _ := json.Marshal(r)

	sum := sha256.Sum256(raw)

	return hex.EncodeToString(sum[:])[:runKey]
}

// runKey is how many hex characters of the digest a run is named by.
const runKey = 12

// RunOf is the measuring session one reading came out of.
//
// From the header where the block names no run, because that is what such a
// reading was always implicitly claiming and the committed library is 614 of
// them. False says the claim came from the header rather than from the reading,
// so a caller reporting which rig a figure is about can say which it has.
func (l Library) RunOf(
	block Block,
) (Run, bool) {
	if got, ok := l.Runs[block.Run]; ok && block.Run != "" {
		return got, true
	}

	return Run{
		Reference:  l.Reference,
		Instrument: l.Instrument,
		Headroom:   l.Headroom,
		Volume:     l.Volume,
		Rig:        l.Rig,
		Open:       l.Open,
		PinnedOn:   l.PinnedOn,
		Baseline:   l.Baseline,
	}, false
}

// Header is the library's own fields read as the run that wrote it last.
//
// What `measure blocks` stamps the readings it takes, so the header and the run
// entry cannot describe different things.
func (l Library) Header() Run {
	got, _ := l.RunOf(Block{})

	return got
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
	// Harmonics is how much energy sits above the fundamental.
	//
	// Separate from the band shares because it is the one figure beyond them
	// that both sides of a comparison measure the same way. See [Spectral].
	Harmonics float64
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
//
// **Harmonics is weighed, which is task 276.** It is the fourth figure both
// sides of a comparison measure the same way: a record and a block go through
// the same `audio.Measure`, and unlike transient and decay the padding cannot
// move it, because a share of the energy above the fundamental does not change
// when silence is added to either end. The blocks span it from 0 to 86 of a
// hundred and four of the eleven genres displace it past what the figure can
// claim, blues-rock and grunge by six points of it.
//
// Weighed at one like a band share, because it is reported on the same scale:
// a block's reading and a genre's displacement are both hundredths of the
// energy, so the two are already comparable without a scale of their own.
func Spectral() Weights {
	return Weights{
		Centroid: 1.0 / 1000.0, Low: 1, Mid: 1, High: 1, Harmonics: 1,
	}
}

// SpectralFor is [Spectral] with the axes an instrument cannot use dropped.
//
// A bass stem holds almost nothing above 2kHz, which is where the high band
// begins. Across the eleven genres this corpus can aim at, the high band
// displaces by exactly 0.00 for every one of them, so weighing it ranks
// candidates on a figure none of them differ by and spends a quarter of the
// distance on noise.
//
// The low band goes with it, for arithmetic rather than for measurement: the
// three shares sum to one, so with the high band dead the low band carries
// nothing the mid band has not already said, and counting both weighs the same
// difference twice.
//
// So a bass is ranked on its centroid, its midrange and its harmonics. Those
// are the figures its records actually differ by: across those eleven genres
// the centroid spans 109Hz to 174Hz, the mid band 0.00 to 0.11 and harmonics
// 0.11 to 0.19, and every other axis is flat or unreadable.
//
// Harmonics is kept rather than dropped, which is task 276 and the reason
// this reads three figures where it used to read two. Four of the eleven
// genres displace it past what the figure can claim, blues-rock and grunge by
// six points of a hundred, and nothing about a bass makes it dead the way the
// high band is.
//
// An instrument this does not know gets [Spectral]. Only the bass corpus has
// been measured, so only the bass case is a reading rather than a guess, and
// guessing that a guitar's high band is dead would be wrong: a guitar's centre
// of gravity sits an octave up.
func SpectralFor(
	instrument string,
) Weights {
	out := Spectral()

	if strings.EqualFold(instrument, "bass") {
		out.Low, out.High = 0, 0
	}

	return out
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

	// Harmonics, where both sides carry a reading.
	//
	// Skipped rather than read as zero where either does not. The three
	// pointer figures were added after the first sweeps ran, so a reading
	// taken before that has none, and counting an absent one as nothing would
	// rank it as the furthest block on the device from any target asking for
	// harmonics at all.
	if mine, theirs, both := pair(got.Harmonics, want.Harmonics); both {
		apart := w.Harmonics * (mine - theirs)
		sum += apart * apart
	}

	return math.Sqrt(sum)
}

// pair is one figure from each side of a comparison, and whether both carry it.
//
// The three figures added after the first sweeps ran are pointers, because a
// reading taken before them has a gap rather than a zero: see [Figures]. So
// every comparison that reads one has the same two lines to write, and this is
// them written once.
func pair(
	mine, theirs *float64,
) (float64, float64, bool) {
	if mine == nil || theirs == nil {
		return 0, 0, false
	}

	return *mine, *theirs, true
}

// of is one figure by the key the corpus names it with.
//
// One owner for which field a key means, because three readers had the same
// switch written out and the fourth figure task 276 weighs would have made it
// four copies.
//
// Zero for a figure a reading does not carry, which is only ever one of the
// three added after the first sweeps ran. Within a single sweep every point
// carries the same set, so a curve whose points have no harmonics reads a
// span of zero and is passed over rather than scored on nothing: see
// [Curves.TowardFrom], which asks what a control reaches before it asks where
// to put it.
func (p Figures) of(
	key audio.Figure,
) float64 {
	switch key {
	case audio.KeyLow:
		return p.Low
	case audio.KeyMid:
		return p.Mid
	case audio.KeyHigh:
		return p.High
	case audio.KeyLevel:
		return p.Level
	case audio.KeyHarmonics:
		return orZero(p.Harmonics)
	case audio.KeyDynamics:
		return orZero(p.Dynamics)
	case audio.KeyLean:
		return orZero(p.Lean)
	case audio.KeyTransient:
		return orZero(p.Transient)
	case audio.KeyDecay:
		return orZero(p.Decay)
	default:
		return p.Centroid
	}
}

// orZero is a figure a reading may not carry, as a number.
//
// Zero for an absent one, which every caller handles by the span it leaves
// rather than by the value: see [Figures.of]. A displacement nobody asked for
// reads the same way, which is how nothing being asked reads everywhere here.
func orZero(
	at *float64,
) float64 {
	if at == nil {
		return 0
	}

	return *at
}
