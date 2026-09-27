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

package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

type TypesPublicTestSuite struct {
	suite.Suite
}

func (s *TypesPublicTestSuite) TestTrustedAcceptsEverythingButAssumed() {
	for _, p := range []catalog.Provenance{
		catalog.ProvOfficial,
		catalog.ProvMeasured,
		catalog.ProvObserved,
		catalog.ProvInherited,
	} {
		s.Require().True(p.Trusted(), "%s should be trusted", p)
	}
}

func (s *TypesPublicTestSuite) TestTrustedRejectsAssumed() {
	s.Require().False(catalog.ProvAssumed.Trusted())
}

func (s *TypesPublicTestSuite) TestProvenanceValuesAreDistinct() {
	all := []catalog.Provenance{
		catalog.ProvOfficial,
		catalog.ProvMeasured,
		catalog.ProvObserved,
		catalog.ProvInherited,
		catalog.ProvAssumed,
	}

	seen := make(map[catalog.Provenance]bool, len(all))
	for _, p := range all {
		s.Require().False(seen[p], "duplicate provenance %q", p)
		seen[p] = true
	}
}

// TestSourceAtAndDestinationAt covers finding a routing position by name.
//
// By name because the number belongs to the device family and the name does
// not: "USB 5/6" means the same thing on every Helix and is a different index
// on some of them, so a hardcoded number is right on one pedal and silently
// wrong on the next.
func (s *TypesPublicTestSuite) TestSourceAtAndDestinationAt() {
	cat := &catalog.Catalog{
		Sources: []string{"Multi (Guitar, Aux, Variax)", "Guitar", "Aux", "USB 1/2"},
		Destinations: []string{
			`Multi (1/4", XLR, Digital, USB 1/2)`, `1/4"`, "XLR", "USB 1/2", "USB 5/6",
		},
	}

	s.Run("a source by its own spelling", func() {
		at, ok := cat.SourceAt("Guitar")
		s.Require().True(ok)
		s.Require().Equal(1, at)
	})

	s.Run("a destination by its own spelling", func() {
		at, ok := cat.DestinationAt("USB 5/6")
		s.Require().True(ok)
		s.Require().Equal(4, at)
	})

	s.Run("case does not matter", func() {
		at, ok := cat.SourceAt("aux")
		s.Require().True(ok)
		s.Require().Equal(2, at)

		at, ok = cat.DestinationAt("usb 1/2")
		s.Require().True(ok)
		s.Require().Equal(3, at)
	})

	s.Run("the entry that has cost an evening", func() {
		// The label names USB and an HX Stomp's Multi does not carry it, so a
		// preset left on 0 sends nothing up the cable however loud the chain.
		at, ok := cat.DestinationAt(`Multi (1/4", XLR, Digital, USB 1/2)`)
		s.Require().True(ok)
		s.Require().Zero(at)
	})

	s.Run("a name this device does not list", func() {
		_, ok := cat.SourceAt("USB 5/6")
		s.Require().False(ok, "a Stomp takes no chain input from USB 5/6")

		at, ok := cat.DestinationAt("S/PDIF")
		s.Require().False(ok)
		s.Require().Zero(at, "and the position is not a usable answer")
	})

	s.Run("a catalog carrying no routing at all", func() {
		bare := &catalog.Catalog{}

		_, ok := bare.SourceAt("Guitar")
		s.Require().False(ok)

		_, ok = bare.DestinationAt(`1/4"`)
		s.Require().False(ok)
	})
}

func TestTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TypesPublicTestSuite))
}
