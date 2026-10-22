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
	"errors"
	"fmt"
)

// ErrUnknownBlock reports a model identifier absent from the catalog.
var ErrUnknownBlock = errors.New("unknown block model")

// ErrOverBudget reports a rig whose blocks exceed a DSP chip's ceiling.
var ErrOverBudget = errors.New("dsp budget exceeded")

// ErrBadTopology reports a rig whose block count, positions or chip
// assignments the device cannot represent.
var ErrBadTopology = errors.New("invalid topology")

// UnknownBlockError names the model identifier that was not found.
type UnknownBlockError struct {
	Model string
}

// Error implements the error interface.
func (e *UnknownBlockError) Error() string {
	return fmt.Sprintf("unknown block model %q", e.Model)
}

// Unwrap returns ErrUnknownBlock so callers can match with errors.Is.
func (*UnknownBlockError) Unwrap() error { return ErrUnknownBlock }

// ErrRefusedBlock is a block the device will not load.
var ErrRefusedBlock = errors.New("the device will not load this block")

// RefusedBlockError names the model the hardware refuses.
type RefusedBlockError struct {
	Model string
}

// Error implements the error interface.
//
// Says what happens rather than only what is wrong, because the failure this
// prevents is silent: the preset writes, the device accepts it, and the chain
// plays without the block in it.
func (e *RefusedBlockError) Error() string {
	return fmt.Sprintf(
		"%s: %s. The catalog offers it because Line 6's files do, and the "+
			"hardware answers a request to load it with a chain it is not in, "+
			"so a preset holding it writes and plays without it",
		ErrRefusedBlock, e.Model)
}

// Unwrap returns ErrRefusedBlock so callers can match with errors.Is.
func (*RefusedBlockError) Unwrap() error { return ErrRefusedBlock }

// OverBudgetError names the chip that overflowed and by how much.
type OverBudgetError struct {
	Chip    int
	Cost    float64
	Ceiling float64
}

// Error implements the error interface.
func (e *OverBudgetError) Error() string {
	return fmt.Sprintf("chip %d costs %.3f, ceiling %.3f", e.Chip, e.Cost, e.Ceiling)
}

// Unwrap returns ErrOverBudget so callers can match with errors.Is.
func (*OverBudgetError) Unwrap() error { return ErrOverBudget }

// TopologyError explains why a rig's shape is not representable.
type TopologyError struct {
	Reason string
}

// Error implements the error interface.
func (e *TopologyError) Error() string {
	return "invalid topology: " + e.Reason
}

// Unwrap returns ErrBadTopology so callers can match with errors.Is.
func (*TopologyError) Unwrap() error { return ErrBadTopology }
