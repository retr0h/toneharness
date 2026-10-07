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

package plan

import (
	"fmt"
	"slices"
)

// where is one processor and one side of its split, which is the grid a position
// is a slot in.
type where struct {
	chip int
	path int
}

// ValidateTopology reports a rig whose shape the device cannot represent:
// too many blocks, a processor that does not exist, or positions on one
// processor that are not the contiguous run 0..n-1.
//
// A rig with no blocks passes. It makes no sound, which is a preset somebody
// meant rather than one they left unfinished.
//
// It needs no catalog — every question it answers is about the rig alone.
func ValidateTopology(
	s Plan,
	lim Limits,
) error {
	// No blocks is a shape the device represents perfectly well: 102 presets in
	// the corpus hold none, and they are the MIDI remotes that drive Spotify or
	// Pro Tools from the footswitches and the blank templates people build from.
	// Every rule below is about where blocks sit, so there is nothing to check.
	if len(s.Blocks) == 0 {
		return nil
	}

	if len(s.Blocks) > lim.MaxBlocks {
		return &TopologyError{
			Reason: fmt.Sprintf(
				"%d blocks, device holds %d",
				len(s.Blocks),
				lim.MaxBlocks,
			),
		}
	}

	// Keyed by processor and by which side of a split the block is on, because a
	// position is a slot in one path's grid rather than an index into the chain.
	byChip := make(map[where][]int)

	for _, b := range s.Blocks {
		if b.DSP < 0 || b.DSP >= lim.Paths {
			return &TopologyError{
				Reason: fmt.Sprintf(
					"block %q is on processor %d, device has %d",
					b.Model, b.DSP, lim.Paths,
				),
			}
		}

		if b.Pos < 0 {
			return &TopologyError{
				Reason: fmt.Sprintf(
					"block %q has negative position %d",
					b.Model,
					b.Pos,
				),
			}
		}

		at := where{chip: b.DSP, path: b.Path()}
		byChip[at] = append(byChip[at], b.Pos)
	}

	for _, at := range sortedWheres(byChip) {
		positions := byChip[at]
		slices.Sort(positions)

		for i, p := range positions {
			// One slot, one block. Not a contiguous run: a position is a slot in
			// the grid the split and the join sit in too, and a real preset leaves
			// gaps. Of 8,970 processor-and-path groups in the preset corpus 6,689
			// have one, and requiring 0..n-1 refused every one of them. What does
			// hold is uniqueness, in all 8,969 groups that state a position.
			if i > 0 && p == positions[i-1] {
				return &TopologyError{
					Reason: fmt.Sprintf(
						"processor %d path %d has two blocks at position %d: %v",
						at.chip,
						at.path,
						p,
						positions,
					),
				}
			}
		}
	}

	return nil
}

// sortedWheres orders the groups, so a refusal names the same one every run.
func sortedWheres(
	of map[where][]int,
) []where {
	out := make([]where, 0, len(of))
	for at := range of {
		out = append(out, at)
	}

	slices.SortFunc(out, func(a, b where) int {
		if a.chip != b.chip {
			return a.chip - b.chip
		}

		return a.path - b.path
	})

	return out
}
