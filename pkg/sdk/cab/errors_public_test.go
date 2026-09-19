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

package cab_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/cab"
)

// ErrorsPublicTestSuite covers what this package refuses, and how.
//
// Each carries the detail as fields rather than only in its sentence, so a
// caller can act on it. A message is for a person; the numbers are for code.
type ErrorsPublicTestSuite struct {
	suite.Suite
}

// TestTooShortSaysWhichSideWasShort covers the detail reaching a caller.
func (s *ErrorsPublicTestSuite) TestTooShortSaysWhichSideWasShort() {
	_, err := cab.Capture(make([]float64, 10), make([]float64, 10), cab.Short)

	s.Require().ErrorIs(err, cab.ErrTooShort)

	var short *cab.TooShortError
	s.Require().ErrorAs(err, &short)

	s.Require().Equal(10, short.Sent)
	s.Require().Equal(10, short.Back)
	s.Require().Equal(cab.Short, short.Taps)
	s.Require().Contains(short.Error(), "what came back")
}

// TestMatchNamesItsOwnTwoSides covers the same error reading differently.
//
// Capture divides what came back by what went out; Match holds a target
// against what is in hand. The same shortage, and "what came back" would be
// the wrong word for one of them.
func (s *ErrorsPublicTestSuite) TestMatchNamesItsOwnTwoSides() {
	_, err := cab.Match(make([]float64, 4), make([]float64, 8), cab.Short)

	var short *cab.TooShortError
	s.Require().ErrorAs(err, &short)

	s.Require().Equal(4, short.Sent)
	s.Require().Equal(8, short.Back)
	s.Require().Contains(short.Error(), "target")
	s.Require().Contains(short.Error(), "what is in hand")
}

// TestSilenceIsNotAShortSignal covers the other way the arithmetic fails.
func (s *ErrorsPublicTestSuite) TestSilenceIsNotAShortSignal() {
	tests := []struct {
		name string
		call func() error
		side string
	}{
		{
			name: "nothing went in",
			call: func() error {
				_, err := cab.Capture(
					make([]float64, cab.Short), make([]float64, cab.Short), cab.Short)

				return err
			},
			side: "what went in",
		},
		{
			name: "nothing is in hand",
			call: func() error {
				_, err := cab.Match(
					make([]float64, cab.Short), make([]float64, cab.Short), cab.Short)

				return err
			},
			side: "what is in hand",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.call()

			s.Require().ErrorIs(err, cab.ErrTooShort)

			var quiet *cab.SilenceError
			s.Require().ErrorAs(err, &quiet)

			s.Require().Equal(tt.side, quiet.Side)
		})
	}
}

// TestALengthNoDeviceLoadsCarriesTheLength covers the taps reaching a caller.
func (s *ErrorsPublicTestSuite) TestALengthNoDeviceLoadsCarriesTheLength() {
	_, err := cab.Capture(make([]float64, 99), make([]float64, 99), 99)

	s.Require().ErrorIs(err, cab.ErrNotPowerOfTwo)

	var bad *cab.BadLengthError
	s.Require().ErrorAs(err, &bad)

	s.Require().Equal(99, bad.Taps)
}

// TestWritingRefusesTheSameLengths holds the writer to the same rule.
func (s *ErrorsPublicTestSuite) TestWritingRefusesTheSameLengths() {
	err := cab.Write(nil, make([]float64, 3), cab.Made{})

	s.Require().ErrorIs(err, cab.ErrNotPowerOfTwo)

	var bad *cab.BadLengthError
	s.Require().ErrorAs(err, &bad)

	s.Require().Equal(3, bad.Taps)
}

// TestEachErrorUnwrapsToItsSentinel covers matching without the struct.
func (s *ErrorsPublicTestSuite) TestEachErrorUnwrapsToItsSentinel() {
	for _, tt := range []struct {
		name string
		err  error
		is   error
	}{
		{"too short", &cab.TooShortError{}, cab.ErrTooShort},
		{"silence", &cab.SilenceError{}, cab.ErrTooShort},
		{"bad length", &cab.BadLengthError{}, cab.ErrNotPowerOfTwo},
	} {
		s.Run(tt.name, func() {
			s.Require().True(errors.Is(tt.err, tt.is))
			s.Require().NotEmpty(tt.err.Error())
		})
	}
}

func TestErrorsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
