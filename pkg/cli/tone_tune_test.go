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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
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

// TuneTestSuite covers solving a chain for a target, without hardware.
//
// The bench is a fake that answers the same sine whatever is asked of it, so
// every slope reads as zero and the loop's arithmetic is what is under test
// rather than an amplifier. What a device decides is whether the slopes were
// true, and only the pedal answers that.
type TuneTestSuite struct {
	suite.Suite

	ctrl    *gomock.Controller
	pedal   *mocks.MockTuner
	genre   *mocks.MockGenres
	players *mocks.MockRecorded
	dry     string
}

func (s *TuneTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockTuner(s.ctrl)
	// Every measuring command asks the client which models the attached
	// device has, rather than reading the built-in HX Stomp catalog, so that
	// --catalog and --device reach them.
	s.pedal.EXPECT().Catalog(gomock.Any()).
		DoAndReturn(func(context.Context) (*catalog.Catalog, error) {
			return catalog.BuiltIn()
		}).AnyTimes()
	s.genre = mocks.NewMockGenres(s.ctrl)
	s.players = mocks.NewMockRecorded(s.ctrl)
	s.dry = "../../resources/dry/bass-di-short.wav"

	// The preset read back and rebuilt: every command here takes its chain off
	// the measuring loop before playing it, sending the output to USB rather
	// than to the socket the measuring lead comes from. What that rewrite does
	// is offTheLoop's own test, so it is answered here rather than asserted.
	s.pedal.EXPECT().
		PresetFile(gomock.Any(), gomock.Any()).
		Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, nil).AnyTimes()
}

// punk is a target with room on every axis, so a chain that cannot move
// arrives rather than churning.
func (s *TuneTestSuite) punk() []audio.Genre {
	// Wide enough that whatever the fake bench answers is inside it, because
	// what is under test here is the loop rather than an amplifier.
	wide := audio.Spread{Low: -1000, Mid: 0, High: 1000}

	return []audio.Genre{{
		Name: "punk", Slug: "punk",
		Across: audio.Across{
			Tracks: 12,
			Low:    wide, Mid: wide, High: wide, Centroid: wide,
		},
	}}
}

// matt is one player's own records, as wide as punk is, so what is under test
// is which target the run read rather than how far the solve got.
func (s *TuneTestSuite) matt() []audio.Player {
	wide := audio.Spread{Low: -1000, Mid: 0, High: 1000}

	return []audio.Player{{
		ID: "matt-freeman", Records: 3,
		Across: audio.Across{
			Tracks: 3,
			Low:    wide, Mid: wide, High: wide, Centroid: wide,
		},
	}}
}

// aimedHigh is a target far enough off that the solve has work to do, and
// tight enough that it cannot shrug the axis off.
func (s *TuneTestSuite) aimedHigh() []audio.Genre {
	return []audio.Genre{{
		Name: "punk", Slug: "punk",
		Across: audio.Across{
			Tracks:   12,
			Centroid: audio.Spread{Low: 380, Mid: 400, High: 420},
		},
	}}
}

// ready makes the pedal answer everything the loop asks of it.
func (s *TuneTestSuite) ready() {
	s.pedal.EXPECT().
		Make(gomock.Any(), gomock.Any()).
		Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
			Model:   catalog.ModelID("HD2_AmpSVBeastBrt"),
			Pos:     0,
			Enabled: true,
		}}}}, nil).AnyTimes()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.pedal.EXPECT().
		Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// The lists as well as the dials. This amplifier's MidFreq is a list of
	// three, so every run of the loop compares it before solving anything, and
	// a suite that expected only Turn was describing a chain of dials only.
	s.pedal.EXPECT().
		Choose(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
}

// opts is a run with the hardware faked out.
func (s *TuneTestSuite) opts() TuneOptions {
	return TuneOptions{
		Client: s.pedal, Genres: s.genre, Players: s.players, Bench: bench{},
		ID: "matt-freeman", Genre: "punk", Corpus: "resources/music/bass",
		Dry: s.dry, Seconds: 0.1, Takes: 2, Passes: 2, Nudge: 0.1,
	}
}

