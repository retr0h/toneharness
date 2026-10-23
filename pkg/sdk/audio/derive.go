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

package audio

import "slices"

// Turning a measurement into a word, by where an artist sits among artists
// measured the same way.
//
// A figure on its own names nothing. 91% of the energy below 250Hz is not
// "scooped", because every isolated bass stem is mostly low: that figure
// describes the instrument. What names something is a player sitting clear of
// the others on a measure all of them were read for.

// Axis is one measure that can become a word, and the two words it becomes.
type Axis struct {
	// Key is the measured figure this axis reads, from MeasuredKeys.
	Key Figure `json:"key"`
	// More is the term for an artist above the rest, Less for one below.
	More string `json:"more"`
	Less string `json:"less"`
	// Why says what the figure is, for the evidence a derived term carries.
	Why string `json:"why"`

	// places is how many decimals this measure is reported to, which is the
	// smallest margin it can resolve. See resolves.
	places int

	// of is the range this axis reads from a gathered measurement.
	//
	// Carried here rather than looked up by Key, so there is no branch for a
	// measure no axis names. A switch over every key would have four arms
	// nothing can reach, which is a thing to delete rather than to test
	// around.
	of func(Across) Spread
}

// Axes is every measure a term may be derived from, and only those.
//
// Three of the six axes a term can move. The other three are left alone for
// reasons that are not oversights:
//
// Sag and reverb Mix have no measurement at all. Touch response and how much
// room is on a recording are not a band share or a centroid, and picking the
// nearest available figure because both happen to exist is how a number ends
// up deciding a control it has nothing to do with.
//
// The compressor's Attack has a measurement, and it measures a different
// thing from the word. transient reads how sharply a note's level rises.
// `percussive` on slap bass describes the character of the attack, the click
// of a string against a fret, which is spectral rather than a rise in level.
//
// Measured across three artists, the one whose rig says `percussive` scored
// lowest of the three. That was suspected to mean transient was really
// reading note density, since it is largest where there is silence to rise
// from. It is not: across 582 chunks of three seconds, with density running
// from nothing to continuous, the correlation is +0.026. transient is sound,
// and Flea rising less sharply than a picked punk bassist is true.
//
// So neither the word nor the figure is wrong, and no threshold between them
// would mean anything. The axis stays out until something measures the click.
//
// Measuring the click was tried, and it cannot be done from these stems. The
// measure was fixed before any music was run: the share of a note's energy
// above 1kHz in its first 30ms. It separates a plain synthetic pluck (0.000)
// from one with a click added (0.095). On five artists' separated bass stems
// every figure sat at the noise floor, 0.0000 to 0.0060, because the stems
// carry almost nothing above 1kHz; the high band reads 0 to 2% for every
// artist measured. The click lives in exactly what separation takes out.
// Recordings that keep the bass's top end, a DI or a separation that does not
// band-limit, would be needed first. A lower cutoff was not tried, because
// choosing one after seeing these figures would fit the measure to them.
var Axes = []Axis{
	{
		Key: KeyMid, More: "mid-forward", Less: "scooped",
		Why:    "share of energy between 250Hz and 2kHz",
		places: 2,
		of:     func(a Across) Spread { return a.Mid },
	},
	{
		Key: KeyCentroid, More: "bright", Less: "dark",
		Why:    "the spectrum's centre of gravity, in hertz",
		places: 0,
		of:     func(a Across) Spread { return a.Centroid },
	},
	{
		// One pole. A high harmonic share is evidence of drive and a low one is
		// not evidence of its absence, so this axis earns `saturated` and
		// nothing earns `clean`.
		//
		// The asymmetry is not a hedge, it is what the corpus shows. Four of
		// the five players earning `saturated` have a drive device written into
		// their own rig: Chuck Dukowski, Tim Commerford, Matt Freeman, and
		// Tobin Esperance with a Malekko B:Assmaster on his board. The high end
		// of this measure tracks something real.
		//
		// The low end tracked nothing. Three of the four players earning `clean`
		// have documented gain. Chi Cheng read 0.081, the second cleanest figure
		// measured, and his own rig quotes Premier Guitar on the Tech 21 SansAmp
		// GT2 he used "for extra balls". Troy Sanders read 0.059, the cleanest of
		// all, and runs a Wren and Cuff Tall Font Russian distortion. Klaus
		// Flouride read 0.077 on Dead Kennedys records nobody would call clean.
		//
		// The reason is the separation rather than the arithmetic. Distortion
		// products are broadband and sit where the guitars sit, so splitting a
		// mix assigns them to the guitar stem, and the dirtier and denser the
		// record the more of them leave. Grunge, Deftones and Mastodon are the
		// densest mixes here and they read the lowest. A measure that reports
		// its own blind spot as a positive finding is worse than one that says
		// nothing: `clean` was being asserted of the records least able to
		// support it.
		//
		// It is not brightness in disguise either, which was the other
		// suspicion. Centroid and this correlate +0.43, and the four `clean`
		// players span 97Hz to 191Hz of centroid while Tim Commerford earns
		// `saturated` at 109Hz. Dark and saturated is reachable, so the axis is
		// not reading the top end twice.
		//
		// What would restore the pole is a measurement of breakup rather than of
		// harmonic content, taken before separation or on a signal that never
		// needed it. `clean` in words.json means "no breakup at any dynamic",
		// and a median share of energy cannot speak about any dynamic: a player
		// who breaks up only when hit hard reads low, and the vocabulary already
		// has `grit-on-attack` and `minimal-drive` for exactly that.
		Key: KeyHarmonics, More: "saturated", Less: "",
		Why:    "share of energy above the fundamental",
		places: 2,
		of:     func(a Across) Spread { return a.Harmonics },
	},
	{
		// Here after being refused twice, because the measure it reads was
		// wrong both times and is not now.
		//
		// What it used to time was the arrangement. Two faults, and the first
		// was not the one that mattered: the onset test counted every frame
		// that happened to get louder, 6,131 of them in 199 seconds of God
		// Save the Queen, thirty a second. Peak-picking brings that track to
		// 718 onsets and its decay from 21.01s to 15.70s, which is still
		// nonsense, so the onsets were never the cause.
		//
		// The cause was the fall. A note was timed to wherever its level
		// finally dropped by 12dB, and in a dense mix that is a section change
		// rather than a string stopping. Giving each note two seconds to die
		// and dropping the ones that do not is what fixed it: the worst
		// drifter in the corpus went from 19.57s of spread across one player's
		// three records to 0.62s.
		//
		// A third rule was tried and refused. Cutting each note off at the
		// next onset is stable too, 0.15s of drift, and what it measures is
		// how fast somebody plays: every player landed between 0.11s and
		// 0.18s, a spread narrower than the drift inside one of them.
		//
		// Now it clears the same test the three above clear. Typical drift
		// inside one player's records over the spread between players, which
		// is what decides whether a figure can carry a comparison at all:
		//
		//	centroid   0.60
		//	decay      0.69
		//	harmonics  0.77
		//	mid        0.79
		//	low        0.79, and -0.994 correlated with mid: the same axis
		//	dynamics   1.05
		//	transient  1.15
		//	high       1.21
		//	even-odd   1.30
		//
		// Everything at or above 1 drifts further inside one player than it
		// separates two, so nothing there can ever earn a word however the
		// corpus grows. That is six of the ten figures measured, and
		// write-a-spec's terms.md says so where somebody reading the
		// vocabulary would otherwise assume a term in it is reachable.
		Key: KeyDecay, More: "long-decay", Less: "short-decay",
		Why:    "how long a note takes to fall to a quarter, in seconds",
		places: 2,
		of:     func(a Across) Spread { return a.Decay.Spread },
	},
}

