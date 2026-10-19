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

package compile_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/compile"
	"github.com/retr0h/toneharness/pkg/sdk/shipped"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// ShippedPublicTestSuite holds the asks this repository ships to the
// vocabulary, which nothing holds anybody else's ask to.
//
// An ask somebody writes gets a note naming the terms nothing defines and
// still builds, because a word nothing defines moves no knob and refusing a
// preset over one would be refusing them the right to describe a sound. The
// asks here are different: they are the examples everybody copies, and every
// one of them said what it meant in a sentence until there was a list.
//
// The asks rather than the rigs, because the words left the rig: how it should
// sound is what somebody wanted, and a rig says only which gear answered.
type ShippedPublicTestSuite struct {
	suite.Suite
}

// words reads what an ask says it should sound like.
func (s *ShippedPublicTestSuite) words(
	open func() (fs.File, error),
) []string {
	f, err := open()
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	spec, err := tone.Load(f)
	s.Require().NoError(err)

	if spec.Ask == nil || spec.Ask.Words == nil {
		return nil
	}

	// The terms alone. What the vocabulary can say about a word does not
	// depend on why it is believed.
	out := make([]string, 0, len(*spec.Ask.Words))
	for _, w := range *spec.Ask.Words {
		out = append(out, w.Term)
	}

	return out
}

// TestEveryShippedAskUsesTheVocabulary covers the terms every shipped ask
// describes a sound with.
//
// Read through the embedded copy rather than off disk, so this counts no
// directories and travels wherever the package does.
func (s *ShippedPublicTestSuite) TestEveryShippedAskUsesTheVocabulary() {
	paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no documents found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			for _, u := range compile.CheckWords(
				s.words(func() (fs.File, error) { return shipped.FS.Open(path) }),
			) {
				s.Require().Fail("no such character term",
					"%q. Add it to pkg/sdk/compile/data/words.json "+
						"with a definition, or use one of: %v", u.Term, u.Near)
			}
		})
	}
}

// TestEveryShippedAskAnswersEachAxisOnce covers an ask arguing with itself.
//
// An ask claiming two terms from one axis has claimed nothing: the two cancel,
// the control stays where the corpus left it, and a build says so on every
// run. mike-dirnt shipped claiming both minimal-drive and grit-on-attack, and
// nothing caught it until the words started moving knobs.
func (s *ShippedPublicTestSuite) TestEveryShippedAskAnswersEachAxisOnce() {
	paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no documents found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			for _, c := range compile.CheckAxes(
				s.words(func() (fs.File, error) { return shipped.FS.Open(path) }),
			) {
				s.Require().Fail("one axis answered twice",
					"%q: %v. Keep the term that says the most and drop the "+
						"rest, or neither will be applied.", c.Axis, c.Terms)
			}
		})
	}
}

// TestEveryExampleUsesTheVocabulary covers the asks the docs point at.
func (s *ShippedPublicTestSuite) TestEveryExampleUsesTheVocabulary() {
	paths, err := filepath.Glob(
		filepath.Join(
			"..",
			"..",
			"testdata",
			"documents",
			"*.yaml",
		),
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no examples found to check")

	for _, path := range paths {
		// The directory holds two other kinds beside the documents: a setup,
		// which says what somebody owns, and a plan, which is the device half of
		// a preset. Neither has a word in it.
		if strings.HasSuffix(path, ".setup.yaml") ||
			strings.HasSuffix(path, ".plan.yaml") {
			continue
		}

		s.Run(filepath.Base(path), func() {
			words := s.words(func() (fs.File, error) { return os.Open(path) })

			s.Require().Empty(compile.CheckWords(words))
			s.Require().Empty(compile.CheckAxes(words))
		})
	}
}

// distinctive is the words in a gear name that say which gear it is.
//
// A maker's name and a cabinet's size are shared by dozens of models, so an
// "Ampeg 8x10" matching an "Ampeg SVT 410HLF" on the word "Ampeg" would call
// a substitution a match. What is left is the part that identifies the unit.
func distinctive(
	name string,
) map[string]bool {
	shared := map[string]bool{
		"the": true, "a": true, "ampeg": true, "legacy": true, "line": true,
		"6": true, "original": true, "pro": true, "brt": true, "nrm": true,
		"ch1": true, "ch2": true, "jump": true, "x": true,
		"2": true, "4": true, "8": true, "10": true, "12": true, "15": true,
	}

	out := map[string]bool{}

	for _, w := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if !shared[w] {
			out[w] = true
		}
	}

	return out
}

