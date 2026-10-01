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

> _"Make my bass sound like Dookie."_
>
> _"I want a punk sound."_
>
> _"What did Geddy Lee actually play on Hemispheres?"_

### Put it on the pedal

> _"Put that on the pedal."_
>
> _"Find me an empty slot and put it there."_
>
> _"What is the pedal playing right now?"_

HX Edit does not have to be running. You get the `.hlx` file too, if you want it
there instead.

### Fix what you just heard

Nothing here can hear, so this is the loop: you play it, you say what is wrong.

> _"Too woolly. Tighten the bottom up."_
>
> _"Closer, but I want the pick to cut more."_
>
> _"Turn it down a bit and measure it again."_

### Find out which words do something

> _"What words can I use, and what does each one do?"_
>
> _"I asked for chunky and nothing moved. What should I have said?"_

Nothing is refused over a word. Twenty-five are defined and only those move a
control; the rest come back named, with the nearest ones that are.

### Ask for somebody who does not ship

> _"Do you have a rig for Tim Commerford, or do you have to research it?"_
>
> _"Research Justin Chancellor's Lateralus rig and write it up with sources."_
>
> _"Why is this rig only medium confidence?"_

Fifteen rigs ship with a citation behind every piece of gear. Anybody else is
research, and if the evidence will not hold up you are told that rather than
handed a plausible rig.

### Ask which genres are ready

> _"Which genres can I aim at, and which are short of the threshold?"_

Eight records from three players, or it is one band's sound wearing a genre's
name.

### Add records to a genre

> _"Add Justin Chancellor to the bass corpus with four Lateralus tracks."_
>
> _"Fetch the records you just named."_
>
> _"Cut the bass out of them."_
>
> _"Measure the corpus and tell me what prog-metal earns now."_
>
> _"Regenerate what ships and open the PR."_

The records and the stems stay on your disk. A pull request carries the manifest
and the measurements, never the audio.

<details>
<summary>What your agent runs for that last one</summary>

```bash
# 1. the manifest first, so the record measured is the one the evidence names
#    resources/music/bass/justin-chancellor/corpus.yaml

# 2. fetch by the link, never a search
mise exec -- just record resources/music/bass/justin-chancellor schism \
  https://open.spotify.com/track/1dMFQX2BqPkR5zjC2DxFUM

# 3. a mix measures the band, so the bass has to come out of it
mise exec -- just stems resources/music/bass/justin-chancellor \
  resources/music/bass/justin-chancellor/stems bass

# 4. a word is earned against the other players, so measure the tree
mise exec -- go run main.go measure players --corpus resources/music/bass
mise exec -- go run main.go measure genres  --corpus resources/music/bass

# 5. roll it into the file the binary embeds
mise exec -- just generate     # writes pkg/sdk/audio/data/genres.json
```

Re-measure only when the records or the stems change. The numbers are committed,
so nobody else runs any of this.

</details>

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
