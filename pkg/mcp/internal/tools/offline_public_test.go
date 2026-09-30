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

package tools_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/mcp/internal/tools"
	"github.com/retr0h/toneharness/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

type OfflinePublicTestSuite struct {
	suite.Suite
	client *mocks.MockClient
}

func (s *OfflinePublicTestSuite) SetupSubTest() {
	s.client = mocks.NewMockClient(gomock.NewController(s.T()))
}

// row is one call and what it should come back with. check reads the
// structured answer of a call that succeeded.
type row struct {
	name  string
	args  any
	setup func(c *mocks.MockClient)
	want  string
	err   bool
	check func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult)
	// allowWrites starts the server the way --allow-writes does.
	allowWrites bool
}

func (s *OfflinePublicTestSuite) run(
	tool string,
	tests []row,
) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup(s.client)
			}

			res := call(s.T(), connect(s.T(), s.client, tt.allowWrites), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if tt.check != nil {
				tt.check(s, res)
			}
		})
	}
}

var errUnreadable = errors.New("catalog unreadable")

// TestCatalogSearch covers finding blocks.
func (s *OfflinePublicTestSuite) TestCatalogSearch() {
	s.run("catalog_list", []row{
		{
			name: "blocks that match",
			args: tools.Search{Subcategory: "Bass", Search: "SVT"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Blocks(gomock.Any(), sdk.Filter{Subcategory: "Bass", Search: "SVT"}).
					Return(sdk.Blocks{Total: 547, Matched: []catalog.Block{
						{ID: "HD2_AmpSVBeastBrt", Params: map[string]catalog.Param{}},
					}}, nil)
			},
			want: "1 of 547 blocks matched",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Blocks
				structured(s.T(), res, &got)
				s.Len(got.Matched, 1)
			},
		},
		{
			name: "a catalog that will not open",
			args: tools.Search{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Blocks(gomock.Any(), sdk.Filter{}).Return(sdk.Blocks{}, errUnreadable)
			},
			want: "catalog unreadable",
			err:  true,
		},
	})
}

// TestCatalogBlock covers reading one block.
func (s *OfflinePublicTestSuite) TestCatalogBlock() {
	s.run("catalog_show", []row{
		{
			name: "a block the catalog has",
			args: tools.ID{ID: "HD2_AmpSVBeastBrt"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Block(gomock.Any(), "HD2_AmpSVBeastBrt").
					Return(catalog.Block{
						ID: "HD2_AmpSVBeastBrt", Name: "Ampeg SVT Brt",
						Params: map[string]catalog.Param{},
					}, nil)
			},
			want: "HD2_AmpSVBeastBrt is Ampeg SVT Brt",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got catalog.Block
				structured(s.T(), res, &got)
				s.Equal("Ampeg SVT Brt", got.Name)
			},
		},
		{
			// An agent is pointed at the tool that finds blocks, not at a
			// command it cannot run.
			name: "a block it does not",
			args: tools.ID{ID: "HD2_Nope"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Block(gomock.Any(), "HD2_Nope").
					Return(catalog.Block{}, fmt.Errorf("%w %q", sdk.ErrNoSuchBlock, "HD2_Nope"))
			},
			want: "call catalog_list",
			err:  true,
		},
		{
			name: "a catalog that will not open",
			args: tools.ID{ID: "HD2_AmpSVBeastBrt"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Block(gomock.Any(), "HD2_AmpSVBeastBrt").
					Return(catalog.Block{}, errUnreadable)
			},
			want: "catalog unreadable",
			err:  true,
		},
	})
}

