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

package recipes_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
)

// ScaffoldPublicTestSuite covers starting one rig from another.
type ScaffoldPublicTestSuite struct {
	suite.Suite
}

// nextLine, lineSep and paraSep end a line for a YAML parser, the way a
// newline does. Written as runes so the source stays readable.
var (
	nextLine = string(rune(0x0085))
	lineSep  = string(rune(0x2028))
	paraSep  = string(rune(0x2029))
)

// subjectBlock is the parent ask's subject as most asks write it.
const subjectBlock = `subject:
  kind: artist
  name: Parent Player
  band: A Band
`

// nestedKinds is why the parent's ask is believed, written as asks marshalled
// by machine write them: the keys after the first in each entry sit at the
// indent a subject's own keys sit at.
//
// Two entries rather than one, so `kind` appears more than once below the
// subject and a copy cannot pass by rewriting whichever it finds first.
const nestedKinds = `evidence:
- url: https://example.test/b
  kind: cited
- url: https://example.test/c
  kind: heard

`

// corrections is a round of somebody listening, as an ask records one.
const corrections = `corrections:
  - ask: less clunky
    reason: the low end was in the way
    verdict: much better
    at: "2026-09-20"

`

// meteor is a rig read off a device. Its keys are in marshalled order, so
// `schema` comes late, and five lines beside the subject's name are
// `  name:` at the same indent.
var meteor = filepath.Join(
	"..", "..", "..", "..", "examples", "rigspec", "dir-angl-meteor.yaml")

// meteorAsk is an ask for the meteor rig, which the examples tree does not
// carry because a rig read off a device answered nobody's written request.
//
// Written here rather than left out, because renaming a copy is renaming its
// subject, the subject is the ask's, and a parent with no ask beside it would
// exercise none of that.
const meteorAsk = `# What the meteor rig is taken to be for.

schema: ToneSpec

subject:
  kind: artist
  name: DIR:ANGL Meteor

confidence: low
`

// parentRig is the pair a copy is made from, in the shapes a rig and the ask
// beside it come in.
type parentRig struct {
	// subject replaces the parent ask's subject block.
	subject string
	// extra is written into the ask, below the subject.
	extra string
	// lead is written above the subject.
	lead string
	// breaks is what ends a line, where that is not a newline.
	breaks string
}

// text writes the parent's rig out.
//
// The gear and the identifier, and nothing about who it is for. The contract
// refuses `subject`, `aliases`, `default`, `extends` and `confidence` here, so
// everything a copy has to rewrite is in the ask beside it.
func (p parentRig) text() string {
	return p.ended(`# A header describing the parent, which the copy does not inherit.
#
# More of it.

schema: RigSpec
version: 2
id: parent

instrument: bass

chain:
  - role: amp
    gear: Ampeg SVT
    evidence:
      - kind: cited
        url: https://example.test/a
    confidence: high
`)
}

// ask writes the parent's ask out: who the rig is for, the names it answers
// to, and how far to trust the whole thing.
func (p parentRig) ask() string {
	subject := subjectBlock
	if p.subject != "" {
		subject = p.subject
	}

	return p.ended(`# A header describing the parent's ask, which the copy does not inherit.
#
# More of it.

schema: ToneSpec
aliases: [other-name]
default: true

` + p.lead + subject + `
` + p.extra + `confidence: medium
`)
}

// ended writes one document with the line endings a row asked for.
func (p parentRig) ended(
	body string,
) string {
	if p.breaks == "" {
		return body
	}

	return strings.ReplaceAll(body, "\n", p.breaks)
}

// parent writes a rig for a copy to be made from, and the ask beside it.
func (s *ScaffoldPublicTestSuite) parent(
	dir string,
	rig parentRig,
) {
	artists := filepath.Join(dir, "artists")
	s.Require().NoError(os.MkdirAll(artists, 0o750))
	s.Require().NoError(os.WriteFile(
		filepath.Join(artists, "parent.yaml"), []byte(rig.text()), 0o600))
	s.Require().NoError(os.WriteFile(
		filepath.Join(artists, "parent.tone.yaml"), []byte(rig.ask()), 0o600))
}

// example writes the meteor rig beside the parent, with an ask of its own.
func (s *ScaffoldPublicTestSuite) example(
	dir string,
) {
	raw, err := os.ReadFile(meteor)
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "artists", "dir-angl-meteor.yaml"), raw, 0o600))
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "artists", "dir-angl-meteor.tone.yaml"),
		[]byte(meteorAsk), 0o600))
}

