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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk"
)

// FilledPublicTestSuite covers saying which rig went into which slot.
//
// One line per slot rather than a summary, because each is a flash write on
// somebody's own pedal and the line carries the path whatever was there was
// kept at. A reply that collapses forty writes into "done" has turned forty
// recoverable changes into lost ones.
type FilledPublicTestSuite struct {
	suite.Suite
}

// TestFilled covers what each shape of run says.
//
// One method and one table, so a case is a row rather than a file.
func (s *FilledPublicTestSuite) TestFilled() {
	kept := sdk.Change{
		Action: sdk.Imported,
		To:     sdk.At{Slot: 30, Name: "Punk"},
		Kept:   []string{"/tmp/11A.hlx"},
	}

	for _, tt := range []struct {
		name string
		in   []sdk.Placed
		want []string
		// missing is what must not be said.
		missing []string
	}{
		{
			// Nothing to say, and a count of nothing would be noise: a setup
			// placing no slots is refused before anything is written.
			name:    "a run that placed nothing",
			missing: []string{"flash write"},
		},
		{
			name: "two slots written",
			in: []sdk.Placed{
				{At: "11A", Rig: "punk", Change: kept},
				{At: "11B", Rig: "pop-punk", Change: kept},
			},
			want: []string{
				"punk", "11A", "pop-punk", "11B", "/tmp/11A.hlx",
				"2 of 2 slots written", "flash write",
			},
		},
		{
			// The count is what somebody reads without reading every line,
			// and the failing line names the rig so the document can be
			// fixed. A rig that will not build is not a reason to leave
			// forty slots unwritten.
			name: "one rig that would not build",
			in: []sdk.Placed{
				{At: "11A", Rig: "punk", Change: kept},
				{At: "11B", Rig: "nope", Err: errors.New("no such rig")},
			},
			want: []string{
				"could not", "nope", "11B", "no such rig",
				"1 of 2 slots written",
			},
		},
		{
			name: "every rig failing",
			in: []sdk.Placed{
				{At: "11A", Rig: "nope", Err: errors.New("no such rig")},
			},
			want:    []string{"could not", "0 of 1 slots written"},
			missing: []string{"/tmp/11A.hlx"},
		},
	} {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(cli.Filled(&out, tt.in))

			for _, want := range tt.want {
				s.Require().Contains(out.String(), want)
			}

			for _, missing := range tt.missing {
				s.Require().NotContains(out.String(), missing)
			}
		})
	}
}

// TestFilledReportsAWriterThatFails covers each seam the page is written at.
//
// Every one, because these lines carry the only paths back from a flash write
// and a swallowed error here is a recoverable change silently reported as
// lost.
func (s *FilledPublicTestSuite) TestFilledReportsAWriterThatFails() {
	wrote := []sdk.Placed{{
		At: "11A", Rig: "punk",
		Change: sdk.Change{
			Action: sdk.Imported,
			To:     sdk.At{Slot: 30, Name: "Punk"},
			Kept:   []string{"/tmp/11A.hlx"},
		},
	}}

	failed := []sdk.Placed{
		{At: "11A", Rig: "nope", Err: errors.New("no such rig")},
	}

	for _, tt := range []struct {
		name  string
		in    []sdk.Placed
		after int
	}{
		{name: "the line saying a rig went in", in: wrote},
		{name: "what the write replaced", in: wrote, after: 1},
		{name: "the count", in: wrote, after: 3},
		{name: "the line saying one could not", in: failed},
		{name: "the count after a failure", in: failed, after: 1},
	} {
		s.Run(tt.name, func() {
			s.Require().Error(cli.Filled(&failAfter{n: tt.after}, tt.in))
		})
	}
}

func TestFilledPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FilledPublicTestSuite))
}
