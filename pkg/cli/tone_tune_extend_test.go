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

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// errBadStatistics stands in for an archive that will not decode.
var errBadStatistics = errors.New("truncated")

// ExtendTestSuite covers writing the rig the measurement asked for.
//
// The shortlist has always been printed and never actionable: a table in a
// terminal is gone at the next command, so acting on it meant reading five rows,
// picking one, finding the gear's real name, working out which side of the
// amplifier it belongs and hand-editing a document. This writes that document.
type ExtendTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockTuner
}

func (s *ExtendTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockTuner(s.ctrl)
}

// cat is the real catalog, because the names and the categories this writes come
// out of it and a fake one would be testing the fake.
func (s *ExtendTestSuite) cat() *catalog.Catalog {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	return cat
}

// TestPlaced covers which side of the amplifier an added block lands.
//
// Position is not decoration: drive ahead of an amplifier overdrives its input
// and drive after it does something else.
//
// One method and one table, so a case is a row rather than a file.
func (s *ExtendTestSuite) TestPlaced() {
	amp := rig.ChainEntry{Gear: "Ampeg SVT", Role: rig.RoleAmp}
	cab := rig.ChainEntry{Gear: "Ampeg 8x10", Role: rig.RoleCab}
	add := rig.ChainEntry{Gear: "Added", Role: rig.RoleEQ}

	for _, tt := range []struct {
		name   string
		chain  []rig.ChainEntry
		before bool
		want   []string
	}{
		{
			name:  "after the amplifier",
			chain: []rig.ChainEntry{amp, cab},
			want:  []string{"Ampeg SVT", "Ampeg 8x10", "Added"},
		},
		{
			name:   "before the amplifier",
			chain:  []rig.ChainEntry{amp, cab},
			before: true,
			want:   []string{"Added", "Ampeg SVT", "Ampeg 8x10"},
		},
		{
			// A chain of effects with no amplifier in it. Index zero, which is
			// what "before the amp" means when there is not one.
			name:   "before an amplifier that is not there",
			chain:  []rig.ChainEntry{cab},
			before: true,
			want:   []string{"Added", "Ampeg 8x10"},
		},
		{
			name:   "an empty chain",
			before: true,
			want:   []string{"Added"},
		},
	} {
		s.Run(tt.name, func() {
			got := placed(tt.chain, add, tt.before)

			names := make([]string, 0, len(got))
			for _, one := range got {
				names = append(names, one.Gear)
			}

			s.Require().Equal(tt.want, names)
		})
	}
}

// TestSide covers reading which end of the amplifier a kind belongs from the
// corpus rather than deciding it here.
func (s *ExtendTestSuite) TestSide() {
	bass, err := corpus.BuiltIn()
	s.Require().NoError(err)

	for _, tt := range []struct {
		name string
		g    corpus.Grammar
		of   catalog.Category
		want string
	}{
		{
			// 81% of pitch blocks in the corpus sit ahead of the amp, which is
			// where slot 05C's octaver is.
			name: "a pitch block",
			g:    bass.Grammar["bass"],
			of:   catalog.CategoryPitch,
			want: "before",
		},
		{
			// 72% of EQs sit after it.
			name: "an equaliser",
			g:    bass.Grammar["bass"],
			of:   catalog.CategoryEQ,
			want: "after",
		},
		{
			// A kind the corpus never counted goes after, which is the safer
			// end: a block ahead of an amplifier changes what it is fed.
			name: "a kind nothing has counted",
			g:    bass.Grammar["bass"],
			of:   catalog.Category("nonesuch"),
			want: "after",
		},
		{
			// An instrument with no grammar is an empty one, and answers the
			// same way.
			name: "an instrument nothing measured",
			of:   catalog.CategoryEQ,
			want: "after",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, side(tt.g, tt.of))
		})
	}
}

