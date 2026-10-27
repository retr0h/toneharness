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

package measured

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// The two ways a reading can fail to be about the chain asking, separate
// because they want different sentences: one names the wrong instrument and
// the other has none to name.
var (
	// ErrOtherInstrument is a reading taken on an instrument that is not the
	// one the chain is for.
	ErrOtherInstrument = errors.New("measured on another instrument")
	// ErrNoInstrument is a reading that cannot say which instrument it was
	// taken on, which is every file written before 2026-09-29.
	ErrNoInstrument = errors.New("names no instrument")
)

// Describes says whether a reading taken on one instrument describes a chain
// built for another, and names the problem where it does not.
//
// Every figure in this package is a figure about one instrument. A bass
// guitar's centre of gravity sits an octave below a guitar's, so a guitar
// chain ranked against bass readings, or a control set from a bass sweep, is
// not a worse answer to the question asked: it is an answer to a different
// one. Nothing in the figures refuses it, either. The solve converges, reports
// its tolerances met, and is wrong about what it was asked, which is the
// failure this project exists to catch.
//
// Both [Library] and [Curves] carry the field and until task 290 nothing on
// any path that builds a preset read it, while all 215 committed readings say
// `bass`. That cost nothing only because no guitar record is measured, so a
// guitar ask reaches no target and no reading is ever consulted for one.
//
// An empty `wants` passes everything, because a caller with no chain in hand
// has to take a reading as measured: `measure slopes` reports across a tree
// rather than building for anybody.
//
// An empty `got` is refused rather than allowed. A reading that cannot say
// what was played through it cannot be shown to be about this chain, and
// "probably bass, everything here is" is the reasoning this guard exists to
// stop. `pkg/cli`'s slope report reaches the same two cases and names both,
// which is where the shape comes from.
func Describes(
	got, wants string,
) error {
	switch {
	case wants == "":
		return nil
	case got == "":
		return fmt.Errorf("%w, and this is for %s", ErrNoInstrument, wants)
	case !strings.EqualFold(got, wants):
		return fmt.Errorf("%w: taken on %s and this is for %s",
			ErrOtherInstrument, got, wants)
	default:
		return nil
	}
}

// InstrumentIn is what a reference recording's filename says was played
// through it, and empty where it cannot tell.
//
// The convention is the corpus's own: `resources/dry/bass-di.wav`. It is a
// filename rather than a field, so it is weaker evidence than one, which is
// why [PlayedOn] asks it second.
//
// Here rather than in `pkg/cli`, where it was, because the readings are what
// it is about and two packages wanted the same answer. `translate.gearIsFor`
// asks the same shape of question of a piece of gear a person wrote down and
// is deliberately separate: a name that does not say which instrument is a
// guess there, and merging the two would make one of the answers wrong.
func InstrumentIn(
	reference string,
) string {
	name := strings.ToLower(filepath.Base(reference))

	for _, known := range []string{"bass", "guitar"} {
		if strings.Contains(name, known) {
			return known
		}
	}

	return ""
}

// playedOn is what a reading was taken on: the field where it says so, and the
// reference recording's filename otherwise.
//
// The fallback is not a guess. Until 2026-09-29 no reading carried the field
// and the reference recording's name was the only trace of it, so 8 of the 215
// committed here say nothing and every one of them names `bass-di.wav`.
// Refusing those outright would have thrown away real readings: it took one
// EQ's controls off John Paul Jones's preset and put the catalog's defaults
// back, which is a worse answer than the one the file can still give.
//
// Empty is still the answer where neither says, and [Describes] refuses that.
func playedOn(
	said string,
	reference Reference,
) string {
	if said != "" {
		return said
	}

	return InstrumentIn(reference.File)
}

// PlayedOn is what was played through this library's readings.
//
// Asked rather than reading `Instrument` directly, everywhere a reading is
// held against a chain. The field is the answer where a reading has one and
// [playedOn] says what happens where it does not.
func (l Library) PlayedOn() string {
	return playedOn(l.Instrument, l.Reference)
}

// PlayedOn is what was played through this block's sweep, for the reason
// [Library.PlayedOn] exists.
func (c Curves) PlayedOn() string {
	return playedOn(c.Instrument, c.Reference)
}
