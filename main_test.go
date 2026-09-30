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
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/toneharness/cmd"
	sdk "github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
)

// mod is this module, so a test can tell its own packages from anybody's.
const mod = "github.com/retr0h/toneharness/"

// MainTestSuite covers the shape of the repository rather than its behaviour.
type MainTestSuite struct {
	suite.Suite
}

// TestThereIsNoTopLevelInternal covers where a private half lives.
//
// Under the package that owns it, as pkg/<name>/internal/, so the compiler
// fences it to exactly that package and it moves when that package moves. At
// the root an internal/ is readable by everything in the module and owned by
// nothing, which is how code ends up somewhere no extraction takes with it.
func (s *MainTestSuite) TestThereIsNoTopLevelInternal() {
	_, err := os.Stat("internal")

	s.Require().ErrorIs(err, fs.ErrNotExist,
		"put a private package under the package that owns it, as pkg/<name>/internal/")
}

// TestEveryManifestSitsUnderAnInstrument holds the music corpus to its shape.
//
// A word is earned by sitting clear of the other players in the tree, and a
// guitar's centre of gravity sits an octave above a bass guitar's, so the
// directory holding a comparison is the instrument. A manifest anywhere else
// is a player nothing compares, or a leftover of a move: three of them
// survived nesting the corpus, byte-identical to the real ones and invisible
// to everything that reads them.
func (s *MainTestSuite) TestEveryManifestSitsUnderAnInstrument() {
	found, err := filepath.Glob(filepath.Join("resources", "music", "*", "*", "corpus.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(found)

	instruments := map[string]bool{"bass": true, "guitar": true}

	for _, at := range found {
		s.Require().True(instruments[filepath.Base(filepath.Dir(filepath.Dir(at)))],
			"%s: the directory above a player names the instrument they play", at)
	}

	// Anything at another depth, which is what a move leaves behind.
	stray, err := filepath.Glob(filepath.Join("resources", "music", "*", "corpus.yaml"))
	s.Require().NoError(err)
	s.Require().Empty(stray,
		"a manifest beside the instrument directories rather than inside one")
}

// TestEveryManifestReads parses every manifest the repository ships.
//
// Nothing read them until this. A note written as a plain multi-line scalar
// with a colon in it, added by hand, made one manifest unparseable and the
// whole suite stayed green: `toneharness measure --manifest` failed on it and
// no test did.
func (s *MainTestSuite) TestEveryManifestReads() {
	found, err := filepath.Glob(filepath.Join("resources", "music", "*", "*", "corpus.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(found)

	for _, at := range found {
		f, err := os.Open(at) //nolint:gosec // a path from this repository
		s.Require().NoError(err)

		m, err := audio.ReadManifest(f)
		s.Require().NoError(f.Close())
		s.Require().NoError(err, "%s does not read", at)
		s.Require().NotEmpty(m.Tracks, "%s names no records", at)
	}
}

// TestEveryRecordIsInTheEraItsRigDescribes holds the shipped rigs to their
// own years.
//
// `rigs records` reports this and reporting was right while four of the
// nine disagreed: which half is wrong is a judgement and a build failing
// would not have made it. They agree now, so the next one to drift should
// stop somebody rather than wait to be noticed.
//
// A record from another era is not a small error. Mike Dirnt's rig described
// his American Idiot rig and his records were Dookie, and measured from the
// right ones he earns the opposite word on two axes.
func (s *MainTestSuite) TestEveryRecordIsInTheEraItsRigDescribes() {
	all, err := sdk.New().Backing(
		context.Background(), filepath.Join("resources", "music", "bass"))
	s.Require().NoError(err)
	s.Require().NotEmpty(all)

	for _, b := range all {
		if len(b.Records) == 0 {
			continue
		}

		// Records arrive before the rig that will use them, so a directory no
		// rig answers to has no era to be held to yet.
		if b.NoRig {
			continue
		}

		s.Require().True(b.Stated(),
			"%s has records and no years to hold them to", b.ID)

		for _, r := range b.Records {
			s.Require().False(r.Outside,
				"%s: %s is from %d, outside the %d–%d this rig describes",
				b.ID, r.Track, r.Year, b.From, b.To)
		}
	}
}

// TestATestFileSaysWhichKindItIs asserts the suffix and the package agree.
//
// CONTRIBUTING gives two kinds of test file and a name for each: a
// `*_public_test.go` in the package's `_test` package exercises what the
// package promises, and a `*_test.go` in the package itself reaches what that
// surface cannot. Nine files once said the second and meant the first, which
// makes a public test look like an internal one and hides how much of a
// package is actually exercised from outside.
//
// export_test.go is neither. It exists to hand an unexported thing to an
// external test and belongs in the package.
func (s *MainTestSuite) TestATestFileSaysWhichKindItIs() {
	fset := token.NewFileSet()

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			// Nothing this repository wrote, and nothing it can fix.
			if name := d.Name(); name == ".git" || name == ".worktrees" ||
				name == "node_modules" {
				return fs.SkipDir
			}

			return nil
		case !strings.HasSuffix(path, "_test.go"),
			strings.HasSuffix(path, "_public_test.go"),
			d.Name() == "export_test.go":
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if err != nil {
			return err
		}

		s.Require().False(strings.HasSuffix(f.Name.Name, "_test"),
			"%s is in package %s, so it is a public test and its name should "+
				"end in _public_test.go", path, f.Name.Name)

		return nil
	})

	s.Require().NoError(err)
}

// TestEverySignatureTakesALinePerParameter holds the rule CONTRIBUTING gives
// a function's parameters.
//
// One to a line, with the closing parenthesis on a line after the last, so
// adding a parameter shows as one added line rather than a rewritten
// signature. No linter checks it: lll and golines only measure length. It
// was broken 753 times before this test existed, and a rule broken that often
// protects nothing.
//
// Every function and method declaration with parameters, test files included.
// Function literals and interface methods are exempt: a literal is usually a
// one-line callback, and an interface lists shapes rather than code anybody
// diffs. Generated files are exempt because nobody writes them.
func (s *MainTestSuite) TestEverySignatureTakesALinePerParameter() {
	fset := token.NewFileSet()

	var broken []string

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			// Nothing this repository wrote, and nothing it can fix. .claude
			// holds other checkouts' worktrees.
			switch d.Name() {
			case ".git", ".worktrees", ".claude", "node_modules":
				return fs.SkipDir
			}

			return nil
		case !strings.HasSuffix(path, ".go"),
			strings.HasSuffix(path, ".gen.go"),
			strings.HasSuffix(path, ".gen_test.go"):
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		if ast.IsGenerated(f) {
			return nil
		}

		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && !oneLinePerParameter(fset, fn.Type.Params) {
				broken = append(broken,
					fmt.Sprintf("%s: %s", fset.Position(fn.Pos()), fn.Name.Name))
			}
		}

		return nil
	})
	s.Require().NoError(err)

	s.Require().Empty(broken,
		"put each parameter on a line of its own and the closing parenthesis "+
			"on the line after the last; see Function signatures in CONTRIBUTING.md")
}

