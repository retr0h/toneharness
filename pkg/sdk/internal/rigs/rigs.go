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

// Package rigs finds and reports the curated knowledge on disk.
//
// A rig is a RigSpec that ships with the project. There is no separate
// rig format: what a person writes by hand, what a preset lifts to, and
// what compiles back down are all the same document.
package rigs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/shipped"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// DefaultDir is where rigs live.
const DefaultDir = "pkg/sdk/shipped"

// Load reads every rig under dir, in identifier order.
//
// A file that does not satisfy the contract stops the walk: a half-read
// knowledge base is worse than a clear complaint about the file to fix.
func Load(
	dir string,
) ([]rig.Spec, error) {
	all, err := readBase(dir)
	if err != nil {
		return nil, err
	}

	return specs(all), nil
}

// readBase reads the rigs a layer sits on, refusing any file that is not a
// rig.
func readBase(
	dir string,
) ([]stored, error) {
	// No directory means the rigs that ship in the binary, which is the
	// case for anyone who has not written their own.
	fsys, name := fs.FS(shipped.FS), "the built-in rigs"
	if dir != "" {
		fsys, name = os.DirFS(dir), dir
	}

	all, broken, err := readFS(fsys, name)
	if err != nil {
		return nil, err
	}

	if len(broken) > 0 {
		return nil, broken[0].err
	}

	return all, nil
}

// readFS reads every rig under the root of fsys, wherever that filesystem
// comes from. name says where that is, for a directory that cannot be read.
//
// A file that will not open or will not decode is handed back rather than
// stopping the walk, so the caller decides what one costs.
func readFS(
	fsys fs.FS,
	name string,
) ([]stored, []brokenFile, error) {
	// Glob drops a directory it cannot read, which would make one nobody may
	// open look like one holding no rigs. A directory that is not there is
	// different: nobody has written a rig into it yet.
	top, err := fs.ReadDir(fsys, ".")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("reading %s: %w", name, err)
	}

	// The same holds one level down, where the glob looks for rigs: an
	// artists/ nobody may open would otherwise hide every rig in it.
	for _, e := range top {
		if !e.IsDir() {
			continue
		}

		if _, err := fs.ReadDir(fsys, e.Name()); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("reading %s: %w", filepath.Join(name, e.Name()), err)
		}
	}

	// The pattern is a constant, so it cannot be malformed.
	paths, _ := fs.Glob(fsys, path.Join("*", "*.yaml"))

	out := make([]stored, 0, len(paths))
	broken := []brokenFile(nil)

	// The asks first, keyed by the stem they belong to, because a rig is what
	// this returns and an ask has to be in hand by the time its rig is built.
	asks, broken := readAsks(fsys, paths, broken)

	for _, p := range paths {
		if isAsk(p) {
			continue
		}

		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			broken = append(broken, brokenFile{
				names: claimed(p, nil),
				err:   fmt.Errorf("opening %s: %w", path.Base(p), err),
			})

			continue
		}

		spec, err := decode(raw, p)
		if err != nil {
			broken = append(broken, brokenFile{names: claimed(p, raw), err: err})

			continue
		}

		held := stored{spec: spec, raw: raw}
		if beside, ok := asks[stem(p)]; ok {
			held.ask, held.askRaw = beside.spec, beside.raw
		}

		out = append(out, held)
	}

	sortEntries(out)

	return out, broken, nil
}

// asked is one ask as read, and the text it came from.
type asked struct {
	spec *tone.Spec
	raw  []byte
}

// readAsks reads every ask a directory holds, by the stem it belongs to.
//
// An ask that will not parse is reported the same way a rig that will not parse
// is, rather than leaving the rig to load without it: a file somebody wrote and
// got wrong is the case where saying so matters, and a rig quietly missing the
// words it was built from is the same bug this split exists to remove.
func readAsks(
	fsys fs.FS,
	paths []string,
	broken []brokenFile,
) (map[string]asked, []brokenFile) {
	out := map[string]asked{}

	for _, p := range paths {
		if !isAsk(p) {
			continue
		}

		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			broken = append(broken, brokenFile{
				names: claimed(p, nil),
				err:   fmt.Errorf("opening %s: %w", path.Base(p), err),
			})

			continue
		}

		spec, err := tone.Load(bytes.NewReader(raw))
		if err != nil {
			broken = append(broken, brokenFile{
				names: claimed(p, raw),
				err:   fmt.Errorf("%s: %w", path.Base(p), err),
			})

			continue
		}

		out[stem(p)] = asked{spec: &spec, raw: raw}
	}

	return out, broken
}

// askSuffix is what names the ask beside a rig.
//
// Two files in one directory rather than two directories, because they are one
// subject and a person editing the words wants the gear in the next tab, not in
// a parallel tree they have to keep in step by hand.
const askSuffix = ".tone.yaml"

// isAsk reports whether a path names the ask rather than the rig.
func isAsk(
	p string,
) bool {
	return strings.HasSuffix(p, askSuffix)
}

