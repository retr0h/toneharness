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

import "github.com/retr0h/toneharness/pkg/sdk/tone/internal/gen"

// The contract's types, named here so nothing outside reaches into gen.
//
// Aliases rather than wrappers, so a ToneSpec built by a caller and one read
// off disk are the same type and nothing converts at the seam. The generated
// package is internal because its names follow whatever the generator decides
// to call them, and a caller pinned to that finds out at compile time when it
// changes.
type (
	// Spec is one sound: the gear that makes it, and what somebody asked for.
	//
	// The whole document since version 2, where it was the ask alone and a
	// RigSpec sat beside it in a second file. `Rig` is required and `Ask` is
	// not, because a sound nobody can name the gear for is a sound nothing can
	// model, which is what the solver always said.
	Spec = gen.ToneSpec
	// SpecSchema is what a document says it is, so a writer states it rather
	// than spelling the string.
	SpecSchema = gen.ToneSpecSchema
	// Ask is a request for a sound: what to sound like, in whatever terms the
	// person has. Every field on it is optional.
	Ask = gen.Ask
	// Rig is the gear that answers an ask, in signal order, with a source for
	// every claim. Real-world names rather than model identifiers, which is
	// what lets one compile for any Helix.
	Rig = gen.Rig
	// Held is one instrument somebody owns. Named for what it is rather than
	// `Instrument`, which is which half of a catalog a build may draw from.
	Held = gen.Held
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
	// PlaysInto is what the pedal is plugged into, when somebody says. The
	// last thing in the path is not always the cabinet block.
	PlaysInto = gen.PlaysInto
	// OwnedKind is whether that is an impulse response or a model.
	OwnedKind = gen.OwnedKind
	// Subject is who or what the request is about.
	Subject = gen.Subject
	// Technique is how the instrument is played, which no device models and
	// which still decides what the rig has to do.
	Technique = gen.Technique
	// Played is an instrument the subject played, where it is known.
	Played = gen.Played
	// Confidence is how far a claim should be trusted, set by a person rather
	// than derived.
	Confidence = gen.Confidence
	// Evidence is why a claim is believed.
	Evidence = gen.Evidence
	// EvidenceKind is what kind of thing that evidence is.
	EvidenceKind = gen.EvidenceKind
	// Word is one thing it should sound like, and why that is believed. The
	// evidence sizes how far it moves a control, so a word that was measured
	// moves further than one a model asserted.
	Word = gen.Word
	// Correction is one round of correction, and what a person made of the
	// result. The only place a human ear is written down.
	Correction = gen.Correction
	// Change is one field a correction moved.
	Change = gen.Change
	// Position is where along the string the instrument is played.
	Position = gen.TechniquePosition
	// Muting is what damps the string.
	Muting = gen.TechniqueMuting
	// Attack is what sets the string moving.
	Attack = gen.TechniqueAttack
	// The rig half's own types, exposed here because one contract defines them
	// and `pkg/sdk/rig` is a view over this package rather than a second
	// generated copy. Both contracts used to define ten of these separately,
	// which produced two Go types of the same shape and different identities.
	//
	// ChainEntry is one piece of gear in a chain, by the name a person uses.
	ChainEntry = gen.ChainEntry
	// Settings is how the gear is set, in musical terms from 0 to 1.
	Settings = gen.Settings
	// Substitute is what the device offered for gear it has no model of.
	Substitute = gen.Substitute
	// PresetMember is one thing a preset holds beside its chain: a snapshot, a
	// processor's routing, a footswitch, a cabinet, an amplifier's remote. One
	// type for all eighteen kinds, because the device uses one shape for them.
	PresetMember = gen.PresetMember
	// Target is what a controller moves, and Controller is the pedal or switch
	// that moves it. Move is one of its ends.
	Target     = gen.Target
	Controller = gen.Controller
	Move       = gen.Move
	// MoveBy and MoveSetting are which kind of thing a move changes.
	MoveBy      = gen.MoveBy
	MoveSetting = gen.MoveSetting
	// Footswitch, Section and Snapshot are what a device wraps a chain in.
	Footswitch = gen.Footswitch
	Section    = gen.Section
	Snapshot   = gen.Snapshot
	// DeviceState is the rest of what a preset holds, kept so one survives a
	// round trip through a format that does not claim to understand all of it.
	DeviceState = gen.DeviceState
	// Capture is whether a cabinet was taken direct or miked.
	Capture = gen.Capture
	// Kind is what a subject is: an artist, a band, a song, a genre, a sound.
	Kind = gen.Kind
	// AskInstrument is the generator's name for the one enum the contract still
	// spells inline, on the ask's own instrument field.
	AskInstrument = gen.AskInstrument
	// Knob is one control, 0 to 1, whatever the device's own range is.
	Knob = gen.Knob
)

