<p align="center">
  <picture>
    <source srcset="asset/logo-dark.svg" media="(prefers-color-scheme: dark)">
    <source srcset="asset/logo-light.svg" media="(prefers-color-scheme: light)">
    <img src="asset/logo-dark.svg" alt="toneharness" width="610">
  </picture>
</p>

<p align="center">Describe a guitar or bass sound, get a Line 6 Helix preset.</p>

<p align="center">
  <a href="https://github.com/retr0h/toneharness/releases/latest"><img alt="release" src="https://img.shields.io/github/release/retr0h/toneharness.svg?style=for-the-badge"></a>
  <a href="https://codecov.io/gh/retr0h/toneharness"><img alt="codecov" src="https://img.shields.io/codecov/c/github/retr0h/toneharness?style=for-the-badge"></a>
  <a href="LICENSE"><img alt="license" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge"></a>
  <a href="https://github.com/retr0h/toneharness/actions/workflows/go.yml"><img alt="build" src="https://img.shields.io/github/actions/workflow/status/retr0h/toneharness/go.yml?style=for-the-badge"></a>
  <a href="https://github.com/goreleaser"><img alt="powered by" src="https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=for-the-badge"></a>
  <a href="https://conventionalcommits.org"><img alt="conventional commits" src="https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge"></a>
  <a href="https://just.systems"><img alt="built with just" src="https://img.shields.io/badge/Built_with-Just-black?style=for-the-badge&logo=just&logoColor=white"></a>
  <img alt="github commit activity" src="https://img.shields.io/github/commit-activity/m/retr0h/toneharness?style=for-the-badge">
  <a href="https://pkg.go.dev/github.com/retr0h/toneharness"><img alt="go reference" src="https://img.shields.io/badge/go-reference-00ADD8?style=for-the-badge&logo=go&logoColor=white"></a>
  <a href="https://github.com/tekk/hovnokod-badge"><img alt="hovnokod" src="https://raw.githubusercontent.com/tekk/hovnokod-badge/main/assets/badges/hovnokod-for-the-badge.svg"></a>
</p>

<p align="center">
<b>Built for agents, there to empower humans.</b>
</p>

<p align="center">
Every block on the device was measured through it, every claim in a rig names
where it came from, and where nothing has been measured the tool says so.
Point an agent at a checkout and tell it what you want to sound like.
</p>

## Features