// oneLinePerParameter reports whether a parameter list follows the rule: each
// parameter starts on a line after the opening parenthesis or the parameter
// before it, and the closing parenthesis sits on a line after the last. A list
// with no parameters stays on one line.
func oneLinePerParameter(
	fset *token.FileSet,
	params *ast.FieldList,
) bool {
	if len(params.List) == 0 {
		return true
	}

	line := func(p token.Pos) int { return fset.Position(p).Line }

	prev := line(params.Opening)

	for _, f := range params.List {
		if line(f.Pos()) <= prev {
			return false
		}

		prev = line(f.End())
	}

	return line(params.Closing) > prev
}

// TestTheSDKTakesNothingElseWithIt holds the device half where the argument
// for it being liftable assumes it is.
//
// The device, its wire and slot addressing are one unit: framing, transport,
// session and addressing. Everything the SDK needs from this module is those
// three, which is what makes "the device half could be its own repository" a
// fact rather than a hope.
//
// Nothing in the compiler stops somebody importing a format package from the
// device half on a Tuesday, and the day that happens the seam is welded shut
// without anybody noticing. Hence a test.
//
// The reverse is asserted by its absence: the format packages are free to
// depend on each other, and none of them reaches the device.
func (s *MainTestSuite) TestTheSDKTakesNothingElseWithIt() {
	unit := map[string]bool{
		mod + "pkg/sdk/internal/device": true,
		mod + "pkg/sdk/internal/wire":   true,
		mod + "pkg/sdk/slot":            true,
	}

	out, err := exec.Command(
		"go", "list", "-deps", "./pkg/sdk/internal/device").Output()
	s.Require().NoError(err)

	for _, dep := range strings.Fields(string(out)) {
		if !strings.HasPrefix(dep, mod) {
			continue
		}

		s.Require().True(unit[dep],
			"%s is not part of the SDK, and the SDK reaching it means the "+
				"device half can no longer be lifted out on its own", dep)
	}
}