// shares says whether two gear names have a distinctive word in common.
func shares(
	gear string,
	names ...string,
) bool {
	want := distinctive(gear)
	if len(want) == 0 {
		return true
	}

	for _, n := range names {
		for w := range distinctive(n) {
			if want[w] {
				return true
			}
		}
	}

	return false
}

// specs reads every rig that ships, by identifier.
func (s *ShippedPublicTestSuite) specs() map[string]tone.Spec {
	paths, err := fs.Glob(shipped.FS, filepath.Join("*", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no documents found to check")

	out := map[string]tone.Spec{}

	for _, p := range paths {
		f, err := shipped.FS.Open(p)
		s.Require().NoError(err)

		spec, err := tone.Load(f)
		s.Require().NoError(f.Close())
		s.Require().NoError(err)

		out[filepath.Base(p)] = spec
	}

	return out
}

// TestEveryShippedGearClaimHasASource covers the first of the two bars a rig
// clears before it may ship: a source somebody opened for every claim.
//
// `llm` alone means a model asserted the gear and nobody confirmed it, which is
// the same thing as made up. Four rigs read that way until this test existed,
// and nothing but reading all of them by hand would have said so: an Ampeg SVT
// for Charlie Hayward, the same for Tommy Caldwell and Cone McCaslin, and a
// Marshall stack for Dusty Hill that two Premier Guitar pieces were opened
// looking for and neither carries.
//
// A corpus-added block is exempt and is not an exception. It is a statistic
// over other people's presets rather than a claim about this player, it says so
// in its own note, and no research would make it anything else.
func (s *ShippedPublicTestSuite) TestEveryShippedGearClaimHasASource() {
	sourced := map[tone.EvidenceKind]bool{
		"cited": true, "heard": true, "user": true, "audio": true, "store": true,
	}

	for name, spec := range s.specs() {
		s.Run(name, func() {
			for _, entry := range spec.Rig.Chain {
				s.Require().NotNil(
					entry.Evidence,
					"%s %q carries no evidence at all", entry.Role, entry.Gear,
				)

				kinds := map[tone.EvidenceKind]bool{}
				for _, ev := range *entry.Evidence {
					kinds[ev.Kind] = true
				}

				if kinds["corpus"] {
					continue
				}

				has := false
				for k := range kinds {
					if sourced[k] {
						has = true

						break
					}
				}

				s.Require().True(
					has,
					"%s %q rests on no source anybody opened. Research it, or "+
						"move the rig to marketplace/community",
					entry.Role, entry.Gear,
				)
			}
		})
	}
}

// TestEveryShippedAmplifierResolvesToItself covers the second bar: the
// amplifier named is the amplifier that comes out.
//
// A gear name resolves to the nearest model when the real thing is not
// emulated, which is right for somebody building a sound and wrong to ship as a
// citation. Berry Oakley's Fender PS400 came out an Ampeg SVT, which is not a
// Berry Oakley rig but an Ampeg SVT with his name on it. Five rigs read that
// way and the document showed nothing: the file says `Fender 400 PS` and only
// the resolve says otherwise.
//
// The amplifier rather than every role, because it is the block that decides
// the sound, and because this device models fourteen bass amplifiers and fewer
// cabinets. A cabinet may substitute where the claim is cited and the caveat
// says so: Geddy Lee's Ampeg V-4B cabinets are cited from three print sources
// in his own words and nothing here models one, and deleting that would throw
// away the research without making the rig any more accurate.
func (s *ShippedPublicTestSuite) TestEveryShippedAmplifierResolvesToItself() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	for name, spec := range s.specs() {
		s.Run(name, func() {
			asked := ""
			for _, entry := range spec.Rig.Chain {
				if entry.Role == tone.RoleAmp {
					asked = entry.Gear

					break
				}
			}

			if asked == "" {
				return
			}

			// No corpus, so nothing is filled and what comes back is the rig.
			got, _, _, _, err := compile.Resolve(
				spec.Id, spec.Rig, compile.Intent{}, cat, nil,
			)
			s.Require().NoError(err)

			for _, b := range got.Blocks {
				block, found := cat.Block(b.Model)
				s.Require().True(found, "model %q is not in the catalog", b.Model)

				if block.Category != catalog.CategoryAmp {
					continue
				}

				s.Require().True(
					shares(asked, block.Name, block.BasedOn),
					"asked for %q and got %q (%s). An amplifier is never "+
						"substituted in a shipped rig: move it to "+
						"marketplace/community and say so in the caveat",
					asked, block.Name, block.BasedOn,
				)
			}
		})
	}
}

// TestNoShippedRigNamesADeviceModel covers the third bar: a rig says what the
// gear is, never what one manufacturer calls their model of it.
//
// The contract's `gear` field has always said so, and twelve shipped documents
// broke it anyway, because the two writers disagreed: the compiler names a
// block it adds by the gear it emulates, so a cabinet read `Legacy 6x10"
// MESA/Boogie Power House`, while the chooser that picks an amplifier by
// measurement named it off the measured library, which files a sweep under Line
// 6's own name. Ten genre rigs therefore said `Cali 400 Ch1` for a MESA/Boogie
// Bass 400+ and one said `Agua Sledge` for an Aguilar Tone Hammer.
//
// A model name and the gear's name are the same string often enough that this
// has to tell them apart rather than refuse every match: the catalog's own entry
// for an Ampeg B-15NF is named "Ampeg B-15NF". What marks a model name is that
// it says something else from the gear underneath it, which is why this asks
// whether the name is part of what the block emulates.
func (s *ShippedPublicTestSuite) TestNoShippedRigNamesADeviceModel() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	// Every model name the device carries, against the gear it emulates, held
	// only where the two say different things.
	models := map[string]catalog.Block{}

	for _, block := range cat.Blocks {
		if block.Name == "" || block.Gear() == "" {
			continue
		}

		if strings.Contains(
			strings.ToLower(block.Gear()), strings.ToLower(block.Name),
		) {
			continue
		}

		models[strings.ToLower(block.Name)] = block
	}

	s.Require().NotEmpty(models, "no model names to check against")

	for name, spec := range s.specs() {
		s.Run(name, func() {
			for _, entry := range spec.Rig.Chain {
				block, named := models[strings.ToLower(strings.TrimSpace(entry.Gear))]

				s.Require().False(
					named,
					"%s is named %q, which is Line 6's model name for a %s. A "+
						"rig names the gear: those labels change between "+
						"releases and mean nothing on other hardware",
					entry.Role, entry.Gear, block.Gear(),
				)
			}
		})
	}
}

