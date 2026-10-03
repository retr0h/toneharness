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

package sdk_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/toneharness/pkg/sdk/internal/wire"
	"github.com/retr0h/toneharness/pkg/sdk/slot"
)

type ClientPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *ClientPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// cancelled is a context whose caller has already stopped waiting.
func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

// bus stands in for the one thing this library needs hardware for.
func (s *ClientPublicTestSuite) bus(
	descs []device.Descriptor,
	err error,
) *mocks.MockOpener {
	b := mocks.NewMockOpener(s.ctrl)
	b.EXPECT().List(gomock.Any()).Return(descs, err).AnyTimes()

	return b
}

// writable is a session that can both read and write.
type writable struct {
	*mocks.MockEditor
	*mocks.MockWriter
}

// pedal is a bus whose one device holds a real HX Stomp preset in every slot,
// and takes whatever is written to it.
func (s *ClientPublicTestSuite) pedal() *mocks.MockOpener {
	body, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	dev := &writable{
		MockEditor: mocks.NewMockEditor(s.ctrl),
		MockWriter: mocks.NewMockWriter(s.ctrl),
	}
	dev.MockEditor.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return([]wire.Preset{
		{Slot: 0, Name: "Chunky Monkey"},
		{Slot: 1, Name: "Longview"},
	}, nil).AnyTimes()
	dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, gomock.Any()).
		Return(body, nil).AnyTimes()
	dev.MockEditor.EXPECT().Close().AnyTimes()
	dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	b := mocks.NewMockOpener(s.ctrl)
	b.EXPECT().Open(gomock.Any()).Return(dev, nil).AnyTimes()

	return b
}

// TestNew covers building a Client.
func (s *ClientPublicTestSuite) TestNew() {
	tests := []struct {
		name string
		opts []sdk.Option
	}{
		{
			// The common case is somebody who wants the built-in catalog,
			// the built-in statistics and whatever is plugged in.
			name: "with nothing said about it",
		},
		{
			// Nothing is opened, so even settings naming nothing real
			// build a Client. What they name is found out on use.
			name: "with every setting",
			opts: []sdk.Option{
				sdk.WithCatalog("no.json"),
				sdk.WithStats("no.json.gz"),
				sdk.WithRigs("no-such-directory"),
				sdk.WithBackupDir("no-such-directory"),
				sdk.WithCapture(io.Discard),
				sdk.WithTrace(io.Discard),
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().NotNil(sdk.New(tt.opts...))
		})
	}
}