// TestBackupsDoNotReachTheDevice holds the backup policy apart from the
// transport.
//
// What gets kept before a write, in which format and where, is a decision
// about somebody's presets, not about USB. It reads a device's answer through
// a Decoder the device flows hand in, so the day it imports the device or its
// wire is the day the policy starts changing when the transport does.
func (s *MainTestSuite) TestBackupsDoNotReachTheDevice() {
	out, err := exec.Command(
		"go", "list", "-deps", "./pkg/sdk/internal/backup").Output()
	s.Require().NoError(err)

	for _, dep := range strings.Fields(string(out)) {
		switch dep {
		case mod + "pkg/sdk/internal/device", mod + "pkg/sdk/internal/wire":
			s.Require().Fail("reaches the transport",
				"pkg/sdk/internal/backup reaches %s; hand it a Decoder instead", dep)
		}
	}
}

// TestTheSDKStandsAlone asserts that pkg/sdk needs nothing outside itself.
//
// The question this answers is whether the library is usable if somebody
// lifts it into a repository of its own, and "it does not import internal/"
// is not that answer. It imported resources/schemas until somebody asked:
// the contract every rig is checked against, and the vocabulary a character
// term comes from, both embedded files sitting outside the directory that
// was supposed to be able to leave.
//
// So the rule is stronger than the one about internal/. Nothing under
// pkg/sdk may reach anything in this module that is not also under pkg/sdk,
// which includes data. Where a package needs a file, the file lives beside
// it, the way the catalog, the corpus and the preset template already do.
func (s *MainTestSuite) TestTheSDKStandsAlone() {
	out, err := exec.Command("go", "list", "./pkg/sdk/...").Output()
	s.Require().NoError(err)

	for _, pkg := range strings.Fields(string(out)) {
		deps, err := exec.Command("go", "list", "-deps", pkg).Output()
		s.Require().NoError(err)

		for _, dep := range strings.Fields(string(deps)) {
			if !strings.HasPrefix(dep, mod) {
				continue
			}

			s.Require().True(strings.HasPrefix(dep, mod+"pkg/sdk"),
				"%s reaches %s, which would not travel with the SDK", pkg, dep)
		}
	}
}

// TestEveryPathThisRepositoryNamesExists holds the paths that move.
//
// A default like `--out pkg/catalog/data/hx-stomp.json.gz` keeps compiling
// after the directory it names has moved, and keeps running: it writes a file
// where nothing reads one. Both generator defaults were wrong for a month
// that way, so regenerating the catalog silently stopped updating the
// embedded copy.
//
// The directory rather than the file, because some of what this repository
// names is not this repository's to ship. The catalog and the gear map come
// out of a licensed HX Edit installation and are not committed, so asserting
// the file would fail everywhere but a machine that has built them. Asserting
// where they go still catches a directory that moved.
//
// Only non-test files. A test names paths that do not exist on purpose.
func (s *MainTestSuite) TestEveryPathThisRepositoryNamesExists() {
	named := regexp.MustCompile(
		`"(pkg|internal|resources|docs|examples)/[A-Za-z0-9_./-]+"`)

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}

		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		body, err := os.ReadFile(path) //nolint:gosec // a path this walk found
		if err != nil {
			return err
		}

		for _, m := range named.FindAllString(string(body), -1) {
			want := filepath.Dir(strings.Trim(m, `"`))

			_, err := os.Stat(want)
			s.Require().NoError(err,
				"%s names a path under %s, which is not there", path, want)
		}

		return nil
	})

	s.Require().NoError(err)
}

// TestEveryPackageIsInTheStructureTree is the other direction.
//
// TestEveryPathThisRepositoryNamesExists checks that a path CONTRIBUTING names
// is there. It cannot catch a package CONTRIBUTING never named, and three had
// gone unnamed: `solve`, which pkg/cli imports in six files, `shipped/artists`,
// and two view packages the tree called by a name nothing has ever been called
// — `corpusview/`, against the real `musicview/` and `presetsview/`.
//
// That tree is how somebody finds their way around before they know the code,
// and the import table beside it is the whole answer to what an outside caller
// may reach for. A wrong name there costs more than no name: it sends a reader
// looking for a directory that is not there.
func (s *MainTestSuite) TestEveryPackageIsInTheStructureTree() {
	page, err := os.ReadFile("CONTRIBUTING.md")
	s.Require().NoError(err)

	held := string(page)

	err = filepath.Walk("pkg", func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return err
		}

		// A directory holding no Go of its own is not a package. data/, gen/,
		// testdata/ and mocks/ are the tree's furniture rather than its shape,
		// and `internal` itself is a marker rather than a package.
		switch info.Name() {
		case "data", "gen", "testdata", "mocks":
			return filepath.SkipDir
		case "internal":
			return nil
		}

		found, err := filepath.Glob(filepath.Join(path, "*.go"))
		if err != nil || len(found) == 0 {
			return err
		}

		// A `package main` under pkg/ is something `go:generate` runs rather
		// than something anybody imports — genrepack, pack, the two datagens.
		// Naming each in the tree would fill it with plumbing.
		body, err := os.ReadFile(found[0]) //nolint:gosec // a path this walk found
		if err != nil {
			return err
		}

		if strings.Contains(string(body), "\npackage main\n") {
			return nil
		}

		// The tree writes a public package as its whole path and an internal
		// one as its basename indented under `internal/`, which is how it
		// stays readable. Either spelling counts as naming it.
		want := path + "/"
		if strings.Contains(path, "/internal/") {
			want = filepath.Base(path) + "/"
		}

		s.Require().Contains(held, want,
			"CONTRIBUTING's project structure does not name %s", path)

		return nil
	})

	s.Require().NoError(err)
}

