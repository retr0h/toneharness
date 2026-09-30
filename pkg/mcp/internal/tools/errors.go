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

package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/retr0h/toneharness/pkg/sdk"
)

var (
	// ErrOneDocument is presets_compile given neither a rig nor a plan, or
	// both. They are the same shape at two levels of resolution, and building
	// from both would mean quietly picking one.
	ErrOneDocument = errors.New("name one of rig or plan")
	// ErrOneValue is device_turn given none of value, choice or switch, or
	// more than one. A device does not coerce, so the kind has to be chosen.
	ErrOneValue = errors.New("name exactly one of value, choice or switch")
	// ErrNoSource is presets_make given nothing to build from.
	ErrNoSource = errors.New("name a rig_id or a rig_path to build from")
	// ErrTwoSources is presets_make given both.
	ErrTwoSources = errors.New("name a rig_id or a rig_path, not both")
	// ErrNotInCatalog is corpus_presets_show's model measured but missing from the
	// catalog it was resolved against, a sign the corpus and catalog have
	// drifted apart.
	ErrNotInCatalog = errors.New("measured but not in the catalog")
	// ErrWouldOverwrite is presets_make or slots_export pointed at a file
	// that already exists, on a server started without --allow-writes.
	ErrWouldOverwrite = errors.New(
		"a file is already there, and replacing it needs the server started with --allow-writes")
)

// mayWrite refuses a path a file already sits at, unless the server was
// started with writes allowed.
//
// An agent picks the path. One naming somebody's own preset would otherwise
// replace it without anybody having agreed to that.
//
// This look is a courtesy: it refuses before a build runs or the pedal is
// claimed. It is not what keeps the file. A file can appear between the look
// and the write, so the write is told the same thing through existing, and
// refuses whatever is there when it lands.
func (h *handlers) mayWrite(
	path string,
) error {
	if h.allowWrites {
		return nil
	}

	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrWouldOverwrite, path)
	}

	return nil
}

// existing is what a write does about a file already at its path: replace it
// on a server started with writes allowed, and keep it otherwise.
func (h *handlers) existing() sdk.Existing {
	if h.allowWrites {
		return sdk.ReplaceExisting
	}

	return sdk.KeepExisting
}

// refused says a write the SDK refused for a file already at path the way
// mayWrite says it, so an agent reads one refusal however late the file
// appeared. Any other error comes back as it was.
func (h *handlers) refused(
	path string,
	err error,
) error {
	if !h.allowWrites && errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w: %s", ErrWouldOverwrite, path)
	}

	return err
}

// notInCatalog wraps ErrNotInCatalog with the model id that could not be
// found, so the agent sees which model and not just an opaque schema failure.
func notInCatalog(
	id string,
) error {
	return fmt.Errorf("%w: %s", ErrNotInCatalog, id)
}

// remedy names the tool to call next, for the errors that have one.
//
// The SDK says what went wrong and leaves the next step to its caller. A
// person at a terminal is told a command; an agent here is told a tool it can
// call. Any other error comes back as it was.
func remedy(
	err error,
) error {
	switch {
	case errors.Is(err, sdk.ErrNoSuchBlock):
		return fmt.Errorf("%w, call catalog_list to find one", err)
	case errors.Is(err, sdk.ErrNoSuchRig):
		return fmt.Errorf("%w, call rigs_list to see the rigs that ship", err)
	case errors.Is(err, sdk.ErrNoDevice):
		// A pedal powered from a charger rather than a data port looks
		// exactly like one that is switched off, and the power light is on
		// either way. Worth saying, because the agent cannot see the light
		// and the person it is talking to will check that first.
		return fmt.Errorf("%w: ask whether it is in a USB data port rather "+
			"than a charger, then call device_hardware", err)
	case errors.Is(err, sdk.ErrBus):
		// The session is gone either way and reopening is usually enough.
		// When it is not, the endpoint has stalled, and the wire README is
		// unambiguous: "the interface will not be claimed again until the
		// device is power cycled." An agent that does not know that retries
		// into a wall, which is worse than a person doing it.
		return fmt.Errorf("%w: call it again, and if it keeps failing ask for "+
			"the pedal to be powered off and on", err)
	case errors.Is(err, sdk.ErrNothingToBuildFrom):
		// The next move is a question, not a guess. A recording is measured
		// against every block the device has; an adjective has to be earned
		// against a population of players before it means anything.
		return fmt.Errorf("%w. Name a record to sound like, a player, or the "+
			"gear itself: `like: { recording: take.wav }` resolves fully, and "+
			"an adjective on its own does not", err)
	case errors.Is(err, sdk.ErrEmptySlot):
		return fmt.Errorf("%w, call slots_list to see which slots hold "+
			"anything", err)
	default:
		return err
	}
}
