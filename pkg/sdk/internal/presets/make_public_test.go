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

package presets_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/internal/presets"
	presetmocks "github.com/retr0h/toneharness/pkg/sdk/internal/presets/mocks"
	"github.com/retr0h/toneharness/pkg/sdk/internal/rigs"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

type MakePublicTestSuite struct {
	suite.Suite
}

// catalogs hands over the catalog at path, however often it is asked.
func (s *MakePublicTestSuite) catalogs(
	path string,
) *presetmocks.MockCatalogs {
	c := presetmocks.NewMockCatalogs(gomock.NewController(s.T()))
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) { return catalog.Open(path) },
	).AnyTimes()

	return c
}

func (s *MakePublicTestSuite) opts(
	id string,
	out string,
) presets.MakeOptions {
	return presets.MakeOptions{
		Deps:       presets.Deps{Catalogs: s.catalogs(filepath.Join("testdata", "catalog.json"))},
		RigID:      id,
		Source:     rigs.Source{Dir: filepath.Join("testdata", "rigs")},
		OutputPath: out,
	}
}

// lib is a reading for the cabinet this fixture rig resolves to.
//
// Its own rather than the shipped library, because `HD2_CabNamed` is a model
// this test invented and the shipped readings know nothing about it. The figure
// is the real `2x15 Brute`'s, which is the quiet half of the pair that was
// found by playing two rigs and hearing one as quiet.
func (s *MakePublicTestSuite) lib() presets.Libraries {
	return readings{at: measured.Library{Blocks: map[string]measured.Block{
		"HD2_CabNamed": {
			ID:       "HD2_CabNamed",
			Category: catalog.CategoryCab,
			Figures:  measured.Figures{Level: -59.62},
		},
		"HD2_Cab8x10SVBeast": {
			ID:       "HD2_Cab8x10SVBeast",
			Category: catalog.CategoryCab,
			Figures:  measured.Figures{Level: -56.76},
		},
	}}}
}

// thin is readings for this chain's cabinet and not for the matched rig's.
//
// So the match has nothing to aim at while the chain it is matching has a
// figure of its own, which is the one case that reaches ErrNoLoudness.
func (s *MakePublicTestSuite) thin() presets.Libraries {
	return readings{at: measured.Library{Blocks: map[string]measured.Block{
		"HD2_CabNamed": {
			ID:       "HD2_CabNamed",
			Category: catalog.CategoryCab,
			Figures:  measured.Figures{Level: -59.62},
		},
	}}}
}

// readings hands over block readings, or refuses to.
//
// Hand-written rather than generated because it is two fields and no
// expectations: what the rows need is readings of their own and one that will
// not open, and a mock with call assertions would say more about the test than
// about the build.
type readings struct {
	at  measured.Library
	err error
}

func (r readings) Library() (measured.Library, error) { return r.at, r.err }

// errNoReadings stands in for a library that will not open.
var errNoReadings = errors.New("no readings")

