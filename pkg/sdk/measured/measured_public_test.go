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
package measured_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// MeasuredPublicTestSuite covers reading measurements and ranking against a
// target.
type MeasuredPublicTestSuite struct {
	suite.Suite

	lib measured.Library
}

func (s *MeasuredPublicTestSuite) SetupTest() {
	f, err := os.Open(filepath.Join("testdata", "library.json"))
	s.Require().NoError(err)

	defer func() { _ = f.Close() }()

	s.lib, err = measured.Load(f)
	s.Require().NoError(err)
}

// dark is a bass target: most of the energy low, centroid a few hundred Hz.
func (s *MeasuredPublicTestSuite) dark() measured.Figures {
	return measured.Figures{Centroid: 300, Low: 80, Mid: 15, High: 5}
}

// TestLoad covers reading a measured library.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuredPublicTestSuite) TestLoad() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary case, including the baseline.
			//
			// The baseline matters as much as the readings. Without it a figure says
			// nothing: 95 Hz is not what a block does to a bass, it is what the bass
			// already was.
			name: "reads a library",
			then: func() {
				s.Require().Equal("HX Stomp", s.lib.Device)
				s.Require().True(s.lib.Isolated)
				s.Require().InDelta(95.3, s.lib.Baseline.Centroid, 0.01)
				s.Require().Equal("abc", s.lib.Reference.SHA256)
				s.Require().Len(s.lib.Blocks, 7)
			},
		},
		{
			// Load, which reads a library of measurements.
			//
			// One method and one table, so a case is a row rather than a file.
			name: "load",
			then: func() {
				for _, tt := range []struct {
					name string
					then func()
				}{
					{
						// The three ways reading fails.
						name: "load refuses what is not a library",
						then: func() {
							tests := []struct {
								name string
								give string
								want string
							}{
								{name: "not JSON", give: "{", want: "decoding the measurements"},
								{
									// A document that parses and names nothing is worse than one
									// that fails, because every lookup against it answers "no
									// block is close" rather than "there is no library".
									name: "no blocks at all",
									give: `{"device":"HX Stomp"}`,
									want: "name no blocks",
								},
							}

							for _, tt := range tests {
								s.Run(tt.name, func() {
									_, err := measured.Load(strings.NewReader(tt.give))

									s.Require().ErrorContains(err, tt.want)
								})
							}
						},
					},
					{
						// The reader itself failing.
						name: "load reports a read failure",
						then: func() {
							_, err := measured.Load(broken{})

							s.Require().ErrorContains(err, "reading the measurements")
						},
					},
				} {
					s.Run(tt.name, func() {
						// A row gets the same fresh state a method used to get.
						s.SetupTest()

						tt.then()
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			tt.then()
		})
	}
}

// TestNearest covers Nearest, which ranks the blocks of one category by how
// close they sit to a target.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuredPublicTestSuite) TestNearest() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary case.
			name: "nearest ranks by distance",
			then: func() {
				got := s.lib.Nearest("amp", s.dark(), measured.Spectral())

				s.Require().NotEmpty(got)
				s.Require().InDelta(0, got[0].Distance, 0.001,
					"a block measuring exactly the target is a distance of zero from it")
				s.Require().Equal("AmpBright", got[len(got)-1].ID,
					"the brightest amp is furthest from a dark target")
			},
		},
		{
			// A cab never answering for an amp.
			name: "nearest stays in its category",
			then: func() {
				for _, m := range s.lib.Nearest("amp", s.dark(), measured.Spectral()) {
					s.Require().Equal(catalog.CategoryAmp, m.Category)
				}

				s.Require().Len(s.lib.Nearest("cab", s.dark(), measured.Spectral()), 1)
			},
		},
		{
			// The point of the two flags.
			//
			// A clipped reading's spectrum is the clipping's, not the
			// block's, so ranking it puts a bright-looking reading of the
			// converters at the top of a list of bright amplifiers. A refusal
			// has no figures at all, and a zero centroid would read as the
			// darkest block on the device.
			name: "nearest leaves out what it cannot trust",
			then: func() {
				got := s.lib.Nearest("amp", s.dark(), measured.Spectral())

				for _, m := range got {
					s.Require().NotEqual("AmpClipped", m.ID, "a clipped reading was ranked")
					s.Require().NotEqual("AmpRefused", m.ID, "a refusal was ranked")
				}

				s.Require().Len(got, 4)
			},
		},
		{
			// Two blocks that measure identically.
			//
			// Ranked by identifier after distance, because a ranking that
			// reshuffles between runs is one nobody can act on twice.
			name: "nearest is stable",
			then: func() {
				first := s.lib.Nearest("amp", s.dark(), measured.Spectral())

				for range 20 {
					s.Require().Equal(first, s.lib.Nearest("amp", s.dark(), measured.Spectral()))
				}

				s.Require().Equal("AmpDark", first[0].ID,
					"of three blocks measuring the target exactly, the first by identifier")
			},
		},
		{
			// Weighing a figure the default ignores.
			name: "weights can be given outright",
			then: func() {
				want := s.dark()
				want.Level = -20

				got := s.lib.Nearest("amp", want, measured.Weights{Level: 1})

				s.Require().Equal("AmpLoud", got[len(got)-1].ID,
					"weighed on level alone, the one 17 dB away is furthest")
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

// TestSpectralIgnoresLoudness is a decision worth a test.
//
// A block that is right and quiet is right: the next block's level control
// fixes it for nothing. Weighing loudness would rank a loud wrong answer over
// a quiet correct one, which is how a chooser ends up preferring whatever
// clips hardest.
func (s *MeasuredPublicTestSuite) TestSpectralIgnoresLoudness() {
	got := s.lib.Nearest("amp", s.dark(), measured.Spectral())

	var dark, loud float64

	for _, m := range got {
		switch m.ID {
		case "AmpDark":
			dark = m.Distance
		case "AmpLoud":
			loud = m.Distance
		}
	}

	s.Require().Equal(dark, loud,
		"two blocks with the same spectrum and 17 dB between them rank alike")
}

// TestMeasured covers Measured, which returns whether this block has a
// reading worth using.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuredPublicTestSuite) TestMeasured() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The flags directly.
			name: "measured says which readings are usable",
			then: func() {
				s.Require().True(s.lib.Blocks["AmpDark"].Measured())
				s.Require().False(s.lib.Blocks["AmpClipped"].Measured())
				s.Require().False(s.lib.Blocks["AmpRefused"].Measured())
			},
		},
		{
			// The guard the flag missed.
			//
			// Clipping is not a cliff at the number a guard picks. `measure
			// blocks` marks Clipped at half a decibel below full scale, and
			// ten readings sit between that and two decibels below it with
			// the converters plainly in them: HD2_AmpSVT4Pro, an Ampeg SVT-4
			// Pro a shipped rig names, reads 3.24% of its energy low and
			// 96.74% high. An Ampeg does not do that.
			name: "a reading at the ceiling is not the blocks",
			then: func() {
				at := func(level float64) measured.Block {
					return measured.Block{Figures: measured.Figures{Level: level}}
				}

				s.Require().False(at(-0.94).Measured(), "an amplifier reading 97% high")
				s.Require().False(at(-1.99).Measured())
				s.Require().True(at(-2.01).Measured())
				s.Require().True(at(-20).Measured(), "the ordinary reading")
			},
		},
		{
			// The campaign's own evidence.
			//
			// The library this replaced had ten readings the near-ceiling
			// line excluded, two of them bass amplifiers, one reading 96.74%
			// of its energy in the high band. An Ampeg does not do that; the
			// converters did, because the chain was measured at full output
			// and fed itself.
			//
			// Measured at -30dB nothing comes near the ceiling, so the guard
			// now excludes nothing. It stays because headroom is a setting
			// and settings get changed.
			name: "no shipped reading sits at the converters ceiling",
			then: func() {
				lib, err := measured.BuiltIn()
				s.Require().NoError(err)

				loud := make([]string, 0, 1)

				for id, block := range lib.Blocks {
					if block.Refused != "" {
						continue
					}

					if block.Clipped || !block.Measured() {
						loud = append(loud, id)
					}
				}

				s.Require().Empty(loud,
					"a reading this loud is the converters' shaping rather than the block's")
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

// broken is a reader that always fails.
type broken struct{}

func (broken) Read(
	[]byte,
) (int, error) {
	return 0, errors.New("no")
}

func TestMeasuredPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasuredPublicTestSuite))
}

