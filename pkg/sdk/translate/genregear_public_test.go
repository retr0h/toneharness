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
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// GenreGearPublicTestSuite covers what a genre takes from its players, and what
// it gets on a switch.
//
// Against the shipped genres rather than a fixture, because the thing being
// checked is that the eleven the corpus can aim at stop producing one chain,
// and a fixture would only check the arithmetic.
type GenreGearPublicTestSuite struct {
	suite.Suite

	deps translate.Deps
}

func (s *GenreGearPublicTestSuite) SetupSuite() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	lib, err := measured.BuiltIn()
	s.Require().NoError(err)

	s.deps = translate.Deps{Catalog: cat, Measured: lib}
}

// forGenre is the ask a person writes when they name only a genre.
func (s *GenreGearPublicTestSuite) forGenre(
	name string,
) tone.Ask {
	bass := tone.AskInstrument("bass")

	return tone.Ask{Genre: []string{name}, Instrument: &bass}
}

// rigNamed answers with one researched rig per player named, so the mode over
// a genre's players is a fixture rather than the marketplace.
func (s *GenreGearPublicTestSuite) rigNamed(
	amps map[string]string,
) func(string) (rig.Spec, bool) {
	return func(who string) (rig.Spec, bool) {
		gear, ok := amps[who]
		if !ok {
			return rig.Spec{}, false
		}

		return rig.Spec{Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: gear},
		}}, true
	}
}

// TestAGenreTakesTheGearItsPlayersUsed covers the tier above measurement.
func (s *GenreGearPublicTestSuite) TestAGenreTakesTheGearItsPlayersUsed() {
	tests := []struct {
		name string
		// amps is what each player's researched rig names.
		amps map[string]string
		// want is the amplifier the chain must carry, and taken whether it
		// came from the players rather than from a measurement.
		want  string
		taken bool
	}{
		{
			// Punk's seven players, with an Ampeg SVT in four of them. The
			// real marketplace spells it three ways across five resolved
			// rigs, which is why the mode is taken per role rather than over
			// an amplifier-and-cabinet pair.
			name: "a majority of its players agree",
			amps: map[string]string{
				"dee-dee-ramone": "Ampeg SVT",
				"karl-alvarez":   "Ampeg SVT",
				"paul-simonon":   "Ampeg SVT",
				"glen-matlock":   "Fender Bassman",
			},
			want:  "Ampeg SVT",
			taken: true,
		},
		{
			// A plurality is not what a genre used, and a measurement is a
			// better answer than a mode nobody holds.
			name: "no majority falls through to the measurements",
			amps: map[string]string{
				"dee-dee-ramone": "Ampeg SVT",
				"karl-alvarez":   "Fender Bassman",
				"paul-simonon":   "MESA/Boogie Bass 400+",
				"glen-matlock":   "Aguilar Tone Hammer",
			},
			taken: false,
		},
		{
			name:  "nobody researched falls through too",
			amps:  map[string]string{},
			taken: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			deps := s.deps
			deps.RigNamed = s.rigNamed(tt.amps)

			built, notes, err := translate.Translate(
				s.forGenre("punk"), tone.Setup{}, deps)
			s.Require().NoError(err)

			said := ""
			for _, n := range notes {
				said += n.Said + "\n"
			}

			if !tt.taken {
				s.Require().Contains(said, "so the gear was measured for instead")

				return
			}

			s.Require().Contains(said, "taken rather than measured for")

			amp := ""

			for _, e := range built.Chain {
				if e.Role == rig.RoleAmp {
					amp = e.Gear
				}
			}

			s.Require().Equal(tt.want, amp)
		})
	}
}

// TestAGenreGetsEffectsSwitchedOff covers what arrives under a footswitch.
//
// The chain a genre rig carried was an amplifier, a cabinet and whatever the
// corpus added on a statistic, with nothing to step on. A genre's records
// cover a range no single setting reaches, so the far end of it goes under a
// switch instead of being averaged away.
func (s *GenreGearPublicTestSuite) TestAGenreGetsEffectsSwitchedOff() {
	built, _, err := translate.Translate(
		s.forGenre("punk"), tone.Setup{}, s.deps)
	s.Require().NoError(err)

	stompable := 0

	for _, e := range built.Chain {
		switch e.Role {
		case rig.RoleDrive, rig.RoleEQ:
		default:
			continue
		}

		stompable++

		s.Require().NotNil(e.Enabled,
			"%s arrived with no enabled field, which the contract reads as on",
			e.Gear)
		s.Require().False(*e.Enabled,
			"%s arrived switched on, so loading the preset plays it", e.Gear)
	}

	s.Require().Positive(stompable, "a genre got nothing to step on")
}

