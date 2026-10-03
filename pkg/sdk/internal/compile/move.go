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

package compile

import (
	"math"
	"slices"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
	"github.com/retr0h/toneharness/pkg/sdk/plan"
)

// Moved is what a word did to a parameter.
type Moved struct {
	// Term is the word that moved it.
	Term string
	// Param is the control it moved. Empty when nothing in the chain answers
	// to this term yet, which is most of the vocabulary.
	Param string
	// From and To are where the parameter was and where it went.
	From float64
	To   float64
	// Against names the axis another term in the same rig also spoke for.
	// Empty unless two words answered one question.
	Against string
	// YieldedTo names the word somebody wrote that this one stood aside for.
	// Empty unless a genre's word met an ask's word on one axis.
	YieldedTo string
	// Because says why the chain could not answer this word, when the chain
	// is the reason. Empty when nothing acts on the word at all, which is a
	// different answer: one says this rig cannot hear it, the other says
	// nobody has taught the project to listen.
	Because string
	// Already says how the chain answers this word without a knob being
	// turned. A rig asking for no room, in a chain holding no reverb, asked
	// for something it already has.
	Already string
	// Weight is how much of a step the word was worth, where 1 is the whole
	// step. Less than that when the term's own evidence measured the gap
	// that earned it and the gap is small.
	Weight float64
}

// Acted says whether the term moved anything.
func (m Moved) Acted() bool { return m.Param != "" }

// Contested says whether another term spoke for the same axis.
func (m Moved) Contested() bool { return m.Against != "" }

// Yielded says whether this word stood aside for one somebody wrote.
func (m Moved) Yielded() bool { return m.YieldedTo != "" }

// Unanswered says whether the chain, rather than this project, is why the
// word moved nothing.
func (m Moved) Unanswered() bool { return m.Because != "" }

// Holds says whether the chain already answers the word as built.
func (m Moved) Holds() bool { return m.Already != "" }

// turn is one term's effect: which kind of block, which parameter, and how
// many steps along it.
//
// A step is signed and scaled rather than a direction, because two shapes of
// axis need different arithmetic. `mids` is a pair, one step either side.
// `drive` is a scale — clean, minimal-drive, grit-on-attack, saturated are
// four positions on one line — so each term carries its own multiple.
type turn struct {
	// axis is the question this move answers, as words.json names it.
	//
	// Stated rather than inferred, because a word may answer more than one and
	// the two tables are then checkable against each other: every axis a word
	// declares has a move, and every move names an axis that word declares.
	axis     string
	category catalog.Category
	param    string
	steps    float64
	// also names where else the same question can be answered, when the
	// block that usually answers it has no such control.
	//
	// An Ampeg B-15NF has no Mid and neither does an Acoustic 360, so
	// `mid-forward` reaches nothing on two of the rigs here, and both earned
	// that word from their own records. An equaliser in the chain has the
	// same band under a different name, which is the one place a word can go
	// looking without inventing anything: the block is already there because
	// somebody put it there.
	also []answers
	// absenceMeans is what it means for the chain to hold no block of this
	// kind, where that already answers the term. Empty where it does not:
	// a chain with no compressor is not a chain with a soft attack, but a
	// chain with no reverb really does have no room on it.
	absenceMeans string
}

// answers is one place a question can be put, as the kind of block and the
// control it calls that band.
type answers struct {
	category catalog.Category
	param    string
}

