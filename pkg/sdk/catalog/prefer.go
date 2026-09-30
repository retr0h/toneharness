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

// Preferred is how far down the list this block's family sits, lowest first.
//
// A name that fits more than one model has to mean one of them, and this is
// the answer. It lives here rather than beside either caller because there
// are two of them: the gear resolver and the compiler both sort a name's
// matches, and two resolvers answering "which Ampeg SVT" differently is a rig
// that compiles into something other than what it resolved to. Spelled out
// twice it was two lists to keep in step, and nothing checked.
//
// Only the families where the choice is a real one are named. There are two.
//
// An amplifier before a preamp. A rig naming "Ampeg SVT" means the amplifier;
// 108 names on this device are both, and until this existed the answer was
// whichever sorted first, which happened to be right because HD2_Amp precedes
// HD2_Preamp. Right by the alphabet is not right by decision.
//
// A mic'd cabinet before a legacy one. The same cabinet ships three times and
// all three carry one name: HD2_Cab1x15TucknGo at 7.2 DSP with nothing but
// floats, and two HD2_CabMicIr models at 2.5 carrying Mic, Angle and
// Position. The mic'd one costs a third of the DSP and gives three more
// controls, one of them the twelve-microphone list the comparison in
// build-a-rig exists for, and 448 of the 1,126 corpus presets holding a
// cabinet hold a mic'd one. Without this the legacy model won every time and
// nothing could reach the others.
//
// The pan variant sits behind the plain one. Pan is a stereo placement
// control and a bass rig has no use for it, so it is a control a solve would
// spend readings on for nothing.
//
// A family not named here sorts last, and among themselves by identifier,
// which is every other collision on the device: two models of one family with
// one name are two spellings of the same thing.
func (b Block) Preferred() int {
	for at, family := range preferences {
		if b.Family == family {
			return at
		}
	}

	return len(preferences)
}

// preferences is the order families are chosen in, the most wanted first.
var preferences = []string{
	"amp", "preamp", "cabmicirs", "cabmicirswithpan", "cab",
}
