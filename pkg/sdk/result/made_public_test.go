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

package result_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/result"
)

// MadePublicTestSuite covers what a build answers with.
type MadePublicTestSuite struct {
	suite.Suite
}

// TestAsked covers whether a Setup stated an output level at all.
//
// The distinction matters to a reader rather than to the arithmetic: a build
// given no target must say nothing about one, and a build that reached its
// target must not be confused with one that was never given it.
func (s *MadePublicTestSuite) TestAsked() {
	for _, tt := range []struct {
		name string
		in   result.Levelled
		want bool
	}{
		{
			// Asked for, and reached. Novoselic's `2x15 Brute` against the
			// `8x10 Ampeg SVT-E`'s reading, which is the 2.9dB that was found
			// by playing the two.
			name: "a level that was asked for",
			in:   result.Levelled{Wanted: -56.76, Was: -59.62},
			want: true,
		},
		{
			// Read but not asked for, which is still something to say.
			name: "a reading with no target",
			in:   result.Levelled{Was: -59.62},
			want: true,
		},
		{
			// A target with nothing read, which is a chain nothing has a
			// level for.
			name: "a target with nothing read",
			in:   result.Levelled{Wanted: -56.76},
			want: true,
		},
		{
			// A Setup that said nothing leaves both at zero, and a target of
			// zero would be a chain asked to come out at full scale, which
			// nobody asks for.
			name: "nothing asked and nothing read",
		},
	} {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Asked())
		})
	}
}

func TestMadePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MadePublicTestSuite))
}