// TestCorpusModel covers how players set one model.
func (s *OfflinePublicTestSuite) TestCorpusModel() {
	id := catalog.ModelID("HD2_AmpSVBeastBrt")
	cat := &catalog.Catalog{Blocks: map[catalog.ModelID]catalog.Block{
		id: {ID: id, Name: "Ampeg SVT Brt", Params: map[string]catalog.Param{}},
	}}

	s.run("corpus_presets_show", []row{
		{
			name: "a model the corpus measured",
			args: tools.ID{ID: string(id)},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					ModelMeasurements(gomock.Any(), string(id)).
					Return(sdk.Measured{
						Stats: &corpus.Stats{Models: map[catalog.ModelID]corpus.ModelStats{
							id: {
								Uses: 26,
								Params: map[string]corpus.ParamStats{
									"Treble": {N: 26, Median: 0.85},
								},
							},
						}},
						Catalog: cat,
						Model:   id,
					}, nil)
			},
			want: "used 26 times",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Model
				structured(s.T(), res, &got)
				s.Equal("Ampeg SVT Brt", got.Block.Name)
				s.InDelta(0.85, got.Params["Treble"].Median, 1e-9)
			},
		},
		{
			name: "a model nobody measured",
			args: tools.ID{ID: "HD2_Nope"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().ModelMeasurements(gomock.Any(), "HD2_Nope").
					Return(sdk.Measured{}, errors.New("HD2_Nope was not measured"))
			},
			want: "was not measured",
			err:  true,
		},
		{
			// The corpus measured this model, but the catalog it was
			// resolved against has since dropped it: corpus and catalog
			// have drifted apart.
			name: "a model the corpus measured but the catalog lacks",
			args: tools.ID{ID: string(id)},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					ModelMeasurements(gomock.Any(), string(id)).
					Return(sdk.Measured{
						Stats: &corpus.Stats{Models: map[catalog.ModelID]corpus.ModelStats{
							id: {Uses: 26},
						}},
						Catalog: &catalog.Catalog{},
						Model:   id,
					}, nil)
			},
			want: tools.ErrNotInCatalog.Error(),
			err:  true,
		},
	})
}

// TestRigsList covers listing the shipped rigs.
func (s *OfflinePublicTestSuite) TestRigsList() {
	s.run("rigs_list", []row{
		{
			name: "the rigs that ship",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Rigs(gomock.Any()).Return(sdk.Rigs{Rigs: []sdk.Known{{}, {}}}, nil)
			},
			want: "2 rigs to build from",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Rigs
				structured(s.T(), res, &got)
				s.Len(got.Rigs, 2)
			},
		},
		{
			name: "rigs that will not read",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Rigs(gomock.Any()).
					Return(sdk.Rigs{}, errors.New("rigs unreadable"))
			},
			want: "rigs unreadable",
			err:  true,
		},
	})
}

// TestToneBuild covers the path an agent reaches the whole project through.
//
// It was unreachable over MCP until this tool existed, because the resolving
// lived in pkg/cli and an agent calls a tool rather than a command.
func (s *OfflinePublicTestSuite) TestToneBuild() {
	s.run("tone_build", []row{
		{
			name: "a request it can answer",
			args: tools.Asked{Spec: "ask.yaml", Setup: "mine.yaml"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Tone(gomock.Any(), sdk.Ask{Spec: "ask.yaml", Setup: "mine.yaml"}).
					Return(sdk.Resolved{
						Rig: rig.Spec{Chain: []rig.ChainEntry{
							{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
						}},
						Notes: translate.Notes{{About: "amp", Said: "chosen by measuring"}},
					}, nil)
			},
			want: "1 blocks, 1 notes",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Resolved
				structured(s.T(), res, &got)
				s.Len(got.Rig.Chain, 1)
				s.Len(got.Notes, 1)
			},
		},
		{
			// A setup is optional, and the answer says what it assumed
			// instead of refusing.
			name: "a request with nothing about what they own",
			args: tools.Asked{Spec: "ask.yaml"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Tone(gomock.Any(), sdk.Ask{Spec: "ask.yaml"}).
					Return(sdk.Resolved{
						Notes: translate.Notes{{About: "setup", Said: "assumed a bass"}},
					}, nil)
			},
			want: "0 blocks, 1 notes",
		},
		{
			// The notes travel even when it failed, because what could not be
			// honoured is the useful half either way.
			name: "a request it could not answer",
			args: tools.Asked{Spec: "ask.yaml"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Tone(gomock.Any(), gomock.Any()).
					Return(sdk.Resolved{
						Notes: translate.Notes{{About: "Ampeg B-15", Said: "no such model"}},
					}, translate.ErrInsisted)
			},
			err: true,
		},
	})
}

