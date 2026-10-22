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
	"os"
	"path/filepath"
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
		s.forGenre("punk"), tone.Setup{}, deps)
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
		// A kind nobody has thought about keeps its own word rather than
		// being confidently called an amplifier.
		{catalog.CategoryReverb, string(catalog.CategoryReverb)},
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
			"Drive": {
				Control: "Drive", Settled: -20, SilentBelow: -50,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
					{Value: 1, Figures: measured.Figures{Mid: 40, Level: -20}},
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
	// value that arrives is the one the sweep recorded.
	at := func(kind string, v float64) measured.Curve {
		return measured.Curve{
			Control: kind, Settled: -20, SilentBelow: -50,
			Points: []measured.Point{
				{Value: v, Figures: measured.Figures{Mid: 40, Level: -20}},
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
					{Value: 1, Figures: measured.Figures{Mid: 40, Level: -20}},
				},
			},
			"MidFreq": {
				Control: "MidFreq", Kind: "int", Settled: -20, SilentBelow: -50,
				Points: []measured.Point{
					{Value: 0, Figures: measured.Figures{Mid: 1, Level: -20}},
					{Value: 2, Figures: measured.Figures{Mid: 40, Level: -20}},
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
	// position that reads 40, which is the true one.
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
