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

package translate_test

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// StompPublicTestSuite covers what a rig gets on its footswitches.
//
// Against the shipped corpus where a rig is involved, because the thing being
// checked is that the 43 here stop carrying the same two kinds, and against
// built tables where a figure or a kind is, because those cover more rows than
// today's data reaches.
type StompPublicTestSuite struct {
	suite.Suite

	deps translate.Deps
}

func (s *StompPublicTestSuite) SetupSuite() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	lib, err := measured.BuiltIn()
	s.Require().NoError(err)

	s.deps = translate.Deps{Catalog: cat, Measured: lib}
}

// TestAnswers covers which kind of block answers one figure.
//
// One method and one table, so a case is a row rather than a file.
func (s *StompPublicTestSuite) TestAnswers() {
	for _, tt := range []struct {
		name string
		key  audio.Figure
		more bool
		want catalog.Category
		ok   bool
	}{
		{
			name: "brighter or darker is an equaliser",
			key:  audio.KeyCentroid, more: true,
			want: catalog.CategoryEQ, ok: true,
		},
		{
			name: "and either way round",
			key:  audio.KeyCentroid,
			want: catalog.CategoryEQ, ok: true,
		},
		{
			// The same control of the same thing, so a target earning both
			// gets one equaliser rather than two.
			name: "the midrange is an equaliser too",
			key:  audio.KeyMid, more: true,
			want: catalog.CategoryEQ, ok: true,
		},
		{
			name: "and the outer bands, which a bass is not weighed on",
			key:  audio.KeyLow,
			want: catalog.CategoryEQ, ok: true,
		},
		{
			name: "and the high band",
			key:  audio.KeyHigh, more: true,
			want: catalog.CategoryEQ, ok: true,
		},
		{
			// Energy above the fundamental is what distortion makes, and
			// `harmonics` measures exactly that share.
			name: "more harmonics is a drive",
			key:  audio.KeyHarmonics, more: true,
			want: catalog.CategoryDrive, ok: true,
		},
		{
			// Nothing on a floor takes harmonics out of a signal.
			name: "fewer is nobody's pedal",
			key:  audio.KeyHarmonics,
		},
		{
			// What the figure measures is a tail, and a tail is a reverb.
			name: "a longer decay is a reverb",
			key:  audio.KeyDecay, more: true,
			want: catalog.CategoryReverb, ok: true,
		},
		{
			// 14 of the 43 are distinctive only on this, and a gate chopping
			// notes is not what any of them wants.
			name: "a shorter one has no answer",
			key:  audio.KeyDecay,
		},
		{
			name: "a figure nothing on a board answers",
			key:  audio.KeyDynamics, more: true,
		},
		{
			name: "and one no word is earned on",
			key:  audio.KeyLevel, more: true,
		},
	} {
		s.Run(tt.name, func() {
			got, ok := translate.Answers(tt.key, tt.more)

			s.Require().Equal(tt.ok, ok)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestAskedFor covers the kinds a target's records ask for, in switch order.
//
// One method and one table, so a case is a row rather than a file.
func (s *StompPublicTestSuite) TestAskedFor() {
	at := func(term string, key audio.Figure, mine, others, margin float64) audio.Derived {
		return audio.Derived{
			Term: term, Key: key, Mine: mine, Others: others, Margin: margin,
		}
	}

	for _, tt := range []struct {
		name     string
		terms    []audio.Derived
		held     map[catalog.Category]bool
		switches int
		want     []catalog.Category
		// term is the word the first kind must be attributed to, where the
		// case is about which of two earns it.
		term string
	}{
		{
			name: "one word, one kind",
			terms: []audio.Derived{
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.07),
			},
			switches: 3,
			want:     []catalog.Category{catalog.CategoryReverb},
		},
		{
			// By how far past the others the records sit, as a share of what
			// the others measure. A rig with more words than switches keeps
			// the ones it is furthest out on.
			name: "the furthest out comes first",
			terms: []audio.Derived{
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.07),
				at("saturated", audio.KeyHarmonics, 0.4, 0.3, 0.9),
			},
			switches: 3,
			want: []catalog.Category{
				catalog.CategoryDrive, catalog.CategoryReverb,
			},
		},
		{
			// A margin is in its figure's own units, so the raw numbers rank
			// by unit rather than by distinctiveness: 45 hertz of centroid
			// against 0.09 of a share is not 500 times as distinctive. As a
			// share of the others' median the second is further out, 0.9
			// against 0.35, and it is what gets the one switch.
			name: "the units do not decide it",
			terms: []audio.Derived{
				at("bright", audio.KeyCentroid, 175, 130, 45),
				at("saturated", audio.KeyHarmonics, 0.19, 0.1, 0.09),
			},
			switches: 1,
			want:     []catalog.Category{catalog.CategoryDrive},
		},
		{
			// Nothing to be outside of, so there is nothing to compare: a
			// figure whose others read zero earns no word in the first place
			// and this is the contract rather than a state the corpus is in.
			name: "others measuring nothing",
			terms: []audio.Derived{
				at("saturated", audio.KeyHarmonics, 0.4, 0, 0.4),
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.07),
			},
			switches: 3,
			want: []catalog.Category{
				catalog.CategoryReverb, catalog.CategoryDrive,
			},
		},
		{
			// The centroid and the midrange are the same control of the same
			// thing, so the second earns nothing.
			name: "two words on one kind give one block",
			terms: []audio.Derived{
				at("bright", audio.KeyCentroid, 180, 130, 45),
				at("mid-forward", audio.KeyMid, 0.2, 0.1, 0.09),
			},
			switches: 3,
			want:     []catalog.Category{catalog.CategoryEQ},
		},
		{
			// A researched rig may already name the pedal its player is
			// documented with, and two of one kind would light the same.
			name: "a kind the chain already holds is not doubled",
			terms: []audio.Derived{
				at("saturated", audio.KeyHarmonics, 0.4, 0.3, 0.9),
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.07),
			},
			held:     map[catalog.Category]bool{catalog.CategoryDrive: true},
			switches: 3,
			want:     []catalog.Category{catalog.CategoryReverb},
		},
		{
			// A block past the last switch is one no foot can reach, and the
			// foot is the whole reason for it.
			name: "more words than switches stops at the switches",
			terms: []audio.Derived{
				at("bright", audio.KeyCentroid, 180, 130, 45),
				at("saturated", audio.KeyHarmonics, 0.4, 0.3, 0.9),
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.07),
			},
			switches: 1,
			want:     []catalog.Category{catalog.CategoryDrive},
		},
		{
			name: "no switches left",
			terms: []audio.Derived{
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.07),
			},
			switches: 0,
			want:     []catalog.Category{},
		},
		{
			name: "a device whose switches are already spoken for",
			terms: []audio.Derived{
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.07),
			},
			switches: -1,
			want:     []catalog.Category{},
		},
		{
			// 14 of the 43 here, and seven more earn no word at all.
			name: "words nothing answers",
			terms: []audio.Derived{
				at("short-decay", audio.KeyDecay, 0.2, 0.4, 0.13),
			},
			switches: 3,
			want:     []catalog.Category{},
		},
		{
			name:     "nothing earned",
			switches: 3,
			want:     []catalog.Category{},
		},
		{
			// Two words equally far out come back in the same order every run
			// rather than however the slice arrived.
			name: "a tie breaks on the kind",
			terms: []audio.Derived{
				at("long-decay", audio.KeyDecay, 0.5, 0.3, 0.5),
				at("saturated", audio.KeyHarmonics, 0.4, 0.3, 0.5),
			},
			switches: 3,
			want: []catalog.Category{
				catalog.CategoryDrive, catalog.CategoryReverb,
			},
		},
		{
			// And two words on one kind, equally far out, break on the word,
			// so which of them the block is attributed to is the same every
			// run too.
			name: "a tie on one kind breaks on the word",
			terms: []audio.Derived{
				at("mid-forward", audio.KeyMid, 0.2, 0.1, 0.1),
				at("bright", audio.KeyCentroid, 200, 100, 100),
			},
			switches: 3,
			want:     []catalog.Category{catalog.CategoryEQ},
			term:     "bright",
		},
	} {
		s.Run(tt.name, func() {
			got := translate.AskedFor(tt.terms, tt.held, tt.switches)

			kinds := make([]catalog.Category, 0, len(got))
			for _, asked := range got {
				kinds = append(kinds, asked.Kind())
			}

			s.Require().Equal(tt.want, kinds)

			if tt.term != "" {
				s.Require().Equal(tt.term, got[0].Term())
			}
		})
	}
}