// TestEveryGeneratedPageIsLeftOutOfTheFormatter holds the two steps of the
// gate to the same answer.
//
// `just ready` runs `just generate` and then `just md-fmt`. A generated page
// that the formatter is allowed to touch is therefore rewritten every run:
// the generator hard-wraps its prose and leaves its tables unpadded, mdformat
// reflows both, and the test comparing the page against its generator fails
// on the next `just test`. Running `just generate` again does not clear it,
// because the next `just ready` reflows the page straight back.
//
// A generated grammar page broke exactly that way the day it was added: it was
// written, and the exclusion list beside it was not. This fails when a page
// is generated and not excluded, rather than a run later and somewhere else.
func (s *MainTestSuite) TestEveryGeneratedPageIsLeftOutOfTheFormatter() {
	rig, err := os.ReadFile("justfile")
	s.Require().NoError(err)

	pages, err := filepath.Glob(filepath.Join("docs", "*.md"))
	s.Require().NoError(err)
	s.Require().NotEmpty(pages)

	for _, page := range pages {
		body, err := os.ReadFile(page) //nolint:gosec // a path this glob found
		s.Require().NoError(err)

		if !strings.Contains(string(body), "Do not edit.") {
			continue
		}

		s.Require().Contains(string(rig), "--exclude '"+page+"'",
			"%s is generated, so mdformat reflowing it would leave the page "+
				"disagreeing with its generator. Add it to md_extra_excludes.",
			page)
	}
}

// knownUnread is every contract field no file names, on purpose.
//
// None, currently. `mutations` was the last one: #135 moved it onto the
// ToneSpec as `corrections` and `tone build` reads it out, because a rebuild
// does not replay a correction and that is worth saying rather than leaving to
// be rediscovered.
//
// The map is here rather than absent so an exemption stays a decision. A field
// that leaves it without gaining a reader fails, a field added to the contract
// that quietly reaches nothing fails, and so does an entry naming a field
// neither contract declares any more.
var knownUnread = map[string]string{}