// TestBlocks covers which models the client answers with, and which
// catalog they come from.
//
// One method and one table, so a case is a row rather than a file.
func (s *ClientPublicTestSuite) TestBlocks() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The two ways a client is told which models exist, and which one
			// wins.
			//
			// One method and one table, so a case is a row rather than a
			// file.
			name: "which catalog the blocks come from",
			then: func() {
				for _, tt := range []struct {
					name string
					then func()
				}{
					{
						// WithCatalog, which reads a generated catalog instead of the
						// built-in one.
						//
						// One method and one table, so a case is a row rather than a
						// file.
						name: "with catalog",
						then: func() {
							for _, tt := range []struct {
								name string
								then func()
							}{
								{
									// Naming a catalog other than the built-in one.
									name: "with catalog",
									then: func() {
										builtIn, err := sdk.New().Blocks(context.Background(), sdk.Filter{})
										s.Require().NoError(err)

										tests := []struct {
											name string
											path string
											err  bool
										}{
											{name: "a catalog of its own", path: fixture("catalog.json")},
											{name: "a catalog that is not there", path: "no.json", err: true},
										}

										for _, tt := range tests {
											s.Run(tt.name, func() {
												got, err := sdk.New(sdk.WithCatalog(tt.path)).
													Blocks(context.Background(), sdk.Filter{})

												if tt.err {
													s.Require().Error(err)

													return
												}

												s.Require().NoError(err)
												s.Require().NotEqual(builtIn.Total, got.Total,
													"the Client read the catalog it was given")
											})
										}
									},
								},
								{
									// Which of the two wins.
									//
									// A catalog somebody generated themselves is a stronger statement
									// than the name of a device this binary happens to carry. Nothing
									// else would notice this being reversed.
									name: "with catalog beats with device",
									then: func() {
										floor, err := sdk.New(sdk.WithDevice("Helix Floor")).
											Blocks(context.Background(), sdk.Filter{})
										s.Require().NoError(err)

										got, err := sdk.New(
											sdk.WithCatalog(fixture("catalog.json")),
											sdk.WithDevice("Helix Floor"),
										).Blocks(context.Background(), sdk.Filter{})
										s.Require().NoError(err)

										s.Require().NotEqual(floor.Total, got.Total,
											"the catalog somebody named is the one that was read")
									},
								},
							} {
								s.Run(tt.name, func() {
									// A row gets the same fresh state a method used to get.
									s.SetupTest()

									tt.then()
								})
							}
						},
					},
					{
						// Using the built-in catalog for another pedal.
						//
						// Checked against the name the listing carries rather than how
						// many blocks it holds. A Helix LT carries the same 661 as an HX
						// Stomp, so a count would pass for the wrong reason on the one
						// case most worth pinning down.
						name: "with device",
						then: func() {
							tests := []struct {
								name   string
								device string
								want   string
								err    bool
							}{
								{
									// The common case, and the device everything here was written
									// against.
									name: "nothing said about it", want: "HX Stomp",
								},
								{name: "a device that ships", device: "Helix Floor", want: "Helix Floor"},
								{name: "loosely matched", device: "helix-lt", want: "Helix LT"},
								{name: "a device nothing ships for", device: "Kemper", err: true},
							}

							for _, tt := range tests {
								s.Run(tt.name, func() {
									got, err := sdk.New(sdk.WithDevice(tt.device)).
										Blocks(context.Background(), sdk.Filter{})

									if tt.err {
										s.Require().Error(err)
										// The useful half of the message: naming what it does carry.
										s.Require().Contains(err.Error(), "HX Stomp")

										return
									}

									s.Require().NoError(err)
									s.Require().Equal(tt.want, got.Device)
								})
							}
						},
					},
				} {
					s.Run(tt.name, func() {
						// A row gets the same fresh state a method used to get.
						s.SetupTest()

						tt.then()
					})
				}
			},
		},
		{
			// Reporting what a device can do.
			name: "blocks",
			then: func() {
				tests := []struct {
					name    string
					ctx     context.Context
					filter  sdk.Filter
					matched bool
					err     bool
				}{
					{
						name:    "the catalog in the binary",
						filter:  sdk.Filter{Search: "klon"},
						matched: true,
					},
					{
						name:   "a filter nothing matches",
						filter: sdk.Filter{Category: "no such category"},
					},
					{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						ctx := tt.ctx
						if ctx == nil {
							ctx = context.Background()
						}

						got, err := sdk.New().Blocks(ctx, tt.filter)

						if tt.err {
							s.Require().Error(err)

							return
						}

						s.Require().NoError(err)
						s.Require().NotZero(got.Total)

						if tt.matched {
							s.Require().NotEmpty(got.Matched)

							return
						}

						s.Require().Empty(got.Matched)
					})
				}
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

// TestWithStats covers naming statistics other than the built-in ones.
func (s *ClientPublicTestSuite) TestWithStats() {
	_, err := sdk.New(sdk.WithStats("no.json.gz")).
		ChainMeasurements(context.Background(), "")

	s.Require().Error(err, "the Client read the statistics it was given")
}

// TestWithSetup covers naming what the person has, so a build fits them.
//
// A client option rather than a call's argument, for the reason the Setup is a
// separate document: it changes on a different clock from a request. What this
// asserts is that the Client read the file it was given, which a path nothing
// is at says loudly enough.
func (s *ClientPublicTestSuite) TestWithSetup() {
	_, err := sdk.New(sdk.WithSetup("no-such-setup.yaml")).
		Make(context.Background(), sdk.Build{
			RigID: "mike-dirnt",
			Out:   filepath.Join(s.T().TempDir(), "o.hlx"),
		})

	s.Require().ErrorContains(err, "no-such-setup.yaml",
		"the Client read the Setup it was given")
}

// TestWithRigs covers naming a directory of rigs.
func (s *ClientPublicTestSuite) TestWithRigs() {
	got, err := sdk.New(sdk.WithRigs("no-such-directory")).
		Rigs(context.Background())

	s.Require().NoError(err)
	s.Require().Equal("no-such-directory", got.Dir)
	s.Require().Empty(got.Rigs)
}

// ownRig is a document of somebody's own, about "Their Player". extra is a line
// for the ask, such as aliases or extends, or empty.
func ownRig(
	stem string,
	id string,
	extra string,
	instrument string,
) map[string]string {
	return map[string]string{
		stem + ".yaml": "schema: ToneSpec\nid: " + id + `

ask:
  genre: [rock]
  ` + extra + `
  subject:
    kind: artist
    name: Their Player

  confidence: high

rig:
  instrument: ` + instrument + `

  chain:
    - role: amp
      gear: Aguilar DB51
      evidence:
        - { kind: cited, note: "a test says so" }
      confidence: high
`,
	}
}

// rigsDir writes files under artists/ in a new directory, and returns it.
//
// A case hands over whatever ownRig produced rather than naming its files one
// at a time.
func (s *ClientPublicTestSuite) rigsDir(
	files map[string]string,
) string {
	dir := filepath.Join(s.T().TempDir(), "rigs")
	s.Require().NoError(os.MkdirAll(filepath.Join(dir, "artists"), 0o750))

	for name, body := range files {
		s.Require().NoError(os.WriteFile(
			filepath.Join(dir, "artists", name), []byte(body), 0o600))
	}

	return dir
}

// TestWithUserRigs covers WithUserRigs, which layers somebody's own directory
// of rigs over the ones that ship, or over the directory WithRigs named.
//
// One method and one table, so a case is a row rather than a file.
func (s *ClientPublicTestSuite) TestWithUserRigs() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Somebody's own rigs layered over the ones that ship: what Rigs
			// lists, and what Rig and Build find.
			name: "with user rigs",
			then: func() {
				tests := []struct {
					name  string
					files map[string]string
					// missing names a directory that is not there.
					missing bool
					// locked leaves the directory unreadable.
					locked bool
					id     string
					// want is the subject Rig finds, which its ask carries.
					want    string
					variant string
					listErr string
					findErr string
				}{
					{
						name:  "a rig of theirs over a shipped one",
						files: ownRig("mine", "mike-dirnt", "", "bass"),
						id:    "mike-dirnt",
						want:  "Their Player",
					},
					{
						name: "an alias of theirs that is a shipped rig's alias",
						// Aliases are the ask's: another name for what somebody wanted.
						files: ownRig("mine", "their-player", "aliases: [DIRNT]", "bass"),
						id:    "mike-dirnt",
						want:  "Their Player",
					},
					{
						name: "a variant of theirs on a shipped rig",
						// What a rig departs from is the ask's too.
						files:   ownRig("mine", "mike-dirnt-live", "extends: mike-dirnt", "bass"),
						id:      "mike-dirnt",
						want:    "Mike Dirnt",
						variant: "mike-dirnt-live",
					},
					{
						name:    "a directory that cannot be read",
						locked:  true,
						id:      "mike-dirnt",
						listErr: "reading",
						findErr: "reading",
					},
					{
						name:    "a directory that is not there",
						missing: true,
						id:      "mike-dirnt",
						want:    "Mike Dirnt",
					},
					{
						name:    "a file of theirs that is not a rig",
						files:   map[string]string{"broken.yaml": "schema: ToneSpec\nid: broken\n"},
						id:      "mike-dirnt",
						want:    "Mike Dirnt",
						listErr: "broken.yaml",
					},
					{
						name:    "a file of theirs that is not a rig, named for the one asked for",
						files:   map[string]string{"mike-dirnt.yaml": "schema: Setup\n"},
						id:      "mike-dirnt",
						listErr: "mike-dirnt.yaml",
						findErr: "mike-dirnt.yaml",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						if tt.locked && os.Geteuid() == 0 {
							s.T().Skip("root reads a directory whatever its mode")
						}

						dir := s.rigsDir(tt.files)

						if tt.missing {
							dir = filepath.Join(dir, "not-there")
						}

						if tt.locked {
							s.Require().NoError(os.Chmod(dir, 0o000))
							s.T().Cleanup(func() { _ = os.Chmod(dir, 0o750) })
						}

						ctx := context.Background()
						client := sdk.New(sdk.WithUserRigs(dir))

						listed, listErr := client.Rigs(ctx)
						shown, showErr := client.Rig(ctx, tt.id)
						out := filepath.Join(s.T().TempDir(), "out.hlx")
						_, buildErr := client.Make(ctx, sdk.Build{RigID: tt.id, Out: out})

						if tt.listErr != "" {
							s.Require().ErrorContains(listErr, tt.listErr)
						} else {
							s.Require().NoError(listErr)
							s.Require().Equal(dir, listed.Dir)

							listedIDs := make([]string, 0, len(listed.Rigs))
							for _, r := range listed.Rigs {
								listedIDs = append(listedIDs, r.ID)
							}

							s.Require().Contains(listedIDs, "flea", "the shipped rigs are still listed")
						}

						if tt.findErr != "" {
							s.Require().ErrorContains(showErr, tt.findErr)
							s.Require().ErrorContains(buildErr, tt.findErr)

							return
						}

						s.Require().NoError(showErr)
						s.Require().NoError(buildErr)
						s.Require().NotNil(shown.Ask, "every rig here has an ask beside it")
						s.Require().NotNil(shown.Ask.Subject)
						s.Require().Equal(tt.want, shown.Ask.Subject.Name)
						s.Require().FileExists(out)

						if tt.variant != "" {
							s.Require().Len(shown.Variants, 1)
							s.Require().Equal(tt.variant, shown.Variants[0].ID)
						}
					})
				}
			},
		},
		{
			// Where Scaffold and Extend write when a Client has a directory
			// of somebody's own, and what Extend copies from.
			name: "user rigs are written to",
			then: func() {
				ctx := context.Background()
				user := s.rigsDir(nil)
				beneath := s.rigsDir(ownRig("guitarist", "guitarist", "", "guitar"))

				tests := []struct {
					name string
					do   func(c *sdk.Client) (sdk.Scaffolded, error)
					opts []sdk.Option
					// instrument is what the report says the new rig is played on.
					instrument string
					// from is the rig the report says it was copied from, and empty for
					// a rig scaffolded from gear, which was copied from nothing.
					from string
				}{
					{
						name: "a rig scaffolded from gear",
						opts: []sdk.Option{sdk.WithUserRigs(user), sdk.WithRigs(beneath)},
						do: func(c *sdk.Client) (sdk.Scaffolded, error) {
							return c.Scaffold(ctx, sdk.NewRig{
								Genre: []string{"rock"},
								ID:    "scaffolded", Name: "Somebody", Instrument: "bass", Amp: "Ampeg SVT",
							})
						},
						instrument: "bass",
					},
					{
						// The copied rig's instrument, since a copy names none.
						name: "a copy of a shipped rig",
						opts: []sdk.Option{sdk.WithUserRigs(user)},
						do: func(c *sdk.Client) (sdk.Scaffolded, error) {
							return c.Extend(ctx, sdk.ExtendRig{From: "mike-dirnt", ID: "copied"})
						},
						instrument: "bass",
						from:       "mike-dirnt",
					},
					{
						name: "a copy of a rig in the directory beneath theirs",
						opts: []sdk.Option{sdk.WithUserRigs(user), sdk.WithRigs(beneath)},
						do: func(c *sdk.Client) (sdk.Scaffolded, error) {
							return c.Extend(ctx, sdk.ExtendRig{From: "guitarist", ID: "copied-guitar"})
						},
						instrument: "guitar",
						from:       "guitarist",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						got, err := tt.do(sdk.New(tt.opts...))
						s.Require().NoError(err)
						s.Require().Equal(user, filepath.Dir(filepath.Dir(got.Path)))
						s.Require().Equal(tt.instrument, got.Instrument)
						s.Require().Equal(tt.from, got.From)
						s.Require().Equal(tt.from != "", got.Copied())
					})
				}
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

// TestWithBackupDir covers where a device slot's old contents go.
func (s *ClientPublicTestSuite) TestWithBackupDir() {
	tests := []struct {
		name string
		dir  func() string
		err  bool
	}{
		{
			name: "a directory it can write",
			dir:  func() string { return s.T().TempDir() },
		},
		{
			// A write whose backup failed does not happen.
			name: "somewhere nothing can be kept",
			dir: func() string {
				path := filepath.Join(s.T().TempDir(), "a-file")
				s.Require().NoError(os.WriteFile(path, nil, 0o600))

				return path
			},
			err: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := tt.dir()

			got, err := sdk.New(sdk.WithDevices(s.pedal()), sdk.WithBackupDir(dir)).
				Copy(context.Background(), slot.Address{}, slot.Address{Slot: 1})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got.Kept, 1)
			s.Require().Equal(dir, filepath.Dir(got.Kept[0]))
		})
	}
}

