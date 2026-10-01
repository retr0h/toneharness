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

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

// Hands is what one right hand does that another does not, measured.
//
// The one thing in this project that describes a player rather than a device or
// a record. A rig is built from somebody's playing, so a preset handed to
// somebody who plays differently is wrong by the difference between the two
// hands, and this is that difference as a number instead of as a direction
// somebody was sure about.
type Hands struct {
	// From and To are the plucking styles compared, as the contract spells
	// them: the figures are what To reads minus what From reads.
	From string `json:"from"`
	To   string `json:"to"`
	// Pairs is how many notes were held against each other.
	//
	// Pairs rather than recordings, because the comparison is per note: the
	// same note on the same instrument at the same pickup setting, played two
	// ways. A difference of two averages over a note range would carry the
	// range's spread rather than the hand's.
	Pairs int `json:"pairs"`
	// Agreed is how many of those pairs moved in the direction Mean says.
	//
	// The useful half of the claim. A mean with a wide spread says little on
	// its own; a mean every pair agreed with says the direction is not in
	// doubt however wide the magnitude is.
	Agreed int `json:"agreed"`
	// Figures are what moved, keyed as the measured keys are, each the mean
	// and the spread of the per-pair difference.
	Figures map[Figure]Moved `json:"figures"`
}

// Moved is one figure's difference between two hands.
type Moved struct {
	// Mean and Median are the per-pair difference, in that figure's own unit.
	// Both, because a skewed set makes them disagree and which one a step is
	// sized from is a decision rather than an accident.
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	// Spread is the standard deviation of the per-pair difference.
	//
	// Reported beside the mean rather than folded into it. On this data it is
	// nearly as large as the mean, which is what says the direction is
	// reliable and the magnitude is not, and so what argues for one step
	// sized by the corpus rather than an offset in hertz.
	Spread float64 `json:"spread"`
}

//go:embed data/hands.json
var handsRaw []byte

// MeasuredHands is every pair of hands anybody has measured.
//
// Read once. The file is small and the figures do not change while a process
// runs, so the first caller pays and the rest read the same answer.
var MeasuredHands = sync.OnceValues(func() ([]Hands, error) {
	return unpackHands(handsRaw)
})

// unpackHands parses the embedded measurements, named so a failure says which
// file rather than which line of JSON.
func unpackHands(
	body []byte,
) ([]Hands, error) {
	var out []Hands
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("reading the measured hands: %w", err)
	}

	return out, nil
}

// HandsBetween is what moves between two plucking styles, in that direction.
//
// Either order answers. The figures are signed, so the reverse of a measured
// pair is that pair negated rather than a thing nobody measured, and a caller
// comparing fingers to a pick should not have to know which way round the file
// happens to store it.
func HandsBetween(
	from, to string,
) (Hands, bool) {
	all, _ := MeasuredHands()

	for _, h := range all {
		switch {
		case h.From == from && h.To == to:
			return h, true
		case h.From == to && h.To == from:
			return h.reversed(), true
		}
	}

	return Hands{}, false
}

// reversed is the same comparison the other way round.
func (h Hands) reversed() Hands {
	out := Hands{
		From:    h.To,
		To:      h.From,
		Pairs:   h.Pairs,
		Agreed:  h.Agreed,
		Figures: make(map[Figure]Moved, len(h.Figures)),
	}

	for key, moved := range h.Figures {
		out.Figures[key] = Moved{
			Mean:   -moved.Mean,
			Median: -moved.Median,
			Spread: moved.Spread,
		}
	}

	return out
}
