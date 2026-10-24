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
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// SetupPublicTestSuite covers the tools that read the person's own setup, and
// the two that act on what it says.
//
// These three are the agent surface for "which slots is this managing": the
// slot map and the favourites are the only record of which presets on the
// pedal belong to this tool rather than to whoever is holding it.
type SetupPublicTestSuite struct {
	suite.Suite
	ctrl   *gomock.Controller
	client *mocks.MockClient
}

func (s *SetupPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
	s.client = mocks.NewMockClient(s.ctrl)
}

// setupWith is a Setup placing the slots and listing the favourites given.
func setupWith(
	slots [][2]string,
	favourites []string,
) tone.Setup {
	out := tone.Setup{Schema: tone.SetupSchema, Version: "4.2.0"}

	if len(slots) > 0 {
		placed := make([]tone.Slotted, 0, len(slots))
		for _, pair := range slots {
			placed = append(placed, tone.Slotted{At: pair[0], Rig: pair[1]})
		}

		out.Slots = &placed
	}

	if len(favourites) > 0 {
		out.Favourites = &favourites
	}

	return out
}

// TestDeviceFav covers going to a favourite.
//
// One method and one table, so a case is a row rather than a file.
func (s *SetupPublicTestSuite) TestDeviceFav() {
	for _, tt := range []struct {
		name string
		args any
		// step is what the client should be asked for, where it is asked at
		// all.
		step sdk.Step
		// chosen is what the client answers with.
		chosen sdk.Chosen
		// fails is the error the client answers with instead.
		fails error
		want  string
		err   bool
	}{
		{
			// No step given is the first one, which needs nothing read off
			// the pedal and so cannot be confused by what is loaded.
			name: "no step is the first favourite",
			args: map[string]any{},
			step: sdk.AtFirst,
			chosen: sdk.Chosen{
				At: "11A", Rig: "punk", Position: 1, Of: 3,
			},
			want: "loaded 11A, favourite 1 of 3; no favourite was playing",
		},
		{
			name: "first, said out loud",
			args: map[string]any{"step": "first"},
			step: sdk.AtFirst,
			chosen: sdk.Chosen{
				At: "11A", Rig: "punk", Position: 1, Of: 3,
			},
			want: "loaded 11A, favourite 1 of 3",
		},
		{
			// Both ends of the move, because a cycle only reads if it says
			// what it moved away from.
			name: "next, from a favourite it recognised",
			args: map[string]any{"step": "next"},
			step: sdk.Onward,
			chosen: sdk.Chosen{
				At: "19B", Rig: "mike-dirnt", Position: 2, Of: 3,
				Playing: "punk", From: "11A",
			},
			want: "loaded 19B, favourite 2 of 3, from 11A",
		},
		{
			name: "prev",
			args: map[string]any{"step": "prev"},
			step: sdk.Back,
			chosen: sdk.Chosen{
				At: "11A", Rig: "punk", Position: 1, Of: 3,
				Playing: "mike-dirnt", From: "19B",
			},
			want: "from 19B",
		},
		{
			// Refused before the pedal is touched, because a step nothing
			// understands is a call to fix rather than a device to ask.
			name: "a step that is not one",
			args: map[string]any{"step": "sideways"},
			want: `step is "sideways"; it takes first, next or prev`,
			err:  true,
		},
		{
			name:  "a setup listing no favourites",
			args:  map[string]any{},
			step:  sdk.AtFirst,
			fails: sdk.ErrNoFavourites,
			want:  "the setup names no favourites",
			err:   true,
		},
	} {
		s.Run(tt.name, func() {
			if tt.want != "" && tt.fails == nil && !tt.err {
				s.client.EXPECT().Favourite(gomock.Any(), tt.step).
					Return(tt.chosen, nil).Times(1)
			}

			if tt.fails != nil {
				s.client.EXPECT().Favourite(gomock.Any(), tt.step).
					Return(sdk.Chosen{}, tt.fails).Times(1)
			}

			pedal := mocks.NewMockSession(s.ctrl)
			held(s.client, pedal)

			res := call(s.T(), connect(s.T(), s.client, true), "device_fav", tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if !tt.err {
				var got sdk.Chosen
				structured(s.T(), res, &got)

				s.Equal(tt.chosen.At, got.At)
				s.Equal(tt.chosen.Position, got.Position)
			}
		})
	}
}

// TestSetupShow covers reading the setup back.
func (s *SetupPublicTestSuite) TestSetupShow() {
	for _, tt := range []struct {
		name  string
		held  tone.Setup
		fails error
		want  string
		err   bool
	}{
		{
			// The two counts, because they are what an agent deciding
			// whether to rebuild needs and the rest is in the structured
			// answer.
			name: "a setup placing slots and listing favourites",
			held: setupWith(
				[][2]string{{"11A", "punk"}, {"18A", "mike-dirnt"}},
				[]string{"11A", "18A"}),
			want: "the setup places 2 slots and lists 2 favourites",
		},
		{
			// What setup init writes. Saying zero rather than staying quiet,
			// because "no slots" is the answer to whether a rebuild will
			// overwrite anything.
			name: "a setup with nothing filled in",
			held: setupWith(nil, nil),
			want: "the setup places 0 slots and lists 0 favourites",
		},
		{
			name:  "no setup to read",
			fails: sdk.ErrNoSlots,
			want:  "the setup names no slots",
			err:   true,
		},
	} {
		s.Run(tt.name, func() {
			s.client.EXPECT().Setup().Return(tt.held, tt.fails).Times(1)

			pedal := mocks.NewMockSession(s.ctrl)
			held(s.client, pedal)

			res := call(s.T(), connect(s.T(), s.client, true), "setup_show", map[string]any{})

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if !tt.err {
				var got tone.Setup
				structured(s.T(), res, &got)

				s.Equal(tt.held.Favourites, got.Favourites)
			}
		})
	}
}

// TestSetupInit covers writing a setup somebody can edit.
func (s *SetupPublicTestSuite) TestSetupInit() {
	for _, tt := range []struct {
		name string
		args any
		// at is where the client should be asked to write.
		at      string
		written sdk.Written
		fails   error
		want    string
		err     bool
	}{
		{
			// Nowhere named is where every command looks, which is how this
			// is meant to be run.
			name:    "nowhere named",
			args:    map[string]any{},
			written: sdk.Written{Path: "/home/someone/.config/toneharness/setup.yaml"},
			want:    "wrote /home/someone/.config/toneharness/setup.yaml",
		},
		{
			name:    "somewhere else",
			args:    map[string]any{"at": "/tmp/setup.yaml"},
			at:      "/tmp/setup.yaml",
			written: sdk.Written{Path: "/tmp/setup.yaml"},
			want:    "uncomment what applies",
		},
		{
			// Reported rather than replaced, whatever the server allows. A
			// setup is the only record of decisions nothing else keeps.
			name:  "a setup already there",
			args:  map[string]any{},
			fails: sdk.ErrSetupExists,
			want:  "a setup is already there",
			err:   true,
		},
	} {
		s.Run(tt.name, func() {
			s.client.EXPECT().StartSetup(tt.at).
				Return(tt.written, tt.fails).Times(1)

			pedal := mocks.NewMockSession(s.ctrl)
			held(s.client, pedal)

			res := call(s.T(), connect(s.T(), s.client, true), "setup_init", tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)
		})
	}
}

// TestSlotsFill covers writing every rig a setup places into its slot.
func (s *SetupPublicTestSuite) TestSlotsFill() {
	for _, tt := range []struct {
		name string
		// placed is what the client answers with.
		placed []sdk.Placed
		fails  error
		want   string
		// why is what the failing entry's serialised reason must say.
		why string
		err bool
	}{
		{
			name: "every rig written",
			placed: []sdk.Placed{
				{At: "11A", Rig: "punk"},
				{At: "11B", Rig: "pop-punk"},
			},
			want: "wrote 2 of 2 slots",
		},
		{
			// One rig failing does not stop the rest, and the count is how
			// an agent learns which half happened.
			name: "one rig that would not build",
			placed: []sdk.Placed{
				{At: "11A", Rig: "punk"},
				{At: "11B", Rig: "nope", Err: errors.New("no such rig"), Why: "no such rig"},
			},
			fails: errors.New("no such rig"),
			want:  "wrote 1 of 2 slots",
			// The reason travels as text, because an error marshals to `{}`
			// and an agent reading that learns a rig failed and nothing
			// about which or why.
			why: "no such rig",
		},
		{
			// Nothing written and an error is the one case that refuses:
			// there is no half-done run to report, so the error is the whole
			// answer.
			name:  "a setup that places nothing",
			fails: sdk.ErrNoSlots,
			want:  "the setup names no slots",
			err:   true,
		},
	} {
		s.Run(tt.name, func() {
			s.client.EXPECT().Fill(gomock.Any()).
				Return(tt.placed, tt.fails).Times(1)

			pedal := mocks.NewMockSession(s.ctrl)
			held(s.client, pedal)

			res := call(s.T(), connect(s.T(), s.client, true), "slots_fill", map[string]any{})

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if !tt.err {
				var got []sdk.Placed
				structured(s.T(), res, &got)

				s.Len(got, len(tt.placed))

				if tt.why != "" {
					s.Contains(got[len(got)-1].Why, tt.why)
				}
			}
		})
	}
}

func TestSetupPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SetupPublicTestSuite))
}