// TestEveryContractFieldReachesSomething holds each contract to what it builds.
//
// John asked for this as a verification: "everything in rigspec needs to turn
// into an action". Performed by hand it found `requires`, a field describing
// impulse responses a device does not ship with that nothing in this
// repository ever read — superseded by a Setup's `owns` before it was wired
// up, and carried for months looking like a feature.
//
// So the check is a test rather than an afternoon. A field is read when some
// file outside the generated types and outside the tests names it.
//
// That is coarse, and the coarseness is the point rather than a shortcut.
// It matches on the field's name alone, so `Evidence` counts as read because
// a CharacterTerm's evidence is read, even though a rig's own is not. Telling
// those apart needs the type checker, and the failure worth catching does not:
// `requires` was a field whose name appeared in no file at all, and it sat
// there for months looking like a feature. This catches that, every time,
// for the cost of parsing one generated file.
//
// Which of the fields it passes are read as themselves is #135's audit, and
// that one wants a person rather than a test.
//
// Both hand-authored contracts, not just the rig. #135 moves what a person
// writes onto the ToneSpec, so a field that reaches nothing is a thing that
// can now happen on either of them, and a guard on one of the two is a guard
// on whichever half the next field did not land in.
func (s *MainTestSuite) TestEveryContractFieldReachesSomething() {
	fields := append(
		s.contractFields("rig", "rigspec.gen.go", "RigSpec"),
		s.contractFields("tone", "tonespec.gen.go", "ToneSpec")...)
	s.Require().NotEmpty(fields)

	// An exemption for a field neither contract declares is a note about
	// something that no longer exists, and it silently stops guarding
	// anything: the loop below only visits fields that are still there.
	for name := range knownUnread {
		s.Require().Contains(fields, name,
			"knownUnread names %s, which neither contract declares any more. "+
				"Take the entry out.", name)
	}

	var body strings.Builder

	for _, dir := range []string{"pkg", "cmd"} {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			switch {
			case err != nil || info.IsDir():
				return err
			case !strings.HasSuffix(path, ".go"):
				return nil
			case strings.HasSuffix(path, "_test.go"):
				return nil
			case strings.HasSuffix(path, ".gen.go"):
				return nil
			}

			at, err := os.ReadFile(path) //nolint:gosec // a path this walk found
			if err != nil {
				return err
			}

			body.Write(at)

			return nil
		})
		s.Require().NoError(err)
	}

	read := body.String()

	for _, name := range fields {
		why, expected := knownUnread[name]
		got := strings.Contains(read, "."+name)

		switch {
		case got && expected:
			s.Require().Fail("a field gained a reader and is still listed as dead",
				"%s is read now, so take it out of knownUnread (%s)",
				name, why)
		case !got && !expected:
			s.Require().Fail("a field in the contract reaches nothing",
				"%s is named by no file that builds anything. Either "+
					"make it do something, take it out of the contract, or "+
					"add it to knownUnread saying which task carries it.", name)
		}
	}
}

// contractFields is every field one generated contract type declares.
//
// Read off the generated type rather than the contract, because the contract
// says `snapshots` and the code says `Snapshots`, and the rule turning one
// into the other belongs to the generator rather than to this test.
func (s *MainTestSuite) contractFields(
	pkg, file, kind string,
) []string {
	at := filepath.Join("pkg", "sdk", pkg, "internal", "gen", file)

	parsed, err := parser.ParseFile(token.NewFileSet(), at, nil, 0)
	s.Require().NoError(err)

	var out []string

	ast.Inspect(parsed, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != kind {
			return true
		}

		body, ok := spec.Type.(*ast.StructType)
		if !ok {
			return false
		}

		for _, f := range body.Fields.List {
			for _, name := range f.Names {
				if name.IsExported() {
					out = append(out, name.Name)
				}
			}
		}

		return false
	})

	return out
}

// TestEveryDomainPageIsIndexed holds the two tables that point at docs/.
//
// docs/README.md is what a person opens, and AGENTS.md's table is what an
// agent reads to pick the one page matching the task. Both list the pages by
// hand, so both fall behind silently: a page nobody indexed is a page nobody
// is sent to, and it goes stale because nobody is reading it either.
//
// Four had fallen off docs/README.md — the two generated grammar pages, the
// algorithm and the measuring loop — while AGENTS.md had them all.
func (s *MainTestSuite) TestEveryDomainPageIsIndexed() {
	pages, err := filepath.Glob(filepath.Join("docs", "*.md"))
	s.Require().NoError(err)
	s.Require().NotEmpty(pages)

	for _, at := range []struct{ file, names string }{
		{filepath.Join("docs", "README.md"), ""},
		{"AGENTS.md", "docs/"},
	} {
		body, err := os.ReadFile(at.file)
		s.Require().NoError(err)

		for _, page := range pages {
			name := filepath.Base(page)
			if name == "README.md" {
				continue
			}

			s.Require().Contains(string(body), at.names+name,
				"%s does not point at docs/%s, so nobody is sent to it",
				at.file, name)
		}
	}
}

// TestContextComesFirst holds the rule CONTRIBUTING gives a context.
//
// "Every method takes a context.Context first." Go's own convention, and the
// reason is that a reader should not have to check: a signature where it
// sometimes leads and sometimes sits third makes cancellation something to
// look up rather than something to know.
//
// Seven functions had it second or third, all of them in the three measuring
// commands ported from Python, where the shape came across with the code. No
// linter checks it, so it drifted silently, the way the signature rule did.
func (s *MainTestSuite) TestContextComesFirst() {
	fset := token.NewFileSet()

	var late []string

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			switch d.Name() {
			case ".git", ".worktrees", ".claude", "node_modules":
				return fs.SkipDir
			}

			return nil
		case !strings.HasSuffix(path, ".go"),
			strings.HasSuffix(path, ".gen.go"),
			strings.HasSuffix(path, ".gen_test.go"):
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			decl, ok := n.(*ast.FuncDecl)
			if !ok || decl.Type.Params == nil {
				return true
			}

			for i, p := range decl.Type.Params.List {
				if i == 0 || !isContext(p.Type) {
					continue
				}

				late = append(late, fmt.Sprintf("%s:%d %s takes a context as "+
					"parameter %d", path, fset.Position(decl.Pos()).Line,
					decl.Name.Name, i+1))
			}

			return true
		})

		return nil
	})

	s.Require().NoError(err)
	s.Require().Empty(late, "a context goes first:\n%s", strings.Join(late, "\n"))
}

