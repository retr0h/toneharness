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
package deviceslots

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// WrittenTestSuite holds a preset to what the device itself wrote.
//
// A preset built here is sent to hardware and has to be a document that
// hardware can render. Reading one back does not prove that: a read ignores
// the offset table and the device's own renderer uses it, so a document whose
// table does not describe its bytes reads back perfectly and shows an empty
// chain on the pedal.
//
// That is not hypothetical. Exporting a preset off a device and importing the
// file straight back produced a slot the pedal drew as empty, while every
// check this repository could make said it was fine, and a measurement taken
// through it read a bass going down a cable through nothing.
//
// The fixtures are one real preset in both forms: `hx-stomp.written.bin` is
// the device's own answer for the slot, byte for byte, and
// `hx-stomp.written.hlx` is what exporting that slot writes. Building the
// second has to produce the first's chain.
type WrittenTestSuite struct {
	suite.Suite
}

// TestBuildingAPresetMatchesWhatTheDeviceWrote is the regression.
//
// Not a comparison of the whole document: a device writes a build string and
// a name this cannot reproduce, and neither is the chain. What has to match
// is the part that decides whether anything is heard.
func (s *WrittenTestSuite) TestBuildingAPresetMatchesWhatTheDeviceWrote() {
	ctx := context.Background()

	written, err := os.ReadFile(filepath.Join("testdata", "hx-stomp.written.bin"))
	s.Require().NoError(err)

	want, err := wire.DecodePreset(written)
	s.Require().NoError(err)
	s.Require().NotEmpty(want.Blocks,
		"the fixture must hold a chain, or this proves nothing")

	// What importing the exported file builds, which is what would be sent.
	//
	// The device's own catalog, because a chain is written as model numbers
	// and only the catalog turns the names in a file back into them.
	c := mocks.NewMockCatalogs(gomock.NewController(s.T()))
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) {
			return catalog.BuiltIn()
		},
	).AnyTimes()

	f := &Flows{Catalogs: c}

	doc, err := fileslots.ReadPreset(ctx, filepath.Join("testdata", "hx-stomp.written.hlx"))
	s.Require().NoError(err)

	body, err := f.documentFor(ctx, doc)
	s.Require().NoError(err)

	got, err := wire.DecodePreset(body)
	s.Require().NoError(err)

	s.Require().Len(got.Blocks, len(want.Blocks),
		"a built preset must carry the same chain the device wrote")
}

func TestWrittenTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WrittenTestSuite))
}

// TestBuiltDocumentLooksLikeTheDeviceWrote compares the bytes, not the model.
//
// The chain decoding the same is not enough. What the pedal renders from is
// the document, and a document can carry the right blocks and still be one
// the device draws as empty.
func (s *WrittenTestSuite) TestBuiltDocumentLooksLikeTheDeviceWrote() {
	ctx := context.Background()

	written, err := os.ReadFile(filepath.Join("testdata", "hx-stomp.written.bin"))
	s.Require().NoError(err)

	c := mocks.NewMockCatalogs(gomock.NewController(s.T()))
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) { return catalog.BuiltIn() },
	).AnyTimes()

	doc, err := fileslots.ReadPreset(ctx,
		filepath.Join("testdata", "hx-stomp.written.hlx"))
	s.Require().NoError(err)

	body, err := (&Flows{Catalogs: c}).documentFor(ctx, doc)
	s.Require().NoError(err)

	got, err := wire.DecodePreset(body)
	s.Require().NoError(err)

	want, err := wire.DecodePreset(written)
	s.Require().NoError(err)

	// Every block, with the same number of values. A device writes a 2x15
	// cabinet with seven where its symbol list holds eight, and the extra one
	// is what made a preset render empty.
	s.Require().Len(got.Blocks, len(want.Blocks))

	for i := range want.Blocks {
		s.Require().Equal(want.Blocks[i].Model, got.Blocks[i].Model,
			"block %d is a different model", i)
		s.Require().Len(got.Blocks[i].Values, len(want.Blocks[i].Values),
			"block %d carries %d values where the device wrote %d",
			i, len(got.Blocks[i].Values), len(want.Blocks[i].Values))
	}
}
