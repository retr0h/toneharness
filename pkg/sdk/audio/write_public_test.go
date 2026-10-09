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

package audio_test

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// WritePublicTestSuite covers keeping a reading as a file somebody can play.
type WritePublicTestSuite struct {
	suite.Suite
}

// TestWrite covers writing samples out and reading them back.
//
// Round-tripped rather than checked against a fixture, because what matters is
// that the decoder this project already uses can read what this writes: a file
// nothing will play is the one failure that would not show up as an error.
//
// One method and one table, so a case is a row rather than a file.
func (s *WritePublicTestSuite) TestWrite() {
	for _, tt := range []struct {
		name string
		// of is what goes in, as samples.
		of []float32
	}{
		{
			name: "a tone",
			of: func() []float32 {
				out := make([]float32, 512)
				for i := range out {
					out[i] = float32(0.5 * math.Sin(2*math.Pi*110*float64(i)/48000))
				}

				return out
			}(),
		},
		{
			// A reading of silence is still a reading, and a chain that went
			// quiet is exactly what somebody would want to listen to.
			name: "silence",
			of:   make([]float32, 64),
		},
		{
			// Full scale either way, which is where a conversion that rounds the
			// wrong way would clip or wrap.
			name: "the ends of the range",
			of:   []float32{1, -1, 0.5, -0.5},
		},
	} {
		s.Run(tt.name, func() {
			at := filepath.Join(s.T().TempDir(), "kept.wav")

			s.Require().NoError(audio.Write(at, tt.of, 48000))

			f, err := os.Open(at)
			s.Require().NoError(err)

			defer func() { s.Require().NoError(f.Close()) }()

			got, rate, err := audio.Read(f)
			s.Require().NoError(err)
			s.Require().Equal(48000, rate, "the rate it was measured at")
			s.Require().Len(got, len(tt.of), "every sample came back")

			for i := range got {
				s.Require().InDelta(float64(tt.of[i]), got[i], 0.001,
					"sample %d came back as it went in", i)
			}
		})
	}
}

// refuses is a destination that fails on whichever operation a row names.
//
// An *os.File will not fail on demand, and the encoder's failures are most of
// what can go wrong in here: the close is what patches the header's lengths, so
// one that is swallowed leaves a file nothing will play.
type refuses struct {
	onWrite bool
	onSeek  bool
}

func (r *refuses) Write(
	p []byte,
) (int, error) {
	if r.onWrite {
		return 0, errors.New("the file would not take it")
	}

	return len(p), nil
}

func (r *refuses) Seek(
	int64,
	int,
) (int64, error) {
	if r.onSeek {
		return 0, errors.New("the file would not rewind")
	}

	return 0, nil
}

// TestWriteToReportsADestinationThatRefuses covers the two ways the encoder
// fails.
//
// One method and one table, so a case is a row rather than a file.
func (s *WritePublicTestSuite) TestWriteToReportsADestinationThatRefuses() {
	for _, tt := range []struct {
		name string
		to   *refuses
		says string
	}{
		{
			name: "a destination that will not take the samples",
			to:   &refuses{onWrite: true},
			says: "putting the samples in",
		},
		{
			// The close, which is where the lengths are patched in.
			name: "a destination that will not rewind for the header",
			to:   &refuses{onSeek: true},
			says: "finishing the header",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().ErrorContains(
				audio.WriteTo(tt.to, []float32{0.1, -0.1}, 48000), tt.says)
		})
	}
}

// TestWriteRefusesNowhereToPutIt covers a path that cannot be written.
func (s *WritePublicTestSuite) TestWriteRefusesNowhereToPutIt() {
	err := audio.Write(filepath.Join(s.T().TempDir(), "no", "such", "dir",
		"kept.wav"), []float32{0.1}, 48000)

	s.Require().ErrorContains(err, "creating")
}

func TestWritePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WritePublicTestSuite))
}
