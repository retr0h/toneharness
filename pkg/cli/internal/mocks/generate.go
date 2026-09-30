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

// Package mocks holds generated test doubles for the cli package's interfaces.
//
// internal, because a double is for this package's own tests and nothing
// outside the repository has a use for one. CONTRIBUTING: "pkg/ holds what
// something outside this repository would call".
package mocks

// The directives, rather than one beside each interface it doubles.
// CONTRIBUTING: "the directives live in a generate.go that holds no code", so
// `just go-generate` regenerates every double from one place and a source file
// carries no build tooling.

//go:generate go tool go.uber.org/mock/mockgen -source=../../pedal.go -destination=pedal.gen.go -package=mocks
//go:generate go tool go.uber.org/mock/mockgen -source=../../tone_build.go -destination=resolver.gen.go -package=mocks
//go:generate go tool go.uber.org/mock/mockgen -source=../../tone_tune.go -destination=tuner.gen.go -package=mocks
//go:generate go tool go.uber.org/mock/mockgen -source=../../types.go -destination=types.gen.go -package=mocks