// isContext reports whether a parameter's type is context.Context.
func isContext(
	of ast.Expr,
) bool {
	at, ok := of.(*ast.SelectorExpr)
	if !ok || at.Sel.Name != "Context" {
		return false
	}

	from, ok := at.X.(*ast.Ident)

	return ok && from.Name == "context"
}

// coversAConcern is every test file named for something other than a
// production file, with what it covers instead.
//
// Each is a decision rather than a drift. Two shapes qualify: a test that
// crosses several files on purpose — a round trip through a reader and a
// writer, a walk over everything that ships — and a file holding fixtures
// with no tests of its own.
//
// What is NOT on this list, and was checked: four files in catalog and rig
// that test one concern inside a bigger production file. Those want the
// production file split, which is the rule's own answer and a separate
// change, not a line here.
var coversAConcern = map[string]string{
	"hints_test.go":                                       "every command this repository names, against the cobra tree",
	"pkg/cli/cli_face_public_test.go":                     "the face a terminal presents, across the package",
	"pkg/cli/measure_public_test.go":                      "the pieces three measuring commands share",
	"pkg/cli/measure_run_public_test.go":                  "a campaign end to end",
	"pkg/cli/internal/paint/paint_public_test.go":         "the visual language, across theme, chain and yaml",
	"pkg/cli/measure_paths_test.go":                       "what the three measuring commands do when refused",
	"pkg/cli/technique_test.go":                           "a helper in rig.go, which #136 renames wholesale",
	"pkg/sdk/catalog/led_public_test.go":                  "Catalog.LEDColour, which CONTRIBUTING lets types.go keep",
	"pkg/sdk/catalog/symbol_public_test.go":               "Catalog.Symbol, which CONTRIBUTING names as allowed there",
	"pkg/sdk/internal/corpusgen/corpusgen_public_test.go": "Run end to end, which drives measure.go and refresh.go",
	"pkg/sdk/plan/testdata_public_test.go":                "fixtures the package's tests build from",
	"pkg/sdk/device_public_test.go":                       "a round trip against real hardware, behind a build tag",
	"pkg/sdk/internal/catalogen/stale_public_test.go":     "the shipped catalog against an installed HX Edit",
	"pkg/sdk/internal/compile/words_moves_public_test.go": "a word becoming knob positions, across the compiler",
	"pkg/sdk/internal/compile/roundtrip_public_test.go":   "a preset lifted and lowered back",
	"pkg/sdk/internal/compile/shipped_public_test.go":     "every rig that ships, compiled",
	"pkg/sdk/internal/device/device_public_test.go":       "the double the package's tests share",
	"pkg/sdk/internal/device/empty_slot_public_test.go":   "what an empty slot holds, behind a build tag",
	"pkg/sdk/internal/device/generate_test.go":            "the mockgen directives, and no tests",
	"pkg/sdk/internal/deviceslots/written_test.go":        "what a write puts on the wire, byte for byte",
	"pkg/sdk/internal/fileslots/fileslots_public_test.go": "listing and showing, which are one job in two files",
	"pkg/sdk/internal/wire/preset_shape_test.go":          "malformed wire data, across the decoder",
	"pkg/sdk/preset/fidelity_public_test.go":              "a preset read and written back unchanged",
	"pkg/mcp/coverage_public_test.go":                     "the command tree against the registered MCP tools",
	"pkg/cli/measure_preset_test.go":                      "a preset a reader can read, for the doubles that stand in for the compiler",
	"pkg/sdk/rig/coverage_public_test.go":                 "how much of the contract the shipped rigs use",
	"pkg/sdk/plan/coverage_public_test.go":                "whether every field a plan models is written down somewhere",
	"pkg/sdk/rig/shipped_public_test.go":                  "every rig that ships, validated",
	"pkg/sdk/tone/shipped_public_test.go":                 "every worked example, read with the loader it names",
}

