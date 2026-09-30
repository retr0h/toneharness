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

package rig_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

// good returns the smallest rig the schema accepts.
func (s *ValidatePublicTestSuite) good() rig.Spec {
	return rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "mike-dirnt",
		Instrument: rig.InstrumentBass,
		Chain:      []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "Ampeg SVT"}},
	}
}

// TestValidate checks a rig against its own contract. A case naming no field
// is one the contract accepts.
func (s *ValidatePublicTestSuite) TestValidate() {
	knob := func(v float64) *rig.Knob { return &v }
	settings := func(v float64) *rig.Settings { return &rig.Settings{Drive: knob(v)} }
	confidence := func(c rig.Confidence) *rig.Confidence { return &c }

	tests := []struct {
		name   string
		mutate func(*rig.Spec)
		field  string
		// a failure that is not the contract refusing a field.
		says string
	}{
		{
			name:   "the smallest useful rig",
			mutate: func(*rig.Spec) {},
		},
		{
			// Absent is not invalid. A hand-written rig carries almost none
			// of this.
			name: "everything optional, filled in",
			mutate: func(r *rig.Spec) {
				r.Chain[0].Settings = settings(0)
				r.Chain[0].Settings.Treble = knob(1)
				r.Chain[0].Evidence = &[]rig.Evidence{{Kind: rig.EvidenceCited}}
			},
		},
		{
			// A rig is checked through JSON, and a value that has no JSON
			// cannot be checked against anything.
			name:   "a knob holding a value that is not a number",
			mutate: func(r *rig.Spec) { r.Chain[0].Settings = settings(math.NaN()) },
			says:   "reading the rig",
		},
		{
			name:   "a document that is not a rig",
			mutate: func(r *rig.Spec) { r.Schema = "L6Preset" },
			field:  "schema",
		},
		{
			name:   "an identifier with spaces",
			mutate: func(r *rig.Spec) { r.ID = "Mike Dirnt" },
			field:  "id",
		},
		{
			name:   "an identifier that is empty",
			mutate: func(r *rig.Spec) { r.ID = "" },
			field:  "id",
		},
		{
			name:   "an instrument the catalog cannot be filtered by",
			mutate: func(r *rig.Spec) { r.Instrument = "theremin" },
			field:  "instrument",
		},
		{
			name:   "a chain holding nothing",
			mutate: func(r *rig.Spec) { r.Chain = nil },
			field:  "chain",
		},
		{
			name:   "gear doing nothing in particular",
			mutate: func(r *rig.Spec) { r.Chain[0].Role = "vibe" },
			field:  "chain[0].role",
		},
		{
			name:   "gear with no name",
			mutate: func(r *rig.Spec) { r.Chain[0].Gear = "" },
			field:  "chain[0].gear",
		},
		{
			name:   "a setting above one",
			mutate: func(r *rig.Spec) { r.Chain[0].Settings = settings(1.5) },
			field:  "chain[0].settings.drive",
		},
		{
			name:   "a setting below nought",
			mutate: func(r *rig.Spec) { r.Chain[0].Settings = settings(-0.1) },
			field:  "chain[0].settings.drive",
		},
		{
			name:   "a confidence nobody can act on",
			mutate: func(r *rig.Spec) { r.Chain[0].Confidence = confidence("certain") },
			field:  "chain[0].confidence",
		},
		{
			name: "evidence of no known kind",
			mutate: func(r *rig.Spec) {
				r.Chain[0].Evidence = &[]rig.Evidence{{Kind: "vibes"}}
			},
			field: "chain[0].evidence[0].kind",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := s.good()
			tt.mutate(&got)

			err := rig.Validate(got)

			if tt.says != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.says)

				return
			}

			if tt.field == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, rig.ErrInvalid)
			s.Require().Contains(err.Error(), tt.field)
		})
	}
}

func TestValidatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidatePublicTestSuite))
}
