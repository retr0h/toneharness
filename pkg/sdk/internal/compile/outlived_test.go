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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// OutlivedTestSuite covers marking the words something wrote over.
//
// The report exists so somebody hand-editing a document can see what their edit
// did, and it printed the step before the edit. A rig stating `Drive` outright
// beats a word, by design and in that order, and a build said "grit-on-attack —
// Drive 0.60 to 0.72" while writing 0.88 into the file.
type OutlivedTestSuite struct {
	suite.Suite
}

// built is a two-block chain whose controls can be set per case.
func (s *OutlivedTestSuite) built(
	drive float64,
	treble float64,
) plan.Plan {
	return plan.Plan{Blocks: []plan.Block{
		{Model: "AmpOne", Params: plan.Params{"Drive": catalog.Float(drive)}},
		{Model: "AmpTwo", Params: plan.Params{"Drive": catalog.Float(treble)}},
	}}
}

// TestOutlived covers which words are marked and which are left alone.
func (s *OutlivedTestSuite) TestOutlived() {
	for _, tt := range []struct {
		name  string
		in    []Moved
		built plan.Plan

		overridden bool
		held       float64
	}{
		{
			// The case this was written for: the word moved it and the
			// document moved it back.
			name:       "a word something wrote over",
			in:         []Moved{{Term: "grit-on-attack", At: 0, Param: "Drive", From: 0.6, To: 0.72}},
			built:      s.built(0.88, 0.5),
			overridden: true,
			held:       0.88,
		},
		{
			// Still where the word left it, so there is nothing to say and
			// the ordinary line stands.
			name:  "a word nothing wrote over",
			in:    []Moved{{Term: "grit-on-attack", At: 0, Param: "Drive", From: 0.6, To: 0.72}},
			built: s.built(0.72, 0.5),
		},
		{
			// By block, not by name. Two blocks carrying a control of the
			// same name is ordinary, and reading the wrong one would mark a
			// word overridden because a different block moved.
			name:  "a control of the same name on another block",
			in:    []Moved{{Term: "grit-on-attack", At: 1, Param: "Drive", From: 0.5, To: 0.5}},
			built: s.built(0.88, 0.5),
		},
		{
			// A word that moved nothing has nothing to be written over.
			name:  "a word that moved nothing",
			in:    []Moved{{Term: "short-decay"}},
			built: s.built(0.88, 0.5),
		},
		{
			// A block index past the chain, which the fit can produce by
			// dropping what a device has no room for. Passed over rather
			// than read, because reading it would panic.
			name:  "a block the chain no longer has",
			in:    []Moved{{Term: "grit-on-attack", At: 9, Param: "Drive", To: 0.72}},
			built: s.built(0.88, 0.5),
		},
		{
			// A control that is no longer a number, which a switch is. Not a
			// value to compare, so nothing is claimed about it.
			name: "a control holding something that is not a number",
			in:   []Moved{{Term: "grit-on-attack", At: 0, Param: "Bright", To: 0.72}},
			built: plan.Plan{Blocks: []plan.Block{{
				Model:  "AmpOne",
				Params: plan.Params{"Bright": catalog.Bool(true)},
			}}},
		},
	} {
		s.Run(tt.name, func() {
			got := append([]Moved(nil), tt.in...)

			outlived(got, tt.built)

			s.Require().Equal(tt.overridden, got[0].Overridden)
			s.Require().Equal(tt.overridden, got[0].Beaten())

			if !tt.overridden {
				s.Require().Zero(got[0].Held)

				return
			}

			s.Require().InDelta(tt.held, got[0].Held, 1e-9)
		})
	}
}

func TestOutlivedTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OutlivedTestSuite))
}
