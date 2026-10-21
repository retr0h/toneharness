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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// ChainTestSuite covers reading what is loaded, and the guard that says
// the reading is of the loop rather than of the chain.
type ChainTestSuite struct {
	suite.Suite

	dry string
}

func (s *ChainTestSuite) SetupTest() {
	s.dry = filepath.Join("..", "..", "resources", "dry", "bass-di-short.wav")
}

// TestChain covers Chain, which reads what the pedal is playing, and changes
// nothing.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChainTestSuite) TestChain() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The primitive nothing provided.
			name: "it reads what is loaded and changes nothing",
			then: func() {
				w := buffer()

				s.Require().NoError(Chain(context.Background(), w, ChainOptions{
					Bench: bench{}, Dry: s.dry, Seconds: 0.1, Takes: 2,
				}))

				said := w.String()
				s.Require().Contains(said, "READS")
				s.Require().Contains(said, "WANDERS")
				s.Require().Contains(said, "below 250Hz")
				s.Require().Contains(said, "centroid")
			},
		},
		{
			// A missing signal.
			name: "a reference that is not there",
			then: func() {
				s.Require().Error(Chain(context.Background(), buffer(), ChainOptions{
					Bench: bench{}, Dry: filepath.Join(s.T().TempDir(), "nothing.wav"),
					Seconds: 0.1, Takes: 2,
				}))
			},
		},
		{
			// Both reads.
			//
			// The floor is measured first and the chain second, and a bench
			// that fails on either has nothing to report: a figure against a
			// floor nobody took is a figure about nothing.
			name: "chain reports a bench that stops answering",
			then: func() {
				for _, tt := range []struct {
					name  string
					after int
				}{
					{"on the floor", 0},
					{"after the floor, on the chain", 3},
				} {
					s.Run(tt.name, func() {
						var calls int

						err := Chain(context.Background(), buffer(), ChainOptions{
							Bench:   stopping{after: tt.after, calls: &calls},
							Dry:     s.dry,
							Seconds: 0.1, Takes: 2,
						})

						s.Require().Error(err)
					})
				}
			},
		},
		{
			// The warning rather than the refusal.
			//
			// This command exists to be pointed at a loop somebody is setting
			// up, so an oscillating one is what they want told about rather
			// than stopped for. Every other command refuses; this one prints
			// and carries on.
			name: "chain says so when the loop is squealing",
			then: func() {
				w := buffer()

				s.Require().NoError(Chain(context.Background(), w, ChainOptions{
					Bench: bench{clipped: true}, Dry: s.dry, Seconds: 0.1, Takes: 2,
					EndsInACab: true,
				}))

				s.Require().Contains(w.String(), "READS", "and it still prints the reading")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// TestSquealing covers every case Squealing answers.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChainTestSuite) TestSquealing() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The invariant.
			//
			// A cabinet is a loudspeaker and a loudspeaker is a low pass, so
			// a chain ending in one can remove energy above 2kHz and cannot
			// add it. Measured on an HX Stomp: a compressor into an Ampeg SVT
			// into an 8x10 gave back 84.3% of its energy above 2kHz against
			// 0.01% in the reference, and the same chain with the amplifier's
			// output at 0.5 gave back 0.0% at a level three decibels lower.
			name: "squealing catches a chain that brightened what it was given",
			then: func() {
				dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

				say, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.843}, dry, nil, true, 0)

				s.Require().True(bad)
				s.Require().Contains(say, "84.3%")
				s.Require().Contains(say, "A loudspeaker cannot add that")
			},
		},
		{
			// The chain that is fine.
			name: "squealing leaves a clean reading alone",
			then: func() {
				dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

				_, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.0}, dry, nil, true, 0)

				s.Require().False(bad, "a cabinet removing high end is a cabinet working")
			},
		},
		{
			// The limit of it.
			//
			// Only a chain ending in a loudspeaker makes the high band a
			// one-way street. Anything else may legitimately be brighter than
			// what it was given, and a drive certainly is.
			name: "squealing says nothing about a chain with no cabinet",
			then: func() {
				_, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.9},
					map[audio.Figure]float64{audio.KeyHigh: 0.0001}, nil, false, 0)

				s.Require().False(bad)
			},
		},
		{
			// A figure nothing measured.
			name: "squealing needs both readings",
			then: func() {
				_, bad := Squealing(
					map[audio.Figure]float64{}, map[audio.Figure]float64{audio.KeyHigh: 0}, nil, true, 0)
				s.Require().False(bad)

				_, bad = Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.9}, map[audio.Figure]float64{}, nil, true, 0)
				s.Require().False(bad)
			},
		},
		{
			// The margin.
			//
			// The invariant is exact and the measurement is not. A chain with
			// 25dB of headroom reads 0.1% above 2kHz against the reference's
			// 0.01% at a level of -55dB, which is the converter's own
			// broadband floor and not a loudspeaker adding anything. Clean
			// readings sit between 0.0% and 0.1% and squealing ones between
			// 39% and 84%, with nothing in between.
			name: "a quiet clean reading is not a squeal",
			then: func() {
				dry := map[audio.Figure]float64{audio.KeyHigh: 0.0001}

				_, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.001}, dry, nil, true, 0)
				s.Require().False(bad, "a tenth of a percent is the floor")

				_, bad = Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.391}, dry, nil, true, 0)
				s.Require().True(bad, "the quietest squeal measured was 39%")
			},
		},
		{
			// The drive.
			//
			// A chain ending in a cabinet may hold something that brightens.
			// An Arbiter Fuzz Face into an Ampeg SVT into an 8x10 read 6.5%
			// above 2kHz through the open loop, and the blocks' own sweeps sum
			// to 6.437%, so the reading is what the chain is rather than the
			// loop oscillating.
			name: "a drive is allowed the brightness its own sweeps account for",
			then: func() {
				dry := map[audio.Figure]float64{audio.KeyHigh: 0.0}
				now := map[audio.Figure]float64{audio.KeyHigh: 0.065}

				_, bad := Squealing(now, dry, nil, true, 0.06437)
				s.Require().False(bad,
					"the chain's own blocks account for 6.437% of 6.5%")

				say, bad := Squealing(now, dry, nil, true, 0)
				s.Require().True(bad,
					"and with nothing justified it is the old refusal")
				s.Require().Contains(say, "6.5%")
			},
		},
		{
			// What justifying does not buy.
			//
			// The ceiling moves with the chain and a squeal still clears it.
			// On that same fuzz chain the quietest squeal anybody has recorded
			// is 39%, against a ceiling of 6.437% plus the margin.
			name: "justifying a drive does not let a squeal through",
			then: func() {
				_, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.391},
					map[audio.Figure]float64{audio.KeyHigh: 0.0},
					nil, true, 0.06437)

				s.Require().True(bad)
			},
		},
		{
			// The refusal says what it was measured against.
			//
			// Three numbers rather than two, because "it gave back more than
			// the reference" is no longer the whole claim: what the chain's
			// own blocks account for is the other half of it, and somebody
			// reading the refusal cannot check it otherwise.
			name: "the refusal names what the chain's blocks account for",
			then: func() {
				say, bad := Squealing(
					map[audio.Figure]float64{audio.KeyHigh: 0.843},
					map[audio.Figure]float64{audio.KeyHigh: 0.0001},
					nil, true, 0.064)

				s.Require().True(bad)
				s.Require().Contains(say, "84.3%")
				s.Require().Contains(say, "6.4%")
				s.Require().Contains(say, "its own blocks account for")
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}

// TestJustifiedBy covers justifiedBy, the ceiling the squeal guard compares a
// reading against.
//
// Against the committed curves rather than fixtures, because the figures are
// the whole claim: the rule is "summed, not maxed" and it rests on the sum
// agreeing with what the chain measured. A fixture would let the rule survive
// the library changing underneath it.
func (s *ChainTestSuite) TestJustifiedBy() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	sweeps := filepath.Join("..", "..", "resources", "sweeps", "hx-stomp")

	fuzzChain := plan.Plan{Blocks: []plan.Block{
		{Model: "HD2_CompressorDeluxeComp", Pos: 0},
		{Model: "HD2_DistArbitratorFuzz", Pos: 1},
		{Model: "HD2_AmpSVBeastBrt", Pos: 2},
		{Model: "HD2_CabMicIr_8x10SVTAV", Pos: 3},
	}}

	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The reading the rule rests on.
			//
			// The chain `--extend` proposed for matt-freeman read 6.5% above
			// 2kHz through the open loop. Its blocks sum to 6.437%, which is
			// why the sum is the rule: the maximum is the amplifier's 4.941%
			// and would have refused it.
			name: "a chain is allowed what its blocks separately reached",
			then: func() {
				got := justifiedBy(fuzzChain, cat, sweeps)

				s.Require().InDelta(0.064, got, 0.005,
					"the fuzz chain's blocks account for about 6.4%")

				var most float64
				for _, b := range fuzzChain.Blocks {
					most = max(most, brightest(
						filepath.Join(sweeps, string(b.Model)+".json")))
				}

				s.Require().Less(most/100, 0.065,
					"the brightest single block does not cover the chain, "+
						"which is why these are summed")
			},
		},
		{
			// A release.
			name: "no sweeps leaves the bare invariant in place",
			then: func() {
				s.Require().Zero(justifiedBy(fuzzChain, cat, ""))
			},
		},
		{
			// A block nothing swept, and a model the catalog does not carry.
			name: "a block with no curves justifies nothing",
			then: func() {
				s.Require().Zero(brightest(
					filepath.Join(sweeps, "HD2_NoSuchBlock.json")))

				made := plan.Plan{Blocks: []plan.Block{
					{Model: "HD2_NoSuchBlock", Pos: 0},
				}}
				s.Require().Zero(justifiedBy(made, cat, sweeps))
			},
		},
		{
			// Two of a kind.
			//
			// Summed per block rather than per model, because a chain may hold
			// two drives and each of them brightens.
			name: "a chain holding two of a kind counts both",
			then: func() {
				one := plan.Plan{Blocks: []plan.Block{
					{Model: "HD2_DistArbitratorFuzz", Pos: 0},
				}}
				two := plan.Plan{Blocks: []plan.Block{
					{Model: "HD2_DistArbitratorFuzz", Pos: 0},
					{Model: "HD2_DistArbitratorFuzz", Pos: 1},
				}}

				s.Require().InDelta(
					2*justifiedBy(one, cat, sweeps),
					justifiedBy(two, cat, sweeps), 1e-9)
			},
		},
		{
			// A file that will not parse.
			name: "a sweep that is not curves justifies nothing",
			then: func() {
				s.Require().Zero(brightest(s.dry))
			},
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			tt.then()
		})
	}
}

