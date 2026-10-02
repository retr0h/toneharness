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

package rigs

import (
	"context"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// Source says where rigs are read from.
//
// The zero value is the rigs that ship in the binary.
type Source struct {
	// Dir is a directory read instead of the rigs that ship. Empty is the
	// rigs that ship.
	Dir string
	// User is somebody's own directory, layered over Dir. A rig in it takes
	// the place of one in Dir when the two share an identifier or alias, in
	// any case. A directory that is not there holds nothing, and one that
	// cannot be read is an error. Empty layers nothing.
	User string
}

// stored is one document as read, and the text it was read from. The text is
// kept because a copy keeps the comments and decoding drops them.
//
// One document holding both halves. The rig says what the gear is and why each
// piece of it is believed to be there; the ask says who it is for, how it should
// sound, and how it is played. The ask is a pointer because a rig without one is
// legal and ordinary: somebody's own directory holds rigs they wrote, and nothing
// obliges them to write down the ask that produced one.
type stored struct {
	// doc is the whole document and raw the text it was read from.
	//
	// One of each since version 2, where a rig and the ask beside it were two
	// files paired by filename stem. The text is kept because copying a rig is
	// copying its text: marshalling loses the comments and the comments are most
	// of what a rig carries.
	doc tone.Spec
	raw []byte
}

// ask is the request half, or nil where nobody wrote one.
func (s stored) askOf() *tone.Ask { return s.doc.Ask }

// spec is the gear half, which every document has.
func (s stored) specOf() rig.Spec { return s.doc.Rig }

// id is what the document is called.
func (s stored) idOf() string { return s.doc.Id }

// brokenFile is a file in somebody's own directory that is not a rig.
type brokenFile struct {
	// names are what the file may have been asked for by: its filename stem,
	// and whatever id and aliases it states where those can be read.
	names []string
	err   error
}

// set is every rig a Source holds, read once.
type set struct {
	user   []stored
	base   []stored
	broken []brokenFile
}

// Catalogs hands over the catalog a new rig's gear is checked against. The
// sdk Client satisfies it, and keeps the catalog it opened.
type Catalogs interface {
	// Catalog returns the catalog, opening it on first use.
	Catalog(ctx context.Context) (*catalog.Catalog, error)
}