// TestAnEffectIsSetFromItsSweep covers tuning rather than defaulting.
func (s *GenreGearPublicTestSuite) TestAnEffectIsSetFromItsSweep() {
	deps := s.deps
	deps.CurvesFor = func(catalog.ModelID) (measured.Curves, bool) {
		return measured.Curves{}, false
	}

	_, notes, err := translate.Translate(
		s.forGenre("punk"), tone.Setup{}, deps)
	s.Require().NoError(err)

	said := ""
	for _, n := range notes {
		said += n.Said + "\n"
	}

	// A block with no sweep says so rather than arriving quietly untouched.
	s.Require().Contains(said, "no sweep for it")
}

// TestTheNearestEffectThatChangesTheNotesIsRefused covers the refusal where a
// build is choosing on somebody's behalf.
//
// The committed library does not quite reach it. Five of this device's 61
// drives add an interval nobody played, and against the eleven genres each
// reads second or third rather than first: pop-punk's nearest of the five is
// a Sub Oct Fuzz 0.69 out against the Ampeg Scrambler's 0.21, and rock's is
// an Octave Fuzz 0.87 against the Metal Zone's 0.68. Those are margins a
// single record moves, so the library is narrowed here to put one in front,
// which is the state the ranking arrives at on its own sooner or later.
func (s *GenreGearPublicTestSuite) TestTheNearestEffectThatChangesTheNotesIsRefused() {
	const (
		octave = "HD2_DM4OctaveFuzz"
		plain  = "HD2_DistHedgehogD9"
	)

	deps := s.deps
	deps.Measured.Blocks = maps.Clone(s.deps.Measured.Blocks)

	// Two drives in the library and nothing else, so which of them reads
	// nearest is not a question about the other 59.
	for id, block := range deps.Measured.Blocks {
		if block.Category == catalog.CategoryDrive && id != octave && id != plain {
			delete(deps.Measured.Blocks, id)
		}
	}

	// The octave fuzz keeps its own reading and the ordinary drive is moved
	// where no bass record sits, so the ranking has to answer with the one
	// that changes the notes.
	was := deps.Measured.Blocks[plain]
	was.Centroid, was.Low, was.Mid, was.High = 9000, 0, 0, 100
	deps.Measured.Blocks[plain] = was

	built, notes, err := translate.Translate(
		s.forGenre("punk"), tone.Setup{}, deps)
	s.Require().NoError(err)

	said := ""
	for _, n := range notes {
		said += n.Said + "\n"
	}

	s.Require().Contains(said,
		"Tycobrahe Octavia read nearest and was refused: it changes the notes "+
			"rather than the tone")

	// And the chain got the one that does not, rather than nothing at all.
	gear := []string(nil)
	for _, e := range built.Chain {
		if e.Role == rig.RoleDrive {
			gear = append(gear, e.Gear)
		}
	}

	s.Require().Equal([]string{"MAXON SD9 Sonic Distortion"}, gear)
}

// TestAnUnmeasuredGenreGetsNothing covers the genre nobody can aim at.
func (s *GenreGearPublicTestSuite) TestAnUnmeasuredGenreGetsNothing() {
	built, _, err := translate.Translate(
		s.forGenre("not-a-measured-genre"), tone.Setup{}, s.deps)
	// Either it refuses outright or it builds nothing stompable. Both are
	// honest; what must not happen is a drive chosen against figures no
	// records back.
	if err != nil {
		s.Require().ErrorIs(err, translate.ErrNothingToBuildFrom)

		return
	}

	for _, e := range built.Chain {
		s.Require().NotEqual(rig.RoleDrive, e.Role,
			"a genre nobody measured got a drive chosen for it")
	}
}

func TestGenreGearPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(GenreGearPublicTestSuite))
}

