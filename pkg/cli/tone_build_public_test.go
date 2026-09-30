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
package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/toneharness/pkg/cli"
	"github.com/retr0h/toneharness/pkg/cli/internal/mocks"
	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/rig"
	"github.com/retr0h/toneharness/pkg/sdk/translate"
)

// ToneBuildPublicTestSuite covers reading a request and writing the rig it
// resolves to.
type ToneBuildPublicTestSuite struct {
	suite.Suite
}

// SetupTest runs each case from the checkout root.
//
// That is where somebody runs a worked example, and an example carrying a
// path only this test can resolve is an example that does not work.
func (s *ToneBuildPublicTestSuite) SetupTest() {
	s.T().Chdir(filepath.Join("..", ".."))
}

// file writes a document and returns where it went.
func (s *ToneBuildPublicTestSuite) file(
	name, body string,
) string {
	at := filepath.Join(s.T().TempDir(), name)
	s.Require().NoError(os.WriteFile(at, []byte(body), 0o600))

	return at
}

// examples is where the worked requests live.
func (s *ToneBuildPublicTestSuite) examples(
	name string,
) string {
	return filepath.Join("examples", "tonespec", name)
}

// TestTheWorkedExampleBuildsARig holds the examples to the code.
//
// A worked example that stopped working is worse than none: it is the first
// thing somebody runs, and it says the tool is broken when the example is.
func (s *ToneBuildPublicTestSuite) TestTheWorkedExampleBuildsARig() {
	out := filepath.Join(s.T().TempDir(), "rig.yaml")

	var buf bytes.Buffer

	s.Require().NoError(cli.ToneBuild(&buf, cli.ToneBuildOptions{
		Ask:   s.examples("like-a-record.yaml"),
		Setup: s.examples("my-setup.yaml"),
		Out:   out,
	}))

	body, err := os.ReadFile(out)
	s.Require().NoError(err)

	spec, err := rig.Load(bytes.NewReader(body))
	s.Require().NoError(err)
	s.Require().NoError(rig.Validate(spec))
	s.Require().Equal(rig.InstrumentBass, spec.Instrument)

	// The comp it named, and an amplifier nobody named.
	s.Require().Len(spec.Chain, 2)
	s.Require().Equal(rig.RoleAmp, spec.Chain[1].Role)

	s.Require().Contains(buf.String(), "closest of 224 measured")
}

// TestWithoutASetupItSaysWhatItAssumed covers the optional half.
//
// Somebody asking what a record sounds like has not necessarily said what
// they own, so the rig says what it assumed rather than refusing.
func (s *ToneBuildPublicTestSuite) TestWithoutASetupItSaysWhatItAssumed() {
	var buf bytes.Buffer

	s.Require().NoError(cli.ToneBuild(&buf, cli.ToneBuildOptions{
		Ask: s.examples("like-a-record.yaml"),
	}))

	s.Require().Contains(buf.String(), "the setup names none")
	s.Require().Contains(buf.String(), "schema: RigSpec",
		"the rig goes to whatever is reading when no file was named")
}

// TestItReportsWhatItCouldNotHonour covers the half of an ask it drops.
func (s *ToneBuildPublicTestSuite) TestItReportsWhatItCouldNotHonour() {
	var buf bytes.Buffer

	s.Require().NoError(cli.ToneBuild(&buf, cli.ToneBuildOptions{
		Ask: s.file("ask.yaml", `schema: ToneSpec
genre: [punk]
gear:
  - gear: LA Studio Comp
    role: comp
`),
	}))

	s.Require().Contains(buf.String(), "could not")

	// What punk measured as, rather than the blanket "nobody has tagged
	// anything" this said of every genre before any were measured.
	//
	// It says a word now rather than "nothing to aim at": punk earned `clean`
	// once Alkaline Trio's records joined it, which is what a genre gaining a
	// player does. The assertion is that the answer names what was measured,
	// not which word it happened to be.
	s.Require().Contains(buf.String(), "records from")
	s.Require().Contains(buf.String(), "measured across")
}

// TestARequestItCannotAnswerFails covers an ask with nothing to build from.
func (s *ToneBuildPublicTestSuite) TestARequestItCannotAnswerFails() {
	var buf bytes.Buffer

	err := cli.ToneBuild(&buf, cli.ToneBuildOptions{
		Ask: s.file("ask.yaml",
			"schema: ToneSpec\ngenre: [rock]\nwords:\n  - term: dark\n"),
	})

	s.Require().ErrorContains(err, "no chain to build")
}

