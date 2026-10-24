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
package catalog

// valveGuts are the controls nothing may set on somebody's behalf.
//
// Modelled valve behaviour rather than tone controls. Sag is the power supply
// drooping under load, Bias and BiasX how hard the output valves are run, Hum
// and Ripple the mains leaking into the signal.
//
// Keyed by the name the catalog gives the parameter, which is the only name
// these have: there is no field on the device marking a control as valve
// behaviour, and on an SV Beast all twelve controls read `generic_knob` from 0
// to 1, so Sag is indistinguishable from Treble by anything the device says.
//
// Every one of them is carried by amplifiers and nothing else, so a name is
// enough and the category does not have to be checked: Sag is on 199 blocks,
// Hum 160, Bias and BiasX 100 each, Ripple 91, and all of those are
// amplifiers. A test walks the catalog and fails if that stops being true,
// because a name reused on a compressor would start silently refusing a
// control somebody does want set.
var valveGuts = map[string]bool{
	"Sag":    true,
	"Bias":   true,
	"BiasX":  true,
	"Hum":    true,
	"Ripple": true,
}

// SetByHand reports whether a control is one somebody sets once and then
// leaves, rather than one anything may turn to answer a target.
//
// Here because three places need the same answer and the set cannot be in any
// of them. A solve filters its knobs with it, a starting position is chosen
// with it, and the compiler writes values with it. The compiler was the one
// that did not have it: a rig came out at Hum 0.875 and BiasX 0.0, where a
// hand-built preset its owner prefers has Sag and Hum at zero set
// deliberately. Two copies of a list of five names is how one of them gets
// missed, so there is one.
//
// What the arithmetic has to say about it, which is the problem. A band share
// closed by rebiasing the output valves closes just as well as one closed with
// the tone stack, and only one of those is what anybody meant. Asked for one
// player, a run took Sag from 0.500 to 1.000 and Hum to 0.775 closing band
// shares, and the chain it produced sounded worse to the person holding the
// pedal than the rig it started from. An Ampeg's sweep answers a request for
// harmonics with Sag at its top stop, so a block's own measurements argue for
// moving it.
//
// It takes these out of what gets decided and leaves them where the rig put
// them. A rig that states a Sag still gets that Sag; what cannot happen is
// something else choosing one.
func SetByHand(
	param string,
) bool {
	return valveGuts[param]
}

// Stompable reports whether a block is the kind of thing somebody puts on a
// footswitch.
//
// One owner, because there were two and they drifted. `compile` kept a map and
// `translate` kept a switch, and when the compressor was added to one of them
// the other went on believing a compressor is not a pedal: a build then put
// one on a switch while the chooser went on counting switches as though it had
// not. Two copies of a judgement is how one of them gets missed, which is the
// same lesson as [SetByHand].
//
// An amplifier and a cabinet are the sound rather than an effect on it, so
// neither is here: a switch on either is a switch spent turning the rig off.
// Everything else a chain holds is something a foot reaches for.
//
// A compressor is, and leaving it out cost two presets every switch they had.
// The corpus stomps 1,154 of them, it is in 85% of bass chains, and in the
// shortest chains it was the only block there was.
func Stompable(
	of Category,
) bool {
	switch of {
	case CategoryComp,
		CategoryDrive,
		CategoryFilter,
		CategoryMod,
		CategoryDelay,
		CategoryReverb,
		CategoryPitch,
		CategoryEQ:
		return true
	default:
		return false
	}
}
