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
package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// PlayersPublicTestSuite covers the table of what each player's records earn.
type PlayersPublicTestSuite struct {
	suite.Suite
}

// render draws a set of players and hands back what was written.
func (s *PlayersPublicTestSuite) render(
	of []audio.Player,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.Players(&buf, of))

	return buf.String()
}

// TestPlayers covers the table of what each player's records earn.
//
// One method and one table, so a case is a row rather than a file.
func (s *PlayersPublicTestSuite) TestPlayers() {
	for _, tt := range []struct {
		name string
		of   []audio.Player
		want []string
	}{
		{
			// The whole contract: the word, and the figures on both sides of
			// the comparison that produced it.
			name: "a player and what earned their word",
			of: []audio.Player{{
				ID:      "pino-palladino",
				Records: 3,
				Terms: []audio.Derived{
					{Term: "dark", Key: audio.KeyCentroid, Mine: 96, Others: 170, Of: 5},
				},
			}},
			want: []string{"pino-palladino", "dark", "96 Hz against 170 Hz"},
		},
		{
			// The other unit. A share printed as hertz reads as nonsense.
			name: "a share is printed as a share",
			of: []audio.Player{{
				ID:      "flea",
				Records: 3,
				Terms: []audio.Derived{
					{Term: "clean", Key: audio.KeyHarmonics, Mine: 0.12, Others: 0.24, Of: 5},
				},
			}},
			want: []string{"12% against 24%"},
		},
		{
			name: "a player clear on more than one axis shows both",
			of: []audio.Player{{
				ID:      "somebody",
				Records: 2,
				Terms: []audio.Derived{
					{Term: "mid-forward", Key: audio.KeyMid, Mine: 0.09, Others: 0.02, Of: 3},
					{Term: "bright", Key: audio.KeyCentroid, Mine: 400, Others: 150, Of: 3},
				},
			}},
			want: []string{"mid-forward, bright", "9% against 2%", "400 Hz against 150 Hz"},
		},
		{
			// What a margin is worth, which the margin alone cannot say. Half
			// the width of everybody else's middle half is half a step.
			name: "a margin is shown against the spread it cleared",
			of: []audio.Player{{
				ID:      "jaco-pastorius",
				Records: 4,
				Terms: []audio.Derived{{
					Term: "bright", Key: audio.KeyCentroid,
					Mine: 259, Others: 138, Of: 28, Margin: 36, Spread: 45,
				}},
			}},
			want: []string{"clear by 36 Hz (80% of their spread)"},
		},
		{
			// A spread of nothing is every other player reading the same
			// figure. There is no scale to be a share of, and printing one
			// would invent precision.
			name: "a word clear of players who all read alike shows no share",
			of: []audio.Player{{
				ID:      "alone",
				Records: 3,
				Terms: []audio.Derived{{
					Term: "bright", Key: audio.KeyCentroid,
					Mine: 400, Others: 150, Of: 3, Margin: 250,
				}},
			}},
			want: []string{"clear by 250 Hz"},
		},
		{
			// The second comparison, which answers a different question from
			// the first: not whether they are dark, but whether they are dark
			// for the music they play.
			name: "a player placed inside their own genre says so",
			of: []audio.Player{{
				ID:      "mike-dirnt",
				Records: 3,
				Within: []audio.InGenre{{
					Genre: "pop-punk",
					Of:    4,
					Terms: []audio.Derived{
						{Term: "scooped", Key: audio.KeyMid, Mine: 0.01, Others: 0.04},
					},
				}},
			}},
			want: []string{"scooped in pop-punk of 4"},
		},
		{
			// Several genres, and several words inside one of them.
			name: "every genre a player is placed in is shown",
			of: []audio.Player{{
				ID:      "dusty-hill",
				Records: 3,
				Within: []audio.InGenre{
					{Genre: "blues-rock", Of: 3, Terms: []audio.Derived{
						{Term: "dark", Key: audio.KeyCentroid, Mine: 90, Others: 150},
					}},
					{Genre: "southern-rock", Of: 6, Terms: []audio.Derived{
						{Term: "mid-forward", Key: audio.KeyMid, Mine: 0.2, Others: 0.04},
						{Term: "bright", Key: audio.KeyCentroid, Mine: 300, Others: 150},
					}},
				},
			}},
			want: []string{
				"dark in blues-rock of 3",
				"mid-forward, bright in southern-rock of 6",
			},
		},
		{
			// Mixed evidence is not a failure, and a blank cell reads as one.
			name: "a player who earned nothing says so",
			of:   []audio.Player{{ID: "les-claypool", Records: 3}},
			want: []string{"nothing"},
		},
		{
			// The count reading as the reason nothing was earned.
			name: "one player is nobody to compare against",
			of:   []audio.Player{{ID: "mike-dirnt", Records: 3}},
			want: []string{"1 player, which is nobody to compare against"},
		},
		{
			name: "several players are counted",
			of: []audio.Player{
				{ID: "flea", Records: 3},
				{ID: "mike-dirnt", Records: 3},
			},
			want: []string{"2 players"},
		},
		{
			name: "no players at all says what to do about it",
			want: []string{"no players to compare"},
		},
		{
			// Two silences that call for different work. An unremarkable player
			// is a fact about them; a scattered one is a fact about which
			// records were chosen.
			name: "a player whose own records disagree is not merely unremarkable",
			of: []audio.Player{{
				ID: "chuck-dukowski", Records: 3,
				Scattered: []audio.Scattered{{
					Key: audio.KeyCentroid, Own: 211, Between: 169, Times: 1.2,
				}},
			}},
			want: []string{
				"nothing: scattered",
				"own centroid spans 211 Hz, 1.2x the others' spread",
			},
		},
		{
			// Beside the words rather than instead of them. A player can be
			// clear on one axis and all over the place on another, and the
			// second is why a word the first predicted never arrived.
			name: "a scattered player who still earned a word shows both",
			of: []audio.Player{{
				ID: "jaco-pastorius", Records: 4,
				Terms: []audio.Derived{{
					Term: "bright", Key: audio.KeyCentroid,
					Mine: 259, Others: 138, Of: 28, Margin: 36, Spread: 45,
				}},
				Scattered: []audio.Scattered{{
					Key: audio.KeyMid, Own: 0.49, Between: 0.26, Times: 1.9,
				}},
			}},
			want: []string{
				"bright",
				"clear by 36 Hz (80% of their spread)",
				"own mid spans 49%, 1.9x the others' spread",
			},
		},
	} {
		s.Run(tt.name, func() {
			got := s.render(tt.of)

			for _, want := range tt.want {
				s.Require().Contains(got, want)
			}
		})
	}
}

func TestPlayersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlayersPublicTestSuite))
}
