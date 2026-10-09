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

	"github.com/stretchr/testify/suite"
)

// OutputTestSuite covers making the computer play through the device a
// measurement means.
//
// Untagged, so it runs on every platform: the seam it stands its own functions
// in is the only thing here that touches one, and the branches either side are
// the same everywhere.
type OutputTestSuite struct {
	suite.Suite

	says func() (string, error)
	sets func(string) error
}

func (s *OutputTestSuite) SetupTest() {
	s.says, s.sets = saysOutput, setsOutput
}

func (s *OutputTestSuite) TearDownTest() {
	saysOutput, setsOutput = s.says, s.sets
}

// TestPlaysThrough covers the four ways asking ends.
//
// One method and one table, so a case is a row rather than a file.
func (s *OutputTestSuite) TestPlaysThrough() {
	declined := errors.New("the platform declined")

	for _, tt := range []struct {
		name string
		// reading answers each call to read the device, in order.
		reading []string
		readErr error
		setErr  error
		// want is the device asked for.
		want string
		// at and moved are what PlaysThrough should answer.
		at    string
		moved bool
		err   error
	}{
		{
			// Already the one that plays, so nothing is touched. Matched as a
			// substring, the way --hardware matches, so the same string picks
			// the same device in both places.
			name:    "already the device that plays",
			reading: []string{"External Headphones"},
			want:    "external headphones",
			at:      "External Headphones",
		},
		{
			// The case this exists for: read, set, read back.
			name:    "moved to the device that plays",
			reading: []string{"John's AirPods Max", "External Headphones"},
			want:    "External Headphones",
			at:      "External Headphones",
			moved:   true,
		},
		{
			name:    "a platform that will not say",
			readErr: declined,
			want:    "External Headphones",
			err:     declined,
		},
		{
			name:    "a platform that will not change",
			reading: []string{"John's AirPods Max"},
			setErr:  declined,
			want:    "External Headphones",
			at:      "John's AirPods Max",
			err:     declined,
		},
		{
			// Read back rather than assumed, and the read back failing is its
			// own answer: something moved and nothing knows where to.
			name:    "a platform that took it and then would not say",
			reading: []string{"John's AirPods Max"},
			want:    "External Headphones",
			at:      "John's AirPods Max",
			moved:   true,
			err:     declined,
		},
	} {
		s.Run(tt.name, func() {
			said := 0

			saysOutput = func() (string, error) {
				if tt.readErr != nil {
					return "", tt.readErr
				}

				if said >= len(tt.reading) {
					return "", declined
				}

				out := tt.reading[said]
				said++

				return out, nil
			}

			setsOutput = func(string) error { return tt.setErr }

			at, moved, err := PlaysThrough(tt.want)

			s.Require().Equal(tt.at, at, "the device in force afterwards")
			s.Require().Equal(tt.moved, moved)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)
		})
	}
}

// TestOutput covers reading the device back through the platform itself.
//
// Either answer is correct and which one depends on the platform: macOS says a
// name and everything else says it cannot. What is not correct is a name and an
// error together, or neither.
func (s *OutputTestSuite) TestOutput() {
	at, err := Output()
	if err != nil {
		s.Require().ErrorIs(err, ErrOutput)
		s.Require().Empty(at)

		return
	}

	s.Require().NotEmpty(at, "a platform that answers names something")
}

// TestOutputErrorsSayWhichHalfFailed covers what the two error types read as.
func (s *OutputTestSuite) TestOutputErrorsSayWhichHalfFailed() {
	for _, tt := range []struct {
		name string
		err  error
		says string
	}{
		{
			name: "the platform refusing",
			err:  &OutputError{Doing: "setting", Said: errRefused},
			says: "setting it",
		},
		{
			name: "a device nothing answers to",
			err:  &NoOutputError{Want: "nope", Had: []string{"a", "b"}},
			says: "Attached: a, b",
		},
		{
			// Nothing can play at all, which is a different sentence from a
			// name that matched nothing.
			name: "a device nothing answers to and nothing to play through",
			err:  &NoOutputError{Want: "nope"},
			says: "nothing can play",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().ErrorIs(tt.err, ErrOutput)
			s.Require().Contains(tt.err.Error(), tt.says)
		})
	}
}

func TestOutputTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OutputTestSuite))
}
