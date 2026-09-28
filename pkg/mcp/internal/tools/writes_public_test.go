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

	"github.com/retr0h/toneharness/pkg/mcp/internal/tools"
	"github.com/retr0h/toneharness/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/slot"
)

type WritesPublicTestSuite struct {
	suite.Suite
	ctrl   *gomock.Controller
	client *mocks.MockClient
}

func (s *WritesPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
	s.client = mocks.NewMockClient(s.ctrl)
}

// run drives one tool through the table. Every row that succeeds is expected
// to answer an sdk.Change carrying the Replaced value the mock returned,
// which is checked here rather than per row.
func (s *WritesPublicTestSuite) run(
	tool string,
	tests []deviceRow,
) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			pedal := mocks.NewMockSession(s.ctrl)

			if tt.setup != nil {
				tt.setup(s.client, pedal)
			}

			held(s.client, pedal)

			res := call(s.T(), connect(s.T(), s.client, true), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if !tt.err {
				var got sdk.Change
				structured(s.T(), res, &got)

				// A move replaces nothing, so it carries no name. Every
				// other row answers with what the mock returned.
				want := "Old Preset"
				if tt.moved {
					want = ""
				}

				s.Equal(want, got.Replaced)
			}
		})
	}
}

// TestPresetImport covers putting a file into a slot.
func (s *WritesPublicTestSuite) TestPresetImport() {
	s.run("slots_import", []deviceRow{
		{
			name: "a preset into a slot",
			args: tools.Put{Preset: "mike.hlx", Slot: "01A"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Import(gomock.Any(), "mike.hlx", slot.Address{}).
					Return(sdk.Change{Replaced: "Old Preset"}, nil)
			},
			want: "put mike.hlx into 01A",
		},
		{
			name: "a label the pedal does not have",
			args: tools.Put{Preset: "mike.hlx", Slot: "nope"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Put{Preset: "mike.hlx", Slot: "01A"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestPresetsCopy covers copying a slot.
func (s *WritesPublicTestSuite) TestPresetsCopy() {
	s.run("slots_copy", []deviceRow{
		{
			name: "one slot onto another",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Copy(gomock.Any(), slot.Address{}, slot.Address{Slot: 1}).
					Return(sdk.Change{Replaced: "Old Preset"}, nil)
			},
			want: "copied 01A to 01B",
		},
		{
			name: "a source that does not parse",
			args: tools.Move{From: "nope", To: "01B"},
			err:  true,
		},
		{
			name: "a destination that does not parse",
			args: tools.Move{From: "01A", To: "nope"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Move{From: "01A", To: "01B"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestPresetsSwap covers exchanging two slots.
func (s *WritesPublicTestSuite) TestPresetsSwap() {
	s.run("slots_swap", []deviceRow{
		{
			name: "two slots",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Swap(gomock.Any(), slot.Address{}, slot.Address{Slot: 1}).
					Return(sdk.Change{Replaced: "Old Preset"}, nil)
			},
			want: "swapped 01A and 01B",
		},
		{
			// One empty slot is a move, and the tool says that rather than
			// calling it an exchange: the agent asked to swap and got
			// something else.
			name: "one slot holding no preset",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Swap(gomock.Any(), slot.Address{}, slot.Address{Slot: 1}).
					Return(sdk.Change{Action: sdk.MovedPreset}, nil)
			},
			want:  "moved 01A to 01B",
			moved: true,
		},
		{
			// One empty slot is a move and is carried out. Two is nothing to
			// move, and the SDK says so; there is no other tool to suggest,
			// because no tool puts a preset somewhere there is not one.
			name: "two slots holding no preset",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Swap(gomock.Any(), slot.Address{}, slot.Address{Slot: 1}).
					Return(sdk.Change{}, &sdk.EmptySwapError{
						First: slot.Address{}, Second: slot.Address{Slot: 1},
					})
			},
			want: "no preset: 01A and 01B, so there is nothing to swap",
			err:  true,
		},
		{
			name: "a source that does not parse",
			args: tools.Move{From: "nope", To: "01B"},
			err:  true,
		},
		{
			name: "a destination that does not parse",
			args: tools.Move{From: "01A", To: "nope"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Move{From: "01A", To: "01B"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

func TestWritesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WritesPublicTestSuite))
}

// TestRigsNew covers writing a rig and the ask beside it.
//
// Not through run, because run checks every answer for the sdk.Change a slot
// move makes and this answers an sdk.Scaffolded.
func (s *WritesPublicTestSuite) TestRigsNew() {
	s.Run("a rig written from the gear it names", func() {
		s.client.EXPECT().Scaffold(gomock.Any(), sdk.NewRig{
			Genre:      []string{"rock"},
			ID:         "matt-freeman",
			Name:       "Matt Freeman",
			Band:       "Rancid",
			Instrument: "bass",
			Amp:        "Ampeg SVT",
			Pedals:     []string{"Boss ODB-3"},
		}).Return(sdk.Scaffolded{ID: "matt-freeman", Name: "Matt Freeman"}, nil)

		res := call(s.T(), connect(s.T(), s.client, true), "rigs_new", tools.Scaffold{
			Genre:      []string{"rock"},
			ID:         "matt-freeman",
			Name:       "Matt Freeman",
			Band:       "Rancid",
			Instrument: "bass",
			Amp:        "Ampeg SVT",
			Pedals:     []string{"Boss ODB-3"},
		})

		s.False(res.IsError)
		s.Contains(text(s.T(), res), "wrote the rig matt-freeman and the ask beside it")

		var got sdk.Scaffolded
		structured(s.T(), res, &got)
		s.Equal("matt-freeman", got.ID)
	})

	s.Run("gear the catalog does not carry", func() {
		s.client.EXPECT().Scaffold(gomock.Any(), gomock.Any()).
			Return(sdk.Scaffolded{}, errors.New("no model for Marshall Nonesuch"))

		res := call(s.T(), connect(s.T(), s.client, true), "rigs_new", tools.Scaffold{
			Genre: []string{"rock"},
			ID:    "nobody", Name: "Nobody", Instrument: "bass", Amp: "Marshall Nonesuch",
		})

		s.True(res.IsError)
		s.Contains(text(s.T(), res), "no model for Marshall Nonesuch")
	})
}

// TestPresetsCompile covers turning a rig or a plan into a preset.
//
// The refusal matters more than the two happy paths. A rig names gear and is
// realised on the way through; a plan already names the models and every knob.
// Given both, nothing can say which one the caller meant, and quietly picking
// one writes a preset somebody did not ask for.
func (s *WritesPublicTestSuite) TestPresetsCompile() {
	s.Run("a rig, realised against the catalog", func() {
		s.client.EXPECT().Compile(gomock.Any(), sdk.Compile{
			Rig: "mine.yaml", Out: "mine.hlx", Existing: sdk.ReplaceExisting,
		}).Return(sdk.Built{Path: "mine.hlx", Blocks: 4}, nil)

		res := call(s.T(), connect(s.T(), s.client, true), "presets_compile",
			tools.Build{Rig: "mine.yaml", Out: "mine.hlx"})

		s.False(res.IsError)
		s.Contains(text(s.T(), res), "mine.hlx holds 4 blocks")

		var got sdk.Built
		structured(s.T(), res, &got)
		s.Equal(4, got.Blocks)
	})

	s.Run("a plan, which already chose its models", func() {
		s.client.EXPECT().Compile(gomock.Any(), sdk.Compile{
			Plan: "tuned.yaml", Template: "slot.hlx", Out: "tuned.hlx",
			Existing: sdk.ReplaceExisting,
		}).Return(sdk.Built{Path: "tuned.hlx", Blocks: 6}, nil)

		res := call(s.T(), connect(s.T(), s.client, true), "presets_compile",
			tools.Build{Plan: "tuned.yaml", Template: "slot.hlx", Out: "tuned.hlx"})

		s.False(res.IsError)
		s.Contains(text(s.T(), res), "tuned.hlx holds 6 blocks")
	})

	s.Run("both a rig and a plan", func() {
		res := call(s.T(), connect(s.T(), s.client, true), "presets_compile",
			tools.Build{Rig: "mine.yaml", Plan: "tuned.yaml", Out: "mine.hlx"})

		s.True(res.IsError)
		s.Contains(text(s.T(), res), tools.ErrOneDocument.Error())
	})

	s.Run("neither a rig nor a plan", func() {
		res := call(s.T(), connect(s.T(), s.client, true), "presets_compile",
			tools.Build{Out: "mine.hlx"})

		s.True(res.IsError)
		s.Contains(text(s.T(), res), tools.ErrOneDocument.Error())
	})

	s.Run("a rig the compiler refuses", func() {
		s.client.EXPECT().Compile(gomock.Any(), gomock.Any()).
			Return(sdk.Built{}, errors.New("over the DSP budget"))

		res := call(s.T(), connect(s.T(), s.client, true), "presets_compile",
			tools.Build{Rig: "mine.yaml", Out: "mine.hlx"})

		s.True(res.IsError)
		s.Contains(text(s.T(), res), "over the DSP budget")
	})
}