// TestExtend covers the document it writes, and the three ways it writes none.
func (s *ExtendTestSuite) TestExtend() {
	made := plan.Plan{Blocks: []plan.Block{{
		Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true,
	}}}

	// A target far enough out that something in the library moves it.
	aims := map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 9000, Tol: 1}}
	now := map[audio.Figure]float64{audio.KeyCentroid: 100}

	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The device, said apart from the sound.
			//
			// A chain that has filled the pedal has somewhere to go on a
			// bigger one, and a chain nothing measured can help does not, so
			// reporting the first as the second would send somebody looking
			// for a block instead of a Helix. Said before the rig is fetched,
			// which is why nothing is expected of the client.
			name: "a chain that has filled the device says so, not that nothing helps",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "extended.yaml")

				eight := plan.Plan{}
				for i := range 8 {
					eight.Blocks = append(eight.Blocks, plan.Block{
						Model: catalog.ModelID("HD2_CompressorDeluxeComp"),
						Pos:   i, Enabled: true,
					})
				}

				opts := TuneOptions{
					Client: s.pedal, ID: "matt-freeman", Genre: "punk", Extend: at,
				}

				w := buffer()

				s.Require().NoError(extend(context.Background(), w, opts,
					eight, aims, now, s.cat()))

				s.Require().Contains(w.String(), "8 blocks")
				s.Require().Contains(w.String(), "HX Stomp")
				s.Require().Contains(w.String(), "a different chain")
				s.Require().NotContains(w.String(), "nothing measured moves")

				_, err := os.Stat(at)
				s.Require().ErrorIs(err, os.ErrNotExist,
					"and nothing is written, because nothing was added")
			},
		},
		{
			// Off unless asked for, because a run that writes files nobody
			// asked for is a run somebody stops using.
			name: "no file named writes nothing and asks the client nothing",
			then: func() {
				opts := TuneOptions{Client: s.pedal, ID: "matt-freeman", Genre: "punk"}

				s.Require().NoError(extend(context.Background(), buffer(), opts,
					made, aims, now, s.cat()))
			},
		},
		{
			name: "the rig it aimed at, with the block added",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "extended.yaml")

				s.pedal.EXPECT().Rig(gomock.Any(), "matt-freeman").
					Return(sdk.Rig{Known: sdk.Known{
						ID: "matt-freeman",
						Rig: rig.Spec{
							Instrument: rig.InstrumentBass,
							Chain: []rig.ChainEntry{
								{Gear: "Ampeg SVT", Role: rig.RoleAmp},
								{Gear: "Ampeg 8x10", Role: rig.RoleCab},
							},
						},
					}}, nil)

				opts := TuneOptions{
					Client: s.pedal, ID: "matt-freeman", Genre: "punk", Extend: at,
				}

				w := buffer()

				s.Require().NoError(extend(context.Background(), w, opts,
					made, aims, now, s.cat()))

				said, err := os.ReadFile(at)
				s.Require().NoError(err)

				// The document says what it is, keeps the chain it had, and
				// carries the block with evidence that it was measured rather
				// than named by anybody.
				body := string(said)
				s.Require().Contains(body, "schema: ToneSpec")
				s.Require().Contains(body, "id: matt-freeman")
				s.Require().Contains(body, "Ampeg SVT")
				s.Require().Contains(body, "kind: corpus")
				s.Require().Contains(body, "the dials ran out")
				s.Require().Contains(w.String(), "Build it and tune it again")
			},
		},
		{
			// The embedded statistics are a compile-time constant and cannot be
			// damaged at run time, so a build that shipped a truncated archive
			// is the only thing that reaches this and the seam is the only way
			// to be that build. It refuses rather than carrying on with no
			// grammar, because every decision below it is the corpus's.
			name: "statistics that will not load",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "extended.yaml")

				was := grammarOf
				grammarOf = func() (*corpus.Stats, error) {
					return nil, errBadStatistics
				}

				defer func() { grammarOf = was }()

				opts := TuneOptions{
					Client: s.pedal, ID: "matt-freeman", Genre: "punk", Extend: at,
				}

				err := extend(context.Background(), buffer(), opts,
					made, aims, now, s.cat())
				s.Require().ErrorIs(err, errBadStatistics)
				s.Require().NoFileExists(at)
			},
		},
		{
			// A target already met has nothing missing, so there is no block
			// the measurement could ask for.
			name: "a chain that met its target",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "extended.yaml")

				opts := TuneOptions{
					Client: s.pedal, ID: "matt-freeman", Genre: "punk", Extend: at,
				}

				w := buffer()

				s.Require().NoError(extend(context.Background(), w, opts, made,
					aims, map[audio.Figure]float64{audio.KeyCentroid: 9000}, s.cat()))

				s.Require().Contains(w.String(), "no block to add")

				_, err := os.Stat(at)
				s.Require().Error(err, "and nothing was written")
			},
		},
		{
			name: "a client that will not read the rig",
			then: func() {
				gone := errors.New("that rig is not there")

				s.pedal.EXPECT().Rig(gomock.Any(), gomock.Any()).
					Return(sdk.Rig{}, gone)

				opts := TuneOptions{
					Client: s.pedal, ID: "matt-freeman", Genre: "punk",
					Extend: filepath.Join(s.T().TempDir(), "extended.yaml"),
				}

				s.Require().ErrorIs(extend(context.Background(), buffer(), opts,
					made, aims, now, s.cat()), gone)
			},
		},
		{
			name: "nowhere to write it",
			then: func() {
				flat := filepath.Join(s.T().TempDir(), "file")
				s.Require().NoError(os.WriteFile(flat, []byte("x"), 0o600))

				s.pedal.EXPECT().Rig(gomock.Any(), gomock.Any()).
					Return(sdk.Rig{Known: sdk.Known{
						ID: "matt-freeman",
						Rig: rig.Spec{Instrument: rig.InstrumentBass, Chain: []rig.ChainEntry{
							{Gear: "Ampeg SVT", Role: rig.RoleAmp},
						}},
					}}, nil)

				opts := TuneOptions{
					Client: s.pedal, ID: "matt-freeman", Genre: "punk",
					Extend: filepath.Join(flat, "under-a-file"),
				}

				s.Require().ErrorContains(extend(context.Background(), buffer(),
					opts, made, aims, now, s.cat()), "creating")
			},
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()
			tt.then()
		})
	}
}

func TestExtendTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ExtendTestSuite))
}

// TestRoom covers how many of a kind a chain may be given.
//
// The figures in the cases are the corpus's own, printed by `ranked`'s comment
// and measured within one series run. They move when records are added, so a
// case that starts failing is a signal to re-read them rather than to change
// the expectation.
func (s *ExtendTestSuite) TestRoom() {
	stats, err := corpus.BuiltIn()
	s.Require().NoError(err)

	bass := stats.Grammar["bass"]

	for _, tt := range []struct {
		name string
		g    corpus.Grammar
		runs map[catalog.Category]int
		of   catalog.Category
		want bool
	}{
		{
			// Nothing of that kind in the chain, so nothing to decide.
			name: "a kind the chain does not hold",
			g:    bass,
			runs: map[catalog.Category]int{},
			of:   catalog.CategoryDrive,
			want: true,
		},
		{
			// 19.6% of bass chains holding a drive hold two, which is the
			// shape 05C is built on and the one the first rule here refused.
			name: "a second drive on a bass chain",
			g:    bass,
			runs: map[catalog.Category]int{catalog.CategoryDrive: 1},
			of:   catalog.CategoryDrive,
			want: true,
		},
		{
			// 26.7%, the most repeated kind on a bass chain.
			name: "a second compressor on a bass chain",
			g:    bass,
			runs: map[catalog.Category]int{catalog.CategoryComp: 1},
			of:   catalog.CategoryComp,
			want: true,
		},
		{
			// `tonal` keeps the amplifier out of the grammar, so Repeats reads
			// zero for it. This is the EVH 5150III appended after an SVT.
			name: "a second amplifier",
			g:    bass,
			runs: map[catalog.Category]int{catalog.CategoryAmp: 1},
			of:   catalog.CategoryAmp,
			want: false,
		},
		{
			// 0.0% for bass. Two cabinets in series is not the blend people
			// build; that one is either side of a split.
			name: "a second cabinet",
			g:    bass,
			runs: map[catalog.Category]int{catalog.CategoryCab: 1},
			of:   catalog.CategoryCab,
			want: false,
		},
		{
			// Two is the most the evidence supports, because Repeats counts
			// chains holding two or more without separating two from three.
			name: "a third drive",
			g:    bass,
			runs: map[catalog.Category]int{catalog.CategoryDrive: 2},
			of:   catalog.CategoryDrive,
			want: false,
		},
		{
			// An instrument nothing measured is an empty grammar: no evidence,
			// no repeat.
			name: "an instrument nothing measured",
			runs: map[catalog.Category]int{catalog.CategoryDrive: 1},
			of:   catalog.CategoryDrive,
			want: false,
		},
		{
			// A kind the corpus never counted answers the same way.
			name: "a kind nothing has counted",
			g:    bass,
			runs: map[catalog.Category]int{catalog.Category("nonesuch"): 1},
			of:   catalog.Category("nonesuch"),
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, room(tt.g, tt.runs, tt.of))
		})
	}
}

