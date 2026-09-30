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

package cab

// This file is what the package comment has pointed at since it was written,
// and what did not exist until somebody followed the reference. Two places said
// "see Limits" and there was no Limits, so the one disclosure that matters most
// here was documented as being documented.

// Limits is what Capture and Match cannot do.
//
// Kept as prose in one place rather than scattered across the functions, because
// the reason it exists is not engineering hygiene. An impulse response is a file
// somebody can sell, and every item below is a way a file can measure right and
// be the wrong thing. A matched cabinet sold as the real thing is a lie somebody
// paid for.
//
// # Match chooses a phase it cannot know
//
// A magnitude response says nothing about phase, and the ratio of two magnitudes
// is a magnitude. So Match invents one, and picks minimum phase.
//
// For a loudspeaker that is close: a speaker is roughly minimum phase, so a
// filter correcting one cabinet towards another lands near enough that the
// difference is hard to hear. That is the case this was built for.
//
// It is wrong for anything holding a reflection. A room, or a second microphone
// at a distance, has a delayed copy of the signal in it, and a delay is not
// minimum phase. Match will reproduce the frequency balance and not the time
// structure, and what a listener notices is the sense of space rather than the
// tone.
//
// **This is invisible to every figure this repository measures.** Band shares
// and a centroid are magnitude. So a correction fitted to a room passes the check
// in full and sounds wrong, which is the one failure mode this project treats as
// worse than an error.
//
// Half of it is fixable and half is not. Where both responses were recovered by
// Capture the complex spectra are in hand, and dividing those rather than their
// magnitudes keeps the real phase; nothing here does that yet, because
// cabinet-to-cabinet does not need it. Where the target is a record's spectrum
// there is no phase to recover, and no implementation can invent the right one.
//
// # Capture needs the two recordings to correspond
//
// Deconvolution divides what came back by what went out, bin by bin, so the two
// have to be the same passage of time. A fixed delay is harmless and shifts the
// answer in time. Drift is not: two audio devices on separate clocks slide apart,
// and twenty samples of slide smears a 1024-tap response.
//
// So Capture wants one device playing and recording, which is `--hardware` given
// a single name. The rig that plays through the computer and records off the
// pedal opens the measuring loop and is better for everything else, and it is
// two clocks.
//
// # A truncated response loses the bottom first
//
// An HX Stomp loads 1024 or 2048 taps, which is 21 or 43 milliseconds. Whatever
// the cabinet did after that is discarded, and what lives in the tail is the low
// end: a 40Hz cycle is 25ms on its own. So the shorter length is not a smaller
// version of the longer one, it is the same response with less resolution
// underneath the fundamental of a bass.
//
// # A quiet bin cannot be divided
//
// Capture regularises, because a bin holding nothing would divide to an enormous
// number that is entirely noise. A sweep has energy everywhere by design and a
// recording of music does not, so a response recovered from a record is
// trustworthy only where the record had something to say. Quiet bins come back
// flattened rather than wrong, which is the safer failure and is still not the
// cabinet.
//
// # None of it has been near a pedal
//
// Both are tested against synthetic speakers. "The arithmetic works" and "the
// device loaded it and it sounded like the target" are different claims and only
// the first is currently true. See the task list's cabinet verification entry.
//
// # What may be sold
//
// A capture of hardware somebody owns is theirs. A correction derived by
// measuring a competitor's commercial impulse response is a copy with extra
// steps. One fitted to a record carries somebody's recording in it. Which of the
// three a file is goes in the file, and anything built for another platform needs
// the same.
const Limits = `Match invents a phase it cannot know, and picks minimum phase: close for a
loudspeaker, wrong for anything with a reflection in it, and invisible to every
figure measured here because those are magnitude.

Capture needs its two recordings to be the same passage of time, so it wants one
audio device rather than two on separate clocks.

1024 or 2048 taps discards whatever the cabinet did after 21 or 43ms, and what
lives in the tail is the low end.

A response recovered from music is trustworthy only where the music had energy.

Neither has been verified on a pedal.

A capture of hardware somebody owns is theirs to sell. One derived from a
commercial impulse response is a copy. One fitted to a record carries somebody's
recording in it.`