// TestMake builds a preset out of a rig.
func (s *MakePublicTestSuite) TestMake() {
	tests := []struct {
		name string
		ctx  context.Context
		id   string
		// rigPath names the document by file instead of by identifier, relative
		// to the rigs this suite's testdata holds.
		rigPath string
		// bothSources names a rig twice and noSource names it not at all,
		// which are the two ways to ask for a rig nobody identified.
		bothSources bool
		noSource    bool
		// alone copies the named document into a directory of its own, and
		// aloneAs renames it on the way, for a row about what a path may be
		// spelt as.
		alone   string
		aloneAs string
		stats   string
		catalog string
		// setup is what the person has. The row writes it to a file, so a
		// path that is not there is spelt as a path rather than a document.
		setup   string
		noSetup string
		// playing is a fragment the compensation has to say. Empty asks for
		// none, which is a build that made none.
		playing string
		// sweeps is the tree of committed curves, which say what a turn of an
		// amplifier's level control is worth. Empty reads none, and a stated
		// level is reported rather than reached.
		sweeps string
		// level is what the level report has to say: the control it moved, or
		// a fragment of why it moved nothing. Empty is a build given no
		// target, which must say nothing about one.
		level string
		// brokenSweep writes a curve that will not parse, where the row is
		// about a sweep this does not have rather than one it lacks.
		brokenSweep bool
		// ownLibrary says whether the row supplies readings of its own. Rows
		// about a level want them, because the blocks this fixture names are
		// invented and the shipped readings know nothing about them.
		ownLibrary bool
		// badLibrary supplies readings that will not open, for the row about a
		// build that cannot measure what it was asked to match.
		badLibrary bool
		// thinLibrary supplies readings that hold this chain's cabinet and
		// not the matched rig's, for the row about a rig there is no figure
		// for.
		thinLibrary bool
		// wroteOver is a control the document states over a word, and held
		// what the preset must end up holding for it. Empty asserts that no
		// word was written over.
		wroteOver string
		held      float64
		// named is what the preset should be called, where the row builds a
		// rig whose subject is not the usual one.
		named string
		// blocks is how many the chain should hold. Zero means the two a rig
		// names, which is what most rows build.
		blocks   int
		out      string
		contains []string
		// absent is what the answer must not say.
		absent   []string
		loadable bool
		written  []string
		err      error
		errText  string
	}{
		{
			name:     "a rig that builds",
			id:       "test-player",
			loadable: true,
			contains: []string{"Test Player"},
		},
		{
			// The same rig by path rather than by identifier, and the words in its
			// ask still reach a control. The point of the whole flag: a document
			// that `tone build` wrote has no identifier to look up, and compiling
			// it instead dropped every word with nothing said.
			name:     "a document by path",
			rigPath:  "own-words.yaml",
			loadable: true,
			contains: []string{"sounds like a wet paper bag"},
		},
		{
			// A path with no extension at all, which nothing writes and a person
			// may still type. One document, so nothing is looked for beside it
			// and the spelling decides nothing.
			name:     "a document whose path has no extension",
			alone:    "own-words.yaml",
			aloneAs:  "own-words",
			named:    "Test Player",
			loadable: true,
			contains: []string{"sounds like a wet paper bag"},
		},
		{
			// A document with no ask is legal and ordinary. Somebody's own
			// directory holds rigs they wrote, and nothing obliges them to write
			// down the request that produced one.
			name:    "a document with no ask",
			rigPath: "gear-only.yaml",
			// The document's own identifier, because the name comes off the ask's
			// subject and there is no subject without one.
			named:    "gear-only",
			loadable: true,
			// The words were in the ask, and there is none, so nothing describes
			// this chain in anybody's words.
			absent: []string{"sounds like a wet paper bag"},
		},
		{
			name:        "a rig named twice",
			bothSources: true,
			err:         presets.ErrOneRig,
		},
		{
			name:     "a rig named no way at all",
			noSource: true,
			err:      presets.ErrOneRig,
		},
		{
			name:    "a rig file that is not there",
			rigPath: "no-such-rig.yaml",
			errText: "no-such-rig.yaml",
		},
		{
			// A word nothing defines is said and not refused. Nothing
			// compiles a character term into a chain, so the preset is
			// written and the note tells whoever wrote it.
			name:     "a rig describing itself in its own words",
			id:       "own-words",
			loadable: true,
			contains: []string{"sounds like a wet paper bag"},
		},
		{
			// A rig names an amp; a rig is several blocks. Whatever the
			// corpus contributed has to be visible before anybody plugs in.
			name:     "what the corpus added unasked",
			id:       "test-player",
			stats:    filepath.Join("testdata", "stats.json.gz"),
			contains: []string{"Minotaur"},
		},
		{
			// Statistics improve a preset and are not needed to produce one,
			// but somebody who named a file asked for those. A preset built
			// without them would be quietly more generic than the one asked
			// for.
			name:    "statistics named and not there",
			id:      "test-player",
			stats:   filepath.Join("testdata", "no-such-stats.gz"),
			errText: "no-such-stats.gz",
		},
		{
			// A song's sections become the preset's snapshots, named.
			name:     "a rig in song sections",
			id:       "in-sections",
			loadable: true,
			written:  []string{`"@name":"Verse"`, `"@name":"Chorus"`},
		},
		{
			// Checked against the chain as built, so a section cannot turn
			// on a pedal the preset does not hold.
			name: "a section naming gear the chain does not hold",
			id:   "wrong-section",
			err:  compile.ErrNoSuchValue,
		},
		{
			name: "a rig nobody has",
			id:   "nobody",
			err:  rigs.ErrNotFound,
		},
		{
			name:    "a catalog it cannot read",
			id:      "test-player",
			catalog: filepath.Join("testdata", "nope.json"),
			errText: "catalog",
		},
		{
			name: "gear the catalog does not model",
			id:   "unbuildable",
			err:  compile.ErrNoSuchGear,
		},
		{
			// Seven heavy pedals plus an amp and a cabinet exceeds what the
			// device can hold, and a preset nobody can load is not a preset.
			name:    "a chain that will not load",
			id:      "too-big",
			errText: "will not load",
		},
		{
			// The same chain on a device with two processors, which has
			// somewhere to put what does not fit on the first. What a build
			// will hold is the catalog's device, not whichever one this was
			// written against.
			name:     "a chain that fits a bigger device",
			id:       "two-paths",
			catalog:  filepath.Join("testdata", "catalog-floor.json"),
			written:  []string{`"dsp1"`},
			loadable: false,
		},
		{
			name:    "nowhere to write the preset",
			id:      "test-player",
			out:     filepath.Join("no", "such", "dir.hlx"),
			errText: "writing",
		},
		{
			// The report has to say what the preset holds. The ordering is
			// deliberate, words first and a stated value beating one, and the
			// report printed the word's step as though it were the outcome:
			// "Drive 0.60 to 0.72" against an 0.88 in the file. Somebody
			// hand-editing a document reads that line to check their edit
			// took, and it told them it had not.
			name:      "a word the document wrote over",
			id:        "stated-over-word",
			named:     "Stated Over Word",
			loadable:  true,
			wroteOver: "Drive",
			held:      0.88,
		},
		{
			// The other pass that writes over a word. `settings` is seven
			// musical words the compiler turns into values, `controls` is the
			// values outright, and they reach the chain through different
			// code. Both have to report what the preset holds.
			name:      "a word a setting wrote over",
			id:        "setting-over-word",
			named:     "Setting Over Word",
			loadable:  true,
			wroteOver: "Drive",
			held:      0.2,
		},
		{
			// The gap that was found by playing two rigs: one read as quiet,
			// and the cabinet alone accounted for 2.9dB of it. With the curve
			// in hand the build closes it.
			name:       "a rig it was asked to come out as loud as",
			id:         "test-player",
			setup:      "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: louder-rig\n",
			sweeps:     filepath.Join("testdata", "sweeps"),
			level:      "ChVol",
			ownLibrary: true,
			loadable:   true,
		},
		{
			// Only the fingerprints are packed into the binary, so a build
			// from a release has no curves. It says so rather than guessing
			// what a turn is worth: the five amplifiers swept disagree by a
			// factor of two about the same control.
			name:       "a loudness with no curves to match it with",
			id:         "test-player",
			setup:      "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: louder-rig\n",
			level:      "no committed sweep",
			ownLibrary: true,
			loadable:   true,
		},
		{
			// Two rigs ending in the same cabinet. Within a decibel is under
			// what anybody reliably hears on a broadband signal, so nothing
			// moves: reporting a control turned a third of a decibel spends
			// somebody's attention on nothing.
			name:       "a rig that ends in the same cabinet",
			id:         "test-player",
			setup:      "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: gear-only\n",
			sweeps:     filepath.Join("testdata", "sweeps"),
			level:      "already within",
			ownLibrary: true,
			loadable:   true,
		},
		{
			// A sweep that will not parse is a sweep this does not have, which
			// is the same answer as not having one. Refusing a whole build
			// over an unreadable curve would be refusing somebody a preset
			// over a file they never asked it to read.
			name:        "a sweep that will not parse",
			id:          "test-player",
			setup:       "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: louder-rig\n",
			brokenSweep: true,
			level:       "no committed sweep",
			ownLibrary:  true,
			loadable:    true,
		},
		{
			// A sweeps tree with nothing in it for this amplifier, which is
			// the same answer as no tree at all and reached by a different
			// path: one file that is not there rather than a directory that
			// is not named.
			name:       "a sweeps tree with no curve for this amplifier",
			id:         "test-player",
			setup:      "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: louder-rig\n",
			sweeps:     filepath.Join("testdata", "rigs"),
			ownLibrary: true,
			level:      "no committed sweep",
			loadable:   true,
		},
		{
			// A rig the Setup names and nothing has. Refused, because the
			// Setup asked for something this cannot do and a build that
			// quietly came out at its own loudness is a build somebody
			// compares against another one.
			name:       "a rig to match that does not exist",
			id:         "test-player",
			setup:      "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: no-such-rig\n",
			ownLibrary: true,
			errText:    "as loud as \"no-such-rig\"",
		},
		{
			// Naming the rig being built. Nothing happens and nothing is
			// reported, which is the right answer rather than an error: a
			// chain is already as loud as itself.
			name:     "the rig being built",
			id:       "test-player",
			setup:    "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: test-player\n",
			sweeps:   filepath.Join("testdata", "sweeps"),
			loadable: true,
		},
		{
			// A rig whose own chain has no cabinet anybody measured, so there
			// is no figure to match. Refused for the same reason.
			name:        "a rig with no measured cabinet",
			id:          "test-player",
			setup:       "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: louder-rig\n",
			thinLibrary: true,
			sweeps:      filepath.Join("testdata", "sweeps"),
			err:         presets.ErrNoLoudness,
		},
		{
			// A rig named as a loudness whose own chain will not realise.
			// Refused and named, because the Setup asked to match something
			// this cannot work out.
			name:       "a rig to match that will not build",
			id:         "test-player",
			setup:      "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: unbuildable\n",
			ownLibrary: true,
			errText:    "which will not build",
		},
		{
			// Nil readings read the ones packed into this binary, which is
			// what every build outside a test does. `HD2_CabNamed` is a model
			// this fixture invented, so the shipped readings have no figure
			// for the chain being built.
			name:     "the readings packed into the binary",
			id:       "test-player",
			setup:    "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: louder-rig\n",
			sweeps:   filepath.Join("testdata", "sweeps"),
			level:    "nothing in the chain has a measured level",
			loadable: true,
		},
		{
			// Readings that will not open. Refused rather than levelled
			// against nothing, because every figure below them is theirs.
			name:       "readings that will not open",
			id:         "test-player",
			setup:      "schema: Setup\nversion: \"4.0.0\"\nmatch_loudness: louder-rig\n",
			badLibrary: true,
			err:        errNoReadings,
		},
		{
			// Every figure in the corpus came off somebody else's playing, so
			// a rig played with a pick is duller in the hands of somebody who
			// uses fingers. With both sides stated the difference is a knob
			// rather than a surprise.
			name:     "a Setup saying how this person plays",
			id:       "picked-player",
			setup:    "schema: Setup\nversion: \"4.0.0\"\ntechnique:\n  attack: fingers\n",
			loadable: true,
			playing:  "you play with fingers",
			named:    "Picked Player",
		},
		{
			// Building for the record is what this did before a Setup could
			// be named, and still the answer for somebody without one.
			name:     "no Setup at all builds for the record",
			id:       "test-player",
			loadable: true,
		},
		{
			name:    "a Setup named and not there",
			id:      "test-player",
			noSetup: "no-such-setup.yaml",
			errText: "no-such-setup.yaml",
		},
		{
			name:    "a Setup that is not a Setup",
			id:      "test-player",
			setup:   "schema: ToneSpec\nversion: \"4.0.0\"\n",
			errText: "reading",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelledContext(),
			id:      "test-player",
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "test.hlx")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			o := s.opts(tt.id, out)

			if tt.rigPath != "" {
				o.RigID = ""
				o.RigPath = filepath.Join("testdata", "rigs", "artists", tt.rigPath)
			}

			if tt.alone != "" {
				as := tt.alone
				if tt.aloneAs != "" {
					as = tt.aloneAs
				}

				at := filepath.Join(dir, as)
				from, readErr := os.ReadFile(
					filepath.Join("testdata", "rigs", "artists", tt.alone))
				s.Require().NoError(readErr)
				s.Require().NoError(os.WriteFile(at, from, 0o600))

				o.RigID = ""
				o.RigPath = at
			}

			if tt.bothSources {
				o.RigID = "test-player"
				o.RigPath = filepath.Join("testdata", "rigs", "artists", "test-player.yaml")
			}

			if tt.noSource {
				o.RigID = ""
				o.RigPath = ""
			}

			if tt.stats != "" {
				o.StatsPath = tt.stats
			}

			if tt.catalog != "" {
				o.Catalogs = s.catalogs(tt.catalog)
			}

			o.SweepsDir = tt.sweeps

			if tt.brokenSweep {
				at := filepath.Join(dir, "sweeps")
				s.Require().NoError(os.MkdirAll(at, 0o750))
				s.Require().NoError(os.WriteFile(
					filepath.Join(at, "HD2_AmpSVBeastBrt.json"),
					[]byte("{not json"), 0o600))

				o.SweepsDir = at
			}

			switch {
			case tt.badLibrary:
				o.Libraries = readings{err: errNoReadings}
			case tt.thinLibrary:
				o.Libraries = s.thin()
			case tt.ownLibrary:
				o.Libraries = s.lib()
			}

			switch {
			case tt.setup != "":
				at := filepath.Join(dir, "setup.yaml")
				s.Require().NoError(os.WriteFile(at, []byte(tt.setup), 0o600))

				o.SetupPath = at
			case tt.noSetup != "":
				o.SetupPath = filepath.Join(dir, tt.noSetup)
			}

			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			made, err := presets.Make(ctx, o)

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			got := built(made)
			s.Require().Equal(out, made.Path)

			for _, want := range tt.contains {
				s.Require().Contains(got, want)
			}

			for _, not := range tt.absent {
				s.Require().NotContains(got, not)
			}

			// What the preset ends up holding, where the document states a
			// control a word also moves. Read off the plan as well as the
			// report, because the two disagreeing is the whole of what this
			// row is about.
			if tt.wroteOver != "" {
				var beaten result.Moved

				for _, m := range made.Moved {
					if m.Beaten() && m.Param == tt.wroteOver {
						beaten = m
					}
				}

				s.Require().NotEmpty(beaten.Term,
					"a word reached %s and the document states it", tt.wroteOver)
				s.Require().InDelta(tt.held, beaten.Held, 1e-9)
				s.Require().NotEqual(beaten.To, beaten.Held,
					"the word's own move is not what the preset holds")

				held, ok := made.Plan.Blocks[beaten.At].Params[tt.wroteOver].Float()
				s.Require().True(ok)
				s.Require().InDelta(tt.held, held,
					1e-9, "and the report agrees with the plan")
			}

			// A row naming a control asserts that control moved; a row
			// naming anything else asserts the reason nothing did. Decided by
			// the row rather than by what came back, because a refusal naming
			// the control it could not turn would otherwise satisfy a row that
			// asked for the turn: "no committed sweep, so what a turn of its
			// ChVol is worth is unmeasured" contains ChVol.
			switch tt.level {
			case "":
				s.Require().False(made.Levelled.Asked(),
					"no rig to match, so nothing is reported")
			case "ChVol":
				s.Require().Equal(tt.level, made.Levelled.Param)
				s.Require().NotEqual(made.Levelled.From, made.Levelled.To,
					"a control named as moved has to have moved")
				s.Require().Empty(made.Levelled.Because)
			default:
				s.Require().Empty(made.Levelled.Param,
					"nothing moved, so no control is named")
				s.Require().Contains(made.Levelled.Because, tt.level)
			}

			if tt.playing == "" {
				s.Require().Empty(made.Playing.Said,
					"nothing was compensated, so there is nothing to say")
			} else {
				s.Require().Contains(made.Playing.Said, tt.playing)
				s.Require().NotEmpty(made.Playing.Terms,
					"a word was added, and the report names which")
			}

			if len(tt.written) > 0 {
				body, err := os.ReadFile(filepath.Clean(out))
				s.Require().NoError(err)

				compact := strings.Join(strings.Fields(string(body)), "")
				for _, want := range tt.written {
					s.Require().Contains(compact, want)
				}
			}

			if !tt.loadable {
				return
			}

			s.Require().NoError(func() error {
				f, err := os.Open(out) //nolint:gosec // a path this test chose
				if err != nil {
					return err
				}

				defer func() { s.Require().NoError(f.Close()) }()

				doc, err := preset.Read(f)
				if err != nil {
					return err
				}

				s.Require().Equal(2162694, doc.Data.Device)

				named := tt.named
				if named == "" {
					named = "Test Player"
				}

				s.Require().Equal(named, doc.Data.Meta.Name)

				spec, err := doc.Spec()
				if err != nil {
					return err
				}

				blocks := tt.blocks
				if blocks == 0 {
					blocks = 2
				}

				s.Require().Len(spec.Blocks, blocks)

				return nil
			}())
		})
	}
}