// read is one of the files a copy was written to.
func (s *ScaffoldPublicTestSuite) read(
	dir, name string,
) string {
	raw, err := os.ReadFile(filepath.Join(dir, "artists", name))
	s.Require().NoError(err)

	return string(raw)
}

// copyOf makes the copy a row asks for, in a directory of its own, and
// returns the directory it went to.
func (s *ScaffoldPublicTestSuite) copyOf(
	rig parentRig,
	withExample bool,
	opts recipes.NewOptions,
) (string, error) {
	dir := s.T().TempDir()
	s.parent(dir, rig)

	if withExample {
		s.example(dir)
	}

	opts.Dir = dir
	opts.ID = "copy"

	_, err := recipes.New(context.Background(), opts)

	return dir, err
}

// TestACopyOfAnAskLessParentStillRecordsWhatItCameFrom covers the one case
// where the link back had nowhere to come from.
//
// A copy takes its ask from the parent's, and a parent with no ask gets a
// scaffolded one instead. That scaffolded one wrote no `extends`, so copying a
// rig read off a device, which is exactly the rig that has no ask, produced a
// copy `recipes show` would never list as a variant of anything. The link is a
// fact about the copy rather than about whether the parent happened to have an
// ask, so it is written either way.
func (s *ScaffoldPublicTestSuite) TestACopyOfAnAskLessParentStillRecordsWhatItCameFrom() {
	dir := s.T().TempDir()
	artists := filepath.Join(dir, "artists")
	s.Require().NoError(os.MkdirAll(artists, 0o750))

	// The rig alone, with no ask beside it, which is what `presets show` writes
	// and what nobody has written an ask for yet.
	raw, err := os.ReadFile(meteor)
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(
		filepath.Join(artists, "dir-angl-meteor.yaml"), raw, 0o600))

	_, err = recipes.New(context.Background(), recipes.NewOptions{
		Dir: dir, ID: "copy", From: "dir-angl-meteor", Name: "My Meteor",
	})
	s.Require().NoError(err)

	ask := s.read(dir, "copy.tone.yaml")
	s.Require().Contains(ask, "extends: dir-angl-meteor")
	s.Require().Contains(ask, "My Meteor")

	// And the link resolves, which is the thing that was actually broken: the
	// copy shows as a variant of what it came from.
	shown, err := recipes.Show(recipes.Source{Dir: dir}, "dir-angl-meteor")
	s.Require().NoError(err)
	s.Require().Len(shown.Variants, 1)
	s.Require().Equal("copy", shown.Variants[0].ID)
}