// TestWithCapture covers keeping what a device answered.
func (s *ClientPublicTestSuite) TestWithCapture() {
	body, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	tests := []struct {
		name  string
		asked bool
	}{
		{name: "when asked", asked: true},
		{name: "when nobody asked"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var kept bytes.Buffer

			opts := []sdk.Option{sdk.WithDevices(s.pedal())}
			if tt.asked {
				opts = append(opts, sdk.WithCapture(&kept))
			}

			_, err := sdk.New(opts...).Preset(context.Background(), slot.Address{})
			s.Require().NoError(err)

			if !tt.asked {
				s.Require().Zero(kept.Len())

				return
			}

			s.Require().Equal(body, kept.Bytes(), "kept verbatim")
		})
	}
}

// TestWithTrace covers where the frames go.
//
// A double sends no frames, so this asks which bus New built: the USB one,
// handed the trace. device's own tests show a session writes its frames there.
func (s *ClientPublicTestSuite) TestWithTrace() {
	var trace bytes.Buffer

	tests := []struct {
		name  string
		opts  []sdk.Option
		trace io.Writer
	}{
		{name: "asked for", opts: []sdk.Option{sdk.WithTrace(&trace)}, trace: &trace},
		{name: "not asked for"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(device.NewUSB(tt.trace), sdk.New(tt.opts...).Opener())
		})
	}
}

