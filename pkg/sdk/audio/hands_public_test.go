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
	"math"
	"os"
	"path/filepath"
	"testing"

	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// HandsPublicTestSuite covers holding one right hand against another.
//
// Synthesised notes rather than the dataset the committed figure came from.
// That audio is somebody else's, is not in the repository and is not in CI, so a
// test reading it would be a test that only ran here.
type HandsPublicTestSuite struct {
	suite.Suite

	root string
}

// note writes one, named the way IDMT-SMT-Bass names them.
func (s *HandsPublicTestSuite) note(
	dir, name string,
	hz float64,
) {
	at := filepath.Join(s.root, dir)
	s.Require().NoError(os.MkdirAll(at, 0o750))

	f, err := os.Create(filepath.Join(at, name)) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	enc := wav.NewEncoder(f, rate, 16, 1, 1)

	scale := math.Pow(2, 15) - 1
	samples := audio.Sine(hz, 0.5, rate, 0.8)
	data := make([]int, 0, len(samples))

	for _, v := range samples {
		data = append(data, int(v*scale))
	}

	s.Require().NoError(enc.Write(&goaudio.IntBuffer{
		Format:         &goaudio.Format{NumChannels: 1, SampleRate: rate},
		Data:           data,
		SourceBitDepth: 16,
	}))
	s.Require().NoError(enc.Close())
	s.Require().NoError(f.Close())
}

// measured holds the two directories against each other.
func (s *HandsPublicTestSuite) measured() audio.Hands {
	got, err := audio.HandsMeasured(
		os.DirFS(s.root), "fingers", "fingerstyle", "pick", "picked")
	s.Require().NoError(err)

	return got
}

// TestHandsMeasured covers pairing two directories of notes and taking the
// difference.
//
// One method and one table, so a case is a row rather than a file.
func (s *HandsPublicTestSuite) TestHandsMeasured() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// A higher note under the picked name, so the centroid moves up and
			// the direction is the one a plectrum reads on the real notes.
			name: "the same note two ways is one pair",
			then: func() {
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_0.wav", 110)
				s.note("picked", "BS_1_EQ_1_PK_NO_1_0.wav", 220)

				got := s.measured()

				s.Require().Equal(1, got.Pairs)
				s.Require().Equal(1, got.Agreed)
				s.Require().Equal("fingers", got.From)
				s.Require().Equal("pick", got.To)
				s.Require().Positive(got.Figures[audio.KeyCentroid].Mean,
					"the picked side reads higher, so the difference is positive")
			},
		},
		{
			// Pairing is on everything but the hand. A note either side has no
			// partner for is dropped, because the whole point is that nothing
			// else differs.
			name: "a note with no partner is not a pair",
			then: func() {
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_0.wav", 110)
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_1.wav", 110)
				s.note("picked", "BS_1_EQ_1_PK_NO_1_0.wav", 220)

				s.Require().Equal(1, s.measured().Pairs)
			},
		},
		{
			// A different bass or a different pickup setting is a different
			// recording, so it pairs with nothing.
			name: "another bass is another note",
			then: func() {
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_0.wav", 110)
				s.note("picked", "BS_2_EQ_1_PK_NO_1_0.wav", 220)

				s.Require().Zero(s.measured().Pairs)
			},
		},
		{
			// A file from somewhere else is skipped rather than guessed at. A
			// wrong pairing would compare two different notes and report the
			// difference as a hand.
			name: "a name this does not recognise is skipped",
			then: func() {
				s.note("fingerstyle", "take-one.wav", 110)
				s.note("picked", "take-one.wav", 220)

				s.Require().Zero(s.measured().Pairs)
			},
		},
		{
			// Nothing paired is not an error. Somebody holding one dataset and
			// not another gets no comparison rather than a failure.
			name: "nothing in common answers with nothing",
			then: func() {
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_0.wav", 110)
				s.note("picked", "BS_1_EQ_1_PK_NO_9_9.wav", 220)

				got := s.measured()

				s.Require().Zero(got.Pairs)
				s.Require().Empty(got.Figures)
			},
		},
		{
			// Agreement is counted on the centroid, and it is the useful half
			// of the claim: a mean every pair agreed with says the direction is
			// not in doubt however wide the spread is.
			name: "a pair moving the other way does not agree",
			then: func() {
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_0.wav", 110)
				s.note("picked", "BS_1_EQ_1_PK_NO_1_0.wav", 880)
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_2_0.wav", 880)
				s.note("picked", "BS_1_EQ_1_PK_NO_2_0.wav", 870)

				got := s.measured()

				s.Require().Equal(2, got.Pairs)
				s.Require().Equal(1, got.Agreed, "one of the two moved the other way")
			},
		},
		{
			name: "a directory that is not there is an error",
			then: func() {
				_, err := audio.HandsMeasured(
					os.DirFS(s.root), "fingers", "nowhere", "pick", "picked")

				s.Require().ErrorContains(err, "nowhere")
			},
		},
		{
			// The second directory, which is read after the first and so has
			// its own branch.
			name: "the other directory not being there is also an error",
			then: func() {
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_0.wav", 110)

				_, err := audio.HandsMeasured(
					os.DirFS(s.root), "fingers", "fingerstyle", "pick", "nowhere")

				s.Require().ErrorContains(err, "nowhere")
			},
		},
		{
			// A name that matches and a file that will not decode. Reported
			// rather than skipped: a note this cannot read is a note missing
			// from a pair, and a pair silently short is a figure measured on
			// fewer notes than it claims.
			name: "a note that is not audio is an error",
			then: func() {
				at := filepath.Join(s.root, "fingerstyle")
				s.Require().NoError(os.MkdirAll(at, 0o750))
				s.Require().NoError(os.WriteFile(
					filepath.Join(at, "BS_1_EQ_1_FS_NO_1_0.wav"),
					[]byte("not a wav"), 0o600))
				s.note("picked", "BS_1_EQ_1_PK_NO_1_0.wav", 220)

				_, err := audio.HandsMeasured(
					os.DirFS(s.root), "fingers", "fingerstyle", "pick", "picked")

				s.Require().ErrorContains(err, "BS_1_EQ_1_FS_NO_1_0.wav")
			},
		},
		{
			// A directory inside a directory of notes is not a note.
			name: "a directory among the notes is skipped",
			then: func() {
				s.note("fingerstyle", "BS_1_EQ_1_FS_NO_1_0.wav", 110)
				s.Require().NoError(os.MkdirAll(
					filepath.Join(s.root, "picked", "BS_1_EQ_1_PK_NO_1_0.wav"), 0o750))

				s.Require().Zero(s.measured().Pairs)
			},
		},
	} {
		s.Run(tt.name, func() {
			// Per row, because SetupTest runs once for the method and the notes
			// a row writes would otherwise pair with the next row's.
			s.root = s.T().TempDir()

			tt.then()
		})
	}
}

func TestHandsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HandsPublicTestSuite))
}