// TestNewFrom covers copying a rig as the start of another.
func (s *ScaffoldPublicTestSuite) TestNewFrom() {
	tests := []struct {
		name   string
		from   string
		kind   string
		who    string
		want   []string
		absent []string
		// wantAsk and absentAsk are the same for the ask beside the copy.
		// Which of the two a row states is which document the decision lives
		// in: the gear is the rig's, and the subject, the names it answers to
		// and the link back to what it was copied from are the ask's.
		wantAsk   []string
		absentAsk []string
		broken    bool
		// unreadable puts a directory where the walk expects a file.
		unreadable bool
		errText    string
		// instrument is what the report says the copy is played on.
		instrument string
		// named and amp are what the report says the copy is called and
		// which amplifier it holds.
		named string
		amp   string
		// cab and pedals are the rest of the gear the report names, and
		// noPedals says it names none.
		cab      string
		pedals   []string
		noPedals bool
		// base is the directory beneath, or empty for the rigs that ship.
		base string
		// rig is the shape of the rig copied from.
		rig parentRig
		// withExample writes the meteor rig beside the parent.
		withExample bool
		// compare makes the same copy without the name as well, and requires
		// the rigs to be the same file and the asks to differ in the
		// subject's name alone.
		compare bool
		// loads and loadsKind are what the copy says it is once read back.
		loads     string
		loadsKind string
	}{
		{
			// Nobody named the copy, so it keeps the parent's name, and the
			// chain comes across whole, so it holds the parent's amp.
			name:       "a copy of a rig in the same directory",
			from:       "parent",
			instrument: "bass",
			named:      "Parent Player",
			amp:        "Ampeg SVT",
			noPedals:   true,
			loads:      "Parent Player",
			want: []string{
				"id: copy",
				// The citation comes across, which is the point and the
				// hazard, so the header says to re-check it.
				"url: https://example.test/a",
				"gear: Ampeg SVT",
				"Every citation below came across with the copy",
			},
			absent: []string{"# A header describing the parent"},
			// `extends` is the ask's, because what one ask departs from is a
			// fact about what was wanted and not about the gear, and it is
			// written beside the key that says what the document is.
			wantAsk: []string{
				"schema: ToneSpec\nextends: parent",
				"all came across with the copy",
				"The corrections did not come across",
			},
			absentAsk: []string{
				// The parent's identity, which is not the copy's.
				"aliases:", "default: true",
				"# A header describing the parent",
			},
		},
		{
			// A verdict is somebody listening to one rig, and a copy is a
			// different one, so a copy has been heard by nobody.
			name:      "a copy of an ask somebody corrected",
			from:      "parent",
			rig:       parentRig{extra: corrections},
			wantAsk:   []string{"extends: parent", "confidence: medium"},
			absentAsk: []string{"corrections:", "less clunky", "much better"},
		},
		{
			name:      "a copy that is one song rather than a player",
			from:      "parent",
			kind:      "song",
			who:       "One Song",
			named:     "One Song",
			amp:       "Ampeg SVT",
			loads:     "One Song",
			loadsKind: "song",
			wantAsk:   []string{"kind: song", "name: One Song", "band: A Band"},
			// The band survives, because the song is still by them.
			absentAsk: []string{"kind: artist", "name: Parent Player"},
		},
		{
			// `kind` is the subject's, and an ask says why it is believed in
			// the same word at the same indent.
			name:      "a copy of an ask that says why it is believed",
			from:      "parent",
			kind:      "song",
			rig:       parentRig{extra: nestedKinds},
			loadsKind: "song",
			wantAsk:   []string{"kind: cited", "kind: heard", "  kind: song"},
		},
		{
			// Nothing matched a flow subject's kind, so asking for one was
			// taken and then ignored.
			name:      "a copy of a rig whose subject is a flow, made one song",
			from:      "parent",
			kind:      "song",
			rig:       parentRig{subject: "subject: { kind: artist, name: Parent Player }\n"},
			loadsKind: "song",
		},
		{
			// The cab here tracks what mike-dirnt.yaml actually says, so
			// this fails whenever research corrects that rig. It has done
			// once already, from an Ampeg 8x10 to the Acoustic 6x10 he names
			// for records. Update it rather than loosening it: what is being
			// tested is that a copy carries the parent's chain through, and
			// an assertion that accepts anything tests nothing.
			name:    "a copy of a rig that ships in the binary",
			from:    "mike-dirnt",
			cab:     "Acoustic 6x10",
			want:    []string{"gear: Ampeg SVT"},
			wantAsk: []string{"extends: mike-dirnt"},
		},
		{
			// An alias belongs to the parent, and `extends` is matched
			// against an id, so a copy made by alias has to record what the
			// alias resolved to or the link never fires.
			name:      "a copy made by one of the parent's aliases",
			from:      "dirnt",
			wantAsk:   []string{"extends: mike-dirnt"},
			absentAsk: []string{"extends: dirnt"},
		},
		{
			// A rig read off a device holds five other lines that are
			// `  name:` at the indent a subject's own keys sit at: a device's
			// name, a snapshot's, and so on. None of them is the subject, and
			// the subject is not in this file at all, so renaming a copy may
			// not touch a byte of it.
			name:        "a renamed copy of a rig read off a device",
			from:        "dir-angl-meteor",
			withExample: true,
			who:         "My Meteor",
			compare:     true,
			loads:       "My Meteor",
			named:       "My Meteor",
			amp:         "ENGL® Fireball 100",
			cab:         `Legacy 4x12" ENGL XXL V30`,
			// Everything else in the chain, in the order the signal meets it.
			pedals: []string{
				"Arbiter Cry Baby",
				"Ibanez® TS808 Tube Screamer®",
				"Mono, Stereo Line 6 Original",
				"Simple Delay",
				"Room",
			},
			// The rig's keys come before `schema`, and all of them are
			// the rig.
			want: []string{"id: copy", "HD2_ReverbRoom"},
			absent: []string{
				"id: dir-angl-meteor",
				"# A rig read off a device, not written by hand.",
			},
			wantAsk: []string{"schema: ToneSpec\nextends: dir-angl-meteor"},
		},
		{
			// Each of these is YAML syntax written bare after `name: `.
			name:  "a name that is a key and a value",
			from:  "parent",
			who:   "a: b",
			loads: "a: b",
		},
		{
			name:  "a name starting with a hash",
			from:  "parent",
			who:   "#hash",
			loads: "#hash",
		},
		{
			name:  "a name starting with a dash",
			from:  "parent",
			who:   "- dash",
			loads: "- dash",
		},
		{
			name:  "a name in quotes",
			from:  "parent",
			who:   `"quoted"`,
			loads: `"quoted"`,
		},
		{
			name:  "a name with a colon and a quote",
			from:  "parent",
			who:   `Bob's: "live"`,
			loads: `Bob's: "live"`,
		},
		{
			// The loader reads YAML 1.1, where a bare Yes is a boolean and a
			// rig's name has to be a string. Yes and No are band names.
			name:  "a name that is a word for yes",
			from:  "parent",
			who:   "Yes",
			loads: "Yes",
		},
		{
			name:  "a name that is a word for no",
			from:  "parent",
			who:   "No",
			loads: "No",
		},
		{
			name:  "a name that is a switch position",
			from:  "parent",
			who:   "on",
			loads: "on",
		},
		{
			name:  "a name that is a letter for no",
			from:  "parent",
			who:   "n",
			loads: "n",
		},
		{
			name:  "a name that is a letter for yes",
			from:  "parent",
			who:   "y",
			loads: "y",
		},
		{
			name:  "a name that is a word for true",
			from:  "parent",
			who:   "true",
			loads: "true",
		},
		{
			name:  "a name that is a number",
			from:  "parent",
			who:   "123",
			loads: "123",
		},
		{
			name:  "a name that is a word for nothing",
			from:  "parent",
			who:   "~",
			loads: "~",
		},
		{
			name: "a name with flow syntax, in a subject written as a flow",
			from: "parent",
			rig: parentRig{
				subject: "subject: { kind: artist, name: Parent Player, band: A Band }\n",
			},
			who:     "a, b}",
			loads:   "a, b}",
			wantAsk: []string{"band: A Band }"},
		},
		{
			// The comment is the parent's, about the line and not the name.
			name: "a name with a comment after it",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: Parent Player # who it is\n",
			},
			who:     "Someone Else",
			loads:   "Someone Else",
			wantAsk: []string{"# who it is"},
		},
		{
			name:      "a name the parent quoted",
			from:      "parent",
			rig:       parentRig{subject: "subject:\n  kind: artist\n  name: 'Parent '' Player'\n"},
			who:       `it's: "live"`,
			loads:     `it's: "live"`,
			absentAsk: []string{"Parent"},
		},
		{
			name: "a name the parent double quoted",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: \"Parent \\\" Player\"\n",
			},
			who:       "Two\nLines",
			loads:     "Two\nLines",
			absentAsk: []string{"Parent"},
		},
		{
			// A name spread over lines cannot be replaced where it stands,
			// so the document is written out again, and what the document
			// says other than the name survives that.
			name: "a name the parent wrote over several lines",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: >-\n    Parent\n    Player\n",
			},
			who:       "a: b",
			loads:     "a: b",
			absentAsk: []string{"Parent"},
			wantAsk:   []string{"confidence: medium"},
		},
		{
			name: "a quoted name the parent closed on a later line",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: \"Parent\n    Player\"\n",
			},
			who:       "a: b",
			loads:     "a: b",
			absentAsk: []string{"Parent"},
		},
		{
			// Replaced where it starts, the rest of the old name would be
			// left on the next line, so it is not replaced there.
			name:      "a bare name the parent carried onto a second line",
			from:      "parent",
			rig:       parentRig{subject: "subject:\n  kind: artist\n  name: Parent\n    Player\n"},
			who:       "a: b",
			loads:     "a: b",
			absentAsk: []string{"Parent"},
		},
		{
			// The anchor is the name's, and the band is written as whatever
			// the name is, so dropping the anchor leaves an ask that has lost
			// a value nothing can supply.
			name: "a name the parent anchored and pointed at",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: &who Parent Player\n  band: *who\n",
			},
			who:       "Someone Else",
			loads:     "Someone Else",
			absentAsk: []string{"Parent Player"},
		},
		{
			// A parser ends a line at any of these, so a file written with
			// them holds more lines than counting newlines finds.
			name:  "a rig whose lines end in carriage returns",
			from:  "parent",
			rig:   parentRig{breaks: "\r"},
			who:   "Someone Else",
			loads: "Someone Else",
			want:  []string{"gear: Ampeg SVT"},
		},
		{
			name:  "a rig whose lines end in next-line characters",
			from:  "parent",
			rig:   parentRig{breaks: nextLine},
			who:   "Someone Else",
			loads: "Someone Else",
		},
		{
			name:  "a rig whose lines end in line separators",
			from:  "parent",
			rig:   parentRig{breaks: lineSep},
			who:   "Someone Else",
			loads: "Someone Else",
		},
		{
			name:  "a rig whose lines end in paragraph separators",
			from:  "parent",
			rig:   parentRig{breaks: paraSep},
			who:   "Someone Else",
			loads: "Someone Else",
		},
		{
			// The separator ends the comment, so the parser counts two lines
			// where counting newlines finds one, and everything below it is
			// a line further down than it looks. Renaming to the name that
			// is already there is the case a comparison cannot catch.
			name:  "a comment above the name carrying a line separator",
			from:  "parent",
			rig:   parentRig{lead: "# a note" + lineSep + "# and more of it\n"},
			who:   "Parent Player",
			loads: "Parent Player",
			wantAsk: []string{
				"# a note" + lineSep + "# and more of it",
				"  name: Parent Player",
			},
		},
		{
			// The rigs beneath are read whole, the way a lookup reads them.
			name:    "a directory beneath that will not load",
			from:    "parent",
			base:    "testdata",
			errText: "broken.yaml",
		},
		{
			name:    "a copy of a rig nobody has",
			from:    "nobody-at-all",
			errText: "no such recipe",
		},
		{
			// A broken rig beside the parent is not the parent, so it does
			// not stop the copy.
			name:    "a rig that is not one, beside the parent",
			from:    "parent",
			broken:  true,
			want:    []string{"id: copy"},
			wantAsk: []string{"schema: ToneSpec\nextends: parent"},
		},
		{
			name:    "a rig that is not one, asked for",
			from:    "broken",
			broken:  true,
			errText: "not a valid rig",
		},
		{
			// A directory named like a rig: the glob matches it and reading
			// it cannot work.
			name:       "a directory wearing a rig's name, asked for",
			from:       "adir",
			unreadable: true,
			errText:    "opening",
		},
		{
			name:       "a directory wearing a rig's name, beside the parent",
			from:       "parent",
			unreadable: true,
			wantAsk:    []string{"extends: parent"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			s.parent(dir, tt.rig)

			if tt.withExample {
				s.example(dir)
			}

			if tt.broken {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "artists", "broken.yaml"),
					[]byte("schema: RigSpec\nid: broken\n"), 0o600))
			}

			if tt.unreadable {
				s.Require().NoError(os.MkdirAll(
					filepath.Join(dir, "artists", "adir.yaml"), 0o750))
			}

			opts := recipes.NewOptions{
				Base: tt.base,
				From: tt.from,
				Kind: tt.kind,
				Name: tt.who,
			}

			o := opts
			o.Dir = dir
			o.ID = "copy"

			got, err := recipes.New(context.Background(), o)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)

			if tt.instrument != "" {
				s.Require().Equal(tt.instrument, got.Instrument)
			}

			if tt.named != "" {
				s.Require().Equal(tt.named, got.Name)
			}

			if tt.amp != "" {
				s.Require().Equal(tt.amp, got.Amp)
			}

			if tt.cab != "" {
				s.Require().Equal(tt.cab, got.Cab)
			}

			if tt.pedals != nil {
				s.Require().Equal(tt.pedals, got.Pedals)
			}

			if tt.noPedals {
				s.Require().Empty(got.Pedals)
			}

			// A copy is a pair, so both halves are read and each row says
			// which of them it is making a claim about.
			body := s.read(dir, "copy.yaml")
			ask := s.read(dir, "copy.tone.yaml")

			for _, want := range tt.want {
				s.Require().Contains(body, want)
			}

			for _, absent := range tt.absent {
				s.Require().NotContains(body, absent)
			}

			for _, want := range tt.wantAsk {
				s.Require().Contains(ask, want)
			}

			for _, absent := range tt.absentAsk {
				s.Require().NotContains(ask, absent)
			}

			if tt.compare {
				unnamed := opts
				unnamed.Name = ""

				other, err := s.copyOf(tt.rig, tt.withExample, unnamed)
				s.Require().NoError(err)

				// The name is not in the rig at all, so renaming a copy has
				// to leave that file byte for byte as the parent wrote it.
				s.Require().Equal(s.read(other, "copy.yaml"), body,
					"the rig does not hold the name")
				s.requireOnlyNameDiffers(s.read(other, "copy.tone.yaml"), ask)
			}

			if tt.broken || tt.unreadable {
				return
			}

			// A copy is a whole rig with a whole ask beside it, so the pair
			// loads on its own. The subject is the ask's, so what a copy was
			// renamed to is checked there rather than on the gear.
			listed, err := recipes.List(recipes.Source{Dir: dir})
			s.Require().NoError(err)

			found := false

			for _, known := range listed.Rigs {
				if known.Rig.ID != "copy" {
					continue
				}

				found = true

				if tt.loads != "" || tt.loadsKind != "" {
					s.Require().NotNil(known.Ask, "a copy is written with its ask")
					s.Require().NotNil(known.Ask.Subject)
				}

				if tt.loads != "" {
					s.Require().Equal(tt.loads, known.Ask.Subject.Name)
				}

				if tt.loadsKind != "" {
					s.Require().Equal(tt.loadsKind, string(known.Ask.Subject.Kind))
				}
			}

			s.Require().True(found, "the copy loads")
		})
	}
}