// TestTheReferenceIsMeasuredRatherThanAssumed covers what the gap is against.
//
// The comparison is to this recording rather than to a number somebody chose,
// so a different reference moves the line with it.
func (s *ChainTestSuite) TestTheReferenceIsMeasuredRatherThanAssumed() {
	signal, err := reference(s.dry, 1)
	s.Require().NoError(err)

	got := figuresOfDry(signal)

	s.Require().Less(got[audio.KeyHigh], 0.01,
		"a bass DI has almost nothing above 2kHz")
	s.Require().Greater(got[audio.KeyLow], 0.5)
}

func TestChainTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ChainTestSuite))
}

// TestReferenceIsFor covers referenceIsFor, which is what a reference
// recording was played on.
//
// One method and one table, so a case is a row rather than a file.
func (s *ChainTestSuite) TestReferenceIsFor() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// What a reading says it is about.
			//
			// Every figure here is a figure about one instrument. A guitar
			// rig ranked against readings taken with a bass is ranked against
			// the wrong distribution, and until this was recorded the only
			// trace of it was a filename.
			name: "instrument of reads it off the reference",
			then: func() {
				tests := []struct {
					dry  string
					want string
				}{
					{filepath.Join("resources", "dry", "bass-di-short.wav"), "bass"},
					{filepath.Join("resources", "dry", "guitar-di.wav"), "guitar"},
					{"BASS-DI.WAV", "bass"},
					{filepath.Join("somewhere", "else", "take-3.wav"), ""},
				}

				for _, tt := range tests {
					s.Run(tt.dry, func() {
						s.Require().Equal(tt.want, referenceIsFor(tt.dry))
					})
				}
			},
		},
		{
			// The honest empty.
			//
			// A reading that does not know what was played through it should
			// not say. The point of recording it is that somebody can tell,
			// and a guess defeats that.
			name: "a reading that cannot name its instrument claims none",
			then: func() {
				s.Require().Empty(referenceIsFor("reference.wav"))
			},
		},
	} {
		s.Run(tt.name, func() {
			// A row gets the same fresh state a method used to get.
			s.SetupTest()

			tt.then()
		})
	}
}
