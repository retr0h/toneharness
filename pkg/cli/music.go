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

package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/toneharness/pkg/cli/internal/paint"
	sdk "github.com/retr0h/toneharness/pkg/sdk"
)

// MusicPlayers prints every player the corpus names.
func MusicPlayers(
	w io.Writer,
	of []sdk.MusicPlayer,
) error {
	rows := make([][]string, 0, len(of))
	tagged, geared := 0, 0

	for _, p := range of {
		if p.Untagged == 0 {
			tagged++
		}

		if p.Rig {
			geared++
		}

		rows = append(rows, []string{
			paint.Accent(w, p.ID),
			strconv.Itoa(p.Records),
			rigOf(w, p),
			listOf(w, p.Bands, "none named"),
			genresOf(w, p),
		})
	}

	return paint.Section{
		Title:   "Who the corpus holds records for",
		Detail:  playersHeld(len(of), tagged, geared),
		Headers: []string{"player", "records", "rig", "bands", "genres"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Right, lipgloss.Left,
			lipgloss.Left, lipgloss.Left,
		},
		Empty: "no players in this corpus",
		Summary: "the directory is the rig's identifier, and a rig named " +
			"anything else is measured by nobody",
	}.Render(w)
}

// rigOf says whether there is gear for a player.
//
// The absence is what the column is for, so that is the one marked. A player
// with records and no rig earns their genres words that nothing can then be
// built from.
func rigOf(
	w io.Writer,
	p sdk.MusicPlayer,
) string {
	if p.Rig {
		return paint.Mute(w, "yes")
	}

	return paint.Info(w, "none")
}

// playersHeld says how many players are held, how many are fully tagged, and
// how many have gear.
//
// The players with no rig are named as a count rather than left to be counted
// off the table, because that is the number somebody is looking for when a
// genre will not build.
func playersHeld(
	players, tagged, geared int,
) string {
	got := Plural(players, "player")

	if players != tagged {
		got += fmt.Sprintf(", %d with every record tagged", tagged)
	} else {
		got += ", every record carrying a genre"
	}

	if geared == players {
		return got + ", all with gear"
	}

	return got + fmt.Sprintf(", %d with no rig", players-geared)
}

// genresOf lists a player's genres, saying how many of their records carry none.
func genresOf(
	w io.Writer,
	p sdk.MusicPlayer,
) string {
	got := listOf(w, p.Genres, "none tagged")
	if p.Untagged == 0 {
		return got
	}

	return got + " " + paint.Info(w, fmt.Sprintf("(%d untagged)", p.Untagged))
}

// MusicGroups prints every genre or band, and what backs each one.
//
// One renderer for both, because the table is the same: a name, how many
// records carry it, and how many players those come from. What differs is
// whether the threshold means anything, which is what `threshold` decides.
func MusicGroups(
	w io.Writer,
	of []sdk.MusicGroup,
	kind string,
	threshold bool,
) error {
	rows := make([][]string, 0, len(of))
	usable, hollow := 0, 0

	for _, g := range of {
		if g.Usable {
			usable++
		}

		if g.Geared == 0 {
			hollow++
		}

		row := []string{
			paint.Accent(w, g.Name),
			strconv.Itoa(g.Records),
			strconv.Itoa(g.Artists),
		}

		if threshold {
			row = append(row, gearedOf(w, g), checkedOf(w, g), readsOf(w, g))
		}

		// Last, because it is the widest and the only one that wraps.
		row = append(row, whoOf(w, g))

		rows = append(rows, row)
	}

	headers := []string{kind, "records", "players"}
	align := []lipgloss.Position{lipgloss.Left, lipgloss.Right, lipgloss.Right}

	if threshold {
		headers = append(headers, "geared", "checked", "reads")
		align = append(align, lipgloss.Left, lipgloss.Left, lipgloss.Left)
	}

	headers = append(headers, "who")
	align = append(align, lipgloss.Left)

	return paint.Section{
		Title:   "What the corpus can be selected by",
		Detail:  groupsHeld(len(of), usable, hollow, kind, threshold),
		Headers: headers,
		Rows:    rows,
		Align:   align,
		Empty:   "no " + kind + "s named in this corpus",
		Summary: groupsSummary(threshold),
	}.Render(w)
}

// gearedOf says how many of a genre's players there is gear for.
//
// Marked when it is none, because a genre with no geared player is one that
// earns its words and then cannot be built: grunge reached the threshold on
// nine records from three players and had gear for none of them.
func gearedOf(
	w io.Writer,
	g sdk.MusicGroup,
) string {
	got := fmt.Sprintf("%d of %d", g.Geared, g.Artists)

	if g.Geared == 0 {
		return paint.Info(w, got)
	}

	return paint.Mute(w, got)
}

// whoOf names the players a group's records come from, with the count.
//
// Named rather than counted. Three players is the threshold, and whether those
// three are three bands or one scene is what decides which record to add next;
// the number alone says neither.
func whoOf(
	w io.Writer,
	g sdk.MusicGroup,
) string {
	if len(g.Who) == 0 {
		return paint.Mute(w, "nobody")
	}

	return paint.Mute(w, strings.Join(g.Who, ", "))
}

// groupsHeld says how many were found, and how many can be aimed at.
func groupsHeld(
	found, usable, hollow int,
	kind string,
	threshold bool,
) string {
	if !threshold {
		return Plural(found, kind)
	}

	return fmt.Sprintf("%s, %d worth aiming at, %d with gear for nobody",
		Plural(found, kind), usable, hollow)
}

// groupsSummary says what the table is for.
func groupsSummary(
	threshold bool,
) string {
	if !threshold {
		return "grouped on the slug, so two spellings of one band count once"
	}

	return "eight records from three players before a genre is a " +
		"distribution: fewer is one band's sound wearing a genre's name"
}