// resolves says whether a margin is a difference this measure can see.
//
// A word clear by less than the figure is reported to is a word clear by
// nothing anybody can check: "bright, clear by 0 Hz" was awarded on a margin
// that rounded away, and one more player measured would have taken it back.
//
// The precision rather than a threshold somebody chose. Measured() reports a
// centroid to the hertz and a band share to two places, so below that there is
// no difference to report rather than a small one.
// offMiddle is how far from the middle of the others a figure has to sit before it
// is worth a word, as a share of their middle half.
//
// A quarter, and the corpus chose it on three axes. Measured across all 21
// genres on mid, centroid and harmonics, the margins as a share of the others'
// spread landed in two groups with nothing between them:
//
//	0.00 0.00 0.01 0.02 0.03 0.04 0.05 0.08 0.11 0.12 0.14 0.16 0.18 0.19 0.21
//	0.38 0.41 0.44 0.46 0.49 0.51 0.57 0.58 0.69 0.83 1.27 2.41
//
// Any threshold between 0.21 and 0.38 produced the same answer, so a quarter
// was the round number inside the gap rather than a line drawn through data.
//
// **Adding decay closed the gap.** On four axes the same measurement runs
// continuously through it: 0.20 0.20 0.20 0.20 0.25 0.27 0.29 0.29 0.30, with
// no break anywhere between 0.02 and 1.00 wider than 0.05. Sixty-three points
// had a gap and seventy-three do not, which is the ordinary fate of a gap found
// in a small sample.
//
// So the quarter is now a chosen threshold rather than one the data picks out,
// and it is kept because moving it would move every word in the corpus on no
// evidence at all. What would justify a different number is a reason to prefer
// it, and nothing here has one.
//
// Without it the median comparison awards almost everything. Southern rock sits
// 2.9Hz from the middle of a population spanning 65Hz, which is 4% of the
// spread, and would read `bright` on it. The precision floor cannot catch that:
// 2.9Hz rounds to 3Hz and resolves fine. What was missing is a scale, and the
// spread is the scale the report already prints a margin against.
//
// A tighter number was available and refused. At a tenth, southern rock earns
// `scooped` and `clean` and pop punk still earns nothing, which separates the
// two genres this corpus most wants separated. That is choosing the threshold
// for the answer it gives, on margins of 6Hz, and it is the mistake the note
// above Axes records somebody already making once with the click measure.
const offMiddle = 0.25

