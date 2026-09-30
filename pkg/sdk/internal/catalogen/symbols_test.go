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

// SymbolsTestSuite covers the table a device names its models by.
type SymbolsTestSuite struct {
	suite.Suite
}

// write puts a symbol file in a fresh directory and returns it.
func (s *SymbolsTestSuite) write(
	body string,
) string {
	dir := s.T().TempDir()
	s.Require().NoError(
		os.WriteFile(filepath.Join(dir, symbolFile), []byte(body), 0o600))

	return dir
}

// TestReadSymbols reads the table a device names its models by.
func (s *SymbolsTestSuite) TestReadSymbols() {
	tests := []struct {
		name    string
		body    string
		absent  bool
		locked  bool
		wantIDs []string
		params  []string
		errText string
	}{
		{
			name: "a table two models long",
			body: `[
				{ "symbol": "HD2_AmpSVBeastNrm",
				  "parameters": [ "Drive", "Bass", "Mid" ] },
				{ "symbol": "HD2_Cab8x10SVBeast",
				  "parameters": [ "Distance" ] }
			]`,
			// Position is the whole point: a device names a model by where it
			// sits.
			wantIDs: []string{"HD2_AmpSVBeastNrm", "HD2_Cab8x10SVBeast"},
			params:  []string{"Drive", "Bass", "Mid"},
		},
		{
			// A catalog without the table still describes what the device can
			// do. Only reading a preset off the hardware needs it.
			name:   "an installation without one",
			absent: true,
		},
		{
			name:    "a table that will not parse",
			body:    "not json",
			errText: symbolFile,
		},
		{
			name:   "a table it cannot read",
			body:   "[]",
			locked: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			if !tt.absent {
				dir = s.write(tt.body)
			}

			if tt.locked {
				s.Require().NoError(
					os.Chmod(filepath.Join(dir, symbolFile), 0o000))
			}

			got, err := readSymbols(dir)

			if tt.errText != "" || tt.locked {
				s.Require().Error(err)

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got, len(tt.wantIDs))

			for i, want := range tt.wantIDs {
				s.Require().Equal(want, string(got[i].ID))
			}

			if tt.params != nil {
				s.Require().Equal(tt.params, got[0].Params)
			}
		})
	}
}

// TestReadLEDColours reads the colours a device names its switches by.
func (s *SymbolsTestSuite) TestReadLEDColours() {
	tests := []struct {
		name    string
		body    string
		absent  bool
		locked  bool
		want    []string
		errText string
	}{
		{
			name: "the colours themselves",
			body: `{
				"blend": { "format": "%.0f" },
				"footswitchLED": { "isDiscrete": true,
				  "format": ["Auto Color", "White", "Green", "Violet"] }
			}`,
			want: []string{"auto color", "white", "green", "violet"},
		},
		{
			name:   "an installation without them",
			absent: true,
		},
		{
			name:    "a file that will not parse",
			body:    "not json",
			errText: controlsFile,
		},
		{
			name:    "a colour list of the wrong shape",
			body:    `{"footswitchLED": {"format": "%.0f"}}`,
			errText: ledControl,
		},
		{
			name:   "a file it cannot read",
			body:   "{}",
			locked: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			path := filepath.Join(dir, controlsFile)

			if !tt.absent {
				s.Require().NoError(os.WriteFile(path, []byte(tt.body), 0o600))
			}

			if tt.locked {
				s.Require().NoError(os.Chmod(path, 0o000))
			}

			got, err := readLEDColours(dir)

			if tt.errText != "" || tt.locked {
				s.Require().Error(err)

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestBuildReportsAnInstallationItCannotRead covers what Build does with the
// two files this one owns. A catalog is generated from somebody's own
// installation, and a file in it that cannot be read is worth complaining
// about rather than quietly producing a catalog nothing can name a model with.
func (s *SymbolsTestSuite) TestBuildReportsAnInstallationItCannotRead() {
	tests := []struct {
		name     string
		symbols  string
		controls string
		errText  string
	}{
		{
			name:     "colours that will not parse",
			symbols:  "[]",
			controls: "not json",
			errText:  controlsFile,
		},
		{
			name:    "a symbol table that will not parse",
			symbols: "not json",
			errText: symbolFile,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.write(tt.symbols)
			s.Require().NoError(os.WriteFile(
				filepath.Join(dir, "amp.models"), []byte("[]"), 0o600))

			if tt.controls != "" {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, controlsFile), []byte(tt.controls), 0o600))
			}

			_, err := Build(Options{ResourcesDir: dir, DeviceID: 1})

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tt.errText)
		})
	}
}

