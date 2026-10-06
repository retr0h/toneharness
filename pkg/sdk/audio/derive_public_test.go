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

package audio_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// DerivePublicTestSuite covers turning a measurement into a word.
type DerivePublicTestSuite struct {
	suite.Suite
}

// terms is what came back, as plain words.
func (s *DerivePublicTestSuite) terms(
	got []audio.Derived,
) []string {
	out := make([]string, 0, len(got))
	for _, d := range got {
		out = append(out, d.Term)
	}

	return out
}

// band is one measure's range around a middle.
func band(
	mid, width float64,
) audio.Spread {
	return audio.Spread{Low: mid - width, Mid: mid, High: mid + width}
}

// at builds an artist sitting tightly at one place on every measure.
//
// Each measure gets a width suited to its own scale. A centroid is hundreds of
// hertz and a band share is between zero and one, so one width for both makes
// the shares span from below nothing to above everything and overlap every
// other artist, which is what the first version of this helper did.
func at(
	centroid, mid, harmonics float64,
) audio.Across {
	return audio.Across{
		Tracks:    3,
		Mid:       band(mid, 0.01),
		Centroid:  band(centroid, 5),
		Harmonics: band(harmonics, 0.01),
	}
}

// wide builds an artist whose ranges are broad enough to overlap a neighbour.
func wide(
	centroid, mid, harmonics float64,
) audio.Across {
	return audio.Across{
		Tracks:    3,
		Mid:       band(mid, 0.10),
		Centroid:  band(centroid, 60),
		Harmonics: band(harmonics, 0.20),
	}
}

