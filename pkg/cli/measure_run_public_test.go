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
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/cli/internal/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
)

// MeasureRunTestSuite covers a campaign end to end, with a pedal that answers
// from memory and a bench that hands back a tone.
type MeasureRunTestSuite struct {
	suite.Suite

	ctrl  *gomock.Controller
	pedal *mocks.MockPedal
	out   string
	dry   string
}

func (s *MeasureRunTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.pedal = mocks.NewMockPedal(s.ctrl)
	s.out = filepath.Join(s.T().TempDir(), "measured.json")
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di.wav")
}

// bench hands back a tone loud enough to measure.
type bench struct {
	quiet   bool
	clipped bool
	err     error
}

func (b bench) Through(
	_ context.Context,
	signal []float32,
) ([]float32, error) {
	if b.err != nil {
		return nil, b.err
	}

	amplitude := 0.2
	if b.quiet {
		amplitude = 0.000001
	}

	// Driven well past the ceiling and clamped there, which is what clipping
	// is: flat tops, an RMS close to one, and harmonics that were never in
	// the signal.
	if b.clipped {
		amplitude = 8
	}

	out := make([]float32, len(signal))
	for i := range out {
		v := amplitude * math.Sin(2*math.Pi*110*float64(i)/sdk.Rate)
		out[i] = float32(math.Max(-1, math.Min(1, v)))
	}

	return out, nil
}

func (bench) Name() string { return "a bench" }

// compiles makes the pedal build any preset it is asked for.
func (s *MeasureRunTestSuite) compiles() {
	s.pedal.EXPECT().
		Compile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.Compile) (sdk.Built, error) {
			s.Require().NoError(os.WriteFile(in.Out, []byte("a preset"), 0o600))

			return sdk.Built{}, nil
		}).AnyTimes()
}

// TestBlocksMeasuresACategory covers a campaign that works.
func (s *MeasureRunTestSuite) TestBlocksMeasuresACategory() {
	s.compiles()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	var buf bytes.Buffer

	s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
		Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
		Seconds: 1, Bench: bench{},
	}))

	lib := s.read()

	s.Require().Len(lib.Blocks, 8)
	s.Require().True(lib.Isolated)
	s.Require().NotEmpty(lib.Reference.SHA256)
	s.Require().Positive(lib.Baseline.Centroid,
		"the empty loop is measured first, or every block is a difference "+
			"from nothing")

	for _, block := range lib.Blocks {
		s.Require().Empty(block.Refused)
		s.Require().False(block.Clipped)
	}
}

// TestBlocksMarksWhatClipped covers a reading at the converters' ceiling.
//
// A clipped spectrum is the clipping's rather than the block's, so it reads
// as a bright block and would sit at the top of a list of bright blocks.
func (s *MeasureRunTestSuite) TestBlocksMarksWhatClipped() {
	s.compiles()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	var buf bytes.Buffer

	s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
		Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
		Seconds: 1, Bench: bench{clipped: true},
	}))

	for _, block := range s.read().Blocks {
		s.Require().True(block.Clipped, "%s was not marked", block.ID)
		s.Require().False(block.Measured(), "a clipped reading is not usable")
	}

	s.Require().Contains(buf.String(), "CLIPPED")
}

// TestBlocksRecordsWhatWouldNotLoad covers a device declining a preset.
//
// A refusal is a result rather than an end. Some of what the catalog lists
// means nothing on its own, and a campaign has to get past them to reach the
// rest.
func (s *MeasureRunTestSuite) TestBlocksRecordsWhatWouldNotLoad() {
	s.compiles()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
		Return(nil)
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
		Return(errors.New("the device stopped taking the message")).AnyTimes()

	var buf bytes.Buffer

	s.Require().NoError(MeasureBlocks(context.Background(), &buf, MeasureOptions{
		Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
		Seconds: 1, Bench: bench{},
	}))

	for _, block := range s.read().Blocks {
		s.Require().NotEmpty(block.Refused)
	}

	s.Require().Contains(buf.String(), "refused")
}

// TestBlocksWritesAsItGoes is why a long run survives a pedal that drops off.
func (s *MeasureRunTestSuite) TestBlocksWritesAsItGoes() {
	s.compiles()

	// The baseline, then one block, then the bench stops answering.
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	var (
		buf   bytes.Buffer
		calls int
	)

	err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
		Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq", Seconds: 1,
		Bench: stopping{after: 3, calls: &calls},
	})

	s.Require().Error(err)

	lib := s.read()
	s.Require().NotEmpty(lib.Blocks,
		"what was measured before it stopped is on disk")
}

// TestBlocksReportsAReferenceItCannotRead covers a missing signal.
func (s *MeasureRunTestSuite) TestBlocksReportsAReferenceItCannotRead() {
	var buf bytes.Buffer

	err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
		Client: s.pedal, Dry: "nowhere.wav", Out: s.out, Seconds: 1,
		Bench: bench{},
	})

	s.Require().ErrorContains(err, "nowhere.wav")
}

// TestBlocksReportsABaselineItCannotTake covers the empty loop failing.
//
// Nothing below it would mean anything, so it stops rather than measuring six
// hundred blocks against nothing.
func (s *MeasureRunTestSuite) TestBlocksReportsABaselineItCannotTake() {
	s.compiles()
	s.pedal.EXPECT().Play(gomock.Any(), gomock.Any()).
		Return(errors.New("no")).AnyTimes()

	var buf bytes.Buffer

	err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
		Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
		Seconds: 1, Bench: bench{},
	})

	s.Require().ErrorContains(err, "loading the empty loop")
}

// TestBlocksReportsAPresetItCannotBuild covers the compiler failing.
func (s *MeasureRunTestSuite) TestBlocksReportsAPresetItCannotBuild() {
	s.pedal.EXPECT().Compile(gomock.Any(), gomock.Any()).
		Return(sdk.Built{}, errors.New("will not build")).AnyTimes()

	var buf bytes.Buffer

	err := MeasureBlocks(context.Background(), &buf, MeasureOptions{
		Client: s.pedal, Dry: s.dry, Out: s.out, Category: "eq",
		Seconds: 1, Bench: bench{},
	})

	s.Require().ErrorContains(err, "building the empty loop")
}

// read is the library a run wrote.
func (s *MeasureRunTestSuite) read() measured.Library {
	f, err := os.Open(s.out)
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	lib, err := measured.Load(f)
	s.Require().NoError(err)

	return lib
}

// turning is a bench that answers normally for a while and then differently.
//
// The noise floor is measured first, from the chain as it sits. A bench that
// was quiet or clipped from the very first reading would make that the
// settled level too, and nothing afterwards would stand out from it.
type turning struct {
	after int
	to    bench
	calls *int
}

func (b turning) Through(
	ctx context.Context,
	signal []float32,
) ([]float32, error) {
	*b.calls++

	if *b.calls > b.after {
		return b.to.Through(ctx, signal)
	}

	return bench{}.Through(ctx, signal)
}

func (turning) Name() string { return "a bench that changes" }

// stopping is a bench that answers a few times and then does not.
type stopping struct {
	after int
	calls *int
}

func (b stopping) Through(
	ctx context.Context,
	signal []float32,
) ([]float32, error) {
	*b.calls++

	if *b.calls > b.after {
		return nil, errors.New("stopped answering")
	}

	return bench{}.Through(ctx, signal)
}

func (stopping) Name() string { return "a bench that stops" }

func TestMeasureRunTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasureRunTestSuite))
}
