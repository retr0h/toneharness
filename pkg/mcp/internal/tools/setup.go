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
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/toneharness/pkg/sdk"
	"github.com/retr0h/toneharness/pkg/sdk/tone"
)

// steps are what device_fav's step takes.
//
// Empty is the first one, so a caller that leaves it out gets the one thing
// that always works: going to the first needs nothing read off the pedal and
// cannot be confused by what is loaded.
var steps = map[string]sdk.Step{
	"":      sdk.AtFirst,
	"first": sdk.AtFirst,
	"next":  sdk.Onward,
	"prev":  sdk.Back,
}

// deviceFav loads one of the setup's favourite slots.
func (h *handlers) deviceFav(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Favourite,
) (*gomcp.CallToolResult, sdk.Chosen, error) {
	step, ok := steps[in.Step]
	if !ok {
		return nil, sdk.Chosen{}, fmt.Errorf(
			"step is %q; it takes first, next or prev", in.Step)
	}

	// `alone` rather than `onPedal`, because Favourite opens a session of its
	// own and a session this server still holds would refuse it.
	chosen, err := alone(ctx, h.pedal, func() (sdk.Chosen, error) {
		return h.client.Favourite(ctx, step)
	})
	if err != nil {
		return nil, sdk.Chosen{}, err
	}

	// What it moved from as well as what it landed on, because a cycle only
	// reads if it says both, and because a step that recognised nothing is a
	// thing the caller should know went to the first.
	if chosen.From == "" {
		return said("loaded %s, favourite %d of %d; no favourite was playing",
			chosen.At, chosen.Position, chosen.Of), chosen, nil
	}

	return said("loaded %s, favourite %d of %d, from %s",
		chosen.At, chosen.Position, chosen.Of, chosen.From), chosen, nil
}

// setupShow answers with what the person has.
func (h *handlers) setupShow(
	_ context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, tone.Setup, error) {
	held, err := h.client.Setup()
	if err != nil {
		return nil, tone.Setup{}, err
	}

	slots := 0
	if held.Slots != nil {
		slots = len(*held.Slots)
	}

	favourites := 0
	if held.Favourites != nil {
		favourites = len(*held.Favourites)
	}

	return said("the setup places %d slots and lists %d favourites",
		slots, favourites), held, nil
}

// setupInit writes a setup somebody can edit.
func (h *handlers) setupInit(
	_ context.Context,
	_ *gomcp.CallToolRequest,
	in StartingSetup,
) (*gomcp.CallToolResult, sdk.Written, error) {
	written, err := h.client.StartSetup(in.At)
	if err != nil {
		return nil, sdk.Written{}, err
	}

	return said("wrote %s; every field is commented out, so uncomment what "+
		"applies", written.Path), written, nil
}