// readsOf says whether a genre can be aimed at, or what it is short of.
func readsOf(
	w io.Writer,
	g sdk.MusicGroup,
) string {
	if g.Usable {
		return paint.OK(w, "a genre")
	}

	short := make([]string, 0, 2)
	if g.ShortRecords > 0 {
		short = append(short, Plural(g.ShortRecords, "record")+" short")
	}

	if g.ShortArtists > 0 {
		short = append(short, Plural(g.ShortArtists, "player")+" short")
	}

	return paint.Info(w, strings.Join(short, ", "))
}

// checkedOf says how many of a genre's records somebody checked.
//
// A genre can reach the threshold entirely on a model's guesses. That still
// reaches it, and somebody aiming at it should know.
func checkedOf(
	w io.Writer,
	g sdk.MusicGroup,
) string {
	checked := g.Records - g.Unsighted

	switch {
	case g.Unsighted == 0:
		return paint.OK(w, "all")
	case checked == 0:
		return paint.Err(w, "none")
	default:
		return paint.Info(w, fmt.Sprintf("%d of %d", checked, g.Records))
	}
}

// MusicRecords prints every recording the corpus names.
func MusicRecords(
	w io.Writer,
	of []sdk.MusicRecord,
) error {
	rows := make([][]string, 0, len(of))
	ready := 0

	for _, r := range of {
		if r.Separated {
			ready++
		}

		rows = append(rows, []string{
			paint.Accent(w, r.Player),
			r.Track,
			strconv.Itoa(r.Year),
			r.Band,
			listOf(w, r.Genres, "none"),
			decidedOf(w, r),
			stemsOf(w, r),
		})
	}

	return paint.Section{
		Title:   "Every record the corpus names",
		Detail:  fmt.Sprintf("%s, %d separated", Plural(len(of), "record"), ready),
		Headers: []string{"player", "track", "year", "band", "genres", "tagged by", "stems"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Left, lipgloss.Right, lipgloss.Left,
			lipgloss.Left, lipgloss.Left, lipgloss.Left,
		},
		Empty: "no records in this corpus",
		Summary: "a record with no stems is named here and measured by " +
			"nothing: separate it with `just stems`",
	}.Render(w)
}

// decidedOf says who put the genres on a record.
func decidedOf(
	w io.Writer,
	r sdk.MusicRecord,
) string {
	switch r.DecidedBy {
	case "":
		return paint.Mute(w, "—")
	case "person":
		return paint.OK(w, "a person")
	default:
		return paint.Info(w, "a model")
	}
}

// stemsOf says whether the bass has been split out yet.
func stemsOf(
	w io.Writer,
	r sdk.MusicRecord,
) string {
	if r.Separated {
		return paint.OK(w, "yes")
	}

	return paint.Err(w, "no")
}

// listOf joins what a record says it is, or says nothing said it.
func listOf(
	w io.Writer,
	of []string,
	none string,
) string {
	if len(of) == 0 {
		return paint.Mute(w, none)
	}

	return strings.Join(of, ", ")
}

// MeasuredGenres prints what each genre measured as against the rest.
//
// Beside MusicGroups rather than folded into it. That one counts what the
// manifests say, which is a file read; this reports a measurement over the
// audio, and the columns that matter are the words earned and how far past the
// line each sits.
func MeasuredGenres(
	w io.Writer,
	of []sdk.MeasuredGenre,
) error {
	rows := make([][]string, 0, len(of))
	earning := 0

	for _, g := range of {
		if len(g.Terms) > 0 {
			earning++
		}

		rows = append(rows, []string{
			paint.Accent(w, g.Name),
			strconv.Itoa(g.Records),
			strconv.Itoa(g.Players),
			strconv.Itoa(g.Against),
			aimedAt(w, g),
			earnedOf(w, g),
		})
	}

	return paint.Section{
		Title:   "What each genre measured as",
		Detail:  measuredHeld(len(of), earning),
		Headers: []string{"genre", "records", "players", "against", "reads", "earns"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Right, lipgloss.Right, lipgloss.Right,
			lipgloss.Left, lipgloss.Left,
		},
		Empty: "no records carry a genre",
		Summary: "a genre earns a word where its middle sits outside the " +
			"middle half of the players who play none of it, and the margin " +
			"says how far past the line",
	}.Render(w)
}

// measuredHeld says how many were measured and how many earned anything.
//
// The two are not the same, and the gap is the finding: a genre can clear the
// record threshold and still sit inside the middle half on every axis.
func measuredHeld(
	found, earning int,
) string {
	return fmt.Sprintf("%s measured, %d earning a word",
		Plural(found, "genre"), earning)
}

// aimedAt says whether enough backs a genre to compute from it.
func aimedAt(
	w io.Writer,
	g sdk.MeasuredGenre,
) string {
	if !g.Usable {
		return paint.Info(w, "not a genre yet")
	}

	if len(g.Terms) == 0 {
		// Enough behind it, and nothing to aim at. The honest answer, and the
		// one somebody asking for the genre needs.
		return paint.Err(w, "sets nothing apart")
	}

	return paint.OK(w, "a genre")
}

// earnedOf names the words a genre earned, with how far past the line each is.
func earnedOf(
	w io.Writer,
	g sdk.MeasuredGenre,
) string {
	if len(g.Terms) == 0 {
		return paint.Mute(w, "—")
	}

	out := make([]string, 0, len(g.Terms))

	for _, t := range g.Terms {
		out = append(out, fmt.Sprintf("%s %s", t.Term,
			paint.Mute(w, fmt.Sprintf("(%.2f v %.2f)", t.Mine, t.Others))))
	}

	return strings.Join(out, "  ")
}
