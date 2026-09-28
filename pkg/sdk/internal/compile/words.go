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
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

// UnknownTerm is a word somebody used to describe a sound that the shipped
// vocabulary does not carry.
//
// Reported rather than refused. A term the vocabulary carries moves controls;
// one it does not carry moves nothing, so an unfamiliar word costs nothing and
// stops nothing, and refusing a preset over it would make the format hostile
// to the person it exists for. Unknown gear is different: there is no model to
// write, so that is an error and stays one.
type UnknownTerm struct {
	// Term is what was asked for.
	Term string
	// Near are terms the vocabulary does carry that look close.
	Near []string
}

// vocabulary is the shipped term list, keyed by axis.
type vocabulary struct {
	Axes map[string]struct {
		About string            `json:"about"`
		Terms map[string]string `json:"terms"`
	} `json:"axes"`
}

// Words returns the terms the vocabulary knows, in order.
//
// Exported because the words are the point: a person writing a rig needs to
// see the list, and a test over the rigs this project ships needs to check
// against it.
func Words() []string {
	var v vocabulary

	// Embedded and written by this repository, so it parses.
	_ = json.Unmarshal(terms, &v)

	// Each word once, because one may answer more than one axis and this is
	// the list of words rather than the list of answers.
	held := map[string]bool{}

	out := []string(nil)

	for _, axis := range v.Axes {
		for term := range axis.Terms {
			if held[term] {
				continue
			}

			held[term] = true

			out = append(out, term)
		}
	}

	sort.Strings(out)

	return out
}

// CheckWords reports the words an ask uses that nothing defines.
//
// An empty result means every word asked for is one the vocabulary carries.
// The words arrive as words rather than as a document, because the ask is what
// holds them and nothing here needs the rest of it.
func CheckWords(
	words []string,
) []UnknownTerm {
	known := Words()
	out := []UnknownTerm(nil)

	for _, word := range words {
		if has(known, word) {
			continue
		}

		out = append(out, UnknownTerm{Term: word, Near: closest(known, word)})
	}

	return out
}

// ContestedAxis is one question an ask answered twice.
//
// Saying "mid-forward" has already said "not scooped". An ask naming both has
// named a direction and its opposite, and the two cancel: apply them in turn
// and the control lands where it started.
type ContestedAxis struct {
	// Axis is the question, named the way the vocabulary names it.
	Axis string
	// Terms are the words used on it, in the order they were written.
	Terms []string
}

// CheckAxes reports the axes an ask answered more than once.
//
// A build already says this out loud for any ask, and that is the right answer
// for somebody else's: a description is theirs to write. The asks this project
// ships are the examples everybody copies, so they are held to one term per
// axis by a test instead.
func CheckAxes(
	words []string,
) []ContestedAxis {
	seen := map[string][]string{}
	order := []string(nil)

	for _, word := range words {
		// Every axis the word answers, because a compound word contradicts on
		// each of them separately: `punchy` against `loose-low-end` is a real
		// contradiction about the low end and says nothing about attack.
		axes, ok := axesOf(word)
		if !ok {
			continue
		}

		for _, axis := range axes {
			if len(seen[axis]) == 0 {
				order = append(order, axis)
			}

			seen[axis] = append(seen[axis], word)
		}
	}

	out := []ContestedAxis(nil)

	for _, axis := range order {
		if len(seen[axis]) > 1 {
			out = append(out, ContestedAxis{Axis: axis, Terms: seen[axis]})
		}
	}

	return out
}