// TestRigShow covers reading one rig.
func (s *OfflinePublicTestSuite) TestRigShow() {
	s.run("rigs_show", []row{
		{
			name: "a rig that ships",
			args: tools.ID{ID: "mike-dirnt"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Rig(gomock.Any(), "mike-dirnt").
					Return(sdk.Rig{Variants: []sdk.Variant{{}}}, nil)
			},
			want: "rig mike-dirnt, extended by 1 others",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Rig
				structured(s.T(), res, &got)
				s.Len(got.Variants, 1)
			},
		},
		{
			name: "a rig that does not",
			args: tools.ID{ID: "nobody"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Rig(gomock.Any(), "nobody").
					Return(sdk.Rig{}, fmt.Errorf("%w %q", sdk.ErrNoSuchRig, "nobody"))
			},
			want: "call rigs_list",
			err:  true,
		},
	})
}

// TestPresetMake covers building from either source.
func (s *OfflinePublicTestSuite) TestPresetMake() {
	dir := s.T().TempDir()
	fresh := filepath.Join(dir, "fresh.hlx")
	held := filepath.Join(dir, "held.hlx")
	s.Require().NoError(os.WriteFile(held, []byte("somebody's preset"), 0o600))
	racy := filepath.Join(dir, "racy.hlx")
	racyRig := filepath.Join(dir, "racy-rig.hlx")
	// A rig that builds, so a real compile has something to write.
	rigFile := filepath.Join("..", "..", "..", "..", "examples", "rigspec", "mike-dirnt.yaml")

	s.run("presets_make", []row{
		{
			name: "a path nothing is at",
			args: tools.Make{RigID: "mike-dirnt", Out: fresh},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Make(gomock.Any(), "mike-dirnt", fresh, sdk.KeepExisting).
					Return(sdk.Made{}, nil)
			},
			want: "wrote " + fresh,
		},
		{
			// No client call is expected, so reaching one fails the row.
			name: "a path a file is at, with writes off",
			args: tools.Make{RigPath: "mine.yaml", Out: held},
			want: tools.ErrWouldOverwrite.Error() + ": " + held,
			err:  true,
		},
		{
			name: "a path a file is at, with writes on",
			args: tools.Make{RigID: "mike-dirnt", Out: held},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Make(gomock.Any(), "mike-dirnt", held, sdk.ReplaceExisting).
					Return(sdk.Made{}, nil)
			},
			want:        "wrote " + held,
			allowWrites: true,
		},
		{
			// Nothing is at the path when the tool decides, and somebody's
			// preset is by the time the file is written. The hook lands it
			// there and then builds for real, so what refuses it can only be
			// the write itself.
			name: "a file that appears between the decision and the write, with writes off",
			args: tools.Make{RigID: "mike-dirnt", Out: racy},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Make(gomock.Any(), "mike-dirnt", racy, sdk.KeepExisting).
					DoAndReturn(func(
						ctx context.Context, id, out string, existing sdk.Existing,
					) (sdk.Made, error) {
						if err := os.WriteFile(out, []byte("somebody's preset"), 0o600); err != nil {
							return sdk.Made{}, err
						}

						return sdk.New().Make(ctx, id, out, existing)
					})
			},
			want: tools.ErrWouldOverwrite.Error() + ": " + racy,
			err:  true,
			check: func(s *OfflinePublicTestSuite, _ *gomcp.CallToolResult) {
				got, err := os.ReadFile(racy) //nolint:gosec // a path this test chose
				s.Require().NoError(err)
				s.Require().Equal("somebody's preset", string(got))
			},
		},
		{
			// The same race on the other source: a rig file compiled for
			// real, through Compile's own write.
			name: "a file that appears between the decision and the write, from a rig file, with writes off",
			args: tools.Make{RigPath: rigFile, Out: racyRig},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Compile(gomock.Any(), sdk.Compile{
						Rig: rigFile, Out: racyRig, Existing: sdk.KeepExisting,
					}).
					DoAndReturn(func(ctx context.Context, in sdk.Compile) (sdk.Built, error) {
						if err := os.WriteFile(in.Out, []byte("somebody's preset"), 0o600); err != nil {
							return sdk.Built{}, err
						}

						return sdk.New().Compile(ctx, in)
					})
			},
			want: tools.ErrWouldOverwrite.Error() + ": " + racyRig,
			err:  true,
			check: func(s *OfflinePublicTestSuite, _ *gomcp.CallToolResult) {
				got, err := os.ReadFile(racyRig) //nolint:gosec // a path this test chose
				s.Require().NoError(err)
				s.Require().Equal("somebody's preset", string(got))
			},
		},
		{
			name: "from a shipped rig",
			args: tools.Make{RigID: "mike-dirnt", Out: "mike.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Make(gomock.Any(), "mike-dirnt", "mike.hlx", sdk.KeepExisting).
					Return(sdk.Made{}, nil)
			},
			want: "wrote mike.hlx from rig mike-dirnt",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Outcome
				structured(s.T(), res, &got)
				s.NotNil(got.FromShipped)
				s.Nil(got.FromRig)
			},
		},
		{
			name: "from a rig file",
			args: tools.Make{RigPath: "mine.yaml", Out: "mine.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Compile(gomock.Any(), sdk.Compile{
						Rig: "mine.yaml", Out: "mine.hlx", Existing: sdk.KeepExisting,
					}).
					Return(sdk.Built{}, nil)
			},
			want: "wrote mine.hlx from mine.yaml",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Outcome
				structured(s.T(), res, &got)
				s.NotNil(got.FromRig)
			},
		},
		{
			name: "a shipped rig that will not build",
			args: tools.Make{RigID: "mike-dirnt", Out: "mike.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sdk.Made{}, errors.New("over budget"))
			},
			want: "over budget",
			err:  true,
		},
		{
			name: "a shipped rig nobody wrote",
			args: tools.Make{RigID: "nobody", Out: "nobody.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Make(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sdk.Made{}, fmt.Errorf("%w %q", sdk.ErrNoSuchRig, "nobody"))
			},
			want: "call rigs_list",
			err:  true,
		},
		{
			name: "a rig file that will not build",
			args: tools.Make{RigPath: "mine.yaml", Out: "mine.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					Return(sdk.Built{}, errors.New("unknown block"))
			},
			want: "unknown block",
			err:  true,
		},
		{
			name: "nothing to build from",
			args: tools.Make{Out: "x.hlx"},
			want: tools.ErrNoSource.Error(),
			err:  true,
		},
		{
			name: "two things to build from",
			args: tools.Make{RigID: "a", RigPath: "b", Out: "x.hlx"},
			want: tools.ErrTwoSources.Error(),
			err:  true,
		},
	})
}

func TestOfflinePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OfflinePublicTestSuite))
}

// TestCorpusMusicPlayers covers who the corpus holds records for.
func (s *OfflinePublicTestSuite) TestCorpusMusicPlayers() {
	s.run("corpus_music_players", []row{
		{
			name: "the players in one instrument's tree",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicPlayers(gomock.Any(), "resources/music/bass").
					Return([]sdk.MusicPlayer{
						{ID: "matt-freeman", Artist: "Matt Freeman", Records: 4},
						{ID: "mike-dirnt", Artist: "Mike Dirnt", Records: 3},
					}, nil)
			},
			want: "2 players in resources/music/bass",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got []sdk.MusicPlayer
				structured(s.T(), res, &got)
				s.Len(got, 2)
				s.Equal(4, got[0].Records)
			},
		},
		{
			name: "a corpus that is not there",
			args: tools.Corpus{Corpus: "nowhere"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicPlayers(gomock.Any(), "nowhere").
					Return(nil, errors.New("no such corpus"))
			},
			want: "no such corpus",
			err:  true,
		},
	})
}

// TestCorpusMusicBands covers the bands behind the records.
func (s *OfflinePublicTestSuite) TestCorpusMusicBands() {
	s.run("corpus_music_bands", []row{
		{
			name: "bands grouped on one slug",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicBands(gomock.Any(), "resources/music/bass").
					Return([]sdk.MusicGroup{
						{Name: "Rancid", Slug: "rancid", Records: 4, Artists: 1},
					}, nil)
			},
			want: "1 bands made the records in resources/music/bass",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got []sdk.MusicGroup
				structured(s.T(), res, &got)
				s.Equal("rancid", got[0].Slug)
			},
		},
		{
			name: "manifests that will not read",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicBands(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("a manifest is malformed"))
			},
			want: "a manifest is malformed",
			err:  true,
		},
	})
}

