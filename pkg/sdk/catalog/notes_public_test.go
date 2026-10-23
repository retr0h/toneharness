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

type NotesPublicTestSuite struct {
	suite.Suite
}

// TestChangesTheNotes covers the refusal a build applies to its own
// candidates.
//
// One method and one table, so a case is a row rather than a file.
func (s *NotesPublicTestSuite) TestChangesTheNotes() {
	for _, tt := range []struct {
		name  string
		block catalog.Block
		want  bool
	}{
		{
			// The category is Line 6's own grouping for the blocks that do
			// nothing else, so no identifier is needed to place one.
			name:  "the whole pitch category, whatever the model is",
			block: catalog.Block{ID: "HD2_PitchSimplePitch", Category: catalog.CategoryPitch},
			want:  true,
		},
		{
			name:  "a synth, which Line 6 also file under pitch",
			block: catalog.Block{ID: "HD2_SynthSubtractive", Category: catalog.CategoryPitch},
			want:  true,
		},
		{
			// The case the whole refusal exists for: a fuzz by category and
			// an octave up by what it does.
			name: "an octave fuzz, which the device files as a drive",
			block: catalog.Block{
				ID: "HD2_DM4OctaveFuzz", Category: catalog.CategoryDrive,
				BasedOn: "Tycobrahe Octavia",
			},
			want: true,
		},
		{
			name: "an octave down, filed as a drive too",
			block: catalog.Block{
				ID: "HD2_DM4SubOctFuzz", Category: catalog.CategoryDrive,
				BasedOn: "PAiA Roctave Divider",
			},
			want: true,
		},
		{
			name: "a ring modulator, which the device files as modulation",
			block: catalog.Block{
				ID: "HD2_MM4RingModulator", Category: catalog.CategoryMod,
			},
			want: true,
		},
		{
			name: "a harmonised delay, which the device files as a delay",
			block: catalog.Block{
				ID: "HD2_DelayHarmonyDelay", Category: catalog.CategoryDelay,
			},
			want: true,
		},
		{
			// Distortion answers with harmonics of what went in, however
			// much of it there is, so the note is still the note.
			name: "a fuzz that only distorts",
			block: catalog.Block{
				ID: "HD2_DistTriangleFuzz", Category: catalog.CategoryDrive,
				BasedOn: "Electro-Harmonix Big Muff Pi",
			},
			want: false,
		},
		{
			name:  "a chorus, which moves the pitch and puts nothing new in",
			block: catalog.Block{ID: "HD2_Chorus", Category: catalog.CategoryMod},
			want:  false,
		},
		{
			name: "an amplifier",
			block: catalog.Block{
				ID: "HD2_AmpAmpegSVTNrm", Category: catalog.CategoryAmp,
			},
			want: false,
		},
		{
			name:  "nothing to go on",
			block: catalog.Block{},
			want:  false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.block.ChangesTheNotes())
		})
	}
}

// TestChangesTheNotesNamesModelsThisDeviceHolds is the guard on the one part
// of this that is written down rather than asked.
//
// The named blocks are keyed by model identifier, which is the only stable
// name a block has and still a name Line 6 could change. A key that no longer
// reaches a model is a refusal that silently stopped happening, so it fails
// here instead.
//
// Each is also checked against the category rule, because a named block that
// Line 6 moved into the pitch category is a line with nothing left to do.
func (s *NotesPublicTestSuite) TestChangesTheNotesNamesModelsThisDeviceHolds() {
	cat, err := catalog.For(catalog.HXStomp)
	s.Require().NoError(err)

	for _, id := range catalog.Interval() {
		s.Run(string(id), func() {
			block, held := cat.Block(id)
			s.Require().True(held, "no longer a model this device holds")
			s.Require().NotEqual(
				catalog.CategoryPitch, block.Category,
				"the category rule already refuses this, so the line is spare")
			s.Require().True(block.ChangesTheNotes())
		})
	}
}

func TestNotesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NotesPublicTestSuite))
}