// TestCatalog covers the catalog a Client names gear against.
func (s *ClientPublicTestSuite) TestCatalog() {
	tests := []struct {
		name string
		path string
		ctx  context.Context
		err  bool
	}{
		{name: "the catalog in the binary"},
		{name: "a catalog of its own", path: fixture("catalog.json")},
		{name: "a catalog that is not there", path: "no.json", err: true},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			client := sdk.New(sdk.WithCatalog(tt.path))

			got, err := client.Catalog(ctx)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			// Opened once and kept, so a renderer reads what the operation did.
			again, err := client.Catalog(ctx)
			s.Require().NoError(err)
			s.Require().Same(got, again)
		})
	}
}

// TestDevices covers reporting what is attached.
func (s *ClientPublicTestSuite) TestDevices() {
	stomp := device.Descriptor{Vendor: 0x0e41, Product: 0x4246, Bus: 20, Address: 3}

	tests := []struct {
		name  string
		bus   *mocks.MockOpener
		want  int
		first string
		err   bool
	}{
		{
			name:  "a device this project knows",
			bus:   s.bus([]device.Descriptor{stomp}, nil),
			want:  1,
			first: "HX Stomp",
		},
		{
			// A bus holds keyboards and webcams. Those are not an answer to
			// what a preset can be written to.
			name: "somebody else's hardware",
			bus:  s.bus([]device.Descriptor{{Vendor: 0x05ac, Product: 0x1234}}, nil),
		},
		{name: "nothing attached", bus: s.bus(nil, nil)},
		{
			name: "a bus that will not answer",
			bus:  s.bus(nil, errors.New("bus unavailable")),
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			found, err := sdk.New(sdk.WithDevices(tt.bus)).Devices(context.Background())

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(found.Devices, tt.want)

			if tt.first != "" {
				s.Require().Equal(tt.first, found.Devices[0].Model)
			}
		})
	}
}

// TestBlock covers reporting one block and what it accepts.
func (s *ClientPublicTestSuite) TestBlock() {
	tests := []struct {
		name    string
		ctx     context.Context
		catalog string
		id      string
		is      error
		err     bool
	}{
		{name: "a block the catalog carries", id: "HD2_DistMinotaur"},
		{name: "one it does not", id: "HD2_NoSuchBlock", is: sdk.ErrNoSuchBlock},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			id:   "HD2_DistMinotaur",
			is:   context.Canceled,
		},
		{
			name:    "a catalog that is not there",
			catalog: "no.json",
			id:      "HD2_DistMinotaur",
			err:     true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New(sdk.WithCatalog(tt.catalog)).Block(ctx, tt.id)

			if tt.err {
				s.Require().Error(err)

				return
			}

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, string(got.ID))
		})
	}
}

// TestModelMeasurements covers how players set one model.
func (s *ClientPublicTestSuite) TestModelMeasurements() {
	tests := []struct {
		name  string
		ctx   context.Context
		model string
		err   bool
	}{
		{name: "one model's distributions", model: "HD2_DistMinotaur"},
		{name: "a model nobody measured", model: "HD2_NoSuchModel", err: true},
		{
			// Refused rather than answered with the grammar of a chain,
			// which is a different question with a method of its own.
			name: "no model at all",
			err:  true,
		},
		{
			name:  "a caller who stopped waiting",
			ctx:   cancelled(),
			model: "HD2_DistMinotaur",
			err:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().ModelMeasurements(ctx, tt.model)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(got.Stats)
			s.Require().True(got.AboutOne())
			s.Require().Equal(tt.model, string(got.Model))
		})
	}
}

// TestChainMeasurements covers what chains tend to hold.
func (s *ClientPublicTestSuite) TestChainMeasurements() {
	tests := []struct {
		name       string
		ctx        context.Context
		instrument string
		err        bool
	}{
		{name: "every chain"},
		{name: "one instrument's chains", instrument: "bass"},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().ChainMeasurements(ctx, tt.instrument)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(got.Stats)
			s.Require().False(got.AboutOne())
			s.Require().Equal(tt.instrument, got.Instrument)
		})
	}
}

// TestRigs covers reading the rigs that ship with this library.
func (s *ClientPublicTestSuite) TestRigs() {
	tests := []struct {
		name string
		ctx  context.Context
		err  bool
	}{
		{
			// No directory means the rigs that ship, which is the case for
			// anyone who has not written their own.
			name: "the rigs that ship",
		},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Rigs(ctx)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(got.Rigs)
		})
	}
}

// TestBacking covers reading which records back each rig.
//
// The two halves live in different files, a rig and a manifest, and only
// reading them together says whether a rig's evidence was measured from
// records made when its gear was.
func (s *ClientPublicTestSuite) TestBacking() {
	tests := []struct {
		name   string
		ctx    context.Context
		corpus string
		err    bool
	}{
		{
			// The rigs that ship, against a corpus directory nobody has: a
			// rig with no records is the ordinary case, not a fault.
			name:   "rigs nobody has measured",
			corpus: filepath.Join("testdata", "nowhere"),
		},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Backing(ctx, tt.corpus)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(got, "the rigs this binary ships")

			for _, b := range got {
				s.Require().Empty(b.Records)
			}
		})
	}
}

// TestRig covers reading one of them.
func (s *ClientPublicTestSuite) TestRig() {
	tests := []struct {
		name string
		ctx  context.Context
		id   string
		is   error
	}{
		{name: "a rig that ships", id: "mike-dirnt"},
		{name: "one nobody wrote", id: "nobody-at-all", is: sdk.ErrNoSuchRig},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			id:   "mike-dirnt",
			is:   context.Canceled,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Rig(ctx, tt.id)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, got.ID)
		})
	}
}

