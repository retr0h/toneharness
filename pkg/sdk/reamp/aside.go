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
package reamp

import (
	"errors"
	"testing"
)

// ErrUnderTest is returned instead of changing anything about the machine a
// test is running on.
var ErrUnderTest = errors.New(
	"a test may not change the computer's own audio settings")

// underTest says whether this is a test binary rather than somebody's run.
//
// **Two functions here reach out and change the machine**, and both are a
// side effect on whoever is sitting at it: the output level and which device
// is the default output. Everything else this package does is confined to a
// buffer.
//
// What that cost, on 2026-10-24. The owner of this pedal was playing bass and
// said "something mutes the system while I'm playing". He was right and the
// cause was the test suite: `MeasureControls` falls back to [Held] when no
// caller injects a level, the sweep suites leave that field nil on all but one
// of their rows, and the options they build carry `Volume: 0`. So every run of
// `go test ./pkg/cli/` called `Held(0)`, which sets the output level to zero.
//
// It cost readings as well as patience. Three amplifiers of a sweep campaign
// failed that afternoon, one of them saying the level "was asked for 38 and
// came back at 0" and two of them reading six consecutive silent positions
// with signal either side. The tests were running at the same time as the
// sweeps, muting the reference partway through, and the sweep then blamed the
// chain. See task 279.
//
// **Injection is the better discipline and this is the floor under it.** A
// suite that passes its own level is clearer about what it is testing, and
// every one of them should; what this adds is that forgetting cannot reach
// somebody's speakers. A guard that depends on every future test remembering
// is a guard that has already failed once.
//
// `testing.Testing()` rather than an environment variable or a build tag,
// because it is true in exactly the binaries this must refuse and needs
// nothing of whoever runs them.
//
// **Wrapped around the defaults rather than put inside the exported calls**,
// which is the difference between a guard and a wall. This package's own
// suites replace `writes` and `setsOutput` with fakes and then exercise the
// whole of [Held] and [PlaysThrough] through them, including the paths where
// the platform refuses or answers something other than what it was told.
// Refusing inside the exported function would have taken those tests away and
// replaced them with a test of the refusal, which is a worse suite and a
// worse guard: what is dangerous is reaching the machine, and a fake is not
// the machine.
// A variable, so the pass-through can be exercised. Left as a function the
// guard calls directly, the branch that reaches the platform is unreachable
// from any test by construction, and a guard with an untested pass-through is
// a guard nobody has checked in the direction that matters: that a real run
// still changes the level.
//
// Flipping it is not a way round the guard. What a test gets by flipping it is
// the fake it passed to [aside], never the machine: `writes` and `setsOutput`
// are the only callers that bind the real platform, and neither is reachable
// from here.
var underTest = testing.Testing

// aside wraps a call that changes the machine so a test binary cannot make it.
//
// Named for what it does rather than for the test: this is the real thing set
// aside, and the only caller is the default each injectable variable is
// declared with.
//
// The wrapper rather than the wrapped is what carries the refusal, so the
// thing being guarded stays a plain function somebody can read.
func aside[T any](
	platform func(T) error,
) func(T) error {
	return func(to T) error {
		if underTest() {
			return ErrUnderTest
		}

		return platform(to)
	}
}
