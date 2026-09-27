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
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/cab"
)

// WritePublicTestSuite covers putting an impulse response where a device can
// load it.
type WritePublicTestSuite struct {
	suite.Suite
}

// made is provenance for a file that never leaves a test.
func (s *WritePublicTestSuite) made() cab.Made {
	return cab.Made{
		How:     cab.Matched,
		From:    "a synthetic roll-off",
		Through: "no hardware, a test",
		When:    time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC),
		Taps:    cab.Long,
	}
}

// TestADeviceCanReadWhatItWrote covers the file being a WAV.
func (s *WritePublicTestSuite) TestADeviceCanReadWhatItWrote() {
	of := make([]float64, cab.Long)
	of[0] = 1

	var buf bytes.Buffer
	s.Require().NoError(cab.Write(&buf, of, s.made()))

	body := buf.Bytes()

	s.Require().Equal("RIFF", string(body[0:4]))
	s.Require().Equal("WAVE", string(body[8:12]))
	s.Require().Equal("fmt ", string(body[12:16]))

	// The device's own rate, because a file at another is resampled on the
	// way in by something nobody here wrote and cannot measure.
	s.Require().Equal(uint32(cab.Rate),
		binary.LittleEndian.Uint32(body[24:28]))

	// Twenty four bits, because an impulse response is mostly very quiet:
	// the tail is where a cabinet's character is, and sixteen puts the
	// quantisation noise floor through it.
	s.Require().Equal(uint16(24), binary.LittleEndian.Uint16(body[34:36]))
	s.Require().Equal(uint16(1), binary.LittleEndian.Uint16(body[22:24]),
		"one channel")
}

// TestTheSizesAddUp covers the header describing the file it is in.
//
// A length field that disagrees with the bytes after it is how a device reads
// half a response and plays the rest as whatever was in memory.
func (s *WritePublicTestSuite) TestTheSizesAddUp() {
	var buf bytes.Buffer
	s.Require().NoError(cab.Write(&buf, make([]float64, cab.Short), s.made()))

	body := buf.Bytes()

	s.Require().Equal(uint32(len(body)-8),
		binary.LittleEndian.Uint32(body[4:8]), "the RIFF size")

	at := bytes.Index(body, []byte("data"))
	s.Require().Positive(at)

	held := binary.LittleEndian.Uint32(body[at+4 : at+8])
	s.Require().Equal(uint32(cab.Short*3), held, "three bytes a sample")
	s.Require().Len(body[at+8:], int(held))
}

// TestWhereItCameFromTravelsInside is what a file leaving the building needs.
//
// The tooling cannot tell a capture from a match from a filter fitted to
// somebody's record, and the difference decides whether it may be sold. A
// note beside the file is lost the first time somebody shares the file alone.
func (s *WritePublicTestSuite) TestWhereItCameFromTravelsInside() {
	var buf bytes.Buffer
	s.Require().NoError(cab.Write(&buf, make([]float64, cab.Long), s.made()))

	body := buf.String()

	s.Require().Contains(body, "LIST")
	s.Require().Contains(body, "INFO")
	s.Require().Contains(body, "toneharness")
	s.Require().Contains(body, "matched")
	s.Require().Contains(body, "a synthetic roll-off")
	s.Require().Contains(body, "2026-09-19")
	s.Require().Contains(body, "no hardware, a test")
}

// TestProvenanceWithoutAChain covers one made from nothing measured.
func (s *WritePublicTestSuite) TestProvenanceWithoutAChain() {
	made := s.made()
	made.Through = ""
	made.How = cab.Captured

	var buf bytes.Buffer
	s.Require().NoError(cab.Write(&buf, make([]float64, cab.Long), made))

	s.Require().Contains(buf.String(), "captured")
	s.Require().NotContains(buf.String(), "through ")
}