// TestStompedable covers which kinds count against a device's switches.
//
// One method and one table, so a case is a row rather than a file.
func (s *StompPublicTestSuite) TestStompedable() {
	for _, tt := range []struct {
		kind catalog.Category
		want bool
	}{
		{catalog.CategoryDrive, true},
		{catalog.CategoryFilter, true},
		{catalog.CategoryMod, true},
		{catalog.CategoryDelay, true},
		{catalog.CategoryReverb, true},
		{catalog.CategoryPitch, true},
		{catalog.CategoryEQ, true},
		// The sound rather than an effect on it: bypassing either mid-song is
		// not a thing anybody does.
		{catalog.CategoryAmp, false},
		{catalog.CategoryCab, false},
		// Set once and left was the reasoning, and the person holding the
		// pedal disagreed: two presets whose whole chain was a compressor,
		// an amplifier and a cabinet reached the hardware with nothing on
		// any switch. The corpus stomps 1,154 of them.
		{catalog.CategoryComp, true},
		{catalog.CategoryGate, false},
		{catalog.CategoryUtility, false},
	} {
		s.Run(string(tt.kind), func() {
			s.Require().Equal(tt.want, translate.Stompedable(tt.kind))
		})
	}
}

// TestSpliced covers where one more block goes in somebody's chain.
//
// One method and one table, so a case is a row rather than a file. Position is
// not decoration: a drive ahead of an amplifier overdrives its input and a
// drive behind it does something else entirely.
func (s *StompPublicTestSuite) TestSpliced() {
	chain := func(roles ...rig.Role) []rig.ChainEntry {
		out := make([]rig.ChainEntry, 0, len(roles))
		for _, role := range roles {
			out = append(out, rig.ChainEntry{Role: role})
		}

		return out
	}

	for _, tt := range []struct {
		name  string
		held  []rig.ChainEntry
		entry rig.Role
		want  []rig.Role
	}{
		{
			name:  "a drive goes in front of the amplifier",
			held:  chain(rig.RoleComp, rig.RoleAmp, rig.RoleCab),
			entry: rig.RoleDrive,
			want:  []rig.Role{rig.RoleComp, rig.RoleDrive, rig.RoleAmp, rig.RoleCab},
		},
		{
			name:  "a reverb goes behind the cabinet",
			held:  chain(rig.RoleComp, rig.RoleAmp, rig.RoleCab),
			entry: rig.RoleReverb,
			want:  []rig.Role{rig.RoleComp, rig.RoleAmp, rig.RoleCab, rig.RoleReverb},
		},
		{
			name:  "an equaliser goes between the cabinet and the reverb",
			held:  chain(rig.RoleAmp, rig.RoleCab, rig.RoleReverb),
			entry: rig.RoleEQ,
			want:  []rig.Role{rig.RoleAmp, rig.RoleCab, rig.RoleEQ, rig.RoleReverb},
		},
		{
			name:  "in front of everything",
			held:  chain(rig.RoleAmp),
			entry: rig.RoleWah,
			want:  []rig.Role{rig.RoleWah, rig.RoleAmp},
		},
		{
			name:  "an empty chain",
			entry: rig.RoleReverb,
			want:  []rig.Role{rig.RoleReverb},
		},
	} {
		s.Run(tt.name, func() {
			got := translate.Spliced(tt.held, rig.ChainEntry{Role: tt.entry})

			roles := make([]rig.Role, 0, len(got))
			for _, entry := range got {
				roles = append(roles, entry.Role)
			}

			s.Require().Equal(tt.want, roles)
		})
	}
}

