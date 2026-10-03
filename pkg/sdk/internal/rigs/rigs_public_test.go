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
package rigs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/internal/rigs"
	"github.com/retr0h/toneharness/pkg/sdk/result"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

type RigsPublicTestSuite struct {
	suite.Suite
}

func (s *RigsPublicTestSuite) good() string  { return "testdata-good" }
func (s *RigsPublicTestSuite) mixed() string { return "testdata" }

// TestLoad reads a directory of rigs.
func (s *RigsPublicTestSuite) TestLoad() {
	locked := s.T().TempDir()
	s.Require().NoError(os.MkdirAll(filepath.Join(locked, "artists"), 0o750))

	path := filepath.Join(locked, "artists", "locked.yaml")
	s.Require().NoError(os.WriteFile(path, []byte("id: locked"), 0o600))
	s.Require().NoError(os.Chmod(path, 0o000))

	unreadable := s.T().TempDir()
	s.Require().NoError(os.Chmod(unreadable, 0o000))
	// Put back so the directory can be removed. Runs before TempDir's own
	// cleanup, which was registered first; a failure shows up there.
	s.T().Cleanup(func() { _ = os.Chmod(unreadable, 0o750) })

	// Two files under different names, each stating the same identifier. The
	// filename is not the identifier, so nothing stops it, and a rig that
	// silently does not load is what the broken list exists to prevent.
	both := s.T().TempDir()
	s.Require().NoError(os.MkdirAll(filepath.Join(both, "artists"), 0o750))

	for _, name := range []string{"one.yaml", "two.yaml"} {
		s.Require().NoError(os.WriteFile(
			filepath.Join(both, "artists", name),
			[]byte(document("twice")), 0o600))
	}

	tests := []struct {
		name  string
		dir   string
		ids   []string
		empty bool
		err   string
		// unprivileged is a row root would pass, since root reads anything.
		unprivileged bool
	}{
		{
			name: "every rig in the directory, sorted",
			dir:  s.good(),
			ids:  []string{"mike-dirnt", "mike-dirnt-longview", "minimal"},
		},
		{
			// A half-read knowledge base is worse than a complaint about the
			// file to fix, so one bad rig fails the whole load.
			name: "one that will not parse stops the load",
			dir:  s.mixed(),
			err:  "broken.yaml",
		},
		{
			name: "one that cannot be opened",
			dir:  locked,
			err:  "opening",
		},
		{
			name:  "a directory holding none",
			dir:   s.T().TempDir(),
			empty: true,
		},
		{
			// Nobody has written a rig of their own yet.
			name:  "a directory that is not there holds none",
			dir:   filepath.Join(s.T().TempDir(), "missing"),
			empty: true,
		},
		{
			// Not the same as holding none: reporting it empty would hide
			// every rig in it without saying why.
			name:         "a directory that cannot be read",
			dir:          unreadable,
			err:          "reading " + unreadable,
			unprivileged: true,
		},
		{
			// No directory is the case for anyone running an installed
			// binary rather than working in a checkout.
			name: "no directory falls back to the built-in rigs",
			dir:  "",
		},
		{
			// Two files claiming one name. Reported rather than dropped: a rig
			// that silently does not load is the failure the broken list exists
			// to prevent, and which of the two was read is the useful half.
			name: "one name claimed by two files",
			dir:  both,
			err:  "both say they are",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.unprivileged && os.Geteuid() == 0 {
				s.T().Skip("root reads a directory whatever its mode")
			}

			all, err := rigs.Load(tt.dir)

			if tt.err != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.err)

				return
			}

			s.Require().NoError(err)

			if tt.empty {
				s.Require().Empty(all)

				return
			}

			if tt.ids == nil {
				s.Require().NotEmpty(all, "rigs ship in the binary")

				return
			}

			s.Require().Equal(tt.ids, specIDs(all))
		})
	}
}