// TestInSeries covers counting what the chain already holds.
//
// Per run, because that is how the corpus counts: two of a kind either side of
// a parallel split is not two in series, and reading it as such put bass cab at
// 16.3% where the real figure is zero.
// TestFits covers fits, which asks the device rather than the corpus.
//
// Against the built-in catalog's real costs, because the point of the check is
// that the figures are Line 6's and not this tool's.
func (s *ExtendTestSuite) TestFits() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	// 26.6% of one processor each, officially, so three fit under the 95%
	// ceiling an HX Stomp has and four do not.
	svt := catalog.ModelID("HD2_AmpSVBeastBrt")

	full := func(n int) plan.Plan {
		made := plan.Plan{}
		for i := range n {
			made.Blocks = append(made.Blocks,
				plan.Block{Model: svt, Pos: i, DSP: 0})
		}

		return made
	}

	for _, tt := range []struct {
		name string
		made plan.Plan
		add  catalog.ModelID
		want bool
	}{
		{
			// Room on both counts.
			name: "an empty chain holds anything the catalog carries",
			made: plan.Plan{},
			add:  svt,
			want: true,
		},
		{
			// The block count. An HX Stomp holds 8, so a chain of 8 holds no
			// more however cheap the next one is.
			name: "a chain at the device's block count holds nothing further",
			made: full(8),
			add:  "HD2_CompressorDeluxeComp",
			want: false,
		},
		{
			// The processor budget, which bites first here. Three SVTs are
			// 79.8% and a fourth would be 106.4% against a ceiling of 95, on a
			// device with room for eight blocks.
			name: "a chain inside the block count can still be over budget",
			made: full(3),
			add:  svt,
			want: false,
		},
		{
			// And two of them leave room for a third.
			name: "two of them leave room for one more",
			made: full(2),
			add:  svt,
			want: true,
		},
		{
			// A model the catalog does not carry has no cost to add, and a
			// guess is not a cost: plan refuses an assumed one at build time,
			// so proposing it here would only move the refusal later.
			name: "a model the catalog does not carry does not fit",
			made: plan.Plan{},
			add:  "HD2_NoSuchBlock",
			want: false,
		},
		{
			// The same, one step back. A chain holding something the catalog
			// cannot price has no budget to compare against, and leaving it
			// out of the sum would answer "it fits" about a chain nobody can
			// add up.
			name: "a chain holding a block the catalog cannot price fits nothing",
			made: plan.Plan{Blocks: []plan.Block{
				{Model: "HD2_NoSuchBlock", Pos: 0},
			}},
			add:  "HD2_CompressorDeluxeComp",
			want: false,
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, fits(tt.made, cat, tt.add))
		})
	}
}

