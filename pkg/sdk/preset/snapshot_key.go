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

package preset

import (
	"strconv"
	"strings"
)

// SnapshotPrefix is what a tone entry holding a snapshot is keyed by.
//
// `snapshot0` through `snapshot7`, numbered from nothing. It lives here rather
// than beside either reader because both the compiler and the editor key the
// same map with it, and it is a fact about the file format rather than about
// either of them: declared twice, the two would have to stay equal for a
// preset to round-trip, and nothing would have said so.
const SnapshotPrefix = "snapshot"

// SnapshotIndex reads the number a snapshot is stored under, or -1.
//
// -1 rather than an error, because the caller is walking every key in a tone
// map and most of them are not snapshots: a processor, a controller
// assignment, whatever a later firmware adds.
func SnapshotIndex(
	key string,
) int {
	rest, ok := strings.CutPrefix(key, SnapshotPrefix)
	if !ok {
		return -1
	}

	n, err := strconv.Atoi(rest)
	if err != nil {
		return -1
	}

	return n
}