// TestEveryTestFileIsNamedForWhatItCovers holds the rule CONTRIBUTING gives a
// test file's name.
//
// A test named for a file that does not exist sends the next reader looking
// for it, and hides that the thing it covers has no test of its own: three
// were named gate, contract and schema, for production files nobody ever
// wrote, while signal.go, validate.go and embed.go had no counterpart at all.
//
// Anything starting export_ is exempt: export_test.go is Go's own idiom for
// handing an unexported thing to an external test, and a package with several
// of them names each for the half it opens up.
func (s *MainTestSuite) TestEveryTestFileIsNamedForWhatItCovers() {
	var adrift []string

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			switch d.Name() {
			case ".git", ".worktrees", ".claude", "node_modules":
				return fs.SkipDir
			}

			return nil
		case !strings.HasSuffix(path, "_test.go"),
			strings.HasSuffix(path, ".gen_test.go"),
			strings.HasPrefix(d.Name(), "export_"),
			d.Name() == "main_test.go",
			d.Name() == "architecture_test.go":
			return nil
		}

		covers := strings.TrimSuffix(path, "_test.go")
		covers = strings.TrimSuffix(covers, "_public") + ".go"

		if _, err := os.Stat(covers); err == nil {
			return nil
		}

		if _, ok := coversAConcern[path]; !ok {
			adrift = append(adrift, path)
		}

		return nil
	})

	s.Require().NoError(err)
	s.Require().Empty(adrift,
		"named for no production file and not listed as covering a concern:\n%s",
		strings.Join(adrift, "\n"))
}

// TestEverySkillLinkResolves holds a skill to the repository it describes.
//
// A skill is instructions an agent follows without checking, so a dead link in
// one is worse than a dead link in a document somebody reads: nobody notices
// until an agent has already acted on the half it could reach.
//
// Written after finding two wrong commands in the first skill on the day it was
// written — `catalog search`, which does not exist, and `presets play --file`,
// which is `--preset`. Both looked right.
func (s *MainTestSuite) TestEverySkillLinkResolves() {
	pages, err := filepath.Glob(filepath.Join(".claude", "skills", "*", "**", "*.md"))
	s.Require().NoError(err)

	top, err := filepath.Glob(filepath.Join(".claude", "skills", "*", "*.md"))
	s.Require().NoError(err)

	pages = append(pages, top...)
	s.Require().NotEmpty(pages, "the skills moved, and this test did not")

	link := regexp.MustCompile(`\]\((\.\.?/[^)]+|references/[^)]+)\)`)

	for _, page := range pages {
		body, err := os.ReadFile(page) //nolint:gosec // a path this glob found
		s.Require().NoError(err)

		for _, m := range link.FindAllStringSubmatch(string(body), -1) {
			// An anchor is a heading rather than a file, and whether one
			// exists is not what this checks.
			at, _, _ := strings.Cut(m[1], "#")
			if at == "" {
				continue
			}

			_, err := os.Stat(filepath.Join(filepath.Dir(page), at))
			s.Require().NoError(err, "%s links to %s, which is not there", page, at)
		}
	}
}

// TestEveryCommandASkillNamesExists is the other half of the same rot.
//
// A skill that tells an agent to run a command the tool does not have sends it
// down a path that fails, and the failure looks like the tool is broken rather
// than like the instructions are.
func (s *MainTestSuite) TestEveryCommandASkillNamesExists() {
	pages, err := filepath.Glob(filepath.Join(".claude", "skills", "*", "**", "*.md"))
	s.Require().NoError(err)

	top, err := filepath.Glob(filepath.Join(".claude", "skills", "*", "*.md"))
	s.Require().NoError(err)

	named := regexp.MustCompile(`go run main\.go ([a-z]+(?: [a-z]+)?)`)
	known := map[string]bool{}

	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		at := strings.TrimSpace(path + " " + c.Name())
		known[at] = true

		for _, sub := range c.Commands() {
			walk(sub, at)
		}
	}

	for _, sub := range cmd.Root().Commands() {
		walk(sub, "")
	}

	for _, page := range append(pages, top...) {
		body, err := os.ReadFile(page) //nolint:gosec // a path this glob found
		s.Require().NoError(err)

		for _, m := range named.FindAllStringSubmatch(string(body), -1) {
			s.Require().True(known[m[1]],
				"%s tells an agent to run %q, which this tool does not have",
				page, m[1])
		}
	}
}