// TestCorpusMusicGenres covers what is tagged, without measuring anything.
func (s *OfflinePublicTestSuite) TestCorpusMusicGenres() {
	s.run("corpus_music_genres", []row{
		{
			name: "genres and the players behind them",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicGenres(gomock.Any(), "resources/music/bass").
					Return([]sdk.MusicGroup{
						{Name: "punk", Slug: "punk", Records: 9, Artists: 3},
						{Name: "grunge", Slug: "grunge", Records: 2, Artists: 1},
					}, nil)
			},
			want: "2 genres are tagged in resources/music/bass",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got []sdk.MusicGroup
				structured(s.T(), res, &got)
				s.Equal(3, got[0].Artists, "the half usually short")
			},
		},
		{
			name: "a corpus that will not read",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicGenres(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("no manifests found"))
			},
			want: "no manifests found",
			err:  true,
		},
	})
}

// TestCorpusMusicRecords covers every recording the corpus names.
func (s *OfflinePublicTestSuite) TestCorpusMusicRecords() {
	s.run("corpus_music_records", []row{
		{
			name: "the records, with their years",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicRecords(gomock.Any(), "resources/music/bass").
					Return([]sdk.MusicRecord{
						{
							Player: "matt-freeman",
							Track:  "Maxwell Murder",
							Year:   1995,
							Band:   "Rancid",
						},
					}, nil)
			},
			want: "1 records in resources/music/bass",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got []sdk.MusicRecord
				structured(s.T(), res, &got)
				s.Equal(1995, got[0].Year)
			},
		},
		{
			name: "a corpus that will not read",
			args: tools.Corpus{Corpus: "nowhere"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MusicRecords(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("no such corpus"))
			},
			want: "no such corpus",
			err:  true,
		},
	})
}

// TestCorpusPresetsChains covers what a chain almost always holds.
func (s *OfflinePublicTestSuite) TestCorpusPresetsChains() {
	s.run("corpus_presets_chains", []row{
		{
			name: "what bass chains hold",
			args: tools.Instrument{Instrument: "bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().ChainMeasurements(gomock.Any(), "bass").
					Return(sdk.Measured{Instrument: "bass"}, nil)
			},
			want: "what bass chains hold",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Measured
				structured(s.T(), res, &got)
				s.Equal("bass", got.Instrument)
			},
		},
		{
			name: "an instrument nothing was measured for",
			args: tools.Instrument{Instrument: "sitar"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().ChainMeasurements(gomock.Any(), "sitar").
					Return(sdk.Measured{}, errors.New("sitar was not measured"))
			},
			want: "sitar was not measured",
			err:  true,
		},
	})
}

