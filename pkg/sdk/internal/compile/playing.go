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
// It compensates on one axis and says so. `attack` is what the vocabulary has
// for the front of a note, and a compressor's Attack is the control that
// decides how much of it gets past. The other half of the difference is
// brightness, and the size of that is not compensated here, because nobody has
// measured it: the direction is not in doubt and a number nobody took is the
// guessing this project exists to remove.

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

// attackAxis is the one axis this compensates on.
const attackAxis = "attack"

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
	// Term is the word that was added. Empty when none was, which is either
	// nothing to compensate or an ask that already spoke for the axis.
	Term string `json:"term,omitempty"`
	// Said is the whole of it in a sentence. Empty when the two right hands
	// were the same, or when either side did not say.
	Said string `json:"said,omitempty"`
}

// compensated is Compensated plus the word, which stays inside.
type compensated struct {
	Word Word
	Said string
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
	spoken string,
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

	// A word somebody wrote for this axis wins, and this says it stood down.
	// Two terms on one axis cancel by design, so appending here would have
	// taken the ask's own word out with it: asking for fingers on a rig whose
	// ask already says audible-pick-attack moved the compressor nowhere.
	if spoken != "" {
		return compensated{
			Said: "you play with " + playing.Attack + " where the rig was " +
				"played with " + subject + ", and the ask already says " +
				spoken + ", so that stands rather than a second word on the " +
				"same axis, which would cancel it",
		}
	}

	term := nearer
	if theirs < mine {
		term = softer
	}

	return compensated{
		Word: Word{Term: term},
		Said: "the rig was played with " + subject + " and you play with " +
			playing.Attack + ", so " + term + " compensates for the front of " +
			"the note. The brightness the two also differ by is not " +
			"compensated, because nothing here has measured how much",
	}
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
