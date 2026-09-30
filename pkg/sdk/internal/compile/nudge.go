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
	"fmt"
	"sort"
	"strings"
)

// This file turns a word somebody said into the figures a solve should aim
// differently at.
//
// The vocabulary already holds both halves. An axis says which question a
// word answers, `measures` says which figure that question is settled by, and
// the move table says which way the word points along it. Nothing new is
// invented here: it is the same three tables the compiler reads to move a
// knob, read instead to move a target.

// Nudge is one measured figure a word asks for more or less of.
type Nudge struct {
	// Term is the word that asked.
	Term string
	// Axis is the question it answers, as the vocabulary names it.
	Axis string
	// Key is the measured figure that question is settled by.
	Key string
	// Up is which way the word points along it. "bright" is up on the
	// centroid and "dark" is down the same axis.
	Up bool
}

// Nudges is what a word asks of the measured figures.
//
// A word may answer more than one axis, so it may ask for more than one thing:
// "punchy" is a tight low end and a hard attack, and both are measured.
//
// An error rather than an empty answer for a word that measures nothing,
// because the two are different and look identical. Four of the ten axes have
// no figure at all — how much room is on a part, how loud the strings are
// under a hand, which pickup was used, whether a filter is moving — and a word
// answering only those cannot move a target however much somebody means it.
func Nudges(
	said string,
) ([]Nudge, error) {
	term, axes, ok := known(said)
	if !ok {
		return nil, fmt.Errorf(
			"%q is not a word this vocabulary knows, and words.json is the list",
			said)
	}

	out := []Nudge(nil)

	for _, axis := range axes {
		key, ok := measures[axis]
		if !ok {
			continue
		}

		up, ok := pointsUp(term, axis)
		if !ok {
			continue
		}

		out = append(out, Nudge{Term: term, Axis: axis, Key: key, Up: up})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf(
			"%q answers %s, which nothing measures, so there is no figure to "+
				"aim differently at", said, strings.Join(axes, " and "))
	}

	// Sorted on the figure, so two words asking about the same run of axes
	// report in the same order every run.
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })

	return out, nil
}

// known resolves what somebody said to a word the vocabulary holds.
//
// The vocabulary holds the adjective and a person says the comparative: the
// word is `dark` and the instruction is "darker". Both mean the same axis in
// the same direction, so the comparative is trimmed and tried rather than
// refused, which is the difference between a tool that takes the instruction
// and one that makes somebody look up the spelling.
//
// Only these two forms. Anything else is refused by name, because a nudge that
// silently matched the wrong word would move the wrong knob and say it worked.
func known(
	said string,
) (string, []string, bool) {
	for _, term := range []string{said, strings.TrimSuffix(said, "er"), comparative(said)} {
		if term == "" {
			continue
		}

		if axes, ok := axesOf(term); ok {
			return term, axes, true
		}
	}

	return "", nil, false
}

// comparative turns "punchier" back into "punchy".
func comparative(
	said string,
) string {
	if !strings.HasSuffix(said, "ier") {
		return ""
	}

	return strings.TrimSuffix(said, "ier") + "y"
}

// pointsUp reads which way a word moves along one axis.
//
// From the move table rather than from the vocabulary, because that is where
// the direction lives: `steps` is how far a control turns and its sign is
// which way. A word with no move on that axis says nothing about it.
func pointsUp(
	term, axis string,
) (bool, bool) {
	for _, t := range turns[term] {
		if t.axis == axis && t.steps != 0 {
			return t.steps > 0, true
		}
	}

	return false, false
}
