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
package device

import (
	"context"
	"errors"
	"testing"
)

// ErrUnderTest is returned instead of opening the pedal on the desk.
var ErrUnderTest = errors.New(
	"a test may not reach a device; build the client with WithDevices")

// underTest says whether this is a test binary rather than somebody's run.
//
// A variable so the pass-through can be exercised, for the reason `reamp`'s
// is: a guard whose open path nothing tests has only been checked in the
// direction that refuses, and the direction that matters to whoever is running
// this is the other one. Flipping it hands a test the fake bus it already
// injected, never the hardware.
var underTest = testing.Testing

// setAside is an Opener that will not open hardware.
//
// **The reason this exists, on 2026-10-24.** A test built an `sdk.Client`
// without `WithDevices`, which falls back to the real USB bus. It then built a
// preset and wrote it to slot 11B of the pedal on the desk. The slot happened
// to be empty so nothing was lost, and that was luck rather than design: the
// same test against a slot holding somebody's work would have replaced it, and
// a flash write has no undo.
//
// `reamp` learned this an hour earlier about the output level, where a suite
// muted the machine it ran on and cost three sweeps. The pattern is the same
// and so is the answer: the thing that reaches out is set aside in a test
// binary, so forgetting cannot reach hardware. A rule that depends on every
// future test remembering `WithDevices` is a rule that has already failed
// once.
//
// **Opening is the narrowest point, and the only one guarded.** Every read and
// every write goes through a session and a session comes from [setAside.Open],
// so one refusal covers all of them and no test loses the ability to exercise
// the layers above with a fake.
//
// [setAside.List] is left alone, because listing cannot change anything: it
// reports what is on the bus and returns. Refusing it would buy no safety and
// would take away the one test that exercises enumeration against a real bus.
type setAside struct {
	// opens is the real thing, kept for the calls that take nothing from the
	// device. Nothing here ever calls its Open.
	opens Opener
}

// Open refuses, which is the whole of it.
func (setAside) Open(
	_ context.Context,
) (Editor, error) {
	return nil, ErrUnderTest
}

// List passes through. Enumerating the bus is a read that changes nothing, and
// a test that wants to know what is attached is answering a question about the
// machine it is running on rather than acting on one.
func (s setAside) List(
	ctx context.Context,
) ([]Descriptor, error) {
	return s.opens.List(ctx)
}
