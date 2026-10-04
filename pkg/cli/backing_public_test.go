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
package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/cli"
	sdk "github.com/retr0h/toneharness/pkg/sdk"
)

// BackingPublicTestSuite covers the table that holds a rig's records to its
// era.
type BackingPublicTestSuite struct {
	suite.Suite
}

// render draws a set of rigs and hands back what was written.
func (s *BackingPublicTestSuite) render(
	all []sdk.Backing,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.Backing(&buf, all))

	return buf.String()
}

// TestBacking covers the table saying which records back each rig.
//
// One method and one table, so a case is a row rather than a file.
func (s *BackingPublicTestSuite) TestBacking() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// The case nobody has to act on.
			name: "records that match the era",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID: "geddy-lee", Era: "Fly by Night through Hemispheres",
					From: 1975, To: 1978,
					Records: []sdk.Record{
						{Track: "anthem", Year: 1975},
						{Track: "la-villa-strangiato", Year: 1978},
					},
				}})

				s.Require().Contains(got, "geddy-lee")
				s.Require().Contains(got, "1975–1978")
				s.Require().Contains(got, "records match the era")
				s.Require().Contains(got, "every record in era")
			},
		},
		{
			// The finding this table exists for.
			name: "every record from another era",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID: "flea", Era: "2012 touring", From: 2012, To: 2012,
					Records: []sdk.Record{
						{Track: "aeroplane", Year: 1995, Outside: true},
						{Track: "suck-my-kiss", Year: 1991, Outside: true},
					},
				}})

				s.Require().Contains(got, "every record is from another era")
				s.Require().Contains(got, "2012")
				s.Require().Contains(got, "1 with something to answer for")
			},
		},
		{
			// The partial case, which is the one
			// somebody can fix by swapping a record.
			name: "some records from another era",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID: "tim-commerford", From: 1992, To: 1992,
					Records: []sdk.Record{
						{Track: "bombtrack", Year: 1992},
						{Track: "sleep-now-in-the-fire", Year: 1999, Outside: true},
					},
				}})

				s.Require().Contains(got, "1 of 2 from another era")
				s.Require().Contains(got, "1992")
			},
		},
		{
			// The row for a directory nobody claims,
			// which otherwise reads as a rig nobody has measured.
			name: "records no rig is named for",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID: "mccartney", NoRig: true,
					Records: []sdk.Record{{Track: "silly-love-songs", Year: 1976}},
				}})

				s.Require().Contains(got, "no rig is named for this directory")
				s.Require().Contains(got, "1 with something to answer for")
			},
		},
		{
			// What cannot be checked, which is worth seeing.
			name: "a rig with no era",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID:      "somebody",
					Records: []sdk.Record{{Track: "one", Year: 1999}},
				}})

				s.Require().Contains(got, "says none")
				s.Require().Contains(got, "no era to hold them to")
			},
		},
		{
			// The ordinary case: gear evidence long
			// before anybody owns the records.
			name: "a rig nobody has measured",
			then: func() {
				got := s.render([]sdk.Backing{{ID: "somebody", From: 1994, To: 1994}})

				s.Require().Contains(got, "none measured")
				s.Require().Contains(got, "nothing measured for it")
			},
		},
		{
			// A rig that applied for a single year.
			name: "one year reads as one year",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID: "pino-palladino", From: 2000, To: 2000,
					Records: []sdk.Record{{Track: "chicken-grease", Year: 2000}},
				}})

				s.Require().Contains(got, "2000")
				s.Require().NotContains(got, "2000–2000")
			},
		},
		{
			// The empty table.
			name: "no rigs at all",
			then: func() {
				got := s.render(nil)

				s.Require().Contains(got, "no rigs to read")
			},
		},
		{
			// Every answer the room column can give.
			//
			// The era check and this one are independent: a rig passes the first by
			// measuring records made when its gear was, and fails the second by measuring
			// records made through something else. All nine rigs that ship pass the era
			// check and five of them read `direct` here.
			name: "the room is reported",
			then: func() {
				for _, tt := range []struct {
					name string
					in   sdk.Backing
					want string
				}{
					{
						name: "a signal that went to the desk",
						in:   sdk.Backing{ID: "geddy-lee", Direct: 2, Captured: 2},
						want: "direct",
					},
					{
						name: "a DI and a microphone at once",
						in:   sdk.Backing{ID: "jaco-pastorius", Both: 2, Captured: 2},
						want: "direct and miked",
					},
					{
						name: "the one that was only ever miked",
						in:   sdk.Backing{ID: "pino-palladino", Captured: 2},
						want: "miked",
					},
					{
						name: "gear only a tour documents",
						in:   sdk.Backing{ID: "tim-commerford", Stage: 2},
						want: "gear from a stage",
					},
					{
						name: "both faults at once",
						in:   sdk.Backing{ID: "paul-mccartney", Direct: 2, Captured: 2, Stage: 2},
						want: "direct, gear from a stage",
					},
					{
						// Silence is not an answer. A rig nobody has asked would read as
						// miked if the count of answers were not kept separately.
						name: "nobody established it",
						in:   sdk.Backing{ID: "bootsy-collins"},
						want: "not established",
					},
				} {
					s.Run(tt.name, func() {
						s.Require().Contains(s.render([]sdk.Backing{tt.in}), tt.want)
					})
				}
			},
		},
		{
			// The orphan case.
			//
			// A directory no rig answers to has no chain to read, so the column has
			// nothing to say and must not guess.
			name: "records no rig is named for say nothing about the room",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID: "nobody", NoRig: true,
					Records: []sdk.Record{{Track: "a-record", Year: 1990}},
				}})

				s.Require().Contains(got, "no rig is named for this directory")
			},
		},
		{
			// The join failing loudly.
			//
			// `played.records` attributes a figure to the instrument that made it. A name
			// no manifest carries attributes it to nothing, and silence there looks
			// identical to an instrument nobody has got to yet.
			name: "an instrument naming a record nobody has",
			then: func() {
				got := s.render([]sdk.Backing{{
					ID: "les-claypool", Era: "Seas of Cheese through Punchbowl",
					From: 1991, To: 1995,
					Records:  []sdk.Record{{Track: "tommy-the-cat", Year: 1991}},
					Misnamed: []string{"jerry-was-a-race-car-drivr"},
				}})

				s.Require().Contains(got, "no record called jerry-was-a-race-car-drivr")
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

func TestBackingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BackingPublicTestSuite))
}
