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
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
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
// document read off disk carries settings somebody already applied, so its rig
// is an answer rather than a request and needs no words at all.
type Intent struct {
	// Words are how it should sound, in the words a person would use.
	Words []Word
	// Attack is what sets the string moving, where the ask says. Empty is an
	// ask that did not say.
	//
	// The subject's, not the person's. Playing below is the other side.
	Attack string
	// Playing is how the person who will play this preset plays, where the
	// Setup says. Empty is a Setup that did not, and compensates for nothing.
	Playing Playing
	// Loudness is the cabinet reading a build should come out as loud as, in
	// the scale the block library measures one in. Nil is a Setup that asked
	// for nothing, which is most of them.
	//
	// A figure by the time it reaches here, resolved from the rig the Setup
	// named: this package builds a chain and has no business reading another
	// rig off disk to find out how loud it is.
	//
	// Beside Playing because it comes off the same document and for the same
	// reason: how somebody wants their own presets to sit changes on the clock
	// a pedal changes on rather than once per sound.
	Loudness *float64
	// Aim is what the records of whoever this rig is for measure as, as a
	// displacement in the units a block is measured in. Nil for a rig nobody
	// has records for, which is most of them.
	//
	// A figure by the time it reaches here, for the reason Loudness is one:
	// this package builds a chain and has no business deciding who a rig is
	// for or reading a corpus to find out.
	//
	// What it is for: an artist rig names the amplifier a source puts somebody
	// on and then had every control at the catalog's default, which for
	// `dee-dee-ramone` was Bass 0.41, Bias 0.74, ChVol 0.8 and Drive 0.6, the
	// factory setting to the digit. The records are listed on the document and
	// nothing read them.
	Aim *measured.Figures
	// Curves answers with what every control of one block does, measured.
	//
	// A reader rather than a map, because which models a chain holds is not
	// known until this package has resolved the gear names onto them, and
	// handing in a map would mean the caller resolving them first to ask the
	// same question twice.
	//
	// Nil where no sweeps were offered, which a release is: only the
	// fingerprints are packed into the binary. False for a block the campaign
	// never reached, which is 436 of this device's 614.
	Curves func(model catalog.ModelID) (measured.Curves, bool)
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
	// Derived says the word was earned by measuring a population rather than
	// written on the ask.
	//
	// Not inferable from the evidence. A genre's word carries audio evidence
	// and so does a word somebody measured off a record themselves, and the
	// two mean different things here: on an axis a person already spoke for,
	// a derived word stands aside rather than cancelling what they said.
	// Which is why this is stated rather than guessed at.
	Derived bool
}