// turns is what a term does, for the terms that do anything.
//
// Six axes of the ten, because six have a control and a direction that is not
// a guess. Mid, Treble and Drive need no explaining. Sag does, and the Pilot's
// Guide explains it: lower values offer tighter responsiveness, higher values
// more touch dynamics and sustain.
//
// The other four axes — decay, string-noise, pickup, movement — describe the
// player and the instrument, and no amplifier, reverb or compressor has a
// control for them. A term from one of those is recorded and moves nothing,
// which a build says out loud.
var turns = map[string][]turn{
	"mid-forward": {{
		axis: "mids", category: catalog.CategoryAmp, param: "Mid", steps: 1,
		also: []answers{{catalog.CategoryEQ, "MidGain"}},
	}},
	"scooped": {{
		axis: "mids", category: catalog.CategoryAmp, param: "Mid", steps: -1,
		also: []answers{{catalog.CategoryEQ, "MidGain"}},
	}},

	"dark": {{
		axis: "highs", category: catalog.CategoryAmp, param: "Treble", steps: -1,
		also: []answers{{catalog.CategoryEQ, "HighGain"}},
	}},
	"bright": {{
		axis: "highs", category: catalog.CategoryAmp, param: "Treble", steps: 1,
		also: []answers{{catalog.CategoryEQ, "HighGain"}},
	}},
	"glassy": {{
		axis: "highs", category: catalog.CategoryAmp, param: "Treble", steps: 1,
		also: []answers{{catalog.CategoryEQ, "HighGain"}},
	}},

	"clean":          {{axis: "drive", category: catalog.CategoryAmp, param: "Drive", steps: -1}},
	"minimal-drive":  {{axis: "drive", category: catalog.CategoryAmp, param: "Drive", steps: -0.5}},
	"grit-on-attack": {{axis: "drive", category: catalog.CategoryAmp, param: "Drive", steps: 0.5}},
	"saturated":      {{axis: "drive", category: catalog.CategoryAmp, param: "Drive", steps: 1}},

	"tight-low-end": {{axis: "low-end", category: catalog.CategoryAmp, param: "Sag", steps: -1}},
	"loose-low-end": {{axis: "low-end", category: catalog.CategoryAmp, param: "Sag", steps: 1}},

	// The first word here that answers two questions, and the reason the
	// vocabulary had to be able to hold one. Players say "punchy" constantly
	// and it is not a synonym for either half: it is a bottom that stops with
	// the note and a front you hear first, together.
	//
	// Composed from the two terms it subsumes rather than given moves of its
	// own. Nothing has measured what punchy is, so inventing a third pair of
	// numbers would be the guessing this project exists to remove, while
	// saying it means tight-low-end and percussive at once is what the word
	// already means.
	"punchy": {
		{axis: "low-end", category: catalog.CategoryAmp, param: "Sag", steps: -1},
		{axis: "attack", category: catalog.CategoryComp, param: "Attack", steps: 1},
	},

	// How much of the room is on the part, which is the reverb's own
	// question and nothing to do with the amplifier.
	"dry": {{
		axis: "space", category: catalog.CategoryReverb, param: "Mix", steps: -1,
		absenceMeans: "this chain has no reverb, so it is already dry",
	}},
	"roomy": {{axis: "space", category: catalog.CategoryReverb, param: "Mix", steps: 1}},

	// A compressor's attack decides how much of the front of a note gets
	// past it. Slow, and the pick is a sound of its own; fast, and notes
	// arrive rather than start. A scale, like drive.
	"soft-attack": {
		{axis: "attack", category: catalog.CategoryComp, param: "Attack", steps: -1},
	},
	"audible-pick-attack": {
		{axis: "attack", category: catalog.CategoryComp, param: "Attack", steps: 0.5},
	},
	"percussive": {
		{axis: "attack", category: catalog.CategoryComp, param: "Attack", steps: 1},
	},
}

// fallbackStep is how far a term moves a parameter the corpus cannot measure.
//
// A tenth of the stated range. Enough to hear, small enough that being wrong
// about it costs little, and used only where too few presets hold this model
// for a spread to mean anything.
const fallbackStep = 0.1

// maxStep is the furthest one term moves a parameter, as a share of its range.
//
// A quarter. The corpus spread is a good step where players mostly agree, and
// for Mid, Treble and Sag three steps in four are a quarter of the range or
// less. Where they disagree wildly the spread is no step at all: Sag on the
// Cali 400 and on the Ampeg SVT's normal channel spreads across half its
// range, so one word would put it on the rail. A word is one opinion, and one
// opinion should not decide the whole of a control.
const maxStep = 0.25

