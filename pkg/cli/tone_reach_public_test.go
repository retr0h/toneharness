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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/solve"
)

// ReachPublicTestSuite covers painting a reachability answer.
//
// The arithmetic is the SDK's, because the same question is asked over MCP and
// neither surface may answer it differently. What is under test here is which
// of three things a person is told, because the three mean different things
// and reading the wrong one costs an afternoon.
type ReachPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	sdk  *mocks.MockReaches
}

func (s *ReachPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.sdk = mocks.NewMockReaches(s.ctrl)
}

func (s *ReachPublicTestSuite) opts() ReachOptions {
	return ReachOptions{Client: s.sdk, ID: "matt-freeman", Genre: "punk"}
}

// answers makes the SDK hand back one reading.
func (s *ReachPublicTestSuite) answers(
	got sdk.Reaching,
) {
	s.sdk.EXPECT().Reach(gomock.Any(), gomock.Any()).Return(got, nil)
}

// TestAnAxisSomeReadingLandedOnIsSaidToBeSo covers the claim worth trusting.
func (s *ReachPublicTestSuite) TestAnAxisSomeReadingLandedOnIsSaidToBeSo() {
	s.answers(sdk.Reaching{
		Rig: "matt-freeman", Genre: "punk", Readings: 193,
		Worth: true, Together: true,
		Axes: []solve.Verdict{
			{Figure: audio.KeyCentroid, Gap: 11, Swing: 1324, Shown: true, Within: true},
		},
		Decides: solve.Verdict{Figure: audio.KeyCentroid, Gap: 11, Within: true},
	})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "matt-freeman against punk")
	s.Require().Contains(w.String(), "193 readings already taken")
	s.Require().Contains(w.String(), "a reading landed there")
	s.Require().Contains(w.String(), "satisfies all 1 axes at once")
	s.Require().Contains(w.String(), "the model flatters")
}

// TestAnAxisNothingCanCloseIsTheHeadline covers the answer that saves the
// afternoon.
//
// The gear being wrong for the sound is a real answer to somebody who owns
// that gear, and it is the one this exists to deliver before five minutes of
// real-time audio rather than after.
func (s *ReachPublicTestSuite) TestAnAxisNothingCanCloseIsTheHeadline() {
	s.answers(sdk.Reaching{
		Rig: "matt-freeman", Genre: "punk", Readings: 40,
		Axes: []solve.Verdict{
			{Figure: audio.KeyHigh, Gap: 400, Swing: 5},
		},
		Decides: solve.Verdict{Figure: audio.KeyHigh, Gap: 400, Swing: 5},
	})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "OUT OF REACH")
	s.Require().Contains(w.String(), "Not worth running")
	s.Require().Contains(w.String(), "Change the chain, not the knobs")
}

// TestAnAxisOnlyTheSwingAllowsIsNotCalledReachable covers the weak claim.
//
// No reading landed near it and the controls merely have more movement than
// the gap. That promises nothing, and wording it as though it did is how a
// bound that flatters the controls becomes a bound somebody trusted.
func (s *ReachPublicTestSuite) TestAnAxisOnlyTheSwingAllowsIsNotCalledReachable() {
	s.answers(sdk.Reaching{
		Rig: "matt-freeman", Genre: "punk", Worth: true, Together: true,
		Axes: []solve.Verdict{
			{Figure: audio.KeyLow, Gap: 4, Swing: 90, Within: true},
		},
		Decides: solve.Verdict{Figure: audio.KeyLow, Within: true},
	})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "nothing rules it out")
	s.Require().NotContains(w.String(), "a reading landed there")
	s.Require().Contains(w.String(), "maybe", "the column says how weak the claim is")
}

// TestAnAxisAlreadyInsideNeedsNoReading covers a target met before starting.
func (s *ReachPublicTestSuite) TestAnAxisAlreadyInsideNeedsNoReading() {
	s.answers(sdk.Reaching{
		Rig: "matt-freeman", Genre: "punk", Worth: true, Together: true,
		Axes: []solve.Verdict{
			{Figure: audio.KeyMid, Gap: 0.2, Met: true, Within: true},
		},
	})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "already there")
}

