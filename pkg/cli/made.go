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

package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/retr0h/toneharness/pkg/cli/internal/paint"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// Made says what was built and what it chose.
//
// Reporting the chain matters as much as writing the file. A generated preset
// is a set of decisions, and a wrong amp should be visible before anybody
// plugs in rather than after.
func Made(
	w io.Writer,
	m sdk.Made,
	cat *catalog.Catalog,
) error {
	return reporting(made(w, m, cat))
}

// made writes the parts, so a failure part way through is reported rather
// than leaving a half-written summary and a success.
func made(
	w io.Writer,
	m sdk.Made,
	cat *catalog.Catalog,
) error {
	if _, err := fmt.Fprintf(
		w, "\n%s%s\n\n", paint.Indent, paint.Title(w, m.Plan.Name),
	); err != nil {
		return err
	}

	if err := paint.Chain(w, m.Plan, cat); err != nil {
		return err
	}

	if err := added(w, m.Added); err != nil {
		return err
	}

	if err := says(w, m.Moved); err != nil {
		return err
	}

	if err := unfamiliar(w, m.Unfamiliar); err != nil {
		return err
	}

	if err := playing(w, m.Playing); err != nil {
		return err
	}

	if err := saysLevel(w, m.Levelled); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "\n%s%s\n\n", paint.Indent, paint.Success(w, "wrote "+m.Path))

	return err
}

// saysLevel says what was done to match another rig's loudness.
//
// After playing, because both come off the Setup and this is the one somebody
// reads to decide whether two presets can be compared at all. Said whether or
// not anything moved: a build that quietly failed to match is a build somebody
// A/Bs against another one and draws a conclusion from.
//
// **It reports a gap and says what the gap is of**, rather than printing
// either figure as the level the chain comes out at. A cabinet is measured
// alone and reads within a decibel of the empty loop, while the chain it ends
// is twenty louder, so "comes out at -59.6dB" would be a measurement nobody
// took. The gap is the part that is real and the part that decides the move.
//
// It used to call the gap one "between their cabinets", which stopped being
// true when the amplifier's own level controls became part of it: two presets
// on the same 8x10 can now differ by 6dB, and before they read as matched
// while one of them was barely audible. See `compile.expected`.
func saysLevel(
	w io.Writer,
	at sdk.Levelled,
) error {
	if !at.Asked() {
		return nil
	}

	if at.Param == "" {
		_, err := fmt.Fprintf(w,
			"\n%s%s %s, whose cabinet reads %.1fdB against this one's %.1fdB: %s\n",
			paint.Indent, paint.Mute(w, "as loud as"), at.Rig,
			at.Wanted, at.Was, at.Because)

		return err
	}

	_, err := fmt.Fprintf(w,
		"\n%s%s %s: %s %g to %g, closing %.1fdB of a %.1fdB gap between "+
			"their cabinets and amplifier levels\n",
		paint.Indent, paint.Mute(w, "as loud as"), at.Rig,
		at.Param, at.From, at.To,
		at.Matches-at.Was, at.Wanted-at.Was)

	return err
}

// playing says what was done about how this person plays.
//
// Last of the four, because it is the only one that comes off the Setup rather
// than off the ask, and reading it after the words makes it clear which of them
// it put there.
func playing(
	w io.Writer,
	held sdk.Playing,
) error {
	if held.Said == "" {
		return nil
	}

	_, err := fmt.Fprintf(w, "\n%s%s %s\n",
		paint.Indent, paint.Mute(w, "playing"), held.Said)

	return err
}

