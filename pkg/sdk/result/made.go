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

import "github.com/retr0h/toneharness/pkg/sdk/plan"

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
	// Path is the file that was written.
	Path string `json:"path"`
}

// Moved is what a word did to a parameter.
type Moved struct {
	// Term is the word that moved it.
	Term string `json:"term"`
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
}

// Measured says whether a measurement sized the move rather than the word
// alone.
func (m Moved) Measured() bool { return m.Acted() && m.Weight > 0 && m.Weight < 1 }

// Acted says whether the term moved anything.
func (m Moved) Acted() bool { return m.Param != "" }

// Contested says whether another term spoke for the same axis.
func (m Moved) Contested() bool { return m.Against != "" }

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
}

// Unfamiliar is a character term nothing defines.
type Unfamiliar struct {
	// Term is the word somebody wrote.
	Term string `json:"term"`
	// Near are the defined terms closest to it, if any are close enough to
	// be worth suggesting.
	Near []string `json:"near"`
}