// TestEveryMarketplaceRigCarriesItsResolution covers a document being the
// whole answer rather than a gesture at one.
//
// A rig may name one amplifier and nothing else, and resolving it writes every
// control the chain ends up holding, so the file describes the preset instead
// of describing the research. Both tiers were half and half: nineteen of
// thirty-one core rigs carried their controls and twelve did not, with nothing
// marking which, and a reader could not tell a rig nobody had resolved from one
// whose gear this device cannot do.
//
// Either direction fails, which is what catches the second kind. Dusty Hill's
// rig cited a Marshall JMP-1 and carried thirteen controls with `BrtDrive` and
// `NrmDrive` in them: a jumped Marshall Super Lead, which is a guitar
// amplifier, resolved into a bass rig before the shortlist was held to the
// instrument. The resolver refuses it today, so the controls were a claim
// nothing could reproduce, and a test asking only "is it resolved" would have
// passed it.
func (s *ShippedPublicTestSuite) TestEveryMarketplaceRigCarriesItsResolution() {
	cat, err := catalog.BuiltIn()
	s.Require().NoError(err)

	paths, err := filepath.Glob(
		filepath.Join("..", "..", "..", "..", "marketplace", "*", "*", "*.yaml"),
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no marketplace rigs found to check")

	for _, path := range paths {
		s.Run(filepath.Join(filepath.Base(filepath.Dir(path)), filepath.Base(path)), func() {
			f, err := os.Open(path)
			s.Require().NoError(err)

			spec, err := tone.Load(f)
			s.Require().NoError(f.Close())
			s.Require().NoError(err)

			// Whether the document says what it resolved to. `built` is the
			// section only a resolve writes, and since version 4 it is the
			// whole of what a resolve wrote: the chain with its controls, the
			// preset's members, the switches, and which catalog answered.
			//
			// It used to read `rig.resolved`, which is gone. That field sat in
			// the authored half beside the gear a person typed, which is the
			// conflation this split exists to end.
			claims := spec.Built != nil

			// No corpus, so nothing is filled from statistics and what comes
			// back is this rig on this device.
			_, _, _, _, err = compile.Resolve(
				spec.Id, spec.Rig, compile.Intent{}, cat, nil,
			)
			if err != nil {
				s.Require().False(
					claims,
					"says what it resolved to and will not resolve: %v. A "+
						"resolution nothing can reproduce is a claim about a "+
						"preset nobody can build, so take the controls out and "+
						"leave the gear and its evidence",
					err,
				)

				return
			}

			s.Require().True(
				claims,
				"resolves on this device and does not say what to. Run "+
					"`rigs resolve --dir <tier> --id %s` so the file carries "+
					"every control the chain holds",
				spec.Id,
			)
		})
	}
}

func TestShippedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShippedPublicTestSuite))
}
