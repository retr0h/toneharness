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

package audio

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WriteSeamTestSuite covers the three ways the file underneath a kept reading
// fails.
//
// In-package, because the seam it stands its own functions in is unexported: a
// real *os.File on a temp directory cannot be made to refuse a write or a close
// on demand, and those are exactly the failures worth reporting rather than
// swallowing.
type WriteSeamTestSuite struct {
	suite.Suite

	madeWith func(string) (io.WriteSeeker, error)
	shutWith func(io.WriteSeeker) error
}

func (s *WriteSeamTestSuite) SetupTest() {
	s.madeWith, s.shutWith = creates, closes
}

func (s *WriteSeamTestSuite) TearDownTest() {
	creates, closes = s.madeWith, s.shutWith
}

// holds takes everything and answers nothing, so a row can fail whichever half
// it means to.
type holds struct{ failWrite bool }

func (h *holds) Write(p []byte) (int, error) {
	if h.failWrite {
		return 0, errors.New("the file would not take it")
	}

	return len(p), nil
}

func (*holds) Seek(int64, int) (int64, error) { return 0, nil }

// TestWriteReportsTheFileUnderneath covers a path that will not open, a file
// that will not take the samples, and a file that will not close.
//
// One method and one table, so a case is a row rather than a file.
func (s *WriteSeamTestSuite) TestWriteReportsTheFileUnderneath() {
	declined := errors.New("the filesystem declined")

	for _, tt := range []struct {
		name string
		// openErr, failWrite and closeErr are which half refuses.
		openErr   error
		failWrite bool
		closeErr  error
		says      string
	}{
		{
			name:    "a path that will not open",
			openErr: declined,
			says:    "creating",
		},
		{
			name:      "a file that will not take the samples",
			failWrite: true,
			says:      "writing",
		},
		{
			// The close is what patches the header's lengths on the way out, so
			// one that fails leaves a file nothing will play. Saying so is the
			// difference between a kept reading and a kept reading somebody
			// believes in.
			name:     "a file that will not close",
			closeErr: declined,
			says:     "closing",
		},
		{
			name: "a file that takes all of it",
		},
	} {
		s.Run(tt.name, func() {
			creates = func(string) (io.WriteSeeker, error) {
				if tt.openErr != nil {
					return nil, tt.openErr
				}

				return &holds{failWrite: tt.failWrite}, nil
			}

			closes = func(io.WriteSeeker) error { return tt.closeErr }

			err := Write("kept.wav", []float32{0.1, -0.1}, 48000)

			if tt.says == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

// TestClosesLetsGoOfWhatCanBeClosed covers the real closer, which is the one
// place a destination that cannot be closed is distinguished from one that
// refuses.
func (s *WriteSeamTestSuite) TestClosesLetsGoOfWhatCanBeClosed() {
	// A destination with no Close is nothing to shut, which is every fake here
	// and not an error.
	s.Require().NoError(s.shutWith(&holds{}))
}

func TestWriteSeamTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WriteSeamTestSuite))
}
