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
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// shownFile is where HX Edit keeps the names it puts in front of a person.
const shownFile = "HX_ModelCatalog.json"

// Renamed is one model the two Line 6 files do not agree about.
type Renamed struct {
	// Model is the identifier both files name.
	Model catalog.ModelID
	// Was is what the .models file calls it, and Now what the app shows.
	Was string
	Now string
	// Took says the app's name is the one that shipped. False where the
	// disagreement is reported and the `.models` name kept, which is most of
	// them: see `wrong`.
	Took bool
}

// shown is what Line 6's own app says about a model, beside what the model
// definition says.
//
// Two facts so far, and they are here together because they come out of one
// read of one file. A third would too.
type shown struct {
	// Name is what the app puts in front of a person.
	Name string
	// Bass is whether the app marks this model as one for bass. Stated here
	// and nowhere else: the model definitions carry no such field, so a
	// cabinet's instrument is knowable only from this file.
	Bass bool
}

// readShown reads what Line 6's own app says about each model.
//
// Line 6 ship every name twice. The `.models` files carry one and this file
// carries another, and for 24 of the 654 models in both they disagree. Twenty
// of those are spacing or an abbreviation. Four are not, and one of them names
// the wrong gear: `HD2_CabMicIr_2x12MatchG25` is "2x12 Match H30" in
// cabmicirs.models and "2x12 Match G25" here, and a G25 and an H30 are
// different speakers. The other three are misspellings the app has right:
// Kinectic for Kinetic, Dyhana for Dhyana, Aqua for Agua.
//
// This file wins for the four, and only the four, named in `wrong` below. A
// name is what a document may say, because the contract's `gear` field refuses
// an identifier, so a model named after a different cabinet is a cabinet
// nothing can ask for correctly.
//
// Not for the other twenty-one, which look cosmetic to a reader and are not
// cosmetic to the matcher. Taking all of them renamed `HD2_AmpSVT4Pro` from
// "SVT-4 Pro" to "Ampeg SVT-4 PRO", and `Ampeg SVT` then fit six of the
// device's models where it had fit four, because the longer name begins with
// the shorter one. That is a change to what every rig naming an Ampeg
// resolves to, which is not a rename and is not what this set out to do.
//
// The rest are reported instead. A disagreement nobody can see is how
// twenty-five of them survived into the shipped catalog.
//
// The same file is the only statement of which models are for bass, which is
// read here rather than in a second pass over the same tree. See [shown.Bass].
//
// Absent on an installation too old to have it, which is not fatal: the
// `.models` names are what this used to ship and they are wrong in four
// places rather than useless. A catalog built without it marks nothing for
// bass, so a ranking that filters on it has nothing to filter by and keeps
// everything, which is the behaviour that shipped before the field existed.
func readShown(
	dir string,
) (map[catalog.ModelID]shown, error) {
	body, err := os.ReadFile(
		filepath.Join(dir, shownFile),
	) //nolint:gosec // the app's own directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading %s: %w", shownFile, err)
	}

	var tree any
	if err := json.Unmarshal(body, &tree); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", shownFile, err)
	}

	out := map[catalog.ModelID]shown{}
	collect(tree, out)

	return out, nil
}

// wrong are the models whose `.models` name says something untrue about the
// gear, where the app's name is taken over it.
//
// A named set because nothing in the data can tell these from the other
// twenty-one. "4x12 Greenback30" against "4x12 Greenback 30" is a space, and
// which side of that is right is nobody's business; "2x12 Match H30" on a G25
// is a different speaker, and "Dyhana" is not a word. That is a judgement
// about names rather than a rule a file could carry, so it is written down.
//
// One of these matters and three are tidiness. The cabinet is the one that
// matters: `2x12 Match H30` reaches five models, two of them G25s, and both
// sit in the `cabmicirs` family so `catalog.Preferred` ties them and the
// identifier breaks the tie. `…MatchG25` sorts first, so a rig asking for the
// H30 got the G25. The other three are misspellings nobody can search for.
var wrong = map[catalog.ModelID]bool{
	"HD2_CabMicIr_2x12MatchG25":        true,
	"HD2_CabMicIr_2x12MatchG25WithPan": true,
	"HD2_AmpLine6Kinetic":              true,
	"HD2_DistDhyanaDrive":              true,
	"HD2_PreampAguaSledge":             true,
}

// collect walks the catalog for every object carrying an id and a name.
//
// Walked rather than decoded into a shape, because the file nests models under
// categories under devices and the nesting is Line 6's to change. What every
// release has in common is that a model is an object with an `id` and a
// `name`, so that is what this looks for. The first name wins, which is the
// one nearest the top of the file.
func collect(
	of any,
	into map[catalog.ModelID]shown,
) {
	switch v := of.(type) {
	case map[string]any:
		id, okID := v["id"].(string)
		name, okName := v["name"].(string)

		if okID && okName && id != "" && name != "" {
			if _, held := into[catalog.ModelID(id)]; !held {
				bass, _ := v["bass"].(bool)
				into[catalog.ModelID(id)] = shown{Name: name, Bass: bass}
			}
		}

		// In key order, because Go's decoder throws an object's own order
		// away and map iteration is deliberately random. The rule above is
		// first wins, so without this the name a duplicated id gets is a
		// coin flip and two runs of `just generate` write two catalogs.
		//
		// Line 6 spell four model names wrong and #249 chose between the two
		// spellings deliberately; a random walk would undo that choice at
		// random. Arrays keep the file's own order, which is why only this
		// branch sorts.
		for _, key := range slices.Sorted(maps.Keys(v)) {
			collect(v[key], into)
		}
	case []any:
		for _, each := range v {
			collect(each, into)
		}
	}
}