// TestTone covers resolving a request into the rig it describes.
//
// The operation the whole project is for, and it lived in pkg/cli until this
// session, which made it unreachable over MCP: an agent calls a tool rather
// than a command.
func (s *ClientPublicTestSuite) TestTone() {
	examples := func(name string) string {
		return filepath.Join("..", "..", "marketplace", "core", "examples", name)
	}

	tests := []struct {
		name string
		ctx  context.Context
		in   sdk.Ask
		is   error
		// gear is a piece the resolved chain must name, for the cases where
		// which chain came back is the point.
		gear  string
		notes bool
	}{
		{
			name: "a request and what somebody owns",
			in: sdk.Ask{
				Spec:  examples("like-a-record.yaml"),
				Setup: examples("mine.setup.yaml"),
			},
			notes: true,
		},
		{
			// A setup is optional: somebody asking what a record sounds like
			// has not necessarily said what is in the room.
			name:  "a request on its own",
			in:    sdk.Ask{Spec: examples("like-a-record.yaml")},
			notes: true,
		},
		{
			name: "an ask that is not there",
			in:   sdk.Ask{Spec: filepath.Join(s.T().TempDir(), "nowhere.yaml")},
			is:   fs.ErrNotExist,
		},
		{
			// Naming a player reaches the rig somebody researched for them,
			// which is the lookup the Client hands to translate. The shipped
			// rigs rather than a double, because that is what it hands over.
			name:  "a request naming a player",
			in:    sdk.Ask{Spec: examples("like-a-player.yaml")},
			gear:  "Ampeg SVT",
			notes: true,
		},
		{
			// Nobody has researched him and nothing else in the ask names gear,
			// so there is nothing to take a chain from. A genre nobody has
			// measured either, because a measured one is a target: punk here
			// would choose an amplifier and there would be nothing to refuse.
			name: "a request naming a player nobody has researched",
			in: sdk.Ask{Spec: s.spec(`schema: ToneSpec
id: cone-mccaslin
ask:
  genre: [rock]
  like:
    artist: Cone McCaslin
rig:
  instrument: bass
  chain:
    - {role: amp, gear: Ampeg SVT}
`)},
			is: sdk.ErrNothingToBuildFrom,
		},
		{
			// The same player in a genre somebody has measured. His rig is still
			// not written down, and pop-punk's records name a target, so an
			// amplifier is chosen by measurement rather than nothing being
			// chosen at all.
			name: "a player nobody has researched, in a measured genre",
			in: sdk.Ask{Spec: s.spec(`schema: ToneSpec
id: cone-mccaslin
ask:
  genre: [pop-punk]
  instrument: bass
  like:
    artist: Cone McCaslin
rig:
  instrument: bass
  chain:
    - {role: amp, gear: Ampeg SVT}
`)},
			notes: true,
		},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			in:   sdk.Ask{Spec: examples("like-a-record.yaml")},
			is:   context.Canceled,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Tone(ctx, tt.in)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(got.Rig.Chain)

			if tt.gear != "" {
				var named string
				for _, entry := range got.Rig.Chain {
					named += entry.Gear + "\n"
				}

				s.Require().Contains(named, tt.gear)
			}

			if tt.notes {
				s.Require().NotEmpty(got.Notes,
					"a resolution says what it made of the request")
			}
		})
	}
}

// spec writes a document and returns where it went, for a case shorter than a
// file of its own is worth.
//
// The `rig:` section is what the contract requires and not what a resolve reads:
// the ask is the request and Translate builds the gear that answers it, so what
// is already there is the previous answer and is replaced.
func (s *ClientPublicTestSuite) spec(
	body string,
) string {
	at := filepath.Join(s.T().TempDir(), "ask.yaml")
	s.Require().NoError(os.WriteFile(at, []byte(body), 0o600))

	return at
}

