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

package tone

import "github.com/retr0h/tonestack/pkg/sdk/tone/internal/gen"

// The contract's types, named here so nothing outside reaches into gen.
//
// Aliases rather than wrappers, so a ToneSpec built by a caller and one read
// off disk are the same type and nothing converts at the seam. The generated
// package is internal because its names follow whatever the generator decides
// to call them, and a caller pinned to that finds out at compile time when it
// changes.
type (
	// Spec is a request for a sound: what to sound like, in whatever terms
	// the person has. Every field is optional.
	Spec = gen.ToneSpec
	// Setup is what somebody has. Separate from the ask because an
	// instrument is a fact about a person rather than about the sound they
	// are chasing today.
	Setup = gen.Setup
	// Like is something to sound like: a player, a band, a song, or a
	// recording handed over to be measured.
	Like = gen.Like
	// Years is the window a request is about, for a player whose sound
	// changed over their career.
	Years = gen.Years
	// Wanted is a piece of gear a request names, and whether to refuse
	// rather than substitute for it.
	Wanted = gen.Wanted
	// Role is what a piece of gear does in the chain.
	Role = gen.Role
	// Nudge is a move along one axis from wherever the last answer landed,
	// which is what conversation is made of.
	Nudge = gen.Nudge
	// Device is the hardware, which decides what a chain may cost.
	Device = gen.Device
	// Instrument is one instrument somebody owns, in what changes the sound.
	Instrument = gen.Instrument
	// Strings is what is on it. Flatwounds and roundwounds are a larger
	// difference than most pedals.
	Strings = gen.Strings
	// Owned is one thing on the device that did not ship with it.
	Owned = gen.Owned
	// OwnedKind is whether that is an impulse response or a model.
	OwnedKind = gen.OwnedKind
)

// What a request may say.
const (
	SchemaName  = "ToneSpec"
	SetupSchema = "Setup"
)

// The strings an instrument may carry.
const (
	StringsRound   = gen.StringsRound
	StringsFlat    = gen.StringsFlat
	StringsTape    = gen.StringsTape
	StringsUnknown = gen.StringsUnknown
)

// What somebody may own that the device did not ship with.
const (
	OwnedIR    = gen.OwnedIR
	OwnedModel = gen.OwnedModel
)
