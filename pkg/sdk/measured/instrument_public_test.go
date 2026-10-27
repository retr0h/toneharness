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

package measured_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// InstrumentPublicTestSuite covers which instrument a reading is about, and
// whether it is about the chain asking.
//
// Task 290. Both reading types have carried the field since 2026-09-29 and
// nothing on any path that builds a preset read it, while all 215 committed
// readings were taken with a bass. That cost nothing only because no guitar
// record is measured, so a guitar ask reaches no target and consults no
// reading. It is the kind of wrong that converges and reports its tolerances
// met.
type InstrumentPublicTestSuite struct {
	suite.Suite
}

// TestDescribes covers whether a reading taken on one instrument describes a
// chain built for another.
//
// One method and one table, so a case is a row rather than a file.
func (s *InstrumentPublicTestSuite) TestDescribes() {
	for _, tt := range []struct {
		name string
		// got is what the reading was taken on, wants what the chain is for.
		got, wants string
		// is names the sentinel expected, or nil where the reading answers.
		is error
	}{
		{
			name:  "a bass reading for a bass chain",
			got:   "bass",
			wants: "bass",
		},
		{
			// The catalog and the corpus spell it in lower case and a
			// hand-written document need not.
			name:  "the spelling is not the question",
			got:   "bass",
			wants: "Bass",
		},
		{
			// The one this exists for. A bass guitar's centre of gravity sits
			// an octave below a guitar's, so the nearest block to the wrong
			// octave is not a near miss.
			name:  "a bass reading for a guitar chain",
			got:   "bass",
			wants: "guitar",
			is:    measured.ErrOtherInstrument,
		},
		{
			name:  "a guitar reading for a bass chain",
			got:   "guitar",
			wants: "bass",
			is:    measured.ErrOtherInstrument,
		},
		{
			// Refused rather than allowed. "Probably bass, everything here
			// is" is the reasoning the guard exists to stop.
			name:  "a reading that cannot say",
			wants: "bass",
			is:    measured.ErrNoInstrument,
		},
		{
			// A caller with no chain in hand has to take a reading as
			// measured: `measure slopes` reports across a tree rather than
			// building for anybody.
			name: "a caller with no chain takes it as measured",
			got:  "bass",
		},
		{
			name: "and takes a reading that cannot say, too",
		},
	} {
		s.Run(tt.name, func() {
			err := measured.Describes(tt.got, tt.wants)

			if tt.is == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.is)
			s.Require().Contains(err.Error(), tt.wants,
				"the refusal names what the chain is for")
		})
	}
}

// TestInstrumentIn covers what a reference recording's filename says was
// played through it.
//
// One method and one table, so a case is a row rather than a file.
func (s *InstrumentPublicTestSuite) TestInstrumentIn() {
	for _, tt := range []struct {
		at   string
		want string
	}{
		{at: filepath.Join("resources", "dry", "bass-di-short.wav"), want: "bass"},
		{at: filepath.Join("resources", "dry", "guitar-di.wav"), want: "guitar"},
		{at: "BASS-DI.WAV", want: "bass"},
		// The honest empty. A reading that does not know what was played
		// through it should not say, because the point of recording it is that
		// somebody can tell and a guess defeats that.
		{at: filepath.Join("somewhere", "else", "take-3.wav")},
		{at: "reference.wav"},
	} {
		s.Run(tt.at, func() {
			s.Require().Equal(tt.want, measured.InstrumentIn(tt.at))
		})
	}
}

// TestLibraryPlayedOn covers which instrument a library of fingerprints is
// about.
//
// One method and one table, so a case is a row rather than a file.
func (s *InstrumentPublicTestSuite) TestLibraryPlayedOn() {
	for _, tt := range []struct {
		name string
		lib  measured.Library
		want string
	}{
		{
			name: "the field, where it says",
			lib:  measured.Library{Instrument: "guitar"},
			want: "guitar",
		},
		{
			// The field wins over the filename, which is what makes it the
			// stronger evidence: a campaign may name its reference anything.
			name: "the field, over a reference that says otherwise",
			lib: measured.Library{
				Instrument: "guitar",
				Reference:  measured.Reference{File: "resources/dry/bass-di.wav"},
			},
			want: "guitar",
		},
		{
			// 8 of the 215 committed readings predate the field and every one
			// of them names this file. Refusing them put the catalog's
			// defaults back on an EQ that had been set from its own sweep.
			name: "the reference, where the field does not",
			lib: measured.Library{
				Reference: measured.Reference{File: "resources/dry/bass-di.wav"},
			},
			want: "bass",
		},
		{
			name: "and nothing, where neither says",
			lib:  measured.Library{Reference: measured.Reference{File: "take-3.wav"}},
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.lib.PlayedOn())
		})
	}
}

// TestCurvesPlayedOn covers the same question of one block's sweep.
//
// One method and one table, so a case is a row rather than a file.
func (s *InstrumentPublicTestSuite) TestCurvesPlayedOn() {
	for _, tt := range []struct {
		name   string
		curves measured.Curves
		want   string
	}{
		{
			name:   "the field, where it says",
			curves: measured.Curves{Instrument: "bass"},
			want:   "bass",
		},
		{
			name: "the reference, where the field does not",
			curves: measured.Curves{
				Reference: measured.Reference{File: "resources/dry/guitar-di.wav"},
			},
			want: "guitar",
		},
		{
			name:   "and nothing, where neither says",
			curves: measured.Curves{},
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.curves.PlayedOn())
		})
	}
}

func TestInstrumentPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InstrumentPublicTestSuite))
}