// TestScaffold covers writing a rig, gear checked first.
func (s *ClientPublicTestSuite) TestScaffold() {
	scaffold := func(ctx context.Context, in sdk.NewRig, opts ...sdk.Option) (scaffolded, error) {
		got, err := sdk.New(opts...).Scaffold(ctx, in)
		if err != nil {
			return scaffolded{}, err
		}

		body, err := os.ReadFile(got.Path)
		s.Require().NoError(err)

		return scaffolded{
			got: got, body: string(body),
		}, nil
	}

	// Each row writes into a directory of its own, because a scaffold is
	// never written over a rig already there.
	into := func() sdk.Option { return sdk.WithRigs(s.T().TempDir()) }

	base := sdk.NewRig{
		Genre: []string{"rock"},
		ID:    "test-player", Name: "Test Player",
		Instrument: "bass", Amp: "Ampeg SVT",
	}

	written, err := scaffold(context.Background(), base, into())
	s.Require().NoError(err)

	// with is the base rig with one field changed.
	with := func(change func(*sdk.NewRig)) sdk.NewRig {
		in := base
		change(&in)

		return in
	}

	tests := []struct {
		name string
		ctx  context.Context
		in   sdk.NewRig
		// nowhere means the Client was given no directory of rigs.
		nowhere bool
		// check is what the written rig must say that the base rig does
		// not, for a row that changes one field.
		check func(got scaffolded)
		err   bool
	}{
		{
			name: "a rig naming gear this device models",
			in:   base,
			check: func(got scaffolded) {
				s.Require().Equal(base.ID, got.got.ID)
				s.Require().Equal(written.body, got.body)
			},
		},
		{
			name: "Name decides who the rig is about",
			in:   with(func(in *sdk.NewRig) { in.Name = "Other Player" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "    name: Other Player")
				s.Require().Contains(written.body, "    name: Test Player")
			},
		},
		{
			name: "Band decides the group the rig names",
			in:   with(func(in *sdk.NewRig) { in.Band = "The Test Band" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "    band: The Test Band")
				s.Require().NotContains(written.body, "band:")
			},
		},
		{
			name: "Instrument decides which instrument the rig is for",
			in:   with(func(in *sdk.NewRig) { in.Instrument = "guitar" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "instrument: guitar")
				s.Require().Contains(written.body, "instrument: bass")
			},
		},
		{
			name: "Amp decides the amplifier in the chain",
			in:   with(func(in *sdk.NewRig) { in.Amp = "Acoustic 360" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "role: amp\n      gear: Acoustic 360")
				s.Require().Contains(written.body, "role: amp\n      gear: Ampeg SVT")
			},
		},
		{
			name: "Cab decides the cabinet in the chain",
			in:   with(func(in *sdk.NewRig) { in.Cab = "Ampeg 8x10" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "role: cab\n      gear: Ampeg 8x10")
				s.Require().NotContains(written.body, "role: cab")
			},
		},
		{
			name: "Pedals decide what goes ahead of the amp",
			in:   with(func(in *sdk.NewRig) { in.Pedals = []string{"Klon"} }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "role: drive\n      gear: Klon")
				s.Require().NotContains(written.body, "role: drive")
			},
		},
		{
			// Checked first, because a rig naming gear no device models is
			// otherwise only found out when somebody builds from it.
			name: "one naming gear nothing emulates",
			in: sdk.NewRig{
				Genre: []string{"rock"},
				ID:    "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "No Such Amplifier",
			},
			err: true,
		},
		{
			name: "an identifier that will not do",
			in:   sdk.NewRig{Genre: []string{"rock"}, ID: "Not An ID", Amp: "Ampeg SVT"},
			err:  true,
		},
		{
			// Refused rather than written wherever the program happened to
			// run.
			name: "a Client given nowhere to write",
			in: sdk.NewRig{
				Genre: []string{"rock"},
				ID:    "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "Ampeg SVT",
			},
			nowhere: true,
			err:     true,
		},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			in:   sdk.NewRig{Genre: []string{"rock"}, ID: "test-player", Amp: "Ampeg SVT"},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			var opts []sdk.Option
			if !tt.nowhere {
				opts = append(opts, into())
			}

			got, err := scaffold(ctx, tt.in, opts...)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			tt.check(got)
		})
	}
}

// scaffolded is what a scaffold answered and the rig it wrote.
type scaffolded struct {
	got sdk.Scaffolded
	// body is the whole document: the ask saying who the rig is for and the
	// gear that answers it. One file since version 2, so a claim about either
	// half is a claim about this text.
	body string
}

// extended is what an extend answered and the document it wrote.
type extended struct {
	got  sdk.Scaffolded
	body string
}

// TestExtend covers starting a rig as a copy of another.
//
// ExtendRig is an input struct, so each of its fields has a row of its own
// that changes that field alone and shows the rig written changing with it. A
// field no path reads would leave its row identical to the baseline.
func (s *ClientPublicTestSuite) TestExtend() {
	extend := func(ctx context.Context, in sdk.ExtendRig, opts ...sdk.Option) (extended, error) {
		got, err := sdk.New(opts...).Extend(ctx, in)
		if err != nil {
			return extended{}, err
		}

		body, err := os.ReadFile(got.Path)
		s.Require().NoError(err)

		return extended{
			got: got, body: string(body),
		}, nil
	}

	// Each row writes into a directory of its own, because a copy is never
	// written over a rig already there.
	into := func() sdk.Option { return sdk.WithRigs(s.T().TempDir()) }

	base, err := extend(
		context.Background(),
		sdk.ExtendRig{From: "mike-dirnt", ID: "the-copy"},
		into(),
	)
	s.Require().NoError(err)

	tests := []struct {
		name string
		ctx  context.Context
		in   sdk.ExtendRig
		// nowhere means the Client was given no directory of rigs.
		nowhere bool
		check   func(got extended)
		is      error
		err     bool
	}{
		{
			name: "From decides which rig is copied",
			in:   sdk.ExtendRig{From: "flea", ID: "the-copy"},
			check: func(got extended) {
				s.Require().Contains(got.body, "extends: flea")
				s.Require().Contains(base.body, "extends: mike-dirnt")
				s.Require().NotEqual(base.body, got.body)
				// The report names what the copy holds, which is the rig
				// it copied: its name and its amp.
				s.Require().Equal("Flea", got.got.Name)
				s.Require().Equal("Gallien-Krueger 2001RB", got.got.Amp)
				// The rig it was copied from, which is what tells a copy
				// from a scaffold after the fact.
				s.Require().Equal("flea", got.got.From)
				s.Require().True(got.got.Copied())
				s.Require().Equal("Mike Dirnt", base.got.Name)
				s.Require().Equal("Ampeg SVT", base.got.Amp)
			},
		},
		{
			name: "ID decides what the copy is called and where it is written",
			in:   sdk.ExtendRig{From: "mike-dirnt", ID: "another-copy"},
			check: func(got extended) {
				s.Require().Equal("another-copy", got.got.ID)
				s.Require().Equal("another-copy.yaml", filepath.Base(got.got.Path))
				s.Require().Contains(got.body, "id: another-copy")
				s.Require().NotContains(base.body, "id: another-copy")
			},
		},
		{
			name: "Name decides who the copy is about",
			in:   sdk.ExtendRig{From: "mike-dirnt", ID: "the-copy", Name: "Somebody Else"},
			check: func(got extended) {
				s.Require().Contains(got.body, "    name: Somebody Else")
				s.Require().Contains(base.body, "    name: Mike Dirnt")
				s.Require().Equal("Somebody Else", got.got.Name)
			},
		},
		{
			name: "Kind decides what the copy is attributed to",
			in:   sdk.ExtendRig{From: "mike-dirnt", ID: "the-copy", Kind: "song"},
			check: func(got extended) {
				s.Require().Contains(got.body, "    kind: song")
				s.Require().Contains(base.body, "    kind: artist")
			},
		},
		{
			name: "a rig to copy that nobody wrote",
			in:   sdk.ExtendRig{From: "nobody-at-all", ID: "the-copy"},
			is:   sdk.ErrNoSuchRig,
		},
		{
			// Without one this would scaffold a rig from no gear at all.
			name: "no rig to copy",
			in:   sdk.ExtendRig{ID: "the-copy"},
			is:   sdk.ErrNoSuchRig,
		},
		{
			// Refused rather than written wherever the program happened to
			// run.
			name:    "a Client given nowhere to write",
			in:      sdk.ExtendRig{From: "mike-dirnt", ID: "the-copy"},
			nowhere: true,
			err:     true,
		},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			in:   sdk.ExtendRig{From: "mike-dirnt", ID: "the-copy"},
			is:   context.Canceled,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			var opts []sdk.Option
			if !tt.nowhere {
				opts = append(opts, into())
			}

			got, err := extend(ctx, tt.in, opts...)

			switch {
			case tt.is != nil:
				s.Require().ErrorIs(err, tt.is)

				return
			case tt.err:
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			tt.check(got)
		})
	}
}

