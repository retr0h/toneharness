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

package packed_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/packed"
)

// PackedPublicTestSuite covers writing a generated file only when it changed.
type PackedPublicTestSuite struct {
	suite.Suite
}

// TestBytes covers Bytes, which gzips a generated blob.
//
// One method and one table, so a case is a row rather than a file.
func (s *PackedPublicTestSuite) TestBytes() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The invariant Refresh rests on.
			//
			// Both generators compare the bytes they would write against the
			// bytes already committed, and gzip's header carries a
			// modification time. Setting it would rewrite the committed file
			// on every run with identical content, which is a diff on a
			// branch that touched nothing and a `just ready` that never
			// settles. Two generators depended on this and neither said it.
			//
			// The header rather than two calls agreeing: gzip stores the time
			// to the second, so compressing the same input twice in one test
			// matches even when the field is set, and an assertion that the
			// two agree cannot see it. This reads the field.
			name: "it carries no modification time",
			then: func() {
				zr, err := gzip.NewReader(bytes.NewReader(packed.Bytes([]byte("anything"))))
				s.Require().NoError(err)

				s.Require().True(zr.ModTime.IsZero(),
					"a time in the header makes the same input compress to different bytes "+
						"on two runs, and Refresh decides by comparing them")
			},
		},
		{
			// The same rule, end to end.
			name: "the same input compresses to the same bytes",
			then: func() {
				raw := []byte(`{"device":"HX Stomp","blocks":661}`)

				s.Require().Equal(packed.Bytes(raw), packed.Bytes(raw))
			},
		},
		{
			// The round trip.
			name: "it gzips what it was given",
			then: func() {
				raw := []byte(`{"device":"HX Stomp"}`)

				zr, err := gzip.NewReader(bytes.NewReader(packed.Bytes(raw)))
				s.Require().NoError(err)

				got, err := io.ReadAll(zr)
				s.Require().NoError(err)
				s.Require().Equal(raw, got)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestRefresh covers Refresh, which writes body to path unless what is there
// already matches, and says whether it wrote.
//
// One method and one table, so a case is a row rather than a file.
func (s *PackedPublicTestSuite) TestRefresh() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The whole point of it.
			name: "refresh writes only what changed",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "hx-stomp.json.gz")

				// A fresh clone has no generated file, which is a write rather than a
				// refusal.
				wrote, err := packed.Refresh(at, []byte("one"))
				s.Require().NoError(err)
				s.Require().True(wrote, "there was nothing there")

				// The same content again leaves it alone, which is how a regenerate stays
				// out of a diff.
				wrote, err = packed.Refresh(at, []byte("one"))
				s.Require().NoError(err)
				s.Require().False(wrote)

				wrote, err = packed.Refresh(at, []byte("two"))
				s.Require().NoError(err)
				s.Require().True(wrote)

				got, err := os.ReadFile(at)
				s.Require().NoError(err)
				s.Require().Equal([]byte("two"), got)
			},
		},
		{
			// A path that is not there.
			name: "refresh reports somewhere it cannot write",
			then: func() {
				at := filepath.Join(s.T().TempDir(), "nowhere", "hx-stomp.json.gz")

				_, err := packed.Refresh(at, []byte("one"))

				s.Require().ErrorContains(err, at)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestPackedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackedPublicTestSuite))
}
