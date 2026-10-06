---
name: measure-music
description: Measure what records actually sound like, and turn the figures into words somebody can check. Covers growing a player's corpus, choosing records that measure the player rather than the studio, fetching audio and proving it is the right recording, separating the bass and reading the figures, deriving the words a player earns against the others, measuring what a genre is displaced on, and holding a rig's records to the era it claims. Use when asked to add records to a corpus, measure a player, a genre or a recording, derive words from figures, or check whether the records behind a rig are the right records.
compatibility: Requires a toneharness checkout with mise, uv and ffmpeg available. Every command runs through `mise exec -- go run main.go`, never a bare `toneharness`.
license: MIT
metadata:
  author: retr0h
  source: https://github.com/retr0h/toneharness
---

# Measure music

## 1. Establish what the corpus already holds

Every run, before naming a player, a record or a genre. The manifests are the
only answer to what has been measured, and these read them rather than the
audio, so they cost a file read:

```bash
mise exec -- go run main.go corpus music --help                                   # which questions exist
mise exec -- go run main.go corpus music players --corpus resources/music/bass --json
mise exec -- go run main.go corpus music records --corpus resources/music/bass --json
mise exec -- go run main.go corpus music genres  --corpus resources/music/bass --json
mise exec -- go run main.go corpus music bands   --corpus resources/music/bass --json
```

`--help` on a checkout compiles the checkout, so it is the source's own answer
rather than a description of it, and it shows the flags a parent registers that
reading one file would miss. Read `cmd/` only to change a command, never to find
out what one does.

Never work from a list written into this skill. Every record added moves what
every player earns, so a list of players, genres, records or figures here is
right the day it is written and wrong after the next download, with nothing
marking the moment. Ask the tool.

The layout is a rule rather than a habit:
[resources/music/README.md](../../../resources/music/README.md) says why the
directory is the instrument, why a band and a genre are fields on a record
instead of directories, and what git will not bring back.

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

| The user asks                                                  | Read                                     |
| -------------------------------------------------------------- | ---------------------------------------- |
| to grow a corpus, or which records to add next                 | [references/growing.md](references/growing.md)   |
| to fetch a record, or whether the file is the right recording   | [references/fetching.md](references/fetching.md) |
| what a recording measures, and what a figure may be read as    | [references/figures.md](references/figures.md)   |
| which words a player earns, or why one was lost                | [references/words.md](references/words.md)       |
| what a genre is, or whether it can be aimed at yet             | [references/genres.md](references/genres.md)     |
| whether the records behind a rig are from the era it claims    | [references/era.md](references/era.md)           |
| what to re-run after records change, and what a PR carries | [references/finishing.md](references/finishing.md) |
| all of it, in order                                            | All seven, in that order                   |

## 3. Say which claim you have

Four different claims, and each is cheap to report as the next one up:

1. the manifest names the record
2. the file on disk is that recording, checked against its length
3. the stems were measured, and these are the figures
4. the figures earn a word against the others measured the same way

Neither the audio nor the measurements are in this repository, so a figure in an
ask is a claim about a recording nobody else here can replay. The url is the only
thing that makes it checkable, and a claim without one is an assertion with
decimal places. If you did not run it, say you did not run it.
Beyond the ladder, [AGENTS.md's what to report](../../../AGENTS.md#what-to-report-every-time)
is the shape every skill answers in: where anything landed by path, what was
decided on somebody's behalf, which rung you reached, and what it cost the device.


Report in this order, because it is the order that avoids rework:

1. what was asked, in the user's words
2. which records are involved, with their links and years
3. what the tool answered, including everything it said on stderr about records
   it could not match either way
4. which of the four claims above you have