// TestTheCLIStandsAlone asserts the CLI half could be its own repository.
//
// The mirror of TestTheSDKStandsAlone, for the other end. main.go, cmd/ and
// the rendering are what a toneharness-cli would be, and the only thing it may
// reach in this module is the library it would import as a dependency.
//
// A generator was the thing that broke it. The two that build the catalog and
// the statistics sat in root internal/ and cmd/ called them directly, so the
// CLI could not have left without taking a build tool for a licensed HX Edit
// installation with it. They belong to the library, whose data they write.
func (s *MainTestSuite) TestTheCLIStandsAlone() {
	for _, pkg := range []string{".", "./cmd", "./pkg/cli/..."} {
		out, err := exec.Command("go", "list", "-deps", pkg).Output()
		s.Require().NoError(err)

		for _, dep := range strings.Fields(string(out)) {
			if !strings.HasPrefix(dep, mod) {
				continue
			}

			switch {
			case dep == mod, dep == mod+"cmd":
			case strings.HasPrefix(dep, mod+"pkg/cli"):
			case strings.HasPrefix(dep, mod+"pkg/mcp"):
			case strings.HasPrefix(dep, mod+"pkg/sdk"):
			default:
				s.Require().Fail("reaches too far",
					"%s reaches %s, which a toneharness-cli would not have",
					pkg, dep)
			}
		}
	}
}

// TestTheMCPStandsAlone holds the MCP server to the SDK, so it can leave for a
// toneharness-mcp repository without taking the CLI with it.
func (s *MainTestSuite) TestTheMCPStandsAlone() {
	out, err := exec.Command("go", "list", "-deps", "./pkg/mcp/...").Output()
	s.Require().NoError(err)

	for _, dep := range strings.Fields(string(out)) {
		if !strings.HasPrefix(dep, mod) {
			continue
		}

		switch {
		case strings.HasPrefix(dep, mod+"pkg/mcp"):
		case strings.HasPrefix(dep, mod+"pkg/sdk"):
		default:
			s.Require().
				Fail("reaches too far", "pkg/mcp reaches %s, which a toneharness-mcp would not have", dep)
		}
	}
}

// TestTheSDKReadsOneVariable holds the library to reading the environment in
// one place.
//
// A library that reads the environment is configured by whoever started the
// process rather than by whoever called it, and two Clients in one process
// cannot differ. So the CLI reads TONEHARNESS_USB_DUMP and TONEHARNESS_USB_DEBUG
// and passes options in. XDG_STATE_HOME is the exception: where state lives by
// default is the platform's convention, the way os.UserConfigDir is.
//
// Only non-test files. A test sets and reads what it likes.
func (s *MainTestSuite) TestTheSDKReadsOneVariable() {
	// What reads the environment, by import path.
	reads := map[string]map[string]bool{
		"os": {
			"Getenv": true, "LookupEnv": true, "Environ": true, "ExpandEnv": true,
		},
		"syscall": {"Getenv": true, "Environ": true},
	}

	fset := token.NewFileSet()

	var found []string

	err := filepath.WalkDir(filepath.Join("pkg", "sdk"),
		func(path string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir(),
				!strings.HasSuffix(path, ".go"),
				strings.HasSuffix(path, "_test.go"):
				return nil
			}

			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}

			// The name each watched package goes by in this file, which an
			// alias changes. A dot import leaves no name to find, so it is
			// reported on its own.
			named := map[string]string{}

			for _, imp := range f.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil || reads[path] == nil {
					continue
				}

				name := path
				if imp.Name != nil {
					name = imp.Name.Name
				}

				switch name {
				case "_":
				case ".":
					found = append(found, fmt.Sprintf("%s: dot import of %s",
						fset.Position(imp.Pos()), path))
				default:
					named[name] = path
				}
			}

			// A literal argument, where the reference is called with one.
			args := map[*ast.SelectorExpr]string{}

			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					sel, isSel := call.Fun.(*ast.SelectorExpr)
					if isSel && len(call.Args) == 1 {
						if lit, isLit := call.Args[0].(*ast.BasicLit); isLit {
							args[sel] = lit.Value
						}
					}
				}

				return true
			})

			// Every reference, called or not: a function value read into a
			// variable reads the environment when it is called later.
			ast.Inspect(f, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				path, ok := named[pkg.Name]
				if !ok || !reads[path][sel.Sel.Name] {
					return true
				}

				found = append(found, fmt.Sprintf("%s: %s.%s(%s)",
					fset.Position(sel.Pos()), path, sel.Sel.Name, args[sel]))

				return true
			})

			return nil
		})
	s.Require().NoError(err)

	s.Require().Len(found, 1,
		"pkg/sdk reads the environment in more places than one: %v; "+
			"read it in cmd and pass an option in", found)
	s.Require().Contains(found[0], `os.Getenv("XDG_STATE_HOME")`,
		"the one variable pkg/sdk reads is XDG_STATE_HOME")
}

func TestMainTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MainTestSuite))
}
