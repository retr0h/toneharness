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

	"github.com/retr0h/tonestack/pkg/sdk"
)

//go:generate go tool go.uber.org/mock/mockgen -source=pedal.go -destination=internal/mocks/pedal.gen.go -package=mocks

// Pedal is what measuring a device needs from one.
//
// An interface rather than *sdk.Client, because a campaign that walks six
// hundred blocks is mostly bookkeeping — which refused, which clipped, what
// order to put them in, what to write when the run stops halfway — and none
// of that should need hardware to check. *sdk.Client satisfies it.
type Pedal interface {
	// Compile turns a rig into a preset a device will load.
	Compile(ctx context.Context, in sdk.Compile) (sdk.Built, error)
	// Play puts a preset in front of the device without storing it.
	Play(ctx context.Context, file string) error
	// Turn moves one control that is a dial, and Choose one that is a list.
	Turn(ctx context.Context, at sdk.Address, value float32) error
	Choose(ctx context.Context, at sdk.Address, value int) error
	// Current reads what the device is playing.
	Current(ctx context.Context, as sdk.Format) (sdk.Reading, error)
}
