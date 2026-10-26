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

package translate

import (
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/config"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// Exposed to this package's external tests. What a genre note says is a
// decision with four answers and only three are reachable through whichever
// records somebody has tagged, so the fourth is tested here directly.
var GenreNote = genreNote

// The same reason, for choosing gear. Three of these answer the cases where the
// catalog and the measured library disagree, or where a genre has nobody to be
// measured against, and neither state is reachable through the data that ships:
// every amplifier it measures is in the catalog, and all three measured genres
// have a population behind them. Reachable or not, they are the contract.
var (
	// Playable is whether a block is for the instrument in hand.
	Playable = playable
	// GroupedBy is the test for whether a block is for the instrument in hand,
	// or nil where its category is not grouped by instrument at all. Read here
	// rather than through a build because a build reaches one grouping and
	// there are two: amplifiers carry a subcategory and cabinets carry a mark.
	GroupedBy = groupedBy
	// Wanted narrows a ranking to the models that are the best of their gear.
	Wanted = wanted
	// DisplacedTo turns a genre into a target a block can be ranked against.
	DisplacedTo = displacedTo
	// Named is the gear a chosen block is written down as, which is never the
	// manufacturer's name for it where the catalog says what it emulates.
	Named = named
)

// Aim is what a request is aiming at. Aliased rather than exported outright,
// because nothing outside this package constructs one in earnest: a build works
// it out from the ask and hands it straight to the ranking.
type Aim = aim

// Aiming builds what a request is aiming at, so the evidence a chosen block
// carries can be read without a corpus standing behind it.
//
// Two of the four answers [aim.because] gives are states no shipped genre is
// in: one that earns no word, and a target carrying no figures at all. Both are
// the contract whether or not today's data reaches them, and the first is the
// ordinary answer for a genre sitting where everything else sits.
func Aiming(
	named string,
	centroid float64,
	records, players int,
	earned []string,
	read map[string]float64,
	caveat string,
) Aim {
	return aim{
		figures: measured.Figures{Centroid: centroid},
		named:   named,
		read:    read,
		records: records,
		players: players,
		earned:  earned,
		caveat:  caveat,
	}
}

// Because is the evidence for choosing one block, written onto the block.
func Because(
	a Aim,
	gear string,
	ranked int,
	sat float64,
	instrument string,
	kind catalog.Category,
	// alike says being nearest meant nothing, because the field was bunched
	// closer than the comparison can resolve. Task 267.
	alike bool,
) tone.Evidence {
	return a.because(gear, ranked, sat, instrument, kind, alike)
}

// Said3 is a list the way a sentence says one.
var Said3 = said3

// ModelsFor is every model of one kind a gear name reaches.
func ModelsFor(
	cat *catalog.Catalog,
	gear string,
	kind catalog.Category,
) map[string]bool {
	return modelsFor(cat, gear, kind)
}

// Commonest is the one piece of gear most of these name.
func Commonest(
	cat *catalog.Catalog,
	said []rig.ChainEntry,
	kind catalog.Category,
) (rig.ChainEntry, int, bool) {
	return commonest(cat, said, kind)
}

// StompedFor is the effects a genre gets, switched off and tuned.
func StompedFor(
	spec tone.Ask,
	setup config.Config,
	deps Deps,
	notes *Notes,
) []rig.ChainEntry {
	said := stompedFor(spec, setup, deps, notes)
	out := make([]rig.ChainEntry, 0, len(said))

	for _, p := range said {
		out = append(out, p.entry)
	}

	return out
}

// Tuned sets a block's controls from its sweep.
func Tuned(
	entry *rig.ChainEntry,
	model catalog.ModelID,
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) {
	tuned(entry, model, spec, deps, notes)
}

// DisplacedToward is where a genre aims, as a displacement from a baseline.
func DisplacedToward(
	got audio.Genre,
	baseline measured.Figures,
) measured.Figures {
	return displacedTo(got, baseline)
}

// Chose picks the cabinet's microphone.
func Chose(
	entry *rig.ChainEntry,
	spec tone.Ask,
	deps Deps,
	notes *Notes,
) {
	chose(entry, spec, deps, notes)
}

// PlayerAim is the figures a player's own records measure as.
func PlayerAim(
	spec tone.Ask,
	deps Deps,
) (map[string]float64, bool) {
	notes := Notes{}

	at, ok := playerTarget(spec, deps, &notes)

	return at.read, ok
}

// AsPlayed narrows a ranking to the blocks that leave the notes alone.
//
// Read here rather than through a build because a build reaches one of its
// three answers. Nothing shipped refuses the nearest block of a category it
// chooses, and no category a build chooses from is refused outright, but both
// are the contract.
var AsPlayed = asPlayed

// AsHeld narrows a ranking to the blocks a measured chain actually holds.
//
// Read here for the reason [AsPlayed] is, and for one more: the cases that
// matter most are the ones the shipped corpus does not produce. Every
// modulation block this device has swept is held by no chain at all, so the
// fall-through where nothing survives is reachable today, and a corpus with
// no chains for the instrument is the contract rather than a state anything
// ships in.
var AsHeld = asHeld

// Exposed to this package's external tests. Which kind of block answers a
// figure, and which kinds a target's records ask for, are both tables with
// more rows than any build reaches: four figures ever earn a word across the
// 43 rigs here and three kinds answer them, so the rest are the contract
// rather than today's data.
var (
	// Answers is the kind of block that answers one figure.
	Answers = answers
	// AskedFor is the kinds a target's records ask for, in switch order.
	AskedFor = askedFor
	// Stompedable is whether a kind is one a foot reaches for, which decides
	// how many switches a chain has already spoken for. The catalog owns the
	// answer since there were two copies of it and they disagreed about a
	// compressor; this alias is what the suite was already written against.
	Stompedable = catalog.Stompable
	// AimedAt is what a rig's own records measure as, and the words they
	// earn, found by the document's own identifier.
	AimedAt = aimedAt
	// Said adds a sentence to the measured evidence a block carries.
	Said = said
)

// Asked is one kind of block a target's records ask for. Aliased rather than
// exported outright, because nothing outside this package constructs one: a
// build reads the words and hands the kinds straight to the ranking.
type Asked = asked

// Kind is the category this one asks for.
func (a Asked) Kind() catalog.Category { return a.kind }

// Term is the word asking, and Margin how far past the line the records sit.
func (a Asked) Term() string    { return a.term }
func (a Asked) Margin() float64 { return a.margin }

// Past is the margin as a share of what everybody else measures, which is the
// only form two figures can be compared in.
func (a Asked) Past() float64 { return a.past }

// Named and Figures are what an aim is for: what it came from, and the target
// in the figures a block is measured in.
func (a Aim) Named() string             { return a.named }
func (a Aim) Figures() measured.Figures { return a.figures }

// Beats is whether the nearest candidate is nearer than the field by more than
// the comparison can resolve. Exposed because the state it guards against is
// the shipped equalisers and the states it allows are everything else, and a
// table covers both plus the ones no field is in.
var Beats = beats

// InstrumentFor is what a rig is played on, from the setup and then the ask.
// Exposed because the states it answers are four and a build reaches two.
var InstrumentFor = instrumentFor

// LikeTheirRig and LikeTheirGenre are the two evidence tiers above
// measurement, exposed for the cross-instrument refusal task 266 added: no
// shipped rig or genre is for a guitar, so a build cannot reach it.
var (
	LikeTheirRig   = likeTheirRig
	LikeTheirGenre = likeTheirGenre
)

// Exposed to this package's external tests: the pieces of a corpus offer.
//
// Each is reachable through [Pizzazz] for the rows today's data produces, and
// each covers more than those. A category nothing holds, a category every
// chain engages, a chain already at the chip's ceiling and a name whose models
// all change the notes are all states the 43 shipped rigs do not reach, and
// all four decide what a person is offered.
var (
	// CarriedButOff ranks a kind of block by how often players put one in and
	// leave it switched off.
	CarriedButOff = carriedButOff
	// Offers is every kind worth a switch, best first, rotated per subject.
	Offers = offers
	// Popular is a model of one kind that chains hold, rotated per subject.
	Popular = popular
	// RanksFor is the models of one kind worth offering, best first.
	RanksFor = ranksFor
	// Rotated is which of those candidates a subject takes.
	Rotated = rotated
	// SpareOn is how much of the chip a chain has not spent.
	SpareOn = spareOn
)

// Offer is one kind of block worth a switch, for a test that reads the order.
type Offer = offer

// Kind is which category an offer names.
func (o Offer) Kind() catalog.Category { return o.kind }

// Share is how often a chain holding one leaves it off.
func (o Offer) Share() float64 { return o.share }
