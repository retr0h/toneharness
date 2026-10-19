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

package result_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// MadePublicTestSuite covers what a build answers with.
type MadePublicTestSuite struct {
	suite.Suite
}

// TestAsked covers whether a Setup stated an output level at all.
//
// The distinction matters to a reader rather than to the arithmetic: a build
// given no target must say nothing about one, and a build that reached its
// target must not be confused with one that was never given it.
func (s *MadePublicTestSuite) TestAsked() {
	for _, tt := range []struct {
		name string
		in   result.Levelled
		want bool
	}{
		{
			// Asked for, and reached. Novoselic's `2x15 Brute` against the
			// `8x10 Ampeg SVT-E`'s reading, which is the 2.9dB that was found
			// by playing the two.
			name: "a level that was asked for",
			in:   result.Levelled{Wanted: -56.76, Was: -59.62},
			want: true,
		},
		{
			// Read but not asked for, which is still something to say.
			name: "a reading with no target",
			in:   result.Levelled{Was: -59.62},
			want: true,
		},
		{
			// A target with nothing read, which is a chain nothing has a
			// level for.
			name: "a target with nothing read",
			in:   result.Levelled{Wanted: -56.76},
			want: true,
		},
		{
			// A Setup that said nothing leaves both at zero, and a target of
			// zero would be a chain asked to come out at full scale, which
			// nobody asks for.
			name: "nothing asked and nothing read",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Asked())
		})
	}
}

// TestBeaten covers whether something wrote over what a word did.
//
// Read before Acted by anything reporting, because a word that moved a control
// and then lost it to a stated value did both and only the second is the
// outcome. The report printed the first and told somebody hand-editing a
// document that their edit had not taken.
func (s *MadePublicTestSuite) TestBeaten() {
	for _, tt := range []struct {
		name string
		in   result.Moved
		want bool
	}{
		{
			// `grit-on-attack` reached Drive and the document stated 0.88.
			name: "a word the document wrote over",
			in: result.Moved{
				Term: "grit-on-attack", Param: "Drive", From: 0.6, To: 0.72,
				Held: 0.88, Overridden: true,
			},
			want: true,
		},
		{
			// Still where the word left it, so the ordinary line stands.
			name: "a word nothing wrote over",
			in: result.Moved{
				Term: "grit-on-attack", Param: "Drive", From: 0.6, To: 0.72,
			},
		},
		{
			name: "a word that moved nothing",
			in:   result.Moved{Term: "short-decay"},
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Beaten())
		})
	}
}

// TestPinned covers whether the corpus is why a word barely moved a control.
//
// One method and one table, so a case is a row rather than a file.
func (s *MadePublicTestSuite) TestPinned() {
	for _, tt := range []struct {
		name string
		of   result.Moved
		want bool
	}{
		{
			// grunge's Mid, which is the reading this exists for: five genre
			// presets differ by 0.034 of it across all five while the words
			// behind them read `scooped` against `mid-forward`.
			name: "the corpus narrowed a whole step to well under the default",
			of:   result.Moved{Param: "Mid", Step: 0.04, Default: 0.1},
			want: true,
		},
		{
			// grunge's Treble, to the digit. Technically narrower and not
			// worth a word about, because saying so prints two numbers that
			// round to each other.
			name: "float dust is not a narrowing",
			of:   result.Moved{Param: "Treble", Step: 0.09999999999999998, Default: 0.1},
			want: false,
		},
		{
			name: "exactly half is not under half",
			of:   result.Moved{Param: "Mid", Step: 0.05, Default: 0.1},
			want: false,
		},
		{
			// The corpus had nothing to say, so the default is what moved it
			// and there is nothing to report.
			name: "a control the corpus cannot measure",
			of:   result.Moved{Param: "Mix", Step: 0.1, Default: 0.1},
			want: false,
		},
		{
			// A word that reached no control has no step to describe, whatever
			// the numbers happen to hold.
			name: "a word that moved nothing",
			of:   result.Moved{Step: 0.001, Default: 0.1},
			want: false,
		},
		{
			// Rather than dividing by zero and reporting every move.
			name: "no default at all",
			of:   result.Moved{Param: "Mid", Step: 0.04},
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.of.Pinned())
		})
	}
}

