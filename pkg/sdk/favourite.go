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
package sdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/retr0h/toneharness/pkg/sdk/slot"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// ErrNoFavourites is returned when the setup names none.
var ErrNoFavourites = errors.New("the setup names no favourites")

// Step says which favourite to go to.
type Step int

const (
	// AtFirst is the first favourite the setup lists, wherever the pedal is
	// now. The one thing that always works: it needs nothing read off the
	// device and cannot be confused by what is loaded.
	AtFirst Step = iota
	// Onward is the favourite after the one being played.
	Onward
	// Back is the one before it.
	Back
)

// Chosen is the favourite a Step landed on.
type Chosen struct {
	// At is the slot now active, as the pedal prints it.
	At string `json:"at"`
	// Rig is what the setup says lives there. Empty where the setup lists the
	// slot as a favourite without saying what is in it, which is allowed and
	// is the one thing that stops a step recognising where it started.
	Rig string `json:"rig"`
	// Position is where At sits in the list, from one.
	Position int `json:"position"`
	// Of is how many favourites there are.
	Of int `json:"of"`
	// Playing is the rig the pedal was playing before the step, as the preset
	// in the edit buffer names itself. Empty where nothing was read, which is
	// every AtFirst.
	Playing string `json:"playing"`
	// From is the favourite slot Playing was recognised as. Empty where the
	// pedal was playing something that is not a favourite, and the step went
	// to the first instead.
	From string `json:"from"`
	// Change is what the device said about the load.
	Change Change `json:"change"`
}

// Favourite makes one of the setup's favourite slots the active preset.
//
// **What this is for.** A person playing moves between three or four sounds
// and leaves the other hundred and twenty alone. `device select` can reach any
// of them and needs the slot number every time, which is the wrong shape for
// the three: it asks somebody to remember a number while they are holding a
// bass. This asks the setup instead.
//
// Nothing is written. The preset goes into the edit buffer and the slot it
// came from is untouched, so this costs no flash write however many times it
// is run.
//
// **Where the pedal is now is read off the pedal, not remembered.** [Onward]
// and [Back] need a current position, and the obvious place to keep one is a
// file beside the setup. That file would be wrong most of the time: the
// footswitch is right there, and a person who steps on it has moved without
// telling this tool. So the edit buffer is read and the rig it names is
// matched against the rig the setup's `slots` puts in each favourite. A pedal
// playing something that is not a favourite is not an error; the step goes to
// the first, which is where somebody asking to cycle wanted to be anyway.
//
// That match is the one thing the pairing buys, and it is why a favourite
// worth cycling is worth an entry in `slots` as well.
func (c *Client) Favourite(
	ctx context.Context,
	step Step,
) (Chosen, error) {
	held, err := c.setup()
	if err != nil {
		return Chosen{}, err
	}

	if held.Favourites == nil || len(*held.Favourites) == 0 {
		return Chosen{}, fmt.Errorf(
			"%w: list them under favourites, in the order to cycle them",
			ErrNoFavourites)
	}

	favourites := *held.Favourites

	// Parsed before the device is opened. A slot the pedal has no position
	// for is a setup to fix rather than a session to claim, and claiming one
	// takes the front panel off whoever is playing.
	at := make([]int, len(favourites))

	for i, label := range favourites {
		index, err := slot.Parse(label)
		if err != nil {
			return Chosen{}, fmt.Errorf("favourite %s: %w", label, err)
		}

		at[i] = index
	}

	return once(ctx, c, func(s *Session) (Chosen, error) {
		return c.stepTo(ctx, s, step, favourites, at, rigsAt(held))
	})
}

// rigsAt is which rig the setup says each slot holds.
//
// Keyed by the slot's label rather than its index, because that is what both
// halves of the setup are written in and parsing one to compare it with the
// other would only add a way to disagree.
func rigsAt(
	held tone.Setup,
) map[string]string {
	out := map[string]string{}

	if held.Slots == nil {
		return out
	}

	for _, placed := range *held.Slots {
		out[placed.At] = placed.Rig
	}

	return out
}

// stepTo works out which favourite to go to and goes there.
func (c *Client) stepTo(
	ctx context.Context,
	s *Session,
	step Step,
	favourites []string,
	at []int,
	rigs map[string]string,
) (Chosen, error) {
	chosen := Chosen{Of: len(favourites)}
	landing := 0

	if step != AtFirst {
		// A pedal that will not say what it is playing is not a reason to
		// refuse: the request was to get to a favourite, and the first is a
		// favourite. The answer says nothing was recognised, which is the
		// same thing it says when the preset loaded is not one of these.
		if reading, err := s.Current(ctx, FormatRig); err == nil {
			chosen.Playing = reading.ID
		}

		if where, ok := heldBy(favourites, rigs, chosen.Playing); ok {
			chosen.From = favourites[where]
			landing = wrapped(where, step, len(favourites))
		}
	}

	chosen.At = favourites[landing]
	chosen.Rig = rigs[chosen.At]
	chosen.Position = landing + 1

	change, err := s.Select(ctx, slot.Address{Slot: at[landing]})
	if err != nil {
		return chosen, fmt.Errorf("going to %s: %w", chosen.At, err)
	}

	chosen.Change = change

	return chosen, nil
}

// heldBy finds which favourite holds the rig the pedal is playing.
//
// The first match wins where two favourites hold the same rig. That is a
// layout nothing can disambiguate: the edit buffer names a rig and both slots
// hold it, so either answer is as true as the other and picking the earlier
// one at least makes cycling repeatable.
func heldBy(
	favourites []string,
	rigs map[string]string,
	playing string,
) (int, bool) {
	if playing == "" {
		return 0, false
	}

	for i, label := range favourites {
		if rigs[label] == playing {
			return i, true
		}
	}

	return 0, false
}

// wrapped is the position one step from here, round the end of the list.
//
// Wrapping rather than stopping, because three favourites cycled with a
// footswitch in the other hand want next to keep working: a person who has
// reached the last one and presses next again means the first, and a command
// that refuses there has made them look at the list to find out why.
func wrapped(
	from int,
	step Step,
	of int,
) int {
	if step == Back {
		return (from - 1 + of) % of
	}

	return (from + 1) % of
}