// TestSaid covers the sentence a chosen block gains beside its measurement.
//
// One method and one table, so a case is a row rather than a file.
func (s *StompPublicTestSuite) TestSaid() {
	measured := "Chosen by measurement rather than by anybody's say-so."
	corpus := "added by the compiler rather than named in the rig."

	for _, tt := range []struct {
		name string
		held []tone.Evidence
		want []string
	}{
		{
			name: "the measured evidence gains it",
			held: []tone.Evidence{{Kind: tone.EvidenceAudio, Note: &measured}},
			want: []string{measured + " and so on"},
		},
		{
			// That one says why a block is in the chain at all, which this
			// does not qualify.
			name: "a corpus note is left alone",
			held: []tone.Evidence{{Kind: tone.EvidenceCorpus, Note: &corpus}},
			want: []string{corpus},
		},
		{
			name: "found behind one it does not qualify",
			held: []tone.Evidence{
				{Kind: tone.EvidenceCorpus, Note: &corpus},
				{Kind: tone.EvidenceAudio, Note: &measured},
			},
			want: []string{corpus, measured + " and so on"},
		},
		{
			name: "evidence with nothing written on it",
			held: []tone.Evidence{{Kind: tone.EvidenceAudio}},
			want: []string{""},
		},
	} {
		s.Run(tt.name, func() {
			held := make([]tone.Evidence, len(tt.held))
			for i, at := range tt.held {
				held[i] = at

				if at.Note != nil {
					was := *at.Note
					held[i].Note = &was
				}
			}

			entry := rig.ChainEntry{Evidence: &held}
			translate.Said(&entry, "and so on")

			for i, want := range tt.want {
				got := ""
				if held[i].Note != nil {
					got = *held[i].Note
				}

				s.Require().Equal(want, got, "evidence %d", i)
			}
		})
	}
}