// move applies an ask's words to whichever blocks answer for them.
//
// Each axis names the kind of block it speaks to, because a word is about a
// part of the sound and not about a box: "roomy" is the reverb's question and
// "mid-forward" is the amplifier's. A chain holding neither still reports the
// words, because a term that moves nothing is still something the rig said and
// reporting only the ones that worked would read as if the rest had.
func move(
	blocks []catalog.Block,
	built plan.Plan,
	terms []heard,
	stats *corpus.Stats,
) []Moved {
	out := make([]Moved, 0, len(terms))
	contested := contested(terms)

	for _, h := range terms {
		term := h.term
		// Two words from one axis are two answers to one question. Neither
		// is applied, because applying both lands back where it started and
		// reads as though the rig said nothing.
		if axis, against := contestedAxis(term, contested); against {
			out = append(out, Moved{Term: term, Against: axis})

			continue
		}

		// Outranked rather than contradicted. A word a genre earned moves
		// nothing on an axis somebody already spoke for, and says whose word
		// took it, because "neither moved" would be a lie about both.
		if to, aside := yields(h, terms); aside {
			out = append(out, Moved{Term: term, YieldedTo: to})

			continue
		}

		moves, ok := turns[term]
		if !ok {
			// A word naming a block rather than a setting is answered by
			// that block being in the chain, and demand has already put one
			// there if the rig had none. So by here it is present, or this
			// device has nothing of the kind.
			if want, names := needs[term]; names {
				out = append(out, satisfied(blocks, term, want))

				continue
			}

			out = append(out, Moved{Term: term})

			continue
		}

		// One move per axis the word answers, each reported on its own. A
		// compound word that reaches one of its controls and not the other has
		// half an answer, and saying so is the whole point of reporting.
		for _, t := range moves {
			out = append(out, moveOne(blocks, built, h, t, stats))
		}
	}

	return out
}

// satisfied reports on a word whose whole meaning is that a block is there.
//
// Naming the block rather than saying "yes": which filter got seated is the
// part somebody reading the preset can disagree with, and "envelope-swept:
// satisfied" invites nobody to check.
func satisfied(
	blocks []catalog.Block,
	term string,
	want catalog.Category,
) Moved {
	if at := indexOf(blocks, want); at >= 0 {
		return Moved{
			Term:    term,
			Already: "the " + blocks[at].Name + " is what this asks for",
		}
	}

	return Moved{
		Term:    term,
		Because: "this device has no " + string(want),
	}
}

// answered finds the block that will take this word, and what that block
// calls the control.
//
// The kind that usually answers first, then wherever else the same question
// can be put. An equaliser is second rather than first because a rig naming
// an amplifier and an equaliser means the amplifier to be the voice.
func answered(
	blocks []catalog.Block,
	t turn,
) (int, string, bool) {
	places := append([]answers{{t.category, t.param}}, t.also...)

	for _, place := range places {
		for i, b := range blocks {
			if b.Category != place.category {
				continue
			}

			if _, held := b.Params[place.param]; held {
				return i, place.param, true
			}
		}
	}

	return 0, "", false
}

// indexOf finds the first block of a kind, or reports that there is none.
//
// The first, because a chain may hold two reverbs and a word is one opinion:
// spreading it over both would be two opinions nobody expressed.
func indexOf(
	blocks []catalog.Block,
	want catalog.Category,
) int {
	for i, b := range blocks {
		if b.Category == want {
			return i
		}
	}

	return -1
}

// apply turns one knob, or reports why it could not.
func apply(
	b catalog.Block,
	params plan.Params,
	h heard,
	t turn,
	key string,
	stats *corpus.Stats,
) Moved {
	term := h.term

	// The search established the control is there; what it cannot establish
	// is that the value is a number. A switch by that name is not a knob.
	p := b.Params[key]

	from, ok := params[key].Float()
	if !ok {
		return Moved{Term: term, Because: "the " + b.Name + " has no " + key}
	}

	to := clamp(from+t.steps*h.weight*step(b, key, p, stats), p.Min, p.Max)

	params[key] = catalog.Float(to)

	return Moved{Term: term, Param: key, From: from, To: to, Weight: h.weight}
}

// step is how far one term moves this parameter.
//
// The corpus spread where there is one: a parameter every player sets the same
// way is one nobody has an opinion about, and a term should barely move it. A
// parameter players disagree about is one where an opinion is worth having.
// Line 6's range would say the same thing about both. No step is wider than
// maxStep of the range, however much players disagree.
func step(
	b catalog.Block,
	key string,
	p catalog.Param,
	stats *corpus.Stats,
) float64 {
	span := p.Max - p.Min

	if stats != nil {
		if d, ok := stats.Param(b.ID, key); ok && d.Spread() > 0 {
			return math.Min(d.Spread(), span*maxStep)
		}
	}

	return span * fallbackStep
}

