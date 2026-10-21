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

package catalogen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// ShownTestSuite covers the name Line 6's own app shows, against the one
// beside the model definition.
type ShownTestSuite struct {
	suite.Suite
}

// shows writes a resources directory holding one .models file and one app
// catalog, and answers with its path.
func (s *ShownTestSuite) shows(
	models string,
	app string,
) string {
	dir := s.T().TempDir()

	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "cab.models"), []byte(models), 0o600))

	if app != "" {
		s.Require().NoError(os.WriteFile(
			filepath.Join(dir, shownFile), []byte(app), 0o600))
	}

	return dir
}

// TestReadShownNames covers reading the app's names.
//
// One method and one table, so a case is a row rather than a file.
func (s *ShownTestSuite) TestReadShownNames() {
	for _, tt := range []struct {
		name string
		app  string
		// dir writes the app catalog as a directory, which fails the read
		// without the file being missing.
		dir bool
		// times is how many reads the row asserts over, for a row about an
		// answer being the same one twice. Zero means once.
		times int
		then  func(got map[string]string, err error)
	}{
		{
			// The nesting is Line 6's to change, so this looks for an object
			// with an id and a name wherever it sits rather than decoding a
			// shape.
			name: "a name nested under categories under devices",
			app: `{"devices":[{"categories":[
				{"models":[{"id":"HD2_CabTest","name":"Test Cab"}]}]}]}`,
			then: func(got map[string]string, err error) {
				s.Require().NoError(err)
				s.Require().Equal("Test Cab", got["HD2_CabTest"])
			},
		},
		{
			// The first wins. Within an array that is the one nearest the top
			// of the file; across an object's keys it is the first key in
			// order, because Go's decoder throws an object's own order away
			// and there is nothing else to be first by.
			name: "a model named twice",
			app: `{"a":[{"id":"HD2_CabTest","name":"First"}],
			       "b":[{"id":"HD2_CabTest","name":"Second"}]}`,
			then: func(got map[string]string, err error) {
				s.Require().NoError(err)
				s.Require().Equal("First", got["HD2_CabTest"])
			},
		},
		{
			// And it is the same answer every time.
			//
			// This row is the one that matters, because the row above passed
			// for as long as the walk was random: map iteration order is
			// randomised per range statement, so a coin flip comes up the same
			// way often enough to look like a rule. It failed in continuous
			// integration and nowhere else.
			//
			// Twenty reads rather than one. Each is an independent flip, so an
			// unsorted walk survives this with a probability of about one in a
			// million.
			name: "and the name a duplicate gets does not move between reads",
			app: `{"a":[{"id":"HD2_CabTest","name":"First"}],
			       "b":[{"id":"HD2_CabTest","name":"Second"}],
			       "c":[{"id":"HD2_CabTest","name":"Third"}],
			       "d":[{"id":"HD2_CabTest","name":"Fourth"}]}`,
			then: func(got map[string]string, err error) {
				s.Require().NoError(err)
				s.Require().Equal("First", got["HD2_CabTest"])
			},
			times: 20,
		},
		{
			// An installation too old to carry the file. Not fatal: the
			// .models names are what this used to ship, and they are wrong in
			// four places rather than useless.
			name: "no app catalog at all",
			app:  "",
			then: func(got map[string]string, err error) {
				s.Require().NoError(err)
				s.Require().Empty(got)
			},
		},
		{
			// Absent is not the same as unreadable. A directory where the file
			// should be fails the read without being missing, and that is a
			// broken installation rather than an old one, so it is said.
			name: "an app catalog that cannot be read",
			dir:  true,
			then: func(_ map[string]string, err error) {
				s.Require().ErrorContains(err, shownFile)
			},
		},
		{
			name: "an app catalog that is not JSON",
			app:  "{",
			then: func(_ map[string]string, err error) {
				s.Require().ErrorContains(err, shownFile)
			},
		},
		{
			// Half an entry is no entry. A model with no name cannot rename
			// anything, and a name with no model has nothing to rename.
			name: "entries missing one half",
			app: `{"models":[{"id":"HD2_CabTest"},{"name":"Nameless"},
			       {"id":"","name":"Blank"},{"id":"HD2_CabOther","name":""}]}`,
			then: func(got map[string]string, err error) {
				s.Require().NoError(err)
				s.Require().Empty(got)
			},
		},
	} {
		s.Run(tt.name, func() {
			dir := s.shows(`[{"symbolicID":"HD2_CabTest","name":"Beside"}]`, tt.app)

			if tt.dir {
				s.Require().NoError(os.Mkdir(filepath.Join(dir, shownFile), 0o750))
			}

			for range max(tt.times, 1) {
				got, err := readShown(dir)

				as := map[string]string{}
				for k, v := range got {
					as[string(k)] = v.Name
				}

				tt.then(as, err)
			}
		})
	}
}