// TestABlockNobodyHasSweptIsNamed covers an incomplete answer.
//
// Said rather than left out. Whatever an unswept block could have moved is
// missing from every number in the table, so a narrow answer has to read as
// one.
func (s *ReachPublicTestSuite) TestABlockNobodyHasSweptIsNamed() {
	s.answers(sdk.Reaching{
		Rig: "matt-freeman", Genre: "punk", Worth: true, Together: true,
		Unswept: []string{"HD2_NeverSwept"},
		Axes: []solve.Verdict{
			{Figure: audio.KeyMid, Met: true, Within: true},
		},
	})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "HD2_NeverSwept has never been swept")
}

// TestATargetNamingNoAxisThisChainReads covers an empty answer.
func (s *ReachPublicTestSuite) TestATargetNamingNoAxisThisChainReads() {
	s.answers(sdk.Reaching{Rig: "matt-freeman", Genre: "punk"})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "Nothing to aim at")
}

// TestWhatTheSDKRefusesIsReported covers an answer that could not be made.
func (s *ReachPublicTestSuite) TestWhatTheSDKRefusesIsReported() {
	wanted := errors.New("no block in this chain has been swept")

	s.sdk.EXPECT().Reach(gomock.Any(), gomock.Any()).
		Return(sdk.Reaching{}, wanted)

	s.Require().ErrorIs(
		Reach(context.Background(), buffer(), s.opts()), wanted)
}

// TestTheAskCarriesEveryTreeTheCallerNamed covers the flags reaching the SDK.
func (s *ReachPublicTestSuite) TestTheAskCarriesEveryTreeTheCallerNamed() {
	var got sdk.ReachAsk

	s.sdk.EXPECT().Reach(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in sdk.ReachAsk) (sdk.Reaching, error) {
			got = in

			return sdk.Reaching{Worth: true}, nil
		})

	opts := s.opts()
	opts.Corpus = "somewhere/music"
	opts.Sweeps = "somewhere/sweeps"

	s.Require().NoError(Reach(context.Background(), buffer(), opts))

	s.Require().Equal("matt-freeman", got.RigID)
	s.Require().Equal("punk", got.Genre)
	s.Require().Equal("somewhere/music", got.Corpus)
	s.Require().Equal("somewhere/sweeps", got.Sweeps)
}

// TestAxesReachableAloneAndNotTogetherIsItsOwnAnswer is the one per-axis could
// never give.
//
// Every axis inside what its own controls can move, and no single set of
// positions reaching them at once. That is the ordinary shape of the problem
// and it is what somebody means by asking whether a rig can sound like
// something.
func (s *ReachPublicTestSuite) TestAxesReachableAloneAndNotTogetherIsItsOwnAnswer() {
	s.answers(sdk.Reaching{
		Rig: "matt-freeman", Genre: "punk", Worth: true,
		Axes: []solve.Verdict{
			{Figure: audio.KeyCentroid, Gap: 11, Shown: true, Within: true, Together: 4.2},
			{Figure: audio.KeyLow, Gap: 2, Shown: true, Within: true, Together: 0.3},
		},
	})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "no one set of positions reaches them together")
	s.Require().Contains(w.String(), "centroid by 4.2")
	s.Require().NotContains(w.String(), "low by",
		"an axis the joint solve met is not named as missed")
}

// TestAChainWithNoMeasuredSlopeSaysSoRatherThanNamingNothing covers the
// sentence with a hole in it.
//
// Not arrived and no axis over a tolerance means no axis could be solved for
// at all, so the joint answer is absent rather than negative.
func (s *ReachPublicTestSuite) TestAChainWithNoMeasuredSlopeSaysSoRatherThanNamingNothing() {
	s.answers(sdk.Reaching{
		Rig: "matt-freeman", Genre: "punk", Worth: true,
		Axes: []solve.Verdict{
			{Figure: audio.KeyCentroid, Gap: 11, Shown: true, Within: true},
		},
	})

	w := buffer()
	s.Require().NoError(Reach(context.Background(), w, s.opts()))

	s.Require().Contains(w.String(), "no control in")
	s.Require().NotContains(w.String(), "misses .")
}

func TestReachPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReachPublicTestSuite))
}
