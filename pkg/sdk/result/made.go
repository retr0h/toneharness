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

package result

import (
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// Made is a preset built from a rig.
//
// The chain matters as much as the file. A generated preset is a set of
// decisions, and a wrong amp should be visible before anybody plugs in rather
// than after.
type Made struct {
	// Plan is the signal path that was built, realised for one device.
	//
	// Named for what it is rather than for `chain`, which since 2026-09-26 means
	// the portable half: gear in signal order, named the way a musician names it.
	// This holds model identifiers and DSP positions, so it is the other one.
	Plan plan.Plan `json:"plan"`
	// Added are the blocks nobody asked for. A rig names an amp; a chain is
	// four or five blocks, and a choice made on the player's behalf has to be
	// visible.
	Added []Added `json:"added"`
	// Unfamiliar are the character terms nothing defines. Said rather than
	// refused: a term moves no knob, so an unfamiliar one costs the preset
	// nothing, and refusing a build over a word would be refusing somebody
	// the right to describe a sound in their own words.
	Unfamiliar []Unfamiliar `json:"unfamiliar"`
	// Moved are the parameters a character term turned, and the terms that
	// turned nothing. A word a rig described itself with that moved no knob
	// is still something the rig said, and reporting only the ones that
	// worked would read as if the rest had.
	Moved []Moved `json:"moved"`
	// Playing is what was done about how this person plays, and why.
	//
	// Only ever set when a Setup said. It is the one decision here taken from
	// what somebody has rather than from what they asked for, so it is
	// reported on its own rather than folded into the words: a knob moved
	// because of a right hand is a different claim from one moved because of a
	// word.
	Playing Playing `json:"playing"`
	// Levelled is what was done to match another rig's loudness, and why.
	//
	// Beside Playing and for the same reason: it is a decision taken from
	// what somebody has rather than from what they asked for, so it is
	// reported on its own rather than folded into the words. A knob moved to
	// land on a level is a different claim from one moved because of a word.
	Levelled Levelled `json:"levelled"`
	// Path is the file that was written.
	Path string `json:"path"`
	// Preset is the document that was written, as it stood when it was written.
	//
	// Here because the plan beside it is not the whole preset: sections,
	// controllers and footswitches are applied to the document rather than to
	// the chain, so a caller that rebuilds a document from the plan gets a
	// preset missing every one of them. `rigs resolve` wants all of it, and
	// before this it got there by writing the file and reading it straight back.
	//
	// Not serialised. A preset is 4,400 lines of device JSON and no reader of
	// this answer has a use for it; a caller that wants the file reads Path.
	Preset *preset.Document `json:"-"`
}

// Playing is the compensation made for how somebody plays.
type Playing struct {
	// Terms are the words added on their behalf, one per axis compensated.
	// Empty when none were.
	Terms []string `json:"terms,omitempty"`
	// Said is the whole of it in a sentence, ready to print.
	Said string `json:"said,omitempty"`
}

// Levelled is what a build did to come out as loud as another rig.
//
// Here rather than in compile because compile already answers with this
// package's types and the dependency only runs one way.
type Levelled struct {
	// Rig is the rig this was asked to come out as loud as.
	Rig string `json:"rig,omitempty"`
	// Wanted is the cabinet reading of the rig this was asked to match, and
	// Was this chain's own cabinet reading. Both in the scale the block
	// library measures a cabinet in.
	//
	// **Not a level the chain comes out at**, and the difference matters. A
	// cabinet is measured in a chain holding that cabinet alone, which reads
	// within a decibel of the empty loop while the chain it ends is twenty
	// louder. What these two figures support is the gap between them, which
	// is what decides the move. Reporting either as "the level" would be
	// claiming a measurement nobody took.
	Wanted float64 `json:"wanted"`
	Was    float64 `json:"was"`
	// Param is the control that was moved, with From and To either side.
	// Empty when nothing moved.
	Param string  `json:"param,omitempty"`
	From  float64 `json:"from,omitempty"`
	To    float64 `json:"to,omitempty"`
	// Matches is where this chain's cabinet reading sits once the move is
	// credited, so Wanted minus this is what is left of the gap. Equal to Was
	// when nothing moved.
	Matches float64 `json:"matches"`
	// Because says why nothing moved, when nothing did. Empty when something
	// did.
	Because string `json:"because,omitempty"`
}

// Asked reports whether a Setup stated a level at all.
//
// A Setup that said nothing leaves both figures at zero, and a zero target
// would be a chain asked to come out at full scale, which nobody asks for.
func (l Levelled) Asked() bool { return l.Wanted != 0 || l.Was != 0 }

// Moved is what a word did to a parameter.
type Moved struct {
	// Term is the word that moved it.
	Term string `json:"term"`
	// At is which block in the chain holds the control, so a later pass can
	// tell whether that control is still where the word left it. Two blocks
	// can carry a control of the same name, which is why the name alone will
	// not do.
	At int `json:"at"`
	// Held is what the preset ends up holding, where that is not what the
	// word wrote, and Overridden says so.
	//
	// The two exist because the report used to print a word's move as though
	// it were the outcome. A rig stating `Attack` outright beats a word, by
	// design and in that order, and a build said "says percussive — Attack
	// 0.04 to 0.06" while writing 0.12 into the file. Somebody hand-editing a
	// document reads that line to check their edit took, and it told them it
	// had not.
	Held       float64 `json:"held,omitempty"`
	Overridden bool    `json:"overridden,omitempty"`
	// Param is the control it moved. Empty when nothing in the chain answers
	// to this term yet, which is most of the vocabulary.
	Param string `json:"param"`
	// From and To are where the parameter was and where it went.
	From float64 `json:"from"`
	To   float64 `json:"to"`
	// Against names the axis another term in the same rig also spoke for.
	// Two words from one axis are two answers to one question, so neither is
	// applied: applying both lands back where it started and reads as though
	// the rig said nothing.
	Against string `json:"against"`
	// YieldedTo names the word somebody wrote that this one stood aside for.
	//
	// A different answer from Against, and reported as one. Against is a
	// contradiction nothing can resolve. This is a word a genre earned meeting
	// a word a person wrote on the same axis, where the person's stands: they
	// said it and a population did not.
	YieldedTo string `json:"yielded_to"`
	// Because says why the chain could not answer this word, when the chain
	// is the reason. Empty when nothing acts on the word at all, which is a
	// different answer: one says this rig cannot hear it, the other says
	// nobody has taught the project to listen.
	Because string `json:"because"`
	// Already says how the chain answers this word without a knob being
	// turned. A rig asking for no room, in a chain holding no reverb, asked
	// for something it already has.
	Already string `json:"already"`
	// Weight is how much of a step the word was worth, where 1 is the whole
	// step. Less than that when the term's own evidence measured the gap
	// that earned it against other players, and the gap is narrow.
	Weight float64 `json:"weight"`
	// Step is how far a whole step moved this control, and Default how far it
	// would have moved one the corpus cannot measure at all.
	//
	// Step below Default is the corpus agreeing about a control rather than
	// disagreeing, and it is how two genres asking for opposite things land
	// in the same place. Carried so a build can say so: five shipped genre
	// presets differ by 0.034 of Mid across all five, on a control running 0
	// to 1, while the words behind them read `scooped` against `mid-forward`.
	Step    float64 `json:"step,omitempty"`
	Default float64 `json:"default,omitempty"`
}

// Pinned says whether the corpus agreed about this control so closely that a
// whole step moves it less than half what it would move one nobody has
// measured.
//
// The honest reading of two builds that match: not that the words said the
// same thing, but that the population this project learns from has no opinion
// about the control those words reach for.
//
// Half rather than any narrowing at all, which is a threshold for reporting
// and not a claim about what anybody can hear. A spread that lands a hair
// under the default is float dust, and saying so prints two numbers that
// round to each other: grunge's Treble measures 0.09999999999999998 against a
// default of 0.1. Below half, the corpus's agreement is the reason the knob
// barely moved, which is worth a reader's attention.
func (m Moved) Pinned() bool {
	return m.Acted() && m.Default > 0 && m.Step < m.Default/2
}

// Measured says whether a measurement sized the move rather than the word
// alone.
func (m Moved) Measured() bool { return m.Acted() && m.Weight > 0 && m.Weight < 1 }

// Beaten says whether something wrote over what this word did.
//
// Checked before Acted by anything reporting, because a word that moved a
// control and then lost it to a stated value did both and only the second is
// the outcome.
func (m Moved) Beaten() bool { return m.Overridden }

// Acted says whether the term moved anything.
func (m Moved) Acted() bool { return m.Param != "" }

// Contested says whether another term spoke for the same axis.
func (m Moved) Contested() bool { return m.Against != "" }

// Yielded says whether this word stood aside for one somebody wrote.
func (m Moved) Yielded() bool { return m.YieldedTo != "" }

// Unanswered says whether the chain, rather than this project, is why the
// word moved nothing.
func (m Moved) Unanswered() bool { return m.Because != "" }

// Holds says whether the chain already answers the word as built.
func (m Moved) Holds() bool { return m.Already != "" }

// Added is a block put in the chain that the rig did not name.
type Added struct {
	// Name is what the block is called.
	Name string `json:"name"`
	// Reason says why it is there.
	Reason string `json:"reason"`
	// Share is how many chains of this kind hold one, from zero to one. Zero
	// where there is no measurement rather than where nothing holds one: a
	// block substituted for gear no model emulates was not counted, and
	// reporting it as none would read as one.
	Share float64 `json:"share"`
	// Bypassed says it arrived switched off, and Running is the share of
	// chains carrying one that have it on.
	//
	// Two figures because they answer two questions. Share says a player would
	// have this block; Running says whether they would use it. Of the guitar
	// chains holding a delay 61.6% run it, and of those holding a drive 47.3%,
	// so adding either switched on decides something the evidence does not
	// support. Bypassed is what the build did about that.
	Bypassed bool    `json:"bypassed,omitempty"`
	Running  float64 `json:"running,omitempty"`
}

// Unfamiliar is a character term nothing defines.
type Unfamiliar struct {
	// Term is the word somebody wrote.
	Term string `json:"term"`
	// Near are the defined terms closest to it, if any are close enough to
	// be worth suggesting.
	Near []string `json:"near"`
}