// TestFind looks one rig up.
func (s *RigsPublicTestSuite) TestFind() {
	tests := []struct {
		name string
		dir  string
		id   string
		want string
		// paired says the answer must carry the ask the gear came with.
		paired bool
		errs   []string
	}{
		{
			name:   "by identifier",
			dir:    s.good(),
			id:     "mike-dirnt",
			want:   "mike-dirnt",
			paired: true,
		},
		{
			name: "whatever case somebody typed",
			dir:  s.good(),
			id:   "MIKE-DIRNT",
			want: "mike-dirnt",
		},
		{
			name: "by an alias the rig claims",
			dir:  s.good(),
			id:   "spare",
			want: "minimal",
		},
		{
			name: "one nobody wrote",
			dir:  s.good(),
			id:   "nobody",
			errs: []string{"no such rig", "nobody"},
		},
		{
			name: "a directory that will not load",
			dir:  s.mixed(),
			id:   "mike-dirnt",
			errs: []string{"broken.yaml"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := rigs.Find(rigs.Source{Dir: tt.dir}, tt.id)

			if tt.errs != nil {
				s.Require().Error(err)

				for _, want := range tt.errs {
					s.Require().Contains(err.Error(), want)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.ID)

			if tt.paired {
				s.Require().NotNil(got.Ask,
					"one read answers with the gear and the ask it came with")
			}
		})
	}
}

// TestList covers List, which reads every rig a Source holds.
//
// One method and one table, so a case is a row rather than a file.
func (s *RigsPublicTestSuite) TestList() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// Writes out what is on the shelf.
			name: "list",
			then: func() {
				tests := []struct {
					name string
					dir  string
					// the identifiers the answer must carry, in any order.
					ids []string
					err bool
				}{
					{
						name: "one entry per rig",
						dir:  s.good(),
						ids:  []string{"mike-dirnt"},
					},
					{
						// A shelf with nothing on it is not a failure. Somebody who
						// just made the directory is owed an empty answer.
						name: "a shelf with nothing on it",
						dir:  s.T().TempDir(),
					},
					{
						name: "a directory that will not load",
						dir:  s.mixed(),
						err:  true,
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						all, err := rigs.List(rigs.Source{Dir: tt.dir})

						if tt.err {
							s.Require().Error(err)

							return
						}

						s.Require().NoError(err)
						s.Require().Equal(tt.dir, all.Dir)

						got := ids(all.Rigs)

						for _, want := range tt.ids {
							s.Require().Contains(got, want)
						}

						if tt.ids == nil {
							s.Require().Empty(all.Rigs)
						}
					})
				}
			},
		},
		{
			// The ask half of a document that is wrong.
			//
			// Reported the same way a chain that will not parse is, rather than
			// loading the gear without it. A file somebody wrote and got wrong
			// is the case where saying so matters, and a rig quietly missing the
			// words it was built from is the bug the ask exists to remove.
			name: "an ask that will not load is reported",
			then: func() {
				tests := []struct {
					name string
					ask  string
					// unreadable takes the mode off the file instead of writing nonsense.
					unreadable bool
					err        string
				}{
					{
						name: "an ask claiming a field the contract refuses",
						ask:  "ask:\n  chian: []\n",
						err:  `"chian" is unsupported`,
					},
					{
						name: "an ask holding no genre",
						ask:  "ask:\n  confidence: low\n",
						err:  "genre",
					},
					{
						name:       "a document nobody may open",
						ask:        "",
						unreadable: true,
						err:        "opening",
					},
				}

				for _, tt := range tests {
					s.Run(tt.name, func() {
						if tt.unreadable && os.Geteuid() == 0 {
							s.T().Skip("root reads a file whatever its mode")
						}

						dir := s.T().TempDir()
						artists := filepath.Join(dir, "artists")
						s.Require().NoError(os.MkdirAll(artists, 0o750))

						at := filepath.Join(artists, "theirs.yaml")
						s.Require().NoError(os.WriteFile(
							at, []byte(document("theirs")+tt.ask), 0o600))

						if tt.unreadable {
							s.Require().NoError(os.Chmod(at, 0o000))
						}

						_, err := rigs.List(rigs.Source{Dir: dir})
						s.Require().Error(err)
						s.Require().Contains(err.Error(), tt.err)
						// Named by the file it is in, which is what somebody has to open.
						s.Require().Contains(err.Error(), "theirs")
					})
				}
			},
		},
	} {
		s.Run(tt.name, func() {
			tt.then()
		})
	}
}

// TestShow writes out one rig.
func (s *RigsPublicTestSuite) TestShow() {
	tests := []struct {
		name string
		dir  string
		id   string
		// the rigs that say they are a small change on this one.
		variants []string
		err      bool
		is       error
	}{
		{
			// The link points the other way — a variant names what it
			// extends — so only reading the whole set answers this.
			name:     "a rig something else departs from",
			dir:      s.good(),
			id:       "mike-dirnt",
			variants: []string{"mike-dirnt-longview"},
		},
		{
			name: "a rig nothing departs from",
			dir:  s.good(),
			id:   "minimal",
		},
		{
			// Matched with errors.Is, so a caller can tell "no such rig"
			// from "the shelf would not open" without reading the message.
			name: "a rig nobody wrote",
			dir:  s.good(),
			id:   "nobody",
			err:  true,
			is:   rigs.ErrNotFound,
		},
		{
			name: "a directory that will not load",
			dir:  s.mixed(),
			id:   "mike-dirnt",
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			one, err := rigs.Show(rigs.Source{Dir: tt.dir}, tt.id)

			if tt.err {
				s.Require().Error(err)

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, one.ID)

			got := make([]string, 0, len(one.Variants))
			for _, v := range one.Variants {
				got = append(got, v.ID)
			}

			s.Require().ElementsMatch(tt.variants, got)
		})
	}
}

// document is the smallest document the contract takes, under the given
// identifier: the gear, which is required, and no ask, which is optional.
//
// A row that needs a broken ask appends one.
func document(
	id string,
) string {
	return "schema: ToneSpec\nid: " + id + "\nrig:\n  instrument: bass\n" +
		"  chain:\n    - {role: amp, gear: Ampeg SVT}\n"
}

// ids reads the identifiers out of a set of rigs, so a test can say which
// were found without also saying what else each one holds.
func ids(
	all []result.Known,
) []string {
	out := make([]string, 0, len(all))
	for _, r := range all {
		out = append(out, r.ID)
	}

	return out
}

// specIDs is the same for the documents Load answers with.
func specIDs(
	all []tone.Spec,
) []string {
	out := make([]string, 0, len(all))
	for _, r := range all {
		out = append(out, r.Id)
	}

	return out
}

func TestRigsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RigsPublicTestSuite))
}