// TestDerive covers which words a set of figures earns.
//
// One method and one table, so a case is a row rather than a file.
func (s *DerivePublicTestSuite) TestDerive() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Derive, which is what an artist's measurements say about
			// them, against others measured the same way.
			//
			// One method and one table, so a case is a row rather than a file.
			name: "derive",
			then: func() {
				for _, tt := range []struct {
					name string
					then func()
				}{
					{
						// The whole mechanism.
						name: "an artist clear of the rest earns the term",
						then: func() {
							got := audio.Derive(
								at(300, 0.30, 0.50),
								map[string]audio.Across{
									"a": at(100, 0.05, 0.10),
									"b": at(120, 0.06, 0.12),
								},
							)

							s.Require().ElementsMatch(
								[]string{"bright", "mid-forward", "saturated"}, s.terms(got))
						},
					},
					{
						// The other direction.
						name: "an artist below the rest earns the other word",
						then: func() {
							got := audio.Derive(
								at(100, 0.02, 0.05),
								map[string]audio.Across{
									"a": at(300, 0.30, 0.50),
									"b": at(320, 0.32, 0.55),
								},
							)

							s.Require().ElementsMatch(
								[]string{"dark", "scooped", "clean"}, s.terms(got))
						},
					},
					{
						// The rule that stops a median lying.
						//
						// Measured on real records, one of the three artists reads 0% mid
						// on two and 15% on a third. His median is 0%, a figure he never
						// plays, and it sits below another artist's 9%. Deriving
						// `scooped` from that median would assert something his own
						// records contradict a third of the time, so a term needs the
						// whole range clear rather than the middle.
						name: "a mixed artist earns nothing",
						then: func() {
							mixed := audio.Across{
								Tracks: 3,
								Mid:    audio.Spread{Low: 0.00, Mid: 0.00, High: 0.15},
							}

							got := audio.Derive(mixed, map[string]audio.Across{
								"dirnt": {Tracks: 3, Mid: audio.Spread{Low: 0.04, Mid: 0.09, High: 0.15}},
								"flea":  {Tracks: 3, Mid: audio.Spread{Low: 0.02, Mid: 0.03, High: 0.06}},
							})

							s.Require().Empty(s.terms(got), "the range straddles the others")
						},
					},
					{
						// Ranges that merely touch.
						name: "overlap earns nothing",
						then: func() {
							got := audio.Derive(
								wide(200, 0.10, 0.20),
								map[string]audio.Across{
									"a": wide(180, 0.09, 0.18),
									"b": wide(220, 0.11, 0.22),
								},
							)

							s.Require().Empty(s.terms(got))
						},
					},
					{
						// A term carrying its own reason.
						//
						// A rig records where every other claim came from, and a word
						// derived from a number is no different.
						name: "it says what earned it",
						then: func() {
							got := audio.Derive(
								at(300, 0.30, 0.50),
								map[string]audio.Across{"a": at(100, 0.05, 0.10)},
							)

							s.Require().NotEmpty(got)

							for _, d := range got {
								s.Require().NotEmpty(d.Key, "which figure")
								s.Require().NotEmpty(d.Why, "and what that figure is")
								s.Require().Equal(2, d.Of, "against how many artists")

								// A term only exists because the two differ, so a report that
								// printed them as the same figure would be reporting nothing.
								s.Require().NotEqual(d.Mine, d.Others)
							}
						},
					},
					{
						// A population of one.
						name: "nobody to compare against derives nothing",
						then: func() {
							s.Require().Empty(audio.Derive(at(300, 0.3, 0.5), nil))
							s.Require().Empty(audio.Derive(audio.Across{}, map[string]audio.Across{
								"a": at(100, 0.05, 0.10),
							}))
						},
					},
					{
						// An artist with no recordings in the population.
						name: "an empty other is not a vote",
						then: func() {
							got := audio.Derive(at(300, 0.30, 0.50), map[string]audio.Across{
								"a":     at(100, 0.05, 0.10),
								"empty": {},
							})

							s.Require().NotEmpty(got)

							for _, d := range got {
								s.Require().Equal(3, d.Of)
							}
						},
					},
					{
						// The property the rule was changed for.
						//
						// Clear of every other artist, one more extreme artist was enough
						// to take a term away from somebody who had plainly earned it.
						// Measured, one player earned three terms against two artists and
						// none against four. Outside the middle half of the others, the
						// extreme artist moves the upper quartile a little and takes
						// nothing away.
						name: "an extreme artist does not take a term away",
						then: func() {
							mine := at(200, 0.10, 0.20)

							three := map[string]audio.Across{
								"a": at(100, 0.10, 0.20),
								"b": at(110, 0.10, 0.20),
								"c": at(120, 0.10, 0.20),
							}
							s.Require().Equal([]string{"bright"}, s.terms(audio.Derive(mine, three)))

							four := map[string]audio.Across{
								"a": at(100, 0.10, 0.20),
								"b": at(110, 0.10, 0.20),
								"c": at(120, 0.10, 0.20),
								"d": at(400, 0.10, 0.20),
							}
							s.Require().Equal([]string{"bright"}, s.terms(audio.Derive(mine, four)),
								"brighter than most is still bright when somebody brighter arrives")
						},
					},
					{
						// A population that is there and has nothing in it.
						//
						// Different from no population at all: the map has artists, and
						// not one of them has a recording. Nobody to sit clear of, so
						// nothing is earned.
						name: "every other artist empty derives nothing",
						then: func() {
							got := audio.Derive(at(300, 0.30, 0.50), map[string]audio.Across{
								"a": {},
								"b": {},
							})

							s.Require().Empty(s.terms(got))
						},
					},
				} {
					s.Run(tt.name, func() {
						tt.then()
					})
				}
			},
		},
		{
			// A deliberate absence.
			//
			// Sag and reverb Mix have no measurement at all, and the compressor's Attack
			// has one that is not trusted: the artist whose rig says `percussive` measured
			// the lowest transient of three. Nothing derives from any of them.
			name: "no axis derives from a measure nobody trusts",
			then: func() {
				for _, ax := range audio.Axes {
					s.Require().NotEqual(audio.KeyTransient, ax.Key)
					s.Require().NotEqual(audio.KeyDecay, ax.Key)
				}

				for _, term := range []string{
					"percussive", "soft-attack", "audible-pick-attack",
					"tight-low-end", "loose-low-end", "dry", "roomy",
				} {
					for _, ax := range audio.Axes {
						s.Require().NotEqual(term, ax.More)
						s.Require().NotEqual(term, ax.Less)
					}
				}
			},
		},
		{
			// The words being spellable.
			//
			// A term the compiler's table does not hold moves nothing, and would arrive as
			// a word that silently does nothing rather than as an error.
			name: "every derived term is one the compiler knows",
			then: func() {
				known := map[string]bool{
					"mid-forward": true, "scooped": true,
					"dark": true, "bright": true, "glassy": true,
					"clean": true, "minimal-drive": true,
					"grit-on-attack": true, "saturated": true,
				}

				for _, ax := range audio.Axes {
					s.Require().True(known[ax.More], "%s is not a term move.go holds", ax.More)
					s.Require().True(known[ax.Less], "%s is not a term move.go holds", ax.Less)
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestDerivePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DerivePublicTestSuite))
}