| Feature                                                                     | Description                                                                                                                                                                                                                                 |
| --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [ToneSpec](pkg/sdk/tone/data/tonespec.openapi.yaml)                         | One file describes a sound: `ask:` is what you want it to sound like and `rig:` names the gear that answers it, the way a player would. Every field follows an OpenAPI contract, and toneharness refuses a document that breaks it          |
| [Marketplace](marketplace/README.md)                                        | A rig names "Ampeg SVT", not a Line 6 model ID, so one is worth sending. A cited core tier ships in the binary and a community tier loads with `--rigs`. Export a slot as a document, send it, and `presets compile` builds it on their end |
| [Every claim sourced](.claude/skills/build-a-rig/references/evidence.md)    | Each piece of gear and each word records where it came from: an interview, a video timestamp, a forum thread, a measurement. A claim a model made says so                                                                                   |
| [Your agent tunes it](.claude/skills/build-a-rig/references/correcting.md)  | Your agent builds a rig for the player you name. Play it, say what is wrong, and it rebuilds. The ask keeps each round: what you asked, what changed, why, and your verdict. The next session starts from what worked                       |
| [Measured, not guessed](.claude/skills/measure-a-device/README.md)          | Every one of the 661 blocks was played through the device and recorded, 19 of them had each control swept, and 52 records across 16 players were measured. Where nothing has been measured the tool says so rather than aiming at a number  |
| [Talks to your Helix](.claude/skills/work-a-device/README.md)               | Read, write, copy, swap and select presets over USB, and a slot is saved to a file before anything overwrites it. Device access is one backend per operating system and only macOS has one so far; everything else here runs anywhere       |
| [MCP server](.mcp.json)                                                     | `toneharness mcp start` gives an agent the catalog, the corpus, the rigs, building and the pedal as tools with typed results. The running server is what advertises them. It cannot write to a pedal without `--allow-writes`               |
| [Go SDK](CONTRIBUTING.md#what-to-import-if-you-are-using-this-as-a-library) | The CLI is flags over `pkg/sdk`. Import it to build presets, read and write a device, or look up what a device can do from your own Go program                                                                                              |

## One text format, and a preset falls out of it

A **ToneSpec** holds both halves of a sound: `ask:` is what you want and `rig:`
is the gear that answers it. YAML somebody can read, write, diff and send to a
friend.

The specification is OpenAPI, and every field carries its own description, so
what a field may say and what gets refused is in the contract rather than in a
page about it: [tonespec.openapi.yaml](pkg/sdk/tone/data/tonespec.openapi.yaml).

It is the only hand-authored format here. The Go types and everything downstream
are generated from it, and [write-a-spec](.claude/skills/write-a-spec/SKILL.md)
is what reads it back in prose, field by field.

```yaml
schema: ToneSpec
id: mike-dirnt
ask:
  genre: [pop-punk, punk]
rig:
  instrument: bass
  chain:
    - role: amp
      gear: Ampeg SVT
      evidence:
        - kind: cited
          url: https://...
```

A rig names **real gear, never Line 6 model identifiers**. "Ampeg SVT" resolves
through the gear map when it is built, so the file stays readable, stays correct
when Line 6 rename a model, and compiles for whichever Helix you own rather than
the one it was written on.

It goes the other way too. Pull a preset off the pedal and it comes back as a
rig, so somebody else's sound becomes a file you can read, change and build
again:

> - _"Export slot 12B as a rig so I can see what is in it."_
> - _"Build this rig somebody sent me and put it on my pedal."_

Every claim in a rig carries its source, which is what makes one worth sending.
[marketplace/](marketplace/) is where they live: a cited core that ships in the
binary, and a community tier you load with `--rigs`. Its README says what the
two tiers are and how to submit one.
[marketplace/core/examples/](marketplace/core/examples/) holds one of each
document with every optional field filled in. Those are what the tests pin and
what to read when you want to see a field used, rather than rigs anybody plays.

## What ships in the binary

| in the binary     | what it is                                                                                               |
| ----------------- | -------------------------------------------------------------------------------------------------------- |
| **661** blocks    | what an HX Stomp models, of which **224** are amplifiers and **133** cabinets. A Helix Floor is **670**. |
| **661** measured  | every one of those blocks, played and recorded on the device rather than read off a spec sheet           |
| **4,324** presets | what other people built, measured into the statistics that say where a control usually sits              |
| **15** rigs       | curated, with a citation behind every piece of gear                                                      |

## Quickstart

Start your agent in a checkout and talk to it. Everything below is something to
type.

> - _"Make my bass sound like Dookie."_
> - _"I want a punk sound."_
> - _"What did Geddy Lee actually play on Hemispheres?"_

### Put it on the pedal

> - _"Put that on the pedal."_
> - _"Find me an empty slot and put it there."_
> - _"What is the pedal playing right now?"_

HX Edit does not have to be running. You get the `.hlx` file too, if you want it
there instead.

### Fix what you just heard

Nothing here can hear, so this is the loop: you play it, you say what is wrong.

> - _"Too woolly. Tighten the bottom up."_
> - _"Closer, but I want the pick to cut more."_
> - _"Turn it down a bit and measure it again."_

### Find out which words do something

> - _"What words can I use, and what does each one do?"_
> - _"I asked for chunky and nothing moved. What should I have said?"_

Nothing is refused over a word. Twenty-five are defined and only those move a
control; the rest come back named, with the nearest ones that are.

### Ask for somebody who does not ship

> - _"Build me a rig for Justin Chancellor's Lateralus sound."_
> - _"Do you have Tim Commerford, or do you have to research him?"_
> - _"Why is this rig only medium confidence?"_

Fifteen rigs ship with a citation behind every piece of gear. Anybody else gets
researched, written up with sources, built to check it resolves, and opened as a
pull request. If the evidence will not hold up you are told that instead, with
what was searched, because a plausible rig looks like knowledge and is not.

### Add a genre, or records to one

> - _"Add Justin Chancellor to the bass corpus and measure what prog-metal
>   earns."_
> - _"Which genres can I aim at, and which are short of the threshold?"_

It fetches the records, cuts the bass out, measures, and opens the pull request.

Two things it will tell you rather than let you find out: a genre needs eight
records from three players before anything may aim at it, and a player whose
records are measured but who has no rig contributes figures nothing can act on.

Your copies of the records stay on your disk.

### What needs the pedal, and what does not

Most of this needs no hardware. A corpus is audio files on disk, so adding
records, measuring players, earning words and measuring a genre all run on a
laptop with nothing plugged in. So does researching a rig, building one, and
writing the preset.

Two things need the pedal, and one of those needs a lead from its output back to
an input:

|                                                        | needs                             |
| ------------------------------------------------------ | --------------------------------- |
| `tone tune`, `tone reach`, `device play`               | the pedal on USB                  |
| `measure blocks`, `measure controls`, `measure slopes` | the pedal, and the measuring loop |

The second row is how the 19 swept blocks in
[resources/sweeps/](resources/sweeps/) were measured, and they are committed, so
nobody re-runs them. Without a loop you lose tuning a chain by measurement,
which is the part that says whether a change did what it meant to. Everything
else works.

## Skills

The CLI is for an agent more than for a person, so the way to use toneharness is
to point an agent at a checkout and say what you want to sound like. Each
skill's own README says how to install and use it.

| Skill                                                         | Answers                                                                          |
| ------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| [build-a-rig](.claude/skills/build-a-rig/README.md)           | _"Make my bass sound like Dookie"_, without guessing at the gear                 |
| [write-a-spec](.claude/skills/write-a-spec/README.md)         | Which document a fact goes in, and why a field was refused                       |
| [measure-music](.claude/skills/measure-music/README.md)       | What a player's records actually sound like, and which words the figures earn    |
| [work-a-device](.claude/skills/work-a-device/README.md)       | What is on the pedal, getting a preset onto it, and moving slots around          |
| [measure-a-device](.claude/skills/measure-a-device/README.md) | What a control actually does, by pushing a known signal through it and listening |

Each follows the [Agent Skills] format: a slim `SKILL.md` that routes, with the
detail in reference files an agent reads only when the question calls for them.
None of them writes down a list the tool can print. A list in a skill is right
the day it is written and wrong after the next change, with nothing marking the
moment.

## Running it

A checkout, which is how the Quickstart above works: an agent reads the skills
out of `.claude/skills/` and runs the tree.

```bash
git clone https://github.com/retr0h/toneharness
cd toneharness
mise exec -- go run main.go --help
```

`mise` supplies the Go version `.mise.toml` declares. `go run main.go` compiles
the tree every time, so what answers is the source rather than a binary that may
be older than the branch, and `--help` is the command reference: no page here
duplicates it.

There are no releases yet, so there is nothing to install. When there are, the
installer and `go install` are the other two ways in:

<details>
<summary>Once a release exists</summary>

```bash
curl -fsSL https://github.com/retr0h/toneharness/raw/main/install.sh | bash
go install github.com/retr0h/toneharness@latest
```

The installer writes to `~/.local/bin` or `/usr/local/bin` and verifies SHA256
checksums. `TONEHARNESS_INSTALL_DIR` moves it, `TONEHARNESS_VERSION` pins one.

A released binary reaches a Helix over USB on macOS. On Linux it builds,
validates and writes presets, and the device commands say they are not supported
yet.

</details>

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, setup, conventions and
the pull request workflow.

## License

The MIT License, see [LICENSE](LICENSE).

[agent skills]: https://agentskills.io
