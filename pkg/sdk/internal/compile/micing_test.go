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
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// MicingTestSuite covers the two values a cabinet keeps in fields of its own.
//
// The contract puts `mic` and `distance` beside `controls` rather than in it,
// on the reasoning that the device stores them as attributes and the catalog
// carries no range. That is true of the legacy cabinets and false of the mic'd
// impulse responses every rig here builds with, so a stated microphone went
// into an attribute nothing reads while the corpus decided the parameter: a
// document asking for microphone 4 built one on microphone 11.
//
// Both shapes are fixtures here, because the fix has to leave the legacy
// cabinet alone.
type MicingTestSuite struct {
	suite.Suite
}

// cabs is a mic'd impulse response carrying both as parameters, and the legacy
// cabinet of the same speaker carrying only one.
func (s *MicingTestSuite) cabs() []catalog.Block {
	return []catalog.Block{
		{
			ID: "MicIr", Category: catalog.CategoryCab,
			Params: map[string]catalog.Param{
				micControl: {
					Key: micControl, Type: catalog.ParamInt,
					Min: 0, Max: 11, Default: catalog.Int(3),
				},
				distanceControl: {
					Key: distanceControl, Type: catalog.ParamFloat,
					Min: 1, Max: 12, Default: catalog.Float(5),
				},
			},
		},
		{
			// No microphone of its own, which is the legacy shape.
			ID: "Legacy", Category: catalog.CategoryCab,
			Params: map[string]catalog.Param{
				distanceControl: {
					Key: distanceControl, Type: catalog.ParamFloat,
					Min: 1, Max: 12, Default: catalog.Float(1),
				},
			},
		},
	}
}

// asked is one chain entry stating a microphone, a distance, or neither.
func (s *MicingTestSuite) asked(
	mic *int,
	distance *float64,
) []*wanted {
	return []*wanted{
		{entry: &rig.ChainEntry{Mic: mic, Distance: distance}},
		{entry: &rig.ChainEntry{Mic: mic, Distance: distance}},
	}
}

// built is two blocks at whatever the catalog says, which is what the corpus
// and the defaults would have left.
func (s *MicingTestSuite) built() []plan.Block {
	return []plan.Block{
		{Model: "MicIr", Params: plan.Params{
			micControl:      catalog.Int(11),
			distanceControl: catalog.Float(1),
		}},
		{Model: "Legacy", Params: plan.Params{
			distanceControl: catalog.Float(1),
		}},
	}
}

// TestStatedMicing covers what a document's own microphone and distance do.
//
// One method and one table, so a case is a row rather than a file.
func (s *MicingTestSuite) TestStatedMicing() {
	at := func(i int) *int { return &i }
	to := func(f float64) *float64 { return &f }

	for _, tt := range []struct {
		name string
		mic  *int
		dist *float64
		// micIr and legacy are what each block should hold afterwards, by
		// control name.
		micIr  map[string]catalog.ParamValue
		legacy map[string]catalog.ParamValue
		says   string
	}{
		{
			// The bug. Stated 4, built 11, and every `settings:` field beside
			// it survived, so nothing looked wrong.
			name: "a stated microphone beats what the corpus left",
			mic:  at(4),
			micIr: map[string]catalog.ParamValue{
				micControl: catalog.Int(4),
			},
		},
		{
			name: "and a stated distance",
			dist: to(2.5),
			micIr: map[string]catalog.ParamValue{
				distanceControl: catalog.Float(2.5),
			},
			legacy: map[string]catalog.ParamValue{
				distanceControl: catalog.Float(2.5),
			},
		},
		{
			name: "both at once",
			mic:  at(0),
			dist: to(12),
			micIr: map[string]catalog.ParamValue{
				micControl:      catalog.Int(0),
				distanceControl: catalog.Float(12),
			},
		},
		{
			// A cabinet with no microphone of its own is not an error. The
			// attribute still carries it for whatever reads that, which is
			// what happened before this existed.
			name: "a cabinet with no microphone keeps its distance and says nothing",
			mic:  at(4),
			dist: to(3),
			legacy: map[string]catalog.ParamValue{
				distanceControl: catalog.Float(3),
			},
		},
		{
			// A float past the end of its control is refused here and named,
			// because a device handed one refuses the whole preset and
			// finding that out from the pedal is worse than from a message.
			name: "a distance the control cannot take is refused",
			dist: to(99),
			says: distanceControl,
		},
		{
			// An integer is not, and that is `within`'s doing rather than
			// this file's: it reads a value as a float and an int answers no,
			// so it returns without comparing. The build still refuses the
			// preset — `plan.Validate` says `parameter "Mic": 20 out of range
			// [0, 11]` — so nothing reaches a device, and the message names
			// the control either way. Recorded rather than changed, because
			// `statedControls` leans on the same gap and widening `within`
			// belongs with that rather than here.
			name: "an integer past its range is left to the budget check",
			mic:  at(20),
			micIr: map[string]catalog.ParamValue{
				micControl: catalog.Int(20),
			},
		},
		{
			// Which is every rig the marketplace ships: none of them states
			// either, so nothing here moves for them.
			name: "a document stating neither changes nothing",
			micIr: map[string]catalog.ParamValue{
				micControl:      catalog.Int(11),
				distanceControl: catalog.Float(1),
			},
		},
	} {
		s.Run(tt.name, func() {
			built := s.built()

			err := statedMicing(built, s.cabs(), s.asked(tt.mic, tt.dist))

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)

			for name, want := range tt.micIr {
				s.Require().Equal(want, built[0].Params[name], "MicIr's %s", name)
			}

			for name, want := range tt.legacy {
				s.Require().Equal(want, built[1].Params[name], "Legacy's %s", name)
			}
		})
	}
}

// TestStatedMicingSkipsWhatItCannotRead covers the rows that are not a chain
// entry at all.
func (s *MicingTestSuite) TestStatedMicingSkipsWhatItCannotRead() {
	built := s.built()
	at := 4

	// A nil row, a row with no entry, and more rows than there are blocks:
	// none is a state a resolve reaches, and each is a panic if it ever does.
	s.Require().NoError(statedMicing(built, s.cabs(), []*wanted{
		nil,
		{},
		{entry: &rig.ChainEntry{Mic: &at}},
	}))

	s.Require().Equal(catalog.Int(11), built[0].Params[micControl],
		"a nil row writes nothing")
}

func TestMicingTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MicingTestSuite))
}
