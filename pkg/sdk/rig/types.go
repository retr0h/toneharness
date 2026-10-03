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

package rig

import "github.com/retr0h/toneharness/pkg/sdk/tone"

// What a rig is made of.
//
// Declared by oapi-codegen from the contract and named here, so that nobody
// outside this package has to hold a generated type. A caller writing
// Spec is writing against a name this project chose; a caller writing
// gen.Rig was writing against whatever the generator happened to call it,
// and finding out at compile time when that changed.
//
// Aliases rather than wrappers. Spec and the generated type are the same
// type, so nothing converts at the seam and a rig built by the compiler is a
// rig a caller can read.
type (
	// Spec is a rig, complete. Sparse when hand-written; the same document
	// carries settings and evidence once anything has been measured.
	Spec = tone.Rig
	// Subject is who or what the rig is attributed to.
	Subject = tone.Subject
	// Kind is what a rig is attributed to: an artist, a band, a song, a
	// genre, or nothing in particular.
	Kind = tone.Kind
	// ChainEntry is one piece of gear in the signal path.
	ChainEntry = tone.ChainEntry
	// Role is what a piece of gear does: amp, cab, drive.
	Role = tone.Role
	// Capture is how the signal reached the tape: direct, miked, or both.
	Capture = tone.Capture
	// Evidence is where a claim came from.
	Evidence = tone.Evidence
	// EvidenceKind is how far somebody has to go to disagree with one.
	EvidenceKind = tone.EvidenceKind
	// Confidence is how far a claim should be trusted.
	Confidence = tone.Confidence
	// Instrument is what the rig is played on.
	Instrument = tone.Instrument
	// Settings are the values a piece of gear is set to, in musical words that
	// mean roughly the same on any amplifier.
	Settings = tone.Settings
	// Knob is one control, from 0 to 1, whatever the device's range is.
	Knob = tone.Knob
	// Substitute stands in for gear no device models.
	Substitute = tone.Substitute
	// Target is the hardware a plan was tuned on.
	Target = tone.Target
	// Controller is a parameter an expression pedal or footswitch moves.
	Controller = tone.Controller
	// Footswitch is what a switch does and how it is lit.
	Footswitch = tone.Footswitch
	// Section is one part of a song, as the roles that play in it.
	Section = tone.Section
	// Move is one control an expression pedal or a footswitch sweeps, said
	// portably: a role and one of the settings vocabulary's words.
	Move = tone.Move
	// MoveBy is what moves a control: an expression pedal, or a footswitch
	// set to sweep rather than switch.
	MoveBy = tone.MoveBy
	// MoveSetting is which control a move reaches, in the same words a rig
	// sets gear with.
	MoveSetting = tone.MoveSetting
	// Played is the instrument a rig is played on, which no device models.
	Played = tone.Played
	// Snapshot is one set of values a preset can recall.
	Snapshot = tone.Snapshot
	// DeviceState is everything a preset carries that this format does not
	// model as musical intent, kept as the device wrote it.
	DeviceState = tone.DeviceState
)

// The values those types may hold.
const (
	// RoleAmp and RoleCab are the two a listing looks for by name.
	RoleAmp = tone.RoleAmp
	RoleCab = tone.RoleCab

	// How the signal reached the tape. `direct` is the one worth counting:
	// it says a cabinet in the chain was not in the recorded signal.
	CaptureDirect = tone.CaptureDirect
	CaptureMiked  = tone.CaptureMiked
	CaptureBoth   = tone.CaptureBoth

	// How far a claim should be trusted. Unstated reads as the lowest,
	// because a rig that says nothing about itself has earned nothing.
	ConfidenceLow    = tone.ConfidenceLow
	ConfidenceMedium = tone.ConfidenceMedium
	ConfidenceHigh   = tone.ConfidenceHigh

	// Where a claim came from, strongest first. The contract says what each
	// one is and why it ranks where it does.
	EvidenceHeard  = tone.EvidenceHeard
	EvidenceCited  = tone.EvidenceCited
	EvidenceAudio  = tone.EvidenceAudio
	EvidenceUser   = tone.EvidenceUser
	EvidenceVideo  = tone.EvidenceVideo
	EvidenceCorpus = tone.EvidenceCorpus
	EvidenceLLM    = tone.EvidenceLLM

	// Not ranked, because it answers where to get something rather than why
	// a claim is believed.
	EvidenceStore = tone.EvidenceStore

	// What moves a control. Named rather than numbered, because the number
	// is one device family's and a rig is meant to outlive it.
	MoveByExpression = tone.MoveByExpression
	MoveByFootswitch = tone.MoveByFootswitch

	// What a rig is played on.
	InstrumentBass   = tone.InstrumentBass
	InstrumentGuitar = tone.InstrumentGuitar

	// What a rig is attributed to.
	KindArtist = tone.KindArtist
	KindBand   = tone.KindBand
	KindGenre  = tone.KindGenre
	KindSong   = tone.KindSong
	KindSound  = tone.KindSound

	// SchemaName is what the one document states. A rig is a section of it
	// since version 2 and no longer names a schema of its own.
	SchemaName = tone.SchemaName

	// What a piece of gear does. Deliberately the same words the catalog
	// groups by, so reading one against the other is a conversion rather
	// than a translation.
	RoleComp    = tone.RoleComp
	RoleDrive   = tone.RoleDrive
	RoleEQ      = tone.RoleEQ
	RoleMod     = tone.RoleMod
	RoleDelay   = tone.RoleDelay
	RoleReverb  = tone.RoleReverb
	RoleFilter  = tone.RoleFilter
	RolePitch   = tone.RolePitch
	RoleWah     = tone.RoleWah
	RoleGate    = tone.RoleGate
	RoleUtility = tone.RoleUtility
	RoleOther   = tone.RoleOther
)