// clears says whether a margin is far enough from the middle of the others to
// mean something, by both floors: the measure's own precision, and a quarter of
// the spread of whoever it was measured against.
//
// A zero spread clears on precision alone. Every other subject read the same
// figure, so there is no scale to be a share of, and inventing one would refuse
// a word the figures plainly support.
func (a Axis) clears(
	margin float64,
	spread float64,
) bool {
	if !a.resolves(margin) {
		return false
	}

	return spread <= 0 || margin >= spread*offMiddle
}

func (a Axis) resolves(
	margin float64,
) bool {
	return to(margin, a.places) > 0
}

// Derived is one term an artist earned, and what earned it.
type Derived struct {
	// Term is the word, spelled as the compiler's own table spells it.
	Term string `json:"term"`
	// Key is the measure it came from, and Why describes that measure.
	Key Figure `json:"key"`
	Why string `json:"why"`
	// Mine is where this artist sat, Others the middle of everyone else.
	Mine   float64 `json:"mine"`
	Others float64 `json:"others"`
	// Of is how many artists the comparison was against, this one included.
	Of int `json:"of"`
	// Margin is how far past the line this player sits, in the measure's own
	// units: how much the other players' median would have to move to take
	// the word away.
	//
	// This is where distinctiveness lives now that the gate does not test for
	// it. Two artists may both be `dark`, and the margin is what says one of
	// them is barely darker than typical and the other is nothing like it.
	//
	// Two words that read alike are not alike. 39% of the energy in the mid
	// band against 4% is a word nothing will overturn; 8% against 4% flipped
	// the first time somebody else was measured, and nothing in the word
	// said which kind it was.
	//
	// This is the only stability figure there is, and it is about the other
	// players rather than this one. Dropping one of a player's own records
	// and deriving again cannot take a word away: the test is their whole
	// range against a quartile, the range is read as a tenth and a ninetieth
	// percentile, and removing a record can only raise the tenth or lower
	// the ninetieth. It was built, run against the nine players here, and
	// reported nothing for every word, which is what the arithmetic says it
	// must do.
	Margin float64 `json:"margin"`
	// Spread is how wide the other players' middle half is, in the measure's
	// own units, and it is what makes Margin readable.
	//
	// A margin is only ever large or small against something. One hertz clear
	// of players spanning two hundred is noise that happened to round the
	// right way; one hertz clear of players spanning three is the axis telling
	// them apart. Nothing could say which of those a word was, because the
	// word carried the distance and not the scale.
	//
	// Not a threshold. Nothing here refuses a word for sitting close, because
	// the floor that does exist is the measure's own precision rather than a
	// number somebody picked, and that decision stands. This is the figure a
	// reader needs to discount a word themselves, and the figure a build needs
	// to move a control less far for a word that barely earned it.
	Spread float64 `json:"spread"`
	// Against is the middle half the comparison was made against, low to high.
	// With Of, it is the whole population a word was earned over.
	//
	// Carried because the population moves. A word is earned against whoever
	// else is measured, so adding players rewrites verdicts nobody touched:
	// pop-punk earned `scooped` until eighteen southern rock records joined the
	// corpus and closed the margin. Stored this way a word says what it beat,
	// so a stale one can be seen rather than inferred.
	Against [2]float64 `json:"against"`
}

