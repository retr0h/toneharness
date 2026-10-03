---
name: build-a-rig
description: Turn what somebody says they want to sound like into a Line 6 Helix preset, with evidence at every step. Covers researching a player's real gear and citing it, writing a ToneSpec, resolving it to a rig, measuring a recording to choose an amplifier, putting a preset on the pedal, tuning live and exporting what worked. Use when asked to make something sound like an artist, a band, a record or a genre, to build or correct a rig or preset, to find out what gear somebody used, to measure a recording, or to get a tone onto a Helix.
compatibility: Requires a toneharness checkout with mise available. Every command runs through `mise exec -- go run main.go`, never a bare `toneharness`.
license: MIT
metadata:
  author: retr0h
  source: https://github.com/retr0h/toneharness
---

# Build a rig

## 1. Establish what the device can actually do

Every run, before naming any gear. The catalog is the only answer to what
exists, and a model it does not carry does not exist:

Start from the tree itself, then the group you need:

```bash
mise exec -- go run main.go --help              # which groups exist
mise exec -- go run main.go catalog --help      # what that group takes
mise exec -- go run main.go catalog list --category amp --json
mise exec -- go run main.go rigs list --json
```

`--help` on a checkout compiles the checkout, so it is the source's own answer
rather than a description of it, and it shows the flags a parent registers that
reading one file would miss. Read `cmd/` only to change a command, never to find
out what one does.

What a document may say is the contract, not a page about it:
`pkg/sdk/tone/data/tonespec.openapi.yaml`, which describes both the `ask:` and the
`rig:` half. Its `description:` fields are the grammar, and the tool refuses a
document that breaks it.

Never work from a list written into this skill. The device carries 661 blocks
across four models and Line 6 rename things between releases, so a list here
is right the day it is written and wrong after the next one, with nothing
marking the moment. The same goes for the shipped rigs, the block categories
and the figures a reading is reported in: ask the tool.

`--json` works on every command. Use it. The painted tables are for somebody
reading a terminal and they leave things out on purpose.

## Two surfaces, one SDK

Every command here has a tool beside it over MCP, named after the command:
`device select` is `device_select`, `corpus music genres` is
`corpus_music_genres`. Use whichever the session offers. **The tools are the
same operations, not a reimplementation**: both surfaces call `pkg/sdk` and
answer with the same types, and a test walks the command tree against the
registered tools both ways, so neither can quietly gain a capability the other
lacks.

`.mcp.json` starts the server with `go run`, so it compiles the working tree
every launch and cannot serve a stale binary.

Three commands have no tool, and the reason is in that test's exempt list:
`measure blocks`, `measure controls` and `measure names` are sweeps. One reading
is about eight seconds, so a twelve-control amplifier is most of an hour, and a
tool that blocks that long is not one anybody can use. Run those from a terminal
where the progress shows and Ctrl-C reaches the session holding the pedal.

## 2. Route

| The user asks                                                    | Read                                                        |
| ---------------------------------------------------------------- | ----------------------------------------------------------- |
| "make it sound like X", "build me a rig", what gear somebody used | [references/research.md](references/research.md)            |
| where to search for gear evidence, and what not to accept         | [references/sources.md](references/sources.md)              |
| to write or change a request, a ToneSpec, a Setup                 | [references/asking.md](references/asking.md)                |
| something too vague to build, or "what should I ask them?"        | [references/vague.md](references/vague.md)                  |
| to measure a recording, or to choose gear by measuring            | [references/measuring.md](references/measuring.md)          |
| to get it onto the pedal, or to read what the pedal holds         | [references/device.md](references/device.md)                |
| to change a rig after hearing it, or what a target is             | [references/correcting.md](references/correcting.md)        |
| anything touching claims, citations or evidence                   | [references/evidence.md](references/evidence.md)            |
| to write the rig down, check it builds, and ship it | [references/shipping.md](references/shipping.md) |
| all of it, in order                                               | All nine, in that order                                    |

## 3. Say which claim you have

Three different claims, and only the first is currently possible here:

1. the rig validates against the catalog
2. HX Edit imported the file
3. the hardware loaded it and it sounded right

Never report one as another, and never describe work as verified on evidence
you did not gather. If you did not run it, say you did not run it. This is the
failure this project cares about most, and machine-readable output makes it
easier to commit rather than harder.

Report in this order, because it is the order that avoids rework:

1. what the request was, in the user's words
2. what was established and from which source, with a link
3. what the tool answered, including every note it returned about what it
   could not honour
4. which of the three claims above you have