// TestReplaceSubject covers replacing a subject's field in an ask's own text.
//
// New cannot reach these: it rewrites an ask that loaded, and a subject that
// loaded carries both fields, because the contract requires them. This is its
// own helper with its own contract, and what it does with a document that has
// no such field is part of that contract.
func (s *ScaffoldPublicTestSuite) TestReplaceSubject() {
	tests := []struct {
		name  string
		body  string
		key   string
		value string
		want  string
		// errText is what rewriting fails with, and nothing is returned.
		errText string
	}{
		{
			name:  "an ask with a name to replace",
			body:  "schema: ToneSpec\nsubject:\n  kind: artist\n  name: Parent Player\n",
			key:   "name",
			value: "Someone Else",
			want:  "  name: Someone Else\n",
		},
		{
			name:  "an ask with a kind to replace",
			body:  "schema: ToneSpec\nsubject:\n  kind: artist\n  name: Parent Player\n",
			key:   "kind",
			value: "song",
			want:  "  kind: song\n",
		},
		{
			name:    "a document that is not YAML",
			body:    "subject: [\n  unclosed\n",
			key:     "name",
			errText: "rewriting the copy's subject",
		},
		{
			name:    "an ask with no subject",
			body:    "schema: ToneSpec\nextends: parent\n",
			key:     "name",
			errText: "no such field to replace",
		},
		{
			name:    "a subject that holds no name",
			body:    "schema: ToneSpec\nsubject:\n  kind: artist\n",
			key:     "name",
			errText: "no such field to replace",
		},
		{
			name:    "a subject that is not a mapping",
			body:    "schema: ToneSpec\nsubject: Parent Player\n",
			key:     "name",
			errText: "no such field to replace",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := recipes.ReplaceSubject(tt.body, tt.key, tt.value)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)
				s.Require().Empty(got)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(got, tt.want)
		})
	}
}

// requireOnlyNameDiffers requires two copied asks to be the same file but for
// the subject's name.
//
// The asks rather than the rigs, because the name is the subject's and the
// subject is the ask's. What the rigs have to be is identical, which the caller
// requires outright.
func (s *ScaffoldPublicTestSuite) requireOnlyNameDiffers(
	plain, named string,
) {
	was := strings.Split(plain, "\n")
	now := strings.Split(named, "\n")
	s.Require().Len(now, len(was))

	var differ []int

	for i := range was {
		if was[i] != now[i] {
			differ = append(differ, i)
		}
	}

	s.Require().Len(differ, 1, "only the subject's name changes")
	s.Require().Equal("  name: DIR:ANGL Meteor", was[differ[0]])
	s.Require().Equal("  name: My Meteor", now[differ[0]])
}

func TestScaffoldPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ScaffoldPublicTestSuite))
}
