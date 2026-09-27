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

package audio

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/retr0h/toneharness/pkg/sdk/internal/slug"
)

// A corpus is audio nobody may redistribute and a record of what it was.
//
// The audio cannot be committed: it is somebody else's, and a repository is
// not a way around that. The record of which songs were measured can be, and
// it is the half that matters to anybody reading a rig afterwards. Mirrors
// how the preset corpus works, where the fetch script and the attribution are
// committed and the payload is not.
//
// So the manifest names tracks and links them. It never names a file.

// Record names one recording and where somebody else can find it.
type Record struct {
	// Track is the recording's name, matching the stem directory measured.
	Track string `yaml:"track"`
	// URL is where to hear it. Not where the audio lives on disk: a link is
	// checkable by somebody who does not have the file.
	//
	// Required. Every figure measured from a record travels into a rig as
	// evidence, and evidence with no source is an assertion.
	URL string `yaml:"url"`
	// Source is where the audio was actually fetched from, when the url alone
	// could not fetch it. Evidence stays with URL, which is the link a person
	// checks; this is only the downloader's way back to the same recording.
	//
	// Both Flea records failed on every Spotify link tried, and the pair that
	// worked lived in one person's shell history until it was written here.
	Source string `yaml:"source"`
	// Year is when the record was made, which is what holds it to the rig it
	// is measured for.
	//
	// Required. A rig's gear claims describe a period, and a record from
	// another one measures another rig: an Acoustic 360 cannot be on a 1966
	// record because Acoustic had not built one.
	Year int `yaml:"year"`
	// At is the part measured, as "1:20" or "1:20-1:45".
	At string `yaml:"at"`
	// Note is anything worth saying about this recording in particular.
	Note string `yaml:"note"`
	// Band is who made the record, where a player was in one.
	//
	// On the record rather than on the player, for the reason the genres below
	// are: a bassist plays in several bands over a career and a record belongs
	// to one. Attaching it to the player would put Parliament-Funkadelic on
	// every Bootsy Collins record, including the ones it was not.
	//
	// Written as somebody writes it, and grouped on its slug, so "Guns N' Roses"
	// and "Guns n Roses" are one band rather than two.
	Band string `yaml:"band"`
	// Genres are what this recording is, as words rather than one of a fixed
	// list.
	//
	// Per track rather than per artist, because a catalogue spans them: the same
	// player is on a punk record and a funk one, and tagging the artist would
	// put both sounds in both distributions. A track may carry several, since
	// pop-punk is punk and saying so twice is cheaper than deciding which one
	// it "really" is.
	//
	// Not an enum. Enumerating genres in code means a release to add one, and
	// what decides whether a genre works is not the type: it is whether enough
	// records carry it to have a distribution. See Genres.
	Genres []string `yaml:"genres"`
	// GenresBy is who decided the genres above: ByModel or ByPerson.
	//
	// Required wherever genres are, because a tag nobody checked and a tag
	// somebody overruled are worth different amounts and look identical. One
	// value for the record rather than one per term: a track is labelled in a
	// single act, and nobody takes punk from a model and pop-punk from a person
	// on the same song.
	GenresBy Decided `yaml:"genres_by"`
}

// Decided is who put a genre on a record.
type Decided string

const (
	// ByModel is a genre a language model asserted, which is the fast path and
	// the default. Checked by nobody, so these are the ones to review.
	ByModel Decided = "llm"
	// ByPerson is a genre somebody stated, or one they overruled a model on.
	// That ends the argument; nothing recomputes it.
	ByPerson Decided = "person"
)

// Manifest is a corpus of recordings, without the recordings.
type Manifest struct {
	// Artist is whose playing this corpus is of.
	Artist string `yaml:"artist"`
	// Tracks is what was measured, one entry per recording.
	Tracks []Record `yaml:"tracks"`
}

// Grouped is one name the corpus can be selected by, and what backs it.
//
// A genre and a band are the same question asked twice: how many records carry
// this, and how many different players do they come from. So they answer with
// one type rather than two that drift.
type Grouped struct {
	// Name is the word or the band, as the manifests spell it.
	Name string `json:"name"`
	// Slug is what a path or a flag says, which is what grouping is done on.
	Slug string `json:"slug"`
	// Records is how many recordings carry it, and Who the players those come
	// from, in order. The count is Artists, which is len(Who) and kept because
	// the threshold is stated as a number.
	Records int      `json:"records"`
	Artists int      `json:"artists"`
	Who     []string `json:"who"`
	// Unsighted is how many of those records a model tagged and nobody checked.
	//
	// Reported rather than deducted. A genre reaching the threshold entirely on
	// a model's guesses still reaches it, and somebody should know that before
	// aiming at it.
	Unsighted int `json:"unsighted"`
}

