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
	"github.com/retr0h/tonestack/pkg/sdk/tone"
)

// Intent is what the ask contributes to a build.
//
// It travels beside the rig rather than inside it, and both halves of that are
// deliberate. A word is what somebody meant and a rig is what answered, so the
// rig holds no prose: two documents saying how it should sound is two places
// for the answer to drift. And the words cannot be resolved before they get
// here either, because turning "mid-forward" into a knob position needs the
// chain that answers it, which does not exist until this package has built it.
// Resolving them anywhere else would mean a second chain builder. Hence beside:
// the prose stays out of the rig, and the resolution stays where the chain is.
//
// The zero value is an ask that was never written down, which is ordinary. A
// RigSpec read off disk carries settings somebody already applied, so it is an
// answer rather than a request and needs no words at all.
type Intent struct {
	// Words are how it should sound, in the words a person would use.
	Words []Word
	// Attack is what sets the string moving, where the ask says. Empty is an
	// ask that did not say.
	Attack string
	// Name is what the preset should be called, which is the subject the ask
	// names: "Mike Dirnt" is what somebody wants to read on the screen.
	//
	// Empty falls back to the rig's identifier. A rig with no ask beside it
	// has no subject to be named after, and its identifier is the one name it
	// has of its own.
	Name string
}

// Word is one thing the ask says it should sound like, and why that is
// believed.
//
// This package's own type rather than the ask's, because what it needs of a
// word is the term and the figures that sized it, and taking the whole shape
// of whichever document carried it would tie the resolution to that document.
//
// The evidence is the reason this is a struct and not a string. A figure
// measured off a record and held against what other players read is the
// difference between a knob moved because somebody listened and a knob moved
// because something guessed, and it is what decides how far the word moves
// its control. A word with none is worth the whole step, which is what every
// rig did before any of this was measured.
type Word struct {
	// Term is the word, as the vocabulary spells it.
	Term string
	// Evidence is why the word is believed, and where it carries both a
	// figure and what that figure was compared against, how far it moves
	// anything. Empty is a word somebody asserted.
	Evidence []tone.Evidence
}
