//go:build darwin

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
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

// VolumeDarwinTestSuite covers the one thing in this package that leaves the
// process.
//
// macOS keeps the output level where the slider does, and AppleScript is the
// only interface to it that needs no signed helper. So this shells out, and
// what a shell does cannot be asserted from a test that runs one: the four
// answers either side of `asks` are reached by standing in for it.
type VolumeDarwinTestSuite struct {
	suite.Suite

	was func(string) ([]byte, error)
}

func (s *VolumeDarwinTestSuite) SetupTest()    { s.was = asks }
func (s *VolumeDarwinTestSuite) TearDownTest() { asks = s.was }

// TestVolume covers volume, which reads the output level.
//
// One method and one table, so a case is a row rather than a file.
func (s *VolumeDarwinTestSuite) TestVolume() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "volume reads what the shell printed",
			then: func() {
				asks = func(string) ([]byte, error) { return []byte("38\n"), nil }

				got, err := volume()

				s.Require().NoError(err)
				s.Require().Equal(38, got, "trailing newline and all")
			},
		},
		{
			name: "a shell that failed",
			then: func() {
				declined := errors.New("no such thing")
				asks = func(string) ([]byte, error) { return nil, declined }

				_, err := volume()

				s.Require().ErrorIs(err, ErrVolume)

				var fault *VolumeError
				s.Require().ErrorAs(err, &fault)
				s.Require().Equal("reading", fault.Doing)
				s.Require().Equal(declined, fault.Said)
			},
		},
		{
			// The answer nobody expects.
			//
			// osascript reports some failures on stdout and exits zero, so a
			// reply that is not a number is not the same case as a shell that
			// failed, and reading it as a level would put an arbitrary
			// integer into a library.
			name: "a shell that printed something else",
			then: func() {
				asks = func(string) ([]byte, error) { return []byte("no volume settings"), nil }

				_, err := volume()

				s.Require().ErrorIs(err, ErrVolume)

				var fault *VolumeError
				s.Require().ErrorAs(err, &fault)
				s.Require().Equal("reading", fault.Doing)
			},
		},
	} {
		// No SetupTest here. Its job is to remember the shell the platform calls
		// before a row replaces it, and a second call would remember the
		// replacement, so TearDownTest would put a stub back.
		s.Run(tt.name, func() { tt.then() })
	}
}

// TestSetVolume covers setVolume, which puts the output level where a
// measurement wants it.
//
// One method and one table, so a case is a row rather than a file.
func (s *VolumeDarwinTestSuite) TestSetVolume() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "set volume says the level it was given",
			then: func() {
				var said string

				asks = func(script string) ([]byte, error) {
					said = script

					return nil, nil
				}

				s.Require().NoError(setVolume(38))
				s.Require().Equal("set volume output volume "+strconv.Itoa(38), said)
			},
		},
		{
			name: "set volume on a shell that failed",
			then: func() {
				asks = func(string) ([]byte, error) { return nil, errors.New("declined") }

				err := setVolume(38)

				s.Require().ErrorIs(err, ErrVolume)

				var fault *VolumeError
				s.Require().ErrorAs(err, &fault)
				s.Require().Equal("setting", fault.Doing)
			},
		},
	} {
		// No SetupTest here. Its job is to remember the shell the platform calls
		// before a row replaces it, and a second call would remember the
		// replacement, so TearDownTest would put a stub back.
		s.Run(tt.name, func() { tt.then() })
	}
}

func TestVolumeDarwinTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VolumeDarwinTestSuite))
}