// Usable reports whether this genre has a distribution worth aiming at.
//
// Eight records from at least three players. Three records by one band is that
// band's sound wearing a genre's name, and a request for the genre would get the
// band: the figures cannot tell the two apart, so the threshold is the only
// thing that can.
//
// A genre under it is reported rather than computed from, which is why this is a
// question and not a filter.
func (n Grouped) Usable() bool { return n.Records >= 8 && n.Artists >= 3 }

// Genres is every genre the manifests name, with what backs each one.
//
// In name order, so two runs read the same. The counts are what decides whether
// a genre can be aimed at, and reporting them is the honest answer while a
// corpus is being built: somebody adding records needs to see how far off the
// threshold each one still is.
func Genres(
	all []Manifest,
) []Grouped {
	return grouped(all,
		func(rec Record) []string { return rec.Genres },
		func(rec Record) bool { return rec.GenresBy == ByModel })
}

// Bands is every band the manifests name, with what backs each one.
//
// The same shape as Genres, because it is the same question: which records are
// this, and how many players made them. Grouped on the slug so two spellings of
// one band are one band.
func Bands(
	all []Manifest,
) []Grouped {
	// A band is not a claim anybody labels, so nothing here is ever unsighted.
	return grouped(all, func(rec Record) []string {
		if rec.Band == "" {
			return nil
		}

		return []string{rec.Band}
	}, func(Record) bool { return false })
}

// grouped counts records and players for whatever a record says it is.
//
// Keyed on the slug and reported under the spelling first seen, so a name typed
// two ways counts once and still reads as somebody wrote it.
func grouped(
	all []Manifest,
	of func(Record) []string,
	unchecked func(Record) bool,
) []Grouped {
	type count struct {
		name      string
		records   int
		unsighted int
		players   map[string]bool
	}

	seen := map[string]*count{}

	for _, m := range all {
		for _, rec := range m.Tracks {
			for _, name := range of(rec) {
				key := slug.Of(name)

				if seen[key] == nil {
					seen[key] = &count{name: name, players: map[string]bool{}}
				}

				seen[key].records++
				seen[key].players[m.Artist] = true

				if unchecked(rec) {
					seen[key].unsighted++
				}
			}
		}
	}

	out := make([]Grouped, 0, len(seen))

	for key, c := range seen {
		who := make([]string, 0, len(c.players))
		for name := range c.players {
			who = append(who, name)
		}

		sort.Strings(who)

		out = append(out, Grouped{
			Name: c.name, Slug: key, Records: c.records, Artists: len(who),
			Who: who, Unsighted: c.unsighted,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })

	return out
}

// Where a timestamp is allowed to point, and what a link has to look like.
//
// The same shapes the rig contract accepts, checked here so a bad one is
// caught while somebody is looking at the manifest. Left until build time it
// surfaces as a validation failure against `chain[0].evidence[1].at`, which
// says nothing about which song was wrong.
var (
	atPattern  = regexp.MustCompile(`^\d{1,2}:\d{2}(:\d{2})?(-\d{1,2}:\d{2}(:\d{2})?)?$`)
	urlPattern = regexp.MustCompile(`^https?://\S+$`)

	// Where a record is allowed to come from.
	//
	// `url` is the evidence a rig quotes, and it names the recording rather
	// than a copy of it: a Spotify track link identifies one master, which is
	// what tells the album take apart from the live one and the remaster. A
	// link to anywhere else names a file, and a file is not a record.
	//
	// `source` is only the downloader's way back to the same audio when the
	// url alone failed, so YouTube belongs there and nowhere else. `just
	// record` runs spotdl, which reads the song from Spotify and fetches
	// audio from YouTube, so these two are also the only hosts it can use.
	//
	// The corpus was already all Spotify and YouTube when this was added.
	// The check is here to keep it that way, because the thing it prevents,
	// a plausible link off a search page standing in for the record, is
	// invisible once the audio is on disk and measuring fine.
	urlHosts    = []string{"open.spotify.com"}
	sourceHosts = []string{"open.spotify.com", "www.youtube.com", "youtu.be", "music.youtube.com"}
)

// hosted reports whether a link points at one of the hosts given.
func hosted(
	link string,
	hosts []string,
) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	return slices.Contains(hosts, u.Host)
}

// ReadManifest reads a corpus manifest.
//
// Unknown fields are refused rather than ignored. A manifest is written by
// hand, `track:` and `tracks:` are one letter apart, and a typo that silently
// measures nothing is worse than one that stops.
func ReadManifest(
	r io.Reader,
) (Manifest, error) {
	var m Manifest

	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("reading manifest: %w", err)
	}

	for _, rec := range m.Tracks {
		if err := rec.check(); err != nil {
			return Manifest{}, err
		}
	}

	return m, nil
}

