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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/rig"
)

type ReadPublicTestSuite struct {
	suite.Suite
}

// spec returns a rig holding the given chain, with nothing else filled in.
func spec(
	chain ...rig.ChainEntry,
) rig.Spec {
	return rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "test",
		Instrument: rig.InstrumentBass,
		Chain:      chain,
	}
}

// evidence returns the kinds as a rig carries them.
func evidence(
	kinds ...rig.EvidenceKind,
) *[]rig.Evidence {
	out := make([]rig.Evidence, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, rig.Evidence{Kind: k})
	}

	return &out
}

// TestGear looks a role up in a chain.
func (s *ReadPublicTestSuite) TestGear() {
	tests := []struct {
		name  string
		chain []rig.ChainEntry
		role  rig.Role
		want  string
		ok    bool
	}{
		{
			name: "a role the chain has",
			chain: []rig.ChainEntry{
				{Role: rig.RoleDrive, Gear: "Klon Centaur"},
				{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			},
			role: rig.RoleAmp,
			want: "Ampeg SVT",
			ok:   true,
		},
		{
			name:  "a role it does not",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp}},
			role:  rig.RoleCab,
		},
		{
			name: "an empty chain",
			role: rig.RoleAmp,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := rig.Gear(spec(tt.chain...), tt.role)

			s.Require().Equal(tt.ok, ok)
			s.Require().Equal(tt.want, got.Gear)

			// GearName is the same lookup with the miss spelled as an empty
			// name, which is what a template wants.
			s.Require().Equal(tt.want, rig.GearName(spec(tt.chain...), tt.role))
		})
	}
}

// TestTrusted says whether every claim in a rig is supported.
func (s *ReadPublicTestSuite) TestTrusted() {
	tests := []struct {
		name  string
		chain []rig.ChainEntry
		rig   []rig.EvidenceKind
		want  bool
	}{
		{
			name:  "a claim nobody supported",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp}},
		},
		{
			// `llm` means a model said so and nobody checked, which is the
			// same standing as nobody having said anything.
			name: "an assertion alone",
			chain: []rig.ChainEntry{
				{Role: rig.RoleAmp, Evidence: evidence(rig.EvidenceLLM)},
			},
		},
		{
			name: "one unsupported claim among supported ones",
			chain: []rig.ChainEntry{
				{Role: rig.RoleAmp, Evidence: evidence(rig.EvidenceCited)},
				{Role: rig.RoleCab, Evidence: evidence(rig.EvidenceLLM)},
			},
		},
		{
			name: "every claim supported",
			chain: []rig.ChainEntry{
				{Role: rig.RoleAmp, Evidence: evidence(rig.EvidenceCited)},
				{Role: rig.RoleCab, Evidence: evidence(rig.EvidenceVideo)},
			},
			want: true,
		},
		{
			// A rig rundown covers every piece of gear in it. Requiring the
			// citation on each entry would only encourage repeating it.
			name:  "evidence on the rig, covering the whole chain",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp}},
			rig:   []rig.EvidenceKind{rig.EvidenceCited},
			want:  true,
		},
		{name: "a rig with no chain at all"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			r := spec(tt.chain...)
			if tt.rig != nil {
				r.Evidence = evidence(tt.rig...)
			}

			s.Require().Equal(tt.want, rig.Trusted(r))
		})
	}
}