func (s *ExtendTestSuite) TestInSeries() {
	drive := catalog.ModelID("HD2_DistObsidian7000")
	amp := catalog.ModelID("HD2_AmpSVBeastBrt")

	for _, tt := range []struct {
		name   string
		blocks []plan.Block
		of     catalog.Category
		want   int
	}{
		{
			name:   "an empty chain",
			blocks: nil,
			of:     catalog.CategoryDrive,
			want:   0,
		},
		{
			name: "two drives in one run",
			blocks: []plan.Block{
				{Model: drive, Pos: 0, Enabled: true},
				{Model: amp, Pos: 1, Enabled: true},
				{Model: drive, Pos: 2, Enabled: true},
			},
			of:   catalog.CategoryDrive,
			want: 2,
		},
		{
			name: "two drives on different processors",
			blocks: []plan.Block{
				{Model: drive, DSP: 0, Pos: 0, Enabled: true},
				{Model: drive, DSP: 1, Pos: 0, Enabled: true},
			},
			of:   catalog.CategoryDrive,
			want: 1,
		},
		{
			// A bypassed block still occupies its position, and a chain that
			// is given a second one still ends up holding two.
			name: "a bypassed drive still counts",
			blocks: []plan.Block{
				{Model: drive, Pos: 0, Enabled: false},
				{Model: drive, Pos: 1, Enabled: true},
			},
			of:   catalog.CategoryDrive,
			want: 2,
		},
		{
			// A model the catalog does not carry cannot be categorised, so it
			// cannot be counted.
			name: "a model the catalog lacks",
			blocks: []plan.Block{
				{Model: catalog.ModelID("HD2_Nonesuch"), Pos: 0, Enabled: true},
			},
			of:   catalog.CategoryDrive,
			want: 0,
		},
	} {
		s.Run(tt.name, func() {
			got := inSeries(plan.Plan{Blocks: tt.blocks}, s.cat())
			s.Require().Equal(tt.want, got[tt.of])
		})
	}
}

// TestRanked covers which block the measurement asks for, and when it asks for
// none.
func (s *ExtendTestSuite) TestRanked() {
	stats, err := corpus.BuiltIn()
	s.Require().NoError(err)

	// A target far enough out that something in the library moves it. The
	// shortlist for it is five blocks across three kinds: amp, drive and mod.
	aims := map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 9000, Tol: 1}}
	now := map[audio.Figure]float64{audio.KeyCentroid: 100}
	met := map[audio.Figure]float64{audio.KeyCentroid: 9000}

	amp := plan.Block{Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true}
	drive := plan.Block{Model: catalog.ModelID("HD2_DistObsidian7000"), Pos: 1, Enabled: true}
	mod := plan.Block{Model: catalog.ModelID("HD2_Chorus"), Pos: 2, Enabled: true}

	for _, tt := range []struct {
		name   string
		blocks []plan.Block
		g      corpus.Grammar
		now    map[audio.Figure]float64
		want   bool
	}{
		{
			// An amplifier on its own: the shortlist offers a drive the chain
			// has none of, so there is something to ask for.
			name:   "a chain with room for what was ranked",
			blocks: []plan.Block{amp},
			g:      stats.Grammar["bass"],
			now:    now,
			want:   true,
		},
		{
			// Every axis inside tolerance, so nothing is missing and the
			// shortlist is never consulted.
			name:   "a chain that met its target",
			blocks: []plan.Block{amp},
			g:      stats.Grammar["bass"],
			now:    met,
			want:   false,
		},
		{
			// The shortlist's five candidates are all amps, drives or mods.
			// A chain holding an amp, a drive and a chorus, with no grammar
			// to license a second of any kind, refuses all five and falls
			// through. That is the honest answer rather than a block added on
			// no evidence.
			name:   "a chain with room for none of them",
			blocks: []plan.Block{amp, drive, mod},
			now:    now,
			want:   false,
		},
	} {
		s.Run(tt.name, func() {
			_, ok := ranked(
				plan.Plan{Blocks: tt.blocks}, aims, tt.now, s.cat(), tt.g)
			s.Require().Equal(tt.want, ok)
		})
	}
}

