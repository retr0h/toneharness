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
package tone_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// LoadPublicTestSuite covers reading a request and a setup off disk.
type LoadPublicTestSuite struct {
	suite.Suite
}

// TestLoad covers Load, which reads a request and checks it against its own
// contract.
//
// One method and one table, so a case is a row rather than a file.
func (s *LoadPublicTestSuite) TestLoad() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The ordinary case.
			name: "reads a request",
			then: func() {
				spec, err := tone.Load(strings.NewReader(`
schema: ToneSpec
id: dirnt-ish
ask:
  genre: [pop-punk, punk]
  words:
    - term: bright
    - term: tight-low-end
      evidence: [{ kind: llm }]
  like:
    artist: Mike Dirnt
    years: { from: 1994, to: 2004 }
  nudges:
    - word: darker
      steps: 2
rig:
  instrument: bass
  chain:
    - role: amp
      gear: Ampeg SVT
`))

				s.Require().NoError(err)
				// The gear is the half that is required, and the ask is why. Both read
				// off one document, which is what version 2 of the contract merged.
				s.Require().Equal("dirnt-ish", spec.Id)
				s.Require().Equal(tone.InstrumentBass, spec.Rig.Instrument)
				s.Require().Equal("Ampeg SVT", spec.Rig.Chain[0].Gear)

				ask := spec.Ask
				s.Require().Equal([]string{"pop-punk", "punk"}, ask.Genre,
					"both, because the corpus tags these records with both")
				s.Require().Len(*ask.Words, 2)
				s.Require().Equal("bright", (*ask.Words)[0].Term)
				// A word carries why it is believed, because that is what sizes how far it
				// moves a control. The first here carries none, which is legal and is what
				// a request somebody typed looks like.
				s.Require().Nil((*ask.Words)[0].Evidence)
				s.Require().Equal("tight-low-end", (*ask.Words)[1].Term)
				s.Require().Len(*(*ask.Words)[1].Evidence, 1)
				s.Require().Equal("Mike Dirnt", *ask.Like.Artist)
				s.Require().Equal(1994, ask.Like.Years.From)
				s.Require().Equal("darker", (*ask.Nudges)[0].Word)
			},
		},
		{
			// Why the raw document is checked first.
			//
			// Decoding drops what the types have no field for, so a request
			// checked after decoding is checked with its own mistake already
			// removed: the line would be gone and nothing said about it.
			name: "a misspelt field is refused",
			then: func() {
				_, err := tone.Load(strings.NewReader("schema: ToneSpec\ngnere: punk\n"))

				s.Require().ErrorIs(err, tone.ErrInvalid)
				s.Require().Contains(err.Error(), "gnere")
			},
		},
		{
			// The rig half, which the schema checks the same way.
			//
			// These were pkg/sdk/rig's own loader's tests until version 2 merged the
			// two contracts. There is one loader now, and the rig is a section of
			// what it reads, so the section's own refusals are checked here.
			name: "a rig the contract will not take is refused",
			then: func() {
				for _, tt := range []struct {
					name string
					rig  string
					says string
				}{
					{
						name: "a field nobody spelled right, inside the chain",
						rig: "  instrument: bass\n  chain:\n" +
							"    - {role: amp, gear: Ampeg SVT, gera: nonsense}\n",
						says: `property "gera" is unsupported`,
					},
					{
						// Every one of these is a thing a person writes, so each lives on
						// the ask and the rig refuses it outright rather than checking its
						// shape.
						name: "what a person writes, which a rig does not carry",
						rig: "  instrument: bass\n  technique: {attack: pick}\n" +
							"  chain:\n    - {role: amp, gear: Ampeg SVT}\n",
						says: `property "technique" is unsupported`,
					},
					{
						name: "a link that is not one",
						rig: "  instrument: bass\n" +
							"  evidence:\n    - {kind: cited, url: mikes-website}\n" +
							"  chain:\n    - {role: amp, gear: Ampeg SVT}\n",
						says: "rig.evidence[0].url",
					},
					{
						// Where in a recording, so it has to be a time.
						name: "a place in a recording, given in words",
						rig: "  instrument: bass\n  evidence:\n" +
							"    - {kind: video, url: \"https://x.test/v\", at: the end}\n" +
							"  chain:\n    - {role: amp, gear: Ampeg SVT}\n",
						says: "rig.evidence[0].at",
					},
					{
						name: "a rig holding no chain",
						rig:  "  instrument: bass\n  chain: []\n",
						says: "chain minimum number of items is 1",
					},
				} {
					s.Run(tt.name, func() {
						_, err := tone.Load(strings.NewReader(
							"schema: ToneSpec\nid: x\nrig:\n" + tt.rig))

						s.Require().ErrorIs(err, tone.ErrInvalid)
						s.Require().Contains(err.Error(), tt.says)
					})
				}
			},
		},
		{
			// The gear is the half that is required.
			//
			// An ask naming no gear and nothing measurable is a sound nothing can
			// model, which is what the solver has always said and what the schema
			// used to allow anyway.
			name: "a document naming no rig is refused",
			then: func() {
				_, err := tone.Load(strings.NewReader(
					"schema: ToneSpec\nid: x\nask:\n  genre: [punk]\n"))

				s.Require().ErrorIs(err, tone.ErrInvalid)
				s.Require().Contains(err.Error(), "rig")
			},
		},
		{
			// A file holding a list.
			name: "a document that is not fields is refused",
			then: func() {
				_, err := tone.Load(strings.NewReader("- one\n- two\n"))

				s.Require().ErrorIs(err, tone.ErrInvalid)
				s.Require().Contains(err.Error(), "is not a set of fields")
			},
		},
		{
			// A file that is not YAML at all.
			name: "unreadable y a m l is refused",
			then: func() {
				_, err := tone.Load(strings.NewReader("\tschema: [unclosed\n"))

				s.Require().Error(err)
				s.Require().Contains(err.Error(), "decoding the ToneSpec")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestLoadSetup covers LoadSetup, which reads what somebody has and checks it
// against its own contract.
//
// One method and one table, so a case is a row rather than a file.
func (s *LoadPublicTestSuite) TestLoadSetup() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The other document.
			name: "reads a setup",
			then: func() {
				setup, err := tone.LoadSetup(strings.NewReader(`
schema: Setup
device:
  model: HX Stomp
instruments:
  - gear: Fender Jazz Bass
    strings: flat
    default: true
owns:
  - kind: ir
    name: Owned 4x10
    slot: 3
`))

				s.Require().NoError(err)
				s.Require().Equal("HX Stomp", setup.Device.Model)
				s.Require().Equal(tone.StringsFlat, *(*setup.Instruments)[0].Strings)
				s.Require().Equal(tone.OwnedIR, (*setup.Owns)[0].Kind)
			},
		},
		{
			// Plays_into.
			//
			// Optional, because it is a thing somebody may not have said and
			// a setup written before the field existed is still a setup. Four
			// spellings and nothing else, because the two amplifier entries
			// are different questions — the instrument input has the
			// amplifier's own preamp in front of its speaker, and the effects
			// return does not — and a free-string field would let somebody
			// write "amp" and mean either.
			name: "what the pedal is plugged into is optional and spelt",
			then: func() {
				held, err := tone.LoadSetup(strings.NewReader(
					"schema: Setup\ndevice:\n  model: HX Stomp\n"))

				s.Require().NoError(err)
				s.Require().Nil(held.PlaysInto, "a setup that does not say says nothing")

				for _, want := range []tone.PlaysInto{
					tone.Pa, tone.Headphones, tone.AmpFront, tone.AmpReturn,
				} {
					got, err := tone.LoadSetup(strings.NewReader(
						"schema: Setup\ndevice:\n  model: HX Stomp\nplays_into: " +
							string(want) + "\n"))

					s.Require().NoError(err, string(want))
					s.Require().Equal(want, *got.PlaysInto)
				}

				_, err = tone.LoadSetup(strings.NewReader(
					"schema: Setup\ndevice:\n  model: HX Stomp\nplays_into: amp\n"))

				s.Require().ErrorIs(err, tone.ErrInvalid,
					"`amp` is two different paths, so the contract will not take it")
			},
		},
		{
			// Passing a Setup where the ask goes.
			//
			// The enum would refuse it anyway and say `schema` is not an
			// allowed value, which is true and unhelpful to somebody who
			// passed the wrong file.
			name: "the wrong document says so",
			then: func() {
				_, err := tone.Load(strings.NewReader("schema: Setup\n"))

				s.Require().ErrorIs(err, tone.ErrInvalid)
				s.Require().Contains(err.Error(), "says Setup, so this is not a ToneSpec")

				_, err = tone.LoadSetup(strings.NewReader("schema: ToneSpec\n"))
				s.Require().Contains(err.Error(), "says ToneSpec, so this is not a Setup")
			},
		},
		{
			// The reader itself failing.
			name: "a read failure is reported",
			then: func() {
				_, err := tone.LoadSetup(iotest{})

				s.Require().ErrorContains(err, "reading the Setup")
			},
		},
		{
			// The case the schema allows.
			//
			// JSON Schema calls 2000000000000000000000 an integer and Go's
			// int cannot hold it, so without the second check this returns a
			// document with the field silently zeroed and no error at all.
			// The contract caps a year at 2100, so the number has to arrive
			// somewhere uncapped: `slot` on an owned impulse response has a
			// minimum and no maximum.
			name: "a number too large for the types is refused",
			then: func() {
				_, err := tone.LoadSetup(strings.NewReader(`
schema: Setup
owns:
  - kind: ir
    name: Owned 4x10
    slot: 2000000000000000000000
`))

				s.Require().ErrorContains(err, "decoding the Setup")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestWrite covers Write, which renders a request.
//
// One method and one table, so a case is a row rather than a file.
func (s *LoadPublicTestSuite) TestWrite() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The round trip.
			name: "writes what it read",
			then: func() {
				// Two genres, because one is the case that hid the defect: the corpus tags
				// the same players punk and pop-punk, and a field holding one word dropped
				// whichever was written second.
				spec := tone.Spec{
					Schema: "ToneSpec",
					Id:     "punky",
					Ask:    &tone.Ask{Genre: []string{"punk", "pop-punk"}},
					Rig:    minimalRig(),
				}

				var buf bytes.Buffer
				s.Require().NoError(tone.Write(&buf, spec))

				back, err := tone.Load(&buf)
				s.Require().NoError(err)
				s.Require().Equal([]string{"punk", "pop-punk"}, back.Ask.Genre,
					"both of them, in the order they were written")
				s.Require().Equal("Ampeg SVT", back.Rig.Chain[0].Gear,
					"the gear travels with the ask now, in one document")
			},
		},
		{
			// The writer itself failing.
			name: "a write failure is reported",
			then: func() {
				err := tone.Write(broken{}, tone.Spec{
					Schema: "ToneSpec",
					Id:     "rocky",
					Ask:    &tone.Ask{Genre: []string{"rock"}},
					Rig:    minimalRig(),
				})

				s.Require().ErrorContains(err, "writing the ToneSpec")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestWriteSetup covers WriteSetup, which renders what somebody has.
//
// One method and one table, so a case is a row rather than a file.
func (s *LoadPublicTestSuite) TestWriteSetup() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The other document's round trip.
			name: "writes a setup",
			then: func() {
				setup := tone.Setup{
					Schema: "Setup",
					Device: &tone.Device{Model: "HX Stomp"},
				}

				var buf bytes.Buffer
				s.Require().NoError(tone.WriteSetup(&buf, setup))

				back, err := tone.LoadSetup(&buf)
				s.Require().NoError(err)
				s.Require().Equal("HX Stomp", back.Device.Model)
			},
		},
		{
			// The check before the render.
			//
			// Writing one that does not meet its own contract would put a
			// file into the world that nothing else will accept.
			name: "an invalid document is not written",
			then: func() {
				var buf bytes.Buffer

				s.Require().ErrorIs(
					tone.Write(&buf, tone.Spec{}), tone.ErrInvalid)
				s.Require().ErrorIs(
					tone.WriteSetup(&buf, tone.Setup{}), tone.ErrInvalid)
				s.Require().Empty(buf.String())
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// minimalRig is the least a document may say about gear, which the contract
// requires: what it is played on, and one block.
func minimalRig() tone.Rig {
	return tone.Rig{
		Instrument: tone.InstrumentBass,
		Chain: []tone.ChainEntry{
			{Role: tone.RoleAmp, Gear: "Ampeg SVT"},
		},
	}
}

// iotest is a reader that always fails.
type iotest struct{}

func (iotest) Read(
	[]byte,
) (int, error) {
	return 0, errors.New("no")
}

// broken is a writer that always fails.
type broken struct{}

func (broken) Write(
	[]byte,
) (int, error) {
	return 0, errors.New("no")
}

func TestLoadPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LoadPublicTestSuite))
}
