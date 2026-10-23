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

package catalog

// interval is every block that adds a pitch nobody played while Line 6 file it
// under something else.
//
// Keyed by model identifier because that is the only stable name a block has:
// Line 6 rename the labels between releases and `BasedOn` is prose. A rename
// here would be a silent miss, so a test walks every key against the catalog
// and fails on one that is no longer a model.
var interval = map[ModelID]bool{
	// Drives. An octave fuzz is a fuzz as far as the category goes, and the
	// device files five of its 61 drives that way.
	"HD2_DM4OctaveFuzz":      true, // Tycobrahe Octavia, octave up
	"HD2_DistTycoctaviaFuzz": true, // Tycobrahe Octavia again, the newer port
	"HD2_DM4SubOctFuzz":      true, // PAiA Roctave Divider, octave down
	"HD2_DistVitalDist":      true, // EarthQuaker Devices Life, octave up
	"Line6BronzeMaster":      true, // Maestro Bass Brassmaster, octave up

	// Modulation. A ring modulator multiplies the signal by a carrier and a
	// frequency shifter adds a constant to every partial. Both answer with
	// sums and differences, which are not the note that went in and not
	// harmonics of it either.
	"HD2_MM4RingModulator":          true,
	"HD2_RingModulatorAMRingMod":    true,
	"HD2_RingModulatorPitchRingMod": true,
	"HD2_MM4FrequencyShifter":       true,

	// Delay. The repeats are harmonised, so what comes back is a note nobody
	// played.
	"HD2_DelayHarmonyDelay": true,
}

// ChangesTheNotes reports whether this block puts energy at a pitch nobody
// played.
//
// The line is between a block that changes how a note sounds and one that
// changes which note it is. Distortion is on the near side however extreme:
// its output is harmonics of what went in, so the note is still the note. An
// octave fuzz, a pitch shifter, a harmoniser, a synth, a ring modulator and a
// frequency shifter are all on the far side, and so is a capo or a detune.
//
// This is the one thing a fingerprint cannot see. A block is measured alone
// through four figures, and an octave fuzz lands among the ordinary fuzzes on
// every one of them: of the 61 swept drives on an HX Stomp, Octave Fuzz reads
// a centroid of 213Hz and 22.0% harmonic energy against Ballistic Fuzz's
// 226Hz and 22.6%, which nothing in those numbers separates. Ranked against
// punk's measured corner, Octave Fuzz came first. So the refusal is stated
// rather than measured, and the figure that would see it does not exist yet:
// energy at half or twice the instrument's lowest note, which needs a sweep
// nobody has run.
//
// Two rules, because the device says it two ways.
//
// A whole category where the category is the answer. Every one of the 31
// models Line 6 group under pitch shifts, harmonises or synthesises, and the
// category's own documentation says so. Nothing in it is ever the nearest
// block to a tone.
//
// A named block where the category is not. The other ten are filed as a
// drive, a modulation or a delay, which is what they also are, and no field
// on this device marks what they do to the pitch.
//
// This refuses a block a build was about to choose on somebody's behalf. Gear
// an ask names by hand is a decision somebody made and never comes through
// here: an octave fuzz is the right answer when it is the pedal a player is
// documented with.
func (b Block) ChangesTheNotes() bool {
	return b.Category == CategoryPitch || interval[b.ID]
}
