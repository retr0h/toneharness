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
package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk"
)

// ChosenPublicTestSuite covers saying which favourite is now playing.
//
// A cycle only reads if it says both ends of the move: "19B, 2 of 3" leaves
// somebody counting, and "11A → 19B" does not.
type ChosenPublicTestSuite struct {
	suite.Suite
}

// TestChosen covers what each kind of step says.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChosenPublicTestSuite) TestChosen() {
	for _, tt := range []struct {
		name string
		in   sdk.Chosen
		want []string
		// missing is what must not be said.
		missing []string
	}{
		{
			// Both ends, which is every step that recognised where it was.
			name: "a step from one favourite to another",
			in: sdk.Chosen{
				At: "19B", Rig: "mike-dirnt", Position: 2, Of: 3,
				Playing: "punk", From: "11A",
				Change: sdk.Change{
					Action: sdk.Selected,
					To:     sdk.At{Slot: 55, Name: "Mike Dirnt"},
				},
			},
			want: []string{"11A", "19B", "mike-dirnt", "favourite 2 of 3"},
		},
		{
			// Going to the first reads nothing off the device, so there is no
			// move to show and nothing to explain.
			name: "going to the first",
			in: sdk.Chosen{
				At: "11A", Rig: "punk", Position: 1, Of: 3,
				Change: sdk.Change{Action: sdk.Selected},
			},
			want:    []string{"11A", "punk", "favourite 1 of 3"},
			missing: []string{"→", "which no favourite holds"},
		},
		{
			// A person who asked for next and got the first is owed the
			// reason, which is the whole point of carrying Playing.
			name: "a pedal playing something that is no favourite",
			in: sdk.Chosen{
				At: "11A", Rig: "punk", Position: 1, Of: 2,
				Playing: "chunky-monkey",
				Change:  sdk.Change{Action: sdk.Selected},
			},
			want: []string{
				"11A", "chunky-monkey", "which no favourite holds",
				"went to the first",
			},
		},
		{
			// A favourite the slot map says nothing about has no rig to
			// print, and a blank line where one would go reads as a bug.
			name: "a favourite the setup places no rig in",
			in: sdk.Chosen{
				At: "24C", Position: 2, Of: 2, From: "11A",
				Change: sdk.Change{Action: sdk.Selected},
			},
			want:    []string{"11A", "24C", "favourite 2 of 2"},
			missing: []string{"which no favourite holds"},
		},
	} {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(cli.Chosen(&out, tt.in))

			for _, want := range tt.want {
				s.Require().Contains(out.String(), want)
			}

			for _, missing := range tt.missing {
				s.Require().NotContains(out.String(), missing)
			}
		})
	}
}

// TestChosenReportsAWriterThatFails covers each seam the page is written at.
//
// Every one, because a writer that fails partway is the shape a broken pipe
// takes and a swallowed error there is output silently truncated.
func (s *ChosenPublicTestSuite) TestChosenReportsAWriterThatFails() {
	full := sdk.Chosen{
		At: "19B", Rig: "mike-dirnt", Position: 2, Of: 3,
		Playing: "punk", From: "11A",
		Change: sdk.Change{Action: sdk.Selected},
	}

	first := sdk.Chosen{
		At: "11A", Rig: "punk", Position: 1, Of: 2,
		Playing: "chunky-monkey",
		Change:  sdk.Change{Action: sdk.Selected},
	}

	for _, tt := range []struct {
		name  string
		in    sdk.Chosen
		after int
	}{
		{name: "the move itself", in: full},
		{name: "the slot alone, where there was no move", in: first},
		{name: "the rig", in: full, after: 1},
		{name: "which of how many", in: full, after: 2},
		{name: "why it went to the first", in: first, after: 3},
	} {
		s.Run(tt.name, func() {
			s.Require().Error(cli.Chosen(&failAfter{n: tt.after}, tt.in))
		})
	}
}

func TestChosenPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ChosenPublicTestSuite))
}