// TestTune covers Tune, which solves a chain's controls for a target and says
// how close it got.
//
// One method and one table, so a case is a row rather than a file.
func (s *TuneTestSuite) TestTune() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Arriving on the first pass.
			name: "a target it can already meet",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				var buf bytes.Buffer
				s.Require().NoError(Tune(context.Background(), &buf, s.opts()))
				s.Require().Contains(buf.String(), "aimed at punk")
			},
		},
		{
			// A target nobody measured.
			name: "a genre nothing is tagged with",
			then: func() {
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Genre = "sea shanty"

				err := Tune(context.Background(), buffer(), opts)
				s.Require().ErrorIs(err, ErrNoTarget)
			},
		},
		{
			// A request that aims at nothing.
			name: "no genre at all",
			then: func() {
				opts := s.opts()
				opts.Genre = ""

				s.Require().ErrorIs(Tune(context.Background(), buffer(), opts), ErrNoTarget)
			},
		},
		{
			// One artist's own records rather than the middle of a population,
			// which is the stronger target when the request names a person.
			name: "a player's own records",
			then: func() {
				s.ready()
				s.players.EXPECT().
					MeasuredPlayers(gomock.Any(), "resources/music/bass").
					Return(s.matt(), nil)

				opts := s.opts()
				opts.Genre = ""
				opts.Player = "matt-freeman"

				var buf bytes.Buffer
				s.Require().NoError(Tune(context.Background(), &buf, opts))
				s.Require().Contains(buf.String(), "matt-freeman's own records")
			},
		},
		{
			// Two targets.
			//
			// A genre is the middle of a population and a player is the person
			// being emulated, so they are different figures. Preferring one
			// quietly would solve for something nobody asked for.
			name: "a genre and a player at once",
			then: func() {
				opts := s.opts()
				opts.Player = "matt-freeman"

				s.Require().ErrorIs(Tune(context.Background(), buffer(), opts), ErrOneTarget)
			},
		},
		{
			// A player with nowhere to measure them from. A genre's figures
			// ship in the binary and a player's cannot, because the recordings
			// are not in this repository.
			name: "a player and no corpus",
			then: func() {
				opts := s.opts()
				opts.Genre = ""
				opts.Player = "matt-freeman"
				opts.Corpus = ""

				s.Require().
					ErrorIs(Tune(context.Background(), buffer(), opts), ErrPlayerNeedsCorpus)
			},
		},
		{
			// A player the tree holds nothing for.
			name: "a player the corpus does not have",
			then: func() {
				s.players.EXPECT().
					MeasuredPlayers(gomock.Any(), gomock.Any()).Return(s.matt(), nil)

				opts := s.opts()
				opts.Genre = ""
				opts.Player = "somebody-else"

				s.Require().ErrorIs(Tune(context.Background(), buffer(), opts), ErrNoTarget)
			},
		},
		{
			// A player whose directory is there and whose records are not.
			// Refused rather than aimed at zero, which every axis would read as
			// a gap the whole way.
			name: "a player with no measured records",
			then: func() {
				s.players.EXPECT().
					MeasuredPlayers(gomock.Any(), gomock.Any()).
					Return([]audio.Player{{ID: "matt-freeman"}}, nil)

				opts := s.opts()
				opts.Genre = ""
				opts.Player = "matt-freeman"

				s.Require().ErrorIs(Tune(context.Background(), buffer(), opts), ErrNoTarget)
			},
		},
		{
			// A tree that will not read, asked for a player.
			name: "the corpus will not read for a player",
			then: func() {
				s.players.EXPECT().MeasuredPlayers(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("no recordings under that tree"))

				opts := s.opts()
				opts.Genre = ""
				opts.Player = "matt-freeman"

				err := Tune(context.Background(), buffer(), opts)
				s.Require().ErrorContains(err, "no recordings under that tree")
			},
		},
		{
			// A tree that is not there.
			name: "the corpus will not read",
			then: func() {
				s.genre.EXPECT().MeasuredGenres(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("no recordings under that tree"))

				err := Tune(context.Background(), buffer(), s.opts())
				s.Require().ErrorContains(err, "no recordings under that tree")
			},
		},
		{
			// A tag with no figures behind it.
			name: "a genre that measures as nothing",
			then: func() {
				s.genre.EXPECT().MeasuredGenres(gomock.Any(), gomock.Any()).
					Return([]audio.Genre{{Name: "punk", Slug: "punk"}}, nil)

				err := Tune(context.Background(), buffer(), s.opts())
				s.Require().ErrorIs(err, ErrNoTarget)
			},
		},
		{
			// Gear the catalog cannot realise.
			name: "a rig that will not build",
			then: func() {
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)
				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any()).
					Return(sdk.Made{}, errors.New("over the DSP budget"))

				err := Tune(context.Background(), buffer(), s.opts())
				s.Require().ErrorContains(err, "over the DSP budget")
			},
		},
		{
			// A chain the solver cannot touch.
			//
			// A block the catalog does not carry contributes no control, so
			// the chain has nothing to turn. Refused rather than reported as
			// arrived: a request that cannot be acted on is not a request
			// that was satisfied.
			name: "a chain with no dial",
			then: func() {
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)
				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any()).
					Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
						Model: catalog.ModelID("HD2_NoSuchBlock"),
					}}}}, nil)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)

				err := Tune(context.Background(), buffer(), s.opts())
				s.Require().ErrorIs(err, solve.ErrNoKnobs)
			},
		},
		{
			// A device that will not load it.
			name: "the pedal refuses the chain",
			then: func() {
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)
				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any()).
					Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
						Model: catalog.ModelID("HD2_AmpSVBeastBrt"),
					}}}}, nil)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
					Return(errors.New("no reply to opcode 21"))

				err := Tune(context.Background(), buffer(), s.opts())
				s.Require().ErrorContains(err, "no reply to opcode 21")
			},
		},
		{
			// A missing signal.
			name: "a reference that is not there",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Dry = "nowhere.wav"

				s.Require().Error(Tune(context.Background(), buffer(), opts))
			},
		},
		{
			// The guard.
			//
			// Every figure measured by pushing a guitar recording through a
			// bass rig describes the recording rather than the chain, so the
			// solve would spend its dials closing a gap that is the wrong
			// instrument rather than the wrong settings. Refused before a
			// single reading is taken.
			name: "tune refuses a reference for the other instrument",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				// The same recording under a name that says guitar, because the reference's
				// instrument is read off its filename: what is in the file is nobody's to
				// know and a path is what somebody typed.
				body, err := os.ReadFile(s.dry)
				s.Require().NoError(err)

				wrong := filepath.Join(s.T().TempDir(), "guitar-di.wav")
				s.Require().NoError(os.WriteFile(wrong, body, 0o600))

				opts := s.opts()
				opts.Dry = wrong

				err = Tune(context.Background(), buffer(), opts)

				s.Require().ErrorIs(err, ErrWrongInstrument)
				s.Require().ErrorContains(err, "Push a bass recording with --dry")
			},
		},
		{
			// --ask pointing nowhere.
			//
			// The ask is the only account of what was asked for, so a round
			// that solved and could not record it is a failure rather than a
			// note: the settings on the pedal last until the next preset is
			// selected and nothing else remembers why they are there.
			name: "tune reports an ask it cannot append to",
			then: func() {
				s.ready()
				s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
					Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Ask = filepath.Join(s.T().TempDir(), "nowhere", "ask.yaml")

				s.Require().Error(Tune(context.Background(), buffer(), opts))
			},
		},
		{
			// --out pointing nowhere.
			name: "tune reports somewhere it cannot write",
			then: func() {
				s.ready()
				s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
					Return(sdk.Reading{Plan: routed()}, nil).AnyTimes()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Out = filepath.Join(s.T().TempDir(), "nowhere", "tuned.yaml")

				s.Require().Error(Tune(context.Background(), buffer(), opts))
			},
		},
		{
			// The read --out depends on.
			//
			// What is kept is the chain as the device reports it after the
			// dials moved, so a file written when that read failed would be a
			// plan claiming settings nobody read back.
			//
			// Only that read. A preset that will not read back is warned
			// about rather than fatal — offTheLoop says so and carries on —
			// which is TestItWarnsWhenItCannotPutTheOutputBack's subject.
			name: "keeping a tune needs the chain the device is playing",
			then: func() {
				s.ready()
				s.pedal.EXPECT().Current(gomock.Any(), gomock.Any()).
					Return(sdk.Reading{}, errors.New("would not say")).AnyTimes()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Out = filepath.Join(s.T().TempDir(), "tuned.yaml")

				s.Require().ErrorContains(
					Tune(context.Background(), buffer(), opts), "would not say")
			},
		},
		{
			// The device refusing a parameter.
			//
			// Aimed at a target it has to work for, because a loop that has
			// already arrived reads no slopes and so never touches a dial.
			name: "a dial that will not move",
			then: func() {
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.aimedHigh(), nil)
				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any()).
					Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
						Model: catalog.ModelID("HD2_AmpSVBeastBrt"),
					}}}}, nil)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)
				// The lists are compared before any dial is read, so this refusal is only
				// reached once MidFreq has been walked.
				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("no reply to opcode 30 within 6s"))

				err := Tune(context.Background(), buffer(), s.opts())
				s.Require().ErrorContains(err, "opcode 30")
			},
		},
		{
			// A solve with nowhere to go.
			//
			// Every slope is zero because the fake bench answers the same
			// signal whatever moves, so no control can close the gap. Refused
			// rather than reported as arrived, because a request nothing can
			// act on is not one that was satisfied.
			name: "a target the chain cannot reach",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).
					Return([]audio.Genre{{
						Name: "punk", Slug: "punk",
						Across: audio.Across{
							Tracks:   12,
							Centroid: audio.Spread{Low: 7000, Mid: 8000, High: 9000},
						},
					}}, nil)

				err := Tune(context.Background(), buffer(), s.opts())
				s.Require().ErrorIs(err, solve.ErrNoKnobs)
			},
		},
		{
			// The applying half.
			//
			// A bench whose answer changes gives every control a slope, so
			// the solve has somewhere to go and the moves reach the pedal.
			// What is asserted is that they were sent, because whether they
			// were the right moves is a question only hardware answers.
			name: "it moves the dials it solved for",
			then: func() {
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.aimedHigh(), nil)
				s.pedal.EXPECT().
					Make(gomock.Any(), gomock.Any()).
					Return(sdk.Made{Plan: plan.Plan{Blocks: []plan.Block{{
						Model: catalog.ModelID("HD2_AmpSVBeastBrt"),
					}}}}, nil)
				s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil)

				s.pedal.EXPECT().Choose(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				moved := 0
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(context.Context, sdk.Address, float32) error {
						moved++

						return nil
					}).AnyTimes()

				opts := s.opts()
				opts.Bench = &sloping{}
				opts.Passes = 1

				var buf bytes.Buffer
				_ = Tune(context.Background(), &buf, opts)

				s.Require().Positive(moved, "the dials were turned")
			},
		},
		{
			// The conversational half of the loop.
			//
			// A nudge lives on the ask rather than on a flag, because it is a
			// decision about how something should sound and it has to survive
			// the session that made it. The run says what moved, since a
			// target that shifted silently cannot be told from a chain that
			// drifted.
			name: "a nudge on the ask moves the target",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Ask = s.askWith("nudges:\n  - word: darker\n")

				var buf bytes.Buffer
				s.Require().NoError(Tune(context.Background(), &buf, opts))

				s.Require().Contains(buf.String(), `"darker" moves centroid`)
				s.Require().Contains(buf.String(), "of a tolerance")
			},
		},
		{
			// A first answer.
			name: "nothing said leaves the target alone",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Ask = s.askWith("")

				var buf bytes.Buffer
				s.Require().NoError(Tune(context.Background(), &buf, opts))

				s.Require().NotContains(buf.String(), "of a tolerance")
			},
		},
		{
			// Refusing rather than ignoring.
			//
			// Running on with the instruction dropped would report a tone
			// nobody asked for.
			name: "a word it cannot use stops the run",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Ask = s.askWith("nudges:\n  - word: chunky\n")

				err := Tune(context.Background(), buffer(), opts)

				s.Require().ErrorContains(err, "chunky")
			},
		},
		{
			// The path being wrong.
			name: "an ask that is not there is reported",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Ask = filepath.Join(s.T().TempDir(), "nowhere.tone.yaml")

				err := Tune(context.Background(), buffer(), opts)

				s.Require().ErrorContains(err, "nowhere.tone.yaml")
			},
		},
		{
			// Both halves of one word.
			//
			// "Punchier" is a tight low end and a hard attack. This target
			// constrains the low band and says nothing about the attack, so
			// one figure moves and the other is reported as free rather than
			// moved quietly or dropped in silence.
			//
			// It is also the comparative of a word ending in y, which
			// resolves back to "punchy" rather than being refused.
			name: "a word moves what it can and says what it cannot",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				opts := s.opts()
				opts.Ask = s.askWith("nudges:\n  - word: punchier\n")

				var buf bytes.Buffer
				s.Require().NoError(Tune(context.Background(), &buf, opts))

				s.Require().Contains(buf.String(), `"punchier" moves low`)
				s.Require().Contains(buf.String(), "which this genre leaves free")
			},
		},
		{
			// A file that is not a ToneSpec.
			name: "an ask it cannot read is reported",
			then: func() {
				s.ready()
				s.genre.EXPECT().
					MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

				at := filepath.Join(s.T().TempDir(), "ask.yaml")
				s.Require().NoError(os.WriteFile(at, []byte("schema: Setup\n"), 0o600))

				opts := s.opts()
				opts.Ask = at

				s.Require().Error(Tune(context.Background(), buffer(), opts))
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// buffer is somewhere for a run's report to go when nothing reads it.
func buffer() *bytes.Buffer { return &bytes.Buffer{} }

// sloping is a bench whose answer changes every time it is asked.
//
// Which is what makes a slope non-zero and so what makes the solver produce a
// move at all. The fake that answers identically is enough to test the
// arithmetic of arriving; nothing reaches the applying half without this.
type sloping struct {
	calls int
	// quietAfter is when it starts answering near-silence, which is what a
	// control turned to its bottom stop does to a chain.
	quietAfter int
}

func (b *sloping) Through(
	ctx context.Context,
	signal []float32,
) ([]float32, error) {
	b.calls++

	if b.quietAfter > 0 && b.calls > b.quietAfter {
		return bench{quiet: true}.Through(ctx, signal)
	}

	out := make([]float32, len(signal))
	hz := 110 + float64(b.calls)*7

	for i := range out {
		out[i] = float32(0.2 * math.Sin(2*math.Pi*hz*float64(i)/sdk.Rate))
	}

	return out, nil
}

func (*sloping) Name() string { return "a bench that answers differently" }

// TestLand covers land, which applies a pass's moves and backs off if they
// muted the chain.
//
// One method and one table, so a case is a row rather than a file.
func (s *TuneTestSuite) TestLand() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The guard on silence.
			//
			// Called directly, because reaching it through a whole run means
			// counting how many readings a chain of eleven dials takes and a
			// test that knows that breaks when a block gains a parameter.
			//
			// The failure it exists for reported the solver working. A first
			// pass reached one tolerance out partly by taking the amplifier's
			// Master from 1.0 to 0.037, and the pass after it read 643
			// tolerances out because every figure was computed on hiss. Level
			// is not an axis any corpus states, so nothing in the target
			// defends it, and two takes of silence agree to the last digit so
			// the noise floor cannot catch it either.
			name: "land backs off when the moves mute the chain",
			then: func() {
				tried := []float32{}
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ sdk.Address, to float32) error {
						tried = append(tried, to)

						return nil
					}).AnyTimes()

				knob := solve.Knob{Block: 0, Param: 6, Control: "Master", At: 1, Low: 0, High: 1}
				steps := []solve.Step{{Knob: knob, By: -0.96, To: 0.04}}

				var buf bytes.Buffer

				err := land(context.Background(), &buf, s.opts(), bench{quiet: true},
					make([]float32, 64), []solve.Knob{knob}, steps, 0)

				s.Require().ErrorIs(err, ErrNoTarget)
				s.Require().Contains(buf.String(), "muted the chain")

				s.Require().Len(tried, backoffs, "it halved its move and tried again")
				s.Require().InDelta(0.04, tried[0], 0.001, "the whole move first")
				s.Require().InDelta(0.52, tried[1], 0.001, "then half of it")
				s.Require().Greater(tried[2], tried[1], "and half again, from where it began")
			},
		},
		{
			// The ordinary pass.
			name: "land accepts moves that keep the chain audible",
			then: func() {
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).Times(1)

				knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}
				knobs := []solve.Knob{knob}
				steps := []solve.Step{{Knob: knob, By: 0.2, To: 0.6}}

				var buf bytes.Buffer

				s.Require().NoError(land(context.Background(), &buf, s.opts(), bench{},
					make([]float32, 64), knobs, steps, -200))

				s.Require().InDelta(0.6, knobs[0].At, 0.001,
					"and the knob remembers where it landed")
			},
		},
		{
			// The same, one layer up.
			name: "land reports a device that refuses",
			then: func() {
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("device refused the request"))

				knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

				var buf bytes.Buffer

				err := land(context.Background(), &buf, s.opts(), bench{},
					make([]float32, 64), []solve.Knob{knob},
					[]solve.Step{{Knob: knob, By: 0.2, To: 0.6}}, -200)

				s.Require().ErrorContains(err, "device refused")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// sloping is a bench whose answer changes every time it is asked.
//
// Which is what makes a slope non-zero and so what makes the solver produce a
// move at all. The fake that answers identically is enough to test the

// TestConverge covers converge, which runs the loop until the target is met
// or it stops improving.
//
// One method and one table, so a case is a row rather than a file.
func (s *TuneTestSuite) TestConverge() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The loop meeting its target and stopping.
			name: "converge arrives",
			then: func() {
				knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

				var buf bytes.Buffer

				// A tolerance wider than anything the bench can read, so the first pass is
				// already inside it.
				aims := map[audio.Figure]solve.Aim{
					audio.KeyCentroid: {Want: 0, Tol: 100000},
				}

				s.Require().NoError(second(converge(context.Background(), &buf, s.opts(),
					bench{}, make([]float32, 64), "preset.hlx", []solve.Knob{knob}, aims, -200)))

				s.Require().Contains(buf.String(), "pass 1")
				s.Require().NotContains(buf.String(), "pass 2", "it stopped when it arrived")
			},
		},
		{
			// A target out of reach.
			//
			// The bench's centroid climbs every reading and the target sits
			// below where it starts, so each pass is further out than the
			// last. Reported rather than run to the pass limit: a chain that
			// cannot reach a target says so, and the gear being wrong for the
			// sound is a real answer to somebody who owns that gear.
			name: "converge stops when it stops improving",
			then: func() {
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

				opts := s.opts()
				opts.Passes = 4

				var buf bytes.Buffer

				aims := map[audio.Figure]solve.Aim{
					audio.KeyCentroid: {Want: 10, Tol: 1},
				}

				s.Require().NoError(second(converge(context.Background(), &buf, opts,
					&sloping{}, make([]float32, 64), "preset.hlx", []solve.Knob{knob}, aims, -200)))

				s.Require().Contains(buf.String(), "stopped improving")
				s.Require().Contains(buf.String(), "will not reach this target")
			},
		},
		{
			// The other way it gives up.
			name: "converge runs out of passes",
			then: func() {
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

				opts := s.opts()
				opts.Passes = 1

				var buf bytes.Buffer

				aims := map[audio.Figure]solve.Aim{
					audio.KeyCentroid: {Want: 9000, Tol: 1},
				}

				s.Require().NoError(second(converge(context.Background(), &buf, opts,
					&sloping{}, make([]float32, 64), "preset.hlx", []solve.Knob{knob}, aims, -200)))

				s.Require().Contains(buf.String(), "1 passes and still")
			},
		},
		{
			// The audio loop going away mid-run.
			//
			// Four places read from it, and every one of them has to say so
			// rather than carry on with whatever the last reading was. A
			// campaign that hangs or invents a figure on block two hundred
			// looks exactly like one still working.
			name: "the bench failing is reported",
			then: func() {
				gone := errors.New("the device stopped answering")

				knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}
				aims := map[audio.Figure]solve.Aim{audio.KeyCentroid: {Want: 400, Tol: 5}}

				s.Run("measuring the noise floor", func() {
					s.ready()
					s.genre.EXPECT().
						MeasuredGenres(gomock.Any(), gomock.Any()).Return(s.punk(), nil)

					opts := s.opts()
					opts.Bench = bench{err: gone}

					s.Require().ErrorIs(Tune(context.Background(), buffer(), opts), gone)
				})

				s.Run("reading where the chain sits", func() {
					s.Require().ErrorIs(second(converge(context.Background(), buffer(),
						s.opts(), bench{err: gone}, make([]float32, 64), "preset.hlx",
						[]solve.Knob{knob}, aims, -200)), gone)
				})

				s.Run("reading a slope", func() {
					s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil).AnyTimes()

					s.Require().ErrorIs(slopes(context.Background(), bench{err: gone},
						make([]float32, 64), s.opts(), []solve.Knob{knob},
						map[audio.Figure]float64{audio.KeyCentroid: 100}), gone)
				})

				s.Run("checking whether a pass muted the chain", func() {
					s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil).AnyTimes()

					s.Require().ErrorIs(land(context.Background(), buffer(), s.opts(),
						bench{err: gone}, make([]float32, 64), []solve.Knob{knob},
						[]solve.Step{{Knob: knob, By: 0.2, To: 0.6}}, -200), gone)
				})
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// TestSlopes covers slopes, which reads what each control does from where the
// chain currently sits.
//
// One method and one table, so a case is a row rather than a file.
func (s *TuneTestSuite) TestSlopes() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A control already at its maximum.
			//
			// Nudged away from whichever stop it is against, because a
			// control at its top cannot be pushed further and would otherwise
			// read as having no slope at all.
			name: "slopes reads downward from a top stop",
			then: func() {
				var asked []float32

				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ sdk.Address, to float32) error {
						asked = append(asked, to)

						return nil
					}).Times(2)

				knobs := []solve.Knob{
					{Block: 0, Param: 6, Control: "Master", At: 1, Low: 0, High: 1},
				}

				s.Require().NoError(slopes(context.Background(), &sloping{},
					make([]float32, 64), s.opts(), knobs,
					map[audio.Figure]float64{audio.KeyCentroid: 100}))

				s.Require().Len(asked, 2)
				s.Require().Less(asked[0], float32(1), "nudged down, not up past its stop")
				s.Require().InDelta(1, asked[1], 0.0001, "and put back where it was")
				s.Require().NotZero(knobs[0].Slope[audio.KeyCentroid],
					"a control at its stop still has a slope")
			},
		},
		{
			// The pedal saying no mid-read.
			name: "slopes reports a device that refuses",
			then: func() {
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("no reply to opcode 30 within 6s"))

				knobs := []solve.Knob{
					{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1},
				}

				err := slopes(context.Background(), &sloping{}, make([]float32, 64),
					s.opts(), knobs, map[audio.Figure]float64{audio.KeyCentroid: 100})

				s.Require().ErrorContains(err, "opcode 30")
			},
		},
		{
			// The second move failing.
			//
			// A control left where a slope was read is a control the next
			// reading is taken through, so failing to put one back skews
			// every figure after it.
			name: "slopes reports a control it cannot put back",
			then: func() {
				first := s.pedal.EXPECT().
					Turn(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("device refused the request")).After(first)

				knobs := []solve.Knob{
					{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1},
				}

				err := slopes(context.Background(), &sloping{}, make([]float32, 64),
					s.opts(), knobs, map[audio.Figure]float64{audio.KeyCentroid: 100})

				s.Require().ErrorContains(err, "device refused")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// TestApplyReportsADeviceThatRefuses covers a move the pedal will not take.
func (s *TuneTestSuite) TestApplyReportsADeviceThatRefuses() {
	s.pedal.EXPECT().Turn(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("device refused the request"))

	knob := solve.Knob{Block: 0, Param: 1, Control: "Bass", At: 0.4, Low: 0, High: 1}

	err := apply(context.Background(), s.opts(), []solve.Knob{knob},
		[]solve.Step{{Knob: knob, By: 0.2, To: 0.6}})

	s.Require().ErrorContains(err, "device refused")
}

// second is the error from a call that also answers what it did, for a case
// that is only asserting the error.
func second(
	_ round,
	err error,
) error {
	return err
}

func TestTuneTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TuneTestSuite))
}