// TestParam covers turning a swept position into a value of its own kind.
func (s *ExtendTestSuite) TestParam() {
	for _, tt := range []struct {
		name  string
		of    catalog.ModelID
		at    measured.Starting
		want  string
		known bool
	}{
		{
			// A float stays a float. A document read through a route that
			// turns 6.0 into 6 is a document the device reads differently.
			name:  "a dial",
			of:    catalog.ModelID("HD2_EQSimple3Band"),
			at:    measured.Starting{Control: "LowGain", Value: -6},
			want:  "-6.0",
			known: true,
		},
		{
			// An integer control, written without a point for the same reason.
			name:  "a control counted in whole numbers",
			of:    catalog.ModelID("HD2_CabMicIr_2x15Brute"),
			at:    measured.Starting{Control: "Mic", Value: 3},
			want:  "3",
			known: true,
		},
		{
			// A switch is a list of two, and a swept position is a number
			// along it. Which number means which setting is the list's own
			// business; a sweep moved an index and read what came back, and
			// guessing the name would write a setting nobody measured.
			name:  "a control that is a switch",
			of:    catalog.ModelID("HD2_AmpAgua51"),
			at:    measured.Starting{Control: "Bright", Value: 1},
			known: false,
		},
		{
			// A sweep and a catalog that disagree. Dropped rather than
			// written, because a value the catalog would refuse is a document
			// that will not build.
			name:  "a control the catalog does not carry",
			of:    catalog.ModelID("HD2_EQSimple3Band"),
			at:    measured.Starting{Control: "Nonesuch", Value: 1},
			known: false,
		},
		{
			name:  "a model the catalog does not carry",
			of:    catalog.ModelID("HD2_Nonesuch"),
			at:    measured.Starting{Control: "LowGain", Value: 1},
			known: false,
		},
	} {
		s.Run(tt.name, func() {
			got, known := param(s.cat(), tt.of, tt.at)

			s.Require().Equal(tt.known, known)

			if !tt.known {
				return
			}

			s.Require().Equal(tt.want, got.String())
		})
	}
}

// TestStartAtReads covers reading a proposed block's sweep off disk.
//
// Every answer is reported, including every way there is no answer. "Its
// controls are at the catalog's defaults" is as much a fact about the document
// somebody is about to read as a value would be, and the message names the
// command that would change it.
func (s *ExtendTestSuite) TestStartAtReads() {
	aims := map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 9000, Tol: 1}}
	now := map[audio.Figure]float64{audio.KeyCentroid: 100}

	swept := measured.Suggestion{Block: measured.Block{
		ID: "HD2_EQSimple3Band", Name: "Simple EQ", Category: catalog.CategoryEQ,
	}}
	bare := measured.Suggestion{Block: measured.Block{
		ID: "HD2_DistMegaphone", Name: "Megaphone", Category: catalog.CategoryDrive,
	}}

	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The sweep has something to say, so the document says it.
			name: "a block with a committed sweep",
			then: func() {
				w := buffer()

				got, set := startAt(w,
					TuneOptions{Sweeps: filepath.Join("testdata", "sweeps")},
					swept, aims, now)

				s.Require().True(set)
				s.Require().NotEmpty(got.Control)
				s.Require().Contains(w.String(), "Simple EQ arrives with")
				s.Require().Contains(w.String(), "times")
				s.Require().Contains(w.String(),
					"starting position", "it must not read as a setting")
			},
		},
		{
			// The common answer, and the one worth reporting well: 19 of the
			// device's 661 blocks have been swept, and none of them is a block
			// the shortlist picks for a chain that already holds an amplifier
			// and a cabinet.
			name: "a block nobody has swept",
			then: func() {
				w := buffer()

				_, set := startAt(w,
					TuneOptions{Sweeps: filepath.Join("testdata", "sweeps")},
					bare, aims, now)

				s.Require().False(set)
				s.Require().Contains(w.String(), "has no committed sweep")
				s.Require().Contains(w.String(),
					"measure controls --model HD2_DistMegaphone",
					"the message has to name what would change it")
			},
		},
		{
			// Only the fingerprints are packed into the binary, so a release
			// has no curves at all and nothing here pretends otherwise.
			name: "no sweeps to read",
			then: func() {
				w := buffer()

				_, set := startAt(w, TuneOptions{}, swept, aims, now)

				s.Require().False(set)
				s.Require().Empty(w.String(),
					"a run that was never given a sweep tree has nothing to report")
			},
		},
		{
			name: "a sweep that will not parse",
			then: func() {
				dir := s.T().TempDir()
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "HD2_EQSimple3Band.json"),
					[]byte("{not json"), 0o600))

				w := buffer()

				_, set := startAt(w, TuneOptions{Sweeps: dir}, swept, aims, now)

				s.Require().False(set)
				s.Require().Contains(w.String(), "arrive at the catalog's defaults")
			},
		},
		{
			// Swept, and with nothing worth writing down. The fixture is a
			// real sweep whose own noise figures are wider than anything its
			// controls do, so every move it reports is inside the
			// measurement's wobble.
			name: "a sweep that says nothing convincingly",
			then: func() {
				w := buffer()

				quiet := measured.Suggestion{Block: measured.Block{
					ID:       "HD2_EQSimpleTilt",
					Name:     "Simple Tilt",
					Category: catalog.CategoryEQ,
				}}

				_, set := startAt(w,
					TuneOptions{Sweeps: filepath.Join("testdata", "sweeps")},
					quiet, aims, now)

				s.Require().False(set)
				s.Require().Contains(w.String(), "is swept, and no position")
			},
		},
	} {
		s.Run(tt.name, tt.then)
	}
}