// TestBuild covers compiling a rig into a preset.
func (s *ClientPublicTestSuite) TestBuild() {
	tests := []struct {
		name string
		ctx  context.Context
		id   string
		// a file somebody already has at the path.
		taken    bool
		existing sdk.Existing
		err      bool
		is       error
	}{
		{name: "a rig that ships", id: "mike-dirnt"},
		{name: "one nobody wrote", id: "nobody-at-all", err: true},
		{name: "a caller who stopped waiting", ctx: cancelled(), id: "mike-dirnt", err: true},
		{
			name:  "a file already there, replaced",
			id:    "mike-dirnt",
			taken: true,
		},
		{
			// The write refuses it, so nothing has to look first.
			name:     "a file already there, kept",
			id:       "mike-dirnt",
			taken:    true,
			existing: sdk.KeepExisting,
			is:       fs.ErrExist,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			out := filepath.Join(s.T().TempDir(), "out.hlx")

			if tt.taken {
				s.Require().NoError(os.WriteFile(out, []byte("somebody's preset"), 0o600))
			}

			got, err := sdk.New().
				Make(ctx, sdk.Build{RigID: tt.id, Out: out, Existing: tt.existing})

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				body, readErr := os.ReadFile(out) //nolint:gosec // a path this test chose
				s.Require().NoError(readErr)
				s.Require().Equal("somebody's preset", string(body))

				return
			}

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().NotEmpty(got.Plan.Blocks)
		})
	}
}

func TestClientPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(ClientPublicTestSuite))
}

// TestMusic covers the four questions the music corpus answers.
//
// One test for all four because they share everything but the call: the same
// tree, the same read of the same manifests, and four views over it. Separate
// tests would assert the same setup four times.
// TestMusicNeedsTheRigsToAnswerWhoHasNone covers the join failing.
//
// A corpus listing says which of its players nothing can be built for, so it
// reads the rigs as well as the recordings and a rig set it cannot read is a
// listing it cannot answer. Reported rather than shrugged off: a listing that
// quietly said nobody had gear would be worse than one that stopped, because
// "no rig" is exactly the answer somebody is looking for.
func (s *ClientPublicTestSuite) TestMusicNeedsTheRigsToAnswerWhoHasNone() {
	c := sdk.New(sdk.WithRigs(s.rigsDir(
		map[string]string{"broken.yaml": "schema: ToneSpec\nid: broken\n"})))

	corpus := s.T().TempDir()

	tests := []struct {
		name  string
		check func() error
	}{
		{
			name:  "who it holds records for",
			check: func() error { _, err := c.MusicPlayers(s.T().Context(), corpus); return err },
		},
		{
			name:  "what it can be selected by",
			check: func() error { _, err := c.MusicGenres(s.T().Context(), corpus); return err },
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().ErrorContains(tt.check(), "broken.yaml")
		})
	}
}

func (s *ClientPublicTestSuite) TestMusic() {
	root := s.T().TempDir()

	dir := filepath.Join(root, "bass", "mike-dirnt")
	s.Require().NoError(os.MkdirAll(dir, 0o750))
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "corpus.yaml"),
		[]byte("artist: Mike Dirnt\ntracks:\n  - track: longview\n"+
			"    url: https://open.spotify.com/track/x\n    year: 1994\n"+
			"    band: Green Day\n    genres: [punk]\n    genres_by: llm\n"), 0o600))

	tests := []struct {
		name  string
		check func(ctx context.Context, c *sdk.Client) error
	}{
		{
			name: "who it holds records for",
			check: func(ctx context.Context, c *sdk.Client) error {
				got, err := c.MusicPlayers(ctx, root)
				if err != nil {
					return err
				}

				s.Require().Len(got, 1)
				s.Require().Equal("mike-dirnt", got[0].ID)
				s.Require().Equal("bass", got[0].Instrument)
				s.Require().Equal([]string{"punk"}, got[0].Genres)

				return nil
			},
		},
		{
			name: "which bands made them",
			check: func(ctx context.Context, c *sdk.Client) error {
				got, err := c.MusicBands(ctx, root)
				if err != nil {
					return err
				}

				s.Require().Len(got, 1)
				s.Require().Equal("Green Day", got[0].Name)
				s.Require().Zero(got[0].Unsighted, "nobody labels a band")

				return nil
			},
		},
		{
			name: "which genres, and how far off usable",
			check: func(ctx context.Context, c *sdk.Client) error {
				got, err := c.MusicGenres(ctx, root)
				if err != nil {
					return err
				}

				s.Require().Len(got, 1)
				s.Require().Equal("punk", got[0].Slug)
				s.Require().False(got[0].Usable, "one record from one player")
				s.Require().Equal(7, got[0].ShortRecords)
				s.Require().Equal(2, got[0].ShortArtists)
				s.Require().Equal(1, got[0].Unsighted, "a model tagged it")

				return nil
			},
		},
		{
			name: "every record, and whether it has stems",
			check: func(ctx context.Context, c *sdk.Client) error {
				got, err := c.MusicRecords(ctx, root)
				if err != nil {
					return err
				}

				s.Require().Len(got, 1)
				s.Require().Equal("longview", got[0].Track)
				s.Require().Equal("llm", got[0].DecidedBy)
				s.Require().False(got[0].Separated, "nothing separated it")

				return nil
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().NoError(tt.check(context.Background(), sdk.New()))
		})

		// The same call refuses a caller who stopped waiting, before it reads
		// anything off disk.
		s.Run(tt.name+", cancelled", func() {
			s.Require().Error(tt.check(cancelled(), sdk.New()))
		})
	}
}

