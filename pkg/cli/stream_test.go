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
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk"
)

// StreamTestSuite covers waiting for the audio stream to stop settling.
//
// The readings a stream hands back while it is settling are not of the chain,
// and there were two of them on the rig this was written against: a baseline
// taken as the middle of three put itself on one of the two, and the same loop
// read 129.5, 170.8 and 154.1 Hz across three runs minutes apart.
type StreamTestSuite struct {
	suite.Suite
}

// levels is a bench answering a stated level per reading, in order, and the
// last one for every reading after those.
//
// The amplitude is what sets the level, so the figures come back from the same
// arithmetic every other double here goes through rather than from a struct
// this filled in.
type levels struct {
	at    []float64
	calls *int
}

func (b levels) Through(
	_ context.Context,
	signal []float32,
) ([]float32, error) {
	i := *b.calls
	*b.calls++

	if i >= len(b.at) {
		i = len(b.at) - 1
	}

	// A sine at the amplitude that lands near the stated level, which is all
	// `stream` reads.
	out := make([]float32, len(signal))
	amp := math.Pow(10, b.at[i]/20) * math.Sqrt2

	for n := range out {
		out[n] = float32(amp * math.Sin(2*math.Pi*110*float64(n)/sdk.Rate))
	}

	return out, nil
}

func (levels) Name() string { return "a bench with a settling stream" }

// TestStream covers how many readings it spends.
//
// One method and one table, so a case is a row rather than a file.
func (s *StreamTestSuite) TestStream() {
	for _, tt := range []struct {
		name string
		// at is the level each reading comes back at, the last repeating.
		at []float64
		// spent is how many readings stream consumes before it is satisfied.
		spent int
	}{
		{
			// The rig this was written against: two readings of the stream
			// and then the loop, to the last digit. Three reads to see two in
			// a row agree.
			name:  "two readings of the stream and then the loop",
			at:    []float64{-58.60, -59.97, -51.04},
			spent: 4,
		},
		{
			// `steady`'s own evidence, where one is unsettled. Two reads to
			// see the pair agree.
			name:  "one reading of the stream",
			at:    []float64{-30, -51.04},
			spent: 3,
		},
		{
			// Settled from the first, which is a stream that was already
			// running. Two reads, because agreeing takes a pair.
			name:  "settled before it was asked",
			at:    []float64{-51.04},
			spent: 2,
		},
		{
			// Never settles, so it gives up rather than spinning. Bounded
			// because a loop that cannot settle has something else wrong.
			name:  "a stream that never settles",
			at:    []float64{-10, -20, -30, -40, -50, -60, -70, -80, -90},
			spent: streamSettles,
		},
	} {
		s.Run(tt.name, func() {
			calls := 0

			s.Require().NoError(stream(context.Background(),
				levels{at: tt.at, calls: &calls}, make([]float32, 4096)))
			s.Require().Equal(tt.spent, calls)
		})
	}
}

// TestStreamReportsABenchThatWillNotRead covers the reading failing.
//
// Said rather than swallowed: a baseline measured on a bench that cannot
// answer is the one reading every other is calibrated against.
func (s *StreamTestSuite) TestStreamReportsABenchThatWillNotRead() {
	boom := errors.New("no bench")

	err := stream(context.Background(),
		bench{err: boom}, make([]float32, 4096))

	s.Require().ErrorIs(err, boom)
}

func TestStreamTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(StreamTestSuite))
}