// TestSaidWithNoEvidence covers a block carrying none, which nothing adds to.
func (s *StompPublicTestSuite) TestSaidWithNoEvidence() {
	entry := rig.ChainEntry{}
	translate.Said(&entry, "and so on")
	s.Require().Nil(entry.Evidence)
}

func TestStompPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(StompPublicTestSuite))
}

// TestAimedAt covers finding a rig's records by the document's own identifier.
//
// One method and one table, so a case is a row rather than a file. This is the
// route task 268 established: a rig for somebody aiming at their own records
// is not circular, and the identifier is what reaches it without reading
// `ask.subject` as a target.
func (s *StompPublicTestSuite) TestAimedAt() {
	for _, tt := range []struct {
		name string
		id   string
		// earns is a word the records must have earned, and figures whether
		// the aim came back with anything to aim at.
		earns   string
		figures bool
		ok      bool
	}{
		{
			name: "a player, which is tried first",
			id:   "jaco-pastorius", earns: "bright", figures: true, ok: true,
		},
		{
			name: "a genre, which is tried after",
			id:   "punk", earns: "long-decay", figures: true, ok: true,
		},
		{
			// hard-rock displaces nothing a bass is weighed on, which task
			// 273 records. It still has records and still earns a word.
			name: "a genre that displaces nothing",
			id:   "hard-rock", earns: "long-decay", ok: true,
		},
		{
			// Under the bar of eight records from three players, so it is
			// reported rather than computed from.
			name: "a genre too thin to aim at",
			id:   "funk",
		},
		{name: "nobody of that name", id: "not-a-player-or-a-genre"},
		{name: "no name at all"},
	} {
		s.Run(tt.name, func() {
			got, terms, ok := translate.AimedAt(tt.id, s.deps)

			s.Require().Equal(tt.ok, ok)

			if !tt.ok {
				s.Require().Empty(terms)

				return
			}

			s.Require().Equal(tt.id, got.Named())

			earned := make([]string, 0, len(terms))
			for _, term := range terms {
				earned = append(earned, term.Term)
			}

			s.Require().Contains(earned, tt.earns)

			// A displacement of nothing is still a target, and the words are
			// what pick the kind rather than the figures.
			if tt.figures {
				s.Require().NotZero(got.Figures().Centroid)
			}
		})
	}
}