// cancelledContext is a context whose caller has already stopped waiting.
func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

// built flattens what a build reported, so a test can assert on the facts of
// it without also asserting on how a terminal paints them.
func built(
	m result.Made,
) string {
	parts := make([]string, 0, 2+2*len(m.Added)+len(m.Unfamiliar))
	parts = append(parts, m.Plan.Name, m.Path)

	for _, a := range m.Added {
		parts = append(parts, a.Name, a.Reason)
	}

	for _, u := range m.Unfamiliar {
		parts = append(parts, u.Term)
	}

	return strings.Join(parts, " ")
}

// IntentPublicTestSuite covers what an ask contributes to a build.
type IntentPublicTestSuite struct {
	suite.Suite
}

// TestIntentOf covers every field an ask may leave out.
//
// Each one is a pointer, so the mapping is where a missing field becomes an
// empty one rather than a nil dereference. A document with no ask is the
// ordinary case, not an error, so the zero intent has to be a legal answer.
func (s *IntentPublicTestSuite) TestIntentOf() {
	words := []tone.Word{
		{Term: "dark"},
		{Term: "scooped", Evidence: &[]tone.Evidence{{Kind: tone.EvidenceLLM}}},
	}
	attack := tone.Technique{Attack: tone.AttackPick}
	subject := tone.Subject{Kind: tone.KindArtist, Name: "Somebody"}

	tests := []struct {
		name string
		ask  *tone.Ask
		// what the intent must carry.
		words  int
		attack string
		who    string
	}{
		{name: "no ask at all, which is legal and ordinary"},
		{name: "an ask that says nothing", ask: &tone.Ask{}},
		{
			name:  "words, one with evidence and one without",
			ask:   &tone.Ask{Words: &words},
			words: 2,
		},
		{
			name:   "how it is played",
			ask:    &tone.Ask{Technique: &attack},
			attack: "pick",
		},
		{
			// The preset takes its name from the subject, and the pedal shows
			// that name on its screen.
			name: "who it is for",
			ask:  &tone.Ask{Subject: &subject},
			who:  "Somebody",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := presets.IntentOf(tt.ask, nil)

			s.Require().Len(got.Words, tt.words)
			s.Require().Equal(tt.attack, got.Attack)
			s.Require().Equal(tt.who, got.Name)

			if tt.words == 2 {
				// The evidence travels with the word it belongs to, which is
				// what sizes how far that word moves a control.
				s.Require().Empty(got.Words[0].Evidence)
				s.Require().Len(got.Words[1].Evidence, 1)
			}
		})
	}
}

