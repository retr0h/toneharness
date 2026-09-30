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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

// closes is a bench that records having been given back.
type closes struct {
	bench

	closed int
}

// Close answers with an error on purpose: benchFor discards it, and a closer
// that only works when Close succeeds is the bug that would go unnoticed.
func (c *closes) Close() error {
	c.closed++

	return errors.New("a close nobody reads")
}

// BenchPublicTestSuite covers choosing the audio loop a measuring run reads
// through.
type BenchPublicTestSuite struct {
	suite.Suite
}

// TestABenchTheCallerHoldsIsHandedBack is what every other test in this package
// relies on.
//
// TestBenchFor covers benchFor, which is the audio loop a measuring run reads
// through, and how to let it.
//
// One method and one table, so a case is a row rather than a file.
func (s *BenchPublicTestSuite) TestBenchFor() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// the named hardware is never looked at. That is the branch the whole suite
			// takes, which is why nothing here needs an interface, a cable and somebody in
			// the room to plug them in.
			name: "a bench the caller holds is handed back",
			then: func() {
				held := bench{}

				got, release, err := benchFor(held, "a name it must not read")

				s.Require().NoError(err)
				s.Require().Equal(held, got)
				s.Require().NotNil(release)

				release()
				release()
			},
		},
		{
			// reamp.TestOpenSaysWhatWasAttachedInstead covers it against the same name.
			name: "hardware nothing answers to is reported",
			then: func() {
				_, _, err := benchFor(nil, "no such interface anybody owns")

				s.Require().Error(err)
			},
		},
		{
			// one this package opened must be, or the audio device stays held until the
			// process exits. Nothing else in the package can tell those two apart, so the
			// assertion is that the closer reached Close.
			name: "a bench this package opened is closed by its closer",
			then: func() {
				held := &closes{}

				was := opens
				defer func() { opens = was }()

				opens = func(hardware string) (opened, error) {
					s.Require().Equal("a name it must read", hardware)

					return held, nil
				}

				got, release, err := benchFor(nil, "a name it must read")

				s.Require().NoError(err)
				s.Require().Equal(opened(held), got)
				s.Require().Zero(held.closed)

				release()

				s.Require().Equal(1, held.closed)
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
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

// TestABenchThisPackageOpenedIsClosedByItsCloser is the branch that owns a
// lifetime.
//

func TestBenchPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BenchPublicTestSuite))
}
