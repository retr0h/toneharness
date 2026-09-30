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
package measured_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// EmbedPublicTestSuite covers the measurements this binary ships.
type EmbedPublicTestSuite struct {
	suite.Suite
}

// TestBuiltInIsReadable covers the shipped file being what it claims.
//
// It is data taken off hardware rather than anything generated from source,
// so nothing but this notices if it is packed wrong or replaced with a
// library measured some other way.
func (s *EmbedPublicTestSuite) TestBuiltInIsReadable() {
	lib, err := measured.BuiltIn()

	s.Require().NoError(err)
	s.Require().Equal("HX Stomp", lib.Device)
	s.Require().True(lib.Isolated)
	s.Require().NotEmpty(lib.Blocks)

	// The reference signal travels with the readings, because every one of
	// them is a comparison against it. A library that does not name it is
	// one nothing later can be held to.
	s.Require().NotEmpty(lib.Reference.SHA256)
	s.Require().Positive(lib.Baseline.Centroid)
}

// TestBuiltInIsParsedOnce covers the cache handing back the same library.
func (s *EmbedPublicTestSuite) TestBuiltInIsParsedOnce() {
	first, err := measured.BuiltIn()
	s.Require().NoError(err)

	again, err := measured.BuiltIn()
	s.Require().NoError(err)
	s.Require().Equal(first, again)
}

// TestPacked covers Packed, which writes a library out the way this package
// embeds one.
//
// One method and one table, so a case is a row rather than a file.
func (s *EmbedPublicTestSuite) TestPacked() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "packed round trips",
			then: func() {
				lib := `{"device":"HX Stomp","isolated":true,` +
					`"blocks":{"A":{"id":"A","category":"amp"}}}`

				var packed bytes.Buffer
				s.Require().NoError(measured.Packed(&packed, strings.NewReader(lib)))

				z, err := gzip.NewReader(&packed)
				s.Require().NoError(err)

				back, err := measured.Load(z)
				s.Require().NoError(err)
				s.Require().Equal("HX Stomp", back.Device)
				s.Require().True(back.Isolated)
				s.Require().Len(back.Blocks, 1)
			},
		},
		{
			// library is refused where somebody can still fix it rather than at the next
			// build.
			name: "packed refuses what it could not read",
			then: func() {
				var packed bytes.Buffer

				s.Require().ErrorContains(
					measured.Packed(&packed, strings.NewReader("{")),
					"decoding the measurements")
				s.Require().Empty(packed.Bytes())
			},
		},
		{
			name: "packed reports a read failure",
			then: func() {
				var packed bytes.Buffer

				s.Require().ErrorContains(
					measured.Packed(&packed, broken{}), "reading the measurements")
			},
		},
		{
			name: "packed reports a write failure",
			then: func() {
				lib := `{"device":"HX Stomp","isolated":true,` +
					`"blocks":{"A":{"id":"A","category":"amp"}}}`

				s.Require().Error(measured.Packed(brokenW{}, strings.NewReader(lib)))
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestPackedRefusesWhatItCouldNotRead is why packing lives beside reading.
//

// TestUnpackRefusesWhatItCannotUse covers the two ways a packed library is
// no good.
//
// Neither can happen to the shipped file, which is why they are worth a test:
// the checks are there for the day somebody regenerates it another way.
func (s *EmbedPublicTestSuite) TestUnpackRefusesWhatItCannotUse() {
	s.Run("not gzip at all", func() {
		_, err := measured.Unpack([]byte("plain text"))

		s.Require().ErrorContains(err, "unpacking the measurements")
	})

	s.Run("measured in a chain", func() {
		// A library taken with more than one block in the path describes the
		// path. Every reading in it carries whatever else was beside it, and
		// ranking blocks against each other on those is comparing chains.
		var packed bytes.Buffer

		z := gzip.NewWriter(&packed)
		_, err := z.Write([]byte(`{"device":"HX Stomp","isolated":false,` +
			`"blocks":{"A":{"id":"A","category":"amp"}}}`))
		s.Require().NoError(err)
		s.Require().NoError(z.Close())

		_, err = measured.Unpack(packed.Bytes())

		s.Require().ErrorContains(err, "describe the chain")
	})

	s.Run("gzip holding nothing usable", func() {
		var packed bytes.Buffer

		z := gzip.NewWriter(&packed)
		_, err := z.Write([]byte("{"))
		s.Require().NoError(err)
		s.Require().NoError(z.Close())

		_, err = measured.Unpack(packed.Bytes())

		s.Require().ErrorContains(err, "decoding the measurements")
	})
}

// brokenW is a writer that always fails.
type brokenW struct{}

func (brokenW) Write(
	[]byte,
) (int, error) {
	return 0, errors.New("no")
}

func TestEmbedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EmbedPublicTestSuite))
}
