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
package compile

import (
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// This file is the one compensation this project can make for the person
// rather than for the record.
//
// Every figure in the corpus was measured off somebody else's playing. A rig
// built from a picked recording is duller played with fingers, because a pick
// puts high-frequency attack into every note that fingers do not, and until
// this ran the difference was discovered at the first rehearsal. The ask could
// already say how the subject played and the Setup can now say how the person
// holding the instrument does, so the gap between them is a knob position.
//
// It compensates on two axes. `attack` is what the vocabulary has for the front
// of a note, and a compressor's Attack is the control that decides how much of
// it gets past. The second is where the middle of the range sits, and it took a
// measurement to earn: IDMT-SMT-Bass holds 468 notes played both ways on the
// same instrument at the same pickup setting, and a plectrum reads brighter in
// all 468. The figure is `pkg/sdk/audio/data/hands.json` and
// `resources/dry/README.md` says where the notes came from.
//
// What it moves is not what anybody expected. A pick does not add treble: the
// high band shifts 0.00 of the energy. It trades the bottom for the middle, 0.17
// of the energy out of the low band and 0.17 into the mid, which is why the axis
// this compensates on is `mids` and not a brightness control the vocabulary does
// not have.
//
// The spread is as wide as the mean, 0.14 against 0.17, so the direction is
// certain and the magnitude is not. That is an argument for one step sized by
// the corpus like every other term, and against an offset in hertz this file
// would have had to invent.

// hands ranks what sets the string moving by how much front of a note it makes.
//
// An order rather than a scale, because the gaps between these are not
// measured and writing them as numbers would claim they were. What the order
// rests on is what is touching the string: flesh, flesh and a plectrum
// together, a plectrum, and a string driven into the fretboard.
//
// Two share a place. A thumb is flesh like fingers, and the rank is about the
// onset rather than about the tone, so a thumb asking for a fingered rig's
// compensation is asking for none.
var hands = map[tone.Attack]int{
	tone.AttackFingers: 0,
	tone.AttackThumb:   0,
	tone.AttackHybrid:  1,
	tone.AttackPick:    2,
	tone.AttackSlap:    3,
}

// attackAxis and midsAxis are the axes this compensates on.
//
// Two, and each is checked against the ask on its own: a request that already
// says `audible-pick-attack` has spoken for the first and said nothing about the
// second, so standing both down together would throw away a compensation
// nothing contested.
const (
	attackAxis = "attack"
	midsAxis   = "mids"
)

// midsFrom is the word that answers a hand moving the mids, and the measurement
// that says it does.
//
// Taken from the committed figure rather than written here, so the direction
// cannot drift from what was measured. Absent is not a failure: a build on a
// checkout whose `hands.json` holds nothing compensates on attack alone, which
// is what every build did before the notes were measured.
func midsFrom(
	subject, mine string,
) (string, bool) {
	got, ok := audio.HandsBetween(subject, mine)
	if !ok {
		return "", false
	}

	return midsWord(got.Figures[audio.KeyMid])
}

// midsWord is which way a measured difference in the mids points.
//
// Split from the lookup so the decision can be read on its own. A figure nobody
// measured and a figure that measured zero are the same answer here, and the
// zero value of Moved collapses them into one check rather than two: there is
// nothing to compensate either way.
func midsWord(
	moved audio.Moved,
) (string, bool) {
	if moved.Mean == 0 {
		return "", false
	}

	// The sign is the whole of it, and it inverts once on the way through.
	//
	// HandsBetween(subject, mine) is what mine reads minus what theirs reads. The
	// preset's knobs were set to reproduce records made with their hand, so a
	// negative mean is my hand putting less energy in the mids than the preset
	// was built around: the answer is to push the mids up, not to scoop them.
	//
	// Worked: a rig for a player who used a plectrum, played by somebody using
	// fingers, reads -0.17. Fingers put less in the mids than the pick the rig
	// was voiced for, so `mid-forward` puts it back. The same relation the attack
	// axis already uses, where a rig played with a pick and a person playing with
	// fingers gets `audible-pick-attack` rather than `soft-attack`.
	if moved.Mean < 0 {
		return "mid-forward", true
	}

	return "scooped", true
}

// nearer and softer are the terms that close a gap on that axis.
//
// One word each way and no size. The vocabulary's attack terms are a scale on
// a compressor's Attack control, so reaching for `percussive` to mean "a wide
// gap" would have worked arithmetically and lied: percussive means the string
// against the fretboard, which is not what somebody playing a picked rig with
// fingers is asking for. The direction is all this knows, and the step comes
// from the corpus the way every other term's does.
const (
	nearer = "audible-pick-attack"
	softer = "soft-attack"
)

// Playing is how the person who will play this preset plays.
//
// Separate from Intent.Attack, which is how the subject played. Both are
// needed and they are different claims: one comes off a record and one comes
// off the Setup.
type Playing struct {
	// Attack is what sets the string moving. Empty is a Setup that did not
	// say, which compensates for nothing.
	Attack string
}

// Compensated is what was done about how this person plays, and why.
//
// Exported because it leaves the package: a knob moved on somebody's behalf
// has to be visible before they plug in, and this is the only decision here
// that comes off the Setup rather than off the ask.
type Compensated struct {
	// Terms are the words that were added, one per axis compensated. Empty
	// when none were, which is either nothing to compensate or an ask that
	// already spoke for every axis this would have.
	//
	// A list since 2026-10-01, when the mids axis was measured and a hand
	// started answering for two. It was one word, and a second axis arriving
	// would have been reported as the first one silently.
	Terms []string `json:"terms,omitempty"`
	// Said is the whole of it in a sentence. Empty when the two right hands
	// were the same, or when either side did not say.
	Said string `json:"said,omitempty"`
}

// termsIn is the words a compensation added, as the names it leaves the package
// under.
func termsIn(
	words []Word,
) []string {
	if len(words) == 0 {
		return nil
	}

	out := make([]string, 0, len(words))
	for _, w := range words {
		out = append(out, w.Term)
	}

	return out
}

// compensated is Compensated plus the words, which stay inside.
type compensated struct {
	// Words is one per axis compensated, which is none, one or two.
	Words []Word
	Said  string
}

// compensate turns the gap between how the subject played and how this person
// plays into a word.
//
// A word rather than a move, so everything downstream is what it already was.
// `demand` will pull a compressor into a chain that has none because the word
// asked for it, `move` will decide how far the term travels from the corpus
// rather than from a guess here, and the run reports it beside every other
// word. One place decides the compensation and nothing else learns a new rule.
//
// Nothing is claimed when either side is unstated, when the two are the same,
// or when a technique arrives that the contract does not carry: a Setup naming
// something this does not rank is a Setup this cannot compare, and inventing a
// rank for it would be worse than saying nothing.
func compensate(
	subject string,
	playing Playing,
	spoken, spokenMids string,
) compensated {
	theirs, ok := hands[tone.Attack(subject)]
	if !ok {
		return compensated{}
	}

	mine, ok := hands[tone.Attack(playing.Attack)]
	if !ok {
		return compensated{}
	}

	if theirs == mine {
		return compensated{}
	}

	out := compensated{}

	// Per axis, because an ask speaks for one at a time. A word somebody wrote
	// wins and this says it stood down: two terms on one axis cancel by design,
	// so appending anyway would take the ask's own word out with it, and asking
	// for fingers on a rig whose ask already says audible-pick-attack moved the
	// compressor nowhere.
	said := make([]string, 0, 2)

	term := nearer
	if theirs < mine {
		term = softer
	}

	if spoken != "" {
		said = append(said, "the front of the note is left to "+spoken+", which "+
			"the ask already says: a second word on that axis would cancel it")
	} else {
		out.Words = append(out.Words, Word{Term: term})
		said = append(said, term+" compensates for the front of the note")
	}

	// The second axis, and it is measured rather than asserted. Absent where
	// nobody has measured these two hands against each other, which is the
	// ordinary answer for every pair but one.
	if mids, ok := midsFrom(subject, playing.Attack); ok {
		if spokenMids != "" {
			said = append(said, "the mids are left to "+spokenMids+", which the "+
				"ask already says")
		} else {
			out.Words = append(out.Words, Word{Term: mids})
			said = append(said, mids+" compensates for the mids, which is where a "+
				"measured hand moves the energy rather than into the treble")
		}
	}

	out.Said = "the rig was played with " + subject + " and you play with " +
		playing.Attack + ", so " + strings.Join(said, ", and ")

	return out
}

// spokenFor is the word an ask already uses for an axis, if any.
//
// Sorted by term so the answer does not depend on the order the ask listed
// them, for the same reason axesOf sorts: a rig naming two words for one axis
// would otherwise report a different one each run.
func spokenFor(
	words []Word,
	axis string,
) string {
	said := make([]string, 0, len(words))

	for _, w := range words {
		axes, ok := axesOf(w.Term)
		if !ok {
			continue
		}

		for _, a := range axes {
			if a == axis {
				said = append(said, w.Term)

				break
			}
		}
	}

	if len(said) == 0 {
		return ""
	}

	sort.Strings(said)

	return said[0]
}
