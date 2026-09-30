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

package compile_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// MovesPublicTestSuite covers a rig saying what a foot reaches becoming the
// assignment a device understands.
type MovesPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *MovesPublicTestSuite) SetupSuite() { s.cat = loadCatalog(&s.Suite) }

// chain is the blocks a move resolves against: an amplifier with a Drive and a
// Bass, a drive pedal with a Gain, and a cabinet with nothing a word reaches.
func (s *MovesPublicTestSuite) chain() []plan.Block {
	return []plan.Block{
		{Model: "HD2_AmpSVBeastNrm", Pos: 0},
		{Model: "HD2_DistMinotaur", Pos: 1},
		{Model: "HD2_Cab8x10SVBeast", Pos: 2},
	}
}

// specWith is a rig carrying only the moves under test.
func specWith(
	moves ...rig.Move,
) rig.Spec {
	return rig.Spec{Moves: &moves}
}

// TestMoves covers Moves, which turns what a rig says a foot reaches into the
// assignments a plan holds.
//
// One method and one table, so a case is a row rather than a file.
func (s *MovesPublicTestSuite) TestMoves() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "moves resolve to assignments",
			then: func() {
				tests := []struct {
					name string
					spec rig.Spec
					want []rig.Controller
				}{
					{
						// Nothing said is not the same as nothing moving, so the plan is
						// left as it was rather than given an empty list.
						name: "a rig that says nothing",
						spec: rig.Spec{},
					},
					{
						name: "a rig with an empty list",
						spec: specWith(),
					},
					{
						// The expression pedal on the amplifier's drive, which is what
						// `drive` reaches on a model that has a Drive.
						name: "a pedal on the amplifier's drive",
						spec: specWith(rig.Move{
							By: rig.MoveByExpression, Role: "amp", Setting: "drive",
							Min: sweep(0.3), Max: sweep(0.85), NoSnapshot: &yes,
						}),
						want: []rig.Controller{{
							Controller: 2, Block: 0, Parameter: "Drive",
							Min: sweep(0.3), Max: sweep(0.85), NoSnapshot: &yes,
						}},
					},
					{
						// The same word on a model that calls the control Gain, which is
						// the whole reason a move names a word rather than a parameter.
						name: "the same word reaching a Gain",
						spec: specWith(rig.Move{
							By: rig.MoveByFootswitch, Role: "drive", Setting: "drive",
						}),
						want: []rig.Controller{{Controller: 1, Block: 1, Parameter: "Gain"}},
					},
					{
						// Two movers on two knobs, which is legal: what is refused is two
						// claims on one mover.
						name: "a pedal and a switch on different knobs",
						spec: specWith(
							rig.Move{By: rig.MoveByExpression, Role: "amp", Setting: "drive"},
							rig.Move{By: rig.MoveByFootswitch, Role: "amp", Setting: "bass"},
						),
						want: []rig.Controller{
							{Controller: 2, Block: 0, Parameter: "Drive"},
							{Controller: 1, Block: 0, Parameter: "Bass"},
						},
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						made := plan.Plan{}

						s.Require().NoError(compile.Moves(&made, tt.spec, s.chain(), s.cat))
						s.Equal(tt.want, made.Controllers)
					})
				}
			},
		},
		{
			// point: the alternative is a preset where the pedal under somebody's foot is
			// on a control that is not there.
			name: "moves refuse what cannot be built",
			then: func() {
				tests := []struct {
					name   string
					spec   rig.Spec
					blocks []plan.Block
					want   string
				}{
					{
						name: "a role the chain has no block for",
						spec: specWith(rig.Move{
							By: rig.MoveByExpression, Role: "reverb", Setting: "mix",
						}),
						want: "reverb",
					},
					{
						name: "a word the model has no control for",
						spec: specWith(rig.Move{
							By: rig.MoveByExpression, Role: "cab", Setting: "drive",
						}),
						want: "drive",
					},
					{
						// Not a word this vocabulary has at all, as distinct from a word
						// the model has no control for. The contract's enum keeps this out
						// of a file, and a caller reaching the SDK directly is not held to
						// it, so the refusal has to be here too.
						name: "a setting that is not one of the words",
						spec: specWith(rig.Move{
							By: rig.MoveByExpression, Role: "amp", Setting: "sparkle",
						}),
						want: "sparkle",
					},
					{
						name: "two moves claiming one mover",
						spec: specWith(
							rig.Move{By: rig.MoveByExpression, Role: "amp", Setting: "drive"},
							rig.Move{By: rig.MoveByExpression, Role: "amp", Setting: "bass"},
						),
						want: "already claimed by moves[0]",
					},
					{
						// A model the catalog has never seen has no role either, so it is
						// the role that refuses rather than a missing catalog entry.
						name: "a block whose model the catalog does not carry",
						spec: specWith(rig.Move{
							By: rig.MoveByExpression, Role: "amp", Setting: "drive",
						}),
						blocks: []plan.Block{{Model: "HD2_NotAModel", Pos: 0}},
						want:   `no "amp"`,
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						made := plan.Plan{}

						blocks := tt.blocks
						if blocks == nil {
							blocks = s.chain()
						}

						err := compile.Moves(&made, tt.spec, blocks, s.cat)
						s.Require().Error(err)
						s.Contains(err.Error(), tt.want)
						s.Nil(made.Controllers, "nothing is written when anything is refused")
					})
				}
			},
		},
		{
			// Building from both would leave the pedal holding two sets nobody combined.
			name: "moves refuse a plan that already assigns",
			then: func() {
				made := plan.Plan{Controllers: []rig.Controller{
					{Controller: 2, Block: 0, Parameter: "Drive"},
				}}

				err := compile.Moves(&made,
					specWith(rig.Move{By: rig.MoveByExpression, Role: "amp", Setting: "drive"}),
					s.chain(), s.cat)

				s.Require().ErrorIs(err, compile.ErrMovesAndControllers)
				s.Len(made.Controllers, 1, "what the plan already said is left alone")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestMovesRefuseWhatCannotBeBuilt covers every way a move does not resolve.
//

// TestMovesRefuseAPlanThatAlreadyAssigns covers a rig's moves over a plan's own.
//

func TestMovesPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(MovesPublicTestSuite))
}
