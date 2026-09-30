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
	"testing"

	"github.com/stretchr/testify/suite"
)

// BenchPublicTestSuite covers choosing the audio loop a measuring run reads
// through.
type BenchPublicTestSuite struct {
	suite.Suite
}

// TestABenchTheCallerHoldsIsHandedBack is what every other test in this package
// relies on.
//
// A caller who supplied one owns its lifetime, so the closer does nothing and
// the named hardware is never looked at. That is the branch the whole suite
// takes, which is why nothing here needs an interface, a cable and somebody in
// the room to plug them in.
func (s *BenchPublicTestSuite) TestABenchTheCallerHoldsIsHandedBack() {
	held := bench{}

	got, release, err := benchFor(held, "a name it must not read")

	s.Require().NoError(err)
	s.Require().Equal(held, got)
	s.Require().NotNil(release)

	release()
	release()
}

// TestHardwareNothingAnswersToIsReported is the other branch, and the only
// test here that opens audio.
//
// One test rather than one per command. Five commands each had one, and each
// spent three to six seconds on continuous integration initialising an audio
// subsystem that is not there — twenty seconds of a unit suite to assert a
// refusal that belongs to whichever function opens the device. This is that
// function.
//
// What the device layer does with the name is reamp's own subject, and
// reamp.TestOpenSaysWhatWasAttachedInstead covers it against the same name.
func (s *BenchPublicTestSuite) TestHardwareNothingAnswersToIsReported() {
	_, _, err := benchFor(nil, "no such interface anybody owns")

	s.Require().Error(err)
}

func TestBenchPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BenchPublicTestSuite))
}