// How far a claim should be trusted.
const (
	ConfidenceLow    = gen.ConfidenceLow
	ConfidenceMedium = gen.ConfidenceMedium
	ConfidenceHigh   = gen.ConfidenceHigh
)

// What kind of thing a piece of evidence is.
const (
	EvidenceAudio  = gen.EvidenceAudio
	EvidenceCited  = gen.EvidenceCited
	EvidenceCorpus = gen.EvidenceCorpus
	EvidenceHeard  = gen.EvidenceHeard
	EvidenceLLM    = gen.EvidenceLLM
	EvidenceStore  = gen.EvidenceStore
	EvidenceUser   = gen.EvidenceUser
	EvidenceVideo  = gen.EvidenceVideo
)

// What an ask is about.
const (
	KindArtist = gen.KindArtist
	KindBand   = gen.KindBand
	KindGenre  = gen.KindGenre
	KindSong   = gen.KindSong
	KindSound  = gen.KindSound
)

// What sets the string moving.
const (
	AttackPick    = gen.AttackPick
	AttackFingers = gen.AttackFingers
	AttackSlap    = gen.AttackSlap
	AttackThumb   = gen.AttackThumb
	AttackHybrid  = gen.AttackHybrid
)

// Where along the string, and what damps it.
const (
	PositionBridge = gen.PositionBridge
	PositionMiddle = gen.PositionMiddle
	PositionNeck   = gen.PositionNeck
	MutingNone     = gen.MutingNone
	MutingPalm     = gen.MutingPalm
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

// What the pedal may be plugged into.
//
// The two amplifier entries are different questions rather than one: the
// instrument input puts the amplifier's own preamp in front of its speaker,
// and the effects return bypasses that preamp and leaves the power section.
const (
	Pa         = gen.Pa
	Headphones = gen.Headphones
	AmpFront   = gen.AmpFront
	AmpReturn  = gen.AmpReturn
	// The rig half's, for the same reason its types are here.
	RoleAmp     = gen.RoleAmp
	RoleCab     = gen.RoleCab
	RoleDrive   = gen.RoleDrive
	RoleComp    = gen.RoleComp
	RoleGate    = gen.RoleGate
	RoleEQ      = gen.RoleEQ
	RoleMod     = gen.RoleMod
	RoleDelay   = gen.RoleDelay
	RoleReverb  = gen.RoleReverb
	RoleWah     = gen.RoleWah
	RolePitch   = gen.RolePitch
	RoleFilter  = gen.RoleFilter
	RoleUtility = gen.RoleUtility
	RoleOther   = gen.RoleOther

	CaptureDirect = gen.CaptureDirect
	CaptureMiked  = gen.CaptureMiked
	CaptureBoth   = gen.CaptureBoth

	MoveByExpression = gen.MoveByExpression
	MoveByFootswitch = gen.MoveByFootswitch

	InstrumentGuitar = gen.InstrumentInstrumentGuitar
	InstrumentBass   = gen.InstrumentInstrumentBass
)

// SchemaToneSpec is the value a ToneSpec's `schema` field carries.
const SchemaToneSpec = gen.ToneSpecSchemaToneSpec