// TestAnEffectIsTunedFromItsSweep covers the controls a sweep sets.
//
// Against the committed sweeps rather than a fixture, because what is being
// checked is that a block added to a chain stops arriving at the catalog's
// default, and a fixture would check the arithmetic [measured.Curves.Toward]
// is already tested on.
//
// rock rather than punk, which this read before task 272. punk sits 126Hz and
// 21.7 points of midrange from the other genres and the MAXON SD9 it chooses
// moves the midrange by 2.0 points across the positions that leave the block
// at working level, so nothing on it reaches what punk asks for and the
// defaults correctly stand. rock asks for 11Hz and 10.9 points, which its
// BOSS Metal Zone can answer.
func (s *GenreGearPublicTestSuite) TestAnEffectIsTunedFromItsSweep() {
	deps := s.deps
	deps.CurvesFor = func(model catalog.ModelID) (measured.Curves, bool) {
		f, err := os.Open(filepath.Join(
			"..", "..", "..", "resources", "sweeps", "hx-stomp",
			string(model)+".json"))
		if err != nil {
			return measured.Curves{}, false
		}

		defer func() { _ = f.Close() }()

		got, err := measured.LoadCurves(f)
		if err != nil {
			return measured.Curves{}, false
		}

		return got, true
	}

	built, notes, err := translate.Translate(
		s.forGenre("rock"), tone.Setup{}, deps)
	s.Require().NoError(err)

	said := ""
	for _, n := range notes {
		said += n.Said + "\n"
	}

	s.Require().Contains(said, "set from its sweep")

	set := 0

	for _, e := range built.Chain {
		switch e.Role {
		case rig.RoleDrive, rig.RoleEQ:
		default:
			continue
		}

		if e.Controls == nil {
			continue
		}

		for name, v := range *e.Controls {
			set++

			// A level pinned at the bottom of its range is the failure this
			// guards: matching figures reads silence as a close match to
			// almost any target, and a block turned down to -60dB silences
			// the chain the moment somebody steps on it.
			f, ok := v.Float()
			s.Require().True(ok, "%s.%s is not a number", e.Gear, name)
			s.Require().Greater(f, -60.0, "%s.%s", e.Gear, name)
		}
	}

	s.Require().Positive(set, "nothing was set from a sweep")
}

// TestPlural covers the word a sentence uses for several blocks of one kind.
//
// The evidence said "amplifiers" whatever the block was, which was true while
// only an amplifier was chosen by measurement. A genre rig carries a drive and
// an equaliser picked the same way, and both arrived claiming to be the
// closest of the 57 and the 8 bass amplifiers this device models.
func (s *GenreGearPublicTestSuite) TestPlural() {
	for _, tt := range []struct {
		kind catalog.Category
		says string
	}{
		{catalog.CategoryAmp, "amplifiers"},
		{catalog.CategoryCab, "cabinets"},
		{catalog.CategoryDrive, "drives"},
		{catalog.CategoryEQ, "equalisers"},
		{catalog.CategoryComp, "compressors"},
		// The kinds task 274 reaches, since a rig gets whichever its own
		// records ask for and a reverb is one of three it can be.
		{catalog.CategoryReverb, "reverbs"},
		{catalog.CategoryDelay, "delays"},
		{catalog.CategoryFilter, "filters"},
		{catalog.CategoryMod, "modulations"},
		{catalog.CategoryWah, "wahs"},
		{catalog.CategoryPitch, "pitch blocks"},
		{catalog.CategoryGate, "gates"},
		// Plumbing, which nothing chooses by measurement: nobody picks a
		// send for how it sounds, so it keeps its own word.
		{catalog.CategoryUtility, string(catalog.CategoryUtility)},
	} {
		s.Run(string(tt.kind), func() {
			got := translate.Because(
				translate.Aim{}, "A Block", 9, 150, "bass", tt.kind)

			s.Require().NotNil(got.Note)
			s.Require().Contains(*got.Note, "9 bass "+tt.says+" this device models")
		})
	}
}

