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

package slug_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
)

type SlugPublicTestSuite struct {
	suite.Suite
}

// TestOf covers turning a name into the shape a path takes.
func (s *SlugPublicTestSuite) TestOf() {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "a player", in: "Mike Dirnt", want: "mike-dirnt"},
		{name: "a band", in: "Red Hot Chili Peppers", want: "red-hot-chili-peppers"},
		{
			// The reason this is one function rather than three. Two spellings
			// of one band must not group as two.
			name: "an apostrophe",
			in:   "Guns N' Roses",
			want: "guns-n-roses",
		},
		{name: "the same band written plainly", in: "Guns n Roses", want: "guns-n-roses"},
		{
			name: "a hyphen already there",
			in:   "Parliament-Funkadelic",
			want: "parliament-funkadelic",
		},
		{name: "a digit", in: "blink-182", want: "blink-182"},
		{name: "a colon and spaces", in: "DIR:ANGL Meteor", want: "dir-angl-meteor"},
		{name: "already a slug", in: "pop-punk", want: "pop-punk"},
		{name: "runs of punctuation", in: "Sum 41 -- All Killer", want: "sum-41-all-killer"},
		{name: "padded", in: "  Flea  ", want: "flea"},
		{
			// An empty path segment is a directory nobody can name, so a name
			// with nothing usable in it answers something rather than nothing.
			name: "nothing usable in it",
			in:   "!!!",
			want: "untitled",
		},
		{name: "empty", in: "", want: "untitled"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := slug.Of(tt.in)
			s.Require().Equal(tt.want, got)

			// Slugging a slug, asked of every row rather than of a chosen few.
			// The corpus groups on the slug, and a name that came back from one
			// round has to survive another: a directory read off disk is already
			// slugged, and slugging it again must not move it.
			s.Require().Equal(got, slug.Of(got),
				"%q does not survive a second pass", tt.in)
		})
	}
}

func TestSlugPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SlugPublicTestSuite))
}
