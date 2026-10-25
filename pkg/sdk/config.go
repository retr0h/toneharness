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
package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/retr0h/toneharness/pkg/sdk/config"
)

// ErrConfigExists is returned rather than writing over one somebody has.
var ErrConfigExists = errors.New("a setup is already there")

// Where the two things a starter setup needs come from, as variables so both
// failures can be exercised.
//
// Neither can fail in a correct build: the home directory is there and the
// contract is embedded. That is the reason they are here rather than called
// directly. A branch nothing can reach is a branch nobody has read, and these
// two are the ones that decide whether somebody gets a file they can edit or a
// message saying why not.
var (
	homeDir         = os.UserHomeDir
	contractVersion = config.Version
)

// ConfigPath is where somebody's own setup lives.
//
// Resolved the way rigs and backups resolve theirs: the XDG directory for the
// kind of file it is, and the documented fallback under the home directory
// where the environment says nothing. A setup is configuration rather than
// data or state, so it is the config directory.
//
// **Exported rather than applied here.** The library reads none of this tool's
// environment and nothing defaults to this path: a Client with no WithConfig
// has no setup, which is what keeps a test from reading whatever the machine
// running it happens to have. The commands and the MCP server are what pass
// it in, so the default is a decision of the thing a person is talking to.
func ConfigPath() (string, error) {
	if config := os.Getenv("XDG_CONFIG_HOME"); config != "" {
		return filepath.Join(config, "toneharness", "config.yaml"), nil
	}

	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("finding where a setup lives: %w", err)
	}

	return filepath.Join(home, ".config", "toneharness", "config.yaml"), nil
}

// Setup answers with what the person has, as the document says it.
//
// The read every other setup-shaped answer is built on, exported because
// "which slots is this managing" is a question worth asking on its own: the
// slot map and the favourites are the only record of which presets belong to
// this tool rather than to whoever was holding the pedal.
func (c *Client) Setup() (config.Config, error) {
	return c.loadConfig()
}

// StartConfig writes a setup somebody can edit, at the path given.
//
// **Why a command rather than a page telling somebody what to type.** Every
// field here is optional and the document is useless until several of them
// are filled in, so the first thing anybody needs is a file with the right
// schema line, the right version and a comment beside each field saying what
// it is for. Written from the contract's own vocabulary, so a starter file
// cannot describe a field that no longer exists.
//
// Nothing is written over. A setup is hand-authored and the only copy of
// decisions nothing else records, so an existing one is reported rather than
// replaced: there is no flag for it, because "replace my setup" is not a thing
// this offers. Renaming the old one is the person's call to make.
//
// Checked through the loader before it lands rather than after, so a template
// that has drifted from the contract never reaches the disk.
func (c *Client) StartConfig(
	at string,
) (Written, error) {
	if at == "" {
		named, err := ConfigPath()
		if err != nil {
			return Written{}, err
		}

		at = named
	}

	if _, err := os.Stat(at); err == nil {
		return Written{}, fmt.Errorf("%w: %s", ErrConfigExists, at)
	}

	body, err := starterSetup()
	if err != nil {
		return Written{}, err
	}

	// Checked through the loader before it lands rather than after. A starter
	// file that does not validate is worse than none: it is one somebody
	// edits for an hour before the first refusal, and the refusal then names
	// their line rather than the one that shipped. Checking first means the
	// bad one never reaches the disk at all.
	if _, err := config.Load(strings.NewReader(body)); err != nil {
		return Written{}, fmt.Errorf("the starter setup does not validate: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		return Written{}, fmt.Errorf("making somewhere for a setup: %w", err)
	}

	if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
		return Written{}, fmt.Errorf("writing %s: %w", at, err)
	}

	return Written{Path: at, Name: config.Name}, nil
}

// starterSetup is the file StartConfig writes.
//
// Commented rather than bare, because a document of optional fields with
// nothing filled in says nothing about what to fill in. Every line that would
// be a guess about somebody's gear is left commented out, so what lands is
// valid on its own and becomes useful one uncommented line at a time.
//
// The version comes out of the contract rather than being typed here, which is
// the whole reason this is a function: a starter file stating a version the
// embedded contract does not is one that fails the moment it is read back.
func starterSetup() (string, error) {
	version, err := contractVersion()
	if err != nil {
		return "", fmt.Errorf("writing a starter setup: %w", err)
	}

	return `# What you have, and what you want done with it.
#
# Read by every build, so a rig comes out fitting your instrument and your
# pedal rather than the record it was measured from. Nothing here is required
# beyond the two lines at the top; uncomment what applies.
schema: Config
version: "` + version + `"

# Which pedal. Decides how many blocks a build may use and which models exist.
# device: HX Stomp

# What you play. More than one is normal and they do not sound the same.
# instruments:
#   - gear: Fender Precision Bass
#     strings: roundwound
#     pickup: split-coil

# Which slot each rig is kept in, so "update my pedal" lands in the same
# places twice. This is the layout ` + "`slots fill`" + ` writes.
# slots:
#   - { at: 11A, rig: punk }
#   - { at: 11B, rig: pop-punk }
#   - { at: 18A, rig: mike-dirnt }

# The slots worth getting to in one step, in the order to cycle them. The
# first is where ` + "`device fav`" + ` lands.
# favourites: ["11A", "18A", "11B"]

# A rig every build should come out as loud as, by its identifier, so your
# own presets stay comparable to each other.
# match_loudness: punk
`, nil
}