// added names the blocks nobody asked for, and why they are there.
func added(
	w io.Writer,
	all []sdk.Added,
) error {
	if len(all) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, a := range all {
		// A block drawn from the corpus can say how common it is. One
		// substituted for gear no model emulates cannot, and appending "0% of
		// chains" to it would read as a measurement.
		line := fmt.Sprintf("%s — %s", a.Name, a.Reason)

		switch {
		case a.Share > 0 && a.Bypassed:
			// Both figures, because they answered two questions and the
			// second is the one that decided the bypass: how many chains
			// carry one, and how many of those run it.
			line += fmt.Sprintf(" (%.0f%% of chains carry one, %.0f%% of those run it)",
				a.Share*100, a.Running*100)
		case a.Share > 0:
			line += fmt.Sprintf(" (%.0f%% of chains)", a.Share*100)
		}

		if _, err := fmt.Fprintf(w, "%s%s %s\n", paint.Indent, paint.Mute(w, "added"), line); err != nil {
			return err
		}
	}

	return nil
}

// says names the knobs a word turned, and the words that turned none.
//
// Both halves, because a term that moved nothing is still something the rig
// said. Reporting only the ones that worked would read as if the rest had.
//
// Labelled `says` rather than `heard`, which is what it printed until
// 2026-09-30. `heard` is a defined evidence kind and the contract is explicit
// about it: a person who played the rig and judged it, which outranks
// everything else because nothing in this system can hear. Printing it over
// every word claimed a human verdict for words nothing had listened to, in a
// tool whose whole argument is that it does not hear anything.
func says(
	w io.Writer,
	all []sdk.Moved,
) error {
	if len(all) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, m := range all {
		line := fmt.Sprintf("%s — nothing acts on this yet", m.Term)

		switch {
		case m.Contested():
			line = fmt.Sprintf(
				"%s — another term already answered for %s, so neither moved",
				m.Term, m.Against)
		case m.Yielded():
			line = fmt.Sprintf(
				"%s — a genre measured this, and you said %s, so yours stands",
				m.Term, m.YieldedTo)
		case m.Holds():
			line = fmt.Sprintf("%s — %s", m.Term, m.Already)
		case m.Beaten():
			// The outcome, and the word's own move beside it so its
			// contribution is still visible. Printing only what the word did
			// said a hand edit had not taken when it had, and printing only
			// the outcome would hide that the word moved at all.
			line = fmt.Sprintf(
				"%s — %s is %.2f, which the document states: the word would "+
					"have made it %.2f",
				m.Term, m.Param, m.Held, m.To)
		case m.Acted():
			line = fmt.Sprintf("%s — %s %.2f to %.2f", m.Term, m.Param, m.From, m.To)
			if m.Measured() {
				// Said out loud, because the same word on the same chain
				// moves a different distance depending on what was measured,
				// and a reader comparing two builds deserves to know why.
				line += fmt.Sprintf(" (%.0f%% of a step: what the records measured)",
					m.Weight*100)
			}

			if m.Pinned() {
				// The reason two genres asking for opposite things come out
				// alike. Without this the only evidence is two files that
				// match, which reads as the words having said nothing rather
				// than as the corpus having no opinion about the control they
				// reach for.
				line += fmt.Sprintf(
					" (a whole step here is %.3f, because the presets measured "+
						"agree that closely; one nobody has measured moves %.3f)",
					m.Step, m.Default)
			}
		case m.Unanswered():
			line = fmt.Sprintf("%s — %s", m.Term, m.Because)
		}

		if _, err := fmt.Fprintf(w, "%s%s %s\n",
			paint.Indent, paint.Mute(w, "says"), line); err != nil {
			return err
		}
	}

	return nil
}

// unfamiliar names the words nothing defines.
func unfamiliar(
	w io.Writer,
	all []sdk.Unfamiliar,
) error {
	if len(all) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, u := range all {
		line := fmt.Sprintf("no such word %q", u.Term)
		if len(u.Near) > 0 {
			line += " — did you mean " + strings.Join(u.Near, ", ") + "?"
		}

		if _, err := fmt.Fprintf(w, "%s%s %s\n",
			paint.Indent, paint.Mute(w, "note"), line); err != nil {
			return err
		}
	}

	return nil
}