// TestMusicRefusesATreeWithNoManifests covers a path pointing at nothing.
//
// Refused rather than answered empty, because the ordinary cause is a wrong
// path and an empty table reads as a corpus that holds nothing.
func (s *ClientPublicTestSuite) TestMusicRefusesATreeWithNoManifests() {
	_, err := sdk.New().MusicPlayers(context.Background(), s.T().TempDir())
	s.Require().Error(err)
}

// TestMeasuredGenres covers every case MeasuredGenres answers.
//
// One method and one table, so a case is a row rather than a file.
func (s *ClientPublicTestSuite) TestMeasuredGenres() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Measuring the audio rather than counting manifests.
			//
			// The slow half: it reads the recordings, where MusicGenres reads
			// what somebody wrote down. Against a tree this test builds, so
			// the assertion does not depend on whichever records somebody has
			// on disk.
			name: "measured genres",
			then: func() {
				root := s.T().TempDir()

				dir := filepath.Join(root, "bass", "a")
				s.Require().NoError(os.MkdirAll(dir, 0o750))
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "corpus.yaml"),
					[]byte("artist: A\ntracks:\n  - track: t\n"+
						"    url: https://open.spotify.com/track/x\n    year: 1994\n"+
						"    genres: [punk]\n    genres_by: llm\n"), 0o600))

				// A manifest naming a record nothing separated, which is a genre measured
				// from nothing rather than an error.
				got, err := sdk.New().MeasuredGenres(context.Background(), root)
				s.Require().NoError(err)
				s.Require().Empty(got, "no stems, so nothing measured")

				_, err = sdk.New().MeasuredGenres(cancelled(), root)
				s.Require().Error(err, "a caller who stopped waiting")
			},
		},
		{
			// A path nobody can walk.
			name: "measured genres refuses a tree that is not there",
			then: func() {
				_, err := sdk.New().MeasuredGenres(
					context.Background(), filepath.Join("testdata", "nowhere"))
				s.Require().Error(err)
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

// TestMeasuredPlayers covers MeasuredPlayers, which is what each player's
// records measure as, and the words that earns them against the others.
//
// One method and one table, so a case is a row rather than a file.
func (s *ClientPublicTestSuite) TestMeasuredPlayers() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Measuring each player's records.
			//
			// The slow half of the pair, the way MeasuredGenres is: this
			// reads the audio where MusicPlayers reads what somebody wrote
			// down. A word is earned by sitting clear of the other players,
			// so one player alone earns nothing and an empty answer is the
			// ordinary result rather than a fault.
			name: "measured players",
			then: func() {
				root := s.T().TempDir()

				dir := filepath.Join(root, "a")
				s.Require().NoError(os.MkdirAll(dir, 0o750))
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "corpus.yaml"),
					[]byte("artist: A\ntracks:\n  - track: t\n"+
						"    url: https://open.spotify.com/track/x\n    year: 1994\n"), 0o600))

				// A manifest naming a record nothing separated, so there is a player and
				// no audio behind them.
				got, err := sdk.New().MeasuredPlayers(context.Background(), root)
				s.Require().NoError(err)
				s.Require().Empty(got, "no stems, so nothing measured")

				_, err = sdk.New().MeasuredPlayers(cancelled(), root)
				s.Require().Error(err, "a caller who stopped waiting")
			},
		},
		{
			// A path nobody can walk.
			name: "measured players refuses a tree that is not there",
			then: func() {
				_, err := sdk.New().MeasuredPlayers(
					context.Background(), filepath.Join("testdata", "nowhere"))
				s.Require().Error(err)
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

// TestMeasuredRecordings covers MeasuredRecordings, which is what a directory
// of recordings measures as, one entry per file and the figures they make
// together.
//
// One method and one table, so a case is a row rather than a file.
func (s *ClientPublicTestSuite) TestMeasuredRecordings() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Measuring a directory of separated audio.
			//
			// A directory of files rather than a corpus tree, and no
			// manifest: this is what somebody points at the output of a
			// separation run before any of it has been filed under a player.
			name: "measured recordings",
			then: func() {
				empty := s.T().TempDir()

				tracks, together, err := sdk.New().MeasuredRecordings(context.Background(), empty)
				s.Require().NoError(err)
				s.Require().Empty(tracks, "a directory holding no audio measures nothing")
				s.Require().Zero(together.Tracks)

				_, _, err = sdk.New().MeasuredRecordings(cancelled(), empty)
				s.Require().Error(err, "a caller who stopped waiting")
			},
		},
		{
			// A wrong path.
			name: "measured recordings refuses a directory that is not there",
			then: func() {
				_, _, err := sdk.New().MeasuredRecordings(
					context.Background(), filepath.Join("testdata", "nowhere"))
				s.Require().Error(err)
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

// TestControlAddressesABlockOnePastItsPosition covers the one place the
// arithmetic between a preset's numbering and the wire's lives.
//
// A device keeps the input at grid 0, so a block a preset records at position P
// answers to P+1. Both the CLI and the MCP server did this themselves for a
// while, which is two copies of one fact and the shape of bug where one gets
// corrected and the other does not.
func (s *ClientPublicTestSuite) TestControlAddressesABlockOnePastItsPosition() {
	s.Require().Equal(
		sdk.Address{Block: 1, Param: 0, Direct: true},
		sdk.Control(0, 0),
		"the first block a preset records is 1 on the wire, because 0 is the input")

	s.Require().Equal(
		sdk.Address{Block: 5, Param: 3, Direct: true},
		sdk.Control(4, 3),
		"measured on an HX Stomp: addressing 5 moved the block recorded at 4")
}
