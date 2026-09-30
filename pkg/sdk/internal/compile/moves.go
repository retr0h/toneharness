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

package compile

import (
	"errors"
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// ErrMovesAndControllers reports moves written over a plan's own assignments.
//
// A move is what somebody wants and a controller is what a device stored.
// Building from both would mean quietly picking one, the same reason a rig
// carries sections or snapshots and never both.
var ErrMovesAndControllers = errors.New(
	"a rig carries moves or a plan's controllers, not both")

// ErrMoveTwice reports two moves claiming the same mover.
//
// One expression pedal cannot be on two knobs at once. A count against the
// device is not the check here, because nothing establishes how many
// controllers a family offers and a guessed limit would refuse rigs that load.
var ErrMoveTwice = errors.New("two moves claim the same mover")

// controllerFor is the number a device files each kind of mover under.
//
// An HX Stomp's expression pedal is 2, and a footswitch set to sweep rather
// than switch is 1. Written here rather than in a rig, because the number is
// this family's and the rig says what moves the control instead.
var controllerFor = map[rig.MoveBy]int{
	rig.MoveByExpression: 2,
	rig.MoveByFootswitch: 1,
}

// Moves turns what a rig says a foot reaches into the assignments a plan holds.
//
// The counterpart of Sections, and it resolves the same two things: a role
// becomes the first block in the built chain the catalog files under it, and a
// setting word becomes whichever parameter that model answers the word with.
// Both are what makes a move portable, because neither a block's position nor
// one manufacturer's parameter name exists until a device has been chosen.
//
// Run against the chain as built and after the fit, because filling adds
// blocks and fitting moves them: a move resolved against the rig as written
// would name a position that means something else by the time it is used.
//
// Everything is checked before anything is appended, so a rig that will not
// build leaves the plan as it was.
func Moves(
	made *plan.Plan,
	spec rig.Spec,
	blocks []plan.Block,
	cat *catalog.Catalog,
) error {
	if spec.Moves == nil || len(*spec.Moves) == 0 {
		return nil
	}

	// A plan read off a device already says what its pedal moves. Turning a
	// rig's moves into more of them would leave the device holding two sets
	// nobody meant to combine.
	if len(made.Controllers) > 0 {
		return ErrMovesAndControllers
	}

	moves := *spec.Moves

	roles := rolesOf(blocks, cat)
	claimed := map[rig.MoveBy]int{}
	out := make([]rig.Controller, 0, len(moves))
	errs := []error(nil)

	for i, mv := range moves {
		field := fmt.Sprintf("moves[%d]", i)

		if first, ok := claimed[mv.By]; ok {
			errs = append(errs, fmt.Errorf("%w: %s.by is %s, already claimed by moves[%d]",
				ErrMoveTwice, field, mv.By, first))

			continue
		}

		claimed[mv.By] = i

		got, err := resolveMove(mv, blocks, roles, cat, field)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		out = append(out, got)
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	made.Controllers = out

	return nil
}

// resolveMove turns one move into the assignment a device understands.
func resolveMove(
	mv rig.Move,
	blocks []plan.Block,
	roles []rig.Role,
	cat *catalog.Catalog,
	field string,
) (rig.Controller, error) {
	at := -1

	for b, role := range roles {
		if role == mv.Role {
			at = b

			break
		}
	}

	if at < 0 {
		return rig.Controller{}, &NoSuchValueError{
			Field: field + ".role",
			Value: string(mv.Role),
			Near:  present(roles),
			Whole: true,
		}
	}

	// The catalog carries it, because rolesOf only gives a block a role when
	// it does: a model this catalog has never seen has no role, so the lookup
	// above has already refused the move.
	blk, _ := cat.Block(blocks[at].Model)

	key, ok := controlNamed(blk, string(mv.Setting))
	if !ok {
		return rig.Controller{}, &NoSuchValueError{
			Field: field + ".setting",
			Value: string(mv.Setting),
			Near:  takes(blk),
			Whole: true,
		}
	}

	return rig.Controller{
		Controller: controllerFor[mv.By],
		Block:      blocks[at].Pos,
		Parameter:  key,
		Min:        mv.Min,
		Max:        mv.Max,
		NoSnapshot: mv.NoSnapshot,
	}, nil
}

// controlNamed is the parameter a model answers one setting word with.
//
// The same resolution a setting gets, so a move and a setting on the same word
// reach the same knob. Anything else would let a rig set `drive` on one
// control and sweep another.
func controlNamed(
	blk catalog.Block,
	word string,
) (string, bool) {
	for _, k := range knobWords {
		if k.word != word {
			continue
		}

		return controlFor(blk, k)
	}

	return "", false
}