// stem is the identifier two paired files share.
//
// The filename is the identifier, which is what the rigs already do: all nine
// that ship state an `id` exactly equal to their own stem. So a pair needs no
// field pointing at its other half, and cannot end up with one that disagrees.
func stem(
	p string,
) string {
	base := path.Base(p)
	if isAsk(p) {
		return strings.TrimSuffix(base, askSuffix)
	}

	return strings.TrimSuffix(base, ".yaml")
}

// decode parses one rig, naming the file it came from when it will not parse.
// A rig is hand-written, so the name is the useful half of the message.
func decode(
	raw []byte,
	name string,
) (rig.Spec, error) {
	spec, err := rig.Load(bytes.NewReader(raw))
	if err != nil {
		return rig.Spec{}, fmt.Errorf("%s: %w", filepath.Base(name), err)
	}

	return spec, nil
}

// read reads every rig a Source holds.
func read(
	src Source,
) (set, error) {
	base, err := readBase(src.Dir)
	if err != nil {
		return set{}, err
	}

	if src.User == "" {
		return set{base: base}, nil
	}

	user, broken, err := readFS(os.DirFS(src.User), src.User)
	if err != nil {
		return set{}, err
	}

	return set{user: user, base: base, broken: broken}, nil
}

// List reads every rig a Source holds.
//
// A file of somebody's own that is not a rig is reported here, every one of
// them, because a listing is where somebody looks for what they wrote.
func List(
	src Source,
) (result.Rigs, error) {
	all, err := read(src)
	if err != nil {
		return result.Rigs{}, err
	}

	if len(all.broken) > 0 {
		errs := make([]error, 0, len(all.broken))
		for _, b := range all.broken {
			errs = append(errs, b.err)
		}

		return result.Rigs{}, errors.Join(errs...)
	}

	dir := src.Dir
	if src.User != "" {
		dir = src.User
	}

	return result.Rigs{Dir: dir, Rigs: known(all.merged())}, nil
}

// Find returns the rig with the given identifier, or one of its aliases, and
// the ask beside it.
//
// Both, because a caller that builds from a rig needs both: the gear comes from
// the rig and how it should sound comes from the ask, and asking for them
// separately would read the directory twice and could answer from two states of
// it. The ask is nil where nobody wrote one down, which is ordinary.
func Find(
	src Source,
	id string,
) (result.Known, error) {
	all, err := read(src)
	if err != nil {
		return result.Known{}, err
	}

	found, err := all.find(id)
	if err != nil {
		return result.Known{}, err
	}

	return result.Known{Rig: found.spec, Ask: found.ask}, nil
}

// Show reads one rig, and what the rest of the set says about it.
func Show(
	src Source,
	id string,
) (result.Rig, error) {
	all, err := read(src)
	if err != nil {
		return result.Rig{}, err
	}

	found, err := all.find(id)
	if err != nil {
		return result.Rig{}, err
	}

	return result.Rig{
		Known:    result.Known{Rig: found.spec, Ask: found.ask},
		Variants: departures(all.merged(), found),
	}, nil
}

// departures names the rigs that are a small change on this one.
//
// A player owns several rigs — by era, by song — and they are siblings rather
// than deltas, because two of them can differ at the amp and a delta assumes
// a spine they may not share. A rig that genuinely is a small change says so
// with `extends`, and this is the other end of that link: reading the
// characteristic rig should show what departs from it.
func departures(
	all []stored,
	of stored,
) []result.Variant {
	out := []result.Variant(nil)

	for _, other := range all {
		// Both ends of the link are the ask's. What one ask departs from is a
		// fact about what was wanted, not about the gear that answered it, and a
		// rig with no ask beside it cannot depart from anything because nothing
		// says what it was for.
		if other.ask == nil || other.ask.Extends == nil {
			continue
		}

		// A rig of somebody's own copied from a shipped one under the same
		// identifier extends a rig it has replaced. It is not its own variant.
		if *other.ask.Extends != of.spec.ID || other.spec.ID == of.spec.ID {
			continue
		}

		out = append(out, result.Variant{
			ID:   other.spec.ID,
			Name: subjectOf(other),
		})
	}

	return out
}

// subjectOf is what a rig's ask calls its subject, or its identifier when no ask
// names one.
func subjectOf(
	e stored,
) string {
	if e.ask == nil || e.ask.Subject == nil {
		return e.spec.ID
	}

	return e.ask.Subject.Name
}

// specs are the rigs of a set of entries, in the same order.
func specs(
	all []stored,
) []rig.Spec {
	out := make([]rig.Spec, 0, len(all))
	for _, e := range all {
		out = append(out, e.spec)
	}

	return out
}

// known are the paired rigs and asks of a set of entries, in the same order.
func known(
	all []stored,
) []result.Known {
	out := make([]result.Known, 0, len(all))
	for _, e := range all {
		out = append(out, result.Known{Rig: e.spec, Ask: e.ask})
	}

	return out
}

// sortEntries puts entries in identifier order.
func sortEntries(
	all []stored,
) {
	sort.SliceStable(all, func(i, j int) bool { return all[i].spec.ID < all[j].spec.ID })
}