// closest returns the terms sharing the most words with what was written.
//
// Word overlap rather than the prefix match a gear name gets. What people
// write here is a phrase in some order, and "pick attack audible" should
// reach "audible-pick-attack" even though neither is a prefix or a substring
// of the other.
//
// The two are not one function because they are not one problem. near, in
// check.go, finds a name somebody typed part of. This finds a term inside a
// sentence somebody wrote. Word overlap cannot reach "Ampeg SVT" from
// "Ampeg", and a prefix cannot reach a phrase whose words are reordered.
func closest(
	known []string,
	term string,
) []string {
	want := words(term)
	if len(want) == 0 {
		return nil
	}

	best, hits := 0, make([]string, 0, len(known))

	for _, k := range known {
		shared := 0

		for w := range words(k) {
			if want[w] {
				shared++
			}
		}

		switch {
		case shared > best:
			// A better match replaces every worse one, reusing the room
			// already taken rather than starting a new slice each time.
			best, hits = shared, append(hits[:0], k)
		case shared == best && shared > 0:
			hits = append(hits, k)
		}
	}

	if best == 0 {
		return nil
	}

	return hits
}

// words splits a term into the words it is made of, however it was spelled.
func words(
	s string,
) map[string]bool {
	out := map[string]bool{}

	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		out[w] = true
	}

	return out
}

// has says whether the vocabulary carries a term.
func has(
	known []string,
	term string,
) bool {
	for _, k := range known {
		if k == term {
			return true
		}
	}

	return false
}

// axesOf says which axes a term answers.
//
// Usually one, and the axis structure exists to catch a contradiction: saying
// `mid-forward` has already said `not scooped`, and a rig claiming both has
// claimed nothing because applying both lands the knob where it started.
//
// More than one where a word is how players actually talk. "Punchy" is a tight
// low end and a hard attack in one word, and a vocabulary that could not hold it
// could not see it contradict `loose-low-end` either, because the word was not
// in the vocabulary at all.
//
// An axis is what makes a term mean something: saying "mid-forward" has
// already said "not scooped", and a rig claiming both has claimed nothing.
func axesOf(
	term string,
) ([]string, bool) {
	var v vocabulary

	// Embedded and written by this repository, so it parses.
	_ = json.Unmarshal(terms, &v)

	var out []string

	for axis, a := range v.Axes {
		if _, ok := a.Terms[term]; ok {
			out = append(out, axis)
		}
	}

	// Sorted, because ranging a map is not an order and a word answering two
	// axes would report them differently each run.
	sort.Strings(out)

	return out, len(out) > 0
}

// Axis is one question the vocabulary asks, with the answers it accepts.
type Axis struct {
	// Name is the question, as the vocabulary names it.
	Name string
	// About is what the question is about.
	About string
	// Words are the answers, in order.
	Words []Defined
}

// Defined is one word, what it means, and what it moves.
type Defined struct {
	// Term is the word itself, and Means what it describes.
	Term  string
	Means string
	// Param is the control it moves and Block the kind of block that carries
	// it, both empty for a word that moves nothing.
	//
	// Four of the ten axes describe the player and the instrument, and no
	// amplifier, reverb or compressor has a control for them. A word from one
	// of those is recorded and moves nothing, which a build says out loud.
	Param string
	Block string
	// Steps is how far it moves that control, as a share of the step the
	// corpus spread allows. Negative moves it down.
	Steps float64
}

// Vocabulary is every axis and every word, with what each one moves.
//
// The shape a reference page needs. Words() answers "is this a word", which is
// what a check wants; this answers "what are the words and what do they do",
// which is what somebody writing an ask wants.
func Vocabulary() []Axis {
	var v vocabulary

	// Embedded and written by this repository, so it parses.
	_ = json.Unmarshal(terms, &v)

	out := make([]Axis, 0, len(v.Axes))

	for name, axis := range v.Axes {
		words := make([]Defined, 0, len(axis.Terms))

		for term, means := range axis.Terms {
			one := Defined{Term: term, Means: means}

			// The move for this axis, because a word may answer more than one
			// and each axis shows the control that answers it rather than
			// whichever move happened to be written first.
			for _, t := range turns[term] {
				if t.axis != name {
					continue
				}

				one.Param, one.Block, one.Steps = t.param, string(t.category), t.steps
			}

			words = append(words, one)
		}

		sort.Slice(words, func(i, j int) bool { return words[i].Term < words[j].Term })
		out = append(out, Axis{Name: name, About: axis.About, Words: words})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out
}
