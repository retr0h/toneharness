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
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// KnobsPublicTestSuite covers a rig's musical words reaching real controls.
type KnobsPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *KnobsPublicTestSuite) SetupSuite() { s.cat = loadCatalog(&s.Suite) }

// dial returns a control counted from lo to hi.
func dial(
	kind catalog.ParamType,
	lo, hi float64,
) catalog.Param {
	def := catalog.Float(lo)
	if kind == catalog.ParamInt {
		def = catalog.Int(int64(lo))
	}

	return catalog.Param{Type: kind, Min: lo, Max: hi, Default: def}
}

// knob returns a pointer to one of a rig's settings.
func knob(
	v float64,
) *rig.Knob {
	out := rig.Knob(v)

	return &out
}

// TestSetKnobs covers which control a word lands on, and what it reads.
func (s *KnobsPublicTestSuite) TestSetKnobs() {
	tests := []struct {
		name   string
		params map[string]catalog.Param
		set    *rig.Settings
		want   plan.Params
		errs   []string
	}{
		{
			name:   "a rig that says nothing",
			params: map[string]catalog.Param{"Drive": dial(catalog.ParamFloat, 0, 1)},
			want:   plan.Params{},
		},
		{
			name: "the words an amplifier answers to",
			params: map[string]catalog.Param{
				"Drive": dial(catalog.ParamFloat, 0, 1),
				"Bass":  dial(catalog.ParamFloat, 0, 1),
				"Mid":   dial(catalog.ParamFloat, 0, 1),
			},
			set: &rig.Settings{Drive: knob(0.47), Bass: knob(0.52), Mid: knob(0.71)},
			want: plan.Params{
				"Drive": catalog.Float(0.47),
				"Bass":  catalog.Float(0.52),
				"Mid":   catalog.Float(0.71),
			},
		},
		{
			// One manufacturer calls the same control different things on
			// different models, so a word reaches whichever of them is there.
			name: "controls this model calls something else",
			params: map[string]catalog.Param{
				"Low":   dial(catalog.ParamFloat, 0, 1),
				"High":  dial(catalog.ParamFloat, 0, 1),
				"ChVol": dial(catalog.ParamFloat, 0, 1),
			},
			set: &rig.Settings{Bass: knob(0.25), Treble: knob(0.75), Level: knob(1)},
			want: plan.Params{
				"Low":   catalog.Float(0.25),
				"High":  catalog.Float(0.75),
				"ChVol": catalog.Float(1),
			},
		},
		{
			// A rig counts every word from 0 to 1 whatever the device counts
			// the control in, so halfway up a control from -12 to 12 is 0.
			name:   "a control counted in something other than 0 to 1",
			params: map[string]catalog.Param{"Level": dial(catalog.ParamFloat, -12, 12)},
			set:    &rig.Settings{Level: knob(0.5)},
			want:   plan.Params{"Level": catalog.Float(0)},
		},
		{
			// A device handed 4.7 for a control counting whole steps refuses
			// the preset rather than rounding it.
			name:   "a control counted in whole steps",
			params: map[string]catalog.Param{"Level": dial(catalog.ParamInt, 0, 10)},
			set:    &rig.Settings{Level: knob(0.47)},
			want:   plan.Params{"Level": catalog.Int(5)},
		},
		{
			// Halfway up a switch is not a position, so the switch is not a
			// control this word can reach and the model is treated as having
			// none.
			name: "a switch by that name",
			params: map[string]catalog.Param{
				"Bright": {Type: catalog.ParamBool, Default: catalog.Bool(false)},
				"Drive":  dial(catalog.ParamFloat, 0, 1),
			},
			set:  &rig.Settings{Treble: knob(1)},
			errs: []string{`no "treble"`, "it has: drive"},
		},
		{
			// Every complaint at once. A rig with two words this model has no
			// control for took two runs to fix when this reported the first.
			name:   "words this model has no control for",
			params: map[string]catalog.Param{"Drive": dial(catalog.ParamFloat, 0, 1)},
			set:    &rig.Settings{Presence: knob(0.4), Mix: knob(0.2)},
			errs:   []string{`no "presence"`, `no "mix"`, "it has: drive"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			blk := catalog.Block{ID: "HD2_Test", Params: tt.params}
			got := plan.Params{}

			err := compile.SetKnobs(got, blk, tt.set, "chain[0].settings")

			if len(tt.errs) > 0 {
				s.Require().Error(err)
				s.Require().ErrorIs(err, compile.ErrNoSuchValue)

				for _, want := range tt.errs {
					s.Require().Contains(err.Error(), want)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestLowerSetsWhatTheRigSaid covers the words reaching a compiled preset.
//
// A plan carries knob positions rather than words, so the words land while the
// rig is resolved. What is held here is that they survive the write.
func (s *KnobsPublicTestSuite) TestLowerSetsWhatTheRigSaid() {
	spec := bassRig("Ampeg SVT", "")
	spec.Chain[0].Settings = &rig.Settings{Drive: knob(0.47)}

	made, _, _, _, err := compile.Resolve("a-rig", spec, compile.Intent{}, s.cat, nil)
	s.Require().NoError(err)

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().NoError(compile.Lower(doc, made, s.cat))

	built, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().Equal(catalog.Float(0.47), built.Blocks[0].Params["Drive"])
}

// TestResolve covers what a rig's own values and words do to a built chain.
//
// One method and one table, so a case is a row rather than a file.
func (s *KnobsPublicTestSuite) TestResolve() {
	for _, tt := range []struct {
		name string
		// set is the seven-word vocabulary, controls the gear's real controls.
		set      *rig.Settings
		controls map[string]catalog.Setting
		// want is the value one control has to end up at.
		at   string
		want catalog.ParamValue
		err  error
	}{
		{
			name: "a word the gear has a control for",
			set:  &rig.Settings{Drive: knob(0.47)},
			at:   "Drive",
			want: catalog.Float(0.47),
		},
		{
			// The same refusal on the build path, where the block is one the
			// catalog chose rather than one somebody named.
			name: "a word the gear has no control for",
			set:  &rig.Settings{Presence: knob(0.4)},
			err:  compile.ErrNoSuchValue,
		},
		{
			// The point of stating controls: the value is used as it stands,
			// with nothing re-derived.
			name:     "a control stated outright",
			controls: map[string]catalog.Setting{"Drive": catalog.Set(catalog.Float(0.8))},
			at:       "Drive",
			want:     catalog.Float(0.8),
		},
		{
			// A value beats a word, because a word is a request and a value is
			// an answer. Both naming the same control is possible and the
			// value wins.
			name:     "a control and a word for the same thing",
			set:      &rig.Settings{Drive: knob(0.47)},
			controls: map[string]catalog.Setting{"Drive": catalog.Set(catalog.Float(0.9))},
			at:       "Drive",
			want:     catalog.Float(0.9),
		},
		{
			name:     "a control the model does not have",
			controls: map[string]catalog.Setting{"NotAKnob": catalog.Set(catalog.Float(1))},
			err:      compile.ErrNoSuchValue,
		},
		{
			// Checked against the catalog's range, because the device refuses a
			// whole preset over one value and a message naming the control is a
			// better way to learn that.
			name:     "a control past the end of its range",
			controls: map[string]catalog.Setting{"Drive": catalog.Set(catalog.Float(99999))},
			err:      catalog.ErrBadParam,
		},
		{
			// A switch has no range to be outside of, so nothing invents one and
			// nothing refuses it for being unlike a number.
			name:     "a control whose value is not a number",
			controls: map[string]catalog.Setting{"Drive": catalog.Set(catalog.Bool(true))},
			at:       "Drive",
			want:     catalog.Bool(true),
		},
	} {
		s.Run(tt.name, func() {
			spec := bassRig("Ampeg SVT", "")
			spec.Chain[0].Settings = tt.set

			if tt.controls != nil {
				held := tt.controls
				spec.Chain[0].Controls = &held
			}

			built, _, _, _, err := compile.Resolve(
				"a-rig", spec, compile.Intent{}, s.cat, nil)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, built.Blocks[0].Params[tt.at])
		})
	}
}

func TestKnobsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KnobsPublicTestSuite))
}
