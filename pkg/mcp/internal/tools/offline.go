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

package tools

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

func (h *handlers) catalogList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Search,
) (*gomcp.CallToolResult, sdk.Blocks, error) {
	found, err := h.client.Blocks(ctx, sdk.Filter{
		Category:    in.Category,
		Subcategory: in.Subcategory,
		Search:      in.Search,
	})
	if err != nil {
		return nil, sdk.Blocks{}, err
	}

	return said("%d of %d blocks matched", len(found.Matched), found.Total), found, nil
}

func (h *handlers) catalogShow(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, catalog.Block, error) {
	block, err := h.client.Block(ctx, in.ID)
	if err != nil {
		return nil, catalog.Block{}, remedy(err)
	}

	return said("%s is %s", block.ID, block.Name), block, nil
}

func (h *handlers) corpusPresetsShow(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, Model, error) {
	measured, err := h.client.ModelMeasurements(ctx, in.ID)
	if err != nil {
		return nil, Model{}, err
	}

	stats := measured.Stats.Models[measured.Model]
	block, found := measured.Catalog.Block(measured.Model)
	if !found {
		return nil, Model{}, notInCatalog(string(measured.Model))
	}

	out := Model{Block: block, Uses: stats.Uses, Params: stats.Params}

	return said("%s is used %d times across the measured presets", in.ID, stats.Uses), out, nil
}

func (h *handlers) rigsList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Rigs, error) {
	found, err := h.client.Rigs(ctx)
	if err != nil {
		return nil, sdk.Rigs{}, err
	}

	return said("%d rigs to build from", len(found.Rigs)), found, nil
}

func (h *handlers) rigsShow(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, sdk.Rig, error) {
	found, err := h.client.Rig(ctx, in.ID)
	if err != nil {
		return nil, sdk.Rig{}, remedy(err)
	}

	return said("rig %s, extended by %d others", in.ID, len(found.Variants)), found, nil
}

func (h *handlers) toneBuild(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Asked,
) (*gomcp.CallToolResult, sdk.Resolved, error) {
	got, err := h.client.Tone(ctx, sdk.Ask{Spec: in.Spec, Setup: in.Setup})
	if err != nil {
		// The notes still travel. A request that could not be honoured has
		// usually said why in them, and the error alone is the half that does
		// not help.
		return nil, got, remedy(err)
	}

	return said("%d blocks, %d notes", len(got.Rig.Chain), len(got.Notes)), got, nil
}

func (h *handlers) presetsMake(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Make,
) (*gomcp.CallToolResult, Outcome, error) {
	switch {
	case in.RigID != "" && in.RigPath != "":
		return nil, Outcome{}, ErrTwoSources
	case in.RigID == "" && in.RigPath == "":
		return nil, Outcome{}, ErrNoSource
	}

	if err := h.mayWrite(in.Out); err != nil {
		return nil, Outcome{}, err
	}

	switch {
	case in.RigID != "":
		made, err := h.client.Make(ctx, in.RigID, in.Out, h.existing())
		if err != nil {
			return nil, Outcome{}, remedy(h.refused(in.Out, err))
		}

		return said("wrote %s from rig %s", in.Out, in.RigID), Outcome{FromShipped: &made}, nil
	default:
		built, err := h.client.Compile(ctx, sdk.Compile{
			Rig:      in.RigPath,
			Out:      in.Out,
			Existing: h.existing(),
		})
		if err != nil {
			return nil, Outcome{}, h.refused(in.Out, err)
		}

		return said("wrote %s from %s", in.Out, in.RigPath), Outcome{FromRig: &built}, nil
	}
}

func (h *handlers) corpusMusicPlayers(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Corpus,
) (*gomcp.CallToolResult, []sdk.MusicPlayer, error) {
	found, err := h.client.MusicPlayers(ctx, in.Corpus)
	if err != nil {
		return nil, nil, err
	}

	return said("%d players in %s", len(found), in.Corpus), found, nil
}

func (h *handlers) corpusMusicBands(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Corpus,
) (*gomcp.CallToolResult, []sdk.MusicGroup, error) {
	found, err := h.client.MusicBands(ctx, in.Corpus)
	if err != nil {
		return nil, nil, err
	}

	return said("%d bands made the records in %s", len(found), in.Corpus), found, nil
}

func (h *handlers) corpusMusicGenres(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Corpus,
) (*gomcp.CallToolResult, []sdk.MusicGroup, error) {
	found, err := h.client.MusicGenres(ctx, in.Corpus)
	if err != nil {
		return nil, nil, err
	}

	return said("%d genres are tagged in %s", len(found), in.Corpus), found, nil
}

func (h *handlers) corpusMusicRecords(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Corpus,
) (*gomcp.CallToolResult, []sdk.MusicRecord, error) {
	found, err := h.client.MusicRecords(ctx, in.Corpus)
	if err != nil {
		return nil, nil, err
	}

	return said("%d records in %s", len(found), in.Corpus), found, nil
}

func (h *handlers) corpusPresetsChains(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Instrument,
) (*gomcp.CallToolResult, sdk.Measured, error) {
	found, err := h.client.ChainMeasurements(ctx, in.Instrument)
	if err != nil {
		return nil, sdk.Measured{}, err
	}

	return said("what %s chains hold, across the measured presets", in.Instrument), found, nil
}

func (h *handlers) rigsRecords(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Corpus,
) (*gomcp.CallToolResult, []sdk.Backing, error) {
	found, err := h.client.Backing(ctx, in.Corpus)
	if err != nil {
		return nil, nil, err
	}

	return said("%d rigs held to the era their ask gives", len(found)), found, nil
}

func (h *handlers) measureGenres(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Corpus,
) (*gomcp.CallToolResult, []audio.Genre, error) {
	found, err := h.client.MeasuredGenres(ctx, in.Corpus)
	if err != nil {
		return nil, nil, err
	}

	earning := 0

	for _, g := range found {
		if len(g.Terms) > 0 {
			earning++
		}
	}

	return said("%d genres measured, %d earning a word", len(found), earning), found, nil
}

func (h *handlers) measurePlayers(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Corpus,
) (*gomcp.CallToolResult, []audio.Player, error) {
	found, err := h.client.MeasuredPlayers(ctx, in.Corpus)
	if err != nil {
		return nil, nil, err
	}

	earning := 0

	for _, p := range found {
		if len(p.Terms) > 0 {
			earning++
		}
	}

	return said("%d players measured, %d earning a word", len(found), earning), found, nil
}

func (h *handlers) measureRecordings(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Recordings,
) (*gomcp.CallToolResult, Recorded, error) {
	tracks, together, err := h.client.MeasuredRecordings(ctx, in.Dir)
	if err != nil {
		return nil, Recorded{}, err
	}

	return said("%d recordings measured", len(tracks)),
		Recorded{Tracks: tracks, Together: together}, nil
}