// TestCommonest covers counting two spellings of one box once.
func (s *GenreGearPublicTestSuite) TestCommonest() {
	cab := func(gear string) rig.ChainEntry {
		return rig.ChainEntry{Role: rig.RoleCab, Gear: gear}
	}

	tests := []struct {
		name string
		said []rig.ChainEntry
		// want is the gear that wins and how many named it.
		want  string
		count int
		ok    bool
	}{
		{
			// The bug this guards. Punk's five researched rigs name this one
			// cabinet two ways, which is four players on one box and was
			// counted as two on each. Resolving each name to a single model
			// does not fix it: "Ampeg 8x10" reaches three cabinets and "8x10
			// Ampeg SVT-810AV" reaches two of those three, so the first of
			// each differs while the catalog says they overlap.
			name: "two spellings of one cabinet count as one",
			said: []rig.ChainEntry{
				cab("Ampeg 8x10"),
				cab("8x10 Ampeg SVT-810AV"),
				cab("Ampeg 8x10"),
				cab("8x10 Ampeg SVT-810AV"),
			},
			want:  "Ampeg 8x10",
			count: 4,
			ok:    true,
		},
		{
			// The spelling the earliest player used, so a tie answers the same
			// way every run.
			name: "the first spelling is the one reported",
			said: []rig.ChainEntry{
				cab("8x10 Ampeg SVT-810AV"),
				cab("Ampeg 8x10"),
			},
			want:  "8x10 Ampeg SVT-810AV",
			count: 2,
			ok:    true,
		},
		{
			name: "two different cabinets stay apart",
			said: []rig.ChainEntry{
				cab("Ampeg 8x10"),
				cab("Ampeg 8x10"),
				cab("Mesa/Boogie 2x15"),
			},
			want:  "Ampeg 8x10",
			count: 2,
			ok:    true,
		},
		{
			// A name the catalog places nowhere is its own group, which is the
			// honest failure: nothing here can tell two unplaceable spellings
			// of one box apart.
			name: "a name the catalog cannot place groups by spelling",
			said: []rig.ChainEntry{
				cab("Not A Cabinet Anybody Models"),
				cab("Not A Cabinet Anybody Models"),
			},
			want:  "Not A Cabinet Anybody Models",
			count: 2,
			ok:    true,
		},
		{name: "nothing named answers nothing", said: nil, ok: false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, count, ok := translate.Commonest(
				s.deps.Catalog, tt.said, catalog.CategoryCab)

			s.Require().Equal(tt.ok, ok)

			if !tt.ok {
				return
			}

			s.Require().Equal(tt.want, got.Gear)
			s.Require().Equal(tt.count, count)
		})
	}
}

// TestModelsFor covers which models a gear name reaches.
func (s *GenreGearPublicTestSuite) TestModelsFor() {
	// Without a catalog there is nothing to resolve against, so a name is
	// its own answer and two names group only if they are spelled alike.
	s.Require().Equal(
		map[string]bool{"Ampeg 8x10": true},
		translate.ModelsFor(nil, "Ampeg 8x10", catalog.CategoryCab))

	// The whole set rather than the nearest, because what the caller asks is
	// whether two names could mean one box.
	got := translate.ModelsFor(
		s.deps.Catalog, "Ampeg 8x10", catalog.CategoryCab)
	s.Require().Greater(len(got), 1)

	// And a name nothing models answers with itself.
	s.Require().Equal(
		map[string]bool{"Nothing Models This": true},
		translate.ModelsFor(
			s.deps.Catalog, "Nothing Models This", catalog.CategoryCab))
}

// TestStompedForRefuses covers the asks that get nothing on a switch.
func (s *GenreGearPublicTestSuite) TestStompedForRefuses() {
	tests := []struct {
		name string
		ask  tone.Ask
	}{
		{
			// Every ask carries a genre field, and `nearestTo` falls back to a
			// recording where one is named, so without the guard a request
			// pointing at a record gained a drive and an equaliser nobody
			// asked for.
			name: "an ask naming no genre",
			ask:  tone.Ask{},
		},
		{
			name: "a genre no records carry",
			ask:  s.forGenre("not-a-measured-genre"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			notes := translate.Notes{}

			s.Require().Empty(translate.StompedFor(
				tt.ask, tone.Setup{}, s.deps, &notes))
		})
	}
}

