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
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

type RigPublicTestSuite struct {
	suite.Suite
}

// knownWith builds a rig and the ask beside it, each carrying everything a
// person can write down, so a test about rendering them is not also a test
// about what either document must hold.
//
// Two halves of one document, because that is what a reader of a rig gets: the
// gear is the rig's and the words, the subject, the technique and the confidence
// are the ask's. A mutator for each, so a case can take either side apart
// without disturbing the other.
func knownWith(
	mutate func(*rig.Spec),
	asked func(*tone.Ask),
) sdk.Known {
	cited := []rig.Evidence{{Kind: rig.EvidenceCited}}

	spec := rig.Spec{
		Instrument: "bass",
		// Confirmed by default, so a rig that says nobody checked it is a
		// case a test has to ask for rather than get by accident.
		Chain: []rig.ChainEntry{
			{Gear: "Ampeg SVT", Role: rig.RoleAmp, Evidence: &cited},
			{Gear: "Ampeg 8x10", Role: rig.RoleCab, Evidence: &cited},
		},
	}

	band := "Green Day"
	era := "Dookie through American Idiot"
	conf := tone.ConfidenceHigh
	pos := tone.PositionBridge
	mute := tone.MutingPalm

	ask := tone.Ask{
		Subject: &tone.Subject{
			Kind: "artist",
			Name: "Mike Dirnt",
			Band: &band,
			Era:  &era,
		},
		Words: &[]tone.Word{{Term: "mid-forward"}, {Term: "gritty"}},
		Technique: &tone.Technique{
			Attack:   tone.AttackPick,
			Position: &pos,
			Muting:   &mute,
		},
		Confidence: &conf,
	}

	if mutate != nil {
		mutate(&spec)
	}

	if asked != nil {
		asked(&ask)
	}

	return sdk.Known{ID: "mike-dirnt", Rig: spec, Ask: &ask}
}