// TestTheShippedLibraryNamesItsInstrumentAndHeadroom holds the packed
// measurements to the two fields that say whether they mean anything.
//
// The counterpart of the genre check, and here for the same reason: an unnamed
// instrument is indistinguishable from a library packed before the field
// existed, so without this a stale pack reads as a claim about no instrument in
// particular. Every figure in here is a figure about a bass.
//
// Headroom is the one that decides whether a reading is of the chain at all. The
// measuring lead makes the chain feed itself, and the library this replaced was
// taken without the output turned down: its full amplifiers read a median 46.4%
// of their energy above 2kHz where the preamps of the same circuits read 0.047%,
// and its bass amplifiers read centroids between 1,250 and 8,374Hz where they
// measure between 119 and 325.
func (s *MeasuredPublicTestSuite) TestTheShippedLibraryNamesItsInstrumentAndHeadroom() {
	lib, err := measured.BuiltIn()
	s.Require().NoError(err)

	s.Require().NotEmpty(lib.Instrument,
		"the packed library names no instrument, so it predates the field: "+
			"re-pack with `just pack-measured` after a campaign")

	// One of the two, not headroom specifically. This asserted a negative
	// headroom until the rig changed under it, on the reasoning that a library
	// taken at full output is a library of the measuring loop. That was true of
	// the rig where the pedal played its own reference back into itself, and is
	// false of the one where the computer plays: there the loop is open, the
	// trim buys nothing, and the level that matters is the computer's own.
	//
	// So what has to hold is that the library says what level it was taken at
	// by one mechanism or the other. Both absent is a library nobody can
	// reproduce, and the test that pinned the wrong one was a test about a
	// cable.
	s.Require().True(lib.Headroom < 0 || lib.Volume > 0,
		"the library records neither a headroom trim nor a pinned output level, "+
			"so nothing says what level these were taken at")
}