// TestStompable covers the effects a rig's own records ask for.
//
// Against the shipped corpus and the committed library, because what is being
// checked is that the 43 rigs here stop carrying the same two kinds. The
// arithmetic behind each half is tested on its own above.
func (s *StompPublicTestSuite) TestStompable() {
	for _, tt := range []struct {
		name string
		id   string
		// chain is what the rig already names, by role.
		chain []rig.Role
		// want is the role of each block it should get, in order, and said a
		// phrase the report has to carry.
		want []rig.Role
		said string
	}{
		{
			// Distinctive on a long decay and nothing else a pedal answers.
			name:  "a player whose notes ring on gets a reverb",
			id:    "glen-matlock",
			chain: []rig.Role{rig.RoleComp, rig.RoleAmp, rig.RoleCab},
			want:  []rig.Role{rig.RoleReverb},
			said:  "long-decay",
		},
		{
			// Bright and saturated, which are two different pedals.
			name:  "a player who is bright and saturated gets both",
			id:    "matt-freeman",
			chain: []rig.Role{rig.RoleComp, rig.RoleAmp, rig.RoleCab},
			want:  []rig.Role{rig.RoleEQ, rig.RoleDrive},
			said:  "saturated",
		},
		{
			// A researched rig already names the pedal he is documented with,
			// so nothing adds him a second drive.
			name: "a kind the rig already names is not doubled",
			id:   "matt-freeman",
			chain: []rig.Role{
				rig.RoleComp, rig.RoleDrive, rig.RoleAmp, rig.RoleCab,
			},
			want: []rig.Role{rig.RoleEQ},
			said: "mid-forward",
		},
		{
			// Distinctive only on a shorter decay, which nothing on a floor
			// answers. 14 of the 43 are in this state.
			name:  "a player nothing answers for gets nothing",
			id:    "les-claypool",
			chain: []rig.Role{rig.RoleComp, rig.RoleAmp, rig.RoleCab},
			said:  "no effect this device has a pedal for",
		},
		{
			name:  "a rig for nobody the corpus measures",
			id:    "not-a-player-or-a-genre",
			chain: []rig.Role{rig.RoleAmp},
		},
		{
			name:  "no identifier",
			chain: []rig.Role{rig.RoleAmp},
		},
	} {
		s.Run(tt.name, func() {
			chain := make([]rig.ChainEntry, 0, len(tt.chain))
			for _, role := range tt.chain {
				chain = append(chain, rig.ChainEntry{Role: role})
			}

			bass := rig.Instrument("bass")
			notes := translate.Notes{}

			got := translate.Stompable(
				tt.id,
				rig.Spec{Instrument: bass, Chain: chain},
				tone.Setup{}, s.deps, &notes)

			roles := make([]rig.Role, 0, len(got))
			for _, at := range got {
				entry := at.Entry
				roles = append(roles, entry.Role)

				// Switched off, every one of them: the base sound is the
				// amplifier until somebody steps on one.
				s.Require().NotNil(entry.Enabled, entry.Gear)
				s.Require().False(*entry.Enabled, entry.Gear)

				// And carrying why its kind is there, not only why its model
				// is: the kind is the decision somebody is most likely to
				// disagree with.
				s.Require().NotNil(entry.Evidence, entry.Gear)
				s.Require().Contains(
					*(*entry.Evidence)[0].Note, "It arrives switched off.")

				// The same sentence comes back beside the block, so a caller
				// never has to match a note to it.
				s.Require().NotEmpty(at.Said)
			}

			s.Require().Equal(tt.want, nilled(roles))

			if tt.said == "" {
				return
			}

			report := ""
			for _, note := range notes {
				report += note.Said + "\n"
			}

			s.Require().Contains(report, tt.said)
		})
	}
}

// TestStompableWithNothingMeasuredOfThatKind covers a kind the records ask for
// that this library has no reading of.
//
// Not a state the committed library is in: it holds 25 reverbs, 61 drives and
// 8 equalisers for this device. It is still the contract, and the honest
// answer is nothing rather than a block picked without a measurement behind
// it.
func (s *StompPublicTestSuite) TestStompableWithNothingMeasuredOfThatKind() {
	deps := s.deps
	deps.Measured.Blocks = maps.Clone(s.deps.Measured.Blocks)

	for id, block := range deps.Measured.Blocks {
		if block.Category == catalog.CategoryReverb {
			delete(deps.Measured.Blocks, id)
		}
	}

	bass := rig.Instrument("bass")
	notes := translate.Notes{}

	got := translate.Stompable("glen-matlock", rig.Spec{
		Instrument: bass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleComp}, {Role: rig.RoleAmp}, {Role: rig.RoleCab},
		},
	}, tone.Setup{}, deps, &notes)

	s.Require().Empty(got)

	report := ""
	for _, note := range notes {
		report += note.Said + "\n"
	}

	s.Require().Contains(report, "nothing of that kind has been measured")
}

// nilled is an empty slice read as nothing, which is what a table says when a
// rig gets no effect.
func nilled(
	roles []rig.Role,
) []rig.Role {
	if len(roles) == 0 {
		return nil
	}

	return roles
}
