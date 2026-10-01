<p align="center">
  <picture>
    <source srcset="docs/assets/logo-dark.svg" media="(prefers-color-scheme: dark)">
    <source srcset="docs/assets/logo-light.svg" media="(prefers-color-scheme: light)">
    <img src="docs/assets/logo-dark.svg" alt="toneharness" width="610">
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

## What ships in the binary

| in the binary     | what it is                                                                                               |
| ----------------- | -------------------------------------------------------------------------------------------------------- |
| **661** blocks    | what an HX Stomp models, of which **224** are amplifiers and **133** cabinets. A Helix Floor is **670**. |
| **661** measured  | every one of those blocks, played and recorded on the device rather than read off a spec sheet           |
| **4,324** presets | what other people built, measured into the statistics that say where a control usually sits              |
| **15** rigs       | curated, with a citation behind every piece of gear                                                      |

## Quickstart

Start your agent in a checkout and say what you want:

- _"Make my bass sound like Dookie"_
- _"Like Mike Dirnt, but punchier"_
- _"What did Geddy Lee actually play on Hemispheres?"_
- _"I want a punk sound, find me an empty slot and let's work there"_

You get back a `.hlx` preset you can load in HX Edit, and if the pedal is
plugged in, the chain on the pedal. Nothing has to be installed first and no
pedal has to be attached to build one.

**Then you play it**, because nothing here can hear. Say what is wrong in your
own words and it turns the dials, rebuilds, and records what worked so the next
session starts from there. With the pedal attached it can measure what comes
back, which is the only thing that tells you whether a change did what it meant
to.

### Say what you mean, in the words it has

Twenty-five words describe a sound here, and "chunky" is not one of them. Ask
for it and you are told so, with the nearest words it does have. `punchy` is a
bottom that stops with the note and a front you hear first; `mid-forward` puts
the part in front of the mix. Ask your agent what the words are and it will read
them out of [words.json](pkg/sdk/internal/compile/data/words.json), which is the
vocabulary itself rather than a page about it.

A genre works the same way. Three are measured, and one needs eight records from
three players before anything may aim at it, so ask which are ready rather than
guessing.

### Asking for somebody nobody has researched yet

Fifteen rigs ship, each with a citation behind every piece of gear. Name
somebody who is not one of them and the agent researches it and writes the rig
with its sources, which is a different and slower thing than building from one
that exists. If the evidence will not hold up, the honest answer is that it
could not be established, not a plausible rig.

### Over MCP

`.mcp.json` starts a server, and every command has a tool named after it:
`device select` is `device_select`. Use whichever your agent prefers; both do
the same work.

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

## Install

```bash
curl -fsSL https://github.com/retr0h/toneharness/raw/main/install.sh | bash
```

`toneharness <command> --help` is the command reference, and no page here
duplicates it. From a checkout it is `go run main.go --help`, which compiles the
tree and answers from the source rather than from a description of it.

Installs to `~/.local/bin` or `/usr/local/bin`, verifying SHA256 checksums.
Override with `TONEHARNESS_INSTALL_DIR=/some/path`, or pin a version with
`TONEHARNESS_VERSION=1.1.1`.

<details>
<summary>Other ways</summary>

```bash
go install github.com/retr0h/toneharness@latest
```

Released binaries reach a Helix over USB on macOS. On Linux they build, validate
and write presets, and the device commands say they are not supported yet.

</details>

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, setup, conventions and
the pull request workflow.

## License

The MIT License, see [LICENSE](LICENSE).

[agent skills]: https://agentskills.io
