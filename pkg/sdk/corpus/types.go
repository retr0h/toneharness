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

// Package corpus holds what a body of real presets says about how a device is
// actually used.
//
// The catalog says what a device *can* do; this says what people *do* with it.
// They answer different questions and neither substitutes for the other. Line
// 6 states a default Treble of 0.77 for the Ampeg SVT's bright channel; across
// the corpus presets using it the median is 0.85. Both are facts, and the second is the one worth
// generating from.
//
// Nothing here is authority. It is a measurement over presets strangers made,
// including their mistakes — which is why it reports a spread alongside every
// median, and why a wide spread should defer to a person.
package corpus

import "github.com/retr0h/toneharness/pkg/sdk/catalog"

// Stats is a whole corpus, measured.
type Stats struct {
	// Device names the hardware these presets were written for.
	Device   string `json:"device"`
	DeviceID int    `json:"device_id"`
	// Presets is how many were measured, so a reader can judge the weight of
	// everything below.
	Presets int `json:"presets"`
	// Models is keyed by the device's own model identifier.
	Models map[catalog.ModelID]ModelStats `json:"models"`
	// Grammar is keyed by instrument — "guitar", "bass" — because a bass
	// chain and a lead chain are built differently and averaging them
	// describes neither.
	Grammar map[string]Grammar `json:"grammar"`
}

// ModelStats is how often one model is used, and how it is set when it is.
type ModelStats struct {
	Uses   int                   `json:"uses"`
	Params map[string]ParamStats `json:"params"`
}

// ParamStats is the distribution of one parameter's values across the corpus.
//
// The quartiles matter as much as the median. A parameter everybody sets the
// same way is one this system can be confident about; a parameter nobody
// agrees on is a judgement call that belongs to the player, and the spread is
// how those are told apart.
type ParamStats struct {
	N      int     `json:"n"`
	Median float64 `json:"median"`
	P25    float64 `json:"p25"`
	P75    float64 `json:"p75"`
}

// Spread is the interquartile range: how much players disagree.
//
// Near zero means consensus and a value worth adopting. Wide means taste, and
// a generated preset should either leave it at the catalog's default or take
// direction from the rig rather than pretend the median means something.
func (p ParamStats) Spread() float64 { return p.P75 - p.P25 }

// Grammar is what a chain for one instrument tends to contain.
type Grammar struct {
	// Chains is how many were measured.
	Chains int `json:"chains"`
	// Categories is keyed by what a block does.
	Categories map[catalog.Category]CategoryStats `json:"categories"`
	// Models counts the chains for this instrument holding each model.
	//
	// Beside the categories rather than in Stats.Models, because a total
	// across instruments is a guitar figure: guitar chains outnumber bass
	// twenty to one, so the compressor bass players reach for loses to the
	// one guitar players do. Empty in statistics measured before it was
	// counted.
	Models map[catalog.ModelID]int `json:"models,omitempty"`
}

// CategoryStats is how often a kind of block appears, and where it sits.
type CategoryStats struct {
	// Chains is how many chains hold at least one.
	Chains int `json:"chains"`
	// Before and After count instances either side of the amp. Position in a
	// chain is not decoration: drive before an amp overdrives its input,
	// drive after it does something else entirely.
	Before int `json:"before"`
	After  int `json:"after"`
	// Engaged is how many of those chains have at least one of these
	// switched on.
	//
	// The figure that separates two questions one number was answering. A
	// chain holding a block and a chain using it are different facts, and the
	// corpus disagrees about them sharply: of the 123 bass chains ending in a
	// cabinet, 73.2% hold a drive and 35.0% have one running, so barely half
	// the players who carry one leave it engaged. A gate reading only
	// presence and then switching the block on decides for them.
	//
	// It goes the other way too, and that is the useful half. 13.0% hold a
	// noise gate and every one of them runs it; 5.7% hold a wah and not one
	// does, which is a footswitch effect sitting there waiting to be stepped
	// on rather than a block anybody wanted in the signal.
	//
	// Empty in statistics measured before it was counted, which reads as
	// nothing engaged and so as a block to add bypassed. That is the safe
	// direction: a block switched off changes no sound.
	Engaged int `json:"engaged,omitempty"`
	// Twice is how many of those chains hold two or more in one series run.
	//
	// Within one run, because that is the only arrangement a rig's chain can
	// express: two amplifiers on opposite sides of a parallel split is a
	// dual-amp rig people really build, and it is not what appending to a
	// flat chain produces. Counting the split as a repeat would read the
	// evidence for a chain nothing here can write.
	//
	// Every block the player placed, bypassed or not. A bypassed drive is a
	// drive the player put in the chain to switch in, so it is evidence about
	// the shape they built; counting only what was enabled answers the
	// different question of what happened to be on at save time, and it moves
	// bass drive from 19.6% of chains to 2.0%.
	//
	// Empty in statistics measured before it was counted.
	Twice int `json:"twice,omitempty"`
}

// Frequency is the share of chains holding at least one of these.
func (c CategoryStats) Frequency(
	chains int,
) float64 {
	if chains == 0 {
		return 0
	}

	return float64(c.Chains) / float64(chains)
}

// Running is the share of chains holding one of these that have one switched
// on.
//
// What decides whether a block the corpus says to add arrives enabled or
// bypassed. A compressor reads 83.3% here, so a chain that has one uses it and
// adding one switched on is what players do. A drive reads 47.8%, so adding one
// switched on is a decision half of them did not make.
//
// Zero where nothing counted it, which reads as bypassed. A block switched off
// changes no sound, so that is the direction to be wrong in.
func (c CategoryStats) Running() float64 {
	if c.Chains == 0 {
		return 0
	}

	return float64(c.Engaged) / float64(c.Chains)
}

// Repeats is the share of chains holding one of these that hold two.
//
// What it is for: deciding whether a chain may be given a second of a kind it
// already holds. Across the corpus every category people place twice sits
// above 4% of the chains that hold one, and the amplifier sits at 1.2% for
// bass and 7.3% for guitar once a parallel split is discounted. So the figure
// separates the kinds a chain may repeat from the one it may not, and it does
// it from the evidence rather than from anybody's assertion.
//
// Zero where nothing counted it, which reads as "do not repeat this". That is
// the safe direction: a chain that stays as it is beats one given a block on
// no evidence.
func (c CategoryStats) Repeats() float64 {
	if c.Chains == 0 {
		return 0
	}

	return float64(c.Twice) / float64(c.Chains)
}

// BeforeAmp is the share of instances that sit ahead of the amp.
func (c CategoryStats) BeforeAmp() float64 {
	total := c.Before + c.After
	if total == 0 {
		return 0
	}

	return float64(c.Before) / float64(total)
}