// TestAGenreBringsItsMeasuredWords covers what asking for a genre contributes.
//
// The measurement ships in the binary, so this costs no audio. Each word arrives
// with the figures behind it, which is what lets a build size the move: a genre
// sitting just past the others moves a control barely at all.
func (s *IntentPublicTestSuite) TestAGenreBringsItsMeasuredWords() {
	usable := ""

	all, err := audio.Shipped()
	s.Require().NoError(err)

	for _, g := range all {
		if g.Usable && len(g.Terms) > 0 {
			usable = g.Slug

			break
		}
	}

	if usable == "" {
		s.T().Skip("no measured genre earns a word in this binary")
	}

	got := presets.IntentOf(&tone.Ask{Genre: []string{usable}}, nil)
	s.Require().NotEmpty(got.Words)

	for _, w := range got.Words {
		s.Require().NotEmpty(w.Term)
		s.Require().NotEmpty(w.Evidence, "a word with no figures moves a full step")

		for _, e := range w.Evidence {
			s.Require().Equal(tone.EvidenceAudio, e.Kind)
			s.Require().NotNil(e.Measured, "what the genre read")
			s.Require().NotNil(e.Against, "and what the rest read")
		}
	}
}

// TestAGenreThatBringsNothing covers the three ways a genre contributes no word.
//
// None of them is an error. Saying why is translate's job, and doing it here too
// would say it twice.
func (s *IntentPublicTestSuite) TestAGenreThatBringsNothing() {
	nothing := "sea-shanty"
	s.Require().Empty(presets.IntentOf(&tone.Ask{Genre: []string{nothing}}, nil).Words,
		"nothing measured")

	empty := ""
	s.Require().Empty(presets.IntentOf(&tone.Ask{Genre: []string{empty}}, nil).Words,
		"an empty genre is no genre")

	// A genre that clears nothing. Punk is measured, clears the record
	// threshold, and sits inside the middle half on every axis.
	all, err := audio.Shipped()
	s.Require().NoError(err)

	for _, g := range all {
		if g.Usable && len(g.Terms) == 0 {
			s.Require().Empty(presets.IntentOf(&tone.Ask{Genre: []string{g.Slug}}, nil).Words,
				"%s is measured and sets nothing apart", g.Slug)
		}
	}
}

func TestIntentPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(IntentPublicTestSuite))
}

func TestMakePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MakePublicTestSuite))
}

// TestAGenreUnderTheThresholdContributesNothing covers the rule that keeps one
// band's sound out of a genre.
//
// Tested directly, because every genre measured into this binary clears the
// threshold and which ones do depends on whichever records somebody tagged.
func (s *IntentPublicTestSuite) TestAGenreUnderTheThresholdContributesNothing() {
	short := audio.Genre{
		Name: "emo", Slug: "emo", Records: 3, Players: 1,
		Terms: []audio.Derived{{Term: "scooped", Key: audio.KeyMid, Mine: 0.01, Others: 0.06}},
	}

	s.Require().Empty(presets.GenreWords(short, true, "bass"),
		"three records by one band is that band, whatever it earned")

	// And the same genre once enough backs it.
	short.Records, short.Players, short.Usable = 8, 3, true
	s.Require().Len(presets.GenreWords(short, true, "bass"), 1)
}

// TestAGenreMeasuredOnAnotherInstrumentContributesNothing is the case that
// would otherwise move a control by the wrong amount.
//
// Every word carries the figures that earned it, and those figures size the
// step. So a bass genre's `scooped` handed to a guitar build does not merely
// point the wrong way: it moves a guitar control by how far a bass sat from
// other basses, and the build reports the word as measured evidence.
//
// Dropped here and explained by translate, which is the split everywhere else
// in this file: saying it twice is how two reports come to disagree.
func (s *IntentPublicTestSuite) TestAGenreMeasuredOnAnotherInstrumentContributesNothing() {
	got := audio.Genre{
		Name: "grunge", Slug: "grunge", Records: 9, Players: 3,
		Instrument: "bass", Usable: true,
		Terms: []audio.Derived{
			{Term: "scooped", Key: audio.KeyMid, Mine: 0.01, Others: 0.06},
		},
	}

	s.Require().Empty(presets.GenreWords(got, true, "guitar"),
		"a bass population says nothing about a guitar")
	s.Require().Len(presets.GenreWords(got, true, "bass"), 1,
		"and everything about a bass")
	s.Require().Len(presets.GenreWords(got, true, ""), 1,
		"an ask naming no instrument leaves it to the Setup")
}