// TestFamilyOf covers the suffix a device's own lists carry.
//
// Only a Helix LT has one. The three that share the unsuffixed pair are not
// interchangeable with each other in general, and the pair they share lists
// four Returns an HX Stomp has no sockets for, so an entry existing is not the
// same as the socket existing.
func (s *SymbolsTestSuite) TestFamilyOf() {
	s.Require().Equal("_lt", familyOf(catalog.HelixLT))

	for _, device := range []int{
		catalog.HXStomp,
		catalog.HXStompXL,
		catalog.HelixFloor,
	} {
		s.Require().Empty(familyOf(device),
			"device %d shares the unsuffixed pair", device)
	}
}

// controls puts a HelixControls.json in a fresh directory and returns it.
func (s *SymbolsTestSuite) controls(
	body string,
) string {
	dir := s.T().TempDir()
	s.Require().NoError(
		os.WriteFile(filepath.Join(dir, controlsFile), []byte(body), 0o600))

	return dir
}

// TestReadRoutingNamesTheSocketsInTheDevicesOwnOrder covers the two lists a
// preset's routing is written against.
//
// TestReadRouting covers readRouting, which reads what a device can take a
// chain's input from and send its.
//
// One method and one table, so a case is a row rather than a file.
func (s *SymbolsTestSuite) TestReadRouting() {
	for _, tt := range []struct {
		name string
		then func()
	}{
		{
			// label claims USB and which an HX Stomp's Multi does not carry, which is the
			// reason these are read rather than written down.
			name: "read routing names the sockets in the devices own order",
			then: func() {
				dir := s.controls(`{
				  "input_type":  {"format": ["Multi", "Guitar", "Aux"]},
				  "output_type": {"format": ["Multi", "USB 1/2", "USB 3/4"]},
				  "input_type_lt":  {"format": ["Guitar", "Aux"]},
				  "output_type_lt": {"format": ["XLR"]}
				}`)

				sources, destinations, err := readRouting(dir, catalog.HXStomp)

				s.Require().NoError(err)
				s.Require().Equal([]string{"Multi", "Guitar", "Aux"}, sources)
				s.Require().Equal([]string{"Multi", "USB 1/2", "USB 3/4"}, destinations,
					"kept as the device spells them: USB 1/2 loses something flattened")

				// An LT reads its own suffixed pair, which is why the suffix exists.
				sources, destinations, err = readRouting(dir, catalog.HelixLT)

				s.Require().NoError(err)
				s.Require().Equal([]string{"Guitar", "Aux"}, sources)
				s.Require().Equal([]string{"XLR"}, destinations)
			},
		},
		{
			// these lists is still a catalog.
			name: "routing is absent rather than fatal",
			then: func() {
				for _, tt := range []struct {
					name string
					dir  string
				}{
					{"no controls file at all", s.T().TempDir()},
					{"a file naming neither list", s.controls(`{"footswitchLED": {}}`)},
				} {
					s.Run(tt.name, func() {
						sources, destinations, err := readRouting(tt.dir, catalog.HXStomp)

						s.Require().NoError(err)
						s.Require().Nil(sources)
						s.Require().Nil(destinations)
					})
				}
			},
		},
		{
			name: "routing reports a file it cannot read",
			then: func() {
				for _, tt := range []struct {
					name string
					body string
					says string
				}{
					{
						name: "a file that is not JSON",
						body: "not json",
						says: "decoding " + controlsFile,
					},
					{
						name: "a sources entry that is not a control",
						body: `{"input_type": 7}`,
						says: "decoding " + sourceControl,
					},
					{
						name: "a destinations entry that is not a control",
						body: `{"output_type": 7}`,
						says: "decoding " + destinationControl,
					},
				} {
					s.Run(tt.name, func() {
						_, _, err := readRouting(s.controls(tt.body), catalog.HXStomp)

						s.Require().ErrorContains(err, tt.says)
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

// TestRoutingIsAbsentRatherThanFatal covers an installation too old to have it.
//

func TestSymbolsTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SymbolsTestSuite))
}