// TestMeasured covers whether a measurement sized the move rather than the
// word alone.
//
// One method and one table, so a case is a row rather than a file.
func (s *MadePublicTestSuite) TestMeasured() {
	for _, tt := range []struct {
		name string
		of   result.Moved
		want bool
	}{
		{
			// A word earned by a narrow margin is worth part of a step, and
			// the report says so, because the same word on the same chain
			// moves a different distance depending on what was measured.
			name: "part of a step",
			of:   result.Moved{Param: "Mid", Weight: 0.6},
			want: true,
		},
		{
			// A whole step is the word alone, with no measurement sizing it.
			name: "a whole step",
			of:   result.Moved{Param: "Mid", Weight: 1},
			want: false,
		},
		{
			name: "no weight at all",
			of:   result.Moved{Param: "Mid"},
			want: false,
		},
		{
			// A weight on a word that reached no control describes nothing.
			name: "a word that moved nothing",
			of:   result.Moved{Weight: 0.6},
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.of.Measured())
		})
	}
}

// TestContested covers whether another term spoke for the same axis.
//
// One method and one table, so a case is a row rather than a file.
func (s *MadePublicTestSuite) TestContested() {
	for _, tt := range []struct {
		name string
		of   result.Moved
		want bool
	}{
		{
			// Two words from one axis are two answers to one question, so
			// neither is applied: applying both lands back where it started
			// and reads as though the rig said nothing.
			name: "another word spoke for the axis",
			of:   result.Moved{Term: "scooped", Against: "mid-forward"},
			want: true,
		},
		{
			name: "nothing contradicted it",
			of:   result.Moved{Term: "scooped", Param: "Mid"},
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.of.Contested())
		})
	}
}

// TestYielded covers whether this word stood aside for one somebody wrote.
//
// One method and one table, so a case is a row rather than a file.
func (s *MadePublicTestSuite) TestYielded() {
	for _, tt := range []struct {
		name string
		of   result.Moved
		want bool
	}{
		{
			// A different answer from Contested, and reported as one: that is
			// a contradiction nothing can resolve, this is a word a genre
			// earned meeting a word a person wrote, where the person's
			// stands. They said it and a population did not.
			name: "a person's word won the axis",
			of:   result.Moved{Term: "scooped", YieldedTo: "mid-forward"},
			want: true,
		},
		{
			name: "nothing stood in its way",
			of:   result.Moved{Term: "scooped", Param: "Mid"},
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.of.Yielded())
		})
	}
}

// TestUnanswered covers whether the chain, rather than this project, is why
// the word moved nothing.
//
// One method and one table, so a case is a row rather than a file.
func (s *MadePublicTestSuite) TestUnanswered() {
	for _, tt := range []struct {
		name string
		of   result.Moved
		want bool
	}{
		{
			// The chain cannot hear the word. A different answer from an
			// empty Because, which says nobody has taught the project to
			// listen for it at all.
			name: "the chain is the reason",
			of:   result.Moved{Term: "static", Because: "this chain has no filter"},
			want: true,
		},
		{
			name: "the word reached a control",
			of:   result.Moved{Term: "scooped", Param: "Mid"},
			want: false,
		},
		{
			name: "nothing acts on the word at all",
			of:   result.Moved{Term: "squelchy"},
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.of.Unanswered())
		})
	}
}

// TestHolds covers whether the chain already answers the word as built.
//
// One method and one table, so a case is a row rather than a file.
func (s *MadePublicTestSuite) TestHolds() {
	for _, tt := range []struct {
		name string
		of   result.Moved
		want bool
	}{
		{
			// A rig asking for no room, in a chain holding no reverb, asked
			// for something it already has.
			name: "the chain answers it without a knob turning",
			of:   result.Moved{Term: "dry", Already: "this chain has no reverb"},
			want: true,
		},
		{
			name: "a knob had to move",
			of:   result.Moved{Term: "scooped", Param: "Mid"},
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.of.Holds())
		})
	}
}

func TestMadePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MadePublicTestSuite))
}