// TestARequestItCannotAnswerStillAnswersAsData covers --json on a failure.
//
// The notes are written whether the build succeeded or not, and the error is
// still returned, because what could not be honoured is the useful half either
// way. An agent reading the data surface gets the reasons rather than an exit
// code and nothing.
func (s *ToneBuildPublicTestSuite) TestARequestItCannotAnswerStillAnswersAsData() {
	var buf bytes.Buffer

	err := cli.ToneBuild(&buf, cli.ToneBuildOptions{
		Ask: s.file("ask.yaml",
			"schema: ToneSpec\ngenre: [rock]\nwords:\n  - term: dark\n"),
		AsData: true,
	})

	s.Require().ErrorContains(err, "no chain to build")
	s.Require().NotEmpty(buf.String(), "and the notes still arrive")
	s.Require().Contains(buf.String(), "{", "as data rather than as a table")
}

// TestADocumentItCannotReadIsReported covers both files.
func (s *ToneBuildPublicTestSuite) TestADocumentItCannotReadIsReported() {
	tests := []struct {
		name string
		opts cli.ToneBuildOptions
		want string
	}{
		{
			name: "an ask that is not there",
			opts: cli.ToneBuildOptions{Ask: "nowhere.yaml"},
			want: "nowhere.yaml",
		},
		{
			name: "an ask that is not a ToneSpec",
			opts: cli.ToneBuildOptions{
				Ask: s.file("ask.yaml", "schema: Setup\n"),
			},
			want: "this is not a ToneSpec",
		},
		{
			name: "a setup that is not there",
			opts: cli.ToneBuildOptions{
				Ask:   s.examples("like-a-record.yaml"),
				Setup: "nowhere.yaml",
			},
			want: "nowhere.yaml",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			s.Require().ErrorContains(cli.ToneBuild(&buf, tt.opts), tt.want)
		})
	}
}

// TestSomewhereItCannotWriteIsReported covers a bad output path.
func (s *ToneBuildPublicTestSuite) TestSomewhereItCannotWriteIsReported() {
	var buf bytes.Buffer

	err := cli.ToneBuild(&buf, cli.ToneBuildOptions{
		Ask: s.examples("like-a-record.yaml"),
		Out: filepath.Join(s.T().TempDir(), "no", "such", "rig.yaml"),
	})

	s.Require().Error(err)
}

// TestAStandInResolvesInsteadOfTheRealThing covers the seam.
//
// Declared here rather than taking *sdk.Client, so a test for how an answer
// looks does not need a catalog and six hundred measurements to produce one.
// And optional, the way every collaborator in this repository is: the cases
// above hand in nothing and reach the real one.
func (s *ToneBuildPublicTestSuite) TestAStandInResolvesInsteadOfTheRealThing() {
	ctrl := gomock.NewController(s.T())
	stub := mocks.NewMockResolver(ctrl)

	stub.EXPECT().
		Tone(gomock.Any(), sdk.Ask{Spec: "ask.yaml", Setup: "mine.yaml"}).
		Return(sdk.Resolved{
			Rig: rig.Spec{
				Schema: rig.SchemaName, ID: "stubbed",
				Instrument: rig.InstrumentBass,
				Chain:      []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "Ampeg SVT"}},
			},
			Notes: translate.Notes{{About: "amp", Said: "handed over by a test"}},
		}, nil)

	var buf bytes.Buffer

	s.Require().NoError(cli.ToneBuild(&buf, cli.ToneBuildOptions{
		Client: stub, Ask: "ask.yaml", Setup: "mine.yaml",
	}))

	s.Require().Contains(buf.String(), "handed over by a test",
		"the notes are reported, not only the rig")
	s.Require().Contains(buf.String(), "Ampeg SVT")
}

// TestAStandInsFailureIsReportedWithItsNotes covers the other half.
//
// A request that could not be honoured has usually said why in the notes, so
// they are printed before the error is returned.
func (s *ToneBuildPublicTestSuite) TestAStandInsFailureIsReportedWithItsNotes() {
	ctrl := gomock.NewController(s.T())
	stub := mocks.NewMockResolver(ctrl)

	stub.EXPECT().
		Tone(gomock.Any(), gomock.Any()).
		Return(sdk.Resolved{
			Notes: translate.Notes{{About: "Ampeg B-15", Said: "no such model"}},
		}, translate.ErrInsisted)

	var buf bytes.Buffer

	err := cli.ToneBuild(&buf, cli.ToneBuildOptions{Client: stub, Ask: "ask.yaml"})

	s.Require().ErrorIs(err, translate.ErrInsisted)
	s.Require().Contains(buf.String(), "no such model")
}

func TestToneBuildPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ToneBuildPublicTestSuite))
}