// TestTunedRefuses covers the cases where the defaults have to stand.
func (s *GenreGearPublicTestSuite) TestTunedRefuses() {
	// A block already carrying controls keeps them, with the swept values
	// merged over the top rather than replacing the map.
	held := map[string]catalog.Setting{"Mix": catalog.Set(catalog.Float(1))}
	entry := rig.ChainEntry{
		Role: rig.RoleDrive, Gear: "Teemah!", Controls: &held,
	}

	deps := s.deps
	deps.CurvesFor = func(catalog.ModelID) (measured.Curves, bool) {
		return measured.Curves{Controls: map[string]measured.Curve{
			// Reaching 29.5 points of midrange above the middle of its own
			// positions, which is more than the 21.7 punk asks for. A dial
			// is set by the size of the ask as well as its direction since
			// task 272, so a curve that cannot reach what is asked declines
			// and this would be testing that instead.
			"Drive": {
				Control: "Drive", Settled: -20, SilentBelow: -50,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
					{Value: 1, Figures: measured.Figures{Mid: 60, Level: -20}},
				},
			},
		}}, true
	}

	notes := translate.Notes{}
	translate.Tuned(
		&entry, "HD2_DistTeemah", s.forGenre("punk"), deps, &notes)

	s.Require().NotNil(entry.Controls)
	s.Require().Contains(*entry.Controls, "Mix", "it dropped what was there")
	s.Require().Contains(*entry.Controls, "Drive", "it set nothing")

	// An ask with nothing to aim at tunes nothing, and a block whose whole
	// sweep moved less than its noise tunes nothing either.
	for _, tt := range []struct {
		name   string
		ask    tone.Ask
		curves measured.Curves
	}{
		{name: "nothing to aim at", ask: tone.Ask{}},
		{
			name: "a sweep that moves nothing",
			ask:  s.forGenre("punk"),
			curves: measured.Curves{Controls: map[string]measured.Curve{
				"Hum": {
					Control: "Hum", Settled: -20, SilentBelow: -50,
					Noise: map[audio.Figure]float64{audio.KeyMid: 50},
					Points: []measured.Point{
						{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
						{Value: 1, Figures: measured.Figures{Mid: 2, Level: -20}},
					},
				},
			}},
		},
	} {
		s.Run(tt.name, func() {
			bare := rig.ChainEntry{Role: rig.RoleDrive, Gear: "Teemah!"}

			d := s.deps
			d.CurvesFor = func(catalog.ModelID) (measured.Curves, bool) {
				return tt.curves, true
			}

			n := translate.Notes{}
			translate.Tuned(&bare, "HD2_DistTeemah", tt.ask, d, &n)

			s.Require().Nil(bare.Controls)
		})
	}
}

