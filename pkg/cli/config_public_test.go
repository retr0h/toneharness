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
	"github.com/retr0h/toneharness/pkg/sdk/config"
)

// SetupPublicTestSuite covers reading a setup back.
//
// The slot map and the favourites are what somebody asks this for: they are
// the only record of which presets on the pedal this tool manages, so they
// are the answer to "which slots will a rebuild overwrite".
type SetupPublicTestSuite struct {
	suite.Suite
}

// held builds a setup out of the parts a case names.
func held(
	device string,
	instruments []string,
	slots [][3]string,
	favourites []string,
	loudness string,
) config.Config {
	out := config.Config{Schema: config.Name, Version: "4.2.0"}

	if device != "" {
		out.Device = &config.Device{Model: device}
	}

	if len(instruments) > 0 {
		gear := make([]config.Held, 0, len(instruments))
		for _, one := range instruments {
			gear = append(gear, config.Held{Gear: one})
		}

		out.Instruments = &gear
	}

	if len(slots) > 0 {
		placed := make([]config.Slotted, 0, len(slots))
		for _, pair := range slots {
			one := config.Slotted{At: pair[0], Rig: pair[1]}

			// A third entry is what this slot is built for, where it differs
			// from the file's own answer.
			if pair[2] != "" {
				into := config.PlaysInto(pair[2])
				one.PlaysInto = &into
			}

			placed = append(placed, one)
		}

		out.Slots = &placed
	}

	if len(favourites) > 0 {
		out.Favourites = &favourites
	}

	if loudness != "" {
		out.MatchLoudness = &loudness
	}

	return out
}

// TestSetup covers what each shape of setup says.
//
// One method and one table, so a case is a row rather than a file.
func (s *SetupPublicTestSuite) TestSetup() {
	for _, tt := range []struct {
		name string
		in   config.Config
		want []string
		// missing is what must not be said.
		missing []string
	}{
		{
			// The whole thing, which is what somebody who has filled one in
			// is looking at.
			name: "a setup with everything on it",
			in: held("HX Stomp", []string{"Fender Precision Bass"},
				[][3]string{{"11A", "punk", ""}, {"18A", "mike-dirnt", ""}},
				[]string{"11A", "18A"}, "punk"),
			want: []string{
				"HX Stomp", "Fender Precision Bass", "as loud as",
				"SLOTS", "11A", "punk", "18A", "mike-dirnt",
				"FAVOURITES", "where device fav lands",
			},
		},
		{
			// A setup with the two lines at the top and nothing else, which
			// is what setup init writes. Each absence says what it costs
			// rather than printing an empty heading.
			name: "a setup with nothing filled in",
			in:   held("", nil, nil, nil, ""),
			want: []string{
				"nothing about the gear",
				"slots fill has nothing to write",
				"device fav has nowhere to go",
			},
		},
		{
			// The one case where a cycle cannot tell it is the current
			// favourite, called out where somebody is reading the list.
			name: "a favourite the slot map says nothing about",
			in: held("", nil, [][3]string{{"11A", "punk", ""}},
				[]string{"11A", "24C"}, ""),
			want: []string{
				"11A", "punk", "24C", "a step cannot recognise it",
			},
		},
		{
			// Two slots holding one rig are the same name twice otherwise,
			// and the difference between them is the whole reason both are
			// there: the first has its cabinet, for headphones, and the
			// second does not, for a real one.
			name: "one rig in two slots, built for two rooms",
			in: held("", nil, [][3]string{
				{"13C", "punk", ""},
				{"13B", "punk", "amp-return"},
			}, nil, ""),
			want: []string{"13C", "13B", "punk", "into amp-return"},
		},
		{
			// Slots without favourites. The two are separate decisions and a
			// person may well make only the first.
			name: "slots placed and no favourites",
			in: held("", nil, [][3]string{{"11A", "punk", ""}},
				nil, ""),
			want:    []string{"11A", "punk", "device fav has nowhere to go"},
			missing: []string{"where device fav lands"},
		},
	} {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(cli.Setup(&out, tt.in))

			for _, want := range tt.want {
				s.Require().Contains(out.String(), want)
			}

			for _, missing := range tt.missing {
				s.Require().NotContains(out.String(), missing)
			}
		})
	}
}

// TestSetupReportsAWriterThatFails covers each seam the page is written at.
func (s *SetupPublicTestSuite) TestSetupReportsAWriterThatFails() {
	full := held("HX Stomp", []string{"Fender Precision Bass"},
		[][3]string{{"11A", "punk", ""}}, []string{"11A"}, "punk")
	bare := held("", nil, nil, nil, "")

	for _, tt := range []struct {
		name  string
		in    config.Config
		after int
	}{
		{name: "the blank line before the gear", in: full},
		{name: "the gear table", in: full, after: 1},
		{name: "the line saying there is no gear", in: bare},
		{name: "the slots heading", in: full, after: 3},
		{name: "the line saying there are no slots", in: bare, after: 1},
		{name: "the blank line before the slots", in: full, after: 4},
		{name: "the slots table", in: full, after: 5},
		{name: "the favourites heading", in: full, after: 6},
		{name: "the line saying there are none", in: bare, after: 3},
		{name: "the blank line before the favourites", in: full, after: 7},
		{name: "the favourites table", in: full, after: 8},
	} {
		s.Run(tt.name, func() {
			s.Require().Error(cli.Setup(&failAfter{n: tt.after}, tt.in))
		})
	}
}

func TestSetupPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SetupPublicTestSuite))
}