// TestRunKey covers identifying a measuring session by what makes its readings
// comparable.
//
// Task 257. A digest of the run's own fields rather than a clock, so two
// sessions on the same rig at the same settings against the same baseline
// collapse into one entry, which is the right answer: downstream they are one
// run.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuredPublicTestSuite) TestRunKey() {
	base := func() measured.Run {
		return measured.Run{
			Reference:  measured.Reference{SHA256: "abc", Seconds: 8},
			Instrument: "bass",
			Headroom:   -30,
			Volume:     38,
			Rig:        []string{"External Headphones", "HX Stomp"},
			Open:       true,
			PinnedOn:   "External Headphones",
			Baseline:   measured.Figures{Centroid: 155.27, Level: -58.55},
		}
	}

	tests := []struct {
		name string
		// moved is what differs from the run above, and same whether the key
		// is expected to match it anyway.
		moved func(*measured.Run)
		same  bool
	}{
		{
			name:  "the same settings key the same",
			moved: func(*measured.Run) {},
			same:  true,
		},
		{
			// Two laptops at 38 put out different levels, and an amplifier's
			// distortion depends on how hard it is driven, so a run at another
			// volume measured every amplifier as a different amplifier.
			name:  "another volume is another run",
			moved: func(r *measured.Run) { r.Volume = 50 },
		},
		{
			// The one that silently invalidates the rest: a level pinned on a
			// device that is not in the signal path is a number about nothing.
			name:  "another device the level was pinned on is another run",
			moved: func(r *measured.Run) { r.PinnedOn = "MacBook Pro Speakers" },
		},
		{
			// The difference that reads an Ampeg SVT at 12.7% of its energy
			// in the mids on one rig and 4.0% on the other.
			name:  "a closed loop is another run",
			moved: func(r *measured.Run) { r.Open = false },
		},
		{
			name:  "another rig is another run",
			moved: func(r *measured.Run) { r.Rig = []string{"HX Stomp"} },
		},
		{
			// A figure says nothing without the empty loop it was read
			// against, so two baselines are two runs however alike the rest.
			name:  "another baseline is another run",
			moved: func(r *measured.Run) { r.Baseline.Centroid = 160 },
		},
		{
			// A different reference invalidates the lot, the way changing a
			// record invalidates the words derived from it.
			name:  "another reference signal is another run",
			moved: func(r *measured.Run) { r.Reference.SHA256 = "def" },
		},
		{
			name:  "another instrument is another run",
			moved: func(r *measured.Run) { r.Instrument = "guitar" },
		},
		{
			name:  "another headroom trim is another run",
			moved: func(r *measured.Run) { r.Headroom = -12 },
		},
	}

	was := base().Key()
	s.Require().Len(was, 12, "twelve hex characters, short enough to read")

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := base()
			tt.moved(&got)

			if tt.same {
				s.Require().Equal(was, got.Key())

				return
			}

			s.Require().NotEqual(was, got.Key())
		})
	}
}

