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

package asking_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/asking"
	"github.com/retr0h/tonestack/pkg/sdk/measured"
)

// AskingPublicTestSuite covers turning a request into the rig it describes.
//
// Against the real catalog and the real measurements, because choosing an
// amplifier by measuring is the half no reasoning about names can do, and a
// stubbed library would exercise the plumbing rather than the answer.
type AskingPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
	lib measured.Library
}

func (s *AskingPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)

	s.lib, err = measured.BuiltIn()
	s.Require().NoError(err)
}

// at is where the worked examples live, from this package.
func (s *AskingPublicTestSuite) at(
	name string,
) string {
	return filepath.Join("..", "..", "..", "..", "examples", "tonespec", name)
}

// file writes a document and returns where it went.
func (s *AskingPublicTestSuite) file(
	name, body string,
) string {
	to := filepath.Join(s.T().TempDir(), name)
	s.Require().NoError(os.WriteFile(to, []byte(body), 0o600))

	return to
}

// TestTheWorkedExampleResolves holds the examples to the code.
//
// A worked example that stopped working is worse than none: it is the first
// thing anybody runs, and it says the tool is broken when the example is.
func (s *AskingPublicTestSuite) TestTheWorkedExampleResolves() {
	got, err := asking.Resolve(context.Background(), asking.Ask{
		Spec:  s.at("like-a-record.yaml"),
		Setup: s.at("my-setup.yaml"),
	}, s.cat, s.lib)

	s.Require().NoError(err)
	s.Require().NotEmpty(got.Rig.Chain)
	s.Require().NotEmpty(got.Notes, "a resolution says what it made of the ask")
}

// TestASetupIsOptional covers a request that says nothing about what is owned.
//
// Somebody asking what a record sounds like has not necessarily said what is
// in the room, and the answer says what it assumed rather than refusing.
func (s *AskingPublicTestSuite) TestASetupIsOptional() {
	got, err := asking.Resolve(context.Background(), asking.Ask{
		Spec: s.at("like-a-record.yaml"),
	}, s.cat, s.lib)

	s.Require().NoError(err)
	s.Require().NotEmpty(got.Rig.Chain)
}

// TestADocumentThatIsNotThereIsReported covers both paths that read a file.
func (s *AskingPublicTestSuite) TestADocumentThatIsNotThereIsReported() {
	absent := filepath.Join(s.T().TempDir(), "nowhere.yaml")

	tests := []struct {
		name string
		in   asking.Ask
	}{
		{name: "the ask", in: asking.Ask{Spec: absent}},
		{
			name: "the setup",
			in:   asking.Ask{Spec: s.at("like-a-record.yaml"), Setup: absent},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := asking.Resolve(context.Background(), tt.in, s.cat, s.lib)

			s.Require().Error(err)
			s.Require().ErrorContains(err, absent,
				"the error names the file somebody has to go and look at")
		})
	}
}

// TestADocumentThatIsNotOneIsReported covers a file that reads but is not a
// ToneSpec.
func (s *AskingPublicTestSuite) TestADocumentThatIsNotOneIsReported() {
	tests := []struct {
		name string
		in   asking.Ask
	}{
		{
			name: "an ask that is not one",
			in:   asking.Ask{Spec: s.file("ask.yaml", "schema: Nonsense\n")},
		},
		{
			name: "a setup that is not one",
			in: asking.Ask{
				Spec:  s.at("like-a-record.yaml"),
				Setup: s.file("setup.yaml", "schema: Nonsense\n"),
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := asking.Resolve(context.Background(), tt.in, s.cat, s.lib)

			s.Require().Error(err)
		})
	}
}

// TestNotesTravelWithAFailure is why both halves come back.
//
// A request that could not be honoured has usually said why in the notes, and
// the error on its own is the half that does not help.
func (s *AskingPublicTestSuite) TestNotesTravelWithAFailure() {
	got, err := asking.Resolve(context.Background(), asking.Ask{
		Spec: s.file("insisted.yaml", `schema: ToneSpec
gear:
  - gear: Ampeg B-15
    role: amp
    insist: true
`),
	}, s.cat, s.lib)

	s.Require().Error(err, "a request that insisted on gear the device has not")
	s.Require().NotEmpty(got.Notes, "and the notes say which gear, and why")
}

// TestACancelledContextIsRefused covers the guard every operation carries.
func (s *AskingPublicTestSuite) TestACancelledContextIsRefused() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := asking.Resolve(ctx, asking.Ask{
		Spec: s.at("like-a-record.yaml"),
	}, s.cat, s.lib)

	s.Require().ErrorIs(err, context.Canceled)
}

func TestAskingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AskingPublicTestSuite))
}
