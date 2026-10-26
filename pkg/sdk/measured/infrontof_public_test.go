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
package measured_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// InFrontOfPublicTestSuite covers reading a library of blocks measured with
// something behind them.
//
// The refusals are the point. A pair reading and an isolated one describe
// different things, and handing one over as the other is what `Library.Rig`
// and `Library.Isolated` exist to prevent.
type InFrontOfPublicTestSuite struct {
	suite.Suite

	dir string
}

func (s *InFrontOfPublicTestSuite) SetupTest() {
	s.dir = s.T().TempDir()
}

// wrote puts a library on disk under the name the lookup asks for.
func (s *InFrontOfPublicTestSuite) wrote(
	after string,
	body string,
) {
	at := filepath.Join(s.dir, "fingerprints-into-"+after+".json")
	s.Require().NoError(os.WriteFile(at, []byte(body), 0o600))
}

// TestInFrontOf covers which files answer and which are refused.
//
// One method and one table, so a case is a row rather than a file.
func (s *InFrontOfPublicTestSuite) TestInFrontOf() {
	const pair = `{"device":"HX Stomp","isolated":false,` +
		`"after":"HD2_AmpSVBeastBrt","baseline":{"level":-51.03},` +
		`"blocks":{"HD2_DistObsidian7000":{"id":"HD2_DistObsidian7000",` +
		`"category":"drive","centroid":104,"level":-40}}}`

	for _, tt := range []struct {
		name string
		// write is the file put on disk, or nothing for an absent one.
		write string
		// as is the model the file is named for.
		as string
		// ask is the model the lookup is given.
		ask string
		// found is whether a usable pair library came back.
		found bool
	}{
		{
			// The case this exists for.
			name:  "a pair reading for the amplifier asked about",
			write: pair,
			as:    "HD2_AmpSVBeastBrt",
			ask:   "HD2_AmpSVBeastBrt",
			found: true,
		},
		{
			// Nothing measured behind that amplifier, which is every
			// amplifier but one today. The caller falls back to the isolated
			// ranking, so this is a thing to do without rather than an error.
			name:  "an amplifier with no readings behind it",
			write: pair,
			as:    "HD2_AmpSVBeastBrt",
			ask:   "HD2_AmpTucknGo",
		},
		{
			name: "no file at all",
			ask:  "HD2_AmpSVBeastBrt",
		},
		{
			// A library saying it holds isolated readings is what the
			// caller's own `Measured` already is, and ranking the same
			// figures twice is not a chain reading.
			// Carrying a block, because a library naming none is refused by
			// `Load` before this is reached and the case would pass for the
			// wrong reason.
			name: "a library claiming isolated readings",
			write: `{"device":"HX Stomp","isolated":true,` +
				`"after":"HD2_AmpSVBeastBrt","baseline":{"level":-51.03},` +
				`"blocks":{"HD2_DistObsidian7000":{"id":"HD2_DistObsidian7000",` +
				`"category":"drive","centroid":104,"level":-40}}}`,
			as:  "HD2_AmpSVBeastBrt",
			ask: "HD2_AmpSVBeastBrt",
		},
		{
			// In a chain, but it cannot say which chain, so it cannot be
			// matched to the amplifier a rig holds.
			name: "a library that does not say what sat behind it",
			write: `{"device":"HX Stomp","isolated":false,` +
				`"baseline":{"level":-51.03},` +
				`"blocks":{"HD2_DistObsidian7000":{"id":"HD2_DistObsidian7000",` +
				`"category":"drive","centroid":104,"level":-40}}}`,
			as:  "HD2_AmpSVBeastBrt",
			ask: "HD2_AmpSVBeastBrt",
		},
		{
			name:  "a file that will not parse",
			write: `not a library`,
			as:    "HD2_AmpSVBeastBrt",
			ask:   "HD2_AmpSVBeastBrt",
		},
		{
			// Parses and names nothing, which `Load` refuses on its own. Here
			// because the two refusals above were written without a block and
			// passed on this instead of on the thing they claim to test.
			name: "a library naming no blocks",
			write: `{"device":"HX Stomp","isolated":false,` +
				`"after":"HD2_AmpSVBeastBrt","baseline":{"level":-51.03},` +
				`"blocks":{}}`,
			as:  "HD2_AmpSVBeastBrt",
			ask: "HD2_AmpSVBeastBrt",
		},
	} {
		s.Run(tt.name, func() {
			s.SetupTest()

			if tt.write != "" {
				s.wrote(tt.as, tt.write)
			}

			got, ok := measured.InFrontOf(s.dir)(catalog.ModelID(tt.ask))

			s.Require().Equal(tt.found, ok)

			if !tt.found {
				return
			}

			s.Require().False(got.Isolated)
			s.Require().Equal("HD2_AmpSVBeastBrt", got.After)
			s.Require().NotEmpty(got.Blocks)
		})
	}
}

// TestInFrontOfWithoutADirectory covers a release, which ships the isolated
// library and no pair readings.
//
// Nil rather than a lookup answering no, so a caller can tell "nothing to
// look in" from "looked and found nothing".
func (s *InFrontOfPublicTestSuite) TestInFrontOfWithoutADirectory() {
	s.Require().Nil(measured.InFrontOf(""))
}

func TestInFrontOfPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InFrontOfPublicTestSuite))
}
