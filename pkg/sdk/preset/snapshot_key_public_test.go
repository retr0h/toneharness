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

package preset_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/preset"
)

// SnapshotKeyPublicTestSuite covers the key a snapshot is stored under.
//
// It lives here because both the compiler and the editor write the same tone
// map with it. Declared once in each, the two would have to stay equal for a
// preset to round-trip and nothing would have said so.
type SnapshotKeyPublicTestSuite struct {
	suite.Suite
}

// TestSnapshotIndex covers SnapshotIndex, which reads the number a
// snapshot is stored under, or -1.
//
// One method and one table, so a case is a row rather than a file.
func (s *SnapshotKeyPublicTestSuite) TestSnapshotIndex() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			name: "snapshot index",
			then: func() {
				tests := []struct {
					name string
					key  string
					want int
				}{
					{"the first, numbered from nothing", "snapshot0", 0},
					{"the last an HX Stomp holds", "snapshot7", 7},
					{"a number past what any device holds is still a number", "snapshot99", 99},
					{"a processor is not a snapshot", "dsp0", -1},
					{"nor is a controller assignment", "controller", -1},
					{"the prefix with nothing after it", "snapshot", -1},
					{"the prefix with something that is not a number", "snapshotA", -1},
					{"a key that merely contains it", "mysnapshot0", -1},
					{"nothing at all", "", -1},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						s.Require().Equal(tt.want, preset.SnapshotIndex(tt.key))
					})
				}
			},
		},
		{
			// back out of it, in two different packages. A prefix changed in one place has
			// to keep that working, which is what makes this worth asserting rather than
			// reading.
			name: "the prefix and the index agree",
			then: func() {
				for at := range 8 {
					key := preset.SnapshotPrefix + strconv.Itoa(at)

					s.Require().Equal(at, preset.SnapshotIndex(key), key)
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestThePrefixAndTheIndexAgree is the round trip both readers rely on.
//

func TestSnapshotKeyPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SnapshotKeyPublicTestSuite))
}