// TestTunedSaysWhenNothingReached covers what the block itself is left saying.
//
// A target's words are what [aim.because] calls "what moves the controls off
// the medians this instrument's presets sit at", and since task 272 a dial
// that cannot reach the ask is left where the catalog put it. A document
// carrying the first sentence and none of the controls is the one thing that
// evidence exists to prevent, so the qualification goes on the block.
//
// Onto the audio evidence and nowhere else: a corpus note says why a block is
// in the chain at all, which this does not qualify.
func (s *GenreGearPublicTestSuite) TestTunedSaysWhenNothingReached() {
	// A sweep whose dial moves the midrange by a point, against punk asking
	// for 21.7.
	short := measured.Curves{Controls: map[string]measured.Curve{
		"Drive": {
			Control: "Drive", Settled: -20, SilentBelow: -50,
			Points: []measured.Point{
				{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
				{Value: 1, Figures: measured.Figures{Mid: 2, Level: -20}},
			},
		},
	}}

	said := "Chosen by measurement rather than by anybody's say-so."
	why := "added by the compiler rather than named in the rig."

	for _, tt := range []struct {
		name string
		held []tone.Evidence
		// want is the note each piece of evidence must read afterwards.
		want []string
	}{
		{
			name: "the measured evidence gains the sentence",
			held: []tone.Evidence{{Kind: tone.EvidenceAudio, Note: &said}},
			want: []string{said + " Its controls were not moved: none of them " +
				"reaches what that asks for, so they are the catalog's defaults."},
		},
		{
			name: "a corpus note is left alone",
			held: []tone.Evidence{{Kind: tone.EvidenceCorpus, Note: &why}},
			want: []string{why},
		},
		{
			// The audio evidence is found past one this does not qualify,
			// rather than only where it happens to come first.
			name: "the measured evidence is found behind another",
			held: []tone.Evidence{
				{Kind: tone.EvidenceCorpus, Note: &why},
				{Kind: tone.EvidenceAudio, Note: &said},
			},
			want: []string{why, said + " Its controls were not moved: none of " +
				"them reaches what that asks for, so they are the catalog's " +
				"defaults."},
		},
		{
			name: "evidence with nothing written on it",
			held: []tone.Evidence{{Kind: tone.EvidenceAudio}},
			want: []string{""},
		},
	} {
		s.Run(tt.name, func() {
			held := slices.Clone(tt.held)
			entry := rig.ChainEntry{
				Role: rig.RoleDrive, Gear: "Teemah!", Evidence: &held,
			}

			d := s.deps
			d.CurvesFor = func(catalog.ModelID) (measured.Curves, bool) {
				return short, true
			}

			n := translate.Notes{}
			translate.Tuned(
				&entry, "HD2_DistTeemah", s.forGenre("punk"), d, &n)

			s.Require().Nil(entry.Controls)

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

// TestTunedSaysNothingWasAsked covers the other reason a block keeps its
// defaults, which is not about the block.
//
// hard-rock displaces the centroid by zero and the midrange by zero, and a
// bass is weighed on those two alone, so there is nothing for any sweep to
// answer. Reporting that as "no control of it reaches what was asked"
// describes a block that was never asked.
func (s *GenreGearPublicTestSuite) TestTunedSaysNothingWasAsked() {
	said := "Chosen by measurement rather than by anybody's say-so."
	held := []tone.Evidence{{Kind: tone.EvidenceAudio, Note: &said}}
	entry := rig.ChainEntry{
		Role: rig.RoleDrive, Gear: "Teemah!", Evidence: &held,
	}

	deps := s.deps
	deps.CurvesFor = func(catalog.ModelID) (measured.Curves, bool) {
		return measured.Curves{Controls: map[string]measured.Curve{
			"Drive": {
				Control: "Drive", Settled: -20, SilentBelow: -50,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
					{Value: 1, Figures: measured.Figures{Mid: 60, Level: -20}},
				},
			},
		}}, true
	}

	notes := translate.Notes{}
	translate.Tuned(
		&entry, "HD2_DistTeemah", s.forGenre("hard-rock"), deps, &notes)

	s.Require().Nil(entry.Controls)

	report := ""
	for _, n := range notes {
		report += n.Said + "\n"
	}

	s.Require().Contains(report, "hard-rock displaces nothing a bass is weighed on")
	s.Require().NotContains(report, "no control of it reaches")

	s.Require().Equal(said+" Its controls were not moved: hard-rock displaces "+
		"none of the figures a bass is weighed on, so there was nothing to aim "+
		"them at.", *held[0].Note)
}

// TestDisplacedTowardTheEdgeOfTheRange covers which end of its own range a
// genre aims at.
//
// The middle is what every genre shares: ten of the eleven this corpus can aim
// at displace between 0.2 and 6.4 points from one baseline, against a field of
// blocks 50 points wide. So a figure a word points at aims at the edge of the
// genre's own distribution instead, and the word says which edge.
func (s *GenreGearPublicTestSuite) TestDisplacedTowardTheEdgeOfTheRange() {
	for _, tt := range []struct {
		// slug is the genre, and up whether the word on its mid band points at
		// more of it or less.
		slug string
		up   bool
	}{
		// punk earns mid-forward, so it aims at the top of its own mid range,
		// 0.237 rather than the 0.114 in the middle.
		{slug: "punk", up: true},
		// grunge earns scooped, which is the same comparison the other way, so
		// it aims at the bottom.
		{slug: "grunge", up: false},
	} {
		s.Run(tt.slug, func() {
			got, ok := audio.ShippedGenre(tt.slug)
			s.Require().True(ok)

			// Against a baseline of nothing, so what comes back is the
			// displacement itself rather than a point.
			at := translate.DisplacedToward(got, measured.Figures{})

			// The spread the genre's own records cover, in the units a block
			// is measured in.
			edge := got.Across.Mid.High
			if !tt.up {
				edge = got.Across.Mid.Low
			}

			want := 100 * (edge - got.Elsewhere[audio.KeyMid])

			s.Require().InDelta(want, at.Mid, 1e-6,
				"%s did not aim at the %s of its own range", tt.slug,
				map[bool]string{true: "top", false: "bottom"}[tt.up])
		})
	}
}

// TestTunedWritesEachControlInItsOwnKind covers the three kinds a swept
// control comes in.
//
// The fault this guards cost four of the eleven genres. Every value was
// written as a float, and of the swept drives' controls 163 are floats, 20 are
// integers and 16 are booleans: the Kinky Boost's `Bright` is one of the
// sixteen, and `presets compile` refused the chain with "parameter Bright:
// expected bool, got float".
func (s *GenreGearPublicTestSuite) TestTunedWritesEachControlInItsOwnKind() {
	// One swept position each, so there is nothing to choose between and the
	// value that arrives is the one the sweep recorded. Reaching 29.5 points
	// of midrange either side of the middle, which covers the 21.7 punk asks
	// for: since task 272 a dial that cannot reach the ask declines.
	at := func(kind string, v float64) measured.Curve {
		return measured.Curve{
			Control: kind, Settled: -20, SilentBelow: -50,
			Points: []measured.Point{
				{Value: v, Figures: measured.Figures{Mid: 60, Level: -20}},
				{Value: v, Figures: measured.Figures{Mid: 1, Level: -20}},
			},
		}
	}

	deps := s.deps
	deps.CurvesFor = func(catalog.ModelID) (measured.Curves, bool) {
		return measured.Curves{Controls: map[string]measured.Curve{
			"Bright": {
				Control: "Bright", Kind: "bool", Settled: -20, SilentBelow: -50,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
					{Value: 1, Figures: measured.Figures{Mid: 60, Level: -20}},
				},
			},
			"MidFreq": {
				Control: "MidFreq", Kind: "int", Settled: -20, SilentBelow: -50,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
					{Value: 2, Figures: measured.Figures{Mid: 60, Level: -20}},
				},
			},
			"Drive": func() measured.Curve {
				c := at("float", 0.75)
				c.Kind = "float"
				c.Points[1].Value = 0.25

				return c
			}(),
		}}, true
	}

	entry := rig.ChainEntry{Role: rig.RoleDrive, Gear: "Kinky Boost"}
	notes := translate.Notes{}

	translate.Tuned(
		&entry, "HD2_DistKinkyBoost", s.forGenre("punk"), deps, &notes)

	s.Require().NotNil(entry.Controls)

	got := *entry.Controls

	// A boolean, written as one. Aiming at punk's mid-forward corner picks the
	// position that reads 60, which is the true one.
	on, ok := got["Bright"].Bool()
	s.Require().True(ok, "Bright did not come back as a boolean")
	s.Require().True(on)

	// An integer, written as one rather than as 2.0.
	n, ok := got["MidFreq"].Int()
	s.Require().True(ok, "MidFreq did not come back as an integer")
	s.Require().Equal(int64(2), n)

	// And a float, which is what every other control is.
	f, ok := got["Drive"].Float()
	s.Require().True(ok, "Drive did not come back as a float")
	s.Require().InDelta(0.75, f, 1e-9)
}

// TestAGenreChoosesItsMicrophone covers the last lever on the cabinet.
//
// Eight of the eleven genres arrived on the same microphone at the same
// distance, because nothing chose them and a corpus median is one answer for
// everybody. The microphone is the biggest mover this device has: across the
// twelve on one cabinet it moves the centroid by a median of 32.7Hz, where the
// whole choice of amplifier for punk turned on 14Hz.
func (s *GenreGearPublicTestSuite) TestAGenreChoosesItsMicrophone() {
	// Every player on one 8x10, so the cabinet is taken rather than measured
	// for and there is a microphone to choose. The cabinet is the real one,
	// because the choosing reads its committed sweep.
	withCab := func(string) (rig.Spec, bool) {
		return rig.Spec{Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			{Role: rig.RoleCab, Gear: "Ampeg 8x10"},
		}}, true
	}

	deps := s.deps
	deps.RigNamed = withCab
	deps.CurvesFor = s.sweeps()

	// Two genres leaning opposite ways on the centroid. punk's records sit
	// 126Hz brighter than the other genres' and the metals' 20 to 25Hz darker,
	// so they must not reach the same microphone.
	picked := map[string]int64{}

	for _, slug := range []string{"punk", "metal"} {
		built, _, err := translate.Translate(
			s.forGenre(slug), tone.Setup{}, deps)
		s.Require().NoError(err)

		for _, e := range built.Chain {
			if e.Role != rig.RoleCab || e.Controls == nil {
				continue
			}

			at, ok := (*e.Controls)["Mic"]
			s.Require().True(ok, "%s: no microphone was chosen", slug)

			n, ok := at.Int()
			s.Require().True(ok, "%s: the microphone is not a whole number", slug)
			picked[slug] = n
		}
	}

	s.Require().Len(picked, 2, "a genre got no cabinet to choose a mic on")
	s.Require().NotEqual(picked["punk"], picked["metal"],
		"a genre 126Hz brighter than the others and one 25Hz darker reached "+
			"the same microphone, which is the collapse this exists to stop")
}

// sweeps reads the committed curves, which is what the choosing rests on.
func (s *GenreGearPublicTestSuite) sweeps() func(catalog.ModelID) (measured.Curves, bool) {
	return func(model catalog.ModelID) (measured.Curves, bool) {
		f, err := os.Open(filepath.Join(
			"..", "..", "..", "resources", "sweeps", "hx-stomp",
			string(model)+".json"))
		if err != nil {
			return measured.Curves{}, false
		}

		defer func() { _ = f.Close() }()

		got, err := measured.LoadCurves(f)
		if err != nil {
			return measured.Curves{}, false
		}

		return got, true
	}
}

// TestChoseRefuses covers the cabinets with no microphone to choose.
func (s *GenreGearPublicTestSuite) TestChoseRefuses() {
	cab := func() rig.ChainEntry {
		return rig.ChainEntry{Role: rig.RoleCab, Gear: "Ampeg 8x10"}
	}

	tests := []struct {
		name string
		// swept is what the sweeps answer with, nil for none offered at all.
		swept func(catalog.ModelID) (measured.Curves, bool)
		ask   tone.Ask
		// gear names the cabinet, where the default is not the point.
		gear string
	}{
		{
			// A release carries the fingerprints and not the sweeps, so the
			// cabinet keeps whatever the compiler fills.
			name: "no sweeps were offered",
			ask:  s.forGenre("punk"),
		},
		{
			name: "no sweep for that cabinet",
			swept: func(catalog.ModelID) (measured.Curves, bool) {
				return measured.Curves{}, false
			},
			ask: s.forGenre("punk"),
		},
		{
			name:  "nothing to aim at",
			swept: s.sweeps(),
			ask:   tone.Ask{},
		},
		{
			// A gear name the catalog places nowhere falls back to the name
			// itself, which is not a model, so there is no block to read a
			// control count off.
			name: "a cabinet the catalog does not model",
			swept: func(catalog.ModelID) (measured.Curves, bool) {
				return measured.Curves{Controls: map[string]measured.Curve{
					"Mic": {Control: "Mic", Settled: -20},
				}}, true
			},
			ask:  s.forGenre("punk"),
			gear: "Nothing Models This Cabinet",
		},
		{
			name: "a sweep with no microphone on it",
			swept: func(catalog.ModelID) (measured.Curves, bool) {
				return measured.Curves{Controls: map[string]measured.Curve{
					"LowCut": {Control: "LowCut", Settled: -20},
				}}, true
			},
			ask: s.forGenre("punk"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			deps := s.deps
			deps.CurvesFor = tt.swept

			entry := cab()
			if tt.gear != "" {
				entry.Gear = tt.gear
			}

			notes := translate.Notes{}

			translate.Chose(&entry, tt.ask, deps, &notes)

			s.Require().Nil(entry.Controls,
				"it set a microphone with nothing to read it from")
		})
	}
}

// TestChoseKeepsWhatWasAlreadySet covers a cabinet arriving with controls.
func (s *GenreGearPublicTestSuite) TestChoseKeepsWhatWasAlreadySet() {
	held := map[string]catalog.Setting{"LowCut": catalog.Set(catalog.Float(37))}
	entry := rig.ChainEntry{
		Role: rig.RoleCab, Gear: "Ampeg 8x10", Controls: &held,
	}

	deps := s.deps
	deps.CurvesFor = s.sweeps()

	notes := translate.Notes{}
	translate.Chose(&entry, s.forGenre("punk"), deps, &notes)

	s.Require().NotNil(entry.Controls)
	s.Require().Contains(*entry.Controls, "LowCut", "it dropped what was there")
	s.Require().Contains(*entry.Controls, "Mic", "it chose no microphone")
}
