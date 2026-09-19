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

	"github.com/retr0h/tonestack/pkg/sdk/internal/presets"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// once opens a Session, makes one call on it and closes it.
//
// What the one-shot device methods are: the CLI runs one operation per
// process, and a caller who wants several opens a Session instead.
func once[T any](
	ctx context.Context,
	c *Client,
	call func(*Session) (T, error),
) (T, error) {
	s, err := c.Open(ctx)
	if err != nil {
		var zero T

		return zero, err
	}

	// Deferred, so a flow that panics still lets the pedal go on its way up
	// the stack. The call's own answer is what the caller needs: a read loop
	// that ended during the call already failed it with that error, and one
	// that ended afterwards changed nothing the call reported.
	defer func() { _ = s.Close() }()

	return call(s)
}

// Presets reports what a setlist on the attached device holds, slot by slot.
//
// Each of the Client's device methods opens a Session, makes one call on it
// and closes it, so each costs a claim and a handshake. A caller making several
// opens a Session instead. For a .hls or .hlb on disk, see Setlist.
func (c *Client) Presets(
	ctx context.Context,
	setlist int,
) (Listing, error) {
	return once(ctx, c, func(s *Session) (Listing, error) {
		return s.Presets(ctx, setlist)
	})
}

// Preset reads one slot on the attached device as the rig it describes.
//
// A rig, not a rendering of one. What comes out compiles back into the preset
// it came from, unchanged.
func (c *Client) Preset(
	ctx context.Context,
	at slot.Address,
) (Reading, error) {
	return once(ctx, c, func(s *Session) (Reading, error) {
		return s.Preset(ctx, at)
	})
}

// Current reads what the attached device is playing, as the rig it describes.
//
// The edit buffer rather than a slot, which is the difference that matters
// after Turn: a control moved with Turn shows here and not in the slot it came
// from, so reading the slot back reads as though nothing happened.
func (c *Client) Current(
	ctx context.Context,
	as Format,
) (Reading, error) {
	return once(ctx, c, func(s *Session) (Reading, error) {
		return s.Current(ctx, as)
	})
}

// Play puts a preset file in front of the attached device without storing it.
//
// Nothing is written to a slot, so this is what auditioning is: the cost of
// trying a chain is the time it takes to hear it rather than a flash write.
func (c *Client) Play(
	ctx context.Context,
	file string,
) error {
	_, err := once(ctx, c, func(s *Session) (struct{}, error) {
		return struct{}{}, s.Play(ctx, file)
	})

	return err
}

// Export writes one slot on the attached device out to a file, as a rig or as
// the device's own file.
//
// existing says what happens to a file already at out, as it does for Build.
func (c *Client) Export(
	ctx context.Context,
	at slot.Address,
	out string,
	as Format,
	existing Existing,
) (Written, error) {
	return once(ctx, c, func(s *Session) (Written, error) {
		return s.Export(ctx, at, out, as, existing)
	})
}

// Import puts a preset file into a slot on the attached device.
//
// Whatever the slot held is gone. A device has no undo, so what was there is
// read and kept first, in the directory WithBackupDir named.
func (c *Client) Import(
	ctx context.Context,
	file string,
	at slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Import(ctx, file, at)
	})
}

// Copy puts what one slot on the attached device holds into another.
//
// The preset moves exactly as it was written. Nothing is decoded and nothing
// is rebuilt, which is what makes this the safest thing to write: a device
// seeks through a preset by a table of byte offsets, and the surest way to
// keep those right is to change nothing.
func (c *Client) Copy(
	ctx context.Context,
	from, to slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Copy(ctx, from, to)
	})
}

// Swap exchanges what two slots on the attached device hold.
//
// This is also how a preset is moved. When one slot holds no preset the swap
// is a move: the preset goes into the empty slot, and the slot it came from
// is emptied the way a device empties one, so nothing is invented.
//
// Two slots that both hold no preset are refused with an EmptySwapError,
// before anything is kept or written.
func (c *Client) Swap(
	ctx context.Context,
	a, b slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Swap(ctx, a, b)
	})
}

// Select makes one preset the active one on the attached device.
//
// The device loads it and starts making that sound. Nothing is written: the
// preset goes into the edit buffer and the slot it came from is untouched, so
// this is the one device operation that changes what you hear without
// changing what the device holds.
func (c *Client) Select(
	ctx context.Context,
	at slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Select(ctx, at)
	})
}

// Turn moves one control on the preset the attached device is playing.
//
// Neither written nor selected: the change lands in the preset in front of
// somebody and is heard at once. The block is its position in the chain and
// the parameter its position in that model's list, which is the only thing
// that identifies either on the wire, and the value is in the parameter's own
// units.
//
// This is the operation a sweep is made of, and writing a preset per step is
// not a substitute: a slot given a new document goes on sounding like what it
// held before.
func (c *Client) Turn(
	ctx context.Context,
	at Address,
	value float32,
) error {
	_, err := once(ctx, c, func(s *Session) (struct{}, error) {
		return struct{}{}, s.Turn(ctx, at, value)
	})

	return err
}

// Choose picks one of a parameter's settings on the preset the attached
// device is playing, for the ones that are a list rather than a range.
func (c *Client) Choose(
	ctx context.Context,
	at Address,
	value int,
) error {
	_, err := once(ctx, c, func(s *Session) (struct{}, error) {
		return struct{}{}, s.Choose(ctx, at, value)
	})

	return err
}

// Switch turns one of a parameter's switches on or off on the preset the
// attached device is playing.
func (c *Client) Switch(
	ctx context.Context,
	at Address,
	on bool,
) error {
	_, err := once(ctx, c, func(s *Session) (struct{}, error) {
		return struct{}{}, s.Switch(ctx, at, on)
	})

	return err
}

// PresetFile reads a standalone .hlx as the rig it describes.
//
// The same rig a slot on a device or in a setlist reads as, which is the
// point: what the device holds and what this tool generates are the same kind
// of thing.
func (c *Client) PresetFile(
	ctx context.Context,
	path string,
) (Reading, error) {
	return c.fileOperations().ShowFile(ctx, path)
}

// Compile says what rig to build, what to build it into, and where the preset
// goes.
//
// A struct rather than three arguments, because three paths of one type are
// too easy to pass in the wrong order.
type Compile struct {
	// Rig is the rig file to build.
	Rig string
	// Template is a preset to write the chain into. Empty uses an untouched
	// one the device itself wrote.
	Template string
	// Out is where the preset is written.
	Out string
	// Existing is what happens to a file already at Out. The zero value,
	// ReplaceExisting, puts the preset in its place; KeepExisting refuses it
	// with an error matching fs.ErrExist.
	Existing Existing
}

// Compile builds a preset from a rig on disk.
func (c *Client) Compile(
	ctx context.Context,
	in Compile,
) (Built, error) {
	return presets.Compile(ctx, presets.CompileOptions{
		Deps:         presets.Deps{Catalogs: c},
		RigPath:      in.Rig,
		TemplatePath: in.Template,
		OutputPath:   in.Out,
		Existing:     in.Existing,
	})
}
