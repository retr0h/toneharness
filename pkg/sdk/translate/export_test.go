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

package translate

// Exposed to this package's external tests. What a genre note says is a
// decision with four answers and only three are reachable through whichever
// records somebody has tagged, so the fourth is tested here directly.
var GenreNote = genreNote

// The same reason, for choosing gear. Three of these answer the cases where the
// catalog and the measured library disagree, or where a genre has nobody to be
// measured against, and neither state is reachable through the data that ships:
// every amplifier it measures is in the catalog, and all three measured genres
// have a population behind them. Reachable or not, they are the contract.
var (
	// Playable is whether a block is for the instrument in hand.
	Playable = playable
	// SplitsByInstrument is whether a category is grouped by what it is played
	// with at all. Only amplifiers reach it today, and they always are, so the
	// cabinet answer is read here rather than through a build.
	SplitsByInstrument = splitsByInstrument
	// DisplacedTo turns a genre into a target a block can be ranked against.
	DisplacedTo = displacedTo
)
