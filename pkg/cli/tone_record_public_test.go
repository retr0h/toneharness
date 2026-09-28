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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// RecordPublicTestSuite covers writing a round of correction onto an ask.
//
// The only place a human ear's judgement will ever be written down, so what
// matters here is that the round survives the round trip and that the verdict
// is left empty for somebody to fill in.
type RecordPublicTestSuite struct {
	suite.Suite

	ask string
}

func (s *RecordPublicTestSuite) SetupTest() {
	s.ask = filepath.Join(s.T().TempDir(), "ask.yaml")

	s.Require().NoError(os.WriteFile(s.ask, []byte(
		"schema: ToneSpec\ngenre: [punk]\n"), 0o600))
}

// read loads the ask back the way anything else would.
func (s *RecordPublicTestSuite) read() tone.Spec {
	f, err := os.Open(s.ask)
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	spec, err := tone.Load(f)
	s.Require().NoError(err)

	return spec
}

// moved is one control the solve turned.
func (s *RecordPublicTestSuite) moved() []solve.Step {
	knob := solve.Knob{
		Block: 1, Param: 0, Control: "HD2_AmpSVBeastBrt Bass",
		Setting: "Bass", At: 0.41, Low: 0, High: 1,
	}

	return []solve.Step{{Knob: knob, By: 0.56, To: 0.97}}
}

// TestARoundIsWrittenDown covers the whole of what a correction holds.
func (s *RecordPublicTestSuite) TestARoundIsWrittenDown() {
	s.Require().NoError(record(s.ask, "punk", s.moved(),
		map[audio.Figure]float64{audio.KeyCentroid: 0.4}, true))

	got := s.read()
	s.Require().NotNil(got.Corrections)
	s.Require().Len(*got.Corrections, 1)

	was := (*got.Corrections)[0]
	s.Require().Equal("punk", was.Ask, "in the words that were used")
	s.Require().NotNil(was.At)
	s.Require().Regexp(`^\d{4}-\d{2}-\d{2}$`, *was.At)

	s.Require().Nil(was.Verdict,
		"absent means not yet heard, which is the state after tuning")

	s.Require().NotNil(was.Reason)
	s.Require().Contains(*was.Reason, "inside its tolerance")

	s.Require().NotNil(was.Changed)
	s.Require().Len(*was.Changed, 1)
	s.Require().Equal("blocks[1].params.Bass", (*was.Changed)[0].Path,
		"a path into the plan it moved, which the contract's pattern accepts")
}

// TestThePathUsesTheSettingRatherThanTheLabel covers the two names a control
// has.
//
// Control carries the model for a person to read and has a space in it. A path
// has a grammar, and the contract refuses one built from the label: the first
// round written was rejected for exactly that.
func (s *RecordPublicTestSuite) TestThePathUsesTheSettingRatherThanTheLabel() {
	s.Require().NoError(record(s.ask, "punk", s.moved(), nil, false))

	got := s.read()
	path := (*(*got.Corrections)[0].Changed)[0].Path

	s.Require().NotContains(path, " ")
	s.Require().NotContains(path, "HD2_")
}

// TestRoundsAccumulate covers the append-only shape.
//
// Never replayed and never rewritten: the plan holds the current settings and
// this holds why they are what they are, in the order they were asked.
func (s *RecordPublicTestSuite) TestRoundsAccumulate() {
	s.Require().NoError(record(s.ask, "punk", s.moved(), nil, true))
	s.Require().NoError(record(s.ask, "darker", s.moved(), nil, false))

	got := *s.read().Corrections
	s.Require().Len(got, 2)
	s.Require().Equal("punk", got[0].Ask)
	s.Require().Equal("darker", got[1].Ask, "in the order they were asked")
}

// TestARoundThatDidNotArriveSaysSo covers the reason a later reader needs.
//
// A round that arrived and one that ran out of room look identical in the
// settings alone, so the reason is what lets somebody judge the interpretation
// rather than only the values.
func (s *RecordPublicTestSuite) TestARoundThatDidNotArriveSaysSo() {
	s.Require().NoError(record(s.ask, "punk", s.moved(),
		map[audio.Figure]float64{audio.KeyCentroid: 7.2}, false))

	was := (*s.read().Corrections)[0]
	s.Require().Contains(*was.Reason, "did not reach it")
	s.Require().Contains(*was.Reason, "7.2")
	s.Require().Contains(*was.Reason, string(audio.KeyCentroid))
}

// TestAnAskThatIsNotThere covers a path nobody wrote.
func (s *RecordPublicTestSuite) TestAnAskThatIsNotThere() {
	err := record(filepath.Join(s.T().TempDir(), "nope.yaml"), "punk", nil, nil, true)

	s.Require().ErrorContains(err, "reading")
}

// TestAnAskThatIsNotAToneSpec covers a file that will not load.
func (s *RecordPublicTestSuite) TestAnAskThatIsNotAToneSpec() {
	s.Require().NoError(os.WriteFile(s.ask, []byte("schema: RigSpec\n"), 0o600))

	s.Require().Error(record(s.ask, "punk", nil, nil, true))
}

// TestAnAskItCannotWrite covers the file going away underneath it.
func (s *RecordPublicTestSuite) TestAnAskItCannotWrite() {
	dir := s.T().TempDir()
	at := filepath.Join(dir, "ask.yaml")

	s.Require().NoError(os.WriteFile(at, []byte(
		"schema: ToneSpec\ngenre: [punk]\n"), 0o600))
	s.Require().NoError(os.Chmod(dir, 0o500))

	defer func() { s.Require().NoError(os.Chmod(dir, 0o700)) }()

	err := record(at, "punk", nil, nil, true)
	s.Require().True(err == nil || strings.Contains(err.Error(), "writing"),
		"a refused write says which file")
}

func TestRecordPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RecordPublicTestSuite))
}