// clamp keeps a value inside what the device accepts.
func clamp(
	v, lo, hi float64,
) float64 {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

// termsOf reads the words an ask describes a sound with, in the order written.
//
// The order is kept because two words on one axis cancel and the report names
// them as the ask wrote them, which is no help if this reordered them first.
func termsOf(
	words []Word,
) []heard {
	if len(words) == 0 {
		return nil
	}

	out := make([]heard, 0, len(words))
	for _, word := range words {
		out = append(out, heard{
			term:    word.Term,
			weight:  weightOf(word),
			derived: word.Derived,
		})
	}

	return out
}

// heard is one word a rig used, and how much of a step it is worth.
type heard struct {
	// term is the word.
	term string
	// weight is how far along the term's own direction to move, where 1 is
	// the whole step. A word nobody measured is worth the whole step,
	// because there is nothing to say it should be worth less.
	weight float64
	// derived says a population earned the word rather than somebody writing
	// it, which decides who yields when two words answer one axis.
	derived bool
}

// measures names the figure each axis is earned from, for the axes a
// measurement can speak to.
//
// The same three [audio.Derive] compares on: the mid band, the centroid and
// how much energy sits above the fundamental. The rest of the vocabulary
// describes a player or an instrument rather than a band of the spectrum, so
// there is no figure to weigh it with.
var measures = map[string]string{
	"mids":  "mid",
	"highs": "centroid",
	"drive": "harmonics",

	// Added once the sweeps measured them. Each is the figure the corpus
	// already reports for that question, so none of it is a new measurement:
	// a low end is the low band's share, a decay is how long a note takes to
	// die away, and an attack is how sharply one starts.
	//
	// The four axes still missing have no figure to map to rather than a
	// figure nobody has taken. Nothing here measures how much room is on a
	// part, how loud the strings are under a hand, which pickup was used, or
	// whether a filter is moving.
	"low-end": "low",
	"decay":   "decay",
	"attack":  "transient",
}

// weightOf reads how far a word's own measurement sits from everybody else's.
//
// The gap as a share of what the others read: a player reading half the
// harmonics of the rest is worth half a step, and one reading none of them is
// worth the whole step. Beyond that it stops counting, because a word is one
// opinion and the cap on a step is what keeps one opinion off the rail.
//
// A word with no measurement, or one measuring something no control answers
// to, is worth the whole step. That is what every rig did before any of this
// was measured, and it stays the answer where nobody has measured anything.
func weightOf(
	w Word,
) float64 {
	axes, ok := axesOf(w.Term)
	if !ok {
		return 1
	}

	// The smallest measured displacement across the axes the word answers.
	//
	// Conservative on purpose, and it is what keeps a compound word from
	// double-dipping. "Punchy" is a tight low end and a hard attack, and a
	// recording may sit far from everybody else's low end while sitting in the
	// middle of everybody else's attack. Taking the larger would let the
	// better-supported half carry a claim the other half does not make.
	//
	// An axis nothing measured is skipped rather than counted as a full step.
	// That is the difference from a single-axis word, where a full step is the
	// best guess available: a compound word with evidence for one of its axes
	// has stated where its support is, and spending a whole step on the other
	// invents the measurement it did not make.
	weight := math.Inf(1)

	for _, axis := range axes {
		key, measured := measures[axis]
		if !measured {
			continue
		}

		if got, found := displaced(w, key); found {
			weight = math.Min(weight, got)
		}
	}

	// Nothing measured any axis it answers, which is what every rig did before
	// any of this was measured and stays the answer where nobody has measured
	// anything.
	if math.IsInf(weight, 1) {
		return 1
	}

	return weight
}

// displaced is how far a word's own measurement sits from everybody else's on
// one figure, as a share of what the others read.
func displaced(
	w Word,
	key string,
) (float64, bool) {
	for _, e := range w.Evidence {
		if e.Measured == nil || e.Against == nil {
			continue
		}

		mine, held := (*e.Measured)[key]
		theirs, also := (*e.Against)[key]

		if !held || !also {
			continue
		}

		// Against the spread where the evidence carries one, because that is
		// what a distance means. Half the width of everybody else's middle half
		// is half a step; clearing the whole width is the whole step.
		//
		// The median is the fallback and was the only answer before a word
		// carried its spread. It reads a distance as a fraction of where the
		// others sit, which answers a different question: a player 10% above a
		// median got a tenth of a step whether the others were packed into a
		// hertz or spread over two hundred. Evidence written before the spread
		// existed still takes that path rather than being refused.
		if e.Spread != nil {
			if width, wide := (*e.Spread)[key]; wide && width > 0 {
				return math.Min(math.Abs(mine-theirs)/width, 1), true
			}
		}

		if theirs == 0 {
			continue
		}

		return math.Min(math.Abs(mine-theirs)/math.Abs(theirs), 1), true
	}

	return 0, false
}

// cancels says whether a word is one of two answers to the same question.
//
// Any of its axes being contested is enough. A compound word that contradicts
// another word about the low end is cancelled whole, because the halves are not
// separable: the word is what somebody said.
func cancels(
	term string,
	contested map[string]bool,
) bool {
	_, against := contestedAxis(term, contested)

	return against
}

// contestedAxis is the first axis of a word that something else answered too,
// for saying which question was asked twice.
func contestedAxis(
	term string,
	contested map[string]bool,
) (string, bool) {
	axes, ok := axesOf(term)
	if !ok {
		return "", false
	}

	for _, axis := range axes {
		if contested[axis] {
			return axis, true
		}
	}

	return "", false
}

// moveOne turns one knob for one of a word's axes, or says why it could not.
func moveOne(
	blocks []catalog.Block,
	built plan.Plan,
	h heard,
	t turn,
	stats *corpus.Stats,
) Moved {
	at, param, found := answered(blocks, t)
	if found {
		return apply(blocks[at], built.Blocks[at].Params, h, t, param, stats)
	}

	at = indexOf(blocks, t.category)
	if at < 0 {
		// A word can ask for what the chain already is. Mix at zero and no
		// reverb at all are the same signal, so a rig asking to stay dry got
		// what it asked for and nothing is missing.
		if t.absenceMeans != "" {
			return Moved{Term: h.term, Already: t.absenceMeans}
		}

		// Otherwise something would answer for this word and this chain holds
		// none of it, which is the rig's shape rather than a gap here.
		return Moved{
			Term:    h.term,
			Because: "this chain holds no " + string(t.category),
		}
	}

	// The block that usually answers is there and has no such control, and
	// nothing else in the chain has one either.
	return Moved{
		Term:    h.term,
		Because: "the " + blocks[at].Name + " has no " + t.param,
	}
}

// contested finds the axes an ask spoke for more than once.
//
// Only words of the same standing contest each other. Two words somebody wrote
// about the low end is a contradiction and neither moves, because this cannot
// know which half they meant. A word they wrote against one a genre earned is
// not a contradiction: they said it and a population did not, so theirs answers
// the axis and the derived one yields, which [yields] reports.
//
// Counting both together made the person lose their own word to a measurement
// of fifteen records nobody asked about, and said "neither moved" as though
// they had contradicted themselves.
func contested(
	terms []heard,
) map[string]bool {
	written := map[string]int{}
	derived := map[string]int{}

	for _, h := range terms {
		axes, ok := axesOf(h.term)
		if !ok {
			continue
		}

		// One count per axis the word answers. A compound word speaks for both
		// of its axes, so it contests either of them on its own.
		for _, axis := range axes {
			if h.derived {
				derived[axis]++

				continue
			}

			written[axis]++
		}
	}

	out := map[string]bool{}

	for axis, n := range written {
		if n > 1 {
			out[axis] = true
		}
	}

	// Derived words contest each other only where nothing written answered the
	// axis. Two genres disagreeing is still a contradiction, and still nothing
	// this can resolve.
	for axis, n := range derived {
		if n > 1 && written[axis] == 0 {
			out[axis] = true
		}
	}

	return out
}

// yields says whether a derived word stands aside for one somebody wrote.
//
// Separate from [contested] because it is a different answer and deserves to be
// reported as one. A contested word is a contradiction nobody can resolve; a
// word that yields was simply outranked, and the build is doing what the person
// asked rather than refusing to choose.
func yields(
	h heard,
	terms []heard,
) (string, bool) {
	if !h.derived {
		return "", false
	}

	axes, ok := axesOf(h.term)
	if !ok {
		return "", false
	}

	for _, axis := range axes {
		for _, other := range terms {
			if other.derived {
				continue
			}

			if also, found := axesOf(other.term); found && slices.Contains(also, axis) {
				return other.term, true
			}
		}
	}

	return "", false
}
