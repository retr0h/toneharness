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
	"github.com/retr0h/toneharness/pkg/sdk/preset"
	"github.com/retr0h/toneharness/pkg/sdk/result"
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

// TestMake builds a preset out of a rig.
func (s *MakePublicTestSuite) TestMake() {
	tests := []struct {
		name     string
		ctx      context.Context
		id       string
		stats    string
		catalog  string
		out      string
		contains []string
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
			if tt.stats != "" {
				o.StatsPath = tt.stats
			}

			if tt.catalog != "" {
				o.Catalogs = s.catalogs(tt.catalog)
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
				s.Require().Equal("Test Player", doc.Data.Meta.Name)

				spec, err := doc.Spec()
				if err != nil {
					return err
				}

				s.Require().Len(spec.Blocks, 2)

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
// empty one rather than a nil dereference. A rig with no ask beside it is the
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
		ask  *tone.Spec
		// what the intent must carry.
		words  int
		attack string
		who    string
	}{
		{name: "no ask at all, which is legal and ordinary"},
		{name: "an ask that says nothing", ask: &tone.Spec{}},
		{
			name:  "words, one with evidence and one without",
			ask:   &tone.Spec{Words: &words},
			words: 2,
		},
		{
			name:   "how it is played",
			ask:    &tone.Spec{Technique: &attack},
			attack: "pick",
		},
		{
			// The preset takes its name from the subject, and the pedal shows
			// that name on its screen.
			name: "who it is for",
			ask:  &tone.Spec{Subject: &subject},
			who:  "Somebody",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := presets.IntentOf(tt.ask)

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

	got := presets.IntentOf(&tone.Spec{Genre: []string{usable}})
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
	s.Require().Empty(presets.IntentOf(&tone.Spec{Genre: []string{nothing}}).Words,
		"nothing measured")

	empty := ""
	s.Require().Empty(presets.IntentOf(&tone.Spec{Genre: []string{empty}}).Words,
		"an empty genre is no genre")

	// A genre that clears nothing. Punk is measured, clears the record
	// threshold, and sits inside the middle half on every axis.
	all, err := audio.Shipped()
	s.Require().NoError(err)

	for _, g := range all {
		if g.Usable && len(g.Terms) == 0 {
			s.Require().Empty(presets.IntentOf(&tone.Spec{Genre: []string{g.Slug}}).Words,
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

	s.Require().Empty(presets.GenreWords(short, true),
		"three records by one band is that band, whatever it earned")

	// And the same genre once enough backs it.
	short.Records, short.Players, short.Usable = 8, 3, true
	s.Require().Len(presets.GenreWords(short, true), 1)
}