// TestBuildTakesTheAppsNameOnlyWhereTheOtherIsWrong covers which
// disagreements move a name and which only get reported.
//
// One method and one table, so a case is a row rather than a file.
func (s *ShownTestSuite) TestBuildTakesTheAppsNameOnlyWhereTheOtherIsWrong() {
	for _, tt := range []struct {
		name  string
		model string
		app   string
		want  string
		took  bool
	}{
		{
			// The cabinet named after a different cabinet, which is the
			// reason any of this exists. A G25 and an H30 are different
			// speakers, and `2x12 Match H30` reached the G25 first because
			// the identifier broke the tie.
			name:  "a name that says the wrong gear is taken",
			model: "HD2_CabMicIr_2x12MatchG25",
			app:   "2x12 Match G25",
			want:  "2x12 Match G25",
			took:  true,
		},
		{
			// Looks cosmetic and is not. Taking every disagreement renamed
			// the SVT-4 Pro to "Ampeg SVT-4 PRO", and `Ampeg SVT` then fit
			// six of the device's models where it had fit four, because the
			// longer name begins with the shorter one.
			name:  "a name that is only spelled differently is kept",
			model: "HD2_AmpSVT4Pro",
			app:   "Ampeg SVT-4 PRO",
			want:  "Beside",
			took:  false,
		},
	} {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			s.Require().NoError(os.WriteFile(
				filepath.Join(dir, "cab.models"),
				[]byte(`[{"symbolicID":"`+tt.model+`","name":"Beside"}]`), 0o600))
			s.Require().NoError(os.WriteFile(
				filepath.Join(dir, shownFile),
				[]byte(`{"models":[{"id":"`+tt.model+`","name":"`+tt.app+`"}]}`), 0o600))

			got, renamed, err := Build(Options{
				ResourcesDir: dir, DeviceID: 2162694, DeviceName: "HX Stomp",
			})

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Blocks[catalog.ModelID(tt.model)].Name)

			s.Require().Len(renamed, 1,
				"every disagreement is reported whether or not it moved a name")
			s.Require().Equal(tt.took, renamed[0].Took)
			s.Require().Equal("Beside", renamed[0].Was)
			s.Require().Equal(tt.app, renamed[0].Now)
		})
	}
}

// TestBuildRefusesAnAppCatalogItCannotRead covers the read failing rather
// than the file being absent.
//
// Absent is an installation too old to carry it and is not fatal. Unreadable
// is a broken installation, and building a catalog from the names that are
// left would ship four wrong ones without saying so.
func (s *ShownTestSuite) TestBuildRefusesAnAppCatalogItCannotRead() {
	dir := s.T().TempDir()

	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "cab.models"),
		[]byte(`[{"symbolicID":"HD2_CabTest","name":"Beside"}]`), 0o600))
	s.Require().NoError(os.Mkdir(filepath.Join(dir, shownFile), 0o750))

	_, _, err := Build(Options{
		ResourcesDir: dir, DeviceID: 2162694, DeviceName: "HX Stomp",
	})

	s.Require().ErrorContains(err, shownFile)
}

// TestBuildReportsRenamesInOneOrder covers the order they are reported in.
//
// Sorted on the identifier, so two runs of one release report the same list.
// A list that shuffled would read as the data having changed.
func (s *ShownTestSuite) TestBuildReportsRenamesInOneOrder() {
	dir := s.T().TempDir()

	// Written with the later identifier first, so the sort has something to do.
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "cab.models"), []byte(`[
			{"symbolicID":"HD2_PreampAguaSledge","name":"Aqua Sledge"},
			{"symbolicID":"HD2_AmpLine6Kinetic","name":"Line 6 Kinectic"}
		]`), 0o600))
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, shownFile), []byte(`{"models":[
			{"id":"HD2_PreampAguaSledge","name":"Agua Sledge"},
			{"id":"HD2_AmpLine6Kinetic","name":"Line 6 Kinetic"}
		]}`), 0o600))

	_, renamed, err := Build(Options{
		ResourcesDir: dir, DeviceID: 2162694, DeviceName: "HX Stomp",
	})

	s.Require().NoError(err)
	s.Require().Len(renamed, 2)
	s.Require().Equal(catalog.ModelID("HD2_AmpLine6Kinetic"), renamed[0].Model)
	s.Require().Equal(catalog.ModelID("HD2_PreampAguaSledge"), renamed[1].Model)
}

func TestShownTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShownTestSuite))
}
