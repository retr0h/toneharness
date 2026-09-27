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

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/corpus"
)

// KnowledgeTestSuite holds every page that quotes a figure to the data.
//
// A page quoting the catalog or the corpus statistics goes stale when Line 6
// ships a release or the corpus grows, and several did before anything checked
// them. So each case below names the file that carries its sentence, and the
// check follows the sentence when it moves rather than being tied to one page.
//
// Most figures are not quoted anywhere any more, on purpose: a skill resolves
// them from the tool at the moment it needs them, because a number written into
// a skill is right the day it is written and wrong after the next change with
// nothing marking the moment. Those have no case here, because there is no
// sentence to hold honest. What is left is the status board, which is prose a
// person reads, and the two skill sentences that earn a figure by arguing from
// it.
type KnowledgeTestSuite struct {
	suite.Suite
	pages map[string]string
	cat   *catalog.Catalog
	stats *corpus.Stats
}

// board is the status board, and controls is the skill page arguing from
// measured figures. Named here so a case reads as prose.
const (
	board    = "docs/knowledge.md"
	controls = ".claude/skills/measure-a-device/references/unclear-controls.md"
	trust    = ".claude/skills/measure-a-device/references/catalog-trust.md"
)

func (s *KnowledgeTestSuite) SetupSuite() {
	s.pages = map[string]string{}

	for _, name := range []string{board, controls, trust} {
		raw, err := os.ReadFile(name)
		s.Require().NoError(err)

		// Prose wraps wherever the formatter puts the line end, so a figure is
		// looked for with every run of whitespace collapsed to one space.
		s.pages[name] = strings.Join(strings.Fields(string(raw)), " ")
	}

	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)

	s.stats, err = corpus.BuiltIn()
	s.Require().NoError(err)
}

// TestTheFiguresMatchTheData recomputes every figure the page quotes.
//
// A failure names the sentence the data now supports. Either the data changed
// and the sentence should follow it, or the page quotes something new and this
// should learn to compute it.
func (s *KnowledgeTestSuite) TestTheFiguresMatchTheData() {
	names, controlCount := s.parameters()

	unclear := 0
	for _, key := range []string{"Sag", "Hum", "Ripple", "Bias", "BiasX"} {
		unclear += names[key]
	}

	brt := s.model("HD2_AmpSVBeastBrt")
	nrmBlock, _ := s.cat.Block("HD2_AmpSVBeastNrm")
	brtBlock, _ := s.cat.Block("HD2_AmpSVBeastBrt")

	tests := []struct {
		name string
		page string
		want string
	}{
		{
			name: "models mapped to real gear",
			page: board,
			want: fmt.Sprintf("%d models", s.named()),
		},
		{
			// The comparison Derive makes needs players to compare, so the
			// board says how many there are. One added without the sentence
			// following it makes the board wrong about its own method, and it
			// is also what decides whether a word stays earned.
			name: "players in the music corpus",
			page: board,
			want: fmt.Sprintf("%d players, %d records", s.players(), s.records()),
		},
		{
			name: "the amp the pipeline follows",
			page: board,
			want: fmt.Sprintf("Drive %.1f–%.1f, default %.2f, DSP %.2f",
				nrmBlock.Params["Drive"].Min, nrmBlock.Params["Drive"].Max,
				s.float(nrmBlock, "Drive"), nrmBlock.DSP.Mono),
		},
		{
			name: "how many controls there are",
			page: controls,
			want: fmt.Sprintf("The catalog holds %d parameter names across %s",
				len(names), thousands(controlCount)),
		},
		{
			name: "the controls a name does not explain",
			page: controls,
			want: fmt.Sprintf("About %d do not", roundTo(unclear, 50)),
		},
		{
			// The argument this sentence makes is that the catalog knows what
			// the device can do and the corpus knows what people do with it.
			// It only works while the two figures still differ.
			name: "a default players move away from",
			page: trust,
			want: fmt.Sprintf(
				"Line 6 state a default Treble of %.2f for one amplifier's bright channel; the median across the presets using it is %.2f",
				s.float(brtBlock, "Treble"),
				brt.Params["Treble"].Median,
			),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Contains(s.pages[tt.page], tt.want,
				"%s no longer matches the embedded data", tt.page)
		})
	}
}

// records counts the recordings every manifest names.
//
// Read with the loader the tool reads them with rather than counted as text: a
// string count agrees with the truth until somebody reflows a manifest, and then
// it is quietly wrong in the direction that makes this test pass.
func (s *KnowledgeTestSuite) records() int {
	held, err := audio.Manifests(os.DirFS("."), filepath.Join("resources", "music"))
	s.Require().NoError(err)

	n := 0
	for _, m := range held {
		n += len(m.Tracks)
	}

	return n
}

// players counts the artists the music corpus holds records for.
//
// One directory each, under the instrument they play, with the manifest
// naming the tracks. The records themselves are not in the repository.
func (s *KnowledgeTestSuite) players() int {
	found, err := filepath.Glob(
		filepath.Join("resources", "music", "*", "*", "corpus.yaml"))
	s.Require().NoError(err)

	return len(found)
}

// model reads one model's measurements, which the page's examples depend on.
func (s *KnowledgeTestSuite) model(
	id catalog.ModelID,
) corpus.ModelStats {
	ms, ok := s.stats.Models[id]
	s.Require().True(ok, "the corpus no longer measures %s", id)

	return ms
}

// float reads a parameter's stated default.
func (s *KnowledgeTestSuite) float(
	b catalog.Block,
	key string,
) float64 {
	v, ok := b.Params[key].Default.Float()
	s.Require().True(ok, "%s %s has no numeric default", b.ID, key)

	return v
}

// named counts the blocks the catalog maps to real-world gear.
func (s *KnowledgeTestSuite) named() int {
	n := 0

	for _, b := range s.cat.Blocks {
		if b.BasedOn != "" {
			n++
		}
	}

	return n
}

// parameters counts how often each parameter name occurs, and all of them.
func (s *KnowledgeTestSuite) parameters() (map[string]int, int) {
	names := map[string]int{}
	total := 0

	for _, b := range s.cat.Blocks {
		for key := range b.Params {
			names[key]++
			total++
		}
	}

	return names, total
}

// roundTo rounds n to the nearest multiple of step, for figures quoted as
// "about".
func roundTo(
	n, step int,
) int {
	return int(math.Round(float64(n)/float64(step))) * step
}

// thousands writes n with a comma between each group of three digits.
func thousands(
	n int,
) string {
	digits := fmt.Sprint(n)

	var b strings.Builder

	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}

		b.WriteRune(d)
	}

	return b.String()
}

func TestKnowledgeTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KnowledgeTestSuite))
}
