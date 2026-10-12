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
	"github.com/retr0h/toneharness/pkg/sdk/plan"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

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
	for _, tt := range []struct {
		name string
		of   catalog.Category
		want string
	}{
		{
			// 81% of pitch blocks in the corpus sit ahead of the amp, which is
			// where slot 05C's octaver is.
			name: "a pitch block",
			of:   catalog.CategoryPitch,
			want: "before",
		},
		{
			// 72% of EQs sit after it.
			name: "an equaliser",
			of:   catalog.CategoryEQ,
			want: "after",
		},
		{
			// A kind the corpus never counted goes after, which is the safer
			// end: a block ahead of an amplifier changes what it is fed.
			name: "a kind nothing has counted",
			of:   catalog.Category("nonesuch"),
			want: "after",
		},
		{
			// An instrument with no grammar answers the same way.
			name: "an instrument nothing measured",
			of:   catalog.CategoryEQ,
			want: "after",
		},
	} {
		s.Run(tt.name, func() {
			instrument := "bass"
			if tt.name == "an instrument nothing measured" {
				instrument = "theremin"
			}

			s.Require().Equal(tt.want, side(tt.of, instrument))
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
