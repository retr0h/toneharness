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
	return &playable{
		MockEditor: mocks.NewMockEditor(s.ctrl),
		MockLoaded: mocks.NewMockLoaded(s.ctrl),
	}
}

// preset is a real one, because Play rebuilds the document a device expects
// and a made-up file would never get as far as the call being tested.
func (s *PlayPublicTestSuite) preset() string {
	return filepath.Join("testdata", "hx-stomp.written.hlx")
}

// TestPlayReplacesWhatIsPlaying covers the ordinary case.
//
// The document handed to the device is the one an Import would have written.
// A preset is seeked through by a table of byte offsets, so one built any
// other way is accepted and then rendered as an empty chain.
func (s *PlayPublicTestSuite) TestPlayReplacesWhatIsPlaying() {
	ctx := context.Background()
	d := s.dev()

	d.MockLoaded.EXPECT().
		WriteCurrent(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, body []byte) error {
			s.Require().NotEmpty(body)

			return nil
		})

	s.Require().NoError((&deviceslots.Flows{}).Play(ctx, d, s.preset()))
}

// TestPlayReportsADeviceThatRefused covers the write failing.
func (s *PlayPublicTestSuite) TestPlayReportsADeviceThatRefused() {
	ctx := context.Background()
	d := s.dev()

	d.MockLoaded.EXPECT().
		WriteCurrent(ctx, gomock.Any()).
		Return(errors.New("usb: gone"))

	err := (&deviceslots.Flows{}).Play(ctx, d, s.preset())

	s.Require().ErrorContains(err, "replacing what is playing")
}

// TestPlayReportsAFileItCannotRead covers a path that is not a preset.
func (s *PlayPublicTestSuite) TestPlayReportsAFileItCannotRead() {
	err := (&deviceslots.Flows{}).Play(
		context.Background(), s.dev(), filepath.Join("testdata", "nope.hlx"))

	s.Require().Error(err)
}

// TestPlayReportsACatalogItCannotRead covers the document failing to build.
//
// Nothing reaches the device in that case, which is the part worth holding:
// a half-built document put in front of a pedal renders as an empty chain.
func (s *PlayPublicTestSuite) TestPlayReportsACatalogItCannotRead() {
	c := flowmocks.NewMockCatalogs(s.ctrl)
	c.EXPECT().Catalog(gomock.Any()).Return(nil, errors.New("no catalog"))

	err := (&deviceslots.Flows{Catalogs: c}).Play(
		context.Background(), s.dev(), s.preset())

	s.Require().ErrorContains(err, "no catalog")
}

// TestPlayNeedsASessionThatCanReplace covers a session without the capability.
func (s *PlayPublicTestSuite) TestPlayNeedsASessionThatCanReplace() {
	err := (&deviceslots.Flows{}).Play(
		context.Background(),
		mocks.NewMockEditor(s.ctrl),
		s.preset(),
	)

	s.Require().ErrorContains(err, "cannot replace what is playing")
}

func TestPlayPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlayPublicTestSuite))
}
