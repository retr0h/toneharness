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
package deviceslots_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/toneharness/pkg/sdk/internal/deviceslots"
	flowmocks "github.com/retr0h/toneharness/pkg/sdk/internal/deviceslots/mocks"
)

// PlayPublicTestSuite covers putting a preset in front of a device without
// storing it.
//
// The distinction from Import is the reason it exists, and it is a hardware
// one: a slot is flash, and a burst of flash writes has taken a setlist past
// what a power cycle could clear. Nothing here may reach a Writer.
type PlayPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

// playable is a session that can replace what a device is playing.
type playable struct {
	*mocks.MockEditor
	*mocks.MockLoaded
}

func (*playable) Close() error { return nil }

func (s *PlayPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *PlayPublicTestSuite) dev() *playable {
	ed := mocks.NewMockEditor(s.ctrl)
	// What the document is built for is checked against what answered, so a
	// session that names no device cannot be written to.
	ed.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()

	return &playable{
		MockEditor: ed,
		MockLoaded: mocks.NewMockLoaded(s.ctrl),
	}
}

// preset is a real one, because Play rebuilds the document a device expects
// and a made-up file would never get as far as the call being tested.
func (s *PlayPublicTestSuite) preset() string {
	return filepath.Join("testdata", "hx-stomp.written.hlx")
}

// TestPlay covers Play, which puts a preset in front of the device without
// storing it anywhere.
//
// One method and one table, so a case is a row rather than a file.
func (s *PlayPublicTestSuite) TestPlay() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary case.
			//
			// The document handed to the device is the one an Import would
			// have written. A preset is seeked through by a table of byte
			// offsets, so one built any other way is accepted and then
			// rendered as an empty chain.
			name: "play replaces what is playing",
			then: func() {
				ctx := context.Background()
				d := s.dev()

				d.MockLoaded.EXPECT().
					WriteCurrent(ctx, gomock.Any()).
					DoAndReturn(func(_ context.Context, body []byte) error {
						s.Require().NotEmpty(body)

						return nil
					})

				s.Require().NoError((&deviceslots.Flows{}).Play(ctx, d, s.preset()))
			},
		},
		{
			// The write failing.
			name: "play reports a device that refused",
			then: func() {
				ctx := context.Background()
				d := s.dev()

				d.MockLoaded.EXPECT().
					WriteCurrent(ctx, gomock.Any()).
					Return(errors.New("usb: gone"))

				err := (&deviceslots.Flows{}).Play(ctx, d, s.preset())

				s.Require().ErrorContains(err, "replacing what is playing")
			},
		},
		{
			// A path that is not a preset.
			name: "play reports a file it cannot read",
			then: func() {
				err := (&deviceslots.Flows{}).Play(
					context.Background(), s.dev(), filepath.Join("testdata", "nope.hlx"))

				s.Require().Error(err)
			},
		},
		{
			// The document failing to build.
			//
			// Nothing reaches the device in that case, which is the part
			// worth holding: a half-built document put in front of a pedal
			// renders as an empty chain.
			name: "play reports a catalog it cannot read",
			then: func() {
				c := flowmocks.NewMockCatalogs(s.ctrl)
				c.EXPECT().Catalog(gomock.Any()).Return(nil, errors.New("no catalog"))

				err := (&deviceslots.Flows{Catalogs: c}).Play(
					context.Background(), s.dev(), s.preset())

				s.Require().ErrorContains(err, "no catalog")
			},
		},
		{
			// The write nothing else guards.
			//
			// Which catalog names the gear is a static option decided before
			// any handshake, and every one of the four models this package
			// recognises opens a session just as readily. A chain resolved
			// against the Stomp's catalog and spliced into a Stomp-shaped
			// blank, written to a Floor, is a document the device accepts and
			// then draws as empty.
			name: "play refuses a catalog for another pedal",
			then: func() {
				d := &playable{
					MockEditor: mocks.NewMockEditor(s.ctrl),
					MockLoaded: mocks.NewMockLoaded(s.ctrl),
				}
				d.MockEditor.EXPECT().Model().
					Return(device.Model{Name: "Helix Floor"}).AnyTimes()

				c := flowmocks.NewMockCatalogs(s.ctrl)
				c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
					func(context.Context) (*catalog.Catalog, error) { return catalog.BuiltIn() },
				)

				err := (&deviceslots.Flows{Catalogs: c}).Play(
					context.Background(), d, s.preset())

				s.Require().ErrorIs(err, deviceslots.ErrWrongDevice)
				s.Require().ErrorContains(err, "Helix Floor")
				s.Require().ErrorContains(err, "HX Stomp")
			},
		},
		{
			// A session without the capability.
			name: "play needs a session that can replace",
			then: func() {
				err := (&deviceslots.Flows{}).Play(
					context.Background(),
					mocks.NewMockEditor(s.ctrl),
					s.preset(),
				)

				s.Require().ErrorContains(err, "cannot replace what is playing")
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

func TestPlayPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlayPublicTestSuite))
}