// check holds one entry to what a rig will accept from it.
func (r Record) check() error {
	if r.Track == "" {
		return fmt.Errorf("reading manifest: an entry has no track name")
	}

	// A record with no link is a claim nobody else can check. The figures
	// measured from it end up in a rig as evidence, and evidence whose
	// source is "a file on somebody's disk" is worth no more than an
	// assertion. Three players had three tracks each and no links between
	// them before this was refused.
	if r.Year == 0 {
		return fmt.Errorf("reading manifest: %s has no year, and a record that "+
			"names no year cannot be held to the era of the rig it measures", r.Track)
	}

	if r.URL == "" {
		return fmt.Errorf("reading manifest: %s has no url, and a record nobody "+
			"can find is a measurement nobody can check", r.Track)
	}

	if !urlPattern.MatchString(r.URL) {
		return fmt.Errorf(
			"reading manifest: %s has a url that is not a link: %q", r.Track, r.URL)
	}

	if !hosted(r.URL, urlHosts) {
		return fmt.Errorf("reading manifest: %s has a url that is not a Spotify "+
			"track: %q. The url names which recording was measured, and only a "+
			"track link does that", r.Track, r.URL)
	}

	if r.Source != "" && !urlPattern.MatchString(r.Source) {
		return fmt.Errorf(
			"reading manifest: %s has a source that is not a link: %q", r.Track, r.Source)
	}

	if r.Source != "" && !hosted(r.Source, sourceHosts) {
		return fmt.Errorf("reading manifest: %s has a source that is not Spotify "+
			"or YouTube: %q. A source is what spotdl fetches the audio from, and "+
			"it reads those two", r.Track, r.Source)
	}

	if err := r.checkGenres(); err != nil {
		return err
	}

	if r.At != "" && !atPattern.MatchString(r.At) {
		return fmt.Errorf(
			`reading manifest: %s has an at that is not a timestamp: %q, wanted "1:20" or "1:20-1:45"`,
			r.Track,
			r.At,
		)
	}

	return nil
}

// checkGenres holds a tag to saying who decided it.
//
// Refused here rather than reported later, for the reason a missing url is: a
// genre with no provenance reads exactly like a sourced one, and the review
// list that exists to catch a model's guess cannot see it.
func (r Record) checkGenres() error {
	switch {
	case len(r.Genres) == 0 && r.GenresBy == "":
		return nil
	case len(r.Genres) == 0:
		return fmt.Errorf("reading manifest: %s says genres_by %q and names no "+
			"genres for it to have decided", r.Track, r.GenresBy)
	case r.GenresBy == "":
		return fmt.Errorf("reading manifest: %s names genres and no genres_by, "+
			"so nothing says whether a model guessed them or somebody checked. "+
			"Wanted %q or %q", r.Track, ByModel, ByPerson)
	case r.GenresBy != ByModel && r.GenresBy != ByPerson:
		return fmt.Errorf("reading manifest: %s has a genres_by of %q, wanted "+
			"%q or %q", r.Track, r.GenresBy, ByModel, ByPerson)
	}

	return nil
}

// Join attaches what the manifest knows to what was measured.
//
// Matched on the recording's name, which for a separated stem is the
// directory holding it and so is the source file's own name. Anything the
// manifest does not mention keeps an empty source rather than being dropped:
// a measurement without a link is still a measurement.
func (m Manifest) Join(
	of []Named,
) []Named {
	out := make([]Named, 0, len(of))

	for _, n := range of {
		for _, rec := range m.Tracks {
			if strings.EqualFold(rec.Track, n.Name) {
				n.Source = rec

				break
			}
		}

		out = append(out, n)
	}

	return out
}

// Unmatched is what the manifest and the recordings disagree about.
//
// Both directions, because both are mistakes somebody wants told. A track
// named in the manifest with nothing measured usually means the separation
// did not run on it; a recording nothing names is one whose evidence will go
// out with no link on it.
func (m Manifest) Unmatched(
	of []Named,
) (missing, unnamed []string) {
	measured := make(map[string]bool, len(of))
	for _, n := range of {
		measured[strings.ToLower(n.Name)] = true
	}

	named := make(map[string]bool, len(m.Tracks))

	for _, rec := range m.Tracks {
		named[strings.ToLower(rec.Track)] = true

		if !measured[strings.ToLower(rec.Track)] {
			missing = append(missing, rec.Track)
		}
	}

	for _, n := range of {
		if !named[strings.ToLower(n.Name)] {
			unnamed = append(unnamed, n.Name)
		}
	}

	return missing, unnamed
}