// TestAGenrePooledAcrossInstrumentsContributesNothing covers figures that
// describe neither instrument.
func (s *IntentPublicTestSuite) TestAGenrePooledAcrossInstrumentsContributesNothing() {
	mixed := audio.Genre{
		Name: "punk", Slug: "punk", Records: 9, Players: 3,
		Terms: []audio.Derived{
			{Term: "scooped", Key: audio.KeyMid, Mine: 0.01, Others: 0.06},
		},
	}

	s.Require().Empty(presets.GenreWords(mixed, true, "bass"),
		"a centre of gravity between two instruments is not either one")
}

// TestARigGetsTheEffectsItsRecordsAskFor covers task 274's half of the build.
//
// Against the shipped rigs, catalog and library rather than this suite's
// fixture, because what is being checked is that a rig in the marketplace gets
// a block its own records ask for: the fixture names invented models and the
// corpus has never heard of its subject.
//
// Glen Matlock is distinctive on a long decay by 0.27 of a second, which is the
// largest margin any of the 43 earns on that figure, and nothing else a
// pedalboard answers. So he gets one reverb, switched off, behind the cabinet.
func (s *MakePublicTestSuite) TestARigGetsTheEffectsItsRecordsAskFor() {
	out := filepath.Join(s.T().TempDir(), "a.hlx")

	made, err := presets.Make(context.Background(), presets.MakeOptions{
		RigID:      "glen-matlock",
		OutputPath: out,
	})
	s.Require().NoError(err)

	// On the block itself, so a document carries why it is there.
	s.Require().Len(made.Chosen, 1)
	s.Require().Equal(rig.RoleReverb, made.Chosen[0].Role)
	s.Require().NotNil(made.Chosen[0].Enabled)
	s.Require().False(*made.Chosen[0].Enabled)

	// And in the report, because a block nobody asked for is a decision made
	// on the player's behalf.
	said := ""
	for _, at := range made.Added {
		if at.Name == made.Chosen[0].Gear {
			said = at.Reason
		}
	}

	s.Require().Contains(said, "long-decay")
	s.Require().Contains(said, "put a reverb on a switch")

	// Behind the cabinet, because position is not decoration: the reverb is
	// the last block in the chain rather than the first.
	s.Require().NotEmpty(made.Plan.Blocks)
	s.Require().Contains(
		string(made.Plan.Blocks[len(made.Plan.Blocks)-1].Model), "Reverb")
}

// TestARigWhoseRecordsAskForNothingGetsNothing covers the other answer.
//
// Les Claypool is distinctive only on a shorter decay, which nothing on a
// floor answers: a gate would cut his notes off and a compressor is set once
// and left. Fourteen of the 43 rigs here are in that state and seven more earn
// no word at all, so nothing is the common answer rather than a failure.
func (s *MakePublicTestSuite) TestARigWhoseRecordsAskForNothingGetsNothing() {
	out := filepath.Join(s.T().TempDir(), "a.hlx")

	made, err := presets.Make(context.Background(), presets.MakeOptions{
		RigID:      "les-claypool",
		OutputPath: out,
	})
	s.Require().NoError(err)

	s.Require().Empty(made.Chosen)
}
