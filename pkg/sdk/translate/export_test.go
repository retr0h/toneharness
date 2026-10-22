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
) tone.Evidence {
	return a.because(gear, ranked, sat, instrument, kind)
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
	setup tone.Setup,
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