// TestRigsRecords covers holding a rig to the era its ask claims.
func (s *OfflinePublicTestSuite) TestRigsRecords() {
	s.run("rigs_records", []row{
		{
			name: "rigs and the records behind them",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Backing(gomock.Any(), "resources/music/bass").
					Return([]sdk.Backing{
						{ID: "matt-freeman", Era: "2024", From: 2024, To: 2024},
					}, nil)
			},
			want: "1 rigs held to the era their ask gives",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got []sdk.Backing
				structured(s.T(), res, &got)
				s.Equal(2024, got[0].From)
			},
		},
		{
			name: "rigs that will not read",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Backing(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("a rig names no era"))
			},
			want: "a rig names no era",
			err:  true,
		},
	})
}

// TestMeasureGenres covers measuring a genre against the players who avoid it.
//
// The count in the answer is genres measured against genres earning a word, and
// they are different numbers on purpose: a genre only two players carry is
// measured and earns nothing, which is the evidence being thin rather than
// something going wrong.
func (s *OfflinePublicTestSuite) TestMeasureGenres() {
	s.run("measure_genres", []row{
		{
			name: "one genre earns a word and one does not",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MeasuredGenres(gomock.Any(), "resources/music/bass").
					Return([]audio.Genre{
						{
							Name: "punk", Slug: "punk", Records: 9, Players: 3, Against: 12,
							Terms: []audio.Derived{{Term: "aggressive"}},
						},
						{Name: "grunge", Slug: "grunge", Records: 2, Players: 1, Against: 14},
					}, nil)
			},
			want: "2 genres measured, 1 earning a word",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got []audio.Genre
				structured(s.T(), res, &got)
				s.Len(got, 2)
				s.Empty(got[1].Terms, "one player earns nothing")
			},
		},
		{
			name: "recordings that will not read",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MeasuredGenres(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("no recordings under that tree"))
			},
			want: "no recordings under that tree",
			err:  true,
		},
	})
}

// TestMeasurePlayers covers what a player's records earn them.
func (s *OfflinePublicTestSuite) TestMeasurePlayers() {
	s.run("measure_players", []row{
		{
			name: "two players, one earning",
			args: tools.Corpus{Corpus: "resources/music/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MeasuredPlayers(gomock.Any(), "resources/music/bass").
					Return([]audio.Player{
						{
							ID: "matt-freeman", Records: 4,
							Terms: []audio.Derived{{Term: "bright"}},
						},
						{ID: "mike-dirnt", Records: 3},
					}, nil)
			},
			want: "2 players measured, 1 earning a word",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got []audio.Player
				structured(s.T(), res, &got)
				s.Equal("matt-freeman", got[0].ID)
			},
		},
		{
			name: "a corpus with nothing in it",
			args: tools.Corpus{Corpus: "empty"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MeasuredPlayers(gomock.Any(), "empty").
					Return(nil, errors.New("no players under empty"))
			},
			want: "no players under empty",
			err:  true,
		},
	})
}

// TestMeasureRecordings covers measuring a directory of separated audio.
func (s *OfflinePublicTestSuite) TestMeasureRecordings() {
	s.run("measure_recordings", []row{
		{
			name: "every file, and what they measure together",
			args: tools.Recordings{Dir: "stems/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MeasuredRecordings(gomock.Any(), "stems/bass").
					Return([]audio.Named{
						{Name: "Maxwell Murder"},
						{Name: "Roots Radicals"},
					}, audio.Across{Tracks: 2}, nil)
			},
			want: "2 recordings measured",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Recorded
				structured(s.T(), res, &got)
				s.Len(got.Tracks, 2)
				s.Equal(2, got.Together.Tracks)
			},
		},
		{
			name: "a directory of nothing readable",
			args: tools.Recordings{Dir: "stems/bass"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().MeasuredRecordings(gomock.Any(), gomock.Any()).
					Return(nil, audio.Across{}, errors.New("no audio in stems/bass"))
			},
			want: "no audio in stems/bass",
			err:  true,
		},
	})
}
