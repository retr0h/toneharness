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

package audio

import "math"

// The keys a measurement is written under in a rig's evidence.
//
// A rig's `measured` block exists so a figure can be compared against the same
// figure taken from somewhere else. That comparison is only possible when both
// sides agree what to call things, so these names are fixed here rather than
// chosen per rig. The contract keeps the field open to any key, the way
// EvidenceKind is open, and this is what the tool writes.
//
// Every one is a number this package produces. A key naming something nothing
// measures would be a promise with nobody behind it.
// Figure names one of the things a reading is reported in.
//
// A type rather than a bare string, because this alphabet had been spelled
// out independently in three packages and had already drifted: the controls
// report printed four of the ten and silently left out the mid band, which
// was measured, stored and never shown. A defined type makes the next
// omission a compile error.
type Figure string

// Every figure there is. Nine of them describe a recording and are what a
// rig's evidence is written under; Level is the tenth and is only ever a
// block's, for the reason MeasuredKeys gives.
const (
	// KeyLow, KeyMid and KeyHigh are each band's share of the energy, 0 to 1.
	KeyLow  Figure = "low"
	KeyMid  Figure = "mid"
	KeyHigh Figure = "high"
	// KeyCentroid is the spectrum's centre of gravity, in hertz.
	KeyCentroid Figure = "centroid"
	// KeyTransient is how sharply notes start, 0 to 1.
	KeyTransient Figure = "transient"
	// KeyDecay is how long a note takes to fall to a quarter, in seconds.
	KeyDecay Figure = "decay"
	// KeyDynamics is the gap between loudest and typical, in decibels.
	KeyDynamics Figure = "dynamics"
	// KeyHarmonics is the share of energy above the fundamental, 0 to 1.
	KeyHarmonics Figure = "harmonics"
	// KeyLean is positive for even harmonics and negative for odd, -1 to 1.
	KeyLean Figure = "lean"
	// KeyLevel is loudness, in dBFS.
	//
	// Not among MeasuredKeys, and the reason is not an oversight. A record's
	// loudness is the mastering engineer's decision and says nothing about
	// the playing, so it is never written as evidence about a player. A
	// block's loudness is a property of the block, and a volume control has
	// nothing else to move, so a curve is reported in it.
	KeyLevel Figure = "level"
)

// Share reports a figure measured as a fraction of the whole.
//
// Which figures those are is stated once, here beside the constants whose own
// comments say "0 to 1", because a sweep reports the same figure as a
// percentage and a corpus as a fraction. Three packages had each written this
// list out for themselves, and an axis added to one and not the others is an
// axis silently off by a hundred.
func (f Figure) Share() bool {
	switch f {
	case KeyLow, KeyMid, KeyHigh, KeyHarmonics:
		return true
	default:
		return false
	}
}

// MeasuredKeys is every key a measurement is written under, in the order it
// is written.
//
// An order rather than a set, because the map a measurement comes back as has
// none, and evidence written twice from the same recording should be the same
// text both times. A diff that moves lines around hides the one line that
// changed.
func MeasuredKeys() []Figure {
	return []Figure{
		KeyLow, KeyMid, KeyHigh,
		KeyCentroid, KeyTransient, KeyDecay, KeyDynamics,
		KeyHarmonics, KeyLean,
	}
}

// Measured is what one recording measures as, keyed for a rig's evidence.
//
// Rounded to what the measurement can honestly claim. A centroid is reported
// to the hertz because the bins are about 10Hz apart, and a share to two
// places because the third moves with which windows the playing fell in.
// A measure a recording could not answer is left out rather than written as
// the number it would have been. An earlier version wrote all nine keys
// always, reasoning that a rig could then tell a figure of zero from a figure
// nobody took. That was backwards: what distinguishes them is the key being
// absent, and writing `decay: 2.95` for a note that never decayed claims a
// measurement where there was only the length of the recording.
// Keyed by plain strings rather than by Figure, because this map goes
// straight into a rig's evidence and the contract keeps that field open to
// any key. The conversion is the boundary between a vocabulary this package
// controls and a document anybody may write.
func (p Profile) Measured() map[string]float64 {
	out := map[string]float64{
		string(KeyLow):       to(p.Low, 2),
		string(KeyMid):       to(p.Mid, 2),
		string(KeyHigh):      to(p.High, 2),
		string(KeyCentroid):  to(p.Centroid, 0),
		string(KeyDynamics):  to(p.DynamicRange, 1),
		string(KeyHarmonics): to(p.Harmonics.Mid, 2),
		string(KeyLean):      to(p.EvenOdd.Mid, 2),
	}

	if p.Transient.Known {
		out[string(KeyTransient)] = to(p.Transient.Value, 2)
	}

	if p.Decay.Known {
		out[string(KeyDecay)] = to(p.Decay.Value, 2)
	}

	return out
}

// Measured is what several recordings measure as together, keyed for a rig's
// evidence.
//
// The middle of each range. The width is not written into a rig: evidence is
// attached per source, and a width belongs to a set of sources rather than to
// any one of them. What the width is for is deciding whether to trust the
// middle at all, which is a judgement made while reading the report rather
// than a number to carry.
func (a Across) Measured() map[string]float64 {
	out := map[string]float64{
		string(KeyLow):       to(a.Low.Mid, 2),
		string(KeyMid):       to(a.Mid.Mid, 2),
		string(KeyHigh):      to(a.High.Mid, 2),
		string(KeyCentroid):  to(a.Centroid.Mid, 0),
		string(KeyDynamics):  to(a.DynamicRange.Mid, 1),
		string(KeyHarmonics): to(a.Harmonics.Mid, 2),
		string(KeyLean):      to(a.EvenOdd.Mid, 2),
	}

	// Written when any recording answered, because a middle taken over three
	// of four records is still a measurement of those three. How many is in
	// the report rather than in the rig: evidence carries figures, and the
	// count is something to weigh them by while reading.
	if a.Transient.From > 0 {
		out[string(KeyTransient)] = to(a.Transient.Mid, 2)
	}

	if a.Decay.From > 0 {
		out[string(KeyDecay)] = to(a.Decay.Mid, 2)
	}

	return out
}

// Spreads is how far apart the records sit on each figure, beside the middle
// Measured reports.
//
// The pair is what makes a player or a genre a target rather than a point. An
// axis their records agree closely about is one an answer has to hit; an axis
// they disagree about is one it need not be precise on, and treating the two the
// same spends controls defending a figure the evidence never agreed on.
//
// Keyed the same way as Measured, and carrying the same two guards: a figure
// no recording answered is absent rather than zero.
func (a Across) Spreads() map[string]Spread {
	out := map[string]Spread{
		string(KeyLow):       a.Low,
		string(KeyMid):       a.Mid,
		string(KeyHigh):      a.High,
		string(KeyCentroid):  a.Centroid,
		string(KeyDynamics):  a.DynamicRange,
		string(KeyHarmonics): a.Harmonics,
		string(KeyLean):      a.EvenOdd,
	}

	if a.Transient.From > 0 {
		out[string(KeyTransient)] = a.Transient.Spread
	}

	if a.Decay.From > 0 {
		out[string(KeyDecay)] = a.Decay.Spread
	}

	return out
}

// to rounds a measurement to the places it can honestly claim.
func to(
	v float64,
	places int,
) float64 {
	scale := math.Pow(10, float64(places))

	return math.Round(v*scale) / scale
}