// TestAmendWritesTheControl covers the control landing in the document.
func (s *ExtendTestSuite) TestAmendWritesTheControl() {
	doc := sdk.Rig{Known: sdk.Known{
		ID: "matt-freeman",
		Rig: rig.Spec{
			Instrument: rig.InstrumentBass,
			Chain: []rig.ChainEntry{
				{Gear: "Ampeg SVT", Role: rig.RoleAmp},
			},
		},
	}}
	pick := measured.Suggestion{Block: measured.Block{
		ID: "HD2_EQSimple3Band", Name: "Simple EQ", Category: catalog.CategoryEQ,
	}}

	for _, tt := range []struct {
		name string
		at   measured.Starting
		set  bool
		want map[string]string
	}{
		{
			// Partial on purpose, which the contract's own field allows: the
			// one control the sweep argued for, and the rest answered from
			// corpus medians the way every other build answers them.
			name: "a position the sweep argued for",
			at:   measured.Starting{Control: "LowGain", Value: -6},
			set:  true,
			want: map[string]string{"LowGain": "-6.0"},
		},
		{
			// No sweep, no controls, and the entry carries none rather than
			// carrying defaults written out as though they were chosen.
			name: "nothing to say",
			set:  false,
		},
		{
			// A sweep and a catalog that disagree. The block is still added,
			// because the block choice did not depend on the control.
			name: "a control the catalog does not carry",
			at:   measured.Starting{Control: "Nonesuch", Value: 1},
			set:  true,
		},
	} {
		s.Run(tt.name, func() {
			got, err := amend(doc, pick, s.cat(), TuneOptions{ID: "matt-freeman"},
				corpus.Grammar{}, tt.at, tt.set)
			s.Require().NoError(err)

			var held *map[string]catalog.Setting

			for _, one := range got.Rig.Chain {
				if one.Gear != "Ampeg SVT" {
					held = one.Controls
				}
			}

			if tt.want == nil {
				s.Require().Nil(held, "no control was argued for, so none is written")

				return
			}

			s.Require().NotNil(held)

			for name, value := range tt.want {
				at, carried := (*held)[name]
				s.Require().True(carried)
				s.Require().Equal(value, at.String())
			}
		})
	}
}
