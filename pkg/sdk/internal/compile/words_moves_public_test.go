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
	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

// WordsMovePublicTestSuite covers the words an ask uses reaching the
// amplifier, which is the whole reason the vocabulary exists.
type WordsMovePublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *WordsMovePublicTestSuite) SetupTest() {
	s.cat = loadCatalog(&s.Suite)
}

// described is the ask beside a rig, saying how it should sound.
//
// Beside rather than on it: the rig from rig is unchanged whatever words go
// with it, which is the split these tests are checking.
func described(
	terms ...string,
) compile.Intent {
	words := make([]compile.Word, 0, len(terms))
	for _, t := range terms {
		words = append(words, compile.Word{Term: t})
	}

	return compile.Intent{Words: words}
}

// TestResolve covers a word reaching a control that is already in the chain.
//
// One method and one table, so a case is a row rather than a file.
func (s *WordsMovePublicTestSuite) TestResolve() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A word landing on the amplifier's own
			// controls, and a chain with no amplifier to land on.
			//
			// One method and one table, so a case is a row rather than a file.
			name: "words reach the amplifier",
			then: func() {
				for _, tt := range []struct {
					name string
					then func()
				}{
					{
						// A word turning a real knob.
						//
						// Drive, because it is the one control every amplifier in this
						// fixture carries. Which parameter each axis moves is covered
						// against a block built for it in move_test.go; what this proves
						// is that a rig's words reach the amplifier at all.
						name: "words reach the amplifier",
						then: func() {
							tests := []struct {
								name string
								term string
								up   bool
							}{
								{name: "pushed harder", term: "saturated", up: true},
								{name: "backed off", term: "clean"},
							}

							for _, tt := range tests {
								s.Run(tt.name, func() {
									plain, _, _, _, err := compile.Resolve(
										"a-rig", bassRig("Ampeg SVT", ""),
										compile.Intent{},
										s.cat,
										nil,
									)
									s.Require().NoError(err)

									got, _, moved, _, err := compile.Resolve(
										"a-rig", bassRig("Ampeg SVT", ""), described(tt.term), s.cat, nil)
									s.Require().NoError(err)

									s.Require().Len(moved, 1)
									s.Require().True(moved[0].Acted())
									s.Require().Equal("Drive", moved[0].Param)

									was := s.paramOf(plain, "Drive")
									now := s.paramOf(got, "Drive")

									if tt.up {
										s.Require().Greater(now, was)

										return
									}

									s.Require().Less(now, was)
								})
							}
						},
					},
					{
						// An ask describing a sound the rig has nowhere to make.
						//
						// The words are still what somebody asked for, so they are
						// reported as moving nothing rather than dropped.
						name: "words survive a chain with no amplifier",
						then: func() {
							spec := rig.Spec{
								Instrument: rig.InstrumentBass,
								Chain: []rig.ChainEntry{
									{Role: rig.RoleOther, Gear: "Klon Centaur"},
								},
							}

							_, _, moved, _, err := compile.Resolve("a-rig", spec, described("mid-forward"), s.cat, nil)

							s.Require().NoError(err)
							s.Require().Len(moved, 1)
							s.Require().False(moved[0].Acted())
						},
					},
				} {
					s.Run(tt.name, func() {
						// A row gets the same fresh state a method used to get.
						s.SetupTest()

						tt.then()
					})
				}
			},
		},
		{
			// The silent case.
			//
			// The zero Intent, which is also what a rig read off disk is built with: it
			// carries settings somebody already applied, so there is nothing for a word to
			// decide.
			name: "an ask that says nothing moves nothing",
			then: func() {
				_, _, moved, _, err := compile.Resolve(
					"a-rig",
					bassRig("Ampeg SVT", ""),
					compile.Intent{},
					s.cat,
					nil,
				)

				s.Require().NoError(err)
				s.Require().Empty(moved)
			},
		},
		{
			// An ask answering one question
			// twice.
			name: "two words for one axis move nothing",
			then: func() {
				plain, _, _, _, err := compile.Resolve(
					"a-rig",
					bassRig("Ampeg SVT", ""),
					compile.Intent{},
					s.cat,
					nil,
				)
				s.Require().NoError(err)

				got, _, moved, _, err := compile.Resolve(
					"a-rig", bassRig("Ampeg SVT", ""),
					described("minimal-drive", "grit-on-attack"), s.cat, nil)
				s.Require().NoError(err)

				s.Require().Len(moved, 2)

				for _, m := range moved {
					s.Require().True(m.Contested())
					s.Require().Equal("drive", m.Against)
				}

				s.Require().InDelta(
					s.paramOf(plain, "Drive"), s.paramOf(got, "Drive"), 1e-9)
			},
		},
		{
			// Which of two answers to one
			// question stands, by who said it.
			//
			// One method and one table, so a case is a row rather than a file.
			name: "who yields on an axis two words answer",
			then: func() {
				for _, tt := range []struct {
					name string
					// written is what somebody put on the ask, derived what a genre earned
					// by measuring. Both land in the same Words, which is the arrangement
					// the precedence has to be read out of.
					written []string
					derived []string
					// acted is every term that moved a control, and yielded every term
					// that stood aside for one somebody wrote. Everything named in neither
					// is contested, which is the third outcome.
					acted     []string
					yielded   map[string]string
					contested []string
				}{
					{
						// The case this method exists for. `clean` is what punk measures as
						// and `grit-on-attack` is what the person asked for; both answer
						// drive. Counting them together cancelled the person's own word
						// against a measurement of records they never mentioned.
						name:    "a word somebody wrote beats one a genre earned",
						written: []string{"grit-on-attack"},
						derived: []string{"clean"},
						acted:   []string{"grit-on-attack"},
						yielded: map[string]string{"clean": "grit-on-attack"},
					},
					{
						// Unchanged, and the reason the rule is about standing rather than
						// about order: this cannot know which half they meant.
						name:      "two words somebody wrote still contradict",
						written:   []string{"minimal-drive", "grit-on-attack"},
						contested: []string{"minimal-drive", "grit-on-attack"},
					},
					{
						// Two genres disagreeing is still a contradiction, and still
						// nothing this can resolve, so nothing written means nothing wins.
						name:      "two genres disagreeing still contradict",
						derived:   []string{"clean", "saturated"},
						contested: []string{"clean", "saturated"},
					},
					{
						// A derived word on an axis nobody else answered is an ordinary
						// word. Yielding is about being outranked, not about provenance.
						name:    "a genre's word on an axis nobody contested",
						derived: []string{"grit-on-attack"},
						acted:   []string{"grit-on-attack"},
					},
					{
						// A word outside the vocabulary answers no axis, so it outranks
						// nothing. `chunky` must not silence what a genre measured: it
						// reaches no control, and a measurement standing aside for it would
						// leave the axis answered by neither.
						name:    "a word nothing defines does not outrank a measurement",
						written: []string{"chunky"},
						derived: []string{"clean"},
						acted:   []string{"clean"},
					},
					{
						// The mirror of it. A derived term nothing defines answers no axis
						// either, so it has nothing to stand aside from.
						name:    "a measurement of a word nothing defines",
						written: []string{"grit-on-attack"},
						derived: []string{"chunky"},
						acted:   []string{"grit-on-attack"},
					},
				} {
					s.Run(tt.name, func() {
						words := make([]compile.Word, 0, len(tt.written)+len(tt.derived))
						for _, t := range tt.written {
							words = append(words, compile.Word{Term: t})
						}

						for _, t := range tt.derived {
							words = append(words, compile.Word{Term: t, Derived: true})
						}

						_, _, moved, _, err := compile.Resolve(
							"a-rig", bassRig("Ampeg SVT", ""), compile.Intent{Words: words}, s.cat, nil)
						s.Require().NoError(err)

						by := map[string]compile.Moved{}
						for _, m := range moved {
							by[m.Term] = m
						}

						for _, term := range tt.acted {
							s.Require().True(by[term].Acted(), "%s moved nothing", term)
						}

						for term, to := range tt.yielded {
							s.Require().True(by[term].Yielded(), "%s did not yield", term)
							s.Require().Equal(to, by[term].YieldedTo)
						}

						for _, term := range tt.contested {
							s.Require().True(by[term].Contested(), "%s was not contested", term)
							s.Require().False(by[term].Yielded(),
								"a contradiction is not a word standing aside")
						}
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			tt.then()
		})
	}
}

// paramOf reads one control off whichever block is the amplifier.
func (s *WordsMovePublicTestSuite) paramOf(
	built plan.Plan,
	key string,
) float64 {
	for _, b := range built.Blocks {
		blk, ok := s.cat.Block(b.Model)
		if !ok || blk.Category != catalog.CategoryAmp {
			continue
		}

		v, ok := b.Params[key].Float()
		s.Require().True(ok, "the amplifier has no %s", key)

		return v
	}

	s.Require().Fail("no amplifier in the chain")

	return 0
}

func TestWordsMovePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WordsMovePublicTestSuite))
}