// TestSamplesSurviveTheRoundTrip covers what is written being what was built.
func (s *WritePublicTestSuite) TestSamplesSurviveTheRoundTrip() {
	of := make([]float64, cab.Short)
	for i := range of {
		of[i] = math.Sin(2*math.Pi*float64(i)/64) * 0.5
	}

	var buf bytes.Buffer
	s.Require().NoError(cab.Write(&buf, of, s.made()))

	got, rate, err := audio.Read(bytes.NewReader(buf.Bytes()))
	s.Require().NoError(err)
	s.Require().Equal(cab.Rate, rate)
	s.Require().Len(got, cab.Short)

	for i := range of {
		s.Require().InDeltaf(of[i], got[i], 1e-5,
			"sample %d came back as something else", i)
	}
}

// TestASamplePastFullScaleIsClampedNotWrapped is a click rather than a clip.
//
// A sample past full scale that wrapped would come back as a loud sample of
// the opposite sign.
func (s *WritePublicTestSuite) TestASamplePastFullScaleIsClampedNotWrapped() {
	of := make([]float64, cab.Short)
	of[0] = 4
	of[1] = -4

	var buf bytes.Buffer
	s.Require().NoError(cab.Write(&buf, of, s.made()))

	got, _, err := audio.Read(bytes.NewReader(buf.Bytes()))
	s.Require().NoError(err)

	s.Require().InDelta(1, got[0], 0.001)
	s.Require().InDelta(-1, got[1], 0.001)
}

// TestOnlyWhatADeviceLoads covers a length no device takes.
func (s *WritePublicTestSuite) TestOnlyWhatADeviceLoads() {
	var buf bytes.Buffer

	for _, taps := range []int{0, 512, 4096} {
		s.Require().ErrorIs(
			cab.Write(&buf, make([]float64, taps), s.made()),
			cab.ErrBadLength, "%d taps", taps)
	}
}

// TestAWriterThatFailsIsReported covers somewhere it cannot write.
//
// Stopped at each stage rather than once. A file is a header, then the
// provenance chunk, then the samples, and the samples are by far the largest
// part: a disk that fills does it there, and stopping only in the first
// hundred bytes would never have reached the write that matters.
func (s *WritePublicTestSuite) TestAWriterThatFailsIsReported() {
	for _, after := range []int{0, 20, 50, 100, 200, 1000, 2000} {
		err := cab.Write(&stops{after: after},
			make([]float64, cab.Short), s.made())

		s.Require().ErrorContains(err, "writing the impulse response",
			"a writer that stopped after %d bytes", after)
	}
}

// TestEveryWriteIsReportedWhenItFails covers the stages a byte count misses.
//
// A file is written in fifteen calls: eleven for the header fields, one for
// the provenance chunk, then the data marker, the length, and the samples.
// Two of those are four bytes each and sit between two much larger writes, so
// stopping a writer after a number of bytes lands in them only by luck. This
// stops on the nth call, which reaches each of them exactly.
func (s *WritePublicTestSuite) TestEveryWriteIsReportedWhenItFails() {
	for call := 1; call <= 15; call++ {
		err := cab.Write(&refuses{on: call},
			make([]float64, cab.Short), s.made())

		s.Require().ErrorContains(err, "writing the impulse response",
			"a writer that refused call %d", call)
	}
}

// refuses is a writer that takes every call but one.
type refuses struct {
	on   int
	seen int
}

func (w *refuses) Write(
	p []byte,
) (int, error) {
	w.seen++

	if w.seen == w.on {
		return 0, errors.New("no room")
	}

	return len(p), nil
}

// stops is a writer that takes a few bytes and then does not.
type stops struct {
	after int
	wrote int
}

func (w *stops) Write(
	p []byte,
) (int, error) {
	if w.wrote+len(p) > w.after {
		return 0, errors.New("no room")
	}

	w.wrote += len(p)

	return len(p), nil
}

func TestWritePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WritePublicTestSuite))
}
