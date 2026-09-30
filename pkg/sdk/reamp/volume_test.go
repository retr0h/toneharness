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

// HeldTestSuite covers putting the output level where a campaign wants it.
//
// Internal and portable, standing in for the platform rather than for the shell
// underneath it. Held's body is only reachable where a platform answers at all:
// everywhere but macOS the first read fails, so on a Linux runner the read back
// — the one thing that catches a platform accepting a level and rounding it —
// could not be executed at all.
type HeldTestSuite struct {
	suite.Suite

	wasReads  func() (int, error)
	wasWrites func(int) error
}

func (s *HeldTestSuite) SetupTest() {
	s.wasReads, s.wasWrites = reads, writes
}

func (s *HeldTestSuite) TearDownTest() {
	reads, writes = s.wasReads, s.wasWrites
}

// answers makes the platform reply with each level in turn, and record what it
// was told to set.
func (s *HeldTestSuite) answers(
	levels []int,
	failAt int,
	setFails bool,
) *int {
	at, told := 0, new(int)

	reads = func() (int, error) {
		if at == failAt {
			at++

			return 0, errors.New("the platform declined")
		}

		got := levels[at]
		at++

		return got, nil
	}

	writes = func(to int) error {
		*told = to

		if setFails {
			return errors.New("the platform declined")
		}

		return nil
	}

	return told
}

// TestHeldMovesTheLevelAndReadsItBack covers every answer Held has.
//
// The read back is not decoration: a platform that accepts the instruction and
// rounds it, which macOS does on some hardware, leaves the rig at a level
// nobody asked for and this is the only place that shows.
func (s *HeldTestSuite) TestHeldMovesTheLevelAndReadsItBack() {
	tests := []struct {
		name     string
		levels   []int
		failAt   int
		setFails bool
		want     int
		moved    bool
		fails    bool
	}{
		{
			name:   "already there, so nothing is set",
			levels: []int{38}, failAt: -1, want: 38,
		},
		{
			name:   "moved, and read back as what was asked for",
			levels: []int{20, 38}, failAt: -1, want: 38, moved: true,
		},
		{
			// The whole reason the read back exists.
			name:   "moved, and the platform rounded it",
			levels: []int{20, 37}, failAt: -1, want: 37, moved: true,
		},
		{
			name:   "the first read failed",
			levels: []int{}, failAt: 0, fails: true,
		},
		{
			name:   "the set failed, and it says where it was",
			levels: []int{20}, failAt: -1, setFails: true, want: 20, fails: true,
		},
		{
			name:   "the read back failed after it moved",
			levels: []int{20}, failAt: 1, moved: true, fails: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.answers(tt.levels, tt.failAt, tt.setFails)

			got, moved, err := Held(38)

			s.Require().Equal(tt.moved, moved)
			s.Require().Equal(tt.want, got)

			if tt.fails {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
		})
	}
}

// TestTheLevelAskedForIsTheLevelSet covers what reaches the platform.
func (s *HeldTestSuite) TestTheLevelAskedForIsTheLevelSet() {
	told := s.answers([]int{20, 38}, -1, false)

	_, _, err := Held(38)

	s.Require().NoError(err)
	s.Require().Equal(38, *told)
}

// TestALevelOutsideTheScaleIsRefusedBeforeThePlatform covers the guard.
func (s *HeldTestSuite) TestALevelOutsideTheScaleIsRefusedBeforeThePlatform() {
	writes = func(int) error {
		s.Require().Fail("a level this cannot mean should not reach the platform")

		return nil
	}

	for _, at := range []int{-1, 101} {
		err := SetVolume(at)

		s.Require().ErrorIs(err, ErrVolume, strconv.Itoa(at))
		s.Require().ErrorContains(err, "0 to 100")
	}
}

// TestVolumeAnswersWhatThePlatformSaid is the one-line wrapper.
func (s *HeldTestSuite) TestVolumeAnswersWhatThePlatformSaid() {
	reads = func() (int, error) { return 42, nil }

	got, err := Volume()

	s.Require().NoError(err)
	s.Require().Equal(42, got)
}

func TestHeldTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HeldTestSuite))
}
