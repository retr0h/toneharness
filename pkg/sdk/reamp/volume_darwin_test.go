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

func (s *VolumeDarwinTestSuite) TestVolumeReadsWhatTheShellPrinted() {
	asks = func(string) ([]byte, error) { return []byte("38\n"), nil }

	got, err := volume()

	s.Require().NoError(err)
	s.Require().Equal(38, got, "trailing newline and all")
}

func (s *VolumeDarwinTestSuite) TestAShellThatFailed() {
	declined := errors.New("no such thing")
	asks = func(string) ([]byte, error) { return nil, declined }

	_, err := volume()

	s.Require().ErrorIs(err, ErrVolume)

	var fault *VolumeError
	s.Require().ErrorAs(err, &fault)
	s.Require().Equal("reading", fault.Doing)
	s.Require().Equal(declined, fault.Said)
}

// TestAShellThatPrintedSomethingElse is the answer nobody expects.
//
// osascript reports some failures on stdout and exits zero, so a reply that is
// not a number is not the same case as a shell that failed, and reading it as
// a level would put an arbitrary integer into a library.
func (s *VolumeDarwinTestSuite) TestAShellThatPrintedSomethingElse() {
	asks = func(string) ([]byte, error) { return []byte("no volume settings"), nil }

	_, err := volume()

	s.Require().ErrorIs(err, ErrVolume)

	var fault *VolumeError
	s.Require().ErrorAs(err, &fault)
	s.Require().Equal("reading", fault.Doing)
}

func (s *VolumeDarwinTestSuite) TestSetVolumeSaysTheLevelItWasGiven() {
	var said string

	asks = func(script string) ([]byte, error) {
		said = script

		return nil, nil
	}

	s.Require().NoError(setVolume(38))
	s.Require().Equal("set volume output volume "+strconv.Itoa(38), said)
}

func (s *VolumeDarwinTestSuite) TestSetVolumeOnAShellThatFailed() {
	asks = func(string) ([]byte, error) { return nil, errors.New("declined") }

	err := setVolume(38)

	s.Require().ErrorIs(err, ErrVolume)

	var fault *VolumeError
	s.Require().ErrorAs(err, &fault)
	s.Require().Equal("setting", fault.Doing)
}

// TestHeldMovesTheLevelAndReadsItBack covers all four of Held's answers.
//
// The read back is not decoration: a platform that accepts the instruction and
// rounds it, which macOS does on some hardware, leaves the rig at a level
// nobody asked for, and this is the only place that shows.
func (s *VolumeDarwinTestSuite) TestHeldMovesTheLevelAndReadsItBack() {
	tests := []struct {
		name    string
		replies []func(string) ([]byte, error)
		want    int
		moved   bool
		fails   bool
	}{
		{
			name: "already there, so nothing is set",
			replies: []func(string) ([]byte, error){
				func(string) ([]byte, error) { return []byte("38"), nil },
			},
			want: 38,
		},
		{
			name: "moved, and read back as what was asked for",
			replies: []func(string) ([]byte, error){
				func(string) ([]byte, error) { return []byte("20"), nil },
				func(string) ([]byte, error) { return nil, nil },
				func(string) ([]byte, error) { return []byte("38"), nil },
			},
			want: 38, moved: true,
		},
		{
			name: "moved, and the platform rounded it",
			replies: []func(string) ([]byte, error){
				func(string) ([]byte, error) { return []byte("20"), nil },
				func(string) ([]byte, error) { return nil, nil },
				func(string) ([]byte, error) { return []byte("37"), nil },
			},
			want: 37, moved: true,
		},
		{
			name: "the first read failed",
			replies: []func(string) ([]byte, error){
				func(string) ([]byte, error) { return nil, errors.New("boom") },
			},
			fails: true,
		},
		{
			name: "the set failed, and it says where it was",
			replies: []func(string) ([]byte, error){
				func(string) ([]byte, error) { return []byte("20"), nil },
				func(string) ([]byte, error) { return nil, errors.New("boom") },
			},
			want: 20, fails: true,
		},
		{
			name: "the read back failed after it moved",
			replies: []func(string) ([]byte, error){
				func(string) ([]byte, error) { return []byte("20"), nil },
				func(string) ([]byte, error) { return nil, nil },
				func(string) ([]byte, error) { return nil, errors.New("boom") },
			},
			moved: true, fails: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			at := 0
			asks = func(script string) ([]byte, error) {
				reply := tt.replies[at]
				at++

				return reply(script)
			}

			got, moved, err := Held(38)

			s.Require().Equal(tt.moved, moved)
			s.Require().Equal(tt.want, got)
			s.Require().Len(tt.replies, at, "every reply is used")

			if tt.fails {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
		})
	}
}

// TestALevelOutsideTheScaleIsRefusedBeforeTheShell covers the guard.
func (s *VolumeDarwinTestSuite) TestALevelOutsideTheScaleIsRefusedBeforeTheShell() {
	asks = func(string) ([]byte, error) {
		s.Require().Fail("a level this cannot mean should not reach the shell")

		return nil, nil
	}

	for _, at := range []int{-1, 101} {
		err := SetVolume(at)

		s.Require().ErrorIs(err, ErrVolume, "%d", at)
		s.Require().ErrorContains(err, "0 to 100")
	}
}

func TestVolumeDarwinTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VolumeDarwinTestSuite))
}
