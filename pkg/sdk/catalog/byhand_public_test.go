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

package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

type ByHandPublicTestSuite struct {
	suite.Suite
}

// TestSetByHand covers the controls nothing may set on somebody's behalf.
//
// One method and one table, so a case is a row rather than a file.
func (s *ByHandPublicTestSuite) TestSetByHand() {
	for _, tt := range []struct {
		name  string
		param string
		want  bool
	}{
		{
			// The power supply drooping under load. The one that cost
			// something: a run took it from 0.500 to 1.000 closing band
			// shares and the chain sounded worse than the rig it started
			// from.
			name:  "Sag",
			param: "Sag",
			want:  true,
		},
		{
			name:  "Bias",
			param: "Bias",
			want:  true,
		},
		{
			name:  "BiasX",
			param: "BiasX",
			want:  true,
		},
		{
			name:  "Hum",
			param: "Hum",
			want:  true,
		},
		{
			name:  "Ripple",
			param: "Ripple",
			want:  true,
		},
		{
			// A tone control on the same amplifier, which is the whole point:
			// nothing the device says separates these two. On an SV Beast all
			// twelve controls read `generic_knob` from 0 to 1.
			name:  "a tone control on the same block",
			param: "Treble",
			want:  false,
		},
		{
			name:  "the drive, which is what a solve is for",
			param: "Drive",
			want:  false,
		},
		{
			// Case matters, because the key is the name the catalog gives the
			// parameter and the device addresses it by that spelling.
			name:  "the same name in another case",
			param: "sag",
			want:  false,
		},
		{
			name:  "no name at all",
			param: "",
			want:  false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, catalog.SetByHand(tt.param))
		})
	}
}

// TestSetByHandNamesControlsOnlyAmplifiersCarry is what lets the answer be a
// name rather than a name and a category.
//
// Every one of these is carried by amplifiers and nothing else, so a
// compressor's Level or a delay's Mix can never be caught by it. If Line 6
// ship a block in another category carrying one of these names, this refuses
// a control somebody does want set, and silently: the build succeeds and the
// value is just the one the rig arrived with.
//
// Also that each is still a control something holds. A name no block carries
// is a guard that has quietly stopped guarding.
func (s *ByHandPublicTestSuite) TestSetByHandNamesControlsOnlyAmplifiersCarry() {
	cat, err := catalog.For(catalog.HXStomp)
	s.Require().NoError(err)

	for _, name := range catalog.ValveGuts() {
		s.Run(name, func() {
			held := 0

			for _, block := range cat.Blocks {
				if _, carries := block.Params[name]; !carries {
					continue
				}

				held++

				s.Require().Equalf(
					catalog.CategoryAmp, block.Category,
					"%s carries %s and is not an amplifier, so the answer "+
						"can no longer be a name on its own",
					block.ID, name)
			}

			s.Require().NotZerof(held,
				"no block carries %s, so the guard guards nothing", name)
		})
	}
}

func TestByHandPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ByHandPublicTestSuite))
}