// TestSourced names the strongest evidence anywhere in a rig.
func (s *ReadPublicTestSuite) TestSourced() {
	tests := []struct {
		name  string
		kinds []rig.EvidenceKind
		rig   []rig.EvidenceKind
		want  rig.EvidenceKind
	}{
		{
			// Nothing in this project can hear, so somebody who played the
			// rig and heard it outranks any citation.
			name: "a person who heard it, above everything",
			kinds: []rig.EvidenceKind{
				rig.EvidenceAudio, rig.EvidenceCited, rig.EvidenceHeard,
			},
			want: rig.EvidenceHeard,
		},
		{
			// An interview says what the player used. A measurement says what
			// the record sounds like, and the record includes the studio.
			name:  "cited over audio",
			kinds: []rig.EvidenceKind{rig.EvidenceAudio, rig.EvidenceCited},
			want:  rig.EvidenceCited,
		},
		{
			// Anybody holding the record can take a measurement again. Nobody
			// can re-run a forum post.
			name:  "audio over a forum thread",
			kinds: []rig.EvidenceKind{rig.EvidenceUser, rig.EvidenceAudio},
			want:  rig.EvidenceAudio,
		},
		{
			// This test used to rank user above everything, on the reading
			// that it meant a person who listened. Every rig and every page of
			// docs used it for a forum thread, so a TalkBass comment outranked
			// an interview, and Jaco Pastorius listed as user-sourced despite
			// two citations. The person who listened is heard now.
			name:  "cited over a forum thread",
			kinds: []rig.EvidenceKind{rig.EvidenceUser, rig.EvidenceCited},
			want:  rig.EvidenceCited,
		},
		{
			name:  "a forum thread over video",
			kinds: []rig.EvidenceKind{rig.EvidenceVideo, rig.EvidenceUser},
			want:  rig.EvidenceUser,
		},
		{
			name:  "video over corpus",
			kinds: []rig.EvidenceKind{rig.EvidenceCorpus, rig.EvidenceVideo},
			want:  rig.EvidenceVideo,
		},
		{
			name:  "corpus over an assertion",
			kinds: []rig.EvidenceKind{rig.EvidenceLLM, rig.EvidenceCorpus},
			want:  rig.EvidenceCorpus,
		},
		{
			name:  "a kind nobody has ranked",
			kinds: []rig.EvidenceKind{rig.EvidenceKind("seance")},
			want:  rig.EvidenceKind("seance"),
		},
		{
			name: "an unranked kind, which does not outrank a known one",
			kinds: []rig.EvidenceKind{
				rig.EvidenceCorpus, rig.EvidenceKind("seance"),
			},
			want: rig.EvidenceCorpus,
		},
		{
			name:  "the chain, beating what the rig itself carries",
			kinds: []rig.EvidenceKind{rig.EvidenceCited},
			rig:   []rig.EvidenceKind{rig.EvidenceCorpus},
			want:  rig.EvidenceCited,
		},
		{
			// Nothing said where it came from, which is the same standing as
			// a model having asserted it.
			name: "a rig nobody supported",
			want: rig.EvidenceLLM,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			entry := rig.ChainEntry{Role: rig.RoleAmp}
			if tt.kinds != nil {
				entry.Evidence = evidence(tt.kinds...)
			}

			r := spec(entry)
			if tt.rig != nil {
				r.Evidence = evidence(tt.rig...)
			}

			s.Require().Equal(tt.want, rig.Sourced(r))
		})
	}
}

// TestCaveats covers Caveats, which is what a rig's evidence says it does not
// show.
func (s *ReadPublicTestSuite) TestCaveats() {
	caveat := func(text string) *[]rig.Evidence {
		return &[]rig.Evidence{{Kind: rig.EvidenceCited, Caveat: &text}}
	}

	for _, tt := range []struct {
		name string
		give rig.Spec
		want []string
	}{
		{
			// The rig's own first, then each entry's in signal order, because
			// that is the order a reader already has in front of them.
			name: "the rig's own, then the chain's, in signal order",
			give: func() rig.Spec {
				out := spec(
					rig.ChainEntry{Role: rig.RoleAmp, Evidence: caveat("the amp")},
					rig.ChainEntry{Role: rig.RoleCab, Evidence: caveat("the cab")},
				)
				out.Evidence = caveat("the rig")

				return out
			}(),
			want: []string{"the rig", "the amp", "the cab"},
		},
		{
			// A substitute is a claim of its own, and the reason one stands in
			// is exactly the kind of thing a caveat carries.
			name: "a substitute's own",
			give: spec(rig.ChainEntry{
				Role: rig.RoleCab,
				Substitute: &rig.Substitute{
					Gear: "Ampeg 8x10", Evidence: caveat("the nearest thing"),
				},
			}),
			want: []string{"the nearest thing"},
		},
		{
			// The same source cited for an amplifier and its cabinet carries
			// the same caveat twice, and saying it twice reads as two
			// reservations rather than one.
			name: "the same caveat twice is one reservation",
			give: spec(
				rig.ChainEntry{Role: rig.RoleAmp, Evidence: caveat("one source")},
				rig.ChainEntry{Role: rig.RoleCab, Evidence: caveat("one source")},
			),
			want: []string{"one source"},
		},
		{
			name: "evidence carrying no caveat",
			give: spec(rig.ChainEntry{
				Role: rig.RoleAmp, Evidence: evidence(rig.EvidenceCited),
			}),
		},
		{
			name: "an empty caveat is not a caveat",
			give: spec(rig.ChainEntry{Role: rig.RoleAmp, Evidence: caveat("")}),
		},
		{
			name: "a rig with no evidence at all",
			give: spec(),
		},
	} {
		s.Run(tt.name, func() {
			got := rig.Caveats(tt.give)

			if len(tt.want) == 0 {
				s.Require().Empty(got)

				return
			}

			s.Require().Equal(tt.want, got)
		})
	}
}

func TestReadPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReadPublicTestSuite))
}