// TestRunOf covers which measuring session one reading came out of.
//
// One method and one table, so a case is a row rather than a file.
func (s *MeasuredPublicTestSuite) TestRunOf() {
	earlier := measured.Run{
		Volume:   20,
		Baseline: measured.Figures{Centroid: 100},
	}

	lib := measured.Library{
		Volume:   38,
		Baseline: measured.Figures{Centroid: 155.27},
		Runs:     map[string]measured.Run{earlier.Key(): earlier},
	}

	tests := []struct {
		name  string
		block measured.Block
		// volume is the run's, and named whether the block named its own run.
		volume int
		named  bool
	}{
		{
			// A reading that says which run took it is read from that run,
			// which is what stops a resumed file relabelling it.
			name:   "a reading that names its run is read from it",
			block:  measured.Block{Run: earlier.Key()},
			volume: 20, named: true,
		},
		{
			// All 614 in the committed library. The header is what such a
			// reading was always implicitly claiming, and false is how a
			// caller tells a claim from a record.
			name:   "a reading that names none falls back to the header",
			block:  measured.Block{},
			volume: 38, named: false,
		},
		{
			// A stamp pointing at a run the file does not hold, which is a
			// file somebody edited. The header is the honest answer and the
			// false says it came from there.
			name:   "a reading naming a run nothing holds falls back too",
			block:  measured.Block{Run: "deadbeefcafe"},
			volume: 38, named: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, named := lib.RunOf(tt.block)

			s.Require().Equal(tt.named, named)
			s.Require().Equal(tt.volume, got.Volume)
		})
	}
}

// TestHeaderIsTheLibrarysOwnRun covers reading the library's fields back as the
// run that wrote it.
//
// What `measure blocks` stamps its readings with, so the header and the run
// entry cannot describe different things.
func (s *MeasuredPublicTestSuite) TestHeaderIsTheLibrarysOwnRun() {
	got := s.lib.Header()

	s.Require().Equal(s.lib.Volume, got.Volume)
	s.Require().Equal(s.lib.Baseline, got.Baseline)
	s.Require().Equal(s.lib.Instrument, got.Instrument)
	s.Require().Equal(s.lib.Reference, got.Reference)
	s.Require().NotEmpty(got.Key())
}

// TestEveryShippedReadingNamesARunTheFileHolds holds the packed measurements to
// what they say about themselves.
//
// Task 257's third item, and the counterpart of the batch validation that holds
// every curve file to its own rig. Two states are honest and the third is not:
//
//   - every reading names a run the file holds, which is a library that knows
//     where each of its figures came from
//   - no reading names one, which is the 614 packed before this was recorded.
//     Silence, and `RunOf` says so by answering false
//
// What may not happen is a mixture of the two with no entry for the silent
// half, because then the readings that name nothing are being described by a
// header built for the ones that do. That is the relabelling this exists to
// catch, and `resume` is where it would come from.
func (s *MeasuredPublicTestSuite) TestEveryShippedReadingNamesARunTheFileHolds() {
	lib, err := measured.BuiltIn()
	s.Require().NoError(err)
	s.Require().NotEmpty(lib.Blocks)

	named, silent := 0, 0

	for id, block := range lib.Blocks {
		if block.Run == "" {
			silent++

			continue
		}

		named++

		s.Require().Contains(lib.Runs, block.Run,
			"%s names a run this library does not hold, so nothing can say "+
				"what rig its figures are about", id)
	}

	if named > 0 && silent > 0 {
		s.Require().Fail(
			"this library holds readings from a recorded run beside readings " +
				"that name none, so the silent ones are described by a header " +
				"built for the others")
	}

	// And the header is readable as a run whichever state the file is in, so a
	// caller always has an answer and knows which it has.
	one, from := lib.RunOf(measured.Block{})
	s.Require().False(from, "a reading naming no run is not a record")
	s.Require().Equal(lib.Baseline, one.Baseline)
}
