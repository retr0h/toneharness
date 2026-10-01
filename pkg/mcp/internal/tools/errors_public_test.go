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

package tools_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/mcp/internal/tools"
	"github.com/retr0h/toneharness/pkg/sdk"
)

// ErrorsPublicTestSuite covers naming the tool to call next.
type ErrorsPublicTestSuite struct {
	suite.Suite
}

// TestRemedy covers every error that has a next step, and the ones that do not.
//
// One method and one table, so a case is a row rather than a file. Registration
// covers the device half by having registered a tool, which is the right test
// for "every tool that opens the pedal says what to do next" and reaches none of
// the errors no tool was mocked to return.
func (s *ErrorsPublicTestSuite) TestRemedy() {
	// An error with no next step, so the wrapper has something to hand back
	// unchanged.
	other := errors.New("the editor interface is in use, quit HX Edit")

	for _, tt := range []struct {
		name string
		err  error
		// want is a fragment the answer must carry, and is what an agent acts
		// on, so it names a tool rather than a command.
		want string
		// is is the sentinel the answer must still match, because a remedy
		// that swallowed what it wrapped would leave every caller's errors.Is
		// returning false.
		is error
	}{
		{
			name: "a block nothing models",
			err:  fmt.Errorf("%w", sdk.ErrNoSuchBlock),
			want: "catalog_list",
			is:   sdk.ErrNoSuchBlock,
		},
		{
			name: "a rig nobody has",
			err:  fmt.Errorf("%w", sdk.ErrNoSuchRig),
			want: "rigs_list",
			is:   sdk.ErrNoSuchRig,
		},
		{
			// A pedal on a charger rather than a data port looks exactly like
			// one that is switched off, and the power light is already on.
			name: "no pedal on the bus",
			err:  fmt.Errorf("%w", sdk.ErrNoDevice),
			want: "USB data port",
			is:   sdk.ErrNoDevice,
		},
		{
			// Retrying is usually enough. When it is not the endpoint has
			// stalled, and only a power cycle clears it, so an agent that does
			// not know retries into a wall.
			name: "the session the bus ended",
			err:  fmt.Errorf("%w: writing to control", sdk.ErrBus),
			want: "powered off and on",
			is:   sdk.ErrBus,
		},
		{
			// Qualified on purpose. Naming a player resolves their rig where
			// somebody has researched them, so offering "a player" unqualified
			// sent an agent back to do what it had already done.
			name: "a request with nothing in it to resolve",
			err:  fmt.Errorf("%w", sdk.ErrNothingToBuildFrom),
			want: "a player somebody has researched",
			is:   sdk.ErrNothingToBuildFrom,
		},
		{
			name: "a slot holding nothing",
			err:  fmt.Errorf("%w", sdk.ErrEmptySlot),
			want: "slots_list",
			is:   sdk.ErrEmptySlot,
		},
		{
			// Handed back as it was, so a tool that fails for its own reasons
			// still says its own thing.
			name: "anything else",
			err:  other,
			want: "quit HX Edit",
			is:   other,
		},
	} {
		s.Run(tt.name, func() {
			got := tools.Remedy(tt.err)

			s.Require().ErrorContains(got, tt.want)
			s.Require().ErrorIs(got, tt.is)
		})
	}
}

func TestErrorsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
