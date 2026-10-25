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

// Package config is the application's own settings: what somebody has, and
// what they want done with it.
//
// **Nothing shares this.** It lives at $XDG_CONFIG_HOME/toneharness/config.yaml,
// it never leaves the machine it is on, and no rig anybody ships or downloads
// refers to it. That is why it is a contract of its own rather than a corner
// of the ToneSpec: a rig is published and read by strangers against a version
// they can see, and a preference is not. While the two shared one contract,
// adding a field here moved the version every shipped rig declared, so one
// person's slot map bumped the number the marketplace validates against three
// times in a day while nothing about rigs changed.
//
// The checking is shared, the contracts are not: see
// [github.com/retr0h/toneharness/pkg/sdk/internal/contract].
package config

import (
	_ "embed"
	"io"

	"github.com/retr0h/toneharness/pkg/sdk/config/internal/gen"
	"github.com/retr0h/toneharness/pkg/sdk/internal/contract"
)

// Schema is what a config file is checked against.
//
// Beside the code that checks it rather than in a directory of its own, for
// the same reason the catalog and the corpus keep their data beside
// themselves: a package that cannot be moved without remembering to bring a
// file with it is a package nobody can move.
//
//go:embed data/config.openapi.yaml
var Schema []byte

// Name is what the document calls itself, and what an error names.
const Name = "Config"

// checker holds the contract, parsed once.
var checker = contract.New(Schema, Name)

type (
	// Config is what somebody has and what they want done with it.
	Config = gen.Config
	// Device is the hardware, which decides what a chain may cost and hold.
	Device = gen.Device
	// Held is one instrument somebody owns, described in what changes the
	// sound.
	Held = gen.Held
	// Owned is content the device does not ship with.
	Owned = gen.Owned
	// PlaysInto is what the pedal is plugged into, when somebody says.
	PlaysInto = gen.PlaysInto
	// Slotted is one rig and the slot somebody keeps it in.
	Slotted = gen.Slotted
	// OwnedKind says what kind of content somebody loaded onto the device.
	OwnedKind = gen.OwnedKind
)

// The kinds of content a device does not ship with.
const (
	OwnedIR    = gen.OwnedIR
	OwnedModel = gen.OwnedModel
)

// The values PlaysInto takes.
const (
	PA         = gen.Pa
	Headphones = gen.Headphones
	AmpFront   = gen.AmpFront
	AmpReturn  = gen.AmpReturn
)

// ErrInvalid reports a config that does not meet its contract.
var ErrInvalid = contract.ErrInvalid

// InvalidError says which part of a config is wrong.
type InvalidError = contract.InvalidError

// Load reads a config and checks it against its own contract.
func Load(
	r io.Reader,
) (Config, error) {
	var held Config

	return held, checker.Read(r, Name, &held)
}

// Validate reports whether a config built in memory meets the contract.
//
// A file read through [Load] is checked on the way in; one assembled in Go has
// never been through that, and nothing in a generated struct stops a required
// field being empty or an enumeration carrying a word that is not in it.
func Validate(
	held Config,
) error {
	return checker.Check(held, Name)
}

// Version is the contract this package reads configs against, as the
// contract's own `info.version` states it.
func Version() (string, error) {
	return checker.Version()
}