// askWith writes a document carrying a nudges block and returns where it went.
//
// The nudges are the ask's, so they are indented with the rest of it.
func (s *TuneTestSuite) askWith(
	nudges string,
) string {
	at := filepath.Join(s.T().TempDir(), "ask.yaml")

	s.Require().NoError(os.WriteFile(at, []byte(`schema: ToneSpec
id: matt-freeman
ask:
  subject:
    kind: artist
    name: Matt Freeman
  instrument: bass
  genre: [punk]
`+nest(nudges)+`rig:
  instrument: bass
  chain:
    - {role: amp, gear: Ampeg SVT}
`), 0o600))

	return at
}

// nest moves an ask's own text under `ask:`. A blank line stays blank.
func nest(
	body string,
) string {
	if body == "" {
		return ""
	}

	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "  " + line
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

// TestKeepTuned covers every case keepTuned answers.
//
// One method and one table, so a case is a row rather than a file.
func (s *TuneTestSuite) TestKeepTuned() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Which artifact --out produces.
			//
			// Both are on the reading, so writing either is one line, and the
			// choice is the point. A rig's settings are seven words shared
			// across every make of amplifier; the positions this loop just
			// solved for are device parameters at exact values, and only the
			// plan has anywhere to put them. Writing the rig would export the
			// chain and throw away the tuning, which is what the first
			// version of this did.
			name: "keep writes a plan rather than a rig",
			then: func() {
				s.pedal.EXPECT().Current(gomock.Any(), sdk.FormatRig).Return(sdk.Reading{
					Name: "matt-freeman",
					ID:   "matt-freeman",
					Plan: plan.Plan{Blocks: []plan.Block{{
						Model:   catalog.ModelID("HD2_AmpSVBeastBrt"),
						Pos:     0,
						Enabled: true,
						Params:  plan.Params{"Master": catalog.Float(0.62)},
					}}},
				}, nil)

				opts := s.opts()
				opts.Out = filepath.Join(s.T().TempDir(), "tuned.yaml")

				var buf bytes.Buffer

				s.Require().NoError(
					keepTuned(context.Background(), &buf, opts, "as-built.hlx"))

				f, err := os.Open(opts.Out)
				s.Require().NoError(err)

				defer func() { _ = f.Close() }()

				got, err := plan.Load(f)
				s.Require().NoError(err)

				s.Require().Len(got.Blocks, 1)
				at, ok := got.Blocks[0].Params["Master"].Float()
				s.Require().True(ok)
				s.Require().InDelta(0.62, at, 0.0001,
					"the knob the solve landed on has to survive being written")
			},
		},
		{
			// The measuring rig not leaking into the answer.
			//
			// The chain that was tuned had been sent to USB alone and turned
			// down 30dB so it would stop feeding itself down the measuring
			// lead. Keeping the preset as the device holds it writes both of
			// those into the plan, and the rig somebody then compiles is
			// silent at the quarter-inch socket and 30dB quiet everywhere
			// else.
			//
			// Neither is anything the solver decided, so neither survives.
			// The dials do.
			name: "keep puts the output back to what was compiled",
			then: func() {
				// What the device is playing: off the loop at USB alone, turned down.
				measuring := map[string]json.RawMessage{
					outputSlot: json.RawMessage(
						`{"@model":"HelixStomp_AppDSPFlowOutputMain",` +
							`"@output":10,"pan":0.5,"gain":-30}`),
				}

				s.pedal.EXPECT().Current(gomock.Any(), sdk.FormatRig).Return(sdk.Reading{
					Plan: plan.Plan{
						Blocks: []plan.Block{{
							Model: catalog.ModelID("HD2_AmpSVBeastBrt"), Pos: 0, Enabled: true,
							Params: plan.Params{"Master": catalog.Float(0.62)},
						}},
						Device: &rig.DeviceState{Routing: &measuring},
					},
				}, nil)

				// What the compiler produced is answered by the suite's own PresetFile,
				// which returns routed(): the entry a preset arrives with, on destination 1
				// at unity. That is what a kept plan should carry.

				opts := s.opts()
				opts.Out = filepath.Join(s.T().TempDir(), "tuned.yaml")

				var buf bytes.Buffer

				s.Require().NoError(
					keepTuned(context.Background(), &buf, opts, "as-built.hlx"))

				f, err := os.Open(opts.Out)
				s.Require().NoError(err)

				defer func() { _ = f.Close() }()

				got, err := plan.Load(f)
				s.Require().NoError(err)

				var fields map[string]any
				s.Require().NoError(
					json.Unmarshal((*got.Device.Routing)[outputSlot], &fields))

				s.Require().InDelta(1, fields["@output"], 0.001,
					"back on the destination the rig compiled to")
				s.Require().InDelta(0, fields["gain"], 0.001,
					"and without the measuring headroom")

				at, ok := got.Blocks[0].Params["Master"].Float()
				s.Require().True(ok)
				s.Require().InDelta(0.62, at, 0.0001, "the solve's own answer is kept")
			},
		},
		{
			// The empty --out.
			//
			// The message named a rig while the file was a plan, which is the
			// one place somebody decides what they are about to get.
			name: "keep says which artifact it would have written",
			then: func() {
				var buf bytes.Buffer

				s.Require().NoError(
					keepTuned(context.Background(), &buf, s.opts(), "as-built.hlx"))

				s.Require().Contains(buf.String(), "--out writes it as a plan")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}