// TestRigs covers listing every rig a directory holds.
func (s *RigPublicTestSuite) TestRigs() {
	tests := []struct {
		name string
		in   sdk.Rigs
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "one row per rig",
			in: sdk.Rigs{
				Dir:  "pkg/sdk/shipped",
				Rigs: []sdk.Known{knownWith(nil, nil)},
			},
			want: []string{"mike-dirnt", "Mike Dirnt", "bass", "Ampeg SVT"},
		},
		{
			// The source column is the one that decides whether to trust the
			// row, so a rig nobody confirmed has to read differently from
			// one somebody did.
			name: "a rig nobody confirmed",
			in: sdk.Rigs{
				Dir: "pkg/sdk/shipped",
				Rigs: []sdk.Known{knownWith(func(r *rig.Spec) {
					r.Chain[0].Evidence = nil
				}, nil)},
			},
			want: []string{"mike-dirnt"},
		},
		{
			name: "a shelf with nothing on it",
			in:   sdk.Rigs{Dir: "pkg/sdk/shipped"},
			want: []string{"no rigs here"},
		},
		{
			name: "nowhere to write it",
			in: sdk.Rigs{
				Dir:  "pkg/sdk/shipped",
				Rigs: []sdk.Known{knownWith(nil, nil)},
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			name: "nowhere to write the empty case either",
			in:   sdk.Rigs{Dir: "pkg/sdk/shipped"},
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Rigs(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestRig covers printing one rig in full.
func (s *RigPublicTestSuite) TestRig() {
	tests := []struct {
		name   string
		in     sdk.Rig
		to     io.Writer
		want   []string
		absent []string
		err    bool
	}{
		{
			// A caveat is what the evidence does not show, and this project
			// cares about that more than about the claim. Only the first
			// sentence: printing them whole put five paragraphs in a nine-line
			// summary and nobody would have read any of them.
			name: "what the evidence does not show",
			in: sdk.Rig{
				Known: knownWith(func(spec *rig.Spec) {
					long := "the producer's answer, not the player's. Dirnt " +
						"names no amp for these sessions anywhere."
					spec.Chain[0].Evidence = &[]rig.Evidence{
						{Kind: rig.EvidenceCited, Caveat: &long},
					}
				}, nil),
			},
			want:   []string{"caveat", "the producer's answer, not the player's."},
			absent: []string{"names no amp for these sessions"},
		},
		{
			// A caveat with no full stop is short enough to print whole, so
			// there is nothing to trim and nothing is.
			name: "a caveat that is one clause",
			in: sdk.Rig{
				Known: knownWith(func(spec *rig.Spec) {
					short := "his live rig rather than the sessions"
					spec.Chain[0].Evidence = &[]rig.Evidence{
						{Kind: rig.EvidenceCited, Caveat: &short},
					}
				}, nil),
			},
			want: []string{"caveat", "his live rig rather than the sessions"},
		},
		{
			name: "everything a person wrote",
			in: sdk.Rig{
				Known: knownWith(nil, nil),
				Variants: []sdk.Variant{
					{ID: "mike-dirnt-longview", Name: "Longview"},
				},
			},
			want: []string{
				"Mike Dirnt", "Green Day", "Dookie through American Idiot",
				// Three stored fields, said as the one sentence a person would.
				"pick, near the bridge, palm muted",
				"mid-forward", "variants", "Longview (mike-dirnt-longview)",
				"high confidence",
			},
		},
		{
			// Nothing about what nobody wrote. An empty band line reads as a
			// band with no name rather than as a player without one.
			name: "an ask with only the required fields",
			in: sdk.Rig{Known: knownWith(nil, func(a *tone.Ask) {
				a.Subject.Band = nil
				a.Subject.Era = nil
				a.Words = nil
				a.Technique = nil
			})},
			absent: []string{"words", "variants", "band", "era", "technique"},
		},
		{
			// An unstated confidence is the lowest one. A rig that says
			// nothing about how far to trust it has not earned anything.
			name: "an unstated confidence reads as low",
			in: sdk.Rig{Known: knownWith(nil, func(a *tone.Ask) {
				a.Confidence = nil
			})},
			want: []string{"low confidence"},
		},
		{
			// A rig nobody has confirmed says so, whatever it claims about
			// itself.
			name: "gear nobody confirmed",
			in: sdk.Rig{Known: knownWith(func(r *rig.Spec) {
				r.Chain[0].Evidence = nil
			}, nil)},
			want: []string{"unverified"},
		},
		{
			// A rig somebody wrote for themselves and never wrote an ask for.
			// Legal and ordinary, so the page says what is missing rather than
			// rendering blanks: the identifier stands in for the name nobody
			// wrote, and nothing records what the rig was built for.
			name: "a document with no ask in it",
			in: sdk.Rig{
				Known: sdk.Known{ID: "mike-dirnt", Rig: knownWith(nil, nil).Rig},
			},
			want: []string{
				"mike-dirnt",
				"no ask beside it",
				"low confidence",
			},
			absent: []string{"Mike Dirnt", "words", "technique", "band", "era"},
		},
		{
			name: "nowhere to write it",
			in:   sdk.Rig{Known: knownWith(nil, nil)},
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Rig(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

// TestScaffolded covers saying what rig was written.
func (s *RigPublicTestSuite) TestScaffolded() {
	full := sdk.Scaffolded{
		ID:         "test-player",
		Name:       "Test Player",
		Instrument: "bass",
		Amp:        "Ampeg SVT",
		Cab:        "Ampeg 8x10",
		Pedals:     []string{"Klon Centaur", "Boss DS-1"},
		Path:       "pkg/sdk/shipped/artists/test-player.yaml",
	}

	tests := []struct {
		name   string
		in     sdk.Scaffolded
		to     io.Writer
		want   []string
		absent []string
		err    bool
	}{
		{
			name: "everything it was given",
			in:   full,
			want: []string{
				"Test Player", "test-player", "Ampeg SVT", "Ampeg 8x10",
				"Klon Centaur, Boss DS-1",
				// Scaffolding from gear is the path that checks every name
				// against the catalog, so it is the one that may say so.
				"every gear name resolves",
				// What to do with it next, which is the point of saying
				// anything at all.
				"presets make --id test-player",
			},
			absent: []string{"copied from"},
		},
		{
			// A copy checks no gear at all: the chain is the parent's, and
			// claiming it resolved would be claiming a check nobody ran.
			name: "a copy of another rig",
			in: sdk.Scaffolded{
				ID: "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "Ampeg SVT",
				From: "mike-dirnt", Path: "x.yaml",
			},
			want: []string{
				"chain copied from mike-dirnt unchanged",
				"presets make --id test-player",
			},
			absent: []string{"every gear name resolves"},
		},
		{
			// Some amps carry their own cabinet, and a blank row would read
			// as a cabinet nobody named.
			name: "an amp that carries its own cabinet",
			in: sdk.Scaffolded{
				ID: "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "Ampeg SVT",
				Path: "x.yaml",
			},
			absent: []string{"cab", "pedals"},
		},
		{
			name: "nowhere to say it",
			in:   full,
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Scaffolded(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

func TestRigPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RigPublicTestSuite))
}
