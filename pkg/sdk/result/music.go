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

package result

// What the music corpus holds, read off the manifests rather than the audio.
//
// A manifest read is a file read. Measuring is minutes of work per record and
// needs audio nobody may redistribute, so these answer what was written down:
// which players, which bands, which genres, and how far each genre is from
// being a distribution worth aiming at.

// MusicPlayer is one player's corpus, as their manifest describes it.
type MusicPlayer struct {
	// Instrument is the tree they sit in, which is what their records are
	// comparable within. Empty where the corpus given was one instrument's own
	// tree rather than the root above them all.
	Instrument string `json:"instrument"`
	// ID is the directory, which is the identifier a rig is joined by.
	ID string `json:"id"`
	// Artist is the name the manifest gives.
	Artist string `json:"artist"`
	// Records is how many recordings the manifest names.
	Records int `json:"records"`
	// Bands and Genres are what those records say they are, each once and in
	// order, so a player who changed band reads as having played in both.
	Bands  []string `json:"bands"`
	Genres []string `json:"genres"`
	// Untagged is how many of their records name no genre at all. A record
	// with none is invisible to every genre, which is worth seeing beside a
	// genre that is short.
	Untagged int `json:"untagged"`
	// Rig says whether any gear is known for them, joined on ID.
	//
	// A player with records and no rig is measured by nobody: their records
	// still earn a genre its words, and a request for that genre then has
	// nothing to build. Five sat here unnoticed until somebody asked why a
	// genre would not build, so the tool says it rather than a page.
	Rig bool `json:"rig"`
}

// MusicGroup is one genre or one band, and what backs it.
//
// The same type for both, because it is the same question asked twice: which
// records are this, and how many different players made them.
type MusicGroup struct {
	// Name is the word or the band, as the manifests spell it.
	Name string `json:"name"`
	// Slug is what a path or a flag says, and what grouping is done on, so two
	// spellings of one band count once.
	Slug string `json:"slug"`
	// Records is how many recordings carry it, and Artists how many different
	// players those come from.
	Records int `json:"records"`
	Artists int `json:"artists"`
	// Who are those players by name, in order. The count alone does not say
	// whether three players are three bands or one scene, and which they are
	// decides what to add next.
	Who []string `json:"who"`
	// Unsighted is how many of those records a model tagged and nobody checked.
	// Always zero for a band, which nobody labels.
	Unsighted int `json:"unsighted"`
	// Usable says whether this has a distribution worth aiming at: eight
	// records from at least three players. Meaningless for a band, and carried
	// anyway so one type serves both.
	Usable bool `json:"usable"`
	// Short says how many more records and players it needs to be usable.
	// Both zero once it is.
	ShortRecords int `json:"short_records"`
	ShortArtists int `json:"short_artists"`
	// Geared is how many of those players any gear is known for.
	//
	// A genre earns its words from records alone, so one can reach the
	// threshold and still be hollow: grunge earned "scooped" and "clean" from
	// nine records across three players, none of whom had a rig. Always zero
	// for a band, which nothing is built from, and carried anyway so one type
	// serves both.
	Geared int `json:"geared"`
}

// MusicRecord is one recording, with the player it was measured for.
type MusicRecord struct {
	// Instrument is the tree it sits in.
	Instrument string `json:"instrument"`
	// Player is the directory it sits in.
	Player string `json:"player"`
	// Track is the recording's name, which is also the stem directory.
	Track string `json:"track"`
	// Year is when it was made.
	Year int `json:"year"`
	// Band is who made it, where the player was in one.
	Band string `json:"band"`
	// Genres are what it is, and DecidedBy who said so.
	Genres    []string `json:"genres"`
	DecidedBy string   `json:"decided_by"`
	// Separated says whether the bass has been split out of it yet. A record
	// nobody separated is named in the manifest and measured by nothing.
	Separated bool `json:"separated"`
}