// Derive is what an artist's measurements say about them, against others
// measured the same way.
//
// An artist earns a term where their whole range sits on one side of the
// middle of the other artists: entirely above the others' median, or entirely
// below it. So a word says this one is consistently darker, or more
// mid-forward, than the typical artist measured.
//
// **A word describes a sound rather than proving it unique, so words overlap.**
// Any number of artists may be darker than the median, and nothing takes the
// word off one of them because another arrived carrying it too. What a genre
// is, is the combination of words it earns rather than the one word nobody
// else has.
//
// This is the second time the rule has been loosened, and both times for the
// same reason: it was answering "is this unusual" where the caller wanted
// "what does this sound like". It began as more than every member of the
// population, and got stricter with every artist added, because each one could
// only raise the highest middle or lower the lowest. With five artists at most
// one could clear an axis in each direction, and one player earned three terms
// against two other artists and none against four, which is more data giving
// fewer answers.
//
// Outside the middle half was better and had the same shape of fault. Grunge
// earned `scooped`, `dark` and `clean`, then twenty-five records of rock, metal
// and nu-metal joined the corpus, the metal players are dark, and grunge earned
// nothing. Grunge had not changed. Darkness had stopped being unusual. These
// words move controls, so a genre's preset was depending on which unrelated
// genres happened to have been measured.
//
// Distinctiveness is still worth knowing and it is Margin's job, not this
// gate's: the word says what the sound is, and the margin says how far from
// typical it sits.
//
// The median and the quartiles are interpolated between neighbours. Taken on a
// single neighbour, the upper quartile of four artists is simply the highest of
// them, which is the oldest rule back again.
//
// The whole range rather than the middle, because the middle alone is not
// enough: one artist reads 0% mid across two records and 15% on a third, so
// his median is a figure he never actually plays, and a term drawn from it
// would assert something his own records contradict.
//
// Nothing is derived where the evidence is mixed. With few artists that will
// be most of the time, and saying nothing is the right answer rather than a
// failure: a rig that names no term keeps the words a person chose.
func Derive(
	mine Across,
	others map[string]Across,
) []Derived {
	if mine.Tracks == 0 || len(others) == 0 {
		return nil
	}

	var out []Derived

	for _, ax := range Axes {
		rest := middles(others, ax.Key)
		if len(rest) == 0 {
			continue
		}

		slices.Sort(rest)
		span := ax.of(mine)

		// On one side of the middle of the rest, and far enough from it to
		// mean something. The quartiles are the scale that decides how far is
		// far enough, and the scale the margin is reported against.
		middle := between(rest, 0.5)
		lower, upper := between(rest, 0.25), between(rest, 0.75)
		spread := upper - lower

		switch {
		case span.Low > middle && ax.clears(span.Low-middle, spread):
			out = append(out, made(
				ax, ax.More, mine, rest, len(others)+1, span.Low-middle, lower, upper))
		case ax.Less != "" && span.High < middle &&
			ax.clears(middle-span.High, spread):
			out = append(out, made(
				ax, ax.Less, mine, rest, len(others)+1, middle-span.High, lower, upper))
		}
	}

	return out
}

// made is one derived term, carrying what it was derived from.
//
// rest arrives sorted, and is never empty: Derive returns early when it is.
func made(
	ax Axis,
	term string,
	mine Across,
	rest []float64,
	of int,
	margin float64,
	lower, upper float64,
) Derived {
	return Derived{
		Term: term,
		Key:  ax.Key,
		Why:  ax.Why,
		Mine: to(mine.Measured()[string(ax.Key)], ax.places),
		// The middle of the others, so a report can say what this was
		// clear of rather than only that it was.
		//
		// Rounded to the axis's places, the same as Mine. A median is an
		// average of floats and comes out as one: 0.22999999999999998 was
		// being written into shipped asks beside a `measured` of 0.23, which
		// reads as a precision nothing here has.
		Others: to(between(rest, 0.5), ax.places),
		Of:     of,
		Margin: margin,
		// Rounded like everything else here. A width is a difference between
		// two interpolated quartiles and arrives with a float's tail on it.
		Spread:  to(upper-lower, ax.places),
		Against: [2]float64{to(lower, ax.places), to(upper, ax.places)},
	}
}

// middles is where each of the other artists sits on one measure.
//
// An artist with no recordings is not a vote. Their Across is empty, and
// counting a zero as a position would drag every comparison toward it.
func middles(
	others map[string]Across,
	key Figure,
) []float64 {
	out := make([]float64, 0, len(others))

	for _, a := range others {
		if a.Tracks == 0 {
			continue
		}

		out = append(out, a.Figures()[string(key)])
	}

	return out
}

// between is the value a share of the way through a sorted list, interpolated
// between the two entries either side of that point.
//
// quantile in measure.go takes the entry the point lands on, which is right
// for a spread over hundreds of windows and wrong for a handful of artists:
// there, the upper quartile of four lands on the highest of them.
func between(
	sorted []float64,
	share float64,
) float64 {
	at := share * float64(len(sorted)-1)
	lo := int(at)

	// Only a list of one has nothing above its only entry. For anything
	// longer, share below one keeps lo below the last index.
	if lo+1 >= len(sorted) {
		return sorted[lo]
	}

	return sorted[lo] + (at-float64(lo))*(sorted[lo+1]-sorted[lo])
}
