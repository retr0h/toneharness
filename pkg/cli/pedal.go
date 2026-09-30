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

package cli

import (
	"context"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

//go:generate go tool go.uber.org/mock/mockgen -source=pedal.go -destination=internal/mocks/pedal.gen.go -package=mocks

// What measuring a device needs from one, split by who needs it.
//
// Interfaces rather than *sdk.Client, because a campaign that walks six
// hundred blocks is mostly bookkeeping: which refused, which clipped, what
// order to put them in, what to write when the run stops halfway. None of
// that should need hardware to check, and *sdk.Client satisfies all of them.
//
// Four of them rather than one, because the three commands here need
// different amounts. CONTRIBUTING: "a consumer declares only the methods it
// needs, so the interface it declares is as small as its use, and two
// consumers of the same thing get two different interfaces rather than one
// that serves neither." Measuring every block never moves a knob; checking
// parameter names never touches a list.
type (
	// Compiles turns a rig into a preset a device will load.
	Compiles interface {
		Compile(ctx context.Context, in sdk.Compile) (sdk.Built, error)
	}

	// Plays puts a preset in front of the device without storing it.
	//
	// Without storing it because a slot is flash, and a campaign that loaded
	// a chain through one would spend a flash write per control.
	Plays interface {
		Play(ctx context.Context, file string) error
	}

	// Turns moves one control that is a dial.
	Turns interface {
		Turn(ctx context.Context, at sdk.Address, value float32) error
	}

	// Chooses moves one that is a list.
	//
	// Apart from Turns because a list is not a dial: a cabinet's twelve
	// microphones have no position between the third and the fourth, so a
	// sweep of one is a different measurement from a sweep of the other.
	Chooses interface {
		Choose(ctx context.Context, at sdk.Address, value int) error
	}

	// Switches moves one that is a switch.
	//
	// Apart from both again, and for the same reason: the device does not
	// coerce, so a switch declines 1.0 where it wants true with the error it
	// gives for a block that is not there.
	Switches interface {
		Switch(ctx context.Context, at sdk.Address, on bool) error
	}

	// ReadsFiles says what a preset file holds, without a device.
	ReadsFiles interface {
		PresetFile(ctx context.Context, path string) (sdk.Reading, error)
	}

	// Names says which models exist, and for which device.
	//
	// Asked of the client rather than read from catalog.BuiltIn, because
	// BuiltIn is the HX Stomp's and every measuring command advertises
	// --catalog and --device. Read straight, those flags were accepted and
	// reached nothing: an LT owner got the Stomp's models, the Stomp's DSP
	// budget, and readings filed under a pedal they do not own.
	Names interface {
		Catalog(ctx context.Context) (*catalog.Catalog, error)
	}

	// Reads says what the device is playing.
	Reads interface {
		Current(ctx context.Context, as sdk.Format) (sdk.Reading, error)
	}
)

// Loader is what measuring every block needs: build a chain, play it, and
// know which models the attached device has.
type Loader interface {
	Compiles
	ReadsFiles
	Plays
	Names
}

// Prober is what checking parameter names needs, which is a Loader that can
// also move a dial and read back what moved.
type Prober interface {
	Loader
	Turns
	Reads
}

// Pedal is the whole of it, which only sweeping a block's controls needs.
type Pedal interface {
	Prober
	Chooses
	Switches
}
